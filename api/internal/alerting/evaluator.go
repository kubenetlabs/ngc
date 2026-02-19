package alerting

import (
	"context"
	"fmt"
	"hash/fnv"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/kubenetlabs/ngc/api/internal/database"
	prom "github.com/kubenetlabs/ngc/api/internal/prometheus"
)

// Evaluator periodically evaluates alert rules against metric values
// and sends webhook notifications when alert state changes.
type Evaluator struct {
	store    database.Store
	prom     *prom.Client
	interval time.Duration
	mu       sync.Mutex
	firing   map[string]*FiringAlert // ruleID -> alert
	webhooks []WebhookConfig
	cancel   context.CancelFunc
}

// FiringAlert represents an alert that is currently in the firing state.
type FiringAlert struct {
	RuleID    string    `json:"ruleId"`
	RuleName  string    `json:"ruleName"`
	Severity  string    `json:"severity"`
	Resource  string    `json:"resource"`
	Metric    string    `json:"metric"`
	Value     float64   `json:"value"`
	Threshold float64   `json:"threshold"`
	Operator  string    `json:"operator"`
	FiredAt   time.Time `json:"firedAt"`
}

// WebhookConfig defines a webhook notification target.
type WebhookConfig struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

// New creates a new Evaluator with the given store, Prometheus client, and webhook configs.
// The evaluation interval is fixed at 60 seconds.
func New(store database.Store, promClient *prom.Client, webhooks []WebhookConfig) *Evaluator {
	return &Evaluator{
		store:    store,
		prom:     promClient,
		interval: 60 * time.Second,
		firing:   make(map[string]*FiringAlert),
		webhooks: webhooks,
	}
}

// Start begins the background evaluation loop. It runs evaluate() once
// immediately, then every interval until the context is cancelled or Stop() is called.
func (e *Evaluator) Start(ctx context.Context) {
	ctx, e.cancel = context.WithCancel(ctx)

	slog.Info("alert evaluator starting", "interval", e.interval, "webhooks", len(e.webhooks))

	go func() {
		// Run an initial evaluation immediately.
		e.evaluate(ctx)

		ticker := time.NewTicker(e.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				slog.Info("alert evaluator stopped")
				return
			case <-ticker.C:
				e.evaluate(ctx)
			}
		}
	}()
}

// Stop cancels the background evaluation goroutine.
func (e *Evaluator) Stop() {
	if e.cancel != nil {
		e.cancel()
	}
}

// GetFiring returns a snapshot of all currently firing alerts.
func (e *Evaluator) GetFiring() []FiringAlert {
	e.mu.Lock()
	defer e.mu.Unlock()

	alerts := make([]FiringAlert, 0, len(e.firing))
	for _, a := range e.firing {
		alerts = append(alerts, *a)
	}
	return alerts
}

