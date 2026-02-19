package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestInferenceMetricsHandler_Summary_HappyPath(t *testing.T) {
	handler := newInferenceMetricsHandler()

	r := chi.NewRouter()
	r.Get("/inference/metrics/summary", handler.Summary)

	req := httptest.NewRequest(http.MethodGet, "/inference/metrics/summary", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp InferenceMetricsSummaryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.AvgTTFT <= 0 {
		t.Errorf("expected avgTTFT > 0, got %f", resp.AvgTTFT)
	}
}

func TestInferenceMetricsHandler_PodMetrics_MissingPool(t *testing.T) {
	handler := newInferenceMetricsHandler()

	r := chi.NewRouter()
	r.Get("/inference/metrics/pods", handler.PodMetrics)

	req := httptest.NewRequest(http.MethodGet, "/inference/metrics/pods", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInferenceMetricsHandler_PodMetrics_HappyPath(t *testing.T) {
	handler := newInferenceMetricsHandler()

	r := chi.NewRouter()
	r.Get("/inference/metrics/pods", handler.PodMetrics)

	req := httptest.NewRequest(http.MethodGet, "/inference/metrics/pods?pool=llama3-70b-prod", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []PodGPUMetricsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
}

func TestInferenceMetricsHandler_Cost_HappyPath(t *testing.T) {
	handler := newInferenceMetricsHandler()

	r := chi.NewRouter()
	r.Get("/inference/metrics/cost", handler.Cost)

	req := httptest.NewRequest(http.MethodGet, "/inference/metrics/cost?pool=llama3-70b-prod", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp CostEstimateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
}

func TestInferenceMetricsHandler_EPPDecisions_HappyPath(t *testing.T) {
	handler := newInferenceMetricsHandler()

	r := chi.NewRouter()
	r.Get("/inference/metrics/epp", handler.EPPDecisions)

	req := httptest.NewRequest(http.MethodGet, "/inference/metrics/epp?pool=llama3-70b-prod", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []EPPDecisionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
}

func TestInferenceMetricsHandler_TTFTHistogram_HappyPath(t *testing.T) {
	handler := newInferenceMetricsHandler()

	r := chi.NewRouter()
	r.Get("/inference/metrics/pool/{pool}/ttft", handler.TTFTHistogram)

	req := httptest.NewRequest(http.MethodGet, "/inference/metrics/pool/test-pool/ttft", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []HistogramBucketResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
}

func TestInferenceMetricsHandler_ByPool_HappyPath(t *testing.T) {
	handler := newInferenceMetricsHandler()

	r := chi.NewRouter()
	r.Get("/inference/metrics/by-pool", handler.ByPool)

	req := httptest.NewRequest(http.MethodGet, "/inference/metrics/by-pool", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]InferenceMetricsSummaryResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp) == 0 {
		t.Fatal("expected at least one pool in by-pool response")
	}
}
