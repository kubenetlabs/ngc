package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakedynamic "k8s.io/client-go/dynamic/fake"

	"github.com/kubenetlabs/ngc/api/internal/inference"
)

func newFakeInferenceStackDynamicClient(objects ...runtime.Object) *fakedynamic.FakeDynamicClient {
	s := runtime.NewScheme()
	s.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "ngf-console.f5.com", Version: "v1alpha1", Kind: "InferenceStack"},
		&unstructured.Unstructured{},
	)
	s.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "ngf-console.f5.com", Version: "v1alpha1", Kind: "InferenceStackList"},
		&unstructured.UnstructuredList{},
	)
	return fakedynamic.NewSimpleDynamicClient(s, objects...)
}

func newTestInferenceStack(name, namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "ngf-console.f5.com/v1alpha1",
			"kind":       "InferenceStack",
			"metadata":   map[string]any{"name": name, "namespace": namespace},
			"spec": map[string]any{
				"modelName":      "test-model",
				"servingBackend": "vllm",
				"pool": map[string]any{
					"gpuType":     "H100",
					"gpuCount":    int64(4),
					"replicas":    int64(2),
					"minReplicas": int64(1),
					"maxReplicas": int64(8),
				},
			},
		},
	}
}

func TestInferenceStackHandler_List_HappyPath(t *testing.T) {
	s1 := newTestInferenceStack("stack-1", "inference")
	s2 := newTestInferenceStack("stack-2", "inference")
	dc := newFakeInferenceStackDynamicClient(s1, s2)
	handler := &InferenceStackHandler{DynamicClient: dc}

	r := chi.NewRouter()
	r.Get("/inferencestacks", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/inferencestacks", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []InferenceStackResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp) != 2 {
		t.Fatalf("expected 2 stacks, got %d", len(resp))
	}
}

func TestInferenceStackHandler_List_NilClient(t *testing.T) {
	handler := &InferenceStackHandler{}

	r := chi.NewRouter()
	r.Get("/inferencestacks", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/inferencestacks", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInferenceStackHandler_Get_HappyPath(t *testing.T) {
	s1 := newTestInferenceStack("stack-1", "inference")
	dc := newFakeInferenceStackDynamicClient(s1)
	handler := &InferenceStackHandler{DynamicClient: dc}

	r := chi.NewRouter()
	r.Get("/inferencestacks/{namespace}/{name}", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/inferencestacks/inference/stack-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp InferenceStackResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Name != "stack-1" {
		t.Errorf("expected name stack-1, got %s", resp.Name)
	}
	if resp.ModelName != "test-model" {
		t.Errorf("expected modelName test-model, got %s", resp.ModelName)
	}
}

func TestInferenceStackHandler_Create_HappyPath(t *testing.T) {
	dc := newFakeInferenceStackDynamicClient()
	provider := inference.NewMockProvider()
	handler := &InferenceStackHandler{DynamicClient: dc, MetricsProvider: provider}

	r := chi.NewRouter()
	r.Post("/inferencestacks", handler.Create)

	body := CreateInferenceStackRequest{
		Name:           "new-stack",
		Namespace:      "inference",
		ModelName:      "test-model",
		ServingBackend: "vllm",
		Pool: CreateInferenceStackPoolReq{
			GPUType:     "A100",
			GPUCount:    2,
			Replicas:    3,
			MinReplicas: 1,
			MaxReplicas: 6,
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/inferencestacks", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInferenceStackHandler_Create_MissingFields(t *testing.T) {
	dc := newFakeInferenceStackDynamicClient()
	handler := &InferenceStackHandler{DynamicClient: dc}

	r := chi.NewRouter()
	r.Post("/inferencestacks", handler.Create)

	body := CreateInferenceStackRequest{
		Name: "no-model",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/inferencestacks", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInferenceStackHandler_Delete_HappyPath(t *testing.T) {
	s1 := newTestInferenceStack("stack-1", "inference")
	dc := newFakeInferenceStackDynamicClient(s1)
	provider := inference.NewMockProvider()
	handler := &InferenceStackHandler{DynamicClient: dc, MetricsProvider: provider}

	r := chi.NewRouter()
	r.Delete("/inferencestacks/{namespace}/{name}", handler.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/inferencestacks/inference/stack-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}
