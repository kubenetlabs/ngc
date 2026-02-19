package clickhouse

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"github.com/kubenetlabs/ngc/api/internal/inference"
)

// --- mock helpers ---

// mockQuerier implements Querier for testing.
type mockQuerier struct {
	queryFn func(ctx context.Context, query string, args ...any) (driver.Rows, error)
	execFn  func(ctx context.Context, query string, args ...any) error
}

func (m *mockQuerier) Query(ctx context.Context, query string, args ...any) (driver.Rows, error) {
	return m.queryFn(ctx, query, args...)
}

func (m *mockQuerier) Exec(ctx context.Context, query string, args ...any) error {
	return m.execFn(ctx, query, args...)
}

// mockColumnType implements driver.ColumnType.
type mockColumnType struct {
	name string
}

func (c mockColumnType) Name() string              { return c.name }
func (c mockColumnType) Nullable() bool             { return false }
func (c mockColumnType) ScanType() reflect.Type     { return reflect.TypeOf("") }
func (c mockColumnType) DatabaseTypeName() string   { return "String" }

// mockRows implements driver.Rows using a list of scan functions.
type mockRows struct {
	scanFuncs []func(dest ...any) error
	idx       int
	err       error
	columns   []string
}

func (r *mockRows) Next() bool {
	if r.idx < len(r.scanFuncs) {
		return true
	}
	return false
}

func (r *mockRows) Scan(dest ...any) error {
	if r.idx >= len(r.scanFuncs) {
		return errors.New("no more rows")
	}
	fn := r.scanFuncs[r.idx]
	r.idx++
	return fn(dest...)
}

func (r *mockRows) ScanStruct(dest any) error    { return nil }
func (r *mockRows) ColumnTypes() []driver.ColumnType {
	types := make([]driver.ColumnType, len(r.columns))
	for i, c := range r.columns {
		types[i] = mockColumnType{name: c}
	}
	return types
}
func (r *mockRows) Totals(dest ...any) error      { return nil }
func (r *mockRows) Columns() []string              { return r.columns }
func (r *mockRows) Close() error                   { return nil }
func (r *mockRows) Err() error                     { return r.err }

// poolScanFunc returns a scan function that writes a sample PoolStatus.
func poolScanFunc(name, ns, model string) func(dest ...any) error {
	return func(dest ...any) error {
		*dest[0].(*string) = name
		*dest[1].(*string) = ns
		*dest[2].(*string) = model
		*dest[3].(*string) = "v1"
		*dest[4].(*string) = "vllm"
		*dest[5].(*string) = "H100"
		*dest[6].(*uint32) = 8
		*dest[7].(*uint32) = 4
		*dest[8].(*uint32) = 4
		*dest[9].(*uint32) = 1
		*dest[10].(*uint32) = 8
		*dest[11].(*string) = "Running"
		*dest[12].(*time.Time) = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		return nil
	}
}

// --- Tests ---

func TestListPools(t *testing.T) {
	q := &mockQuerier{
		queryFn: func(_ context.Context, _ string, _ ...any) (driver.Rows, error) {
			return &mockRows{
				scanFuncs: []func(dest ...any) error{
					poolScanFunc("pool-a", "ns1", "llama3-70b"),
					poolScanFunc("pool-b", "ns2", "mistral-7b"),
				},
			}, nil
		},
	}
	p := NewProvider(q)

	pools, err := p.ListPools(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pools) != 2 {
		t.Fatalf("expected 2 pools, got %d", len(pools))
	}
	if pools[0].Name != "pool-a" {
		t.Errorf("expected pool-a, got %s", pools[0].Name)
	}
	if pools[1].GPUType != "H100" {
		t.Errorf("expected H100, got %s", pools[1].GPUType)
	}
}

func TestListPools_QueryError(t *testing.T) {
	q := &mockQuerier{
		queryFn: func(_ context.Context, _ string, _ ...any) (driver.Rows, error) {
			return nil, errors.New("connection lost")
		},
	}
	p := NewProvider(q)

	_, err := p.ListPools(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, errors.Unwrap(err)) && err.Error() == "" {
		t.Errorf("expected wrapped error")
	}
}

