package inference

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// mockPools defines the static set of inference pools returned by the mock.
var mockPools = []PoolStatus{
	{
		Name:           "llama3-70b-prod",
		Namespace:      "inference",
		ModelName:      "meta-llama/Llama-3-70B-Instruct",
		ModelVersion:   "v1.2",
		ServingBackend: "vllm",
		GPUType:        "H100",
		GPUCount:       4,
		Replicas:       6,
		ReadyReplicas:  6,
		MinReplicas:    2,
		MaxReplicas:    12,
		Selector:       map[string]string{"model": "llama3-70b", "env": "prod"},
		Status:         "Ready",
		CreatedAt:      time.Now().Add(-72 * time.Hour),
	},
	{
		Name:           "mixtral-8x7b-staging",
		Namespace:      "inference",
		ModelName:      "mistralai/Mixtral-8x7B-Instruct-v0.1",
		ModelVersion:   "v0.1",
		ServingBackend: "vllm",
		GPUType:        "A100",
		GPUCount:       2,
		Replicas:       3,
		ReadyReplicas:  3,
		MinReplicas:    1,
		MaxReplicas:    8,
		Selector:       map[string]string{"model": "mixtral-8x7b", "env": "staging"},
		Status:         "Ready",
		CreatedAt:      time.Now().Add(-48 * time.Hour),
	},
	{
		Name:           "phi3-mini-dev",
		Namespace:      "dev",
		ModelName:      "microsoft/Phi-3-mini-4k-instruct",
		ServingBackend: "triton",
		GPUType:        "L40S",
		GPUCount:       1,
		Replicas:       2,
		ReadyReplicas:  1,
		MinReplicas:    1,
		MaxReplicas:    4,
		Selector:       map[string]string{"model": "phi3-mini", "env": "dev"},
		Status:         "Degraded",
		CreatedAt:      time.Now().Add(-24 * time.Hour),
	},
	{
		Name:           "codellama-34b-prod",
		Namespace:      "inference",
		ModelName:      "codellama/CodeLlama-34b-Instruct-hf",
		ModelVersion:   "v2.0",
		ServingBackend: "tgi",
		GPUType:        "A100",
		GPUCount:       2,
		Replicas:       4,
		ReadyReplicas:  4,
		MinReplicas:    2,
		MaxReplicas:    8,
		Selector:       map[string]string{"model": "codellama-34b", "env": "prod"},
		Status:         "Ready",
		CreatedAt:      time.Now().Add(-120 * time.Hour),
	},
}

// MockProvider generates realistic synthetic inference metrics.
type MockProvider struct {
	rng *rand.Rand
}

