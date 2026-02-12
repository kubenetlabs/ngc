package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
	"github.com/kubenetlabs/ngc/api/internal/kubernetes"
)

// CoexistenceHandler handles NGINX Ingress Controller / NGF coexistence API requests.
type CoexistenceHandler struct{}

// CoexistenceOverview represents the coexistence status of KIC and NGF in a cluster.
type CoexistenceOverview struct {
	KIC             ControllerSummary `json:"kic"`
	NGF             ControllerSummary `json:"ngf"`
	SharedResources []SharedResource  `json:"sharedResources"`
	Conflicts       []Conflict        `json:"conflicts"`
}

// ControllerSummary summarizes a controller's presence and resources.
type ControllerSummary struct {
	Installed     bool            `json:"installed"`
	Version       string          `json:"version,omitempty"`
	ResourceCount int             `json:"resourceCount"`
	Namespaces    []string        `json:"namespaces"`
	Resources     []ResourceCount `json:"resources"`
}

// ResourceCount tracks the count of a specific resource kind.
type ResourceCount struct {
	Kind  string `json:"kind"`
	Count int    `json:"count"`
}

// SharedResource represents a resource used by both controllers.
type SharedResource struct {
	Kind      string   `json:"kind"`
	Name      string   `json:"name"`
	Namespace string   `json:"namespace"`
	UsedBy    []string `json:"usedBy"` // ["kic", "ngf"]
}

// Conflict represents a conflict between KIC and NGF resources.
type Conflict struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Severity    string `json:"severity"` // "high", "medium", "low"
	Resource    string `json:"resource"`
}

// MigrationReadinessResponse represents the readiness assessment for migrating from KIC to NGF.
type MigrationReadinessResponse struct {
	Score           float64             `json:"score"` // 0-100
	Status          string              `json:"status"`
	Categories      []ReadinessCategory `json:"categories"`
	Blockers        []string            `json:"blockers"`
	Recommendations []string            `json:"recommendations"`
}

// ReadinessCategory represents a scored category in the migration readiness assessment.
type ReadinessCategory struct {
	Name    string  `json:"name"`
	Score   float64 `json:"score"`
	Status  string  `json:"status"` // "pass", "warn", "fail"
	Details string  `json:"details"`
}

// GVRs for KIC resources.
var (
	ingressGVR         = schema.GroupVersionResource{Group: "networking.k8s.io", Version: "v1", Resource: "ingresses"}
	virtualServerGVR   = schema.GroupVersionResource{Group: "k8s.nginx.org", Version: "v1", Resource: "virtualservers"}
	virtualServerRouteGVR = schema.GroupVersionResource{Group: "k8s.nginx.org", Version: "v1", Resource: "virtualserverroutes"}
	transportServerGVR = schema.GroupVersionResource{Group: "k8s.nginx.org", Version: "v1", Resource: "transportservers"}
)

// coexistenceData holds discovered data used by both Overview and MigrationReadiness.
type coexistenceData struct {
	// KIC resources
	ingresses          *unstructured.UnstructuredList
	virtualServers     *unstructured.UnstructuredList
	virtualServerRoutes *unstructured.UnstructuredList
	transportServers   *unstructured.UnstructuredList

	// NGF resources
	gatewayClasses []string // controller names
	gatewayCount   int
	httpRouteCount int
	ngfNamespaces  map[string]bool

	// Computed
	kicNamespaces map[string]bool

	// Backend references keyed by "namespace/name"
	ingressBackends  map[string][]int32 // service key -> ports
	httpRouteBackends map[string][]int32 // service key -> ports

	// Hostnames
	ingressHostnames  map[string]string // hostname -> "namespace/name"
	httpRouteHostnames map[string]string // hostname -> "namespace/name"
}

// Overview returns the coexistence status overview.
func (h *CoexistenceHandler) Overview(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	data, err := h.discover(r.Context(), k8s)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("discovering resources: %v", err))
		return
	}

	overview := h.buildOverview(data)
	writeJSON(w, http.StatusOK, overview)
}

