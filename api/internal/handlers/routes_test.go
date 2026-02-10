package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kubenetlabs/ngc/api/internal/kubernetes"
)

func TestRouteHandler_List(t *testing.T) {
	scheme := setupScheme(t)

	route1 := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "route-1",
			Namespace: "ns1",
		},
		Spec: gatewayv1.HTTPRouteSpec{
			Rules: []gatewayv1.HTTPRouteRule{
				{
					BackendRefs: []gatewayv1.HTTPBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name: "service-1",
								},
							},
						},
					},
				},
			},
		},
	}
	route2 := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "route-2",
			Namespace: "ns2",
		},
		Spec: gatewayv1.HTTPRouteSpec{
			Rules: []gatewayv1.HTTPRouteRule{
				{
					BackendRefs: []gatewayv1.HTTPBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name: "service-2",
								},
							},
						},
					},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(route1, route2).Build()
	k8sClient := kubernetes.NewForTest(fakeClient)
	handler := &RouteHandler{}

	t.Run("list all routes", func(t *testing.T) {
		r := chi.NewRouter()
		r.Use(contextMiddleware(k8sClient))
		r.Get("/api/v1/httproutes", handler.List)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/httproutes", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp []HTTPRouteResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(resp) != 2 {
			t.Fatalf("expected 2 routes, got %d", len(resp))
		}
	})

	t.Run("list routes with namespace filter", func(t *testing.T) {
		r := chi.NewRouter()
		r.Use(contextMiddleware(k8sClient))
		r.Get("/api/v1/httproutes", handler.List)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/httproutes?namespace=ns1", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp []HTTPRouteResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(resp) != 1 {
			t.Fatalf("expected 1 route, got %d", len(resp))
		}
		if resp[0].Name != "route-1" {
			t.Errorf("expected route-1, got %s", resp[0].Name)
		}
		if resp[0].Namespace != "ns1" {
			t.Errorf("expected namespace ns1, got %s", resp[0].Namespace)
		}
	})
}

