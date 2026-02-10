package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/kubenetlabs/ngc/api/internal/inference"
)

func newInferenceHandler() *InferenceHandler {
	return &InferenceHandler{Provider: inference.NewMockProvider()}
}

func newInferenceMetricsHandler() *InferenceMetricsHandler {
	return &InferenceMetricsHandler{Provider: inference.NewMockProvider()}
}

func TestInferenceHandler_ListPools(t *testing.T) {
	handler := newInferenceHandler()

	r := chi.NewRouter()
	r.Get("/inference/pools", handler.ListPools)

	req := httptest.NewRequest(http.MethodGet, "/inference/pools", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []InferencePoolResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp) != 4 {
		t.Fatalf("expected 4 pools, got %d", len(resp))
	}

	// Verify first pool has expected fields
	found := false
	for _, p := range resp {
		if p.Name == "llama3-70b-prod" {
			found = true
			if p.GPUType != "H100" {
				t.Errorf("expected GPU type H100, got %s", p.GPUType)
			}
			if p.Status == nil {
				t.Error("expected status to be populated")
			}
			if p.CreatedAt == "" {
				t.Error("expected createdAt to be set")
			}
		}
	}
	if !found {
		t.Error("expected to find llama3-70b-prod in response")
	}
}

func TestInferenceHandler_GetPool(t *testing.T) {
	handler := newInferenceHandler()

	t.Run("existing pool", func(t *testing.T) {
		r := chi.NewRouter()
		r.Get("/inference/pools/{name}", handler.GetPool)

		req := httptest.NewRequest(http.MethodGet, "/inference/pools/llama3-70b-prod", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp InferencePoolResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Name != "llama3-70b-prod" {
			t.Errorf("expected name llama3-70b-prod, got %s", resp.Name)
		}
		if resp.GPUType != "H100" {
			t.Errorf("expected GPU type H100, got %s", resp.GPUType)
		}
		if resp.ModelName == "" {
			t.Error("expected modelName to be set")
		}
	})

	t.Run("nonexistent pool", func(t *testing.T) {
		r := chi.NewRouter()
		r.Get("/inference/pools/{name}", handler.GetPool)

		req := httptest.NewRequest(http.MethodGet, "/inference/pools/nonexistent", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", w.Code)
		}

		var resp map[string]string
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp["error"] == "" {
			t.Error("expected error message")
		}
	})
}

func TestInferenceMetricsHandler_Summary(t *testing.T) {
	handler := newInferenceMetricsHandler()

	r := chi.NewRouter()
	r.Get("/inference/metrics/summary", handler.Summary)

	req := httptest.NewRequest(http.MethodGet, "/inference/metrics/summary?pool=llama3-70b-prod", nil)
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
		t.Error("expected positive AvgTTFT")
	}
	if resp.AvgGPUUtil <= 0 {
		t.Error("expected positive AvgGPUUtil")
	}
	if resp.AvgTPS <= 0 {
		t.Error("expected positive AvgTPS")
	}
}

func TestInferenceMetricsHandler_PodMetrics(t *testing.T) {
	handler := newInferenceMetricsHandler()

	t.Run("with pool param", func(t *testing.T) {
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

		if len(resp) != 6 {
			t.Fatalf("expected 6 pods, got %d", len(resp))
		}

		for _, p := range resp {
			if p.PodName == "" {
				t.Error("expected non-empty pod name")
			}
			if p.GPUUtilPct < 0 || p.GPUUtilPct > 100 {
				t.Errorf("GPU util out of range: %f", p.GPUUtilPct)
			}
		}
	})

	t.Run("missing pool param", func(t *testing.T) {
		r := chi.NewRouter()
		r.Get("/inference/metrics/pods", handler.PodMetrics)

		req := httptest.NewRequest(http.MethodGet, "/inference/metrics/pods", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})
}

func TestInferenceMetricsHandler_Cost(t *testing.T) {
	handler := newInferenceMetricsHandler()

	t.Run("with pool param", func(t *testing.T) {
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

		if resp.GPUType != "H100" {
			t.Errorf("expected H100, got %s", resp.GPUType)
		}
		if resp.HourlyRate <= 0 {
			t.Error("expected positive hourly rate")
		}
		if resp.MonthlyCost <= resp.DailyCost {
			t.Error("monthly cost should exceed daily cost")
		}
	})

	t.Run("missing pool param", func(t *testing.T) {
		r := chi.NewRouter()
		r.Get("/inference/metrics/cost", handler.Cost)

		req := httptest.NewRequest(http.MethodGet, "/inference/metrics/cost", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})
}

func TestInferenceMetricsHandler_EPPDecisions(t *testing.T) {
	handler := newInferenceMetricsHandler()

	r := chi.NewRouter()
	r.Get("/inference/metrics/epp-decisions", handler.EPPDecisions)

	req := httptest.NewRequest(http.MethodGet, "/inference/metrics/epp-decisions?pool=llama3-70b-prod&limit=5", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []EPPDecisionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp) != 5 {
		t.Fatalf("expected 5 decisions, got %d", len(resp))
	}

	for _, d := range resp {
		if d.RequestID == "" {
			t.Error("expected non-empty request ID")
		}
		if d.SelectedPod == "" {
			t.Error("expected non-empty selected pod")
		}
	}
}

func TestInferenceMetricsHandler_TTFTHistogram(t *testing.T) {
	handler := newInferenceMetricsHandler()

	r := chi.NewRouter()
	r.Get("/inference/metrics/ttft-histogram/{pool}", handler.TTFTHistogram)

	req := httptest.NewRequest(http.MethodGet, "/inference/metrics/ttft-histogram/llama3-70b-prod", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []HistogramBucketResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp) != 8 {
		t.Fatalf("expected 8 buckets, got %d", len(resp))
	}
}

func TestInferenceMetricsHandler_Timeseries(t *testing.T) {
	handler := newInferenceMetricsHandler()

	for _, tc := range []struct {
		name    string
		path    string
		handler http.HandlerFunc
	}{
		{"TPSThroughput", "/inference/metrics/tps-throughput/{pool}", handler.TPSThroughput},
		{"QueueDepthSeries", "/inference/metrics/queue-depth/{pool}", handler.QueueDepthSeries},
		{"GPUUtilSeries", "/inference/metrics/gpu-util/{pool}", handler.GPUUtilSeries},
		{"KVCacheSeries", "/inference/metrics/kv-cache/{pool}", handler.KVCacheSeries},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := chi.NewRouter()
			r.Get(tc.path, tc.handler)

			url := "/inference/metrics/" + tc.name + "/llama3-70b-prod"
			// Use the actual route path pattern for the request
			switch tc.name {
			case "TPSThroughput":
				url = "/inference/metrics/tps-throughput/llama3-70b-prod"
			case "QueueDepthSeries":
				url = "/inference/metrics/queue-depth/llama3-70b-prod"
			case "GPUUtilSeries":
				url = "/inference/metrics/gpu-util/llama3-70b-prod"
			case "KVCacheSeries":
				url = "/inference/metrics/kv-cache/llama3-70b-prod"
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
			}

			var resp []TimeseriesPointResponse
			if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if len(resp) != 60 {
				t.Fatalf("expected 60 points, got %d", len(resp))
			}

			for _, p := range resp {
				if p.Timestamp == "" {
					t.Error("expected non-empty timestamp")
				}
			}
		})
	}
}
