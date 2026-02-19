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

func newRouteForSimulation(t *testing.T) *kubernetes.Client {
	t.Helper()
	scheme := setupScheme(t)

	pathPrefix := gatewayv1.PathMatchPathPrefix
	pathExact := gatewayv1.PathMatchExact
	pathRegex := gatewayv1.PathMatchRegularExpression
	methodGet := gatewayv1.HTTPMethodGet
	headerExact := gatewayv1.HeaderMatchExact

	ns := gatewayv1.Namespace("default")
	port := gatewayv1.PortNumber(80)
	weight := int32(100)

	route := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-route",
			Namespace: "default",
		},
		Spec: gatewayv1.HTTPRouteSpec{
			Rules: []gatewayv1.HTTPRouteRule{
				// Rule 0: exact path match /health
				{
					Matches: []gatewayv1.HTTPRouteMatch{
						{
							Path: &gatewayv1.HTTPPathMatch{
								Type:  &pathExact,
								Value: strPtr("/health"),
							},
						},
					},
					BackendRefs: []gatewayv1.HTTPBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name:      "health-svc",
									Namespace: &ns,
									Port:      &port,
								},
								Weight: &weight,
							},
						},
					},
				},
				// Rule 1: prefix match /api + method GET + header
				{
					Matches: []gatewayv1.HTTPRouteMatch{
						{
							Path: &gatewayv1.HTTPPathMatch{
								Type:  &pathPrefix,
								Value: strPtr("/api"),
							},
							Method: &methodGet,
							Headers: []gatewayv1.HTTPHeaderMatch{
								{
									Type:  &headerExact,
									Name:  "X-Version",
									Value: "v2",
								},
							},
						},
					},
					BackendRefs: []gatewayv1.HTTPBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name: "api-v2-svc",
								},
							},
						},
					},
				},
				// Rule 2: regex path match
				{
					Matches: []gatewayv1.HTTPRouteMatch{
						{
							Path: &gatewayv1.HTTPPathMatch{
								Type:  &pathRegex,
								Value: strPtr(`^/users/\d+$`),
							},
						},
					},
					BackendRefs: []gatewayv1.HTTPBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name: "users-svc",
								},
							},
						},
					},
				},
				// Rule 3: catch-all (no matches = matches everything)
				{
					BackendRefs: []gatewayv1.HTTPBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name: "default-svc",
								},
							},
						},
					},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(route).Build()
	return kubernetes.NewForTest(fakeClient)
}

func strPtr(s string) *string {
	return &s
}

func TestRouteHandler_Simulate_ExactPathMatch(t *testing.T) {
	k8s := newRouteForSimulation(t)
	handler := &RouteHandler{}

	r := chi.NewRouter()
	r.Use(contextMiddleware(k8s))
	r.Post("/{namespace}/{name}/simulate", handler.Simulate)

	body := `{"method":"GET","path":"/health"}`
	req := httptest.NewRequest(http.MethodPost, "/default/test-route/simulate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp SimulateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Matched {
		t.Error("expected matched=true for exact path /health")
	}
	if resp.MatchedRule != 0 {
		t.Errorf("expected matchedRule=0, got %d", resp.MatchedRule)
	}
	if len(resp.Backends) == 0 {
		t.Fatal("expected at least one backend")
	}
	if resp.Backends[0].Name != "health-svc" {
		t.Errorf("expected backend health-svc, got %s", resp.Backends[0].Name)
	}
	// Verify backend fields are populated
	if resp.Backends[0].Weight == nil {
		t.Error("expected weight to be set")
	}
	if resp.Backends[0].Port == nil {
		t.Error("expected port to be set")
	}
	if resp.Backends[0].Namespace == nil {
		t.Error("expected namespace to be set")
	}
}

func TestRouteHandler_Simulate_PrefixPathMatch(t *testing.T) {
	k8s := newRouteForSimulation(t)
	handler := &RouteHandler{}

	r := chi.NewRouter()
	r.Use(contextMiddleware(k8s))
	r.Post("/{namespace}/{name}/simulate", handler.Simulate)

	body := `{"method":"GET","path":"/api/users","headers":{"X-Version":"v2"}}`
	req := httptest.NewRequest(http.MethodPost, "/default/test-route/simulate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp SimulateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Matched {
		t.Error("expected matched=true for prefix /api with method GET and header X-Version:v2")
	}
	// Rule 0 (/health exact) doesn't match /api, but rule 1 does
	if resp.MatchedRule != 1 {
		t.Errorf("expected matchedRule=1, got %d", resp.MatchedRule)
	}
}

