package clickhouse

import (
	"context"
	"fmt"
	"strings"

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
	client *Client
}

// compile-time interface check
var _ inference.MetricsProvider = (*Provider)(nil)

// NewProvider creates a ClickHouse-backed metrics provider.
func NewProvider(client *Client) *Provider {
	return &Provider{client: client}
}

func (p *Provider) ListPools(ctx context.Context) ([]inference.PoolStatus, error) {
	rows, err := p.client.Conn().Query(ctx, queryListPools)
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
	rows, err := p.client.Conn().Query(ctx, queryGetPool, name)
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

func (p *Provider) GetMetricsSummary(ctx context.Context, pool string) (*inference.MetricsSummary, error) {
	rows, err := p.client.Conn().Query(ctx, queryMetricsSummary, pool, pool)
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
	rows, err := p.client.Conn().Query(ctx, queryPodMetrics, pool)
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
	rows, err := p.client.Conn().Query(ctx, queryRecentEPPDecisions, pool, limit)
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
	rows, err := p.client.Conn().Query(ctx, queryTTFTHistogram, pool)
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
	return p.queryTimeseries(ctx, queryTPSThroughput, pool, "GetTPSThroughput")
}

func (p *Provider) GetQueueDepthSeries(ctx context.Context, pool string) ([]inference.TimeseriesPoint, error) {
	return p.queryTimeseries(ctx, queryQueueDepthSeries, pool, "GetQueueDepthSeries")
}

func (p *Provider) GetGPUUtilSeries(ctx context.Context, pool string) ([]inference.TimeseriesPoint, error) {
	return p.queryTimeseries(ctx, queryGPUUtilSeries, pool, "GetGPUUtilSeries")
}

func (p *Provider) GetKVCacheSeries(ctx context.Context, pool string) ([]inference.TimeseriesPoint, error) {
	return p.queryTimeseries(ctx, queryKVCacheSeries, pool, "GetKVCacheSeries")
}

// queryTimeseries is a helper for all timeseries queries that return (timestamp, value) rows.
func (p *Provider) queryTimeseries(ctx context.Context, query, pool, label string) ([]inference.TimeseriesPoint, error) {
	rows, err := p.client.Conn().Query(ctx, query, pool)
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

	replicaCount := ps.Replicas
	totalHourly := hourlyRate * float64(replicaCount) * float64(ps.GPUCount)

	return &inference.CostEstimate{
		GPUType:      ps.GPUType,
		ReplicaCount: replicaCount,
		HourlyRate:   totalHourly,
		DailyCost:    totalHourly * 24,
		MonthlyCost:  totalHourly * 24 * 30,
	}, nil
}
