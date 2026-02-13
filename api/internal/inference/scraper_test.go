package inference

import (
	"math"
	"testing"
	"time"
)

func TestParsePrometheusValue(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		metric   string
		wantVal  float64
		wantFound bool
	}{
		{
			name:      "simple gauge",
			body:      "vllm:num_requests_running 5\n",
			metric:    "vllm:num_requests_running",
			wantVal:   5,
			wantFound: true,
		},
		{
			name:      "gauge with labels",
			body:      `vllm:gpu_cache_usage_perc{model="llama"} 0.65` + "\n",
			metric:    "vllm:gpu_cache_usage_perc",
			wantVal:   0.65,
			wantFound: true,
		},
		{
			name: "value with optional timestamp",
			body: "vllm:generation_tokens_total 50000 1700000000\n",
			metric:    "vllm:generation_tokens_total",
			wantVal:   50000,
			wantFound: true,
		},
		{
			name: "value with labels and timestamp",
			body: `vllm:num_requests_waiting{model="llama3"} 42 1700000000` + "\n",
			metric:    "vllm:num_requests_waiting",
			wantVal:   42,
			wantFound: true,
		},
		{
			name:      "metric not found",
			body:      "vllm:num_requests_running 5\n",
			metric:    "vllm:nonexistent",
			wantVal:   0,
			wantFound: false,
		},
		{
			name: "skips comment and type lines",
			body: `# HELP vllm:num_requests_running Number of running requests
# TYPE vllm:num_requests_running gauge
vllm:num_requests_running 3
`,
			metric:    "vllm:num_requests_running",
			wantVal:   3,
			wantFound: true,
		},
		{
			name:      "doesn't match prefix",
			body:      "vllm:num_requests_running_total 100\n",
			metric:    "vllm:num_requests_running",
			wantVal:   0,
			wantFound: false,
		},
		{
			name:      "histogram sum",
			body:      "vllm:time_to_first_token_seconds_sum 123.456\n",
			metric:    "vllm:time_to_first_token_seconds_sum",
			wantVal:   123.456,
			wantFound: true,
		},
		{
			name:      "histogram count",
			body:      "vllm:time_to_first_token_seconds_count 1000\n",
			metric:    "vllm:time_to_first_token_seconds_count",
			wantVal:   1000,
			wantFound: true,
		},
		{
			name:      "float value",
			body:      "vllm:avg_generation_throughput_toks_per_s 45.67\n",
			metric:    "vllm:avg_generation_throughput_toks_per_s",
			wantVal:   45.67,
			wantFound: true,
		},
		{
			name:      "zero value is found",
			body:      "vllm:num_requests_waiting 0\n",
			metric:    "vllm:num_requests_waiting",
			wantVal:   0,
			wantFound: true,
		},
		{
			name:      "empty body",
			body:      "",
			metric:    "anything",
			wantVal:   0,
			wantFound: false,
		},
		{
			name: "multiple metrics",
			body: `vllm:num_requests_running 5
vllm:num_requests_waiting 3
vllm:gpu_cache_usage_perc 0.42
`,
			metric:    "vllm:num_requests_waiting",
			wantVal:   3,
			wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, found := parsePrometheusValue(tt.body, tt.metric)
			if found != tt.wantFound {
				t.Errorf("found = %v, want %v", found, tt.wantFound)
			}
			if found && math.Abs(val-tt.wantVal) > 0.001 {
				t.Errorf("value = %f, want %f", val, tt.wantVal)
			}
		})
	}
}

func TestFirstFound(t *testing.T) {
	body := `vllm_num_requests_running 7
vllm:num_requests_running 5
`
	// Should return the first match (vllm: prefix tried first).
	val := firstFound(body, "vllm:num_requests_running", "vllm_num_requests_running")
	if val != 5 {
		t.Errorf("expected 5 (vllm: prefix), got %f", val)
	}

	// When first name not found, falls back to second.
	val = firstFound(body, "vllm:nonexistent", "vllm_num_requests_running")
	if val != 7 {
		t.Errorf("expected 7 (fallback), got %f", val)
	}

	// When neither found, returns 0.
	val = firstFound(body, "nope", "also_nope")
	if val != 0 {
		t.Errorf("expected 0, got %f", val)
	}
}

