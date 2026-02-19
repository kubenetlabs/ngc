package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakedynamic "k8s.io/client-go/dynamic/fake"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
	"github.com/kubenetlabs/ngc/api/internal/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newFakeXCDynamicClient(objects ...*unstructured.Unstructured) *fakedynamic.FakeDynamicClient {
	s := runtime.NewScheme()
	s.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "ngf-console.f5.com", Version: "v1alpha1", Kind: "DistributedCloudPublish"},
		&unstructured.Unstructured{},
	)
	s.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "ngf-console.f5.com", Version: "v1alpha1", Kind: "DistributedCloudPublishList"},
		&unstructured.UnstructuredList{},
	)
	// Use custom list kinds to handle the irregular plural "distributedcloudpublishes"
	gvr := schema.GroupVersionResource{Group: "ngf-console.f5.com", Version: "v1alpha1", Resource: "distributedcloudpublishes"}
	dc := fakedynamic.NewSimpleDynamicClientWithCustomListKinds(s,
		map[schema.GroupVersionResource]string{
			gvr: "DistributedCloudPublishList",
		},
	)
	// Pre-create objects via the dynamic client to use the correct GVR
	for _, obj := range objects {
		ns := obj.GetNamespace()
		_, _ = dc.Resource(gvr).Namespace(ns).Create(context.Background(), obj, metav1.CreateOptions{})
	}
	return dc
}

func newTestXCPublish(name, namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "ngf-console.f5.com/v1alpha1",
			"kind":       "DistributedCloudPublish",
			"metadata":   map[string]any{"name": name, "namespace": namespace},
			"spec": map[string]any{
				"httpRouteRef": "test-route",
			},
			"status": map[string]any{
				"phase": "Published",
			},
		},
	}
}

// xcContextMiddleware injects a kubernetes client with dynamic client for XC tests.
func xcContextMiddleware(dc *fakedynamic.FakeDynamicClient) func(http.Handler) http.Handler {
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

func TestXCHandler_Status_Connected(t *testing.T) {
	p1 := newTestXCPublish("pub-1", "default")
	p2 := newTestXCPublish("pub-2", "default")
	dc := newFakeXCDynamicClient(p1, p2)
	handler := &XCHandler{}

	r := chi.NewRouter()
	r.Use(xcContextMiddleware(dc))
	r.Get("/xc/status", handler.Status)

	req := httptest.NewRequest(http.MethodGet, "/xc/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp XCStatusResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Connected {
		t.Error("expected connected=true")
	}
	if resp.PublishCount != 2 {
		t.Errorf("expected publishCount=2, got %d", resp.PublishCount)
	}
}

func TestXCHandler_Status_NoClusterContext(t *testing.T) {
	handler := &XCHandler{}

	r := chi.NewRouter()
	r.Get("/xc/status", handler.Status)

	req := httptest.NewRequest(http.MethodGet, "/xc/status", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", w.Code, w.Body.String())
	}
}

func TestXCHandler_Publish_HappyPath(t *testing.T) {
	dc := newFakeXCDynamicClient()
	handler := &XCHandler{}

	r := chi.NewRouter()
	r.Use(xcContextMiddleware(dc))
	r.Post("/xc/publish", handler.Publish)

	body := XCPublishRequest{
		Name:         "test-pub",
		Namespace:    "default",
		HTTPRouteRef: "my-route",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/xc/publish", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp XCPublishResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Name != "test-pub" {
		t.Errorf("expected name test-pub, got %s", resp.Name)
	}
}

func TestXCHandler_Publish_MissingFields(t *testing.T) {
	dc := newFakeXCDynamicClient()
	handler := &XCHandler{}

	r := chi.NewRouter()
	r.Use(xcContextMiddleware(dc))
	r.Post("/xc/publish", handler.Publish)

	body := XCPublishRequest{
		Name: "test-pub",
		// Missing HTTPRouteRef
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/xc/publish", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestXCHandler_GetPublish_HappyPath(t *testing.T) {
	p1 := newTestXCPublish("pub-1", "default")
	dc := newFakeXCDynamicClient(p1)
	handler := &XCHandler{}

	r := chi.NewRouter()
	r.Use(xcContextMiddleware(dc))
	r.Get("/xc/publish/{id}", handler.GetPublish)

	// parsePublishID with no "/" defaults to "default" namespace
	req := httptest.NewRequest(http.MethodGet, "/xc/publish/pub-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp XCPublishResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Name != "pub-1" {
		t.Errorf("expected name pub-1, got %s", resp.Name)
	}
}

func TestXCHandler_DeletePublish_HappyPath(t *testing.T) {
	p1 := newTestXCPublish("pub-1", "default")
	dc := newFakeXCDynamicClient(p1)
	handler := &XCHandler{}

	r := chi.NewRouter()
	r.Use(xcContextMiddleware(dc))
	r.Delete("/xc/publish/{id}", handler.DeletePublish)

	req := httptest.NewRequest(http.MethodDelete, "/xc/publish/pub-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestXCHandler_Metrics_NoPublishes(t *testing.T) {
	// With no cluster context (nil dynamic client), Metrics should return zeros
	handler := &XCHandler{}

	r := chi.NewRouter()
	r.Get("/xc/metrics", handler.Metrics)

	req := httptest.NewRequest(http.MethodGet, "/xc/metrics", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp XCMetricsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.TotalRequests != 0 {
		t.Errorf("expected totalRequests=0, got %d", resp.TotalRequests)
	}
	if len(resp.Regions) != 0 {
		t.Errorf("expected 0 regions, got %d", len(resp.Regions))
	}
}

func TestXCHandler_Metrics_WithPublishesNoProm(t *testing.T) {
	// With publishes but no Prometheus client, Metrics should return zeros
	p1 := newTestXCPublish("pub-1", "default")
	dc := newFakeXCDynamicClient(p1)
	handler := &XCHandler{Prom: nil}

	r := chi.NewRouter()
	r.Use(xcContextMiddleware(dc))
	r.Get("/xc/metrics", handler.Metrics)

	req := httptest.NewRequest(http.MethodGet, "/xc/metrics", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp XCMetricsResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.TotalRequests != 0 {
		t.Errorf("expected totalRequests=0, got %d", resp.TotalRequests)
	}
	if resp.ErrorRate != 0 {
		t.Errorf("expected errorRate=0, got %f", resp.ErrorRate)
	}
	if len(resp.Regions) != 0 {
		t.Errorf("expected 0 regions, got %d", len(resp.Regions))
	}
}