func TestRouteHandler_Get(t *testing.T) {
	scheme := setupScheme(t)

	route := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-route",
			Namespace: "test-ns",
		},
		Spec: gatewayv1.HTTPRouteSpec{
			Hostnames: []gatewayv1.Hostname{"example.com"},
			Rules: []gatewayv1.HTTPRouteRule{
				{
					BackendRefs: []gatewayv1.HTTPBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name: "test-service",
								},
							},
						},
					},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(route).Build()
	k8sClient := kubernetes.NewForTest(fakeClient)
	handler := &RouteHandler{}

	t.Run("successful get", func(t *testing.T) {
		r := chi.NewRouter()
		r.Use(contextMiddleware(k8sClient))
		r.Get("/{namespace}/{name}", handler.Get)

		req := httptest.NewRequest(http.MethodGet, "/test-ns/test-route", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp HTTPRouteResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Name != "test-route" {
			t.Errorf("expected name test-route, got %s", resp.Name)
		}
		if resp.Namespace != "test-ns" {
			t.Errorf("expected namespace test-ns, got %s", resp.Namespace)
		}
		if len(resp.Hostnames) != 1 {
			t.Errorf("expected 1 hostname, got %d", len(resp.Hostnames))
		}
		if resp.Hostnames[0] != "example.com" {
			t.Errorf("expected hostname example.com, got %s", resp.Hostnames[0])
		}
	})

	t.Run("route not found", func(t *testing.T) {
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

func TestRouteHandler_Create(t *testing.T) {
	scheme := setupScheme(t)
	handler := &RouteHandler{}

	t.Run("successful create", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		k8sClient := kubernetes.NewForTest(fakeClient)

		r := chi.NewRouter()
		r.Use(contextMiddleware(k8sClient))
		r.Post("/api/v1/httproutes", handler.Create)

		body := `{
			"name": "my-route",
			"namespace": "default",
			"parentRefs": [{"name": "my-gateway"}],
			"rules": [
				{
					"matches": [{"path": {"type": "PathPrefix", "value": "/"}}],
					"backendRefs": [{"name": "my-service", "port": 80}]
				}
			]
		}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/httproutes", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
		}

		var resp HTTPRouteResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Name != "my-route" {
			t.Errorf("expected name my-route, got %s", resp.Name)
		}
		if resp.Namespace != "default" {
			t.Errorf("expected namespace default, got %s", resp.Namespace)
		}
		if len(resp.Rules) != 1 {
			t.Fatalf("expected 1 rule, got %d", len(resp.Rules))
		}
	})

	t.Run("missing required fields", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		k8sClient := kubernetes.NewForTest(fakeClient)

		r := chi.NewRouter()
		r.Use(contextMiddleware(k8sClient))
		r.Post("/api/v1/httproutes", handler.Create)

		body := `{"name": "my-route"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/httproutes", strings.NewReader(body))
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
		r.Post("/api/v1/httproutes", handler.Create)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/httproutes", strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})
}

func TestRouteHandler_Update(t *testing.T) {
	scheme := setupScheme(t)
	handler := &RouteHandler{}

	t.Run("successful update", func(t *testing.T) {
		existing := &gatewayv1.HTTPRoute{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-route",
				Namespace: "default",
			},
			Spec: gatewayv1.HTTPRouteSpec{
				Rules: []gatewayv1.HTTPRouteRule{
					{
						BackendRefs: []gatewayv1.HTTPBackendRef{
							{
								BackendRef: gatewayv1.BackendRef{
									BackendObjectReference: gatewayv1.BackendObjectReference{
										Name: "old-service",
									},
								},
							},
						},
					},
				},
			},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()
		k8sClient := kubernetes.NewForTest(fakeClient)

		r := chi.NewRouter()
		r.Use(contextMiddleware(k8sClient))
		r.Put("/{namespace}/{name}", handler.Update)

		body := `{
			"parentRefs": [{"name": "my-gateway"}],
			"rules": [
				{
					"matches": [{"path": {"type": "PathPrefix", "value": "/api"}}],
					"backendRefs": [{"name": "new-service", "port": 8080}]
				},
				{
					"matches": [{"path": {"type": "Exact", "value": "/health"}}],
					"backendRefs": [{"name": "health-service", "port": 80}]
				}
			]
		}`
		req := httptest.NewRequest(http.MethodPut, "/default/test-route", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp HTTPRouteResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(resp.Rules) != 2 {
			t.Errorf("expected 2 rules after update, got %d", len(resp.Rules))
		}
	})

	t.Run("nonexistent route", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		k8sClient := kubernetes.NewForTest(fakeClient)

		r := chi.NewRouter()
		r.Use(contextMiddleware(k8sClient))
		r.Put("/{namespace}/{name}", handler.Update)

		body := `{
			"parentRefs": [{"name": "my-gateway"}],
			"rules": [{"backendRefs": [{"name": "svc", "port": 80}]}]
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

func TestRouteHandler_Delete(t *testing.T) {
	scheme := setupScheme(t)
	handler := &RouteHandler{}

	t.Run("successful delete", func(t *testing.T) {
		existing := &gatewayv1.HTTPRoute{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-route",
				Namespace: "default",
			},
			Spec: gatewayv1.HTTPRouteSpec{
				Rules: []gatewayv1.HTTPRouteRule{
					{
						BackendRefs: []gatewayv1.HTTPBackendRef{
							{
								BackendRef: gatewayv1.BackendRef{
									BackendObjectReference: gatewayv1.BackendObjectReference{
										Name: "service-1",
									},
								},
							},
						},
					},
				},
			},
		}
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()
		k8sClient := kubernetes.NewForTest(fakeClient)

		r := chi.NewRouter()
		r.Use(contextMiddleware(k8sClient))
		r.Delete("/{namespace}/{name}", handler.Delete)

		req := httptest.NewRequest(http.MethodDelete, "/default/test-route", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
		}

		var resp map[string]string
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp["message"] != "httproute deleted" {
			t.Errorf("expected 'httproute deleted' message, got %s", resp["message"])
		}
	})

	t.Run("nonexistent route", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		k8sClient := kubernetes.NewForTest(fakeClient)

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