func TestUpdateCounters_FirstObservation(t *testing.T) {
	s := &metricsScraper{counters: make(map[string]*podCounters)}

	pm := parsedPodMetrics{
		tokensTotal: 1000,
		ttftSum:     5.0,
		ttftCount:   100,
	}

	tokensDelta, ttftAvg := s.updateCounters("test/ns/pod-0", pm)

	// First observation: no delta should be computed.
	if tokensDelta != 0 {
		t.Errorf("first observation tokensDelta = %d, want 0", tokensDelta)
	}
	if ttftAvg != 0 {
		t.Errorf("first observation ttftAvg = %f, want 0", ttftAvg)
	}
}

func TestUpdateCounters_NormalDelta(t *testing.T) {
	s := &metricsScraper{counters: make(map[string]*podCounters)}
	key := "test/ns/pod-0"

	// First observation — seeds the counters.
	s.updateCounters(key, parsedPodMetrics{
		tokensTotal: 1000,
		ttftSum:     5.0,
		ttftCount:   100,
	})

	// Second observation — should compute deltas.
	tokensDelta, ttftAvg := s.updateCounters(key, parsedPodMetrics{
		tokensTotal: 1500,
		ttftSum:     7.5,
		ttftCount:   150,
	})

	if tokensDelta != 500 {
		t.Errorf("tokensDelta = %d, want 500", tokensDelta)
	}
	// ttftAvg = ((7.5 - 5.0) / (150 - 100)) * 1000 = (2.5 / 50) * 1000 = 50ms
	if math.Abs(ttftAvg-50.0) > 0.01 {
		t.Errorf("ttftAvg = %f, want 50.0", ttftAvg)
	}
}

func TestUpdateCounters_CounterReset(t *testing.T) {
	s := &metricsScraper{counters: make(map[string]*podCounters)}
	key := "test/ns/pod-0"

	// First observation with high values.
	s.updateCounters(key, parsedPodMetrics{
		tokensTotal: 10000,
		ttftSum:     50.0,
		ttftCount:   1000,
	})

	// Pod restarts — counters reset to low values.
	tokensDelta, ttftAvg := s.updateCounters(key, parsedPodMetrics{
		tokensTotal: 100,
		ttftSum:     0.5,
		ttftCount:   10,
	})

	// Should be 0, not a huge negative number.
	if tokensDelta != 0 {
		t.Errorf("counter reset tokensDelta = %d, want 0", tokensDelta)
	}
	if ttftAvg != 0 {
		t.Errorf("counter reset ttftAvg = %f, want 0", ttftAvg)
	}

	// Next normal observation after reset should work.
	tokensDelta, ttftAvg = s.updateCounters(key, parsedPodMetrics{
		tokensTotal: 200,
		ttftSum:     1.5,
		ttftCount:   30,
	})
	if tokensDelta != 100 {
		t.Errorf("post-reset tokensDelta = %d, want 100", tokensDelta)
	}
	// (1.5-0.5)/(30-10)*1000 = (1.0/20)*1000 = 50ms
	if math.Abs(ttftAvg-50.0) > 0.01 {
		t.Errorf("post-reset ttftAvg = %f, want 50.0", ttftAvg)
	}
}

func TestUpdateCounters_ZeroTokensDelta(t *testing.T) {
	s := &metricsScraper{counters: make(map[string]*podCounters)}
	key := "test/ns/pod-0"

	// Seed.
	s.updateCounters(key, parsedPodMetrics{tokensTotal: 1000})

	// Same value (no new tokens).
	tokensDelta, _ := s.updateCounters(key, parsedPodMetrics{tokensTotal: 1000})
	if tokensDelta != 0 {
		t.Errorf("same-value tokensDelta = %d, want 0", tokensDelta)
	}
}

func TestCleanupStaleCounters(t *testing.T) {
	s := &metricsScraper{counters: make(map[string]*podCounters)}

	// Create some counters with different lastSeen times.
	now := time.Now()
	s.counters["fresh"] = &podCounters{lastSeen: now}
	s.counters["stale"] = &podCounters{lastSeen: now.Add(-20 * time.Minute)}
	s.counters["borderline"] = &podCounters{lastSeen: now.Add(-9 * time.Minute)}

	s.cleanupStaleCounters(10 * time.Minute)

	if _, ok := s.counters["fresh"]; !ok {
		t.Error("fresh counter was incorrectly removed")
	}
	if _, ok := s.counters["stale"]; ok {
		t.Error("stale counter was not removed")
	}
	if _, ok := s.counters["borderline"]; !ok {
		t.Error("borderline counter was incorrectly removed")
	}
}