func TestListPools_Empty(t *testing.T) {
	q := &mockQuerier{
		queryFn: func(_ context.Context, _ string, _ ...any) (driver.Rows, error) {
			return &mockRows{}, nil
		},
	}
	p := NewProvider(q)

	pools, err := p.ListPools(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pools) != 0 {
		t.Fatalf("expected 0 pools, got %d", len(pools))
	}
}

func TestGetPool_Found(t *testing.T) {
	q := &mockQuerier{
		queryFn: func(_ context.Context, _ string, _ ...any) (driver.Rows, error) {
			return &mockRows{
				scanFuncs: []func(dest ...any) error{
					poolScanFunc("my-pool", "default", "llama3-70b"),
				},
			}, nil
		},
	}
	p := NewProvider(q)

	ps, err := p.GetPool(context.Background(), "my-pool")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ps.Name != "my-pool" {
		t.Errorf("expected my-pool, got %s", ps.Name)
	}
	if ps.ModelName != "llama3-70b" {
		t.Errorf("expected llama3-70b, got %s", ps.ModelName)
	}
}

func TestGetPool_NotFound(t *testing.T) {
	q := &mockQuerier{
		queryFn: func(_ context.Context, _ string, _ ...any) (driver.Rows, error) {
			return &mockRows{}, nil
		},
	}
	p := NewProvider(q)

	_, err := p.GetPool(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for missing pool")
	}
	if !contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %s", err.Error())
	}
}

func TestUpsertPool(t *testing.T) {
	called := false
	q := &mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...any) error {
			called = true
			return nil
		},
	}
	p := NewProvider(q)

	err := p.UpsertPool(context.Background(), inference.PoolStatus{
		Name:      "test-pool",
		Namespace: "default",
		GPUType:   "A100",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("exec was not called")
	}
}

func TestUpsertPool_Error(t *testing.T) {
	q := &mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...any) error {
			return errors.New("disk full")
		},
	}
	p := NewProvider(q)

	err := p.UpsertPool(context.Background(), inference.PoolStatus{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDeletePool(t *testing.T) {
	called := false
	q := &mockQuerier{
		execFn: func(_ context.Context, _ string, args ...any) error {
			called = true
			if args[0] != "pool-x" || args[1] != "ns-y" {
				t.Errorf("unexpected args: %v", args)
			}
			return nil
		},
	}
	p := NewProvider(q)

	err := p.DeletePool(context.Background(), "pool-x", "ns-y")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("exec was not called")
	}
}

func TestGetMetricsSummary(t *testing.T) {
	q := &mockQuerier{
		queryFn: func(_ context.Context, _ string, _ ...any) (driver.Rows, error) {
			return &mockRows{
				scanFuncs: []func(dest ...any) error{
					func(dest ...any) error {
						*dest[0].(*float64) = 45.5  // AvgTTFT
						*dest[1].(*float64) = 120.0  // P95TTFT
						*dest[2].(*float64) = 200.0  // P99TTFT
						*dest[3].(*float64) = 35.0   // AvgTPS
						*dest[4].(*uint64) = 100000   // TotalTokens
						*dest[5].(*float64) = 5.0    // AvgQueueDepth
						*dest[6].(*float64) = 0.65   // AvgKVCachePct
						*dest[7].(*float64) = 0.80   // PrefixCacheHitRate
						*dest[8].(*float64) = 0.75   // AvgGPUUtil
						return nil
					},
				},
			}, nil
		},
	}
	p := NewProvider(q)

	ms, err := p.GetMetricsSummary(context.Background(), "test-pool")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ms.AvgTTFT != 45.5 {
		t.Errorf("expected AvgTTFT=45.5, got %f", ms.AvgTTFT)
	}
	if ms.TotalTokens != 100000 {
		t.Errorf("expected TotalTokens=100000, got %d", ms.TotalTokens)
	}
}

func TestGetMetricsSummary_Empty(t *testing.T) {
	q := &mockQuerier{
		queryFn: func(_ context.Context, _ string, _ ...any) (driver.Rows, error) {
			return &mockRows{}, nil
		},
	}
	p := NewProvider(q)

	ms, err := p.GetMetricsSummary(context.Background(), "empty-pool")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ms.AvgTTFT != 0 {
		t.Errorf("expected zero AvgTTFT, got %f", ms.AvgTTFT)
	}
}

func TestGetPodMetrics(t *testing.T) {
	q := &mockQuerier{
		queryFn: func(_ context.Context, _ string, _ ...any) (driver.Rows, error) {
			return &mockRows{
				scanFuncs: []func(dest ...any) error{
					func(dest ...any) error {
						*dest[0].(*string) = "pod-a"
						*dest[1].(*string) = "node-1"
						*dest[2].(*uint8) = 0
						*dest[3].(*string) = "H100"
						*dest[4].(*uint16) = 3
						*dest[5].(*float64) = 0.45
						*dest[6].(*uint8) = 1
						*dest[7].(*float64) = 0.85
						*dest[8].(*uint32) = 40000
						*dest[9].(*uint32) = 81920
						*dest[10].(*uint16) = 65
						*dest[11].(*uint16) = 2
						return nil
					},
				},
			}, nil
		},
	}
	p := NewProvider(q)

	pods, err := p.GetPodMetrics(context.Background(), "test-pool")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pods) != 1 {
		t.Fatalf("expected 1 pod, got %d", len(pods))
	}
	if pods[0].PodName != "pod-a" {
		t.Errorf("expected pod-a, got %s", pods[0].PodName)
	}
	if pods[0].GPUUtilPct != 0.85 {
		t.Errorf("expected GPUUtilPct=0.85, got %f", pods[0].GPUUtilPct)
	}
}

