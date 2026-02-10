package clickhouse

import (
	"context"
	"fmt"

	"github.com/kubenetlabs/ngc/api/internal/inference"
)

// Provider implements inference.MetricsProvider backed by ClickHouse.
type Provider struct {
	client *Client
}

// compile-time interface check
var _ inference.MetricsProvider = (*Provider)(nil)

// NewProvider creates a ClickHouse-backed metrics provider.
func NewProvider(client *Client) *Provider {
	return &Provider{client: client}
}

func notConnected(queryName string) error {
	return fmt.Errorf("clickhouse not configured: %s query requires active connection", queryName)
}

func (p *Provider) ListPools(_ context.Context) ([]inference.PoolStatus, error) {
	return nil, notConnected("ListPools")
}

func (p *Provider) GetPool(_ context.Context, name string) (*inference.PoolStatus, error) {
	_ = name
	return nil, notConnected("GetPool")
}

func (p *Provider) GetMetricsSummary(_ context.Context, pool string) (*inference.MetricsSummary, error) {
	_ = pool
	return nil, notConnected("GetMetricsSummary")
}

func (p *Provider) GetPodMetrics(_ context.Context, pool string) ([]inference.PodMetrics, error) {
	_ = pool
	return nil, notConnected("GetPodMetrics")
}

func (p *Provider) GetRecentEPPDecisions(_ context.Context, pool string, limit int) ([]inference.EPPDecision, error) {
	_, _ = pool, limit
	return nil, notConnected("GetRecentEPPDecisions")
}

func (p *Provider) GetTTFTHistogram(_ context.Context, pool string) ([]inference.HistogramBucket, error) {
	_ = pool
	return nil, notConnected("GetTTFTHistogram")
}

func (p *Provider) GetTPSThroughput(_ context.Context, pool string) ([]inference.TimeseriesPoint, error) {
	_ = pool
	return nil, notConnected("GetTPSThroughput")
}

func (p *Provider) GetQueueDepthSeries(_ context.Context, pool string) ([]inference.TimeseriesPoint, error) {
	_ = pool
	return nil, notConnected("GetQueueDepthSeries")
}

func (p *Provider) GetGPUUtilSeries(_ context.Context, pool string) ([]inference.TimeseriesPoint, error) {
	_ = pool
	return nil, notConnected("GetGPUUtilSeries")
}

func (p *Provider) GetKVCacheSeries(_ context.Context, pool string) ([]inference.TimeseriesPoint, error) {
	_ = pool
	return nil, notConnected("GetKVCacheSeries")
}

func (p *Provider) GetCostEstimate(_ context.Context, pool string) (*inference.CostEstimate, error) {
	_ = pool
	return nil, notConnected("GetCostEstimate")
}
