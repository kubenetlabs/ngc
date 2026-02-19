package prometheus

import (
	"context"
	"fmt"
	"testing"
	"time"

	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// mockAPI implements promv1.API with a configurable Query function.
// All other methods return "not implemented" errors.
type mockAPI struct {
	promv1.API
	queryFn func(ctx context.Context, query string, ts time.Time, opts ...promv1.Option) (model.Value, promv1.Warnings, error)
}

func (m *mockAPI) Query(ctx context.Context, query string, ts time.Time, opts ...promv1.Option) (model.Value, promv1.Warnings, error) {
	if m.queryFn != nil {
		return m.queryFn(ctx, query, ts, opts...)
	}
	return nil, nil, fmt.Errorf("query not configured")
}

// newTestClient creates a Client with the given mock API for testing.
func newTestClient(api promv1.API) *Client {
	return &Client{api: api, url: "http://test-prometheus:9090"}
}

func TestClusterSelector_Empty(t *testing.T) {
	got := clusterSelector("")
	if got != "" {
		t.Errorf("clusterSelector(\"\") = %q, want empty", got)
	}
}

func TestClusterSelector_WithCluster(t *testing.T) {
	got := clusterSelector("prod")
	want := `cluster_name="prod",`
	if got != want {
		t.Errorf("clusterSelector(\"prod\") = %q, want %q", got, want)
	}
}

func TestQueryScalar_Vector(t *testing.T) {
	api := &mockAPI{
		queryFn: func(_ context.Context, _ string, _ time.Time, _ ...promv1.Option) (model.Value, promv1.Warnings, error) {
			return model.Vector{
				{Value: 42.5, Timestamp: model.Now()},
			}, nil, nil
		},
	}

	client := newTestClient(api)
	val, err := client.QueryScalar(context.Background(), "test_query", time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 42.5 {
		t.Errorf("QueryScalar = %f, want 42.5", val)
	}
}

func TestQueryScalar_Scalar(t *testing.T) {
	api := &mockAPI{
		queryFn: func(_ context.Context, _ string, _ time.Time, _ ...promv1.Option) (model.Value, promv1.Warnings, error) {
			return &model.Scalar{Value: 99.9, Timestamp: model.Now()}, nil, nil
		},
	}

	client := newTestClient(api)
	val, err := client.QueryScalar(context.Background(), "test_query", time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 99.9 {
		t.Errorf("QueryScalar = %f, want 99.9", val)
	}
}

func TestQueryScalar_EmptyVector(t *testing.T) {
	api := &mockAPI{
		queryFn: func(_ context.Context, _ string, _ time.Time, _ ...promv1.Option) (model.Value, promv1.Warnings, error) {
			return model.Vector{}, nil, nil
		},
	}

	client := newTestClient(api)
	_, err := client.QueryScalar(context.Background(), "test_query", time.Now())
	if err == nil {
		t.Error("expected error for empty vector, got nil")
	}
}

func TestQueryScalar_APIError(t *testing.T) {
	api := &mockAPI{
		queryFn: func(_ context.Context, _ string, _ time.Time, _ ...promv1.Option) (model.Value, promv1.Warnings, error) {
			return nil, nil, fmt.Errorf("connection refused")
		},
	}

	client := newTestClient(api)
	_, err := client.QueryScalar(context.Background(), "test_query", time.Now())
	if err == nil {
		t.Error("expected error for API failure, got nil")
	}
}

func TestSummary(t *testing.T) {
	api := &mockAPI{
		queryFn: func(_ context.Context, query string, _ time.Time, _ ...promv1.Option) (model.Value, promv1.Warnings, error) {
			// Return different values based on query content.
			var val model.SampleValue
			switch {
			case containsStr(query, "status=~\"5..\""):
				val = 0.02 // error rate
			case containsStr(query, "rate(nginx_gateway_fabric_http_requests_total"):
				val = 150.0 // request rate
			case containsStr(query, "0.50"):
				val = 45.0 // p50
			case containsStr(query, "0.95"):
				val = 120.0 // p95
			case containsStr(query, "0.99"):
				val = 250.0 // p99
			case containsStr(query, "connections_active"):
				val = 50.0
			default:
				return model.Vector{}, nil, nil
			}
			return model.Vector{{Value: val, Timestamp: model.Now()}}, nil, nil
		},
	}

	client := newTestClient(api)
	summary, err := client.Summary(context.Background(), time.Now(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if summary.RequestsPerSec == 0 {
		t.Error("RequestsPerSec should be non-zero")
	}
	if summary.ActiveConnections != 50.0 {
		t.Errorf("ActiveConnections = %f, want 50.0", summary.ActiveConnections)
	}
}

func TestSummary_WithCluster(t *testing.T) {
	var queriedWithCluster bool
	api := &mockAPI{
		queryFn: func(_ context.Context, query string, _ time.Time, _ ...promv1.Option) (model.Value, promv1.Warnings, error) {
			if containsStr(query, `cluster_name="prod"`) {
				queriedWithCluster = true
			}
			return model.Vector{{Value: 1.0, Timestamp: model.Now()}}, nil, nil
		},
	}

	client := newTestClient(api)
	_, err := client.Summary(context.Background(), time.Now(), "prod")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !queriedWithCluster {
		t.Error("expected queries to include cluster selector")
	}
}

func TestByRoute(t *testing.T) {
	api := &mockAPI{
		queryFn: func(_ context.Context, _ string, _ time.Time, _ ...promv1.Option) (model.Value, promv1.Warnings, error) {
			return model.Vector{
				{
					Metric: model.Metric{
						"httproute_namespace": "default",
						"httproute_name":      "web-route",
						"hostname":            "example.com",
					},
					Value:     12.5,
					Timestamp: model.Now(),
				},
				{
					Metric: model.Metric{
						"httproute_namespace": "production",
						"httproute_name":      "api-route",
						"hostname":            "api.example.com",
					},
					Value:     8.3,
					Timestamp: model.Now(),
				},
			}, nil, nil
		},
	}

	client := newTestClient(api)
	routes, err := client.ByRoute(context.Background(), time.Now(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(routes) != 2 {
		t.Fatalf("ByRoute returned %d routes, want 2", len(routes))
	}
	if routes[0].Namespace != "default" || routes[0].Name != "web-route" {
		t.Errorf("first route = %s/%s, want default/web-route", routes[0].Namespace, routes[0].Name)
	}
	if routes[0].RequestsPerSec != 12.5 {
		t.Errorf("first route RequestsPerSec = %f, want 12.5", routes[0].RequestsPerSec)
	}
}

func TestByGateway(t *testing.T) {
	api := &mockAPI{
		queryFn: func(_ context.Context, _ string, _ time.Time, _ ...promv1.Option) (model.Value, promv1.Warnings, error) {
			return model.Vector{
				{
					Metric: model.Metric{
						"gateway_namespace": "infra",
						"gateway_name":      "main-gw",
					},
					Value:     200.0,
					Timestamp: model.Now(),
				},
			}, nil, nil
		},
	}

	client := newTestClient(api)
	gateways, err := client.ByGateway(context.Background(), time.Now(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gateways) != 1 {
		t.Fatalf("ByGateway returned %d gateways, want 1", len(gateways))
	}
	if gateways[0].Namespace != "infra" || gateways[0].Name != "main-gw" {
		t.Errorf("gateway = %s/%s, want infra/main-gw", gateways[0].Namespace, gateways[0].Name)
	}
	if gateways[0].RequestsPerSec != 200.0 {
		t.Errorf("RequestsPerSec = %f, want 200.0", gateways[0].RequestsPerSec)
	}
}

func TestByRoute_NonVectorResult(t *testing.T) {
	api := &mockAPI{
		queryFn: func(_ context.Context, _ string, _ time.Time, _ ...promv1.Option) (model.Value, promv1.Warnings, error) {
			return &model.Scalar{Value: 1.0, Timestamp: model.Now()}, nil, nil
		},
	}

	client := newTestClient(api)
	routes, err := client.ByRoute(context.Background(), time.Now(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if routes != nil {
		t.Errorf("expected nil routes for non-vector result, got %d", len(routes))
	}
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
