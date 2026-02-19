package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func newTestBundle(name, namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "ngf-console.f5.com/v1alpha1",
			"kind":       "GatewayBundle",
			"metadata":   map[string]any{"name": name, "namespace": namespace},
			"spec": map[string]any{
				"gatewayClassName": "nginx",
				"listeners": []any{
					map[string]any{"name": "http", "port": int64(80), "protocol": "HTTP"},
				},
			},
		},
	}
}

func TestGatewayBundleHandler_List_HappyPath(t *testing.T) {
	b1 := newTestBundle("b1", "default")
	b2 := newTestBundle("b2", "default")
	dc := newFakeDynamicClient(b1, b2)
	handler := &GatewayBundleHandler{DynamicClient: dc}

	r := chi.NewRouter()
	r.Get("/bundles", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/bundles", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []GatewayBundleResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp) != 2 {
		t.Fatalf("expected 2 bundles, got %d", len(resp))
	}
}

func TestGatewayBundleHandler_List_NilClient(t *testing.T) {
	handler := &GatewayBundleHandler{}

	r := chi.NewRouter()
	r.Get("/bundles", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/bundles", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGatewayBundleHandler_Get_HappyPath(t *testing.T) {
	b1 := newTestBundle("b1", "default")
	dc := newFakeDynamicClient(b1)
	handler := &GatewayBundleHandler{DynamicClient: dc}

	r := chi.NewRouter()
	r.Get("/bundles/{namespace}/{name}", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/bundles/default/b1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp GatewayBundleResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Name != "b1" {
		t.Errorf("expected name b1, got %s", resp.Name)
	}
}

func TestGatewayBundleHandler_Get_NotFound(t *testing.T) {
	dc := newFakeDynamicClient()
	handler := &GatewayBundleHandler{DynamicClient: dc}

	r := chi.NewRouter()
	r.Get("/bundles/{namespace}/{name}", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/bundles/default/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGatewayBundleHandler_Create_HappyPath(t *testing.T) {
	dc := newFakeDynamicClient()
	handler := &GatewayBundleHandler{DynamicClient: dc}

	r := chi.NewRouter()
	r.Post("/bundles", handler.Create)

	body := map[string]any{
		"name":             "new-b",
		"namespace":        "default",
		"gatewayClassName": "nginx",
		"listeners": []map[string]any{
			{"name": "http", "port": 80, "protocol": "HTTP"},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/bundles", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGatewayBundleHandler_Create_MissingFields(t *testing.T) {
	dc := newFakeDynamicClient()
	handler := &GatewayBundleHandler{DynamicClient: dc}

	r := chi.NewRouter()
	r.Post("/bundles", handler.Create)

	body := map[string]any{
		"name": "no-ns",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/bundles", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGatewayBundleHandler_Delete_HappyPath(t *testing.T) {
	b1 := newTestBundle("b1", "default")
	dc := newFakeDynamicClient(b1)
	handler := &GatewayBundleHandler{DynamicClient: dc}

	r := chi.NewRouter()
	r.Delete("/bundles/{namespace}/{name}", handler.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/bundles/default/b1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}