func TestRoundTo(t *testing.T) {
	tests := []struct {
		v      float64
		places int
		want   float64
	}{
		{1.2345, 2, 1.23},
		{1.235, 2, 1.24},
		{0, 2, 0},
		{100.0, 0, 100},
		{99.999, 1, 100.0},
	}

	for _, tt := range tests {
		got := roundTo(tt.v, tt.places)
		if math.Abs(got-tt.want) > 0.001 {
			t.Errorf("roundTo(%f, %d) = %f, want %f", tt.v, tt.places, got, tt.want)
		}
	}
}

func TestParseVLLMMetrics(t *testing.T) {
	body := `# HELP vllm:num_requests_running Number of requests currently running on GPU.
# TYPE vllm:num_requests_running gauge
vllm:num_requests_running 5
# HELP vllm:num_requests_waiting Number of requests waiting to be processed.
# TYPE vllm:num_requests_waiting gauge
vllm:num_requests_waiting 3
# HELP vllm:gpu_cache_usage_perc GPU KV-cache usage.
# TYPE vllm:gpu_cache_usage_perc gauge
vllm:gpu_cache_usage_perc 0.42
# HELP vllm:avg_generation_throughput_toks_per_s Average generation throughput in tokens/s.
# TYPE vllm:avg_generation_throughput_toks_per_s gauge
vllm:avg_generation_throughput_toks_per_s 45.67
# HELP vllm:generation_tokens_total Number of generation tokens processed.
# TYPE vllm:generation_tokens_total counter
vllm:generation_tokens_total 50000
# HELP vllm:time_to_first_token_seconds Histogram of time to first token in seconds.
# TYPE vllm:time_to_first_token_seconds histogram
vllm:time_to_first_token_seconds_sum 123.456
vllm:time_to_first_token_seconds_count 1000
`

	s := &metricsScraper{counters: make(map[string]*podCounters)}
	pm := s.parseVLLMMetrics(body, "cluster1", "pool1", "pod-0", "node-0")

	if pm.requestsInFlight != 5 {
		t.Errorf("requestsInFlight = %d, want 5", pm.requestsInFlight)
	}
	if pm.queueDepth != 3 {
		t.Errorf("queueDepth = %d, want 3", pm.queueDepth)
	}
	if math.Abs(pm.kvCachePct-42.0) > 0.01 {
		t.Errorf("kvCachePct = %f, want 42.0", pm.kvCachePct)
	}
	if math.Abs(pm.tps-45.67) > 0.01 {
		t.Errorf("tps = %f, want 45.67", pm.tps)
	}
	if math.Abs(pm.tokensTotal-50000) > 0.01 {
		t.Errorf("tokensTotal = %f, want 50000", pm.tokensTotal)
	}
	if math.Abs(pm.ttftSum-123.456) > 0.001 {
		t.Errorf("ttftSum = %f, want 123.456", pm.ttftSum)
	}
	if math.Abs(pm.ttftCount-1000) > 0.01 {
		t.Errorf("ttftCount = %f, want 1000", pm.ttftCount)
	}
}

func TestParseVLLMMetrics_OlderPrefix(t *testing.T) {
	body := `vllm_num_requests_running 8
vllm_num_requests_waiting 2
vllm_gpu_cache_usage_perc 0.55
vllm_avg_generation_throughput_toks_per_s 30.0
vllm_generation_tokens_total 10000
vllm_time_to_first_token_seconds_sum 10.0
vllm_time_to_first_token_seconds_count 200
`

	s := &metricsScraper{counters: make(map[string]*podCounters)}
	pm := s.parseVLLMMetrics(body, "c", "p", "pod", "node")

	if pm.requestsInFlight != 8 {
		t.Errorf("requestsInFlight = %d, want 8", pm.requestsInFlight)
	}
	if pm.queueDepth != 2 {
		t.Errorf("queueDepth = %d, want 2", pm.queueDepth)
	}
	if math.Abs(pm.kvCachePct-55.0) > 0.01 {
		t.Errorf("kvCachePct = %f, want 55.0", pm.kvCachePct)
	}
	if math.Abs(pm.tps-30.0) > 0.01 {
		t.Errorf("tps = %f, want 30.0", pm.tps)
	}
}