func TestGetRecentEPPDecisions(t *testing.T) {
	now := time.Now().UTC()
	q := &mockQuerier{
		queryFn: func(_ context.Context, _ string, _ ...any) (driver.Rows, error) {
			return &mockRows{
				scanFuncs: []func(dest ...any) error{
					func(dest ...any) error {
						*dest[0].(*time.Time) = now
						*dest[1].(*string) = "req-001"
						*dest[2].(*string) = "pod-a"
						*dest[3].(*string) = "least-load"
						*dest[4].(*uint32) = 3
						*dest[5].(*float64) = 0.45
						*dest[6].(*uint8) = 1
						*dest[7].(*uint32) = 4
						*dest[8].(*uint32) = 150
						return nil
					},
				},
			}, nil
		},
	}
	p := NewProvider(q)

	decisions, err := p.GetRecentEPPDecisions(context.Background(), "test-pool", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(decisions) != 1 {
		t.Fatalf("expected 1 decision, got %d", len(decisions))
	}
	if decisions[0].RequestID != "req-001" {
		t.Errorf("expected req-001, got %s", decisions[0].RequestID)
	}
	if decisions[0].Reason != "least-load" {
		t.Errorf("expected least-load, got %s", decisions[0].Reason)
	}
}

func TestGetTTFTHistogram(t *testing.T) {
	q := &mockQuerier{
		queryFn: func(_ context.Context, _ string, _ ...any) (driver.Rows, error) {
			return &mockRows{
				scanFuncs: []func(dest ...any) error{
					func(dest ...any) error {
						*dest[0].(*float64) = 0
						*dest[1].(*float64) = 50
						*dest[2].(*uint64) = 12
						return nil
					},
					func(dest ...any) error {
						*dest[0].(*float64) = 50
						*dest[1].(*float64) = 100
						*dest[2].(*uint64) = 25
						return nil
					},
				},
			}, nil
		},
	}
	p := NewProvider(q)

	buckets, err := p.GetTTFTHistogram(context.Background(), "test-pool")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(buckets) != 2 {
		t.Fatalf("expected 2 buckets, got %d", len(buckets))
	}
	if buckets[0].Count != 12 {
		t.Errorf("expected count=12, got %d", buckets[0].Count)
	}
	if buckets[1].RangeStart != 50 {
		t.Errorf("expected rangeStart=50, got %f", buckets[1].RangeStart)
	}
}

func TestTimeseries(t *testing.T) {
	now := time.Now().UTC()

	makeTimeseriesQuerier := func() *mockQuerier {
		return &mockQuerier{
			queryFn: func(_ context.Context, _ string, _ ...any) (driver.Rows, error) {
				return &mockRows{
					scanFuncs: []func(dest ...any) error{
						func(dest ...any) error {
							*dest[0].(*time.Time) = now
							*dest[1].(*float64) = 42.5
							return nil
						},
						func(dest ...any) error {
							*dest[0].(*time.Time) = now.Add(time.Minute)
							*dest[1].(*float64) = 43.0
							return nil
						},
					},
				}, nil
			},
		}
	}

	tests := []struct {
		name string
		fn   func(*Provider) ([]inference.TimeseriesPoint, error)
	}{
		{"GetTPSThroughput", func(p *Provider) ([]inference.TimeseriesPoint, error) {
			return p.GetTPSThroughput(context.Background(), "pool")
		}},
		{"GetQueueDepthSeries", func(p *Provider) ([]inference.TimeseriesPoint, error) {
			return p.GetQueueDepthSeries(context.Background(), "pool")
		}},
		{"GetActiveRequestsSeries", func(p *Provider) ([]inference.TimeseriesPoint, error) {
			return p.GetActiveRequestsSeries(context.Background(), "pool")
		}},
		{"GetGPUUtilSeries", func(p *Provider) ([]inference.TimeseriesPoint, error) {
			return p.GetGPUUtilSeries(context.Background(), "pool")
		}},
		{"GetKVCacheSeries", func(p *Provider) ([]inference.TimeseriesPoint, error) {
			return p.GetKVCacheSeries(context.Background(), "pool")
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := NewProvider(makeTimeseriesQuerier())
			pts, err := tc.fn(p)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(pts) != 2 {
				t.Fatalf("expected 2 points, got %d", len(pts))
			}
			if pts[0].Value != 42.5 {
				t.Errorf("expected 42.5, got %f", pts[0].Value)
			}
		})
	}
}

func TestTimeseries_QueryError(t *testing.T) {
	q := &mockQuerier{
		queryFn: func(_ context.Context, _ string, _ ...any) (driver.Rows, error) {
			return nil, errors.New("timeout")
		},
	}
	p := NewProvider(q)

	_, err := p.GetTPSThroughput(context.Background(), "pool")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestGetCostEstimate_KnownGPU(t *testing.T) {
	q := &mockQuerier{
		queryFn: func(_ context.Context, _ string, _ ...any) (driver.Rows, error) {
			return &mockRows{
				scanFuncs: []func(dest ...any) error{
					poolScanFunc("cost-pool", "default", "llama3-70b"),
				},
			}, nil
		},
	}
	p := NewProvider(q)

	cost, err := p.GetCostEstimate(context.Background(), "cost-pool")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// H100 @ $3.50/hr * 4 replicas * 8 GPUs = $112/hr
	if cost.HourlyRate != 112.0 {
		t.Errorf("expected hourly rate 112.0, got %f", cost.HourlyRate)
	}
	if cost.DailyCost != 112.0*24 {
		t.Errorf("expected daily cost %f, got %f", 112.0*24, cost.DailyCost)
	}
}

func TestGetCostEstimate_UnknownGPU(t *testing.T) {
	q := &mockQuerier{
		queryFn: func(_ context.Context, _ string, _ ...any) (driver.Rows, error) {
			return &mockRows{
				scanFuncs: []func(dest ...any) error{
					func(dest ...any) error {
						*dest[0].(*string) = "pool-z"
						*dest[1].(*string) = "default"
						*dest[2].(*string) = "model"
						*dest[3].(*string) = "v1"
						*dest[4].(*string) = "triton"
						*dest[5].(*string) = "MI300X" // unknown GPU
						*dest[6].(*uint32) = 1
						*dest[7].(*uint32) = 2
						*dest[8].(*uint32) = 2
						*dest[9].(*uint32) = 1
						*dest[10].(*uint32) = 4
						*dest[11].(*string) = "Running"
						*dest[12].(*time.Time) = time.Now()
						return nil
					},
				},
			}, nil
		},
	}
	p := NewProvider(q)

	cost, err := p.GetCostEstimate(context.Background(), "pool-z")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Unknown GPU defaults to $1.00/hr * 2 replicas * 1 GPU = $2/hr
	if cost.HourlyRate != 2.0 {
		t.Errorf("expected hourly rate 2.0, got %f", cost.HourlyRate)
	}
}

func TestGetCostEstimate_PoolNotFound(t *testing.T) {
	q := &mockQuerier{
		queryFn: func(_ context.Context, _ string, _ ...any) (driver.Rows, error) {
			return &mockRows{}, nil
		},
	}
	p := NewProvider(q)

	_, err := p.GetCostEstimate(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for missing pool")
	}
}

// contains is a simple substring check helper.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