// MigrationReadiness returns the readiness assessment for migration.
func (h *CoexistenceHandler) MigrationReadiness(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	data, err := h.discover(r.Context(), k8s)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("discovering resources: %v", err))
		return
	}

	overview := h.buildOverview(data)
	readiness := h.assessReadiness(data, overview)
	writeJSON(w, http.StatusOK, readiness)
}

// discover gathers all KIC and NGF resources from the cluster.
func (h *CoexistenceHandler) discover(ctx context.Context, k8s *kubernetes.Client) (*coexistenceData, error) {
	data := &coexistenceData{
		kicNamespaces:      make(map[string]bool),
		ngfNamespaces:      make(map[string]bool),
		ingressBackends:    make(map[string][]int32),
		httpRouteBackends:  make(map[string][]int32),
		ingressHostnames:   make(map[string]string),
		httpRouteHostnames: make(map[string]string),
	}

	dc := k8s.DynamicClient()

	// Detect KIC: Ingress resources
	ingresses, err := dc.Resource(ingressGVR).Namespace("").List(ctx, metav1.ListOptions{})
	if err != nil {
		// Ingress API should always be available, but handle gracefully
		ingresses = &unstructured.UnstructuredList{}
	}
	data.ingresses = ingresses

	// Detect KIC: VirtualServer CRDs (may not exist)
	vs, err := dc.Resource(virtualServerGVR).Namespace("").List(ctx, metav1.ListOptions{})
	if err != nil {
		vs = &unstructured.UnstructuredList{}
	}
	data.virtualServers = vs

	// Detect KIC: VirtualServerRoute CRDs (may not exist)
	vsr, err := dc.Resource(virtualServerRouteGVR).Namespace("").List(ctx, metav1.ListOptions{})
	if err != nil {
		vsr = &unstructured.UnstructuredList{}
	}
	data.virtualServerRoutes = vsr

	// Detect KIC: TransportServer CRDs (may not exist)
	ts, err := dc.Resource(transportServerGVR).Namespace("").List(ctx, metav1.ListOptions{})
	if err != nil {
		ts = &unstructured.UnstructuredList{}
	}
	data.transportServers = ts

	// Collect KIC namespaces and backends from Ingresses
	for _, ing := range ingresses.Items {
		ns := ing.GetNamespace()
		data.kicNamespaces[ns] = true
		h.extractIngressBackends(&ing, data)
		h.extractIngressHostnames(&ing, data)
	}
	for _, v := range vs.Items {
		data.kicNamespaces[v.GetNamespace()] = true
		h.extractVirtualServerHostnames(&v, data)
	}
	for _, v := range vsr.Items {
		data.kicNamespaces[v.GetNamespace()] = true
	}
	for _, t := range ts.Items {
		data.kicNamespaces[t.GetNamespace()] = true
	}

	// Detect NGF: Gateway API resources using typed client
	gateways, err := k8s.ListGateways(ctx, "")
	if err != nil {
		gateways = nil
	}
	data.gatewayCount = len(gateways)
	for _, gw := range gateways {
		data.ngfNamespaces[gw.Namespace] = true
	}

	httpRoutes, err := k8s.ListHTTPRoutes(ctx, "")
	if err != nil {
		httpRoutes = nil
	}
	data.httpRouteCount = len(httpRoutes)
	for _, hr := range httpRoutes {
		data.ngfNamespaces[hr.Namespace] = true
		// Extract HTTPRoute backends
		for _, rule := range hr.Spec.Rules {
			for _, br := range rule.BackendRefs {
				backendNS := hr.Namespace
				if br.Namespace != nil {
					backendNS = string(*br.Namespace)
				}
				key := backendNS + "/" + string(br.Name)
				var port int32
				if br.Port != nil {
					port = int32(*br.Port)
				}
				data.httpRouteBackends[key] = appendUniquePort(data.httpRouteBackends[key], port)
			}
		}
		// Extract HTTPRoute hostnames
		for _, hostname := range hr.Spec.Hostnames {
			data.httpRouteHostnames[string(hostname)] = hr.Namespace + "/" + hr.Name
		}
	}

	classes, err := k8s.ListGatewayClasses(ctx)
	if err != nil {
		classes = nil
	}
	data.gatewayClasses = make([]string, 0, len(classes))
	for _, gc := range classes {
		data.gatewayClasses = append(data.gatewayClasses, string(gc.Spec.ControllerName))
	}

	return data, nil
}

