package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
