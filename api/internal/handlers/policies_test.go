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
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
	"github.com/kubenetlabs/ngc/api/internal/kubernetes"
)

func newFakePolicyDynamicClient(objects ...runtime.Object) *fakedynamic.FakeDynamicClient {
	s := runtime.NewScheme()
	// Register all 4 policy types
	for _, gvk := range []schema.GroupVersionKind{
		{Group: "gateway.nginx.org", Version: "v1alpha1", Kind: "RateLimitPolicy"},
		{Group: "gateway.nginx.org", Version: "v1alpha1", Kind: "RateLimitPolicyList"},
		{Group: "gateway.nginx.org", Version: "v1alpha1", Kind: "ClientSettingsPolicy"},
		{Group: "gateway.nginx.org", Version: "v1alpha1", Kind: "ClientSettingsPolicyList"},
		{Group: "gateway.networking.k8s.io", Version: "v1alpha3", Kind: "BackendTLSPolicy"},
		{Group: "gateway.networking.k8s.io", Version: "v1alpha3", Kind: "BackendTLSPolicyList"},
		{Group: "gateway.nginx.org", Version: "v1alpha1", Kind: "ObservabilityPolicy"},
		{Group: "gateway.nginx.org", Version: "v1alpha1", Kind: "ObservabilityPolicyList"},
	} {
		if gvk.Kind[len(gvk.Kind)-4:] == "List" {
			s.AddKnownTypeWithName(gvk, &unstructured.UnstructuredList{})
		} else {
			s.AddKnownTypeWithName(gvk, &unstructured.Unstructured{})
		}
	}
	return fakedynamic.NewSimpleDynamicClient(s, objects...)
}

func newTestPolicy(name, namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "gateway.nginx.org/v1alpha1",
			"kind":       "RateLimitPolicy",
			"metadata":   map[string]any{"name": name, "namespace": namespace},
			"spec": map[string]any{
				"rate":  "10r/s",
				"burst": int64(20),
			},
		},
	}
}

func policyContextMiddleware(dc *fakedynamic.FakeDynamicClient) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scheme := runtime.NewScheme()
			fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
			k8s := kubernetes.NewForTestWithDynamic(fakeClient, dc)
			ctx := cluster.WithClient(r.Context(), k8s)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func TestPolicyHandler_List_HappyPath(t *testing.T) {
	p1 := newTestPolicy("rate-1", "default")
	dc := newFakePolicyDynamicClient(p1)
	handler := &PolicyHandler{}

	r := chi.NewRouter()
	r.Use(policyContextMiddleware(dc))
	r.Get("/policies/{type}", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/policies/ratelimit", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []PolicyResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp) != 1 {
		t.Fatalf("expected 1 policy, got %d", len(resp))
	}
}

func TestPolicyHandler_List_UnknownType(t *testing.T) {
	dc := newFakePolicyDynamicClient()
	handler := &PolicyHandler{}

	r := chi.NewRouter()
	r.Use(policyContextMiddleware(dc))
	r.Get("/policies/{type}", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/policies/bogus", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPolicyHandler_List_NoClusterContext(t *testing.T) {
	handler := &PolicyHandler{}

	r := chi.NewRouter()
	// No middleware — no cluster context
	r.Get("/policies/{type}", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/policies/ratelimit", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPolicyHandler_Get_HappyPath(t *testing.T) {
	p1 := newTestPolicy("rate-1", "default")
	dc := newFakePolicyDynamicClient(p1)
	handler := &PolicyHandler{}

	r := chi.NewRouter()
	r.Use(policyContextMiddleware(dc))
	r.Get("/policies/{type}/{name}", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/policies/ratelimit/rate-1?namespace=default", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp PolicyResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Name != "rate-1" {
		t.Errorf("expected name rate-1, got %s", resp.Name)
	}
}

func TestPolicyHandler_Get_NotFound(t *testing.T) {
	dc := newFakePolicyDynamicClient()
	handler := &PolicyHandler{}

	r := chi.NewRouter()
	r.Use(policyContextMiddleware(dc))
	r.Get("/policies/{type}/{name}", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/policies/ratelimit/nonexistent?namespace=default", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPolicyHandler_Create_HappyPath(t *testing.T) {
	dc := newFakePolicyDynamicClient()
	handler := &PolicyHandler{}

	r := chi.NewRouter()
	r.Use(policyContextMiddleware(dc))
	r.Post("/policies/{type}", handler.Create)

	body := CreatePolicyRequest{
		Name:      "new-rate",
		Namespace: "default",
		Spec:      map[string]interface{}{"rate": "10r/s"},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/policies/ratelimit", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPolicyHandler_Delete_HappyPath(t *testing.T) {
	p1 := newTestPolicy("rate-1", "default")
	dc := newFakePolicyDynamicClient(p1)
	handler := &PolicyHandler{}

	r := chi.NewRouter()
	r.Use(policyContextMiddleware(dc))
	r.Delete("/policies/{type}/{name}", handler.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/policies/ratelimit/rate-1?namespace=default", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPolicyHandler_Conflicts_Returns501(t *testing.T) {
	handler := &PolicyHandler{}

	r := chi.NewRouter()
	r.Get("/policies/{type}/conflicts", handler.Conflicts)

	req := httptest.NewRequest(http.MethodGet, "/policies/ratelimit/conflicts", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Fatalf("expected status 501, got %d: %s", w.Code, w.Body.String())
	}
}
