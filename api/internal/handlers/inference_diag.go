package handlers

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"time"

	ch "github.com/kubenetlabs/ngc/api/internal/clickhouse"
)

// InferenceDiagHandler handles inference diagnostics endpoints.
type InferenceDiagHandler struct {
	CHClient *ch.Client
}

// SlowInferenceResponse is the response for the slow inference diagnostics endpoint.
type SlowInferenceResponse struct {
	Pool         string        `json:"pool"`
	TimeRange    string        `json:"timeRange"`
	SlowRequests []SlowRequest `json:"slowRequests"`
	Correlations []Correlation `json:"correlations"`
}

// SlowRequest represents a single slow inference request.
type SlowRequest struct {
	RequestID   string  `json:"requestId"`
	Model       string  `json:"model"`
	TTFTMs      float64 `json:"ttftMs"`
	TotalMs     float64 `json:"totalMs"`
	InputTokens int     `json:"inputTokens"`
	QueueDepth  int     `json:"queueDepth"`
	GPUUtil     float64 `json:"gpuUtil"`
	KVCacheUtil float64 `json:"kvCacheUtil"`
	Timestamp   string  `json:"timestamp"`
}

// Correlation represents a correlation between a factor and slow inference.
type Correlation struct {
	Factor      string  `json:"factor"`      // "queue_depth", "gpu_util", "kv_cache", "input_length"
	Correlation float64 `json:"correlation"` // -1.0 to 1.0
	Impact      string  `json:"impact"`      // "high", "medium", "low"
}

// ReplayRequest is the request body for the replay endpoint.
type ReplayRequest struct {
	RequestID string `json:"requestId"`
	Pool      string `json:"pool"`
}

// ReplayResponse is the response for the replay endpoint.
type ReplayResponse struct {
	RequestID  string       `json:"requestId"`
	OriginalMs float64      `json:"originalMs"`
	ReplayMs   float64      `json:"replayMs"`
	Steps      []ReplayStep `json:"steps"`
}

// ReplayStep represents a step in the request lifecycle replay.
type ReplayStep struct {
	Name       string  `json:"name"`
	DurationMs float64 `json:"durationMs"`
	Details    string  `json:"details"`
}

// BenchmarkRequest is the request body for the benchmark endpoint.
type BenchmarkRequest struct {
	Pool            string `json:"pool"`
	ConcurrentUsers int    `json:"concurrentUsers"`
	DurationSec     int    `json:"durationSec"`
	PromptTokens    int    `json:"promptTokens"`
}

// BenchmarkResponse is the response for the benchmark endpoint.
type BenchmarkResponse struct {
	Pool          string           `json:"pool"`
	DurationSec   int              `json:"durationSec"`
	TotalRequests int              `json:"totalRequests"`
	SuccessRate   float64          `json:"successRate"`
	AvgTTFTMs     float64          `json:"avgTtftMs"`
	P50TTFTMs     float64          `json:"p50TtftMs"`
	P95TTFTMs     float64          `json:"p95TtftMs"`
	P99TTFTMs     float64          `json:"p99TtftMs"`
	AvgThroughput float64          `json:"avgThroughput"` // tokens/sec
	Errors        []BenchmarkError `json:"errors"`
}

// BenchmarkError represents a class of errors encountered during the benchmark.
type BenchmarkError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Count   int    `json:"count"`
}

