package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/kubenetlabs/ngc/api/internal/kubernetes"
)

func TestTopologyHandler_Full_EmptyCluster(t *testing.T) {
	scheme := setupScheme(t)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	k8s := kubernetes.NewForTest(fakeClient)

	handler := &TopologyHandler{}
	r := chi.NewRouter()
	r.Use(contextMiddleware(k8s))
	r.Get("/topology", handler.Full)

	req := httptest.NewRequest(http.MethodGet, "/topology", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp TopologyResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(resp.Nodes))
	}
	if len(resp.Edges) != 0 {
		t.Errorf("expected 0 edges, got %d", len(resp.Edges))
	}
}

func TestTopologyHandler_Full_WithResources(t *testing.T) {
	scheme := setupScheme(t)

	ns := gatewayv1.Namespace("default")
	port := gatewayv1.PortNumber(80)

	gw := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Name: "test-gw", Namespace: "default"},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: "nginx",
			Listeners: []gatewayv1.Listener{
				{Name: "http", Port: 80, Protocol: gatewayv1.HTTPProtocolType},
			},
		},
	}

	hr := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{Name: "test-route", Namespace: "default"},
		Spec: gatewayv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{
					{Name: "test-gw", Namespace: &ns},
				},
			},
			Rules: []gatewayv1.HTTPRouteRule{
				{
					BackendRefs: []gatewayv1.HTTPBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name:      "test-svc",
									Namespace: &ns,
									Port:      &port,
								},
							},
						},
					},
				},
			},
		},
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "test-svc", Namespace: "default"},
		Spec: corev1.ServiceSpec{
			ClusterIP: "10.0.0.1",
			Type:      corev1.ServiceTypeClusterIP,
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gw, hr, svc).
		Build()
	k8s := kubernetes.NewForTest(fakeClient)

	handler := &TopologyHandler{}
	r := chi.NewRouter()
	r.Use(contextMiddleware(k8s))
	r.Get("/topology", handler.Full)

	req := httptest.NewRequest(http.MethodGet, "/topology", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp TopologyResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should have: 1 gateway + 1 httproute + 1 service = 3 nodes
	if len(resp.Nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(resp.Nodes))
	}

	// Should have: route->gateway (parentRef) + route->service (backendRef) = 2 edges
	if len(resp.Edges) != 2 {
		t.Errorf("expected 2 edges, got %d", len(resp.Edges))
	}

	// Verify node types exist
	nodeTypes := make(map[string]bool)
	for _, n := range resp.Nodes {
		nodeTypes[n.Type] = true
	}
	for _, expected := range []string{"gateway", "httproute", "service"} {
		if !nodeTypes[expected] {
			t.Errorf("expected node type %s, not found", expected)
		}
	}
}

func TestTopologyHandler_Full_DanglingBackendRef(t *testing.T) {
	scheme := setupScheme(t)

	ns := gatewayv1.Namespace("default")
	port := gatewayv1.PortNumber(80)

	gw := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Name: "test-gw", Namespace: "default"},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: "nginx",
			Listeners: []gatewayv1.Listener{
				{Name: "http", Port: 80, Protocol: gatewayv1.HTTPProtocolType},
			},
		},
	}

	// Route references a service that doesn't exist
	hr := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{Name: "test-route", Namespace: "default"},
		Spec: gatewayv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{
					{Name: "test-gw", Namespace: &ns},
				},
			},
			Rules: []gatewayv1.HTTPRouteRule{
				{
					BackendRefs: []gatewayv1.HTTPBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name:      "missing-svc",
									Namespace: &ns,
									Port:      &port,
								},
							},
						},
					},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gw, hr).
		Build()
	k8s := kubernetes.NewForTest(fakeClient)

	handler := &TopologyHandler{}
	r := chi.NewRouter()
	r.Use(contextMiddleware(k8s))
	r.Get("/topology", handler.Full)

	req := httptest.NewRequest(http.MethodGet, "/topology", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp TopologyResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should still have a placeholder node for the missing service
	foundPlaceholder := false
	for _, n := range resp.Nodes {
		if n.Type == "service" && n.Name == "missing-svc" && n.Status == "error" {
			foundPlaceholder = true
		}
	}
	if !foundPlaceholder {
		t.Error("expected placeholder node for missing service with status 'error'")
	}
}

func TestTopologyHandler_Full_NoClusterContext(t *testing.T) {
	handler := &TopologyHandler{}

	r := chi.NewRouter()
	r.Get("/topology", handler.Full)

	req := httptest.NewRequest(http.MethodGet, "/topology", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTopologyHandler_ByGateway(t *testing.T) {
	scheme := setupScheme(t)

	ns := gatewayv1.Namespace("default")
	port := gatewayv1.PortNumber(80)

	gw := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Name: "my-gateway", Namespace: "default"},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: "nginx",
			Listeners: []gatewayv1.Listener{
				{Name: "http", Port: 80, Protocol: gatewayv1.HTTPProtocolType},
			},
		},
	}

	hr := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{Name: "my-route", Namespace: "default"},
		Spec: gatewayv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{
					{Name: "my-gateway", Namespace: &ns},
				},
			},
			Rules: []gatewayv1.HTTPRouteRule{
				{
					BackendRefs: []gatewayv1.HTTPBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name:      "my-svc",
									Namespace: &ns,
									Port:      &port,
								},
							},
						},
					},
				},
			},
		},
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "my-svc", Namespace: "default"},
		Spec: corev1.ServiceSpec{
			ClusterIP: "10.0.0.1",
			Type:      corev1.ServiceTypeClusterIP,
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gw, hr, svc).
		Build()
	k8s := kubernetes.NewForTest(fakeClient)

	handler := &TopologyHandler{}
	r := chi.NewRouter()
	r.Use(contextMiddleware(k8s))
	r.Get("/topology/by-gateway/{name}", handler.ByGateway)

	req := httptest.NewRequest(http.MethodGet, "/topology/by-gateway/my-gateway?namespace=default", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp TopologyResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should have: 1 gateway + 1 httproute + 1 service = 3 nodes
	if len(resp.Nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(resp.Nodes))
	}

	// Should have: route->gateway (parentRef) + route->service (backendRef) = 2 edges
	if len(resp.Edges) != 2 {
		t.Errorf("expected 2 edges, got %d", len(resp.Edges))
	}

	// Verify we got the correct gateway
	foundGateway := false
	for _, n := range resp.Nodes {
		if n.Type == "gateway" && n.Name == "my-gateway" {
			foundGateway = true
		}
	}
	if !foundGateway {
		t.Error("expected to find gateway node 'my-gateway'")
	}
}

