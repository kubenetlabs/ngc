package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
	"github.com/kubenetlabs/ngc/api/internal/kubernetes"
)

func TestConfigHandler_GetConfig(t *testing.T) {
	t.Run("with connected kube client (OSS)", func(t *testing.T) {
		scheme := setupScheme(t)
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		k8sClient := kubernetes.NewForTest(fakeClient)
		handler := &ConfigHandler{}

		r := chi.NewRouter()
		r.Use(contextMiddleware(k8sClient))
		r.Get("/api/v1/config", handler.GetConfig)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/config", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp configResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if !resp.Connected {
			t.Error("expected connected to be true")
		}
		// Without enterprise CRDs in fake, should return OSS
		if resp.Edition != "oss" {
			t.Errorf("expected edition oss, got %s", resp.Edition)
		}
		if resp.Version == "" {
			t.Error("expected version to be set")
		}
	})

	t.Run("with nil kube client (no context)", func(t *testing.T) {
		handler := &ConfigHandler{}

		// No context middleware — simulates no cluster set
		req := httptest.NewRequest(http.MethodGet, "/api/v1/config", nil)
		w := httptest.NewRecorder()

		handler.GetConfig(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp configResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Connected {
			t.Error("expected connected to be false")
		}
		if resp.Edition != "unknown" {
			t.Errorf("expected edition unknown, got %s", resp.Edition)
		}
		if resp.Version == "" {
			t.Error("expected version to be set")
		}
	})

	t.Run("with cluster name in context", func(t *testing.T) {
		scheme := setupScheme(t)
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		k8sClient := kubernetes.NewForTest(fakeClient)
		handler := &ConfigHandler{}

		r := chi.NewRouter()
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctx := cluster.WithClient(r.Context(), k8sClient)
				ctx = cluster.WithClusterName(ctx, "production")
				next.ServeHTTP(w, r.WithContext(ctx))
			})
		})
		r.Get("/api/v1/config", handler.GetConfig)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/config", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp configResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Cluster != "production" {
			t.Errorf("expected cluster production, got %s", resp.Cluster)
		}
	})
}
