package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kubenetlabs/ngc/api/internal/kubernetes"
)

func setupSchemeWithAlpha2(t *testing.T) *runtime.Scheme {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add client-go scheme: %v", err)
	}
	if err := gatewayv1.Install(scheme); err != nil {
		t.Fatalf("failed to add gateway-api v1 scheme: %v", err)
	}
	if err := gatewayv1alpha2.Install(scheme); err != nil {
		t.Fatalf("failed to add gateway-api v1alpha2 scheme: %v", err)
	}
	return scheme
}

func TestTLSRouteHandler_List_Empty(t *testing.T) {
	scheme := setupSchemeWithAlpha2(t)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	k8sClient := kubernetes.NewForTest(fakeClient)
	handler := &TLSRouteHandler{}

	r := chi.NewRouter()
	r.Use(contextMiddleware(k8sClient))
	r.Get("/api/v1/tlsroutes", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tlsroutes", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp []TLSRouteResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp) != 0 {
		t.Fatalf("expected 0 tls routes, got %d", len(resp))
	}
}

func TestTLSRouteHandler_List_WithData(t *testing.T) {
	scheme := setupSchemeWithAlpha2(t)

	tlsRoute1 := &gatewayv1alpha2.TLSRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tls-route-1",
			Namespace: "ns1",
		},
		Spec: gatewayv1alpha2.TLSRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{{Name: "my-gw"}},
			},
			Hostnames: []gatewayv1alpha2.Hostname{"example.com"},
			Rules: []gatewayv1alpha2.TLSRouteRule{
				{
					BackendRefs: []gatewayv1.BackendRef{
						{
							BackendObjectReference: gatewayv1.BackendObjectReference{
								Name: "svc-1",
							},
						},
					},
				},
			},
		},
	}
	tlsRoute2 := &gatewayv1alpha2.TLSRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tls-route-2",
			Namespace: "ns2",
		},
		Spec: gatewayv1alpha2.TLSRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{{Name: "my-gw"}},
			},
			Rules: []gatewayv1alpha2.TLSRouteRule{
				{
					BackendRefs: []gatewayv1.BackendRef{
						{
							BackendObjectReference: gatewayv1.BackendObjectReference{
								Name: "svc-2",
							},
						},
					},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tlsRoute1, tlsRoute2).Build()
	k8sClient := kubernetes.NewForTest(fakeClient)
	handler := &TLSRouteHandler{}

	t.Run("list all tls routes", func(t *testing.T) {
		r := chi.NewRouter()
		r.Use(contextMiddleware(k8sClient))
		r.Get("/api/v1/tlsroutes", handler.List)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/tlsroutes", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp []TLSRouteResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(resp) != 2 {
			t.Fatalf("expected 2 tls routes, got %d", len(resp))
		}
	})

	t.Run("list tls routes with namespace filter", func(t *testing.T) {
		r := chi.NewRouter()
		r.Use(contextMiddleware(k8sClient))
		r.Get("/api/v1/tlsroutes", handler.List)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/tlsroutes?namespace=ns1", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp []TLSRouteResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(resp) != 1 {
			t.Fatalf("expected 1 tls route, got %d", len(resp))
		}
		if resp[0].Name != "tls-route-1" {
			t.Errorf("expected tls-route-1, got %s", resp[0].Name)
		}
		if resp[0].Namespace != "ns1" {
			t.Errorf("expected namespace ns1, got %s", resp[0].Namespace)
		}
	})
}

func TestTLSRouteHandler_Get_Found(t *testing.T) {
	scheme := setupSchemeWithAlpha2(t)

	tlsRoute := &gatewayv1alpha2.TLSRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-tls-route",
			Namespace: "default",
		},
		Spec: gatewayv1alpha2.TLSRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{{Name: "my-gw"}},
			},
			Hostnames: []gatewayv1alpha2.Hostname{"secure.example.com"},
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tlsRoute).Build()
	k8sClient := kubernetes.NewForTest(fakeClient)
	handler := &TLSRouteHandler{}

	r := chi.NewRouter()
	r.Use(contextMiddleware(k8sClient))
	r.Get("/{namespace}/{name}", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/default/my-tls-route", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp TLSRouteResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Name != "my-tls-route" {
		t.Errorf("expected name my-tls-route, got %s", resp.Name)
	}
	if resp.Namespace != "default" {
		t.Errorf("expected namespace default, got %s", resp.Namespace)
	}
}

func TestTLSRouteHandler_Get_NotFound(t *testing.T) {
	scheme := setupSchemeWithAlpha2(t)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	k8sClient := kubernetes.NewForTest(fakeClient)
	handler := &TLSRouteHandler{}

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
}

func TestTLSRouteHandler_Create_HappyPath(t *testing.T) {
	scheme := setupSchemeWithAlpha2(t)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	k8sClient := kubernetes.NewForTest(fakeClient)
	handler := &TLSRouteHandler{}

	r := chi.NewRouter()
	r.Use(contextMiddleware(k8sClient))
	r.Post("/api/v1/tlsroutes", handler.Create)

	body := `{
		"name": "my-tls-route",
		"namespace": "default",
		"parentRefs": [{"name": "my-gw"}],
		"hostnames": ["secure.example.com"],
		"rules": [
			{
				"backendRefs": [{"name": "tls-svc", "port": 443}]
			}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tlsroutes", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp TLSRouteResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Name != "my-tls-route" {
		t.Errorf("expected name my-tls-route, got %s", resp.Name)
	}
	if resp.Namespace != "default" {
		t.Errorf("expected namespace default, got %s", resp.Namespace)
	}
	if len(resp.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(resp.Rules))
	}
}

func TestTLSRouteHandler_Create_BadRequest(t *testing.T) {
	scheme := setupSchemeWithAlpha2(t)
	handler := &TLSRouteHandler{}

	t.Run("missing required fields", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		k8sClient := kubernetes.NewForTest(fakeClient)

		r := chi.NewRouter()
		r.Use(contextMiddleware(k8sClient))
		r.Post("/api/v1/tlsroutes", handler.Create)

		body := `{"name": "my-tls-route"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tlsroutes", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		k8sClient := kubernetes.NewForTest(fakeClient)

		r := chi.NewRouter()
		r.Use(contextMiddleware(k8sClient))
		r.Post("/api/v1/tlsroutes", handler.Create)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/tlsroutes", strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})
}

func TestTLSRouteHandler_Delete_HappyPath(t *testing.T) {
	scheme := setupSchemeWithAlpha2(t)

	existing := &gatewayv1alpha2.TLSRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tls-route-to-delete",
			Namespace: "default",
		},
		Spec: gatewayv1alpha2.TLSRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{{Name: "my-gw"}},
			},
		},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()
	k8sClient := kubernetes.NewForTest(fakeClient)
	handler := &TLSRouteHandler{}

	r := chi.NewRouter()
	r.Use(contextMiddleware(k8sClient))
	r.Delete("/{namespace}/{name}", handler.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/default/tls-route-to-delete", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["message"] != "tlsroute deleted" {
		t.Errorf("expected 'tlsroute deleted' message, got %s", resp["message"])
	}
}

func TestTLSRouteHandler_List_NoClusterContext(t *testing.T) {
	handler := &TLSRouteHandler{}

	r := chi.NewRouter()
	// No contextMiddleware — no cluster client in context.
	r.Get("/api/v1/tlsroutes", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tlsroutes", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["error"] == "" {
		t.Error("expected error message")
	}
}
