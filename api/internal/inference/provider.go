package inference

import "context"

// MetricsProvider abstracts the data source for inference metrics.
// The mock implementation generates synthetic data; the ClickHouse
// implementation queries real telemetry tables.
type MetricsProvider interface {
	ListPools(ctx context.Context) ([]PoolStatus, error)
	GetPool(ctx context.Context, name string) (*PoolStatus, error)
	UpsertPool(ctx context.Context, pool PoolStatus) error
	DeletePool(ctx context.Context, name, namespace string) error
	GetMetricsSummary(ctx context.Context, pool string) (*MetricsSummary, error)
	GetPodMetrics(ctx context.Context, pool string) ([]PodMetrics, error)
	GetRecentEPPDecisions(ctx context.Context, pool string, limit int) ([]EPPDecision, error)
	GetTTFTHistogram(ctx context.Context, pool string) ([]HistogramBucket, error)
	GetTPSThroughput(ctx context.Context, pool string) ([]TimeseriesPoint, error)
	GetQueueDepthSeries(ctx context.Context, pool string) ([]TimeseriesPoint, error)
	GetGPUUtilSeries(ctx context.Context, pool string) ([]TimeseriesPoint, error)
	GetKVCacheSeries(ctx context.Context, pool string) ([]TimeseriesPoint, error)
	GetCostEstimate(ctx context.Context, pool string) (*CostEstimate, error)
}
