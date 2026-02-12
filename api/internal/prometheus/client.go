package prometheus

import (
	"context"
	"fmt"
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
	client, err := promapi.NewClient(promapi.Config{Address: url})
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
func (c *Client) Summary(ctx context.Context, end time.Time, clusterName string) (*MetricsSummary, error) {
	summary := &MetricsSummary{}
	cs := clusterSelector(clusterName)

	if val, err := c.queryScalar(ctx, fmt.Sprintf(`sum(rate(nginx_gateway_fabric_http_requests_total{%s}[5m]))`, cs), end); err == nil {
		summary.RequestsPerSec = val
	}

	if val, err := c.queryScalar(ctx, fmt.Sprintf(`sum(rate(nginx_gateway_fabric_http_requests_total{%sstatus=~"5.."}[5m])) / sum(rate(nginx_gateway_fabric_http_requests_total{%s}[5m]))`, cs, cs), end); err == nil {
		summary.ErrorRate = val
	}

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

	if val, err := c.queryScalar(ctx, fmt.Sprintf(`sum(nginx_gateway_fabric_connections_active{%s})`, cs), end); err == nil {
		summary.ActiveConnections = val
	}

	return summary, nil
}

// ByRoute returns per-route RED metrics, optionally filtered by cluster.
func (c *Client) ByRoute(ctx context.Context, end time.Time, clusterName string) ([]RouteMetrics, error) {
	cs := clusterSelector(clusterName)
	result, _, err := c.api.Query(ctx,
		fmt.Sprintf(`sum by (httproute_namespace, httproute_name, hostname) (rate(nginx_gateway_fabric_http_requests_total{%s}[5m]))`, cs),
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
			Namespace:      string(s.Metric["httproute_namespace"]),
			Name:           string(s.Metric["httproute_name"]),
			Hostname:       string(s.Metric["hostname"]),
			RequestsPerSec: float64(s.Value),
		})
	}
	return routes, nil
}

// ByGateway returns per-gateway metrics, optionally filtered by cluster.
func (c *Client) ByGateway(ctx context.Context, end time.Time, clusterName string) ([]GatewayMetrics, error) {
	cs := clusterSelector(clusterName)
	result, _, err := c.api.Query(ctx,
		fmt.Sprintf(`sum by (gateway_namespace, gateway_name) (rate(nginx_gateway_fabric_http_requests_total{%s}[5m]))`, cs),
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
		gateways = append(gateways, GatewayMetrics{
			Namespace:      string(s.Metric["gateway_namespace"]),
			Name:           string(s.Metric["gateway_name"]),
			RequestsPerSec: float64(s.Value),
		})
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
