package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kubenetlabs/ngc/api/internal/database"
)

func TestAuditHandler_List(t *testing.T) {
	t.Run("returns 200 with empty entries", func(t *testing.T) {
		store := database.NewMockStore()
		handler := &AuditHandler{Store: store}

		r := chi.NewRouter()
		r.Get("/api/v1/audit", handler.List)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/audit", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp struct {
			Entries []database.AuditEntry `json:"entries"`
			Total   int64                 `json:"total"`
		}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(resp.Entries) != 0 {
			t.Errorf("expected 0 entries, got %d", len(resp.Entries))
		}
		if resp.Total != 0 {
			t.Errorf("expected total 0, got %d", resp.Total)
		}
	})

	t.Run("returns 200 with entries after inserting", func(t *testing.T) {
		store := database.NewMockStore()
		handler := &AuditHandler{Store: store}

		entry1 := database.AuditEntry{
			ID:        "audit-1",
			Timestamp: time.Now().UTC(),
			User:      "admin",
			Action:    "create",
			Resource:  "Gateway",
			Name:      "my-gw",
			Namespace: "default",
		}
		entry2 := database.AuditEntry{
			ID:        "audit-2",
			Timestamp: time.Now().UTC(),
			User:      "admin",
			Action:    "update",
			Resource:  "HTTPRoute",
			Name:      "my-route",
			Namespace: "default",
		}

		if err := store.InsertAuditEntry(context.Background(), entry1); err != nil {
			t.Fatalf("failed to insert entry: %v", err)
		}
		if err := store.InsertAuditEntry(context.Background(), entry2); err != nil {
			t.Fatalf("failed to insert entry: %v", err)
		}

		r := chi.NewRouter()
		r.Get("/api/v1/audit", handler.List)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/audit", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp struct {
			Entries []database.AuditEntry `json:"entries"`
			Total   int64                 `json:"total"`
		}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(resp.Entries) != 2 {
			t.Errorf("expected 2 entries, got %d", len(resp.Entries))
		}
		if resp.Total != 2 {
			t.Errorf("expected total 2, got %d", resp.Total)
		}
	})

	t.Run("returns 503 when Store is nil", func(t *testing.T) {
		handler := &AuditHandler{Store: nil}

		r := chi.NewRouter()
		r.Get("/api/v1/audit", handler.List)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/audit", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status 503, got %d", w.Code)
		}
	})
}

func TestAuditHandler_Diff(t *testing.T) {
	t.Run("returns 404 for nonexistent ID", func(t *testing.T) {
		store := database.NewMockStore()
		handler := &AuditHandler{Store: store}

		r := chi.NewRouter()
		r.Get("/api/v1/audit/{id}", handler.Diff)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/nonexistent", nil)
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
			t.Error("expected error message in response")
		}
	})

	t.Run("returns 200 for existing entry", func(t *testing.T) {
		store := database.NewMockStore()
		handler := &AuditHandler{Store: store}

		entry := database.AuditEntry{
			ID:         "audit-diff-1",
			Timestamp:  time.Now().UTC(),
			User:       "admin",
			Action:     "update",
			Resource:   "Gateway",
			Name:       "my-gw",
			Namespace:  "default",
			BeforeJSON: `{"replicas": 1}`,
			AfterJSON:  `{"replicas": 3}`,
		}
		if err := store.InsertAuditEntry(context.Background(), entry); err != nil {
			t.Fatalf("failed to insert entry: %v", err)
		}

		r := chi.NewRouter()
		r.Get("/api/v1/audit/{id}", handler.Diff)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/audit-diff-1", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]interface{}
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp["id"] != "audit-diff-1" {
			t.Errorf("expected id audit-diff-1, got %v", resp["id"])
		}
		if resp["action"] != "update" {
			t.Errorf("expected action update, got %v", resp["action"])
		}
		if resp["resource"] != "Gateway" {
			t.Errorf("expected resource Gateway, got %v", resp["resource"])
		}
		if resp["beforeJson"] != `{"replicas": 1}` {
			t.Errorf("expected beforeJson, got %v", resp["beforeJson"])
		}
		if resp["afterJson"] != `{"replicas": 3}` {
			t.Errorf("expected afterJson, got %v", resp["afterJson"])
		}
	})

	t.Run("returns 503 when Store is nil", func(t *testing.T) {
		handler := &AuditHandler{Store: nil}

		r := chi.NewRouter()
		r.Get("/api/v1/audit/{id}", handler.Diff)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/any-id", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("expected status 503, got %d", w.Code)
		}
	})
}
