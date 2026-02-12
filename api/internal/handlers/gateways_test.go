package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
	"github.com/kubenetlabs/ngc/api/internal/kubernetes"
)

func setupScheme(t *testing.T) *runtime.Scheme {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add client-go scheme: %v", err)
	}
	if err := gatewayv1.Install(scheme); err != nil {
		t.Fatalf("failed to add gateway-api scheme: %v", err)
	}
	return scheme
}

// contextMiddleware injects a kubernetes client into the request context for testing.
func contextMiddleware(k8s *kubernetes.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := cluster.WithClient(r.Context(), k8s)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func TestGatewayHandler_List(t *testing.T) {
	scheme := setupScheme(t)

	gw1 := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gateway-1",
			Namespace: "ns1",
		},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: "nginx",
			Listeners: []gatewayv1.Listener{
				{
					Name:     "http",
					Port:     80,
					Protocol: gatewayv1.HTTPProtocolType,
				},
			},
		},
	}
	gw2 := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gateway-2",
			Namespace: "ns2",
		},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: "nginx",
			Listeners: []gatewayv1.Listener{
				{
					Name:     "http",
					Port:     80,
					Protocol: gatewayv1.HTTPProtocolType,
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(gw1, gw2).Build()
	k8sClient := kubernetes.NewForTest(fakeClient)
	handler := &GatewayHandler{}

	t.Run("list all gateways", func(t *testing.T) {
		r := chi.NewRouter()
		r.Use(contextMiddleware(k8sClient))
		r.Get("/api/v1/gateways", handler.List)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/gateways", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp []GatewayResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(resp) != 2 {
			t.Fatalf("expected 2 gateways, got %d", len(resp))
		}
	})

	t.Run("list gateways with namespace filter", func(t *testing.T) {
		r := chi.NewRouter()
		r.Use(contextMiddleware(k8sClient))
		r.Get("/api/v1/gateways", handler.List)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/gateways?namespace=ns1", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp []GatewayResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(resp) != 1 {
			t.Fatalf("expected 1 gateway, got %d", len(resp))
		}
		if resp[0].Name != "gateway-1" {
			t.Errorf("expected gateway-1, got %s", resp[0].Name)
		}
		if resp[0].Namespace != "ns1" {
			t.Errorf("expected namespace ns1, got %s", resp[0].Namespace)
		}
	})
}

