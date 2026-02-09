package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

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
	handler := &GatewayHandler{KubeClient: k8sClient}

	t.Run("list all gateways", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/gateways", nil)
		w := httptest.NewRecorder()

		handler.List(w, req)

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
		req := httptest.NewRequest(http.MethodGet, "/api/v1/gateways?namespace=ns1", nil)
		w := httptest.NewRecorder()

		handler.List(w, req)

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
	handler := &GatewayHandler{KubeClient: k8sClient}

	t.Run("successful get", func(t *testing.T) {
		r := chi.NewRouter()
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
	handler := &GatewayHandler{KubeClient: k8sClient}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/gatewayclasses", nil)
	w := httptest.NewRecorder()

	handler.ListClasses(w, req)

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
	handler := &GatewayHandler{KubeClient: k8sClient}

	t.Run("successful get", func(t *testing.T) {
		r := chi.NewRouter()
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