// evaluate fetches all enabled alert rules from the store and evaluates each one.
// If a rule's threshold is exceeded and it is not already firing, the alert is added
// to the firing map and a webhook notification is sent. If a rule was firing but is
// now resolved, it is removed and a resolved notification is sent.
func (e *Evaluator) evaluate(ctx context.Context) {
	if e.store == nil {
		slog.Debug("alert evaluator: store not configured, skipping evaluation")
		return
	}

	rules, err := e.store.ListAlertRules(ctx)
	if err != nil {
		slog.Error("alert evaluator: failed to list alert rules", "error", err)
		return
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Track which rule IDs are still active (enabled) so we can resolve
	// alerts for rules that were deleted or disabled.
	activeRuleIDs := make(map[string]bool, len(rules))

	for _, rule := range rules {
		if !rule.Enabled {
			// If a disabled rule is currently firing, resolve it.
			if alert, ok := e.firing[rule.ID]; ok {
				slog.Info("alert resolved (rule disabled)",
					"rule_id", rule.ID,
					"rule_name", rule.Name,
				)
				go e.sendWebhook(*alert, true)
				delete(e.firing, rule.ID)
			}
			continue
		}

		activeRuleIDs[rule.ID] = true

		value, exceeded := e.evaluateRule(rule)

		if exceeded {
			if _, alreadyFiring := e.firing[rule.ID]; !alreadyFiring {
				alert := &FiringAlert{
					RuleID:    rule.ID,
					RuleName:  rule.Name,
					Severity:  rule.Severity,
					Resource:  rule.Resource,
					Metric:    rule.Metric,
					Value:     value,
					Threshold: rule.Threshold,
					Operator:  rule.Operator,
					FiredAt:   time.Now().UTC(),
				}
				e.firing[rule.ID] = alert
				slog.Warn("alert firing",
					"rule_id", rule.ID,
					"rule_name", rule.Name,
					"severity", rule.Severity,
					"metric", rule.Metric,
					"value", value,
					"threshold", rule.Threshold,
					"operator", rule.Operator,
				)
				go e.sendWebhook(*alert, false)
			}
		} else {
			// Threshold no longer exceeded — resolve if currently firing.
			if alert, ok := e.firing[rule.ID]; ok {
				slog.Info("alert resolved",
					"rule_id", rule.ID,
					"rule_name", rule.Name,
					"metric", rule.Metric,
					"value", value,
				)
				go e.sendWebhook(*alert, true)
				delete(e.firing, rule.ID)
			}
		}
	}

	// Resolve alerts for rules that no longer exist (deleted).
	for ruleID, alert := range e.firing {
		if !activeRuleIDs[ruleID] {
			slog.Info("alert resolved (rule removed)",
				"rule_id", ruleID,
				"rule_name", alert.RuleName,
			)
			go e.sendWebhook(*alert, true)
			delete(e.firing, ruleID)
		}
	}
}

// metricToPromQL maps an alert rule's metric name and resource to a PromQL query string.
// Returns an empty string for unknown metrics.
func metricToPromQL(metric, resource string) string {
	switch metric {
	case "error_rate":
		return fmt.Sprintf(`sum(rate(nginx_gateway_fabric_http_requests_total{status=~"5..",httproute_name="%s"}[5m])) / sum(rate(nginx_gateway_fabric_http_requests_total{httproute_name="%s"}[5m]))`, resource, resource)
	case "latency_p99":
		return fmt.Sprintf(`histogram_quantile(0.99, sum(rate(nginx_gateway_fabric_http_request_duration_seconds_bucket{httproute_name="%s"}[5m])) by (le)) * 1000`, resource)
	case "request_rate":
		return fmt.Sprintf(`sum(rate(nginx_gateway_fabric_http_requests_total{httproute_name="%s"}[5m]))`, resource)
	case "gpu_util":
		return fmt.Sprintf(`avg(gpu_utilization{pool="%s"})`, resource)
	case "queue_depth":
		return fmt.Sprintf(`avg(queue_depth{pool="%s"})`, resource)
	case "kv_cache_util":
		return fmt.Sprintf(`avg(kv_cache_utilization{pool="%s"})`, resource)
	case "memory_usage":
		return fmt.Sprintf(`avg(container_memory_usage_bytes{pod=~"%s.*"}) / avg(container_spec_memory_limit_bytes{pod=~"%s.*"}) * 100`, resource, resource)
	default:
		return ""
	}
}

// evaluateRule queries Prometheus for the metric value defined by the rule.
// If Prometheus is unavailable or the query fails, it falls back to synthetic
// values for graceful degradation.
func (e *Evaluator) evaluateRule(rule database.AlertRule) (float64, bool) {
	var value float64

	// Try Prometheus first.
	if e.prom != nil {
		query := metricToPromQL(rule.Metric, rule.Resource)
		if query != "" {
			v, err := e.prom.QueryScalar(context.Background(), query, time.Now())
			if err == nil {
				value = v
				slog.Debug("alert evaluator: using prometheus value",
					"rule_id", rule.ID, "metric", rule.Metric, "value", value)
			} else {
				slog.Debug("alert evaluator: prometheus query failed, falling back to synthetic",
					"rule_id", rule.ID, "metric", rule.Metric, "error", err)
				value = syntheticMetricValue(rule.Metric, rule.Resource)
			}
		} else {
			slog.Debug("alert evaluator: no promql mapping, using synthetic",
				"rule_id", rule.ID, "metric", rule.Metric)
			value = syntheticMetricValue(rule.Metric, rule.Resource)
		}
	} else {
		value = syntheticMetricValue(rule.Metric, rule.Resource)
	}

	var exceeded bool
	switch rule.Operator {
	case "gt":
		exceeded = value > rule.Threshold
	case "lt":
		exceeded = value < rule.Threshold
	case "eq":
		exceeded = math.Abs(value-rule.Threshold) < 0.001
	default:
		slog.Warn("alert evaluator: unknown operator", "operator", rule.Operator, "rule_id", rule.ID)
		return value, false
	}

	return value, exceeded
}

// syntheticMetricValue produces a deterministic mock value for a metric name
// and resource combination. The value is based on a hash of the metric+resource
// string combined with a time component that changes every 2 minutes,
// producing values that vary slowly over time.
func syntheticMetricValue(metric, resource string) float64 {
	// Use a time bucket so values change every 2 minutes for variation.
	timeBucket := time.Now().Unix() / 120

	h := fnv.New64a()
	h.Write([]byte(metric))
	h.Write([]byte(resource))

	// Write time bucket as bytes.
	for i := 0; i < 8; i++ {
		h.Write([]byte{byte(timeBucket >> (i * 8))})
	}

	hashVal := h.Sum64()

	// Map known metric names to realistic value ranges.
	switch metric {
	case "error_rate":
		// Error rate: 0-15% (most values low, occasionally high)
		return float64(hashVal%1500) / 100.0
	case "expiry_days":
		// Certificate expiry: 0-365 days
		return float64(hashVal % 366)
	case "gpu_util":
		// GPU utilization: 0-100%
		return float64(hashVal%10000) / 100.0
	case "latency_p99":
		// P99 latency: 0-5000 ms
		return float64(hashVal % 5001)
	case "request_rate":
		// Requests per second: 0-10000
		return float64(hashVal % 10001)
	case "memory_usage":
		// Memory usage: 0-100%
		return float64(hashVal%10000) / 100.0
	case "queue_depth":
		// Queue depth: 0-500
		return float64(hashVal % 501)
	case "kv_cache_util":
		// KV cache utilization: 0-100%
		return float64(hashVal%10000) / 100.0
	default:
		// Generic: 0-100
		return float64(hashVal%10000) / 100.0
	}
}
