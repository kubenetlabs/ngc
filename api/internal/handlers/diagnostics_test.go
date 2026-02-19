package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kubenetlabs/ngc/api/internal/kubernetes"
)

func TestDiagnosticsHandler_RouteCheck_NoClusterContext(t *testing.T) {
	handler := &DiagnosticsHandler{}

	r := chi.NewRouter()
	r.Post("/api/v1/diagnostics/route-check", handler.RouteCheck)

	body := `{"namespace":"default","routeName":"my-route"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/diagnostics/route-check", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}
}

func TestDiagnosticsHandler_RouteCheck_BadJSON(t *testing.T) {
	scheme := setupScheme(t)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	k8sClient := kubernetes.NewForTest(fakeClient)
	handler := &DiagnosticsHandler{}

	r := chi.NewRouter()
	r.Use(contextMiddleware(k8sClient))
	r.Post("/api/v1/diagnostics/route-check", handler.RouteCheck)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/diagnostics/route-check", bytes.NewBufferString("not json"))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestDiagnosticsHandler_RouteCheck_MissingFields(t *testing.T) {
	scheme := setupScheme(t)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	k8sClient := kubernetes.NewForTest(fakeClient)
	handler := &DiagnosticsHandler{}

	r := chi.NewRouter()
	r.Use(contextMiddleware(k8sClient))
	r.Post("/api/v1/diagnostics/route-check", handler.RouteCheck)

	body := `{"namespace":"","routeName":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/diagnostics/route-check", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestDiagnosticsHandler_RouteCheck_UnsupportedRouteKind(t *testing.T) {
	scheme := setupScheme(t)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	k8sClient := kubernetes.NewForTest(fakeClient)
	handler := &DiagnosticsHandler{}

	r := chi.NewRouter()
	r.Use(contextMiddleware(k8sClient))
	r.Post("/api/v1/diagnostics/route-check", handler.RouteCheck)

	body := `{"namespace":"default","routeName":"my-route","routeKind":"GRPCRoute"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/diagnostics/route-check", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp RouteCheckResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "unhealthy" {
		t.Errorf("expected status 'unhealthy', got %q", resp.Status)
	}

	if len(resp.Checks) != 1 {
		t.Fatalf("expected 1 check, got %d", len(resp.Checks))
	}
	if resp.Checks[0].Status != "skip" {
		t.Errorf("expected check status 'skip', got %q", resp.Checks[0].Status)
	}
}

func TestDiagnosticsHandler_RouteCheck_RouteNotFound(t *testing.T) {
	scheme := setupScheme(t)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	k8sClient := kubernetes.NewForTest(fakeClient)
	handler := &DiagnosticsHandler{}

	r := chi.NewRouter()
	r.Use(contextMiddleware(k8sClient))
	r.Post("/api/v1/diagnostics/route-check", handler.RouteCheck)

	body := `{"namespace":"default","routeName":"nonexistent"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/diagnostics/route-check", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp RouteCheckResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "unhealthy" {
		t.Errorf("expected status 'unhealthy', got %q", resp.Status)
	}

	if len(resp.Checks) != 6 {
		t.Fatalf("expected 6 checks, got %d", len(resp.Checks))
	}

	// First check: fail (route not found)
	if resp.Checks[0].Name != "Route Exists" || resp.Checks[0].Status != "fail" {
		t.Errorf("expected first check 'Route Exists'='fail', got %q=%q", resp.Checks[0].Name, resp.Checks[0].Status)
	}

	// Remaining checks: skip
	for i := 1; i < 6; i++ {
		if resp.Checks[i].Status != "skip" {
			t.Errorf("expected check %d status 'skip', got %q", i, resp.Checks[i].Status)
		}
	}
}

func TestDiagnosticsHandler_RouteCheck_HealthyRoute(t *testing.T) {
	scheme := setupScheme(t)

	gwName := gatewayv1.ObjectName("my-gateway")
	pathPrefix := gatewayv1.PathMatchPathPrefix
	pathValue := "/"
	svcPort := gatewayv1.PortNumber(80)

	route := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-route",
			Namespace: "default",
		},
		Spec: gatewayv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{
					{Name: gwName},
				},
			},
			Rules: []gatewayv1.HTTPRouteRule{
				{
					Matches: []gatewayv1.HTTPRouteMatch{
						{
							Path: &gatewayv1.HTTPPathMatch{
								Type:  &pathPrefix,
								Value: &pathValue,
							},
						},
					},
					BackendRefs: []gatewayv1.HTTPBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name: "my-svc",
									Port: &svcPort,
								},
							},
						},
					},
				},
			},
		},
		Status: gatewayv1.HTTPRouteStatus{
			RouteStatus: gatewayv1.RouteStatus{
				Parents: []gatewayv1.RouteParentStatus{
					{
						Conditions: []metav1.Condition{
							{Type: "Accepted", Status: metav1.ConditionTrue},
							{Type: "ResolvedRefs", Status: metav1.ConditionTrue},
						},
					},
				},
			},
		},
	}

	gw := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-gateway",
			Namespace: "default",
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

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-svc",
			Namespace: "default",
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(route, gw, svc).
		WithStatusSubresource(route).
		Build()
	k8sClient := kubernetes.NewForTest(fakeClient)
	handler := &DiagnosticsHandler{}

	r := chi.NewRouter()
	r.Use(contextMiddleware(k8sClient))
	r.Post("/api/v1/diagnostics/route-check", handler.RouteCheck)

	body := `{"namespace":"default","routeName":"my-route"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/diagnostics/route-check", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var resp RouteCheckResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "healthy" {
		t.Errorf("expected status 'healthy', got %q", resp.Status)
	}

	if len(resp.Checks) != 6 {
		t.Fatalf("expected 6 checks, got %d", len(resp.Checks))
	}

	expectedChecks := []struct {
		name   string
		status string
	}{
		{"Route Exists", "pass"},
		{"Parent Gateway Attached", "pass"},
		{"Listener Match", "pass"},
		{"Backend Health", "pass"},
		{"Route Accepted", "pass"},
		{"Route Resolved", "pass"},
	}

	for i, expected := range expectedChecks {
		if resp.Checks[i].Name != expected.name {
			t.Errorf("check %d: expected name %q, got %q", i, expected.name, resp.Checks[i].Name)
		}
		if resp.Checks[i].Status != expected.status {
			t.Errorf("check %d (%s): expected status %q, got %q", i, expected.name, expected.status, resp.Checks[i].Status)
		}
	}
}

func TestDiagnosticsHandler_RouteCheck_MissingBackend(t *testing.T) {
	scheme := setupScheme(t)

	gwName := gatewayv1.ObjectName("my-gateway")
	svcPort := gatewayv1.PortNumber(80)

	route := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-route",
			Namespace: "default",
		},
		Spec: gatewayv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{
					{Name: gwName},
				},
			},
			Rules: []gatewayv1.HTTPRouteRule{
				{
					BackendRefs: []gatewayv1.HTTPBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name: "missing-svc",
									Port: &svcPort,
								},
							},
						},
					},
				},
			},
		},
		Status: gatewayv1.HTTPRouteStatus{
			RouteStatus: gatewayv1.RouteStatus{
				Parents: []gatewayv1.RouteParentStatus{
					{
						Conditions: []metav1.Condition{
							{Type: "Accepted", Status: metav1.ConditionTrue},
							{Type: "ResolvedRefs", Status: metav1.ConditionFalse},
						},
					},
				},
			},
		},
	}

	gw := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-gateway",
			Namespace: "default",
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

	// No services exist — backend check should fail
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(route, gw).
		WithStatusSubresource(route).
		Build()
	k8sClient := kubernetes.NewForTest(fakeClient)
	handler := &DiagnosticsHandler{}

	r := chi.NewRouter()
	r.Use(contextMiddleware(k8sClient))
	r.Post("/api/v1/diagnostics/route-check", handler.RouteCheck)

	body := `{"namespace":"default","routeName":"my-route"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/diagnostics/route-check", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var resp RouteCheckResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "unhealthy" {
		t.Errorf("expected status 'unhealthy', got %q", resp.Status)
	}

	// Backend Health should be "fail", Route Resolved should be "fail"
	backendCheck := findCheck(resp.Checks, "Backend Health")
	if backendCheck == nil {
		t.Fatal("Backend Health check not found")
	}
	if backendCheck.Status != "fail" {
		t.Errorf("expected Backend Health status 'fail', got %q", backendCheck.Status)
	}

	resolvedCheck := findCheck(resp.Checks, "Route Resolved")
	if resolvedCheck == nil {
		t.Fatal("Route Resolved check not found")
	}
	if resolvedCheck.Status != "fail" {
		t.Errorf("expected Route Resolved status 'fail', got %q", resolvedCheck.Status)
	}
}