// SlowInference returns information about slow inference requests.
func (h *InferenceDiagHandler) SlowInference(w http.ResponseWriter, r *http.Request) {
	pool := r.URL.Query().Get("pool")
	if pool == "" {
		pool = "default-pool"
	}

	timeRange := r.URL.Query().Get("timeRange")
	if timeRange == "" {
		timeRange = "1h"
	}

	// Generate deterministic mock data using pool name as seed.
	seed := int64(0)
	for _, c := range pool {
		seed += int64(c)
	}
	rng := rand.New(rand.NewSource(seed))

	models := []string{"llama-3.1-70b", "llama-3.1-8b", "mistral-7b", "qwen-2.5-72b"}
	now := time.Now().UTC()

	slowRequests := make([]SlowRequest, 0, 8)
	for i := 0; i < 8; i++ {
		queueDepth := rng.Intn(20) + 5
		gpuUtil := 0.7 + rng.Float64()*0.29
		kvCache := 0.5 + rng.Float64()*0.45
		inputTokens := 500 + rng.Intn(3500)

		// TTFT correlates with queue depth and GPU util
		ttft := 80.0 + float64(queueDepth)*12.0 + gpuUtil*50.0 + rng.Float64()*30.0
		total := ttft + float64(inputTokens)*0.08 + rng.Float64()*100.0

		slowRequests = append(slowRequests, SlowRequest{
			RequestID:   fmt.Sprintf("req-%s-%04d", pool, 1000+i),
			Model:       models[rng.Intn(len(models))],
			TTFTMs:      math.Round(ttft*100) / 100,
			TotalMs:     math.Round(total*100) / 100,
			InputTokens: inputTokens,
			QueueDepth:  queueDepth,
			GPUUtil:     math.Round(gpuUtil*1000) / 1000,
			KVCacheUtil: math.Round(kvCache*1000) / 1000,
			Timestamp:   now.Add(-time.Duration(rng.Intn(3600)) * time.Second).Format(time.RFC3339),
		})
	}

	correlations := []Correlation{
		{
			Factor:      "queue_depth",
			Correlation: 0.87,
			Impact:      "high",
		},
		{
			Factor:      "gpu_util",
			Correlation: 0.62,
			Impact:      "medium",
		},
		{
			Factor:      "kv_cache",
			Correlation: 0.71,
			Impact:      "high",
		},
		{
			Factor:      "input_length",
			Correlation: 0.45,
			Impact:      "medium",
		},
	}

	writeJSON(w, http.StatusOK, SlowInferenceResponse{
		Pool:         pool,
		TimeRange:    timeRange,
		SlowRequests: slowRequests,
		Correlations: correlations,
	})
}

// Replay replays a recorded inference request for debugging.
func (h *InferenceDiagHandler) Replay(w http.ResponseWriter, r *http.Request) {
	var req ReplayRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.RequestID == "" {
		writeError(w, http.StatusBadRequest, "requestId is required")
		return
	}

	if req.Pool == "" {
		req.Pool = "default-pool"
	}

	// Generate deterministic replay data based on requestID.
	seed := int64(0)
	for _, c := range req.RequestID {
		seed += int64(c)
	}
	rng := rand.New(rand.NewSource(seed))

	eppRouting := 1.5 + rng.Float64()*2.0
	queueWait := 10.0 + rng.Float64()*20.0
	modelLoading := 30.0 + rng.Float64()*40.0
	inference := 150.0 + rng.Float64()*100.0
	responseStreaming := 20.0 + rng.Float64()*20.0

	originalTotal := eppRouting + queueWait + modelLoading + inference + responseStreaming
	// Replay is slightly faster (no real queue wait, model already loaded)
	replayTotal := eppRouting + (queueWait * 0.1) + (modelLoading * 0.2) + inference + responseStreaming

	steps := []ReplayStep{
		{
			Name:       "EPP Routing",
			DurationMs: math.Round(eppRouting*100) / 100,
			Details:    fmt.Sprintf("Endpoint Picker selected backend in pool %q based on least-load strategy", req.Pool),
		},
		{
			Name:       "Queue Wait",
			DurationMs: math.Round(queueWait*100) / 100,
			Details:    fmt.Sprintf("Request waited in queue behind %d pending requests", 3+rng.Intn(10)),
		},
		{
			Name:       "Model Loading",
			DurationMs: math.Round(modelLoading*100) / 100,
			Details:    "Model weights verified in GPU memory, KV cache allocated",
		},
		{
			Name:       "Inference",
			DurationMs: math.Round(inference*100) / 100,
			Details:    fmt.Sprintf("Generated %d tokens at %.1f tokens/sec", 150+rng.Intn(350), 40.0+rng.Float64()*30.0),
		},
		{
			Name:       "Response Streaming",
			DurationMs: math.Round(responseStreaming*100) / 100,
			Details:    "Streamed response chunks back to client via SSE",
		},
	}

	writeJSON(w, http.StatusOK, ReplayResponse{
		RequestID:  req.RequestID,
		OriginalMs: math.Round(originalTotal*100) / 100,
		ReplayMs:   math.Round(replayTotal*100) / 100,
		Steps:      steps,
	})
}

