package inference

import (
	"context"
	"testing"
)

func TestListPools(t *testing.T) {
	p := NewMockProvider()
	pools, err := p.ListPools(context.Background())
	if err != nil {
		t.Fatalf("ListPools returned error: %v", err)
	}
	if len(pools) != 4 {
		t.Fatalf("expected 4 pools, got %d", len(pools))
	}
	for _, pool := range pools {
		if pool.Name == "" {
			t.Error("pool has empty name")
		}
		if pool.GPUType == "" {
			t.Error("pool has empty GPU type")
		}
	}
}

func TestGetPool(t *testing.T) {
	p := NewMockProvider()

	pool, err := p.GetPool(context.Background(), "llama3-70b-prod")
	if err != nil {
		t.Fatalf("GetPool returned error: %v", err)
	}
	if pool.Name != "llama3-70b-prod" {
		t.Errorf("expected name llama3-70b-prod, got %s", pool.Name)
	}
	if pool.GPUType != "H100" {
		t.Errorf("expected GPU type H100, got %s", pool.GPUType)
	}

	_, err = p.GetPool(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent pool, got nil")
	}
}

func TestGetMetricsSummary(t *testing.T) {
	p := NewMockProvider()
	s, err := p.GetMetricsSummary(context.Background(), "llama3-70b-prod")
	if err != nil {
		t.Fatalf("GetMetricsSummary returned error: %v", err)
	}
	if s.AvgTTFT < 0 || s.AvgTTFT > 500 {
		t.Errorf("AvgTTFT out of range: %f", s.AvgTTFT)
	}
	if s.AvgGPUUtil < 0 || s.AvgGPUUtil > 100 {
		t.Errorf("AvgGPUUtil out of range: %f", s.AvgGPUUtil)
	}
}

func TestGetPodMetrics(t *testing.T) {
	p := NewMockProvider()
	pods, err := p.GetPodMetrics(context.Background(), "llama3-70b-prod")
	if err != nil {
		t.Fatalf("GetPodMetrics returned error: %v", err)
	}
	if len(pods) != 6 {
		t.Fatalf("expected 6 pods for llama3-70b-prod, got %d", len(pods))
	}
	for _, pod := range pods {
		if pod.GPUUtilPct < 0 || pod.GPUUtilPct > 100 {
			t.Errorf("GPU util out of range for %s: %f", pod.PodName, pod.GPUUtilPct)
		}
	}

	_, err = p.GetPodMetrics(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent pool")
	}
}

func TestGetRecentEPPDecisions(t *testing.T) {
	p := NewMockProvider()
	decisions, err := p.GetRecentEPPDecisions(context.Background(), "llama3-70b-prod", 10)
	if err != nil {
		t.Fatalf("GetRecentEPPDecisions returned error: %v", err)
	}
	if len(decisions) != 10 {
		t.Fatalf("expected 10 decisions, got %d", len(decisions))
	}
	validStrategies := map[string]bool{"least_queue": true, "kv_cache": true, "prefix_affinity": true, "composite": true}
	for _, d := range decisions {
		if !validStrategies[d.Reason] {
			t.Errorf("invalid strategy: %s", d.Reason)
		}
	}
}

func TestGetTTFTHistogram(t *testing.T) {
	p := NewMockProvider()
	buckets, err := p.GetTTFTHistogram(context.Background(), "llama3-70b-prod")
	if err != nil {
		t.Fatalf("GetTTFTHistogram returned error: %v", err)
	}
	if len(buckets) != 8 {
		t.Fatalf("expected 8 buckets, got %d", len(buckets))
	}
	for _, b := range buckets {
		if b.Count < 0 {
			t.Errorf("negative count in bucket [%f, %f]: %d", b.RangeStart, b.RangeEnd, b.Count)
		}
	}
}

func TestTimeseries(t *testing.T) {
	p := NewMockProvider()
	for _, tc := range []struct {
		name string
		fn   func(context.Context, string) ([]TimeseriesPoint, error)
	}{
		{"TPSThroughput", p.GetTPSThroughput},
		{"QueueDepthSeries", p.GetQueueDepthSeries},
		{"ActiveRequestsSeries", p.GetActiveRequestsSeries},
		{"GPUUtilSeries", p.GetGPUUtilSeries},
		{"KVCacheSeries", p.GetKVCacheSeries},
	} {
		t.Run(tc.name, func(t *testing.T) {
			points, err := tc.fn(context.Background(), "llama3-70b-prod")
			if err != nil {
				t.Fatalf("returned error: %v", err)
			}
			if len(points) != 60 {
				t.Fatalf("expected 60 points, got %d", len(points))
			}
			for i := 1; i < len(points); i++ {
				if !points[i].Timestamp.After(points[i-1].Timestamp) {
					t.Error("timestamps not ascending")
					break
				}
			}
		})
	}
}

func TestGetCostEstimate(t *testing.T) {
	p := NewMockProvider()
	cost, err := p.GetCostEstimate(context.Background(), "llama3-70b-prod")
	if err != nil {
		t.Fatalf("GetCostEstimate returned error: %v", err)
	}
	if cost.GPUType != "H100" {
		t.Errorf("expected H100, got %s", cost.GPUType)
	}
	if cost.HourlyRate <= 0 {
		t.Error("expected positive hourly rate")
	}
	if cost.MonthlyCost <= cost.DailyCost {
		t.Error("monthly should be > daily")
	}

	_, err = p.GetCostEstimate(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent pool")
	}
}
