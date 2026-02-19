package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
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

	r := chi.NewRouter()
	r.Post("/migration/apply", handler.Apply)

	body := ApplyRequest{ImportID: "test-import-123", DryRun: false}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/migration/apply", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Fatalf("expected status 501, got %d: %s", w.Code, w.Body.String())
	}
}

func TestMigrationHandler_Validate_Returns501(t *testing.T) {
	handler := &MigrationHandler{}

	r := chi.NewRouter()
	r.Post("/migration/validate", handler.Validate)

	body := ValidateRequest{ImportID: "test-import-123"}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/migration/validate", bytes.NewReader(bodyBytes))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Fatalf("expected status 501, got %d: %s", w.Code, w.Body.String())
	}
}
