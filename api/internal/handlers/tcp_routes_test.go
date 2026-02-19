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
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kubenetlabs/ngc/api/internal/kubernetes"
)

func TestTCPRouteHandler_List_Empty(t *testing.T) {
	scheme := setupSchemeWithAlpha2(t)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	k8sClient := kubernetes.NewForTest(fakeClient)
	handler := &TCPRouteHandler{}

	r := chi.NewRouter()
	r.Use(contextMiddleware(k8sClient))
	r.Get("/api/v1/tcproutes", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tcproutes", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp []TCPRouteResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp) != 0 {
		t.Fatalf("expected 0 tcp routes, got %d", len(resp))
	}
}

func TestTCPRouteHandler_List_WithData(t *testing.T) {
	scheme := setupSchemeWithAlpha2(t)

	tcpRoute1 := &gatewayv1alpha2.TCPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tcp-route-1",
			Namespace: "ns1",
		},
		Spec: gatewayv1alpha2.TCPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{{Name: "my-gw"}},
			},
			Rules: []gatewayv1alpha2.TCPRouteRule{
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
	tcpRoute2 := &gatewayv1alpha2.TCPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tcp-route-2",
			Namespace: "ns2",
		},
		Spec: gatewayv1alpha2.TCPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{{Name: "my-gw"}},
			},
			Rules: []gatewayv1alpha2.TCPRouteRule{
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

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tcpRoute1, tcpRoute2).Build()
	k8sClient := kubernetes.NewForTest(fakeClient)
	handler := &TCPRouteHandler{}

	t.Run("list all tcp routes", func(t *testing.T) {
		r := chi.NewRouter()
		r.Use(contextMiddleware(k8sClient))
		r.Get("/api/v1/tcproutes", handler.List)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/tcproutes", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp []TCPRouteResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(resp) != 2 {
			t.Fatalf("expected 2 tcp routes, got %d", len(resp))
		}
	})

	t.Run("list tcp routes with namespace filter", func(t *testing.T) {
		r := chi.NewRouter()
		r.Use(contextMiddleware(k8sClient))
		r.Get("/api/v1/tcproutes", handler.List)

		req := httptest.NewRequest(http.MethodGet, "/api/v1/tcproutes?namespace=ns1", nil)
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp []TCPRouteResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(resp) != 1 {
			t.Fatalf("expected 1 tcp route, got %d", len(resp))
		}
		if resp[0].Name != "tcp-route-1" {
			t.Errorf("expected tcp-route-1, got %s", resp[0].Name)
		}
		if resp[0].Namespace != "ns1" {
			t.Errorf("expected namespace ns1, got %s", resp[0].Namespace)
		}
	})
}

func TestTCPRouteHandler_Get_Found(t *testing.T) {
	scheme := setupSchemeWithAlpha2(t)

	tcpRoute := &gatewayv1alpha2.TCPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-tcp-route",
			Namespace: "default",
		},
		Spec: gatewayv1alpha2.TCPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{{Name: "my-gw"}},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tcpRoute).Build()
	k8sClient := kubernetes.NewForTest(fakeClient)
	handler := &TCPRouteHandler{}

	r := chi.NewRouter()
	r.Use(contextMiddleware(k8sClient))
	r.Get("/{namespace}/{name}", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/default/my-tcp-route", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resp TCPRouteResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Name != "my-tcp-route" {
		t.Errorf("expected name my-tcp-route, got %s", resp.Name)
	}
	if resp.Namespace != "default" {
		t.Errorf("expected namespace default, got %s", resp.Namespace)
	}
}

func TestTCPRouteHandler_Get_NotFound(t *testing.T) {
	scheme := setupSchemeWithAlpha2(t)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	k8sClient := kubernetes.NewForTest(fakeClient)
	handler := &TCPRouteHandler{}

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

func TestTCPRouteHandler_Create_HappyPath(t *testing.T) {
	scheme := setupSchemeWithAlpha2(t)

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	k8sClient := kubernetes.NewForTest(fakeClient)
	handler := &TCPRouteHandler{}

	r := chi.NewRouter()
	r.Use(contextMiddleware(k8sClient))
	r.Post("/api/v1/tcproutes", handler.Create)

	body := `{
		"name": "my-tcp-route",
		"namespace": "default",
		"parentRefs": [{"name": "my-gw"}],
		"rules": [
			{
				"backendRefs": [{"name": "tcp-svc", "port": 3306}]
			}
		]
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tcproutes", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp TCPRouteResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Name != "my-tcp-route" {
		t.Errorf("expected name my-tcp-route, got %s", resp.Name)
	}
	if resp.Namespace != "default" {
		t.Errorf("expected namespace default, got %s", resp.Namespace)
	}
	if len(resp.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(resp.Rules))
	}
}

func TestTCPRouteHandler_Create_BadRequest(t *testing.T) {
	scheme := setupSchemeWithAlpha2(t)
	handler := &TCPRouteHandler{}

	t.Run("missing required fields", func(t *testing.T) {
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		k8sClient := kubernetes.NewForTest(fakeClient)

		r := chi.NewRouter()
		r.Use(contextMiddleware(k8sClient))
		r.Post("/api/v1/tcproutes", handler.Create)

		body := `{"name": "my-tcp-route"}`
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tcproutes", strings.NewReader(body))
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
		r.Post("/api/v1/tcproutes", handler.Create)

		req := httptest.NewRequest(http.MethodPost, "/api/v1/tcproutes", strings.NewReader("{invalid"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", w.Code)
		}
	})
}

func TestTCPRouteHandler_Delete_HappyPath(t *testing.T) {
	scheme := setupSchemeWithAlpha2(t)

	existing := &gatewayv1alpha2.TCPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tcp-route-to-delete",
			Namespace: "default",
		},
		Spec: gatewayv1alpha2.TCPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{{Name: "my-gw"}},
			},
		},
	}
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(existing).Build()
	k8sClient := kubernetes.NewForTest(fakeClient)
	handler := &TCPRouteHandler{}

	r := chi.NewRouter()
	r.Use(contextMiddleware(k8sClient))
	r.Delete("/{namespace}/{name}", handler.Delete)

	req := httptest.NewRequest(http.MethodDelete, "/default/tcp-route-to-delete", nil)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp["message"] != "tcproute deleted" {
		t.Errorf("expected 'tcproute deleted' message, got %s", resp["message"])
	}
}

func TestTCPRouteHandler_List_NoClusterContext(t *testing.T) {
	handler := &TCPRouteHandler{}

	r := chi.NewRouter()
	// No contextMiddleware — no cluster client in context.
	r.Get("/api/v1/tcproutes", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tcproutes", nil)
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
