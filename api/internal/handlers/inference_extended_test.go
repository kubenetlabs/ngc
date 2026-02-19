package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/kubenetlabs/ngc/api/internal/inference"
)

// cleanupPool registers a t.Cleanup that removes a pool created during the test
// from the mock provider's global state so it doesn't leak into other tests.
func cleanupPool(t *testing.T, provider *inference.MockProvider, name, namespace string) {
	t.Helper()
	t.Cleanup(func() {
		_ = provider.DeletePool(context.Background(), name, namespace)
	})
}

// ---------------------------------------------------------------------------
// CreatePool
// ---------------------------------------------------------------------------

func TestInferenceHandler_CreatePool_HappyPath(t *testing.T) {
	dc := newFakeInferenceStackDynamicClient()
	provider := inference.NewMockProvider()
	handler := &InferenceHandler{Provider: provider, DynamicClient: dc}
	cleanupPool(t, provider, "new-pool", "inference")

	r := chi.NewRouter()
	r.Post("/inference/pools", handler.CreatePool)

	body := CreatePoolRequest{
		Name:           "new-pool",
		Namespace:      "inference",
		ModelName:      "llama-3.1-70b",
		ServingBackend: "vllm",
		GPUType:        "H100",
		GPUCount:       4,
		Replicas:       2,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/inference/pools", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp InferenceStackResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Name != "new-pool" {
		t.Errorf("expected name new-pool, got %s", resp.Name)
	}
	if resp.Namespace != "inference" {
		t.Errorf("expected namespace inference, got %s", resp.Namespace)
	}
	if resp.ModelName != "llama-3.1-70b" {
		t.Errorf("expected modelName llama-3.1-70b, got %s", resp.ModelName)
	}
	if resp.ServingBackend != "vllm" {
		t.Errorf("expected servingBackend vllm, got %s", resp.ServingBackend)
	}
	if resp.Pool.GPUType != "H100" {
		t.Errorf("expected gpuType H100, got %s", resp.Pool.GPUType)
	}
	if resp.Pool.GPUCount != 4 {
		t.Errorf("expected gpuCount 4, got %d", resp.Pool.GPUCount)
	}
	if resp.Pool.Replicas != 2 {
		t.Errorf("expected replicas 2, got %d", resp.Pool.Replicas)
	}
}

