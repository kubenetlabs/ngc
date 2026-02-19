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

func TestGRPCRouteHandler_List_Empty(t *testing.T) {
	scheme := setupScheme(t)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	k8sClient := kubernetes.NewForTest(fakeClient)
	handler := &GRPCRouteHandler{}

	r := chi.NewRouter()
	r.Use(contextMiddleware(k8sClient))
	r.Get("/api/v1/grpcroutes", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/grpcroutes", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp []GRPCRouteResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp) != 0 {
		t.Fatalf("expected 0 grpc routes, got %d", len(resp))
	}
}

func TestGRPCRouteHandler_List_WithData(t *testing.T) {
	scheme := setupScheme(t)

	port80 := gatewayv1.PortNumber(80)
	grpcRoute1 := &gatewayv1.GRPCRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grpc-route-1",
			Namespace: "ns1",
		},
		Spec: gatewayv1.GRPCRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{{Name: "my-gw"}},
			},
			Rules: []gatewayv1.GRPCRouteRule{
				{
					BackendRefs: []gatewayv1.GRPCBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name: "svc-1",
									Port: &port80,
								},
							},
						},
					},
				},
			},
		},
	}
	grpcRoute2 := &gatewayv1.GRPCRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grpc-route-2",
			Namespace: "ns2",
		},
		Spec: gatewayv1.GRPCRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{{Name: "my-gw"}},
			},
			Rules: []gatewayv1.GRPCRouteRule{
				{
					BackendRefs: []gatewayv1.GRPCBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name: "svc-2",
									Port: &port80,
								},
							},
						},
					},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(grpcRoute1, grpcRoute2).Build()
	k8sClient := kubernetes.NewForTest(fakeClient)
	handler := &GRPCRouteHandler{}

	t.Run("list all grpc routes", func(t *testing.T) {
		r := chi.NewRouter()
		r.Use(contextMiddleware(k8sClient))
		r.Get("/api/v1/grpcroutes", handler.List)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/grpcroutes", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp []GRPCRouteResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(resp) != 2 {
			t.Fatalf("expected 2 grpc routes, got %d", len(resp))
		}
	})

	t.Run("list grpc routes with namespace filter", func(t *testing.T) {
		r := chi.NewRouter()
		r.Use(contextMiddleware(k8sClient))
		r.Get("/api/v1/grpcroutes", handler.List)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/grpcroutes?namespace=ns1", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp []GRPCRouteResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(resp) != 1 {
			t.Fatalf("expected 1 grpc route, got %d", len(resp))
		}
		if resp[0].Name != "grpc-route-1" {
			t.Errorf("expected grpc-route-1, got %s", resp[0].Name)
		}
		if resp[0].Namespace != "ns1" {
			t.Errorf("expected namespace ns1, got %s", resp[0].Namespace)
		}
	})
}

func TestGRPCRouteHandler_Get_Found(t *testing.T) {
	scheme := setupScheme(t)

	grpcRoute := &gatewayv1.GRPCRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-grpc-route",
			Namespace: "default",
		},
		Spec: gatewayv1.GRPCRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{{Name: "my-gw"}},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(grpcRoute).Build()
	k8sClient := kubernetes.NewForTest(fakeClient)
	handler := &GRPCRouteHandler{}

	r := chi.NewRouter()
	r.Use(contextMiddleware(k8sClient))
	r.Get("/{namespace}/{name}", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/default/my-grpc-route", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp GRPCRouteResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Name != "my-grpc-route" {
		t.Errorf("expected name my-grpc-route, got %s", resp.Name)
	}
	if resp.Namespace != "default" {
		t.Errorf("expected namespace default, got %s", resp.Namespace)
	}
}

func TestGRPCRouteHandler_Get_NotFound(t *testing.T) {
	scheme := setupScheme(t)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	k8sClient := kubernetes.NewForTest(fakeClient)
	handler := &GRPCRouteHandler{}

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

func TestGRPCRouteHandler_Create_HappyPath(t *testing.T) {
	scheme := setupScheme(t)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	k8sClient := kubernetes.NewForTest(fakeClient)
	handler := &GRPCRouteHandler{}

	r := chi.NewRouter()
	r.Use(contextMiddleware(k8sClient))
	r.Post("/api/v1/grpcroutes", handler.Create)

	body := `{
		"name": "my-grpc-route",
		"namespace": "default",
		"parentRefs": [{"name": "my-gw"}],
		"rules": [
			{
				"matches": [{"method": {"type": "Exact", "service": "mypackage.MyService", "method": "DoThing"}}],
				"backendRefs": [{"name": "grpc-svc", "port": 50051}]
			}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/grpcroutes", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp GRPCRouteResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Name != "my-grpc-route" {
		t.Errorf("expected name my-grpc-route, got %s", resp.Name)
	}
	if resp.Namespace != "default" {
		t.Errorf("expected namespace default, got %s", resp.Namespace)
	}
	if len(resp.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(resp.Rules))
	}
}

func TestGRPCRouteHandler_Create_BadRequest(t *testing.T) {
	scheme := setupScheme(t)
	handler := &GRPCRouteHandler{}

	t.Run("missing required fields", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		k8sClient := kubernetes.NewForTest(fakeClient)

		r := chi.NewRouter()
		r.Use(contextMiddleware(k8sClient))
		r.Post("/api/v1/grpcroutes", handler.Create)

		body := `{"name": "my-grpc-route"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/grpcroutes", strings.NewReader(body))
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
		r.Post("/api/v1/grpcroutes", handler.Create)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/grpcroutes", strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})
}

func TestGRPCRouteHandler_Delete_HappyPath(t *testing.T) {
	scheme := setupScheme(t)

	existing := &gatewayv1.GRPCRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "grpc-route-to-delete",
			Namespace: "default",
		},
		Spec: gatewayv1.GRPCRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{{Name: "my-gw"}},
			},
		},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()
	k8sClient := kubernetes.NewForTest(fakeClient)
	handler := &GRPCRouteHandler{}

	r := chi.NewRouter()
	r.Use(contextMiddleware(k8sClient))
	r.Delete("/{namespace}/{name}", handler.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/default/grpc-route-to-delete", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["message"] != "grpcroute deleted" {
		t.Errorf("expected 'grpcroute deleted' message, got %s", resp["message"])
	}
}

func TestGRPCRouteHandler_List_NoClusterContext(t *testing.T) {
	handler := &GRPCRouteHandler{}

	r := chi.NewRouter()
	// No contextMiddleware — no cluster client in context.
	r.Get("/api/v1/grpcroutes", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/grpcroutes", nil)
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
