package clickhouse

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
	"github.com/kubenetlabs/ngc/api/internal/inference"
)

// GPU hourly pricing used for cost estimation.
var gpuPricing = map[string]float64{
	"H100": 3.50,
	"A100": 2.21,
	"L40S": 1.14,
	"T4":   0.35,
}

// Provider implements inference.MetricsProvider backed by ClickHouse.
type Provider struct {
	conn Querier
}

// compile-time interface check
var _ inference.MetricsProvider = (*Provider)(nil)

// NewProvider creates a ClickHouse-backed metrics provider from a Querier.
func NewProvider(q Querier) *Provider {
	return &Provider{conn: q}
}

// NewProviderFromClient creates a Provider from a *Client for backward compatibility.
func NewProviderFromClient(client *Client) *Provider {
	return NewProvider(client.Conn())
}

// clusterFilter extracts the cluster name from context for ClickHouse filtering.
// Returns "" when no cluster is specified (matches all clusters).
func clusterFilter(ctx context.Context) string {
	return cluster.ClusterNameFromContext(ctx)
}

func (p *Provider) ListPools(ctx context.Context) ([]inference.PoolStatus, error) {
	cn := clusterFilter(ctx)
	rows, err := p.conn.Query(ctx, queryListPools, cn, cn)
	if err != nil {
		return nil, fmt.Errorf("ListPools query: %w", err)
	}
	defer rows.Close()

	var pools []inference.PoolStatus
	for rows.Next() {
		var ps inference.PoolStatus
		if err := rows.Scan(
			&ps.Name, &ps.Namespace, &ps.ModelName, &ps.ModelVersion,
			&ps.ServingBackend, &ps.GPUType, &ps.GPUCount,
			&ps.Replicas, &ps.ReadyReplicas, &ps.MinReplicas, &ps.MaxReplicas,
			&ps.Status, &ps.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("ListPools scan: %w", err)
		}
		pools = append(pools, ps)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("ListPools rows: %w", err)
	}
	return pools, nil
}

func (p *Provider) GetPool(ctx context.Context, name string) (*inference.PoolStatus, error) {
	cn := clusterFilter(ctx)
	rows, err := p.conn.Query(ctx, queryGetPool, name, cn, cn)
	if err != nil {
		return nil, fmt.Errorf("GetPool query: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("GetPool rows: %w", err)
		}
		return nil, fmt.Errorf("pool %q not found", name)
	}

	var ps inference.PoolStatus
	if err := rows.Scan(
		&ps.Name, &ps.Namespace, &ps.ModelName, &ps.ModelVersion,
		&ps.ServingBackend, &ps.GPUType, &ps.GPUCount,
		&ps.Replicas, &ps.ReadyReplicas, &ps.MinReplicas, &ps.MaxReplicas,
		&ps.Status, &ps.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("GetPool scan: %w", err)
	}
	return &ps, nil
}

func (p *Provider) UpsertPool(ctx context.Context, pool inference.PoolStatus) error {
	cn := clusterFilter(ctx)
	// ClickHouse ReplacingMergeTree will handle deduplication on (name, namespace).
	err := p.conn.Exec(ctx, queryUpsertPool,
		pool.Name, pool.Namespace, pool.ModelName, pool.ModelVersion,
		pool.ServingBackend, pool.GPUType, pool.GPUCount,
		pool.Replicas, pool.ReadyReplicas, pool.MinReplicas, pool.MaxReplicas,
		pool.Status, cn, pool.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("UpsertPool: %w", err)
	}
	return nil
}

func (p *Provider) DeletePool(ctx context.Context, name, namespace string) error {
	err := p.conn.Exec(ctx, queryDeletePool, name, namespace)
	if err != nil {
		return fmt.Errorf("DeletePool: %w", err)
	}
	return nil
}

func (p *Provider) GetMetricsSummary(ctx context.Context, pool string) (*inference.MetricsSummary, error) {
	cn := clusterFilter(ctx)
	rows, err := p.conn.Query(ctx, queryMetricsSummary, pool, pool, cn, cn)
	if err != nil {
		return nil, fmt.Errorf("GetMetricsSummary query: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("GetMetricsSummary rows: %w", err)
		}
		return &inference.MetricsSummary{}, nil
	}

	var ms inference.MetricsSummary
	if err := rows.Scan(
		&ms.AvgTTFT, &ms.P95TTFT, &ms.P99TTFT,
		&ms.AvgTPS, &ms.TotalTokens,
		&ms.AvgQueueDepth, &ms.AvgKVCachePct,
		&ms.PrefixCacheHitRate, &ms.AvgGPUUtil,
	); err != nil {
		return nil, fmt.Errorf("GetMetricsSummary scan: %w", err)
	}
	return &ms, nil
}

