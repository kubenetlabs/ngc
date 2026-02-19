package alerting

import (
	"context"
	"testing"
	"time"

	"github.com/kubenetlabs/ngc/api/internal/database"
)

func TestMetricToPromQL_KnownMetrics(t *testing.T) {
	tests := []struct {
		metric   string
		resource string
		wantNon  bool // expect non-empty result
	}{
		{"error_rate", "my-route", true},
		{"latency_p99", "my-route", true},
		{"request_rate", "my-route", true},
		{"gpu_util", "my-pool", true},
		{"queue_depth", "my-pool", true},
		{"kv_cache_util", "my-pool", true},
		{"memory_usage", "my-pod", true},
		{"unknown_metric", "x", false},
	}

	for _, tt := range tests {
		t.Run(tt.metric, func(t *testing.T) {
			got := metricToPromQL(tt.metric, tt.resource)
			if tt.wantNon && got == "" {
				t.Errorf("metricToPromQL(%q, %q) = empty, want non-empty", tt.metric, tt.resource)
			}
			if !tt.wantNon && got != "" {
				t.Errorf("metricToPromQL(%q, %q) = %q, want empty", tt.metric, tt.resource, got)
			}
		})
	}
}

func TestMetricToPromQL_ContainsResource(t *testing.T) {
	resource := "test-resource-xyz"
	for _, metric := range []string{"error_rate", "latency_p99", "request_rate", "gpu_util", "queue_depth", "kv_cache_util", "memory_usage"} {
		got := metricToPromQL(metric, resource)
		if got == "" {
			t.Fatalf("metricToPromQL(%q, %q) returned empty", metric, resource)
		}
		if !contains(got, resource) {
			t.Errorf("metricToPromQL(%q, %q) = %q, does not contain resource name", metric, resource, got)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstr(s, substr)
}

func searchSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestSyntheticMetricValue_Ranges(t *testing.T) {
	tests := []struct {
		metric string
		min    float64
		max    float64
	}{
		{"error_rate", 0, 15},
		{"expiry_days", 0, 365},
		{"gpu_util", 0, 100},
		{"latency_p99", 0, 5000},
		{"request_rate", 0, 10000},
		{"memory_usage", 0, 100},
		{"queue_depth", 0, 500},
		{"kv_cache_util", 0, 100},
		{"unknown_metric", 0, 100},
	}

	for _, tt := range tests {
		t.Run(tt.metric, func(t *testing.T) {
			// Test with multiple resources to exercise hash diversity.
			for _, res := range []string{"res-a", "res-b", "res-c"} {
				val := syntheticMetricValue(tt.metric, res)
				if val < tt.min || val > tt.max {
					t.Errorf("syntheticMetricValue(%q, %q) = %f, want [%f, %f]", tt.metric, res, val, tt.min, tt.max)
				}
			}
		})
	}
}

func TestNew_CreatesEvaluator(t *testing.T) {
	store := database.NewMockStore()
	webhooks := []WebhookConfig{{URL: "http://example.com/webhook"}}

	eval := New(store, nil, webhooks)

	if eval.store != store {
		t.Error("store not set correctly")
	}
	if eval.prom != nil {
		t.Error("expected nil prom client")
	}
	if len(eval.webhooks) != 1 {
		t.Errorf("webhooks count = %d, want 1", len(eval.webhooks))
	}
	if eval.interval != 60*time.Second {
		t.Errorf("interval = %v, want 60s", eval.interval)
	}
}

func TestGetFiring_EmptyInitially(t *testing.T) {
	eval := New(database.NewMockStore(), nil, nil)
	firing := eval.GetFiring()

	if len(firing) != 0 {
		t.Errorf("GetFiring() returned %d alerts, want 0", len(firing))
	}
}

func TestEvaluateRule_OperatorGT(t *testing.T) {
	store := database.NewMockStore()
	ctx := context.Background()

	// Create a rule with a very low threshold so synthetic value is likely to exceed it.
	rule := database.AlertRule{
		ID:        "rule-gt-1",
		Name:      "High Error Rate",
		Resource:  "my-route",
		Metric:    "error_rate",
		Operator:  "gt",
		Threshold: -1, // any positive synthetic value will exceed this
		Severity:  "critical",
		Enabled:   true,
	}
	if err := store.CreateAlertRule(ctx, rule); err != nil {
		t.Fatal(err)
	}

	eval := New(store, nil, nil)
	eval.evaluate(ctx)

	firing := eval.GetFiring()
	if len(firing) != 1 {
		t.Fatalf("GetFiring() returned %d alerts, want 1", len(firing))
	}
	if firing[0].RuleID != "rule-gt-1" {
		t.Errorf("firing alert rule ID = %q, want %q", firing[0].RuleID, "rule-gt-1")
	}
}

func TestEvaluateRule_OperatorLT(t *testing.T) {
	store := database.NewMockStore()
	ctx := context.Background()

	// Create a rule with a threshold higher than possible synthetic value.
	rule := database.AlertRule{
		ID:        "rule-lt-1",
		Name:      "Low Queue Depth",
		Resource:  "my-pool",
		Metric:    "queue_depth",
		Operator:  "lt",
		Threshold: 999999, // synthetic queue_depth max is 500
		Severity:  "info",
		Enabled:   true,
	}
	if err := store.CreateAlertRule(ctx, rule); err != nil {
		t.Fatal(err)
	}

	eval := New(store, nil, nil)
	eval.evaluate(ctx)

	firing := eval.GetFiring()
	if len(firing) != 1 {
		t.Fatalf("GetFiring() returned %d alerts, want 1", len(firing))
	}
}

func TestEvaluateRule_DisabledRuleResolves(t *testing.T) {
	store := database.NewMockStore()
	ctx := context.Background()

	rule := database.AlertRule{
		ID:        "rule-disable-1",
		Name:      "Test Alert",
		Resource:  "test",
		Metric:    "error_rate",
		Operator:  "gt",
		Threshold: -1,
		Severity:  "warning",
		Enabled:   true,
	}
	if err := store.CreateAlertRule(ctx, rule); err != nil {
		t.Fatal(err)
	}

	eval := New(store, nil, nil) // no webhooks
	eval.evaluate(ctx)

	if len(eval.GetFiring()) != 1 {
		t.Fatal("expected alert to be firing after first evaluation")
	}

	// Disable the rule.
	rule.Enabled = false
	if err := store.UpdateAlertRule(ctx, rule); err != nil {
		t.Fatal(err)
	}

	eval.evaluate(ctx)

	if len(eval.GetFiring()) != 0 {
		t.Error("expected alert to be resolved after disabling rule")
	}
}

func TestEvaluateRule_DeletedRuleResolves(t *testing.T) {
	store := database.NewMockStore()
	ctx := context.Background()

	rule := database.AlertRule{
		ID:        "rule-delete-1",
		Name:      "Test Alert",
		Resource:  "test",
		Metric:    "error_rate",
		Operator:  "gt",
		Threshold: -1,
		Severity:  "warning",
		Enabled:   true,
	}
	if err := store.CreateAlertRule(ctx, rule); err != nil {
		t.Fatal(err)
	}

	eval := New(store, nil, nil)
	eval.evaluate(ctx)

	if len(eval.GetFiring()) != 1 {
		t.Fatal("expected alert to be firing after first evaluation")
	}

	// Delete the rule.
	if err := store.DeleteAlertRule(ctx, rule.ID); err != nil {
		t.Fatal(err)
	}

	eval.evaluate(ctx)

	if len(eval.GetFiring()) != 0 {
		t.Error("expected alert to be resolved after deleting rule")
	}
}

func TestEvaluateRule_UnknownOperator(t *testing.T) {
	store := database.NewMockStore()
	ctx := context.Background()

	rule := database.AlertRule{
		ID:        "rule-unknown-op",
		Name:      "Bad Operator",
		Resource:  "test",
		Metric:    "error_rate",
		Operator:  "gte", // unsupported
		Threshold: 0,
		Severity:  "info",
		Enabled:   true,
	}
	if err := store.CreateAlertRule(ctx, rule); err != nil {
		t.Fatal(err)
	}

	eval := New(store, nil, nil)
	eval.evaluate(ctx)

	if len(eval.GetFiring()) != 0 {
		t.Error("expected no firing alerts for unknown operator")
	}
}

func TestEvaluate_NilStore(t *testing.T) {
	eval := New(nil, nil, nil)
	// Should not panic when store is nil.
	eval.evaluate(context.Background())

	if len(eval.GetFiring()) != 0 {
		t.Error("expected no firing alerts with nil store")
	}
}
