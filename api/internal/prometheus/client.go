package prometheus

import (
	"context"
	"fmt"
	"net/http"
	"time"

	promapi "github.com/prometheus/client_golang/api"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// Client wraps a Prometheus HTTP API client.
type Client struct {
	api promv1.API
	url string
}

// New creates a new Prometheus client pointing at the given URL.
func New(url string) (*Client, error) {
	client, err := promapi.NewClient(promapi.Config{
		Address: url,
		Client:  &http.Client{Timeout: 15 * time.Second},
	})
	if err != nil {
		return nil, fmt.Errorf("create prometheus client: %w", err)
	}
	return &Client{api: promv1.NewAPI(client), url: url}, nil
}

// MetricsSummary holds aggregated RED metrics.
type MetricsSummary struct {
	TotalRequests     float64 `json:"totalRequests"`
	ErrorRate         float64 `json:"errorRate"`
	AvgLatencyMs      float64 `json:"avgLatencyMs"`
	P50LatencyMs      float64 `json:"p50LatencyMs"`
	P95LatencyMs      float64 `json:"p95LatencyMs"`
	P99LatencyMs      float64 `json:"p99LatencyMs"`
	RequestsPerSec    float64 `json:"requestsPerSec"`
	ActiveConnections float64 `json:"activeConnections"`
}

// RouteMetrics holds metrics for a single route.
type RouteMetrics struct {
	Namespace      string  `json:"namespace"`
	Name           string  `json:"name"`
	Hostname       string  `json:"hostname"`
	RequestsPerSec float64 `json:"requestsPerSec"`
	ErrorRate      float64 `json:"errorRate"`
	AvgLatencyMs   float64 `json:"avgLatencyMs"`
	P95LatencyMs   float64 `json:"p95LatencyMs"`
}

// GatewayMetrics holds metrics for a single gateway.
type GatewayMetrics struct {
	Namespace         string  `json:"namespace"`
	Name              string  `json:"name"`
	RequestsPerSec    float64 `json:"requestsPerSec"`
	ErrorRate         float64 `json:"errorRate"`
	AvgLatencyMs      float64 `json:"avgLatencyMs"`
	ActiveConnections float64 `json:"activeConnections"`
}

// clusterSelector returns a PromQL label selector for the given cluster.
// Returns "" when clusterName is empty (matches all clusters).
func clusterSelector(clusterName string) string {
	if clusterName == "" {
		return ""
	}
	return fmt.Sprintf(`cluster_name="%s",`, clusterName)
}

// Summary returns aggregated RED metrics, optionally filtered by cluster.
//
// Supports two metric naming schemes:
//   - nginx_gateway_fabric_* (NGF with observability policy / rich labels)
//   - nginx_http_* (NGF basic nginx receiver metrics)
//
// The method tries the richer scheme first and falls back to basic metrics.
func (c *Client) Summary(ctx context.Context, end time.Time, clusterName string) (*MetricsSummary, error) {
	summary := &MetricsSummary{}
	cs := clusterSelector(clusterName)

	// Requests per second — try rich labels first, fall back to basic nginx metrics
	if val, err := c.queryScalar(ctx, fmt.Sprintf(`sum(rate(nginx_gateway_fabric_http_requests_total{%s}[5m]))`, cs), end); err == nil {
		summary.RequestsPerSec = val
	} else if val, err := c.queryScalar(ctx, `sum(rate(nginx_http_requests_total[5m]))`, end); err == nil {
		summary.RequestsPerSec = val
	}

	// Error rate — only available with status labels (rich scheme)
	if val, err := c.queryScalar(ctx, fmt.Sprintf(`sum(rate(nginx_gateway_fabric_http_requests_total{%sstatus=~"5.."}[5m])) / sum(rate(nginx_gateway_fabric_http_requests_total{%s}[5m]))`, cs, cs), end); err == nil {
		summary.ErrorRate = val
	}

	// Latency percentiles — try rich histogram, fall back unavailable
	if val, err := c.queryScalar(ctx, fmt.Sprintf(`histogram_quantile(0.50, sum(rate(nginx_gateway_fabric_http_request_duration_seconds_bucket{%s}[5m])) by (le)) * 1000`, cs), end); err == nil {
		summary.P50LatencyMs = val
		summary.AvgLatencyMs = val
	}

	if val, err := c.queryScalar(ctx, fmt.Sprintf(`histogram_quantile(0.95, sum(rate(nginx_gateway_fabric_http_request_duration_seconds_bucket{%s}[5m])) by (le)) * 1000`, cs), end); err == nil {
		summary.P95LatencyMs = val
	}

	if val, err := c.queryScalar(ctx, fmt.Sprintf(`histogram_quantile(0.99, sum(rate(nginx_gateway_fabric_http_request_duration_seconds_bucket{%s}[5m])) by (le)) * 1000`, cs), end); err == nil {
		summary.P99LatencyMs = val
	}

	// Active connections — try rich labels first, fall back to basic
	if val, err := c.queryScalar(ctx, fmt.Sprintf(`sum(nginx_gateway_fabric_connections_active{%s})`, cs), end); err == nil {
		summary.ActiveConnections = val
	} else if val, err := c.queryScalar(ctx, `sum(nginx_http_connection_count_connections)`, end); err == nil {
		summary.ActiveConnections = val
	}

	// Total requests counter — try rich labels, fall back to basic
	if val, err := c.queryScalar(ctx, fmt.Sprintf(`sum(nginx_gateway_fabric_http_requests_total{%s})`, cs), end); err == nil {
		summary.TotalRequests = val
	} else if val, err := c.queryScalar(ctx, `sum(nginx_http_requests_total)`, end); err == nil {
		summary.TotalRequests = val
	}

	return summary, nil
}

// ByRoute returns per-route RED metrics, optionally filtered by cluster.
//
// With rich NGF metrics (nginx_gateway_fabric_*), routes are broken out by
// httproute_namespace/httproute_name/hostname labels. With basic nginx metrics
// (nginx_http_*), per-route breakdown is not available from Prometheus, so we
// return a single "all-routes" aggregate entry to still show some data.
func (c *Client) ByRoute(ctx context.Context, end time.Time, clusterName string) ([]RouteMetrics, error) {
	cs := clusterSelector(clusterName)

	// Try rich labels first
	result, _, err := c.api.Query(ctx,
		fmt.Sprintf(`sum by (httproute_namespace, httproute_name, hostname) (rate(nginx_gateway_fabric_http_requests_total{%s}[5m]))`, cs),
		end,
	)
	if err == nil {
		if vec, ok := result.(model.Vector); ok && len(vec) > 0 {
			var routes []RouteMetrics
			for _, s := range vec {
				routes = append(routes, RouteMetrics{
					Namespace:      string(s.Metric["httproute_namespace"]),
					Name:           string(s.Metric["httproute_name"]),
					Hostname:       string(s.Metric["hostname"]),
					RequestsPerSec: float64(s.Value),
				})
			}
			return routes, nil
		}
	}

	// Fall back to basic nginx metrics — aggregate by pod (best available grouping)
	result, _, err = c.api.Query(ctx,
		`sum by (pod) (rate(nginx_http_requests_total[5m]))`,
		end,
	)
	if err != nil {
		return nil, fmt.Errorf("query route metrics: %w", err)
	}

	vec, ok := result.(model.Vector)
	if !ok {
		return nil, nil
	}

	var routes []RouteMetrics
	for _, s := range vec {
		routes = append(routes, RouteMetrics{
			Namespace:      "nginx-gateway",
			Name:           string(s.Metric["pod"]),
			Hostname:       "*",
			RequestsPerSec: float64(s.Value),
		})
	}
	return routes, nil
}

// ByGateway returns per-gateway metrics, optionally filtered by cluster.
//
// With rich NGF metrics, gateways are broken out by gateway_namespace/gateway_name
// labels. With basic nginx metrics, we group by the pod's app label to identify
// which gateway instance the traffic flows through.
func (c *Client) ByGateway(ctx context.Context, end time.Time, clusterName string) ([]GatewayMetrics, error) {
	cs := clusterSelector(clusterName)

	// Try rich labels first
	result, _, err := c.api.Query(ctx,
		fmt.Sprintf(`sum by (gateway_namespace, gateway_name) (rate(nginx_gateway_fabric_http_requests_total{%s}[5m]))`, cs),
		end,
	)
	if err == nil {
		if vec, ok := result.(model.Vector); ok && len(vec) > 0 {
			var gateways []GatewayMetrics
			for _, s := range vec {
				gateways = append(gateways, GatewayMetrics{
					Namespace:      string(s.Metric["gateway_namespace"]),
					Name:           string(s.Metric["gateway_name"]),
					RequestsPerSec: float64(s.Value),
				})
			}
			return gateways, nil
		}
	}

	// Fall back to basic nginx metrics — group by app label
	result, _, err = c.api.Query(ctx,
		`sum by (app) (rate(nginx_http_requests_total[5m]))`,
		end,
	)
	if err != nil {
		return nil, fmt.Errorf("query gateway metrics: %w", err)
	}

	vec, ok := result.(model.Vector)
	if !ok {
		return nil, nil
	}

	var gateways []GatewayMetrics
	for _, s := range vec {
		name := string(s.Metric["app"])
		gateways = append(gateways, GatewayMetrics{
			Namespace:      "nginx-gateway",
			Name:           name,
			RequestsPerSec: float64(s.Value),
		})
	}

	// Also try to get active connections from basic metrics
	for i := range gateways {
		if val, err := c.queryScalar(ctx, `sum(nginx_http_connection_count_connections)`, end); err == nil {
			gateways[i].ActiveConnections = val
		}
	}

	return gateways, nil
}

// queryScalar executes a Prometheus query and returns a single scalar value.
func (c *Client) queryScalar(ctx context.Context, query string, t time.Time) (float64, error) {
	result, _, err := c.api.Query(ctx, query, t)
	if err != nil {
		return 0, err
	}
	switch v := result.(type) {
	case model.Vector:
		if len(v) > 0 {
			return float64(v[0].Value), nil
		}
	case *model.Scalar:
		return float64(v.Value), nil
	}
	return 0, fmt.Errorf("no data for query: %s", query)
}
