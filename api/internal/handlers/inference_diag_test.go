package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestInferenceDiagHandler_SlowInference_Defaults(t *testing.T) {
	handler := &InferenceDiagHandler{}

	r := chi.NewRouter()
	r.Get("/inference/diagnostics/slow", handler.SlowInference)

	req := httptest.NewRequest(http.MethodGet, "/inference/diagnostics/slow", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp SlowInferenceResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Pool != "default-pool" {
		t.Errorf("expected pool default-pool, got %s", resp.Pool)
	}
	if resp.TimeRange != "1h" {
		t.Errorf("expected timeRange 1h, got %s", resp.TimeRange)
	}
	if len(resp.SlowRequests) != 8 {
		t.Errorf("expected 8 slow requests, got %d", len(resp.SlowRequests))
	}
	if len(resp.Correlations) != 4 {
		t.Errorf("expected 4 correlations, got %d", len(resp.Correlations))
	}

	// Verify correlation structure
	for _, c := range resp.Correlations {
		if c.Factor == "" {
			t.Error("expected non-empty factor")
		}
		if c.Impact == "" {
			t.Error("expected non-empty impact")
		}
	}
}

func TestInferenceDiagHandler_SlowInference_WithParams(t *testing.T) {
	handler := &InferenceDiagHandler{}

	r := chi.NewRouter()
	r.Get("/inference/diagnostics/slow", handler.SlowInference)

	req := httptest.NewRequest(http.MethodGet, "/inference/diagnostics/slow?pool=custom-pool&timeRange=24h", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp SlowInferenceResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Pool != "custom-pool" {
		t.Errorf("expected pool custom-pool, got %s", resp.Pool)
	}
	if resp.TimeRange != "24h" {
		t.Errorf("expected timeRange 24h, got %s", resp.TimeRange)
	}
}

func TestInferenceDiagHandler_Replay_HappyPath(t *testing.T) {
	handler := &InferenceDiagHandler{}

	r := chi.NewRouter()
	r.Post("/inference/diagnostics/replay", handler.Replay)

	body := ReplayRequest{
		RequestID: "req-123",
		Pool:      "test-pool",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/inference/diagnostics/replay", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp ReplayResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.RequestID != "req-123" {
		t.Errorf("expected requestId req-123, got %s", resp.RequestID)
	}
	if len(resp.Steps) != 5 {
		t.Errorf("expected 5 steps, got %d", len(resp.Steps))
	}
	if resp.OriginalMs <= resp.ReplayMs {
		t.Errorf("expected originalMs (%f) > replayMs (%f)", resp.OriginalMs, resp.ReplayMs)
	}
}

func TestInferenceDiagHandler_Replay_MissingRequestID(t *testing.T) {
	handler := &InferenceDiagHandler{}

	r := chi.NewRouter()
	r.Post("/inference/diagnostics/replay", handler.Replay)

	body := ReplayRequest{}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/inference/diagnostics/replay", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInferenceDiagHandler_Benchmark_HappyPath(t *testing.T) {
	handler := &InferenceDiagHandler{}

	r := chi.NewRouter()
	r.Post("/inference/diagnostics/benchmark", handler.Benchmark)

	body := BenchmarkRequest{
		Pool:            "test-pool",
		ConcurrentUsers: 10,
		DurationSec:     30,
		PromptTokens:    256,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/inference/diagnostics/benchmark", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp BenchmarkResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Pool != "test-pool" {
		t.Errorf("expected pool test-pool, got %s", resp.Pool)
	}
	if resp.TotalRequests <= 0 {
		t.Errorf("expected positive totalRequests, got %d", resp.TotalRequests)
	}
	if resp.SuccessRate < 0.95 {
		t.Errorf("expected successRate >= 0.95, got %f", resp.SuccessRate)
	}
	if resp.AvgTTFTMs <= 0 {
		t.Error("expected positive avgTtftMs")
	}
}

func TestInferenceDiagHandler_Benchmark_MissingPool(t *testing.T) {
	handler := &InferenceDiagHandler{}

	r := chi.NewRouter()
	r.Post("/inference/diagnostics/benchmark", handler.Benchmark)

	body := BenchmarkRequest{}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/inference/diagnostics/benchmark", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInferenceDiagHandler_Benchmark_DefaultParams(t *testing.T) {
	handler := &InferenceDiagHandler{}

	r := chi.NewRouter()
	r.Post("/inference/diagnostics/benchmark", handler.Benchmark)

	// Only pool provided, other params should use defaults
	body := BenchmarkRequest{Pool: "default-pool"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/inference/diagnostics/benchmark", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp BenchmarkResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.DurationSec != 30 {
		t.Errorf("expected default durationSec 30, got %d", resp.DurationSec)
	}
}