// buildOverview constructs the CoexistenceOverview from discovered data.
func (h *CoexistenceHandler) buildOverview(data *coexistenceData) CoexistenceOverview {
	overview := CoexistenceOverview{
		SharedResources: make([]SharedResource, 0),
		Conflicts:       make([]Conflict, 0),
	}

	// Build KIC summary
	ingressCount := len(data.ingresses.Items)
	vsCount := len(data.virtualServers.Items)
	vsrCount := len(data.virtualServerRoutes.Items)
	tsCount := len(data.transportServers.Items)
	kicTotal := ingressCount + vsCount + vsrCount + tsCount

	overview.KIC = ControllerSummary{
		Installed:     kicTotal > 0,
		ResourceCount: kicTotal,
		Namespaces:    mapKeys(data.kicNamespaces),
		Resources:     make([]ResourceCount, 0),
	}

	if ingressCount > 0 {
		overview.KIC.Resources = append(overview.KIC.Resources, ResourceCount{Kind: "Ingress", Count: ingressCount})
	}
	if vsCount > 0 {
		overview.KIC.Resources = append(overview.KIC.Resources, ResourceCount{Kind: "VirtualServer", Count: vsCount})
	}
	if vsrCount > 0 {
		overview.KIC.Resources = append(overview.KIC.Resources, ResourceCount{Kind: "VirtualServerRoute", Count: vsrCount})
	}
	if tsCount > 0 {
		overview.KIC.Resources = append(overview.KIC.Resources, ResourceCount{Kind: "TransportServer", Count: tsCount})
	}

	// Try to detect KIC version from Ingress annotations
	overview.KIC.Version = h.detectKICVersion(data)

	// Build NGF summary
	ngfTotal := data.gatewayCount + data.httpRouteCount + len(data.gatewayClasses)

	overview.NGF = ControllerSummary{
		Installed:     ngfTotal > 0,
		ResourceCount: ngfTotal,
		Namespaces:    mapKeys(data.ngfNamespaces),
		Resources:     make([]ResourceCount, 0),
	}

	if len(data.gatewayClasses) > 0 {
		overview.NGF.Resources = append(overview.NGF.Resources, ResourceCount{Kind: "GatewayClass", Count: len(data.gatewayClasses)})
	}
	if data.gatewayCount > 0 {
		overview.NGF.Resources = append(overview.NGF.Resources, ResourceCount{Kind: "Gateway", Count: data.gatewayCount})
	}
	if data.httpRouteCount > 0 {
		overview.NGF.Resources = append(overview.NGF.Resources, ResourceCount{Kind: "HTTPRoute", Count: data.httpRouteCount})
	}

	// Detect NGF version from GatewayClass controller names
	for _, cn := range data.gatewayClasses {
		if strings.Contains(strings.ToLower(cn), "nginx") {
			// Extract version hint from controller name if present
			overview.NGF.Version = cn
			break
		}
	}

	// Find shared resources: Services referenced by both Ingress backends and HTTPRoute backendRefs
	for svcKey, ingressPorts := range data.ingressBackends {
		if httpRoutePorts, ok := data.httpRouteBackends[svcKey]; ok {
			parts := strings.SplitN(svcKey, "/", 2)
			ns, name := parts[0], parts[1]
			overview.SharedResources = append(overview.SharedResources, SharedResource{
				Kind:      "Service",
				Name:      name,
				Namespace: ns,
				UsedBy:    []string{"kic", "ngf"},
			})

			// Check for port conflicts on shared services
			conflicts := findPortConflicts(ingressPorts, httpRoutePorts)
			for _, port := range conflicts {
				overview.Conflicts = append(overview.Conflicts, Conflict{
					Type:        "port-conflict",
					Description: fmt.Sprintf("Service %s/%s port %d is used by both Ingress and HTTPRoute backends", ns, name, port),
					Severity:    "high",
					Resource:    svcKey,
				})
			}
		}
	}

	// Detect hostname overlaps between Ingress and HTTPRoute
	for hostname, ingressRef := range data.ingressHostnames {
		if httpRouteRef, ok := data.httpRouteHostnames[hostname]; ok {
			overview.Conflicts = append(overview.Conflicts, Conflict{
				Type:        "hostname-overlap",
				Description: fmt.Sprintf("Hostname %q is used by both Ingress %s and HTTPRoute %s", hostname, ingressRef, httpRouteRef),
				Severity:    "medium",
				Resource:    hostname,
			})
		}
	}

	return overview
}

