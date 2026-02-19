package handlers

import (
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
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
	"github.com/kubenetlabs/ngc/api/internal/kubernetes"
)

func newFakeCoexistenceDynamicClient(objects ...runtime.Object) *fakedynamic.FakeDynamicClient {
	s := runtime.NewScheme()
	// Ingress (networking.k8s.io/v1)
	s.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "networking.k8s.io", Version: "v1", Kind: "Ingress"},
		&unstructured.Unstructured{},
	)
	s.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "networking.k8s.io", Version: "v1", Kind: "IngressList"},
		&unstructured.UnstructuredList{},
	)
	// VirtualServer (k8s.nginx.org/v1)
	s.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "k8s.nginx.org", Version: "v1", Kind: "VirtualServer"},
		&unstructured.Unstructured{},
	)
	s.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "k8s.nginx.org", Version: "v1", Kind: "VirtualServerList"},
		&unstructured.UnstructuredList{},
	)
	// VirtualServerRoute (k8s.nginx.org/v1)
	s.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "k8s.nginx.org", Version: "v1", Kind: "VirtualServerRoute"},
		&unstructured.Unstructured{},
	)
	s.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "k8s.nginx.org", Version: "v1", Kind: "VirtualServerRouteList"},
		&unstructured.UnstructuredList{},
	)
	// TransportServer (k8s.nginx.org/v1)
	s.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "k8s.nginx.org", Version: "v1", Kind: "TransportServer"},
		&unstructured.Unstructured{},
	)
	s.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "k8s.nginx.org", Version: "v1", Kind: "TransportServerList"},
		&unstructured.UnstructuredList{},
	)
	return fakedynamic.NewSimpleDynamicClient(s, objects...)
}

func newTestIngress(name, namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": "networking.k8s.io/v1",
			"kind":       "Ingress",
			"metadata": map[string]any{
				"name":      name,
				"namespace": namespace,
				"annotations": map[string]any{
					"kubernetes.io/ingress.class": "nginx",
				},
			},
			"spec": map[string]any{
				"rules": []any{
					map[string]any{
						"host": "example.com",
						"http": map[string]any{
							"paths": []any{
								map[string]any{
									"path": "/",
									"backend": map[string]any{
										"service": map[string]any{
											"name": "web-svc",
											"port": map[string]any{"number": int64(80)},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func coexistenceContextMiddleware(dc *fakedynamic.FakeDynamicClient, gatewayObjs ...runtime.Object) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scheme := setupScheme(nil)
			if scheme == nil {
				scheme = runtime.NewScheme()
			}
			builder := fake.NewClientBuilder().WithScheme(scheme)
			if len(gatewayObjs) > 0 {
				objs := make([]runtime.Object, len(gatewayObjs))
				copy(objs, gatewayObjs)
				for _, o := range objs {
					if co, ok := o.(metav1.Object); ok {
						_ = co
					}
				}
				// Convert to client.Object for WithObjects
				clientObjs := make([]interface{ GetName() string }, 0)
				_ = clientObjs
				for _, o := range gatewayObjs {
					builder = builder.WithRuntimeObjects(o)
				}
			}
			fakeClient := builder.Build()
			k8s := kubernetes.NewForTestWithDynamic(fakeClient, dc)
			ctx := cluster.WithClient(r.Context(), k8s)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func TestCoexistenceHandler_Overview_EmptyCluster(t *testing.T) {
	dc := newFakeCoexistenceDynamicClient()
	scheme := setupScheme(t)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	k8s := kubernetes.NewForTestWithDynamic(fakeClient, dc)

	handler := &CoexistenceHandler{}
	r := chi.NewRouter()
	r.Use(contextMiddleware(k8s))
	r.Get("/coexistence/overview", handler.Overview)

	req := httptest.NewRequest(http.MethodGet, "/coexistence/overview", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp CoexistenceOverview
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.KIC.Installed {
		t.Error("expected kic.installed=false in empty cluster")
	}
	if resp.NGF.Installed {
		t.Error("expected ngf.installed=false in empty cluster")
	}
}

func TestCoexistenceHandler_Overview_KICOnly(t *testing.T) {
	ing1 := newTestIngress("web-ingress", "production")
	dc := newFakeCoexistenceDynamicClient(ing1)
	scheme := setupScheme(t)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	k8s := kubernetes.NewForTestWithDynamic(fakeClient, dc)

	handler := &CoexistenceHandler{}
	r := chi.NewRouter()
	r.Use(contextMiddleware(k8s))
	r.Get("/coexistence/overview", handler.Overview)

	req := httptest.NewRequest(http.MethodGet, "/coexistence/overview", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp CoexistenceOverview
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.KIC.Installed {
		t.Error("expected kic.installed=true with Ingress present")
	}
	if resp.KIC.ResourceCount != 1 {
		t.Errorf("expected kic.resourceCount=1, got %d", resp.KIC.ResourceCount)
	}
}

func TestCoexistenceHandler_MigrationReadiness_NoCRDs(t *testing.T) {
	// Only KIC resources, no Gateway API
	ing1 := newTestIngress("web-ingress", "production")
	dc := newFakeCoexistenceDynamicClient(ing1)
	scheme := setupScheme(t)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	k8s := kubernetes.NewForTestWithDynamic(fakeClient, dc)

	handler := &CoexistenceHandler{}
	r := chi.NewRouter()
	r.Use(contextMiddleware(k8s))
	r.Get("/coexistence/readiness", handler.MigrationReadiness)

	req := httptest.NewRequest(http.MethodGet, "/coexistence/readiness", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp MigrationReadinessResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Without Gateway API CRDs, score should be partial at best (not "ready")
	if resp.Status == "ready" {
		t.Errorf("expected status != 'ready' when no CRDs, got %s (score: %f)", resp.Status, resp.Score)
	}
	if resp.Score >= 75 {
		t.Errorf("expected score < 75 when no CRDs, got %f", resp.Score)
	}
}

func TestCoexistenceHandler_MigrationReadiness_FullyReady(t *testing.T) {
	// KIC ingress + Gateway API resources present
	ing1 := newTestIngress("web-ingress", "production")
	dc := newFakeCoexistenceDynamicClient(ing1)
	scheme := setupScheme(t)

	gc := &gatewayv1.GatewayClass{
		ObjectMeta: metav1.ObjectMeta{Name: "nginx"},
		Spec: gatewayv1.GatewayClassSpec{
			ControllerName: "gateway.nginx.org/nginx-gateway-controller",
		},
	}
	gw := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Name: "test-gw", Namespace: "default"},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: "nginx",
			Listeners: []gatewayv1.Listener{
				{Name: "http", Port: 80, Protocol: gatewayv1.HTTPProtocolType},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gc, gw).
		Build()
	k8s := kubernetes.NewForTestWithDynamic(fakeClient, dc)

	handler := &CoexistenceHandler{}
	r := chi.NewRouter()
	r.Use(contextMiddleware(k8s))
	r.Get("/coexistence/readiness", handler.MigrationReadiness)

	req := httptest.NewRequest(http.MethodGet, "/coexistence/readiness", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp MigrationReadinessResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Score < 75 {
		t.Errorf("expected score >= 75 when fully ready, got %f", resp.Score)
	}
}

func TestCoexistenceHandler_Overview_NoClusterContext(t *testing.T) {
	handler := &CoexistenceHandler{}

	r := chi.NewRouter()
	r.Get("/coexistence/overview", handler.Overview)

	req := httptest.NewRequest(http.MethodGet, "/coexistence/overview", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", w.Code, w.Body.String())
	}
}
