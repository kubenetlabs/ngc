package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kubenetlabs/ngc/api/internal/database"
)

func TestAlertHandler_Create(t *testing.T) {
	t.Run("returns 201 with valid payload", func(t *testing.T) {
		store := database.NewMockStore()
		handler := &AlertHandler{Store: store}

		r := chi.NewRouter()
		r.Post("/api/v1/alerts", handler.Create)

		body := `{
			"name": "High Error Rate",
			"description": "Alert when error rate exceeds threshold",
			"resource": "gateway",
			"metric": "error_rate",
			"operator": "gt",
			"threshold": 0.05,
			"severity": "critical"
		}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/alerts", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
		}

		var resp database.AlertRule
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Name != "High Error Rate" {
			t.Errorf("expected name 'High Error Rate', got %s", resp.Name)
		}
		if resp.Resource != "gateway" {
			t.Errorf("expected resource 'gateway', got %s", resp.Resource)
		}
		if resp.Operator != "gt" {
			t.Errorf("expected operator 'gt', got %s", resp.Operator)
		}
		if resp.Threshold != 0.05 {
			t.Errorf("expected threshold 0.05, got %f", resp.Threshold)
		}
		if resp.Severity != "critical" {
			t.Errorf("expected severity 'critical', got %s", resp.Severity)
		}
		if !resp.Enabled {
			t.Error("expected enabled to be true by default")
		}
		if resp.ID == "" {
			t.Error("expected non-empty ID")
		}
	})

	t.Run("returns 400 with missing fields", func(t *testing.T) {
		store := database.NewMockStore()
		handler := &AlertHandler{Store: store}

		r := chi.NewRouter()
		r.Post("/api/v1/alerts", handler.Create)

		body := `{"name": "Incomplete Rule"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/alerts", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("returns 400 with invalid operator", func(t *testing.T) {
		store := database.NewMockStore()
		handler := &AlertHandler{Store: store}

		r := chi.NewRouter()
		r.Post("/api/v1/alerts", handler.Create)

		body := `{
			"name": "Bad Operator",
			"resource": "gateway",
			"metric": "error_rate",
			"operator": "gte",
			"threshold": 0.05,
			"severity": "critical"
		}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/alerts", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]string
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if !strings.Contains(resp["error"], "operator") {
			t.Errorf("expected error about operator, got: %s", resp["error"])
		}
	})

	t.Run("returns 400 with invalid severity", func(t *testing.T) {
		store := database.NewMockStore()
		handler := &AlertHandler{Store: store}

		r := chi.NewRouter()
		r.Post("/api/v1/alerts", handler.Create)

		body := `{
			"name": "Bad Severity",
			"resource": "gateway",
			"metric": "error_rate",
			"operator": "gt",
			"threshold": 0.05,
			"severity": "urgent"
		}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/alerts", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]string
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if !strings.Contains(resp["error"], "severity") {
			t.Errorf("expected error about severity, got: %s", resp["error"])
		}
	})
}

func TestAlertHandler_List(t *testing.T) {
	t.Run("returns 200 with empty rules", func(t *testing.T) {
		store := database.NewMockStore()
		handler := &AlertHandler{Store: store}

		r := chi.NewRouter()
		r.Get("/api/v1/alerts", handler.List)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp []database.AlertRule
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(resp) != 0 {
			t.Errorf("expected 0 rules, got %d", len(resp))
		}
	})

	t.Run("returns 503 when Store is nil", func(t *testing.T) {
		handler := &AlertHandler{Store: nil}

		r := chi.NewRouter()
		r.Get("/api/v1/alerts", handler.List)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status 503, got %d", w.Code)
		}
	})
}

func TestAlertHandler_Get(t *testing.T) {
	t.Run("returns 200 for existing rule", func(t *testing.T) {
		store := database.NewMockStore()
		handler := &AlertHandler{Store: store}

		rule := database.AlertRule{
			ID:        "rule-1",
			Name:      "Test Rule",
			Resource:  "gateway",
			Metric:    "error_rate",
			Operator:  "gt",
			Threshold: 0.05,
			Severity:  "critical",
			Enabled:   true,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		if err := store.CreateAlertRule(context.Background(), rule); err != nil {
			t.Fatalf("failed to create rule: %v", err)
		}

		r := chi.NewRouter()
		r.Get("/api/v1/alerts/{id}", handler.Get)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts/rule-1", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp database.AlertRule
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.ID != "rule-1" {
			t.Errorf("expected id rule-1, got %s", resp.ID)
		}
		if resp.Name != "Test Rule" {
			t.Errorf("expected name 'Test Rule', got %s", resp.Name)
		}
	})

	t.Run("returns 404 for nonexistent rule", func(t *testing.T) {
		store := database.NewMockStore()
		handler := &AlertHandler{Store: store}

		r := chi.NewRouter()
		r.Get("/api/v1/alerts/{id}", handler.Get)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts/nonexistent", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", w.Code)
		}
	})
}

