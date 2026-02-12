package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// MigrationHandler handles NGINX config migration API requests.
type MigrationHandler struct{}

// --- Request / Response types ---

// ImportRequest describes the source configuration to import.
type ImportRequest struct {
	Source  string `json:"source"`  // "file", "cluster", "url"
	Content string `json:"content"` // raw NGINX config or Ingress YAML
	Format  string `json:"format"`  // "nginx-conf", "ingress-yaml", "virtualserver-yaml"
}

// ImportResponse is returned after a successful import.
type ImportResponse struct {
	ID            string               `json:"id"`
	ResourceCount int                  `json:"resourceCount"`
	Resources     []DiscoveredResource `json:"resources"`
}

// DiscoveredResource represents a single resource found during import.
type DiscoveredResource struct {
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	APIVersion string `json:"apiVersion"`
}

// AnalysisRequest asks for analysis of a previous import.
type AnalysisRequest struct {
	ImportID string `json:"importId"`
}

// AnalysisResponse contains the migration compatibility analysis.
type AnalysisResponse struct {
	ImportID       string         `json:"importId"`
	OverallScore   float64        `json:"overallScore"`
	TotalResources int            `json:"totalResources"`
	Convertible    int            `json:"convertible"`
	NeedsReview    int            `json:"needsReview"`
	Unsupported    int            `json:"unsupported"`
	Items          []AnalysisItem `json:"items"`
}

// AnalysisItem describes the migration analysis for a single resource.
type AnalysisItem struct {
	Source     DiscoveredResource `json:"source"`
	Target     string             `json:"target"`
	Confidence string             `json:"confidence"`
	Issues     []string           `json:"issues"`
	Notes      []string           `json:"notes"`
}

// GenerateRequest asks for Gateway API resource generation.
type GenerateRequest struct {
	ImportID string `json:"importId"`
}

// GenerateResponse contains the generated Gateway API resources.
type GenerateResponse struct {
	ImportID  string              `json:"importId"`
	Resources []GeneratedResource `json:"resources"`
	YAML      string              `json:"yaml"`
}

// GeneratedResource is a single generated Gateway API resource.
type GeneratedResource struct {
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	APIVersion string `json:"apiVersion"`
	YAML       string `json:"yaml"`
}

// ApplyRequest asks to apply generated resources to a cluster.
type ApplyRequest struct {
	ImportID  string              `json:"importId"`
	DryRun    bool                `json:"dryRun"`
	Resources []GeneratedResource `json:"resources"`
}

// ApplyResponse describes the result of applying resources.
type ApplyResponse struct {
	Applied int      `json:"applied"`
	Skipped int      `json:"skipped"`
	Errors  []string `json:"errors"`
	DryRun  bool     `json:"dryRun"`
}

// ValidateRequest asks for validation of migrated resources.
type ValidateRequest struct {
	ImportID string `json:"importId"`
}

// ValidateResponse describes the validation result.
type ValidateResponse struct {
	ImportID string            `json:"importId"`
	Status   string            `json:"status"`
	Checks   []ValidationCheck `json:"checks"`
}

// ValidationCheck is a single validation check result.
type ValidationCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

// generateID produces a random hex ID suitable for tracking imports.
func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// Import imports an existing NGINX configuration.
func (h *MigrationHandler) Import(w http.ResponseWriter, r *http.Request) {
	var req ImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "content is required")
		return
	}
	if req.Format == "" {
		writeError(w, http.StatusBadRequest, "format is required")
		return
	}

	id := generateID()
	resources := discoverResources(req.Content, req.Format)

	writeJSON(w, http.StatusOK, ImportResponse{
		ID:            id,
		ResourceCount: len(resources),
		Resources:     resources,
	})
}

// discoverResources parses the supplied content and returns discovered resources.
func discoverResources(content, format string) []DiscoveredResource {
	var resources []DiscoveredResource

	switch format {
	case "nginx-conf":
		resources = parseNginxConf(content)
	case "ingress-yaml":
		resources = parseIngressYAML(content)
	case "virtualserver-yaml":
		resources = parseVirtualServerYAML(content)
	default:
		// Unknown format — return empty list.
		return nil
	}

	return resources
}

// parseNginxConf extracts mock resources from NGINX config by counting
// server and location blocks.
func parseNginxConf(content string) []DiscoveredResource {
	var resources []DiscoveredResource

	// Count server blocks → each becomes a Gateway candidate.
	serverCount := strings.Count(content, "server {") + strings.Count(content, "server{")
	if serverCount == 0 {
		serverCount = 1 // at minimum one gateway from any config
	}
	for i := 0; i < serverCount; i++ {
		resources = append(resources, DiscoveredResource{
			Kind:       "Gateway",
			Name:       fmt.Sprintf("nginx-gateway-%d", i+1),
			Namespace:  "default",
			APIVersion: "gateway.networking.k8s.io/v1",
		})
	}

	// Count location blocks → each becomes an HTTPRoute candidate.
	locationCount := strings.Count(content, "location ") + strings.Count(content, "location\t")
	if locationCount == 0 {
		locationCount = 1
	}
	for i := 0; i < locationCount; i++ {
		resources = append(resources, DiscoveredResource{
			Kind:       "HTTPRoute",
			Name:       fmt.Sprintf("nginx-route-%d", i+1),
			Namespace:  "default",
			APIVersion: "gateway.networking.k8s.io/v1",
		})
	}

	return resources
}

