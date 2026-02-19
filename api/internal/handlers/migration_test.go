package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakedynamic "k8s.io/client-go/dynamic/fake"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
	"github.com/kubenetlabs/ngc/api/internal/kubernetes"
)

func TestMigrationHandler_Import_NginxConf(t *testing.T) {
	handler := &MigrationHandler{}

	r := chi.NewRouter()
	r.Post("/migration/import", handler.Import)

	body := ImportRequest{
		Source:  "file",
		Content: "server {\n  listen 80;\n  location / {\n    proxy_pass http://backend;\n  }\n  location /api {\n    proxy_pass http://api-backend;\n  }\n}",
		Format:  "nginx-conf",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/migration/import", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp ImportResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.ID == "" {
		t.Error("expected non-empty ID")
	}
	if resp.ResourceCount == 0 {
		t.Error("expected at least one resource")
	}

	// Should contain both Gateway and HTTPRoute resources
	hasGateway := false
	hasRoute := false
	for _, r := range resp.Resources {
		if r.Kind == "Gateway" {
			hasGateway = true
		}
		if r.Kind == "HTTPRoute" {
			hasRoute = true
		}
	}
	if !hasGateway {
		t.Error("expected Gateway resource from nginx conf")
	}
	if !hasRoute {
		t.Error("expected HTTPRoute resource from nginx conf")
	}
}

func TestMigrationHandler_Import_IngressYAML(t *testing.T) {
	handler := &MigrationHandler{}

	r := chi.NewRouter()
	r.Post("/migration/import", handler.Import)

	content := `apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-ingress
  namespace: production
spec:
  rules:
  - host: example.com`

	body := ImportRequest{
		Source:  "file",
		Content: content,
		Format:  "ingress-yaml",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/migration/import", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp ImportResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Resources) == 0 {
		t.Fatal("expected at least one resource")
	}
	found := false
	for _, r := range resp.Resources {
		if r.Kind == "Ingress" && r.Name == "my-ingress" && r.Namespace == "production" {
			found = true
		}
	}
	if !found {
		t.Error("expected Ingress resource with name 'my-ingress' in namespace 'production'")
	}
}

func TestMigrationHandler_Import_VirtualServerYAML(t *testing.T) {
	handler := &MigrationHandler{}

	r := chi.NewRouter()
	r.Post("/migration/import", handler.Import)

	content := `apiVersion: k8s.nginx.org/v1
kind: VirtualServer
metadata:
  name: vs-1
  namespace: default
---
apiVersion: k8s.nginx.org/v1
kind: VirtualServerRoute
metadata:
  name: vsr-1
  namespace: default`

	body := ImportRequest{
		Source:  "file",
		Content: content,
		Format:  "virtualserver-yaml",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/migration/import", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp ImportResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.ResourceCount != 2 {
		t.Fatalf("expected 2 resources, got %d", resp.ResourceCount)
	}
}