// Benchmark runs a benchmark against an inference pool.
func (h *InferenceDiagHandler) Benchmark(w http.ResponseWriter, r *http.Request) {
	var req BenchmarkRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.Pool == "" {
		writeError(w, http.StatusBadRequest, "pool is required")
		return
	}

	if req.ConcurrentUsers <= 0 {
		req.ConcurrentUsers = 10
	}
	if req.DurationSec <= 0 {
		req.DurationSec = 30
	}
	if req.PromptTokens <= 0 {
		req.PromptTokens = 128
	}

	// Generate deterministic benchmark results based on request parameters.
	seed := int64(req.ConcurrentUsers*1000 + req.DurationSec*100 + req.PromptTokens)
	for _, c := range req.Pool {
		seed += int64(c)
	}
	rng := rand.New(rand.NewSource(seed))

	// Scale total requests by concurrent users and duration.
	// Assume roughly 1 request per user per 2 seconds.
	totalRequests := req.ConcurrentUsers * req.DurationSec / 2

	// Higher concurrency increases latency.
	concurrencyFactor := 1.0 + float64(req.ConcurrentUsers-1)*0.03
	// More prompt tokens increases latency.
	tokenFactor := 1.0 + float64(req.PromptTokens-128)*0.002

	baseTTFT := 45.0 * concurrencyFactor * tokenFactor
	avgTTFT := baseTTFT + rng.Float64()*10.0
	p50TTFT := avgTTFT * 0.85
	p95TTFT := avgTTFT * 2.1
	p99TTFT := avgTTFT * 3.5

	// Success rate decreases slightly with higher concurrency.
	successRate := math.Max(0.95, 1.0-float64(req.ConcurrentUsers)*0.001-rng.Float64()*0.005)

	// Throughput in tokens/sec.
	avgThroughput := float64(req.PromptTokens) * float64(req.ConcurrentUsers) * 0.8 / concurrencyFactor

	// Calculate error distribution.
	errorCount := totalRequests - int(float64(totalRequests)*successRate)
	errors := make([]BenchmarkError, 0)
	if errorCount > 0 {
		timeoutCount := int(float64(errorCount) * 0.6)
		overloadCount := errorCount - timeoutCount
		if timeoutCount > 0 {
			errors = append(errors, BenchmarkError{
				Code:    504,
				Message: "Gateway Timeout",
				Count:   timeoutCount,
			})
		}
		if overloadCount > 0 {
			errors = append(errors, BenchmarkError{
				Code:    503,
				Message: "Service Unavailable (pool overloaded)",
				Count:   overloadCount,
			})
		}
	}

	writeJSON(w, http.StatusOK, BenchmarkResponse{
		Pool:          req.Pool,
		DurationSec:   req.DurationSec,
		TotalRequests: totalRequests,
		SuccessRate:   math.Round(successRate*10000) / 10000,
		AvgTTFTMs:     math.Round(avgTTFT*100) / 100,
		P50TTFTMs:     math.Round(p50TTFT*100) / 100,
		P95TTFTMs:     math.Round(p95TTFT*100) / 100,
		P99TTFTMs:     math.Round(p99TTFT*100) / 100,
		AvgThroughput: math.Round(avgThroughput*100) / 100,
		Errors:        errors,
	})
}