// parseIngressYAML counts YAML documents with kind: Ingress.
func parseIngressYAML(content string) []DiscoveredResource {
	var resources []DiscoveredResource

	docs := strings.Split(content, "---")
	idx := 0
	for _, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}
		if strings.Contains(doc, "kind: Ingress") || strings.Contains(doc, "kind:Ingress") {
			name := extractYAMLField(doc, "name")
			ns := extractYAMLField(doc, "namespace")
			if name == "" {
				name = fmt.Sprintf("ingress-%d", idx+1)
			}
			if ns == "" {
				ns = "default"
			}
			resources = append(resources, DiscoveredResource{
				Kind:       "Ingress",
				Name:       name,
				Namespace:  ns,
				APIVersion: "networking.k8s.io/v1",
			})
			idx++
		}
	}

	if len(resources) == 0 {
		resources = append(resources, DiscoveredResource{
			Kind:       "Ingress",
			Name:       "ingress-1",
			Namespace:  "default",
			APIVersion: "networking.k8s.io/v1",
		})
	}

	return resources
}

// parseVirtualServerYAML counts YAML documents with kind: VirtualServer.
func parseVirtualServerYAML(content string) []DiscoveredResource {
	var resources []DiscoveredResource

	docs := strings.Split(content, "---")
	idx := 0
	for _, doc := range docs {
		doc = strings.TrimSpace(doc)
		if doc == "" {
			continue
		}
		kind := ""
		apiVersion := ""
		if strings.Contains(doc, "kind: VirtualServer") || strings.Contains(doc, "kind:VirtualServer") {
			kind = "VirtualServer"
			apiVersion = "k8s.nginx.org/v1"
		} else if strings.Contains(doc, "kind: VirtualServerRoute") || strings.Contains(doc, "kind:VirtualServerRoute") {
			kind = "VirtualServerRoute"
			apiVersion = "k8s.nginx.org/v1"
		} else if strings.Contains(doc, "kind: TransportServer") || strings.Contains(doc, "kind:TransportServer") {
			kind = "TransportServer"
			apiVersion = "k8s.nginx.org/v1alpha1"
		}
		if kind == "" {
			continue
		}
		name := extractYAMLField(doc, "name")
		ns := extractYAMLField(doc, "namespace")
		if name == "" {
			name = fmt.Sprintf("%s-%d", strings.ToLower(kind), idx+1)
		}
		if ns == "" {
			ns = "default"
		}
		resources = append(resources, DiscoveredResource{
			Kind:       kind,
			Name:       name,
			Namespace:  ns,
			APIVersion: apiVersion,
		})
		idx++
	}

	if len(resources) == 0 {
		resources = append(resources, DiscoveredResource{
			Kind:       "VirtualServer",
			Name:       "virtualserver-1",
			Namespace:  "default",
			APIVersion: "k8s.nginx.org/v1",
		})
	}

	return resources
}

// extractYAMLField does a simple line-based extraction of a YAML field value.
// It looks for the first line matching "  <field>: <value>" (under metadata:).
func extractYAMLField(doc, field string) string {
	lines := strings.Split(doc, "\n")
	inMetadata := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "metadata:" {
			inMetadata = true
			continue
		}
		if inMetadata {
			// Stop if we hit a non-indented line that isn't a metadata field.
			if len(line) > 0 && line[0] != ' ' && line[0] != '\t' {
				break
			}
			prefix := field + ":"
			if strings.HasPrefix(trimmed, prefix) {
				val := strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
				val = strings.Trim(val, "\"'")
				return val
			}
		}
	}
	return ""
}