func (p *Provider) GetPodMetrics(ctx context.Context, pool string) ([]inference.PodMetrics, error) {
	cn := clusterFilter(ctx)
	rows, err := p.conn.Query(ctx, queryPodMetrics, pool, pool, cn, cn)
	if err != nil {
		return nil, fmt.Errorf("GetPodMetrics query: %w", err)
	}
	defer rows.Close()

	var pods []inference.PodMetrics
	for rows.Next() {
		var pm inference.PodMetrics
		if err := rows.Scan(
			&pm.PodName, &pm.NodeName, &pm.GPUID, &pm.GPUType,
			&pm.QueueDepth, &pm.KVCacheUtilPct, &pm.PrefixCacheState,
			&pm.GPUUtilPct, &pm.GPUMemUsedMB, &pm.GPUMemTotalMB,
			&pm.GPUTemperatureC, &pm.RequestsInFlight,
		); err != nil {
			return nil, fmt.Errorf("GetPodMetrics scan: %w", err)
		}
		pods = append(pods, pm)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetPodMetrics rows: %w", err)
	}
	return pods, nil
}

func (p *Provider) GetRecentEPPDecisions(ctx context.Context, pool string, limit int) ([]inference.EPPDecision, error) {
	cn := clusterFilter(ctx)
	rows, err := p.conn.Query(ctx, queryRecentEPPDecisions, pool, pool, cn, cn, limit)
	if err != nil {
		return nil, fmt.Errorf("GetRecentEPPDecisions query: %w", err)
	}
	defer rows.Close()

	var decisions []inference.EPPDecision
	for rows.Next() {
		var d inference.EPPDecision
		if err := rows.Scan(
			&d.Timestamp, &d.RequestID, &d.SelectedPod, &d.Reason,
			&d.QueueDepth, &d.KVCachePct, &d.PrefixCacheHit,
			&d.CandidatesConsidered, &d.DecisionLatencyUs,
		); err != nil {
			return nil, fmt.Errorf("GetRecentEPPDecisions scan: %w", err)
		}
		decisions = append(decisions, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetRecentEPPDecisions rows: %w", err)
	}
	return decisions, nil
}

func (p *Provider) GetTTFTHistogram(ctx context.Context, pool string) ([]inference.HistogramBucket, error) {
	cn := clusterFilter(ctx)
	rows, err := p.conn.Query(ctx, queryTTFTHistogram, pool, cn, cn)
	if err != nil {
		return nil, fmt.Errorf("GetTTFTHistogram query: %w", err)
	}
	defer rows.Close()

	var buckets []inference.HistogramBucket
	for rows.Next() {
		var b inference.HistogramBucket
		if err := rows.Scan(&b.RangeStart, &b.RangeEnd, &b.Count); err != nil {
			return nil, fmt.Errorf("GetTTFTHistogram scan: %w", err)
		}
		buckets = append(buckets, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetTTFTHistogram rows: %w", err)
	}
	return buckets, nil
}

func (p *Provider) GetTPSThroughput(ctx context.Context, pool string) ([]inference.TimeseriesPoint, error) {
	cn := clusterFilter(ctx)
	return p.queryTimeseries(ctx, queryTPSThroughput, pool, cn, "GetTPSThroughput")
}

func (p *Provider) GetQueueDepthSeries(ctx context.Context, pool string) ([]inference.TimeseriesPoint, error) {
	cn := clusterFilter(ctx)
	return p.queryTimeseries(ctx, queryQueueDepthSeries, pool, cn, "GetQueueDepthSeries")
}

func (p *Provider) GetActiveRequestsSeries(ctx context.Context, pool string) ([]inference.TimeseriesPoint, error) {
	cn := clusterFilter(ctx)
	return p.queryTimeseries(ctx, queryActiveRequestsSeries, pool, cn, "GetActiveRequestsSeries")
}

func (p *Provider) GetGPUUtilSeries(ctx context.Context, pool string) ([]inference.TimeseriesPoint, error) {
	cn := clusterFilter(ctx)
	return p.queryTimeseries(ctx, queryGPUUtilSeries, pool, cn, "GetGPUUtilSeries")
}

func (p *Provider) GetKVCacheSeries(ctx context.Context, pool string) ([]inference.TimeseriesPoint, error) {
	cn := clusterFilter(ctx)
	return p.queryTimeseries(ctx, queryKVCacheSeries, pool, cn, "GetKVCacheSeries")
}

// queryTimeseries is a helper for all timeseries queries that return (timestamp, value) rows.
func (p *Provider) queryTimeseries(ctx context.Context, query, pool, clusterName, label string) ([]inference.TimeseriesPoint, error) {
	rows, err := p.conn.Query(ctx, query, pool, clusterName, clusterName)
	if err != nil {
		return nil, fmt.Errorf("%s query: %w", label, err)
	}
	defer rows.Close()

	var points []inference.TimeseriesPoint
	for rows.Next() {
		var pt inference.TimeseriesPoint
		if err := rows.Scan(&pt.Timestamp, &pt.Value); err != nil {
			return nil, fmt.Errorf("%s scan: %w", label, err)
		}
		points = append(points, pt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s rows: %w", label, err)
	}
	return points, nil
}

func (p *Provider) GetCostEstimate(ctx context.Context, pool string) (*inference.CostEstimate, error) {
	ps, err := p.GetPool(ctx, pool)
	if err != nil {
		return nil, fmt.Errorf("GetCostEstimate: %w", err)
	}

	gpuType := strings.ToUpper(ps.GPUType)
	hourlyRate, ok := gpuPricing[gpuType]
	if !ok {
		// Default to a conservative estimate for unknown GPU types.
		hourlyRate = 1.00
	}

	replicaCount := int(ps.Replicas)
	totalHourly := hourlyRate * float64(replicaCount) * float64(ps.GPUCount)

	return &inference.CostEstimate{
		GPUType:      ps.GPUType,
		ReplicaCount: replicaCount,
		HourlyRate:   totalHourly,
		DailyCost:    totalHourly * 24,
		MonthlyCost:  totalHourly * 24 * 30,
	}, nil
}

// SlowRequestRow holds a single slow inference request from ClickHouse.
type SlowRequestRow struct {
	Timestamp   time.Time
	PoolName    string
	TTFTMs      float64
	TPS         float64
	TotalTokens uint64
	QueueDepth  uint32
	KVCachePct  float64
	GPUUtilPct  float64
}

// CorrelationRow holds TTFT correlation values from ClickHouse.
type CorrelationRow struct {
	QueueDepth  float64
	GPUUtil     float64
	KVCache     float64
	InputLength float64
}

// GetSlowRequests returns the top N requests with highest TTFT in a time window.
func (p *Provider) GetSlowRequests(ctx context.Context, pool string, minutes, limit int) ([]SlowRequestRow, error) {
	cn := clusterFilter(ctx)
	rows, err := p.conn.Query(ctx, querySlowRequests, pool, pool, cn, cn, minutes, limit)
	if err != nil {
		return nil, fmt.Errorf("GetSlowRequests query: %w", err)
	}
	defer rows.Close()

	var results []SlowRequestRow
	for rows.Next() {
		var r SlowRequestRow
		if err := rows.Scan(
			&r.Timestamp, &r.PoolName, &r.TTFTMs, &r.TPS,
			&r.TotalTokens, &r.QueueDepth, &r.KVCachePct, &r.GPUUtilPct,
		); err != nil {
			return nil, fmt.Errorf("GetSlowRequests scan: %w", err)
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("GetSlowRequests rows: %w", err)
	}
	return results, nil
}

// GetTTFTCorrelations computes Pearson correlations between TTFT and other metrics.
func (p *Provider) GetTTFTCorrelations(ctx context.Context, pool string, minutes int) (*CorrelationRow, error) {
	cn := clusterFilter(ctx)
	rows, err := p.conn.Query(ctx, queryTTFTCorrelations, pool, pool, cn, cn, minutes)
	if err != nil {
		return nil, fmt.Errorf("GetTTFTCorrelations query: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("GetTTFTCorrelations rows: %w", err)
		}
		return nil, nil
	}

	var c CorrelationRow
	if err := rows.Scan(&c.QueueDepth, &c.GPUUtil, &c.KVCache, &c.InputLength); err != nil {
		return nil, fmt.Errorf("GetTTFTCorrelations scan: %w", err)
	}
	return &c, nil
}