func TestTopologyHandler_ByGateway_NotFound(t *testing.T) {
	scheme := setupScheme(t)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	k8s := kubernetes.NewForTest(fakeClient)

	handler := &TopologyHandler{}
	r := chi.NewRouter()
	r.Use(contextMiddleware(k8s))
	r.Get("/topology/by-gateway/{name}", handler.ByGateway)

	req := httptest.NewRequest(http.MethodGet, "/topology/by-gateway/nonexistent?namespace=default", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestTopologyHandler_ByGateway_MultipleGateways(t *testing.T) {
	scheme := setupScheme(t)

	ns := gatewayv1.Namespace("default")
	port := gatewayv1.PortNumber(80)

	// Gateway A with its own route and service.
	gwA := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Name: "gw-a", Namespace: "default"},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: "nginx",
			Listeners: []gatewayv1.Listener{
				{Name: "http", Port: 80, Protocol: gatewayv1.HTTPProtocolType},
			},
		},
	}

	routeA := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{Name: "route-a", Namespace: "default"},
		Spec: gatewayv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{
					{Name: "gw-a", Namespace: &ns},
				},
			},
			Rules: []gatewayv1.HTTPRouteRule{
				{
					BackendRefs: []gatewayv1.HTTPBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name:      "svc-a",
									Namespace: &ns,
									Port:      &port,
								},
							},
						},
					},
				},
			},
		},
	}

	svcA := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "svc-a", Namespace: "default"},
		Spec: corev1.ServiceSpec{
			ClusterIP: "10.0.0.1",
			Type:      corev1.ServiceTypeClusterIP,
		},
	}

	// Gateway B with its own route and service.
	gwB := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{Name: "gw-b", Namespace: "default"},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: "nginx",
			Listeners: []gatewayv1.Listener{
				{Name: "http", Port: 80, Protocol: gatewayv1.HTTPProtocolType},
			},
		},
	}

	routeB := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{Name: "route-b", Namespace: "default"},
		Spec: gatewayv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{
					{Name: "gw-b", Namespace: &ns},
				},
			},
			Rules: []gatewayv1.HTTPRouteRule{
				{
					BackendRefs: []gatewayv1.HTTPBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name:      "svc-b",
									Namespace: &ns,
									Port:      &port,
								},
							},
						},
					},
				},
			},
		},
	}

	svcB := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "svc-b", Namespace: "default"},
		Spec: corev1.ServiceSpec{
			ClusterIP: "10.0.0.2",
			Type:      corev1.ServiceTypeClusterIP,
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gwA, gwB, routeA, routeB, svcA, svcB).
		Build()
	k8s := kubernetes.NewForTest(fakeClient)

	handler := &TopologyHandler{}
	r := chi.NewRouter()
	r.Use(contextMiddleware(k8s))
	r.Get("/topology/by-gateway/{name}", handler.ByGateway)

	// Request subgraph for gw-a only.
	req := httptest.NewRequest(http.MethodGet, "/topology/by-gateway/gw-a?namespace=default", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp TopologyResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should have: 1 gateway (gw-a) + 1 httproute (route-a) + 1 service (svc-a) = 3 nodes
	if len(resp.Nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(resp.Nodes))
		for _, n := range resp.Nodes {
			t.Logf("  node: %s (type=%s)", n.Name, n.Type)
		}
	}

	// Should have: route-a->gw-a (parentRef) + route-a->svc-a (backendRef) = 2 edges
	if len(resp.Edges) != 2 {
		t.Errorf("expected 2 edges, got %d", len(resp.Edges))
	}

	// Verify gw-b, route-b, svc-b are NOT in the result.
	for _, n := range resp.Nodes {
		if n.Name == "gw-b" || n.Name == "route-b" || n.Name == "svc-b" {
			t.Errorf("unexpected node %q in filtered subgraph", n.Name)
		}
	}

	// Verify gw-a, route-a, svc-a ARE in the result.
	expectedNames := map[string]bool{"gw-a": false, "route-a": false, "svc-a": false}
	for _, n := range resp.Nodes {
		if _, ok := expectedNames[n.Name]; ok {
			expectedNames[n.Name] = true
		}
	}
	for name, found := range expectedNames {
		if !found {
			t.Errorf("expected node %q in filtered subgraph, but not found", name)
		}
	}
}
