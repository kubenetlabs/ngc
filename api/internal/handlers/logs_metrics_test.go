package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestLogHandler_Query_NilCH(t *testing.T) {
	handler := &LogHandler{CH: nil}

	r := chi.NewRouter()
	r.Post("/logs/query", handler.Query)

	req := httptest.NewRequest(http.MethodPost, "/logs/query", strings.NewReader(`{"limit":10}`))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "clickhouse not configured") {
		t.Errorf("expected clickhouse not configured message, got %s", w.Body.String())
	}
}

func TestLogHandler_TopN_NilCH(t *testing.T) {
	handler := &LogHandler{CH: nil}

	r := chi.NewRouter()
	r.Get("/logs/topn", handler.TopN)

	req := httptest.NewRequest(http.MethodGet, "/logs/topn", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "clickhouse not configured") {
		t.Errorf("expected clickhouse not configured message, got %s", w.Body.String())
	}
}

func TestMetricsHandler_Summary_NilProm(t *testing.T) {
	handler := &MetricsHandler{Prom: nil}

	r := chi.NewRouter()
	r.Get("/metrics/summary", handler.Summary)

	req := httptest.NewRequest(http.MethodGet, "/metrics/summary", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "prometheus not configured") {
		t.Errorf("expected prometheus not configured message, got %s", w.Body.String())
	}
}

func TestMetricsHandler_ByRoute_NilProm(t *testing.T) {
	handler := &MetricsHandler{Prom: nil}

	r := chi.NewRouter()
	r.Get("/metrics/by-route", handler.ByRoute)

	req := httptest.NewRequest(http.MethodGet, "/metrics/by-route", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "prometheus not configured") {
		t.Errorf("expected prometheus not configured message, got %s", w.Body.String())
	}
}

func TestMetricsHandler_ByGateway_NilProm(t *testing.T) {
	handler := &MetricsHandler{Prom: nil}

	r := chi.NewRouter()
	r.Get("/metrics/by-gateway", handler.ByGateway)

	req := httptest.NewRequest(http.MethodGet, "/metrics/by-gateway", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "prometheus not configured") {
		t.Errorf("expected prometheus not configured message, got %s", w.Body.String())
	}
}