// assessReadiness scores migration readiness from KIC to NGF.
func (h *CoexistenceHandler) assessReadiness(data *coexistenceData, overview CoexistenceOverview) MigrationReadinessResponse {
	categories := make([]ReadinessCategory, 0, 4)
	blockers := make([]string, 0)
	recommendations := make([]string, 0)

	// Category 1: Gateway API CRDs Installed (25 points)
	crdCategory := ReadinessCategory{
		Name:  "Gateway API CRDs Installed",
		Score: 0,
	}
	// If we were able to list Gateway API resources (even if empty), CRDs are installed.
	// The presence of GatewayClasses, Gateways, or HTTPRoutes indicates the CRDs exist.
	crdInstalled := len(data.gatewayClasses) > 0 || data.gatewayCount > 0 || data.httpRouteCount > 0
	if crdInstalled {
		crdCategory.Score = 25
		crdCategory.Status = "pass"
		crdCategory.Details = "Gateway, HTTPRoute, and GatewayClass CRDs are installed"
	} else {
		crdCategory.Status = "fail"
		crdCategory.Details = "Gateway API CRDs are not installed or no resources found"
		blockers = append(blockers, "Gateway API CRDs must be installed before migration")
		recommendations = append(recommendations, "Install Gateway API CRDs: kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/latest/download/standard-install.yaml")
	}
	categories = append(categories, crdCategory)

	// Category 2: NGF Controller Running (25 points)
	ngfCategory := ReadinessCategory{
		Name:  "NGF Controller Running",
		Score: 0,
	}
	ngfControllerFound := false
	for _, cn := range data.gatewayClasses {
		if strings.Contains(strings.ToLower(cn), "nginx") {
			ngfControllerFound = true
			break
		}
	}
	if ngfControllerFound {
		ngfCategory.Score = 25
		ngfCategory.Status = "pass"
		ngfCategory.Details = "NGINX Gateway Fabric controller is running with a registered GatewayClass"
	} else if len(data.gatewayClasses) > 0 {
		ngfCategory.Score = 10
		ngfCategory.Status = "warn"
		ngfCategory.Details = fmt.Sprintf("GatewayClasses found but none with an NGINX controller name (found: %s)", strings.Join(data.gatewayClasses, ", "))
		recommendations = append(recommendations, "Deploy NGINX Gateway Fabric controller and ensure a GatewayClass with an NGINX controller name exists")
	} else {
		ngfCategory.Status = "fail"
		ngfCategory.Details = "No GatewayClasses found; NGF controller does not appear to be running"
		blockers = append(blockers, "NGINX Gateway Fabric controller must be deployed before migration")
		recommendations = append(recommendations, "Install NGINX Gateway Fabric: https://docs.nginx.com/nginx-gateway-fabric/installation/")
	}
	categories = append(categories, ngfCategory)

	// Category 3: Resource Compatibility (25 points)
	compatCategory := ReadinessCategory{
		Name:  "Resource Compatibility",
		Score: 0,
	}
	ingressCount := len(data.ingresses.Items)
	vsCount := len(data.virtualServers.Items)
	tsCount := len(data.transportServers.Items)
	totalKICResources := ingressCount + vsCount + tsCount

	if totalKICResources == 0 {
		// No KIC resources to migrate
		compatCategory.Score = 25
		compatCategory.Status = "pass"
		compatCategory.Details = "No KIC resources to migrate"
	} else {
		// Ingress = easy (1.0 convertibility), VirtualServer = medium (0.5), TransportServer = hard (0.2)
		convertibleScore := float64(ingressCount)*1.0 + float64(vsCount)*0.5 + float64(tsCount)*0.2
		maxScore := float64(totalKICResources) * 1.0
		ratio := convertibleScore / maxScore
		compatCategory.Score = ratio * 25

		if ratio >= 0.8 {
			compatCategory.Status = "pass"
		} else if ratio >= 0.5 {
			compatCategory.Status = "warn"
		} else {
			compatCategory.Status = "fail"
		}

		details := make([]string, 0)
		if ingressCount > 0 {
			details = append(details, fmt.Sprintf("%d Ingress resources (easy to convert)", ingressCount))
		}
		if vsCount > 0 {
			details = append(details, fmt.Sprintf("%d VirtualServer resources (medium complexity)", vsCount))
			recommendations = append(recommendations, "VirtualServer resources require manual conversion to HTTPRoute; review each for advanced features")
		}
		if tsCount > 0 {
			details = append(details, fmt.Sprintf("%d TransportServer resources (hard to convert)", tsCount))
			recommendations = append(recommendations, "TransportServer resources may require TCPRoute/TLSRoute; verify Gateway API support for your use case")
		}
		compatCategory.Details = strings.Join(details, "; ")
	}
	categories = append(categories, compatCategory)

	// Category 4: No Conflicts (25 points)
	conflictCategory := ReadinessCategory{
		Name:  "No Conflicts",
		Score: 25,
	}

	portConflicts := 0
	hostnameOverlaps := 0
	for _, c := range overview.Conflicts {
		switch c.Type {
		case "port-conflict":
			portConflicts++
		case "hostname-overlap":
			hostnameOverlaps++
		}
	}

	totalConflicts := len(overview.Conflicts)
	if totalConflicts > 0 {
		// Deduct points per conflict, min 0
		deduction := float64(totalConflicts) * 5
		if deduction > 25 {
			deduction = 25
		}
		conflictCategory.Score = 25 - deduction

		if conflictCategory.Score >= 15 {
			conflictCategory.Status = "warn"
		} else {
			conflictCategory.Status = "fail"
		}

		details := make([]string, 0)
		if portConflicts > 0 {
			details = append(details, fmt.Sprintf("%d port conflict(s)", portConflicts))
			recommendations = append(recommendations, "Resolve port conflicts on shared services before migration to avoid traffic disruption")
		}
		if hostnameOverlaps > 0 {
			details = append(details, fmt.Sprintf("%d hostname overlap(s)", hostnameOverlaps))
			recommendations = append(recommendations, "Review hostname overlaps between Ingress and HTTPRoute resources to prevent routing conflicts")
		}
		conflictCategory.Details = strings.Join(details, "; ")
	} else {
		conflictCategory.Status = "pass"
		conflictCategory.Details = "No conflicts detected between KIC and NGF resources"
	}
	categories = append(categories, conflictCategory)

	// Calculate overall score
	var totalScore float64
	for _, cat := range categories {
		totalScore += cat.Score
		if cat.Score < 25*0.25 { // Below 25% of category max
			blockers = append(blockers, fmt.Sprintf("%s: %s", cat.Name, cat.Details))
		}
	}

	// Determine status
	status := "not-ready"
	if totalScore >= 75 {
		status = "ready"
	} else if totalScore >= 50 {
		status = "partial"
	}

	return MigrationReadinessResponse{
		Score:           totalScore,
		Status:          status,
		Categories:      categories,
		Blockers:        blockers,
		Recommendations: recommendations,
	}
}