func TestGatewayHandler_Get(t *testing.T) {
	scheme := setupScheme(t)

	gw := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-gateway",
			Namespace: "test-ns",
		},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: "nginx",
			Listeners: []gatewayv1.Listener{
				{
					Name:     "http",
					Port:     80,
					Protocol: gatewayv1.HTTPProtocolType,
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(gw).Build()
	k8sClient := kubernetes.NewForTest(fakeClient)
	handler := &GatewayHandler{}

	t.Run("successful get", func(t *testing.T) {
		r := chi.NewRouter()
		r.Use(contextMiddleware(k8sClient))
		r.Get("/{namespace}/{name}", handler.Get)

		req := httptest.NewRequest(http.MethodGet, "/test-ns/test-gateway", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp GatewayResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Name != "test-gateway" {
			t.Errorf("expected name test-gateway, got %s", resp.Name)
		}
		if resp.Namespace != "test-ns" {
			t.Errorf("expected namespace test-ns, got %s", resp.Namespace)
		}
		if resp.GatewayClassName != "nginx" {
			t.Errorf("expected gatewayClassName nginx, got %s", resp.GatewayClassName)
		}
	})

	t.Run("gateway not found", func(t *testing.T) {
		r := chi.NewRouter()
		r.Use(contextMiddleware(k8sClient))
		r.Get("/{namespace}/{name}", handler.Get)

		req := httptest.NewRequest(http.MethodGet, "/bad-ns/bad-name", nil)
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

func TestGatewayHandler_ListClasses(t *testing.T) {
	scheme := setupScheme(t)

	gc := &gatewayv1.GatewayClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "nginx",
		},
		Spec: gatewayv1.GatewayClassSpec{
			ControllerName: "gateway.nginx.org/nginx-gateway-controller",
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(gc).Build()
	k8sClient := kubernetes.NewForTest(fakeClient)
	handler := &GatewayHandler{}

	r := chi.NewRouter()
	r.Use(contextMiddleware(k8sClient))
	r.Get("/api/v1/gatewayclasses", handler.ListClasses)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/gatewayclasses", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp []GatewayClassResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp) != 1 {
		t.Fatalf("expected 1 gateway class, got %d", len(resp))
	}
	if resp[0].Name != "nginx" {
		t.Errorf("expected name nginx, got %s", resp[0].Name)
	}
	if resp[0].ControllerName != "gateway.nginx.org/nginx-gateway-controller" {
		t.Errorf("expected controller name, got %s", resp[0].ControllerName)
	}
}

func TestGatewayHandler_GetClass(t *testing.T) {
	scheme := setupScheme(t)

	gc := &gatewayv1.GatewayClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "nginx",
		},
		Spec: gatewayv1.GatewayClassSpec{
			ControllerName: "gateway.nginx.org/nginx-gateway-controller",
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(gc).Build()
	k8sClient := kubernetes.NewForTest(fakeClient)
	handler := &GatewayHandler{}

	t.Run("successful get", func(t *testing.T) {
		r := chi.NewRouter()
		r.Use(contextMiddleware(k8sClient))
		r.Get("/{name}", handler.GetClass)

		req := httptest.NewRequest(http.MethodGet, "/nginx", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp GatewayClassResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Name != "nginx" {
			t.Errorf("expected name nginx, got %s", resp.Name)
		}
	})

	t.Run("gateway class not found", func(t *testing.T) {
		r := chi.NewRouter()
		r.Use(contextMiddleware(k8sClient))
		r.Get("/{name}", handler.GetClass)

		req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
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

// newFakeDynamicClient creates a fake dynamic client with the GatewayBundle GVR registered.
func newFakeDynamicClient(objects ...runtime.Object) *fakedynamic.FakeDynamicClient {
	s := runtime.NewScheme()
	s.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "ngf-console.f5.com", Version: "v1alpha1", Kind: "GatewayBundle"},
		&unstructured.Unstructured{},
	)
	s.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "ngf-console.f5.com", Version: "v1alpha1", Kind: "GatewayBundleList"},
		&unstructured.UnstructuredList{},
	)
	return fakedynamic.NewSimpleDynamicClient(s, objects...)
}

func TestGatewayHandler_Create(t *testing.T) {
	scheme := setupScheme(t)
	handler := &GatewayHandler{}

	t.Run("successful create", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		dc := newFakeDynamicClient()
		k8sClient := kubernetes.NewForTestWithDynamic(fakeClient, dc)

		r := chi.NewRouter()
		r.Use(contextMiddleware(k8sClient))
		r.Post("/api/v1/gateways", handler.Create)

		body := `{
			"name": "my-gateway",
			"namespace": "default",
			"gatewayClassName": "nginx",
			"listeners": [{"name": "http", "port": 80, "protocol": "HTTP"}]
		}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/gateways", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
		}

		var resp GatewayBundleResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Name != "my-gateway" {
			t.Errorf("expected name my-gateway, got %s", resp.Name)
		}
		if resp.Namespace != "default" {
			t.Errorf("expected namespace default, got %s", resp.Namespace)
		}
		if resp.GatewayClassName != "nginx" {
			t.Errorf("expected gatewayClassName nginx, got %s", resp.GatewayClassName)
		}
		if len(resp.Listeners) != 1 {
			t.Fatalf("expected 1 listener, got %d", len(resp.Listeners))
		}
	})

	t.Run("missing required fields", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		dc := newFakeDynamicClient()
		k8sClient := kubernetes.NewForTestWithDynamic(fakeClient, dc)

		r := chi.NewRouter()
		r.Use(contextMiddleware(k8sClient))
		r.Post("/api/v1/gateways", handler.Create)

		body := `{"name": "my-gateway"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/gateways", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		dc := newFakeDynamicClient()
		k8sClient := kubernetes.NewForTestWithDynamic(fakeClient, dc)

		r := chi.NewRouter()
		r.Use(contextMiddleware(k8sClient))
		r.Post("/api/v1/gateways", handler.Create)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/gateways", strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})
}

func TestGatewayHandler_Update(t *testing.T) {
	scheme := setupScheme(t)
	handler := &GatewayHandler{}

	t.Run("successful update", func(t *testing.T) {
		// Create an existing GatewayBundle as unstructured for the dynamic client.
		existingBundle := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "ngf-console.f5.com/v1alpha1",
				"kind":       "GatewayBundle",
				"metadata": map[string]any{
					"name":      "test-gw",
					"namespace": "default",
				},
				"spec": map[string]any{
					"gatewayClassName": "nginx",
					"listeners": []any{
						map[string]any{"name": "http", "port": int64(80), "protocol": "HTTP"},
					},
				},
			},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		dc := newFakeDynamicClient(existingBundle)
		k8sClient := kubernetes.NewForTestWithDynamic(fakeClient, dc)

		r := chi.NewRouter()
		r.Use(contextMiddleware(k8sClient))
		r.Put("/{namespace}/{name}", handler.Update)

		body := `{
			"gatewayClassName": "nginx",
			"listeners": [
				{"name": "http", "port": 80, "protocol": "HTTP"},
				{"name": "https", "port": 443, "protocol": "HTTPS"}
			]
		}`
		req := httptest.NewRequest(http.MethodPut, "/default/test-gw", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp GatewayBundleResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(resp.Listeners) != 2 {
			t.Errorf("expected 2 listeners after update, got %d", len(resp.Listeners))
		}
	})

	t.Run("nonexistent gateway", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		dc := newFakeDynamicClient()
		k8sClient := kubernetes.NewForTestWithDynamic(fakeClient, dc)

		r := chi.NewRouter()
		r.Use(contextMiddleware(k8sClient))
		r.Put("/{namespace}/{name}", handler.Update)

		body := `{
			"gatewayClassName": "nginx",
			"listeners": [{"name": "http", "port": 80, "protocol": "HTTP"}]
		}`
		req := httptest.NewRequest(http.MethodPut, "/default/nonexistent", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Errorf("expected status 404, got %d", w.Code)
		}
	})
}

func TestGatewayHandler_Delete(t *testing.T) {
	scheme := setupScheme(t)
	handler := &GatewayHandler{}

	t.Run("successful delete", func(t *testing.T) {
		existingBundle := &unstructured.Unstructured{
			Object: map[string]any{
				"apiVersion": "ngf-console.f5.com/v1alpha1",
				"kind":       "GatewayBundle",
				"metadata": map[string]any{
					"name":      "test-gw",
					"namespace": "default",
				},
				"spec": map[string]any{
					"gatewayClassName": "nginx",
					"listeners": []any{
						map[string]any{"name": "http", "port": int64(80), "protocol": "HTTP"},
					},
				},
			},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		dc := newFakeDynamicClient(existingBundle)
		k8sClient := kubernetes.NewForTestWithDynamic(fakeClient, dc)

		r := chi.NewRouter()
		r.Use(contextMiddleware(k8sClient))
		r.Delete("/{namespace}/{name}", handler.Delete)

		req := httptest.NewRequest(http.MethodDelete, "/default/test-gw", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]string
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp["message"] != "gateway deleted" {
			t.Errorf("expected 'gateway deleted' message, got %s", resp["message"])
		}
	})

	t.Run("nonexistent gateway", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		dc := newFakeDynamicClient()
		k8sClient := kubernetes.NewForTestWithDynamic(fakeClient, dc)

		r := chi.NewRouter()
		r.Use(contextMiddleware(k8sClient))
		r.Delete("/{namespace}/{name}", handler.Delete)

		req := httptest.NewRequest(http.MethodDelete, "/default/nonexistent", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", w.Code)
		}
	})
}