func TestMigrationHandler_Import_BadRequest(t *testing.T) {
	handler := &MigrationHandler{}

	r := chi.NewRouter()
	r.Post("/migration/import", handler.Import)

	tests := []struct {
		name string
		body ImportRequest
	}{
		{
			name: "missing content",
			body: ImportRequest{Format: "nginx-conf"},
		},
		{
			name: "missing format",
			body: ImportRequest{Content: "server { }"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPost, "/migration/import", bytes.NewReader(bodyBytes))
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestMigrationHandler_Analysis_HappyPath(t *testing.T) {
	handler := &MigrationHandler{}

	r := chi.NewRouter()
	r.Post("/migration/analysis", handler.Analysis)

	body := AnalysisRequest{ImportID: "test-import-123"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/migration/analysis", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp AnalysisResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.OverallScore <= 0 {
		t.Errorf("expected overallScore > 0, got %f", resp.OverallScore)
	}
	if resp.Convertible != 2 {
		t.Errorf("expected 2 convertible, got %d", resp.Convertible)
	}
	if resp.TotalResources != 4 {
		t.Errorf("expected 4 total resources, got %d", resp.TotalResources)
	}
}

func TestMigrationHandler_Analysis_MissingImportID(t *testing.T) {
	handler := &MigrationHandler{}

	r := chi.NewRouter()
	r.Post("/migration/analysis", handler.Analysis)

	body := AnalysisRequest{}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/migration/analysis", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMigrationHandler_Generate_HappyPath(t *testing.T) {
	handler := &MigrationHandler{}

	r := chi.NewRouter()
	r.Post("/migration/generate", handler.Generate)

	body := GenerateRequest{ImportID: "test-import-123"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/migration/generate", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp GenerateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Resources) != 2 {
		t.Errorf("expected 2 resources, got %d", len(resp.Resources))
	}
	if !strings.Contains(resp.YAML, "---") {
		t.Error("expected YAML to contain document separator ---")
	}
}

func TestMigrationHandler_Apply_DryRun(t *testing.T) {
	handler := &MigrationHandler{}

	r := chi.NewRouter()
	r.Post("/migration/apply", handler.Apply)

	body := ApplyRequest{
		ImportID: "test-import-123",
		DryRun:   true,
		Resources: []GeneratedResource{
			{Kind: "Gateway", Name: "gw", Namespace: "default"},
			{Kind: "HTTPRoute", Name: "rt", Namespace: "default"},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/migration/apply", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp ApplyResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if !resp.DryRun {
		t.Error("expected dryRun=true in response")
	}
	if resp.Applied != 2 {
		t.Errorf("expected 2 applied, got %d", resp.Applied)
	}
}

func TestMigrationHandler_Apply_LiveMode(t *testing.T) {
	handler := &MigrationHandler{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "gateway.networking.k8s.io", Version: "v1", Kind: "Gateway"},
		&unstructured.Unstructured{},
	)
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "gateway.networking.k8s.io", Version: "v1", Kind: "GatewayList"},
		&unstructured.UnstructuredList{},
	)
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "gateway.networking.k8s.io", Version: "v1", Kind: "HTTPRoute"},
		&unstructured.Unstructured{},
	)
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "gateway.networking.k8s.io", Version: "v1", Kind: "HTTPRouteList"},
		&unstructured.UnstructuredList{},
	)
	dc := fakedynamic.NewSimpleDynamicClient(scheme)
	k8sClient := kubernetes.NewForTestWithDynamic(nil, dc)

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := cluster.WithClient(r.Context(), k8sClient)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Post("/migration/apply", handler.Apply)

	gatewayYAML := "apiVersion: gateway.networking.k8s.io/v1\nkind: Gateway\nmetadata:\n  name: test-gw\n  namespace: default\nspec:\n  gatewayClassName: nginx"
	routeYAML := "apiVersion: gateway.networking.k8s.io/v1\nkind: HTTPRoute\nmetadata:\n  name: test-route\n  namespace: default\nspec:\n  parentRefs:\n  - name: test-gw"

	body := ApplyRequest{
		ImportID: "test-import-123",
		DryRun:   false,
		Resources: []GeneratedResource{
			{Kind: "Gateway", Name: "test-gw", Namespace: "default", APIVersion: "gateway.networking.k8s.io/v1", YAML: gatewayYAML},
			{Kind: "HTTPRoute", Name: "test-route", Namespace: "default", APIVersion: "gateway.networking.k8s.io/v1", YAML: routeYAML},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/migration/apply", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp ApplyResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.DryRun {
		t.Error("expected dryRun=false in response")
	}
	if resp.Applied != 2 {
		t.Errorf("expected 2 applied, got %d", resp.Applied)
	}
	if resp.Skipped != 0 {
		t.Errorf("expected 0 skipped, got %d", resp.Skipped)
	}
	if len(resp.Errors) != 0 {
		t.Errorf("expected no errors, got %v", resp.Errors)
	}
}