// extractIngressBackends extracts backend service references from an Ingress resource.
func (h *CoexistenceHandler) extractIngressBackends(ing *unstructured.Unstructured, data *coexistenceData) {
	ns := ing.GetNamespace()

	// Extract from spec.rules[].http.paths[].backend.service
	rules, found, _ := unstructured.NestedSlice(ing.Object, "spec", "rules")
	if found {
		for _, rule := range rules {
			ruleMap, ok := rule.(map[string]interface{})
			if !ok {
				continue
			}
			paths, found, _ := unstructured.NestedSlice(ruleMap, "http", "paths")
			if !found {
				continue
			}
			for _, path := range paths {
				pathMap, ok := path.(map[string]interface{})
				if !ok {
					continue
				}
				svcName, found, _ := unstructured.NestedString(pathMap, "backend", "service", "name")
				if !found {
					continue
				}
				key := ns + "/" + svcName
				portNum, found, _ := unstructured.NestedFieldNoCopy(pathMap, "backend", "service", "port", "number")
				if found {
					if p, ok := portNum.(int64); ok {
						data.ingressBackends[key] = appendUniquePort(data.ingressBackends[key], int32(p))
					}
				} else {
					data.ingressBackends[key] = appendUniquePort(data.ingressBackends[key], 0)
				}
			}
		}
	}

	// Extract from spec.defaultBackend.service
	defaultSvc, found, _ := unstructured.NestedString(ing.Object, "spec", "defaultBackend", "service", "name")
	if found {
		key := ns + "/" + defaultSvc
		portNum, found, _ := unstructured.NestedFieldNoCopy(ing.Object, "spec", "defaultBackend", "service", "port", "number")
		if found {
			if p, ok := portNum.(int64); ok {
				data.ingressBackends[key] = appendUniquePort(data.ingressBackends[key], int32(p))
			}
		} else {
			data.ingressBackends[key] = appendUniquePort(data.ingressBackends[key], 0)
		}
	}
}