// Analysis analyzes an imported configuration for migration compatibility.
func (h *MigrationHandler) Analysis(w http.ResponseWriter, r *http.Request) {
	var req AnalysisRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if req.ImportID == "" {
		writeError(w, http.StatusBadRequest, "importId is required")
		return
	}

	// Build mock analysis items representing a realistic migration scenario.
	items := []AnalysisItem{
		{
			Source: DiscoveredResource{
				Kind: "Ingress", Name: "web-ingress", Namespace: "default",
				APIVersion: "networking.k8s.io/v1",
			},
			Target:     "HTTPRoute",
			Confidence: "high",
			Issues:     []string{},
			Notes:      []string{"Direct mapping available for path-based routing rules"},
		},
		{
			Source: DiscoveredResource{
				Kind: "Ingress", Name: "api-ingress", Namespace: "default",
				APIVersion: "networking.k8s.io/v1",
			},
			Target:     "HTTPRoute",
			Confidence: "high",
			Issues:     []string{},
			Notes:      []string{"TLS termination will be moved to Gateway listener"},
		},
		{
			Source: DiscoveredResource{
				Kind: "VirtualServer", Name: "app-vs", Namespace: "default",
				APIVersion: "k8s.nginx.org/v1",
			},
			Target:     "HTTPRoute",
			Confidence: "medium",
			Issues:     []string{"Rate limiting annotations require policy attachment"},
			Notes:      []string{"Most routing rules map to HTTPRoute matches"},
		},
		{
			Source: DiscoveredResource{
				Kind: "TransportServer", Name: "tcp-ts", Namespace: "default",
				APIVersion: "k8s.nginx.org/v1alpha1",
			},
			Target:     "TCPRoute",
			Confidence: "low",
			Issues:     []string{"TCPRoute support varies by implementation", "Session persistence not directly supported"},
			Notes:      []string{"Manual review recommended"},
		},
	}

	convertible := 0
	needsReview := 0
	unsupported := 0
	for _, item := range items {
		switch item.Confidence {
		case "high":
			convertible++
		case "medium":
			needsReview++
		case "low":
			unsupported++
		}
	}

	total := len(items)
	score := 0.0
	if total > 0 {
		score = float64(convertible*100+needsReview*60+unsupported*20) / float64(total)
	}

	writeJSON(w, http.StatusOK, AnalysisResponse{
		ImportID:       req.ImportID,
		OverallScore:   score,
		TotalResources: total,
		Convertible:    convertible,
		NeedsReview:    needsReview,
		Unsupported:    unsupported,
		Items:          items,
	})
}

// Generate produces Gateway API resources from the analyzed configuration.
func (h *MigrationHandler) Generate(w http.ResponseWriter, r *http.Request) {
	var req GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if req.ImportID == "" {
		writeError(w, http.StatusBadRequest, "importId is required")
		return
	}

	gatewayYAML := `apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: migrated-gateway
  namespace: default
spec:
  gatewayClassName: nginx
  listeners:
  - name: http
    port: 80
    protocol: HTTP
  - name: https
    port: 443
    protocol: HTTPS
    tls:
      mode: Terminate
      certificateRefs:
      - name: tls-secret`

	routeYAML := `apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: migrated-route
  namespace: default
spec:
  parentRefs:
  - name: migrated-gateway
  hostnames:
  - "app.example.com"
  rules:
  - matches:
    - path:
        type: PathPrefix
        value: /
    backendRefs:
    - name: app-service
      port: 80`

	combinedYAML := gatewayYAML + "\n---\n" + routeYAML

	resources := []GeneratedResource{
		{
			Kind:       "Gateway",
			Name:       "migrated-gateway",
			Namespace:  "default",
			APIVersion: "gateway.networking.k8s.io/v1",
			YAML:       gatewayYAML,
		},
		{
			Kind:       "HTTPRoute",
			Name:       "migrated-route",
			Namespace:  "default",
			APIVersion: "gateway.networking.k8s.io/v1",
			YAML:       routeYAML,
		},
	}

	writeJSON(w, http.StatusOK, GenerateResponse{
		ImportID:  req.ImportID,
		Resources: resources,
		YAML:      combinedYAML,
	})
}

// Apply applies generated Gateway API resources to the cluster.
func (h *MigrationHandler) Apply(w http.ResponseWriter, r *http.Request) {
	var req ApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if req.ImportID == "" {
		writeError(w, http.StatusBadRequest, "importId is required")
		return
	}

	resourceCount := len(req.Resources)
	if resourceCount == 0 {
		// Default to a mock resource count when none are provided.
		resourceCount = 2
	}

	if req.DryRun {
		writeJSON(w, http.StatusOK, ApplyResponse{
			Applied: resourceCount,
			Skipped: 0,
			Errors:  []string{},
			DryRun:  true,
		})
		return
	}

	// Real implementation requires dynamic client to apply unstructured resources.
	// Return 501 until cluster-backed apply is implemented.
	writeError(w, http.StatusNotImplemented, "cluster-backed apply is not yet implemented; use dry-run mode to preview")
}

// Validate validates migrated resources against the running gateway.
func (h *MigrationHandler) Validate(w http.ResponseWriter, r *http.Request) {
	var req ValidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if req.ImportID == "" {
		writeError(w, http.StatusBadRequest, "importId is required")
		return
	}

	// Real implementation requires cluster access to validate applied resources.
	// Return 501 until cluster-backed validation is implemented.
	writeError(w, http.StatusNotImplemented, "cluster-backed validation is not yet implemented")
}