func TestInferenceHandler_CreatePool_MissingName(t *testing.T) {
	dc := newFakeInferenceStackDynamicClient()
	handler := &InferenceHandler{Provider: inference.NewMockProvider(), DynamicClient: dc}

	r := chi.NewRouter()
	r.Post("/inference/pools", handler.CreatePool)

	body := CreatePoolRequest{
		Namespace:      "inference",
		ModelName:      "llama-3.1-70b",
		ServingBackend: "vllm",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/inference/pools", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInferenceHandler_CreatePool_MissingNamespace(t *testing.T) {
	dc := newFakeInferenceStackDynamicClient()
	handler := &InferenceHandler{Provider: inference.NewMockProvider(), DynamicClient: dc}

	r := chi.NewRouter()
	r.Post("/inference/pools", handler.CreatePool)

	body := CreatePoolRequest{
		Name:           "pool",
		ModelName:      "model",
		ServingBackend: "vllm",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/inference/pools", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInferenceHandler_CreatePool_MissingModelName(t *testing.T) {
	dc := newFakeInferenceStackDynamicClient()
	handler := &InferenceHandler{Provider: inference.NewMockProvider(), DynamicClient: dc}

	r := chi.NewRouter()
	r.Post("/inference/pools", handler.CreatePool)

	body := CreatePoolRequest{
		Name:           "pool",
		Namespace:      "inference",
		ServingBackend: "vllm",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/inference/pools", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInferenceHandler_CreatePool_MissingServingBackend(t *testing.T) {
	dc := newFakeInferenceStackDynamicClient()
	handler := &InferenceHandler{Provider: inference.NewMockProvider(), DynamicClient: dc}

	r := chi.NewRouter()
	r.Post("/inference/pools", handler.CreatePool)

	body := CreatePoolRequest{
		Name:      "pool",
		Namespace: "inference",
		ModelName: "model",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/inference/pools", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInferenceHandler_CreatePool_NilDynamicClient(t *testing.T) {
	handler := &InferenceHandler{Provider: inference.NewMockProvider()}

	r := chi.NewRouter()
	r.Post("/inference/pools", handler.CreatePool)

	body := CreatePoolRequest{
		Name:           "pool",
		Namespace:      "default",
		ModelName:      "model",
		ServingBackend: "vllm",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/inference/pools", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInferenceHandler_CreatePool_WithEPP(t *testing.T) {
	dc := newFakeInferenceStackDynamicClient()
	provider := inference.NewMockProvider()
	handler := &InferenceHandler{Provider: provider, DynamicClient: dc}
	cleanupPool(t, provider, "epp-pool", "inference")

	r := chi.NewRouter()
	r.Post("/inference/pools", handler.CreatePool)

	body := CreatePoolRequest{
		Name:           "epp-pool",
		Namespace:      "inference",
		ModelName:      "llama-3.1-70b",
		ServingBackend: "vllm",
		GPUType:        "H100",
		GPUCount:       4,
		Replicas:       2,
		MinReplicas:    1,
		MaxReplicas:    8,
		EPP: &CreateInferenceStackEPPReq{
			Strategy: "least-load",
			Weights: &InferenceStackWeightsResp{
				QueueDepth:     40,
				KVCache:        30,
				PrefixAffinity: 30,
			},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/inference/pools", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp InferenceStackResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.EPP == nil {
		t.Fatal("expected EPP config in response")
	}
	if resp.EPP.Strategy != "least-load" {
		t.Errorf("expected strategy least-load, got %s", resp.EPP.Strategy)
	}
	if resp.EPP.Weights == nil {
		t.Fatal("expected EPP weights in response")
	}
	if resp.EPP.Weights.QueueDepth != 40 {
		t.Errorf("expected queueDepth 40, got %d", resp.EPP.Weights.QueueDepth)
	}
}

func TestInferenceHandler_CreatePool_InvalidJSON(t *testing.T) {
	dc := newFakeInferenceStackDynamicClient()
	handler := &InferenceHandler{Provider: inference.NewMockProvider(), DynamicClient: dc}

	r := chi.NewRouter()
	r.Post("/inference/pools", handler.CreatePool)

	req := httptest.NewRequest(http.MethodPost, "/inference/pools", bytes.NewReader([]byte("{invalid")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// UpdatePool
// ---------------------------------------------------------------------------

func TestInferenceHandler_UpdatePool_HappyPath(t *testing.T) {
	s1 := newTestInferenceStack("my-pool", "inference")
	dc := newFakeInferenceStackDynamicClient(s1)
	handler := &InferenceHandler{Provider: inference.NewMockProvider(), DynamicClient: dc}

	r := chi.NewRouter()
	r.Put("/inference/pools/{name}", handler.UpdatePool)

	replicas := 5
	body := UpdatePoolRequest{
		ModelName: "updated-model",
		Replicas:  &replicas,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/inference/pools/my-pool", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp InferenceStackResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ModelName != "updated-model" {
		t.Errorf("expected modelName updated-model, got %s", resp.ModelName)
	}
	if resp.Pool.Replicas != 5 {
		t.Errorf("expected replicas 5, got %d", resp.Pool.Replicas)
	}
}

func TestInferenceHandler_UpdatePool_PartialUpdate(t *testing.T) {
	s1 := newTestInferenceStack("partial-pool", "inference")
	dc := newFakeInferenceStackDynamicClient(s1)
	handler := &InferenceHandler{Provider: inference.NewMockProvider(), DynamicClient: dc}

	r := chi.NewRouter()
	r.Put("/inference/pools/{name}", handler.UpdatePool)

	// Only update GPU type, leave everything else unchanged.
	body := UpdatePoolRequest{
		GPUType: "A100",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/inference/pools/partial-pool", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp InferenceStackResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Pool.GPUType != "A100" {
		t.Errorf("expected gpuType A100, got %s", resp.Pool.GPUType)
	}
	// Original values should be preserved.
	if resp.ModelName != "test-model" {
		t.Errorf("expected modelName test-model (unchanged), got %s", resp.ModelName)
	}
	if resp.Pool.Replicas != 2 {
		t.Errorf("expected replicas 2 (unchanged), got %d", resp.Pool.Replicas)
	}
}

func TestInferenceHandler_UpdatePool_NotFound(t *testing.T) {
	dc := newFakeInferenceStackDynamicClient()
	handler := &InferenceHandler{Provider: inference.NewMockProvider(), DynamicClient: dc}

	r := chi.NewRouter()
	r.Put("/inference/pools/{name}", handler.UpdatePool)

	body := UpdatePoolRequest{ModelName: "updated"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/inference/pools/nonexistent", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInferenceHandler_UpdatePool_NilDynamicClient(t *testing.T) {
	handler := &InferenceHandler{Provider: inference.NewMockProvider()}

	r := chi.NewRouter()
	r.Put("/inference/pools/{name}", handler.UpdatePool)

	body := UpdatePoolRequest{ModelName: "updated"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/inference/pools/some-pool", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInferenceHandler_UpdatePool_InvalidJSON(t *testing.T) {
	s1 := newTestInferenceStack("json-pool", "inference")
	dc := newFakeInferenceStackDynamicClient(s1)
	handler := &InferenceHandler{Provider: inference.NewMockProvider(), DynamicClient: dc}

	r := chi.NewRouter()
	r.Put("/inference/pools/{name}", handler.UpdatePool)

	req := httptest.NewRequest(http.MethodPut, "/inference/pools/json-pool", bytes.NewReader([]byte("{bad")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInferenceHandler_UpdatePool_WithEPP(t *testing.T) {
	s1 := newTestInferenceStack("epp-update-pool", "inference")
	dc := newFakeInferenceStackDynamicClient(s1)
	handler := &InferenceHandler{Provider: inference.NewMockProvider(), DynamicClient: dc}

	r := chi.NewRouter()
	r.Put("/inference/pools/{name}", handler.UpdatePool)

	body := UpdatePoolRequest{
		EPP: &CreateInferenceStackEPPReq{
			Strategy: "prefix-affinity",
			Weights: &InferenceStackWeightsResp{
				QueueDepth:     20,
				KVCache:        30,
				PrefixAffinity: 50,
			},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/inference/pools/epp-update-pool", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp InferenceStackResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.EPP == nil {
		t.Fatal("expected EPP in response")
	}
	if resp.EPP.Strategy != "prefix-affinity" {
		t.Errorf("expected strategy prefix-affinity, got %s", resp.EPP.Strategy)
	}
}

// ---------------------------------------------------------------------------
// DeletePool
// ---------------------------------------------------------------------------

func TestInferenceHandler_DeletePool_HappyPath(t *testing.T) {
	s1 := newTestInferenceStack("del-pool", "inference")
	dc := newFakeInferenceStackDynamicClient(s1)
	handler := &InferenceHandler{Provider: inference.NewMockProvider(), DynamicClient: dc}

	r := chi.NewRouter()
	r.Delete("/inference/pools/{name}", handler.DeletePool)

	req := httptest.NewRequest(http.MethodDelete, "/inference/pools/del-pool", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["message"] != "inference pool deleted" {
		t.Errorf("expected message 'inference pool deleted', got %q", resp["message"])
	}
	if resp["name"] != "del-pool" {
		t.Errorf("expected name del-pool, got %s", resp["name"])
	}
	if resp["namespace"] != "inference" {
		t.Errorf("expected namespace inference, got %s", resp["namespace"])
	}
}

func TestInferenceHandler_DeletePool_NotFound(t *testing.T) {
	dc := newFakeInferenceStackDynamicClient()
	handler := &InferenceHandler{Provider: inference.NewMockProvider(), DynamicClient: dc}

	r := chi.NewRouter()
	r.Delete("/inference/pools/{name}", handler.DeletePool)

	req := httptest.NewRequest(http.MethodDelete, "/inference/pools/nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInferenceHandler_DeletePool_NilDynamicClient(t *testing.T) {
	handler := &InferenceHandler{Provider: inference.NewMockProvider()}

	r := chi.NewRouter()
	r.Delete("/inference/pools/{name}", handler.DeletePool)

	req := httptest.NewRequest(http.MethodDelete, "/inference/pools/some-pool", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// DeployPool
// ---------------------------------------------------------------------------

func TestInferenceHandler_DeployPool_HappyPath(t *testing.T) {
	s1 := newTestInferenceStack("deploy-pool", "inference")
	dc := newFakeInferenceStackDynamicClient(s1)
	handler := &InferenceHandler{Provider: inference.NewMockProvider(), DynamicClient: dc}

	r := chi.NewRouter()
	r.Post("/inference/pools/{name}/deploy", handler.DeployPool)

	req := httptest.NewRequest(http.MethodPost, "/inference/pools/deploy-pool/deploy", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["message"] != "reconciliation triggered" {
		t.Errorf("expected message 'reconciliation triggered', got %v", resp["message"])
	}
	if resp["name"] != "deploy-pool" {
		t.Errorf("expected name deploy-pool, got %v", resp["name"])
	}
	if resp["namespace"] != "inference" {
		t.Errorf("expected namespace inference, got %v", resp["namespace"])
	}
}

func TestInferenceHandler_DeployPool_NotFound(t *testing.T) {
	dc := newFakeInferenceStackDynamicClient()
	handler := &InferenceHandler{Provider: inference.NewMockProvider(), DynamicClient: dc}

	r := chi.NewRouter()
	r.Post("/inference/pools/{name}/deploy", handler.DeployPool)

	req := httptest.NewRequest(http.MethodPost, "/inference/pools/nonexistent/deploy", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInferenceHandler_DeployPool_NilDynamicClient(t *testing.T) {
	handler := &InferenceHandler{Provider: inference.NewMockProvider()}

	r := chi.NewRouter()
	r.Post("/inference/pools/{name}/deploy", handler.DeployPool)

	req := httptest.NewRequest(http.MethodPost, "/inference/pools/some-pool/deploy", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// GetEPP
// ---------------------------------------------------------------------------

func TestInferenceHandler_GetEPP_HappyPath(t *testing.T) {
	obj := newTestInferenceStack("epp-pool", "inference")
	obj.Object["spec"].(map[string]any)["epp"] = map[string]any{
		"strategy": "least-load",
		"weights": map[string]any{
			"queueDepth":     int64(40),
			"kvCache":        int64(30),
			"prefixAffinity": int64(30),
		},
	}
	dc := newFakeInferenceStackDynamicClient(obj)
	handler := &InferenceHandler{Provider: inference.NewMockProvider(), DynamicClient: dc}

	r := chi.NewRouter()
	r.Get("/inference/epp", handler.GetEPP)

	req := httptest.NewRequest(http.MethodGet, "/inference/epp?pool=epp-pool", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp EPPConfigResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Pool != "epp-pool" {
		t.Errorf("expected pool epp-pool, got %s", resp.Pool)
	}
	if resp.Strategy != "least-load" {
		t.Errorf("expected strategy least-load, got %s", resp.Strategy)
	}
	if resp.Weights == nil {
		t.Fatal("expected weights to be present")
	}
	if resp.Weights.QueueDepth != 40 {
		t.Errorf("expected queueDepth 40, got %d", resp.Weights.QueueDepth)
	}
	if resp.Weights.KVCache != 30 {
		t.Errorf("expected kvCache 30, got %d", resp.Weights.KVCache)
	}
	if resp.Weights.PrefixAffinity != 30 {
		t.Errorf("expected prefixAffinity 30, got %d", resp.Weights.PrefixAffinity)
	}
}

func TestInferenceHandler_GetEPP_NoEPPConfig(t *testing.T) {
	// Stack exists but has no EPP config — should return empty strategy and nil weights.
	obj := newTestInferenceStack("no-epp", "inference")
	dc := newFakeInferenceStackDynamicClient(obj)
	handler := &InferenceHandler{Provider: inference.NewMockProvider(), DynamicClient: dc}

	r := chi.NewRouter()
	r.Get("/inference/epp", handler.GetEPP)

	req := httptest.NewRequest(http.MethodGet, "/inference/epp?pool=no-epp", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp EPPConfigResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Pool != "no-epp" {
		t.Errorf("expected pool no-epp, got %s", resp.Pool)
	}
	if resp.Strategy != "" {
		t.Errorf("expected empty strategy, got %s", resp.Strategy)
	}
	if resp.Weights != nil {
		t.Errorf("expected nil weights, got %+v", resp.Weights)
	}
}

func TestInferenceHandler_GetEPP_MissingPool(t *testing.T) {
	handler := &InferenceHandler{Provider: inference.NewMockProvider()}

	r := chi.NewRouter()
	r.Get("/inference/epp", handler.GetEPP)

	req := httptest.NewRequest(http.MethodGet, "/inference/epp", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInferenceHandler_GetEPP_PoolNotFound(t *testing.T) {
	dc := newFakeInferenceStackDynamicClient()
	handler := &InferenceHandler{Provider: inference.NewMockProvider(), DynamicClient: dc}

	r := chi.NewRouter()
	r.Get("/inference/epp", handler.GetEPP)

	req := httptest.NewRequest(http.MethodGet, "/inference/epp?pool=nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// UpdateEPP
// ---------------------------------------------------------------------------

func TestInferenceHandler_UpdateEPP_HappyPath(t *testing.T) {
	obj := newTestInferenceStack("epp-update", "inference")
	dc := newFakeInferenceStackDynamicClient(obj)
	handler := &InferenceHandler{Provider: inference.NewMockProvider(), DynamicClient: dc}

	r := chi.NewRouter()
	r.Put("/inference/epp", handler.UpdateEPP)

	body := UpdateEPPRequest{
		Pool:     "epp-update",
		Strategy: "prefix-affinity",
		Weights: &InferenceStackWeightsResp{
			QueueDepth:     20,
			KVCache:        30,
			PrefixAffinity: 50,
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/inference/epp", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp EPPConfigResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Pool != "epp-update" {
		t.Errorf("expected pool epp-update, got %s", resp.Pool)
	}
	if resp.Strategy != "prefix-affinity" {
		t.Errorf("expected strategy prefix-affinity, got %s", resp.Strategy)
	}
	if resp.Weights == nil {
		t.Fatal("expected weights in response")
	}
	if resp.Weights.QueueDepth != 20 {
		t.Errorf("expected queueDepth 20, got %d", resp.Weights.QueueDepth)
	}
	if resp.Weights.KVCache != 30 {
		t.Errorf("expected kvCache 30, got %d", resp.Weights.KVCache)
	}
	if resp.Weights.PrefixAffinity != 50 {
		t.Errorf("expected prefixAffinity 50, got %d", resp.Weights.PrefixAffinity)
	}
}

func TestInferenceHandler_UpdateEPP_MissingPool(t *testing.T) {
	dc := newFakeInferenceStackDynamicClient()
	handler := &InferenceHandler{Provider: inference.NewMockProvider(), DynamicClient: dc}

	r := chi.NewRouter()
	r.Put("/inference/epp", handler.UpdateEPP)

	body := UpdateEPPRequest{Strategy: "least-load"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/inference/epp", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInferenceHandler_UpdateEPP_MissingStrategy(t *testing.T) {
	dc := newFakeInferenceStackDynamicClient()
	handler := &InferenceHandler{Provider: inference.NewMockProvider(), DynamicClient: dc}

	r := chi.NewRouter()
	r.Put("/inference/epp", handler.UpdateEPP)

	body := UpdateEPPRequest{Pool: "some-pool"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/inference/epp", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInferenceHandler_UpdateEPP_NilDynamicClient(t *testing.T) {
	handler := &InferenceHandler{Provider: inference.NewMockProvider()}

	r := chi.NewRouter()
	r.Put("/inference/epp", handler.UpdateEPP)

	body := UpdateEPPRequest{Pool: "pool", Strategy: "least-load"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/inference/epp", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInferenceHandler_UpdateEPP_PoolNotFound(t *testing.T) {
	dc := newFakeInferenceStackDynamicClient()
	handler := &InferenceHandler{Provider: inference.NewMockProvider(), DynamicClient: dc}

	r := chi.NewRouter()
	r.Put("/inference/epp", handler.UpdateEPP)

	body := UpdateEPPRequest{Pool: "nonexistent", Strategy: "least-load"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/inference/epp", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInferenceHandler_UpdateEPP_InvalidJSON(t *testing.T) {
	dc := newFakeInferenceStackDynamicClient()
	handler := &InferenceHandler{Provider: inference.NewMockProvider(), DynamicClient: dc}

	r := chi.NewRouter()
	r.Put("/inference/epp", handler.UpdateEPP)

	req := httptest.NewRequest(http.MethodPut, "/inference/epp", bytes.NewReader([]byte("{bad")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInferenceHandler_UpdateEPP_WithoutWeights(t *testing.T) {
	obj := newTestInferenceStack("epp-no-weights", "inference")
	dc := newFakeInferenceStackDynamicClient(obj)
	handler := &InferenceHandler{Provider: inference.NewMockProvider(), DynamicClient: dc}

	r := chi.NewRouter()
	r.Put("/inference/epp", handler.UpdateEPP)

	body := UpdateEPPRequest{
		Pool:     "epp-no-weights",
		Strategy: "random",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/inference/epp", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp EPPConfigResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Strategy != "random" {
		t.Errorf("expected strategy random, got %s", resp.Strategy)
	}
}

// ---------------------------------------------------------------------------
// GetAutoscaling
// ---------------------------------------------------------------------------

func TestInferenceHandler_GetAutoscaling_HappyPath(t *testing.T) {
	obj := newTestInferenceStack("as-pool", "inference")
	dc := newFakeInferenceStackDynamicClient(obj)
	handler := &InferenceHandler{Provider: inference.NewMockProvider(), DynamicClient: dc}

	r := chi.NewRouter()
	r.Get("/inference/autoscaling", handler.GetAutoscaling)

	req := httptest.NewRequest(http.MethodGet, "/inference/autoscaling?pool=as-pool", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp AutoscalingConfigResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Pool != "as-pool" {
		t.Errorf("expected pool as-pool, got %s", resp.Pool)
	}
	if resp.Replicas != 2 {
		t.Errorf("expected replicas 2, got %d", resp.Replicas)
	}
	if resp.MinReplicas != 1 {
		t.Errorf("expected minReplicas 1, got %d", resp.MinReplicas)
	}
	if resp.MaxReplicas != 8 {
		t.Errorf("expected maxReplicas 8, got %d", resp.MaxReplicas)
	}
}

func TestInferenceHandler_GetAutoscaling_MissingPool(t *testing.T) {
	handler := &InferenceHandler{Provider: inference.NewMockProvider()}

	r := chi.NewRouter()
	r.Get("/inference/autoscaling", handler.GetAutoscaling)

	req := httptest.NewRequest(http.MethodGet, "/inference/autoscaling", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInferenceHandler_GetAutoscaling_PoolNotFound(t *testing.T) {
	dc := newFakeInferenceStackDynamicClient()
	handler := &InferenceHandler{Provider: inference.NewMockProvider(), DynamicClient: dc}

	r := chi.NewRouter()
	r.Get("/inference/autoscaling", handler.GetAutoscaling)

	req := httptest.NewRequest(http.MethodGet, "/inference/autoscaling?pool=nonexistent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// UpdateAutoscaling
// ---------------------------------------------------------------------------

func TestInferenceHandler_UpdateAutoscaling_HappyPath(t *testing.T) {
	obj := newTestInferenceStack("as-update", "inference")
	dc := newFakeInferenceStackDynamicClient(obj)
	handler := &InferenceHandler{Provider: inference.NewMockProvider(), DynamicClient: dc}

	r := chi.NewRouter()
	r.Put("/inference/autoscaling", handler.UpdateAutoscaling)

	minR := 2
	maxR := 10
	replicas := 4
	body := UpdateAutoscalingRequest{
		Pool:        "as-update",
		MinReplicas: &minR,
		MaxReplicas: &maxR,
		Replicas:    &replicas,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/inference/autoscaling", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp AutoscalingConfigResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Pool != "as-update" {
		t.Errorf("expected pool as-update, got %s", resp.Pool)
	}
	if resp.Replicas != 4 {
		t.Errorf("expected replicas 4, got %d", resp.Replicas)
	}
	if resp.MinReplicas != 2 {
		t.Errorf("expected minReplicas 2, got %d", resp.MinReplicas)
	}
	if resp.MaxReplicas != 10 {
		t.Errorf("expected maxReplicas 10, got %d", resp.MaxReplicas)
	}
}

func TestInferenceHandler_UpdateAutoscaling_PartialUpdate(t *testing.T) {
	obj := newTestInferenceStack("as-partial", "inference")
	dc := newFakeInferenceStackDynamicClient(obj)
	handler := &InferenceHandler{Provider: inference.NewMockProvider(), DynamicClient: dc}

	r := chi.NewRouter()
	r.Put("/inference/autoscaling", handler.UpdateAutoscaling)

	// Only update replicas, leave min/max unchanged.
	replicas := 6
	body := UpdateAutoscalingRequest{
		Pool:     "as-partial",
		Replicas: &replicas,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/inference/autoscaling", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp AutoscalingConfigResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Replicas != 6 {
		t.Errorf("expected replicas 6, got %d", resp.Replicas)
	}
	// Original min/max from newTestInferenceStack should be preserved.
	if resp.MinReplicas != 1 {
		t.Errorf("expected minReplicas 1 (unchanged), got %d", resp.MinReplicas)
	}
	if resp.MaxReplicas != 8 {
		t.Errorf("expected maxReplicas 8 (unchanged), got %d", resp.MaxReplicas)
	}
}

func TestInferenceHandler_UpdateAutoscaling_MissingPool(t *testing.T) {
	dc := newFakeInferenceStackDynamicClient()
	handler := &InferenceHandler{Provider: inference.NewMockProvider(), DynamicClient: dc}

	r := chi.NewRouter()
	r.Put("/inference/autoscaling", handler.UpdateAutoscaling)

	body := UpdateAutoscalingRequest{}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/inference/autoscaling", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInferenceHandler_UpdateAutoscaling_NilDynamicClient(t *testing.T) {
	handler := &InferenceHandler{Provider: inference.NewMockProvider()}

	r := chi.NewRouter()
	r.Put("/inference/autoscaling", handler.UpdateAutoscaling)

	body := UpdateAutoscalingRequest{Pool: "some-pool"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/inference/autoscaling", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInferenceHandler_UpdateAutoscaling_PoolNotFound(t *testing.T) {
	dc := newFakeInferenceStackDynamicClient()
	handler := &InferenceHandler{Provider: inference.NewMockProvider(), DynamicClient: dc}

	r := chi.NewRouter()
	r.Put("/inference/autoscaling", handler.UpdateAutoscaling)

	replicas := 3
	body := UpdateAutoscalingRequest{
		Pool:     "nonexistent",
		Replicas: &replicas,
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/inference/autoscaling", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInferenceHandler_UpdateAutoscaling_InvalidJSON(t *testing.T) {
	dc := newFakeInferenceStackDynamicClient()
	handler := &InferenceHandler{Provider: inference.NewMockProvider(), DynamicClient: dc}

	r := chi.NewRouter()
	r.Put("/inference/autoscaling", handler.UpdateAutoscaling)

	req := httptest.NewRequest(http.MethodPut, "/inference/autoscaling", bytes.NewReader([]byte("{bad")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

// ---------------------------------------------------------------------------
// findInferenceStackByName / getDynamicClient
// ---------------------------------------------------------------------------

func TestInferenceHandler_findInferenceStackByName_Found(t *testing.T) {
	s1 := newTestInferenceStack("find-me", "ns1")
	s2 := newTestInferenceStack("other", "ns2")
	dc := newFakeInferenceStackDynamicClient(s1, s2)
	handler := &InferenceHandler{Provider: inference.NewMockProvider(), DynamicClient: dc}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	obj, err := handler.findInferenceStackByName(req, "find-me")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if obj.GetName() != "find-me" {
		t.Errorf("expected name find-me, got %s", obj.GetName())
	}
	if obj.GetNamespace() != "ns1" {
		t.Errorf("expected namespace ns1, got %s", obj.GetNamespace())
	}
}

func TestInferenceHandler_findInferenceStackByName_NotFound(t *testing.T) {
	dc := newFakeInferenceStackDynamicClient()
	handler := &InferenceHandler{Provider: inference.NewMockProvider(), DynamicClient: dc}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	_, err := handler.findInferenceStackByName(req, "missing")
	if err == nil {
		t.Fatal("expected error for missing stack")
	}
}

func TestInferenceHandler_findInferenceStackByName_NilClient(t *testing.T) {
	handler := &InferenceHandler{Provider: inference.NewMockProvider()}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	_, err := handler.findInferenceStackByName(req, "anything")
	if err == nil {
		t.Fatal("expected error when no dynamic client available")
	}
}

func TestInferenceHandler_getDynamicClient_UsesHandlerField(t *testing.T) {
	dc := newFakeInferenceStackDynamicClient()
	handler := &InferenceHandler{Provider: inference.NewMockProvider(), DynamicClient: dc}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	result := handler.getDynamicClient(req)
	if result == nil {
		t.Fatal("expected non-nil dynamic client from handler field")
	}
}

func TestInferenceHandler_getDynamicClient_NilFallback(t *testing.T) {
	handler := &InferenceHandler{Provider: inference.NewMockProvider()}

	// No cluster context in the request, so fallback should also be nil.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	result := handler.getDynamicClient(req)
	if result != nil {
		t.Fatal("expected nil dynamic client when no handler field or cluster context")
	}
}