// extractIngressHostnames extracts hostnames from an Ingress resource.
func (h *CoexistenceHandler) extractIngressHostnames(ing *unstructured.Unstructured, data *coexistenceData) {
	ns := ing.GetNamespace()
	name := ing.GetName()
	ref := ns + "/" + name

	rules, found, _ := unstructured.NestedSlice(ing.Object, "spec", "rules")
	if !found {
		return
	}
	for _, rule := range rules {
		ruleMap, ok := rule.(map[string]interface{})
		if !ok {
			continue
		}
		host, found, _ := unstructured.NestedString(ruleMap, "host")
		if found && host != "" {
			data.ingressHostnames[host] = ref
		}
	}
}

// extractVirtualServerHostnames extracts the host from a VirtualServer resource.
func (h *CoexistenceHandler) extractVirtualServerHostnames(vs *unstructured.Unstructured, data *coexistenceData) {
	ns := vs.GetNamespace()
	name := vs.GetName()
	ref := ns + "/" + name

	host, found, _ := unstructured.NestedString(vs.Object, "spec", "host")
	if found && host != "" {
		data.ingressHostnames[host] = ref
	}
}

// detectKICVersion attempts to determine KIC version from resource annotations.
func (h *CoexistenceHandler) detectKICVersion(data *coexistenceData) string {
	// Check Ingress annotations for KIC class indicator
	for _, ing := range data.ingresses.Items {
		annotations := ing.GetAnnotations()
		if annotations == nil {
			continue
		}
		// Check for kubernetes.io/ingress.class annotation
		if class, ok := annotations["kubernetes.io/ingress.class"]; ok {
			if strings.Contains(strings.ToLower(class), "nginx") {
				return "nginx-ingress-controller"
			}
		}
	}

	// If there are VirtualServer resources, KIC is definitely present
	if len(data.virtualServers.Items) > 0 {
		return "nginx-ingress-controller (VirtualServer CRDs detected)"
	}

	return ""
}

// findPortConflicts returns ports that appear in both slices.
func findPortConflicts(a, b []int32) []int32 {
	set := make(map[int32]bool, len(a))
	for _, p := range a {
		if p != 0 {
			set[p] = true
		}
	}
	conflicts := make([]int32, 0)
	for _, p := range b {
		if p != 0 && set[p] {
			conflicts = append(conflicts, p)
		}
	}
	return conflicts
}

// appendUniquePort appends a port to the slice only if it's not already present.
func appendUniquePort(ports []int32, port int32) []int32 {
	for _, p := range ports {
		if p == port {
			return ports
		}
	}
	return append(ports, port)
}

// mapKeys returns sorted keys from a bool map.
func mapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Sort for deterministic output
	sortStrings(keys)
	return keys
}

// sortStrings sorts a string slice in place.
func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