// NewMockProvider returns a new mock provider.
func NewMockProvider() *MockProvider {
	return &MockProvider{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (m *MockProvider) ListPools(_ context.Context) ([]PoolStatus, error) {
	pools := make([]PoolStatus, len(mockPools))
	copy(pools, mockPools)
	for i := range pools {
		pools[i].AvgGPUUtil = m.varyFloat(72, 15)
	}
	return pools, nil
}

func (m *MockProvider) GetPool(_ context.Context, name string) (*PoolStatus, error) {
	for _, p := range mockPools {
		if p.Name == name {
			p.AvgGPUUtil = m.varyFloat(72, 15)
			return &p, nil
		}
	}
	return nil, fmt.Errorf("pool %q not found", name)
}

func (m *MockProvider) GetMetricsSummary(_ context.Context, _ string) (*MetricsSummary, error) {
	return &MetricsSummary{
		AvgTTFT:            m.varyFloat(120, 40),
		P95TTFT:            m.varyFloat(280, 60),
		P99TTFT:            m.varyFloat(450, 80),
		AvgTPS:             m.varyFloat(85, 20),
		TotalTokens:        int64(m.varyFloat(2_500_000, 500_000)),
		AvgQueueDepth:      m.varyFloat(4.5, 2),
		AvgKVCachePct:      m.varyFloat(62, 15),
		PrefixCacheHitRate: m.varyFloat(0.35, 0.1),
		AvgGPUUtil:         m.varyFloat(72, 12),
	}, nil
}

func (m *MockProvider) GetPodMetrics(_ context.Context, pool string) ([]PodMetrics, error) {
	p := m.findPool(pool)
	if p == nil {
		return nil, fmt.Errorf("pool %q not found", pool)
	}

	pods := make([]PodMetrics, p.Replicas)
	for i := range pods {
		pods[i] = PodMetrics{
			PodName:          fmt.Sprintf("%s-pod-%d", pool, i),
			NodeName:         fmt.Sprintf("gpu-node-%d", i%3),
			GPUID:            i % p.GPUCount,
			GPUType:          p.GPUType,
			QueueDepth:       m.varyInt(5, 4),
			KVCacheUtilPct:   m.varyFloat(60, 20),
			PrefixCacheState: m.rng.Float64() > 0.6,
			GPUUtilPct:       m.varyFloat(70, 20),
			GPUMemUsedMB:     m.varyInt(60000, 15000),
			GPUMemTotalMB:    81920,
			GPUTemperatureC:  m.varyInt(65, 10),
			RequestsInFlight: m.varyInt(3, 2),
		}
	}
	return pods, nil
}

func (m *MockProvider) GetRecentEPPDecisions(_ context.Context, pool string, limit int) ([]EPPDecision, error) {
	if limit <= 0 {
		limit = 20
	}
	strategies := []string{"least_queue", "kv_cache", "prefix_affinity", "composite"}
	p := m.findPool(pool)
	replicas := 4
	if p != nil {
		replicas = p.Replicas
	}

	decisions := make([]EPPDecision, limit)
	for i := range decisions {
		decisions[i] = EPPDecision{
			Timestamp:            time.Now().Add(-time.Duration(i) * time.Second),
			RequestID:            fmt.Sprintf("req-%s-%04d", pool, m.rng.Intn(10000)),
			SelectedPod:          fmt.Sprintf("%s-pod-%d", pool, m.rng.Intn(replicas)),
			Reason:               strategies[m.rng.Intn(len(strategies))],
			QueueDepth:           m.varyInt(4, 3),
			KVCachePct:           m.varyFloat(55, 20),
			PrefixCacheHit:       m.rng.Float64() > 0.65,
			CandidatesConsidered: replicas,
			DecisionLatencyUs:    m.varyInt(150, 80),
		}
	}
	return decisions, nil
}

func (m *MockProvider) GetTTFTHistogram(_ context.Context, _ string) ([]HistogramBucket, error) {
	// Log-normal-ish distribution of TTFT values.
	buckets := []HistogramBucket{
		{RangeStart: 0, RangeEnd: 50, Count: m.varyInt(45, 15)},
		{RangeStart: 50, RangeEnd: 100, Count: m.varyInt(120, 30)},
		{RangeStart: 100, RangeEnd: 150, Count: m.varyInt(200, 40)},
		{RangeStart: 150, RangeEnd: 200, Count: m.varyInt(170, 35)},
		{RangeStart: 200, RangeEnd: 250, Count: m.varyInt(100, 25)},
		{RangeStart: 250, RangeEnd: 300, Count: m.varyInt(55, 15)},
		{RangeStart: 300, RangeEnd: 400, Count: m.varyInt(30, 10)},
		{RangeStart: 400, RangeEnd: 500, Count: m.varyInt(12, 5)},
	}
	return buckets, nil
}

func (m *MockProvider) GetTPSThroughput(_ context.Context, _ string) ([]TimeseriesPoint, error) {
	return m.generateTimeseries(60, 85, 25), nil
}

func (m *MockProvider) GetQueueDepthSeries(_ context.Context, _ string) ([]TimeseriesPoint, error) {
	return m.generateTimeseries(60, 5, 4), nil
}

func (m *MockProvider) GetGPUUtilSeries(_ context.Context, _ string) ([]TimeseriesPoint, error) {
	return m.generateTimeseries(60, 72, 15), nil
}

func (m *MockProvider) GetKVCacheSeries(_ context.Context, _ string) ([]TimeseriesPoint, error) {
	return m.generateTimeseries(60, 60, 18), nil
}

func (m *MockProvider) GetCostEstimate(_ context.Context, pool string) (*CostEstimate, error) {
	p := m.findPool(pool)
	if p == nil {
		return nil, fmt.Errorf("pool %q not found", pool)
	}

	rates := map[string]float64{
		"H100": 3.50,
		"A100": 2.10,
		"L40S": 1.20,
		"T4":   0.55,
	}
	hourly := rates[p.GPUType] * float64(p.GPUCount)
	return &CostEstimate{
		GPUType:      p.GPUType,
		ReplicaCount: p.Replicas,
		HourlyRate:   hourly * float64(p.Replicas),
		DailyCost:    hourly * float64(p.Replicas) * 24,
		MonthlyCost:  hourly * float64(p.Replicas) * 24 * 30,
	}, nil
}

// --- helpers ---

func (m *MockProvider) findPool(name string) *PoolStatus {
	for _, p := range mockPools {
		if p.Name == name {
			return &p
		}
	}
	return nil
}

func (m *MockProvider) varyFloat(base, spread float64) float64 {
	v := base + (m.rng.Float64()*2-1)*spread
	return math.Round(v*100) / 100
}

func (m *MockProvider) varyInt(base, spread int) int {
	v := base + int((m.rng.Float64()*2-1)*float64(spread))
	if v < 0 {
		return 0
	}
	return v
}

func (m *MockProvider) generateTimeseries(points int, base, spread float64) []TimeseriesPoint {
	now := time.Now()
	ts := make([]TimeseriesPoint, points)
	for i := range ts {
		ts[i] = TimeseriesPoint{
			Timestamp: now.Add(-time.Duration(points-i) * time.Minute),
			Value:     m.varyFloat(base, spread),
		}
	}
	return ts
}