func TestRouteHandler_Simulate_PrefixNoMatch(t *testing.T) {
	k8s := newRouteForSimulation(t)
	handler := &RouteHandler{}

	r := chi.NewRouter()
	r.Use(contextMiddleware(k8s))
	r.Post("/{namespace}/{name}/simulate", handler.Simulate)

	// Path matches /api prefix but wrong method header value
	body := `{"method":"GET","path":"/api/users","headers":{"X-Version":"v1"}}`
	req := httptest.NewRequest(http.MethodPost, "/default/test-route/simulate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp SimulateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Rule 1 doesn't match (wrong header value), but rule 3 (catch-all) matches
	if !resp.Matched {
		t.Error("expected matched=true (catch-all rule should match)")
	}
	if resp.MatchedRule != 3 {
		t.Errorf("expected matchedRule=3 (catch-all), got %d", resp.MatchedRule)
	}
}

func TestRouteHandler_Simulate_RegexPathMatch(t *testing.T) {
	k8s := newRouteForSimulation(t)
	handler := &RouteHandler{}

	r := chi.NewRouter()
	r.Use(contextMiddleware(k8s))
	r.Post("/{namespace}/{name}/simulate", handler.Simulate)

	body := `{"method":"GET","path":"/users/42"}`
	req := httptest.NewRequest(http.MethodPost, "/default/test-route/simulate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp SimulateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Matched {
		t.Error("expected matched=true for regex /users/42")
	}
	if resp.MatchedRule != 2 {
		t.Errorf("expected matchedRule=2 (regex), got %d", resp.MatchedRule)
	}
}

func TestRouteHandler_Simulate_MethodMismatch(t *testing.T) {
	k8s := newRouteForSimulation(t)
	handler := &RouteHandler{}

	r := chi.NewRouter()
	r.Use(contextMiddleware(k8s))
	r.Post("/{namespace}/{name}/simulate", handler.Simulate)

	// Rule 1 requires GET, sending POST
	body := `{"method":"POST","path":"/api/users","headers":{"X-Version":"v2"}}`
	req := httptest.NewRequest(http.MethodPost, "/default/test-route/simulate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp SimulateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Rule 1 won't match due to method, but rule 3 catch-all still matches
	if !resp.Matched {
		t.Error("expected matched=true (catch-all)")
	}
	if resp.MatchedRule != 3 {
		t.Errorf("expected matchedRule=3 (catch-all), got %d", resp.MatchedRule)
	}

	// Verify match details contain reasons for earlier rule failures
	if len(resp.MatchDetails) < 2 {
		t.Fatal("expected at least 2 match details")
	}
	if resp.MatchDetails[1].Matched {
		t.Error("expected rule 1 to not match due to method mismatch")
	}
}

func TestRouteHandler_Simulate_MissingHeader(t *testing.T) {
	k8s := newRouteForSimulation(t)
	handler := &RouteHandler{}

	r := chi.NewRouter()
	r.Use(contextMiddleware(k8s))
	r.Post("/{namespace}/{name}/simulate", handler.Simulate)

	// Rule 1 requires X-Version header, not provided
	body := `{"method":"GET","path":"/api/users"}`
	req := httptest.NewRequest(http.MethodPost, "/default/test-route/simulate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp SimulateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Rule 1 won't match (missing header), catch-all rule 3 matches
	if resp.MatchedRule != 3 {
		t.Errorf("expected matchedRule=3 (catch-all), got %d", resp.MatchedRule)
	}
}

func TestRouteHandler_Simulate_CatchAllRule(t *testing.T) {
	k8s := newRouteForSimulation(t)
	handler := &RouteHandler{}

	r := chi.NewRouter()
	r.Use(contextMiddleware(k8s))
	r.Post("/{namespace}/{name}/simulate", handler.Simulate)

	// A completely random path should only match the catch-all rule 3
	body := `{"method":"DELETE","path":"/some/random/path"}`
	req := httptest.NewRequest(http.MethodPost, "/default/test-route/simulate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp SimulateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.Matched {
		t.Error("expected matched=true for catch-all")
	}
	if resp.MatchedRule != 3 {
		t.Errorf("expected matchedRule=3, got %d", resp.MatchedRule)
	}
	if len(resp.Backends) == 0 {
		t.Fatal("expected at least one backend")
	}
	if resp.Backends[0].Name != "default-svc" {
		t.Errorf("expected backend default-svc, got %s", resp.Backends[0].Name)
	}
}

func TestRouteHandler_Simulate_RouteNotFound(t *testing.T) {
	scheme := setupScheme(t)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	k8s := kubernetes.NewForTest(fakeClient)
	handler := &RouteHandler{}

	r := chi.NewRouter()
	r.Use(contextMiddleware(k8s))
	r.Post("/{namespace}/{name}/simulate", handler.Simulate)

	body := `{"method":"GET","path":"/"}`
	req := httptest.NewRequest(http.MethodPost, "/default/nonexistent/simulate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRouteHandler_Simulate_NoClusterContext(t *testing.T) {
	handler := &RouteHandler{}

	r := chi.NewRouter()
	r.Post("/{namespace}/{name}/simulate", handler.Simulate)

	body := `{"method":"GET","path":"/"}`
	req := httptest.NewRequest(http.MethodPost, "/default/test-route/simulate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRouteHandler_Simulate_MatchDetails(t *testing.T) {
	k8s := newRouteForSimulation(t)
	handler := &RouteHandler{}

	r := chi.NewRouter()
	r.Use(contextMiddleware(k8s))
	r.Post("/{namespace}/{name}/simulate", handler.Simulate)

	body := `{"method":"GET","path":"/health"}`
	req := httptest.NewRequest(http.MethodPost, "/default/test-route/simulate", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp SimulateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Should have 4 match details (one per rule)
	if len(resp.MatchDetails) != 4 {
		t.Fatalf("expected 4 match details, got %d", len(resp.MatchDetails))
	}

	// Rule 0: matched (exact /health)
	if !resp.MatchDetails[0].Matched {
		t.Error("expected rule 0 to match")
	}

	// Rule 1: not matched (/health doesn't have prefix /api)
	// Actually /health doesn't start with /api, and also missing X-Version header
	if resp.MatchDetails[1].Matched {
		t.Error("expected rule 1 to not match")
	}

	// Rule 3: catch-all should also match
	if !resp.MatchDetails[3].Matched {
		t.Error("expected rule 3 (catch-all) to match")
	}
	if resp.MatchDetails[3].Reason == "" {
		t.Error("expected non-empty reason for catch-all match")
	}
}