func TestDiagnosticsHandler_Trace_NoClusterContext(t *testing.T) {
	handler := &DiagnosticsHandler{}

	r := chi.NewRouter()
	r.Post("/api/v1/diagnostics/trace", handler.Trace)

	body := `{"host":"example.com","path":"/"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/diagnostics/trace", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}
}

func TestDiagnosticsHandler_Trace_MissingFields(t *testing.T) {
	scheme := setupScheme(t)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	k8sClient := kubernetes.NewForTest(fakeClient)
	handler := &DiagnosticsHandler{}

	r := chi.NewRouter()
	r.Use(contextMiddleware(k8sClient))
	r.Post("/api/v1/diagnostics/trace", handler.Trace)

	body := `{"host":"","path":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/diagnostics/trace", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestDiagnosticsHandler_Trace_NoMatchingGateway(t *testing.T) {
	scheme := setupScheme(t)

	// Gateway with hostname that doesn't match the trace request
	hostname := gatewayv1.Hostname("other.example.com")
	gw := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-gateway",
			Namespace: "default",
		},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: "nginx",
			Listeners: []gatewayv1.Listener{
				{
					Name:     "http",
					Port:     80,
					Protocol: gatewayv1.HTTPProtocolType,
					Hostname: &hostname,
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(gw).Build()
	k8sClient := kubernetes.NewForTest(fakeClient)
	handler := &DiagnosticsHandler{}

	r := chi.NewRouter()
	r.Use(contextMiddleware(k8sClient))
	r.Post("/api/v1/diagnostics/trace", handler.Trace)

	body := `{"host":"app.example.com","path":"/"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/diagnostics/trace", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var resp TraceResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Matched {
		t.Error("expected matched=false")
	}

	if len(resp.Steps) != 4 {
		t.Fatalf("expected 4 steps, got %d", len(resp.Steps))
	}

	if resp.Steps[0].Status != "fail" {
		t.Errorf("expected step 1 status 'fail', got %q", resp.Steps[0].Status)
	}
	for i := 1; i < 4; i++ {
		if resp.Steps[i].Status != "skip" {
			t.Errorf("expected step %d status 'skip', got %q", i+1, resp.Steps[i].Status)
		}
	}
}

func TestDiagnosticsHandler_Trace_FullMatch(t *testing.T) {
	scheme := setupScheme(t)

	gwName := gatewayv1.ObjectName("my-gateway")
	pathPrefix := gatewayv1.PathMatchPathPrefix
	pathValue := "/"
	svcPort := gatewayv1.PortNumber(80)

	gw := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-gateway",
			Namespace: "default",
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

	route := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-route",
			Namespace: "default",
		},
		Spec: gatewayv1.HTTPRouteSpec{
			CommonRouteSpec: gatewayv1.CommonRouteSpec{
				ParentRefs: []gatewayv1.ParentReference{
					{Name: gwName},
				},
			},
			Hostnames: []gatewayv1.Hostname{"app.example.com"},
			Rules: []gatewayv1.HTTPRouteRule{
				{
					Matches: []gatewayv1.HTTPRouteMatch{
						{
							Path: &gatewayv1.HTTPPathMatch{
								Type:  &pathPrefix,
								Value: &pathValue,
							},
						},
					},
					BackendRefs: []gatewayv1.HTTPBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name: "my-svc",
									Port: &svcPort,
								},
							},
						},
					},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(gw, route).Build()
	k8sClient := kubernetes.NewForTest(fakeClient)
	handler := &DiagnosticsHandler{}

	r := chi.NewRouter()
	r.Use(contextMiddleware(k8sClient))
	r.Post("/api/v1/diagnostics/trace", handler.Trace)

	body := `{"host":"app.example.com","path":"/api/test","method":"GET"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/diagnostics/trace", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d; body: %s", w.Code, w.Body.String())
	}

	var resp TraceResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Matched {
		t.Error("expected matched=true")
	}

	if resp.MatchedRoute == nil || *resp.MatchedRoute != "default/my-route" {
		t.Errorf("expected matchedRoute 'default/my-route', got %v", resp.MatchedRoute)
	}

	if len(resp.Steps) != 4 {
		t.Fatalf("expected 4 steps, got %d", len(resp.Steps))
	}

	for i, step := range resp.Steps {
		if step.Status != "pass" {
			t.Errorf("step %d (%s): expected status 'pass', got %q", i, step.Name, step.Status)
		}
	}
}

// findCheck looks up a diagnostic check by name.
func findCheck(checks []DiagnosticCheck, name string) *DiagnosticCheck {
	for i := range checks {
		if checks[i].Name == name {
			return &checks[i]
		}
	}
	return nil
}