func TestAlertHandler_Update(t *testing.T) {
	t.Run("returns 200 on successful update", func(t *testing.T) {
		store := database.NewMockStore()
		handler := &AlertHandler{Store: store}

		rule := database.AlertRule{
			ID:        "rule-upd-1",
			Name:      "Original Name",
			Resource:  "gateway",
			Metric:    "error_rate",
			Operator:  "gt",
			Threshold: 0.05,
			Severity:  "warning",
			Enabled:   true,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		if err := store.CreateAlertRule(context.Background(), rule); err != nil {
			t.Fatalf("failed to create rule: %v", err)
		}

		r := chi.NewRouter()
		r.Put("/api/v1/alerts/{id}", handler.Update)

		body := `{"name": "Updated Name", "threshold": 0.10}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/alerts/rule-upd-1", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp database.AlertRule
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Name != "Updated Name" {
			t.Errorf("expected name 'Updated Name', got %s", resp.Name)
		}
		if resp.Threshold != 0.10 {
			t.Errorf("expected threshold 0.10, got %f", resp.Threshold)
		}
	})

	t.Run("returns 404 for nonexistent rule", func(t *testing.T) {
		store := database.NewMockStore()
		handler := &AlertHandler{Store: store}

		r := chi.NewRouter()
		r.Put("/api/v1/alerts/{id}", handler.Update)

		body := `{"name": "Updated Name"}`
		req := httptest.NewRequest(http.MethodPut, "/api/v1/alerts/nonexistent", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", w.Code)
		}
	})
}

func TestAlertHandler_Delete(t *testing.T) {
	t.Run("returns 200 on successful delete", func(t *testing.T) {
		store := database.NewMockStore()
		handler := &AlertHandler{Store: store}

		rule := database.AlertRule{
			ID:        "rule-del-1",
			Name:      "To Delete",
			Resource:  "gateway",
			Metric:    "error_rate",
			Operator:  "gt",
			Threshold: 0.05,
			Severity:  "info",
			Enabled:   true,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		if err := store.CreateAlertRule(context.Background(), rule); err != nil {
			t.Fatalf("failed to create rule: %v", err)
		}

		r := chi.NewRouter()
		r.Delete("/api/v1/alerts/{id}", handler.Delete)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/alerts/rule-del-1", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]string
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp["message"] != "alert rule deleted" {
			t.Errorf("expected 'alert rule deleted', got %s", resp["message"])
		}

		// Verify it's actually deleted
		got, err := store.GetAlertRule(context.Background(), "rule-del-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Error("expected rule to be deleted, but it still exists")
		}
	})

	t.Run("returns 404 for nonexistent rule", func(t *testing.T) {
		store := database.NewMockStore()
		handler := &AlertHandler{Store: store}

		r := chi.NewRouter()
		r.Delete("/api/v1/alerts/{id}", handler.Delete)

		req := httptest.NewRequest(http.MethodDelete, "/api/v1/alerts/nonexistent", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", w.Code)
		}
	})
}

func TestAlertHandler_Toggle(t *testing.T) {
	t.Run("flips enabled state", func(t *testing.T) {
		store := database.NewMockStore()
		handler := &AlertHandler{Store: store}

		rule := database.AlertRule{
			ID:        "rule-toggle-1",
			Name:      "Toggle Test",
			Resource:  "gateway",
			Metric:    "error_rate",
			Operator:  "gt",
			Threshold: 0.05,
			Severity:  "warning",
			Enabled:   true,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		if err := store.CreateAlertRule(context.Background(), rule); err != nil {
			t.Fatalf("failed to create rule: %v", err)
		}

		r := chi.NewRouter()
		r.Post("/api/v1/alerts/{id}/toggle", handler.Toggle)

		// First toggle: true -> false
		req := httptest.NewRequest(http.MethodPost, "/api/v1/alerts/rule-toggle-1/toggle", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp database.AlertRule
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Enabled {
			t.Error("expected enabled to be false after first toggle")
		}

		// Second toggle: false -> true
		req = httptest.NewRequest(http.MethodPost, "/api/v1/alerts/rule-toggle-1/toggle", nil)
		w = httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200 on second toggle, got %d", w.Code)
		}

		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if !resp.Enabled {
			t.Error("expected enabled to be true after second toggle")
		}
	})

	t.Run("returns 404 for nonexistent rule", func(t *testing.T) {
		store := database.NewMockStore()
		handler := &AlertHandler{Store: store}

		r := chi.NewRouter()
		r.Post("/api/v1/alerts/{id}/toggle", handler.Toggle)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/alerts/nonexistent/toggle", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", w.Code)
		}
	})
}

func TestAlertHandler_Firing(t *testing.T) {
	t.Run("returns 200 with empty array when Evaluator is nil", func(t *testing.T) {
		store := database.NewMockStore()
		handler := &AlertHandler{Store: store, Evaluator: nil}

		r := chi.NewRouter()
		r.Get("/api/v1/alerts/firing", handler.Firing)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts/firing", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp []interface{}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(resp) != 0 {
			t.Errorf("expected 0 firing alerts, got %d", len(resp))
		}
	})
}