func TestMigrationHandler_Validate_Passed(t *testing.T) {
	handler := &MigrationHandler{}

	scheme := runtime.NewScheme()

	gvr := schema.GroupVersionResource{Group: "gateway.networking.k8s.io", Version: "v1", Resource: "gateways"}
	dc := fakedynamic.NewSimpleDynamicClientWithCustomListKinds(scheme,
		map[schema.GroupVersionResource]string{
			gvr: "GatewayList",
		},
	)

	// Pre-create a Gateway in the fake dynamic client via Create
	existingGW := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "gateway.networking.k8s.io/v1",
			"kind":       "Gateway",
			"metadata": map[string]interface{}{
				"name":      "test-gw",
				"namespace": "default",
			},
			"status": map[string]interface{}{
				"conditions": []interface{}{
					map[string]interface{}{
						"type":    "Accepted",
						"status":  "True",
						"message": "Gateway accepted",
					},
					map[string]interface{}{
						"type":    "Programmed",
						"status":  "True",
						"message": "Gateway programmed",
					},
				},
			},
		},
	}
	_, err := dc.Resource(gvr).Namespace("default").Create(context.Background(), existingGW, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("failed to pre-create gateway: %v", err)
	}
	k8sClient := kubernetes.NewForTestWithDynamic(nil, dc)

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := cluster.WithClient(r.Context(), k8sClient)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Post("/migration/validate", handler.Validate)

	body := ValidateRequest{
		ImportID: "test-import-123",
		Resources: []GeneratedResource{
			{Kind: "Gateway", Name: "test-gw", Namespace: "default", APIVersion: "gateway.networking.k8s.io/v1"},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/migration/validate", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp ValidateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "passed" {
		t.Errorf("expected status 'passed', got %q", resp.Status)
	}
	if len(resp.Checks) < 1 {
		t.Fatal("expected at least one check")
	}
	// Should have exists check + condition checks
	foundExists := false
	for _, c := range resp.Checks {
		if strings.Contains(c.Name, "/exists") && c.Status == "passed" {
			foundExists = true
		}
	}
	if !foundExists {
		t.Error("expected a passed exists check")
	}
}

func TestMigrationHandler_Validate_NotFound(t *testing.T) {
	handler := &MigrationHandler{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "gateway.networking.k8s.io", Version: "v1", Kind: "Gateway"},
		&unstructured.Unstructured{},
	)
	scheme.AddKnownTypeWithName(
		schema.GroupVersionKind{Group: "gateway.networking.k8s.io", Version: "v1", Kind: "GatewayList"},
		&unstructured.UnstructuredList{},
	)
	dc := fakedynamic.NewSimpleDynamicClient(scheme)
	k8sClient := kubernetes.NewForTestWithDynamic(nil, dc)

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := cluster.WithClient(r.Context(), k8sClient)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Post("/migration/validate", handler.Validate)

	body := ValidateRequest{
		ImportID: "test-import-123",
		Resources: []GeneratedResource{
			{Kind: "Gateway", Name: "nonexistent-gw", Namespace: "default", APIVersion: "gateway.networking.k8s.io/v1"},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/migration/validate", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp ValidateResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "failed" {
		t.Errorf("expected status 'failed', got %q", resp.Status)
	}
	if len(resp.Checks) != 1 {
		t.Fatalf("expected 1 check, got %d", len(resp.Checks))
	}
	if resp.Checks[0].Status != "failed" {
		t.Errorf("expected check status 'failed', got %q", resp.Checks[0].Status)
	}
}

func TestMigrationHandler_Validate_MissingResources(t *testing.T) {
	handler := &MigrationHandler{}

	scheme := runtime.NewScheme()
	dc := fakedynamic.NewSimpleDynamicClient(scheme)
	k8sClient := kubernetes.NewForTestWithDynamic(nil, dc)

	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := cluster.WithClient(r.Context(), k8sClient)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})
	r.Post("/migration/validate", handler.Validate)

	body := ValidateRequest{ImportID: "test-import-123"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/migration/validate", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}
