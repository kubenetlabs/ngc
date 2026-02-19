package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
	"github.com/kubenetlabs/ngc/api/internal/database"
	"github.com/kubenetlabs/ngc/api/internal/xc"
)

var distributedCloudPublishGVR = schema.GroupVersionResource{
	Group:    "ngf-console.f5.com",
	Version:  "v1alpha1",
	Resource: "distributedcloudpublishes",
}

// XC request/response types

// XCStatusResponse represents XC connectivity status.
type XCStatusResponse struct {
	Connected      bool   `json:"connected"`
	PublishCount   int    `json:"publishCount"`
	XCConnected    bool   `json:"xcConnected"`
	Tenant         string `json:"tenant,omitempty"`
}

// XCPublishRequest represents a request to create a DistributedCloudPublish resource.
type XCPublishRequest struct {
	Name             string                 `json:"name"`
	Namespace        string                 `json:"namespace"`
	HTTPRouteRef     string                 `json:"httpRouteRef"`
	InferencePoolRef string                 `json:"inferencePoolRef,omitempty"`
	PublicHostname   string                 `json:"publicHostname,omitempty"`
	OriginAddress    string                 `json:"originAddress,omitempty"`
	WAFEnabled       bool                   `json:"wafEnabled,omitempty"`
	WAFPolicyName    string                 `json:"wafPolicyName,omitempty"`
	DistributedCloud map[string]interface{} `json:"distributedCloud,omitempty"`
}

// XCPublishResponse represents a DistributedCloudPublish resource response.
type XCPublishResponse struct {
	Name               string `json:"name"`
	Namespace          string `json:"namespace"`
	HTTPRouteRef       string `json:"httpRouteRef"`
	InferencePoolRef   string `json:"inferencePoolRef,omitempty"`
	Phase              string `json:"phase"`
	XCLoadBalancerName string `json:"xcLoadBalancerName,omitempty"`
	XCOriginPoolName   string `json:"xcOriginPoolName,omitempty"`
	XCVirtualIP        string `json:"xcVirtualIP,omitempty"`
	XCDNS              string `json:"xcDNS,omitempty"`
	WAFPolicyAttached  string `json:"wafPolicyAttached,omitempty"`
	LastSyncedAt       string `json:"lastSyncedAt,omitempty"`
	CreatedAt          string `json:"createdAt"`
}

// XCMetricsResponse represents cross-cluster traffic metrics.
type XCMetricsResponse struct {
	TotalRequests int64      `json:"totalRequests"`
	AvgLatencyMs  float64    `json:"avgLatencyMs"`
	ErrorRate     float64    `json:"errorRate"`
	Regions       []XCRegion `json:"regions"`
}

// XCRegion represents metrics for a single region.
type XCRegion struct {
	Name      string  `json:"name"`
	Requests  int64   `json:"requests"`
	LatencyMs float64 `json:"latencyMs"`
}

// XCCredentialsRequest represents a request to save XC credentials.
type XCCredentialsRequest struct {
	Tenant    string `json:"tenant"`
	APIToken  string `json:"apiToken"`
	Namespace string `json:"namespace"`
}

// XCCredentialsResponse represents XC credentials (token masked).
type XCCredentialsResponse struct {
	Tenant     string `json:"tenant"`
	Namespace  string `json:"namespace"`
	Configured bool   `json:"configured"`
}

// XCTestConnectionResponse represents a connection test result.
type XCTestConnectionResponse struct {
	Connected bool   `json:"connected"`
	Message   string `json:"message"`
}

// XCPreviewRequest represents a request to preview an XC publish configuration.
type XCPreviewRequest struct {
	Namespace      string `json:"namespace"`
	HTTPRouteRef   string `json:"httpRouteRef"`
	PublicHostname string `json:"publicHostname,omitempty"`
	OriginAddress  string `json:"originAddress,omitempty"`
	WAFEnabled     bool   `json:"wafEnabled,omitempty"`
	WAFPolicyName  string `json:"wafPolicyName,omitempty"`
}

// XCPreviewResponse represents the derived XC configuration for review.
type XCPreviewResponse struct {
	LoadBalancer *xc.HTTPLoadBalancer `json:"loadBalancer"`
	OriginPool   *xc.OriginPoolConfig `json:"originPool"`
	WAFPolicy    *string              `json:"wafPolicy,omitempty"`
}

// WAFPolicyResponse represents a WAF policy available in XC.
type WAFPolicyResponse struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Mode        string `json:"mode,omitempty"`
}

// XCHandler handles F5 Distributed Cloud API requests.
type XCHandler struct {
	Store database.Store
}

// getDynamicClient returns the dynamic client from the cluster context.
func (h *XCHandler) getDynamicClient(r *http.Request) dynamic.Interface {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		return nil
	}
	return k8s.DynamicClient()
}

// getXCClient creates an XC API client from stored credentials.
func (h *XCHandler) getXCClient(r *http.Request) (*xc.Client, error) {
	creds, err := h.Store.GetXCCredentials(r.Context())
	if err != nil {
		return nil, fmt.Errorf("reading XC credentials: %w", err)
	}
	if creds == nil {
		return nil, fmt.Errorf("XC credentials not configured")
	}
	return xc.New(creds.Tenant, creds.APIToken), nil
}

// --- Credential Management ---

// SaveCredentials stores XC connection credentials.
func (h *XCHandler) SaveCredentials(w http.ResponseWriter, r *http.Request) {
	var req XCCredentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.Tenant == "" || req.APIToken == "" {
		writeError(w, http.StatusBadRequest, "tenant and apiToken are required")
		return
	}

	if req.Namespace == "" {
		req.Namespace = "default"
	}

	creds := database.XCCredentials{
		Tenant:    req.Tenant,
		APIToken:  req.APIToken,
		Namespace: req.Namespace,
	}

	if err := h.Store.SaveXCCredentials(r.Context(), creds); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("saving credentials: %v", err))
		return
	}

	slog.Info("XC credentials saved", "tenant", req.Tenant, "namespace", req.Namespace)
	writeJSON(w, http.StatusOK, XCCredentialsResponse{
		Tenant:     req.Tenant,
		Namespace:  req.Namespace,
		Configured: true,
	})
}

// GetCredentials returns stored XC credentials (token masked).
func (h *XCHandler) GetCredentials(w http.ResponseWriter, r *http.Request) {
	creds, err := h.Store.GetXCCredentials(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("reading credentials: %v", err))
		return
	}

	if creds == nil {
		writeJSON(w, http.StatusOK, XCCredentialsResponse{Configured: false})
		return
	}

	writeJSON(w, http.StatusOK, XCCredentialsResponse{
		Tenant:     creds.Tenant,
		Namespace:  creds.Namespace,
		Configured: true,
	})
}

// DeleteCredentials removes stored XC credentials.
func (h *XCHandler) DeleteCredentials(w http.ResponseWriter, r *http.Request) {
	if err := h.Store.DeleteXCCredentials(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("deleting credentials: %v", err))
		return
	}

	slog.Info("XC credentials deleted")
	writeJSON(w, http.StatusOK, map[string]string{"message": "XC credentials deleted"})
}

// TestConnection verifies XC connectivity using stored credentials.
func (h *XCHandler) TestConnection(w http.ResponseWriter, r *http.Request) {
	client, err := h.getXCClient(r)
	if err != nil {
		writeJSON(w, http.StatusOK, XCTestConnectionResponse{
			Connected: false,
			Message:   err.Error(),
		})
		return
	}

	if err := client.TestConnection(r.Context()); err != nil {
		writeJSON(w, http.StatusOK, XCTestConnectionResponse{
			Connected: false,
			Message:   fmt.Sprintf("Connection failed: %v", err),
		})
		return
	}

	writeJSON(w, http.StatusOK, XCTestConnectionResponse{
		Connected: true,
		Message:   fmt.Sprintf("Successfully connected to tenant %q", client.Tenant()),
	})
}

// --- Status & Metrics ---

// Status returns the cross-cluster connectivity status.
func (h *XCHandler) Status(w http.ResponseWriter, r *http.Request) {
	resp := XCStatusResponse{}

	// Check XC API connectivity.
	creds, _ := h.Store.GetXCCredentials(r.Context())
	if creds != nil {
		resp.Tenant = creds.Tenant
		client := xc.New(creds.Tenant, creds.APIToken)
		if err := client.TestConnection(r.Context()); err == nil {
			resp.XCConnected = true
		}
	}

	// Check CRD publish count.
	dc := h.getDynamicClient(r)
	if dc != nil {
		list, err := dc.Resource(distributedCloudPublishGVR).Namespace("").List(r.Context(), metav1.ListOptions{})
		if err == nil {
			resp.Connected = true
			resp.PublishCount = len(list.Items)
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// Metrics returns cross-cluster traffic metrics (mock data for now).
func (h *XCHandler) Metrics(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, XCMetricsResponse{
		TotalRequests: 45230,
		AvgLatencyMs:  12.5,
		ErrorRate:     0.02,
		Regions: []XCRegion{
			{Name: "us-east-1", Requests: 15000, LatencyMs: 10.2},
			{Name: "eu-west-1", Requests: 20000, LatencyMs: 15.8},
			{Name: "ap-southeast-1", Requests: 10230, LatencyMs: 11.5},
		},
	})
}

// --- Publish CRUD ---

// Preview generates a preview of the XC HTTP LB config derived from an HTTPRoute.
func (h *XCHandler) Preview(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	var req XCPreviewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.HTTPRouteRef == "" {
		writeError(w, http.StatusBadRequest, "httpRouteRef is required")
		return
	}
	if req.Namespace == "" {
		req.Namespace = "default"
	}

	// Fetch the HTTPRoute.
	route, err := k8s.GetHTTPRoute(r.Context(), req.Namespace, req.HTTPRouteRef)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("HTTPRoute %s/%s not found: %v", req.Namespace, req.HTTPRouteRef, err))
		return
	}

	// Determine the Gateway's external address.
	gatewayAddress := "pending"
	if len(route.Spec.ParentRefs) > 0 {
		parentRef := route.Spec.ParentRefs[0]
		gwNs := req.Namespace
		if parentRef.Namespace != nil {
			gwNs = string(*parentRef.Namespace)
		}
		gw, gwErr := k8s.GetGateway(r.Context(), gwNs, string(parentRef.Name))
		if gwErr == nil {
			for _, addr := range gw.Status.Addresses {
				gatewayAddress = addr.Value
				break
			}
		}
	}

	// Get XC namespace from stored credentials.
	xcNamespace := "default"
	creds, _ := h.Store.GetXCCredentials(r.Context())
	if creds != nil {
		xcNamespace = creds.Namespace
	}

	// Build the preview using the mapper.
	opts := xc.MapOptions{
		XCNamespace:    xcNamespace,
		PublicHostname: req.PublicHostname,
		WAFEnabled:     req.WAFEnabled,
		WAFPolicyName:  req.WAFPolicyName,
		OriginPort:     80,
		OriginTLS:      false,
	}

	// Detect port and TLS from Gateway listeners.
	if len(route.Spec.ParentRefs) > 0 {
		parentRef := route.Spec.ParentRefs[0]
		gwNs := req.Namespace
		if parentRef.Namespace != nil {
			gwNs = string(*parentRef.Namespace)
		}
		gw, gwErr := k8s.GetGateway(r.Context(), gwNs, string(parentRef.Name))
		if gwErr == nil {
			for _, l := range gw.Spec.Listeners {
				if parentRef.SectionName != nil && string(*parentRef.SectionName) != string(l.Name) {
					continue
				}
				opts.OriginPort = int32(l.Port)
				if l.Protocol == "HTTPS" || l.Protocol == "TLS" {
					opts.OriginTLS = true
				}
				break
			}
		}
	}

	// Allow origin address override (e.g. when local hostname differs from public IP).
	originAddr := gatewayAddress
	if req.OriginAddress != "" {
		originAddr = req.OriginAddress
	}

	lb := xc.MapHTTPRouteToLoadBalancer(route, originAddr, opts)
	pool := xc.BuildOriginPool(req.HTTPRouteRef, originAddr, opts.OriginPort, opts.OriginTLS)

	preview := XCPreviewResponse{
		LoadBalancer: lb,
		OriginPool:   pool,
	}
	if req.WAFEnabled {
		policyName := req.WAFPolicyName
		if policyName == "" {
			policyName = "ngf-default-waf"
		}
		preview.WAFPolicy = &policyName
	}

	writeJSON(w, http.StatusOK, preview)
}

// Publish creates a new cross-cluster service publication.
// It creates the CRD in K8s and also calls the XC API to create the HTTP LB and origin pool.
func (h *XCHandler) Publish(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	dc := h.getDynamicClient(r)
	if dc == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	var req XCPublishRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.Name == "" || req.HTTPRouteRef == "" {
		writeError(w, http.StatusBadRequest, "name and httpRouteRef are required")
		return
	}

	if req.Namespace == "" {
		req.Namespace = "default"
	}

	// Build the distributedCloud spec for the CRD.
	if req.DistributedCloud == nil {
		req.DistributedCloud = map[string]interface{}{}
	}

	// Get XC credentials for tenant/namespace info.
	creds, _ := h.Store.GetXCCredentials(r.Context())
	if creds != nil {
		req.DistributedCloud["tenant"] = creds.Tenant
		req.DistributedCloud["namespace"] = creds.Namespace
	}
	if req.PublicHostname != "" {
		req.DistributedCloud["publicHostname"] = req.PublicHostname
	}
	if req.WAFEnabled {
		policyName := req.WAFPolicyName
		if policyName == "" {
			policyName = "ngf-default-waf"
		}
		req.DistributedCloud["wafPolicy"] = policyName
	}

	// Create the CRD object in K8s.
	obj := toXCPublishUnstructured(req)
	created, err := dc.Resource(distributedCloudPublishGVR).Namespace(req.Namespace).Create(r.Context(), obj, metav1.CreateOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("creating distributedcloudpublish: %v", err))
		return
	}

	resp := toXCPublishResponse(created)

	// Attempt to create XC resources via the XC API.
	if creds != nil {
		xcClient := xc.New(creds.Tenant, creds.APIToken)

		// Fetch the HTTPRoute to derive config.
		route, routeErr := k8s.GetHTTPRoute(r.Context(), req.Namespace, req.HTTPRouteRef)
		if routeErr != nil {
			slog.Warn("could not fetch HTTPRoute for XC publish", "error", routeErr)
		} else {
			// Determine gateway address.
			gatewayAddress := "pending"
			var originPort int32 = 80
			originTLS := false
			if len(route.Spec.ParentRefs) > 0 {
				parentRef := route.Spec.ParentRefs[0]
				gwNs := req.Namespace
				if parentRef.Namespace != nil {
					gwNs = string(*parentRef.Namespace)
				}
				gw, gwErr := k8s.GetGateway(r.Context(), gwNs, string(parentRef.Name))
				if gwErr == nil {
					for _, addr := range gw.Status.Addresses {
						gatewayAddress = addr.Value
						break
					}
					for _, l := range gw.Spec.Listeners {
						if parentRef.SectionName != nil && string(*parentRef.SectionName) != string(l.Name) {
							continue
						}
						originPort = int32(l.Port)
						if l.Protocol == "HTTPS" || l.Protocol == "TLS" {
							originTLS = true
						}
						break
					}
				}
			}

			xcNs := creds.Namespace
			opts := xc.MapOptions{
				XCNamespace:    xcNs,
				PublicHostname: req.PublicHostname,
				WAFEnabled:     req.WAFEnabled,
				WAFPolicyName:  req.WAFPolicyName,
				OriginPort:     originPort,
				OriginTLS:      originTLS,
			}

			// Allow origin address override (e.g. when local hostname differs from public IP).
			publishOriginAddr := gatewayAddress
			if req.OriginAddress != "" {
				publishOriginAddr = req.OriginAddress
			}

			// Create origin pool.
			pool := xc.BuildOriginPool(req.HTTPRouteRef, publishOriginAddr, originPort, originTLS)
			_, poolErr := xcClient.CreateOriginPool(r.Context(), xcNs, *pool)
			if poolErr != nil {
				slog.Warn("failed to create XC origin pool", "error", poolErr)
			} else {
				resp.XCOriginPoolName = pool.Metadata.Name
				slog.Info("created XC origin pool", "name", pool.Metadata.Name)
			}

			// Create HTTP LB.
			lb := xc.MapHTTPRouteToLoadBalancer(route, publishOriginAddr, opts)
			_, lbErr := xcClient.CreateHTTPLoadBalancer(r.Context(), xcNs, *lb)
			if lbErr != nil {
				slog.Warn("failed to create XC HTTP load balancer", "error", lbErr)
			} else {
				resp.XCLoadBalancerName = lb.Metadata.Name
				slog.Info("created XC HTTP load balancer", "name", lb.Metadata.Name)
			}

			if req.WAFEnabled {
				policyName := req.WAFPolicyName
				if policyName == "" {
					policyName = "ngf-default-waf"
				}
				resp.WAFPolicyAttached = policyName
			}
		}
	}

	writeJSON(w, http.StatusCreated, resp)
}

// ListPublishes returns all DistributedCloudPublish resources.
func (h *XCHandler) ListPublishes(w http.ResponseWriter, r *http.Request) {
	dc := h.getDynamicClient(r)
	if dc == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	list, err := dc.Resource(distributedCloudPublishGVR).Namespace("").List(r.Context(), metav1.ListOptions{})
	if err != nil {
		writeJSON(w, http.StatusOK, []XCPublishResponse{})
		return
	}

	resp := make([]XCPublishResponse, 0, len(list.Items))
	for i := range list.Items {
		resp = append(resp, toXCPublishResponse(&list.Items[i]))
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetPublish returns a specific publication by ID.
func (h *XCHandler) GetPublish(w http.ResponseWriter, r *http.Request) {
	dc := h.getDynamicClient(r)
	if dc == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	ns, name := parsePublishID(chi.URLParam(r, "id"))

	obj, err := dc.Resource(distributedCloudPublishGVR).Namespace(ns).Get(r.Context(), name, metav1.GetOptions{})
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("getting distributedcloudpublish %s/%s: %v", ns, name, err))
		return
	}

	writeJSON(w, http.StatusOK, toXCPublishResponse(obj))
}

// DeletePublish removes a cross-cluster service publication and its XC resources.
func (h *XCHandler) DeletePublish(w http.ResponseWriter, r *http.Request) {
	dc := h.getDynamicClient(r)
	if dc == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	ns, name := parsePublishID(chi.URLParam(r, "id"))

	// Attempt to clean up XC resources before deleting the CRD.
	obj, getErr := dc.Resource(distributedCloudPublishGVR).Namespace(ns).Get(r.Context(), name, metav1.GetOptions{})
	if getErr == nil {
		h.cleanupXCResources(r, obj)
	}

	err := dc.Resource(distributedCloudPublishGVR).Namespace(ns).Delete(r.Context(), name, metav1.DeleteOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("deleting distributedcloudpublish %s/%s: %v", ns, name, err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"message":   "distributedcloudpublish deleted",
		"name":      name,
		"namespace": ns,
	})
}

// cleanupXCResources deletes XC HTTP LB and origin pool resources for a publish.
func (h *XCHandler) cleanupXCResources(r *http.Request, obj *unstructured.Unstructured) {
	creds, err := h.Store.GetXCCredentials(r.Context())
	if err != nil || creds == nil {
		return
	}

	xcClient := xc.New(creds.Tenant, creds.APIToken)
	xcNs := creds.Namespace

	// Read the XC resource names from status.
	status, _, _ := unstructured.NestedMap(obj.Object, "status")
	if status != nil {
		lbName, _, _ := unstructured.NestedString(status, "xcLoadBalancerName")
		if lbName != "" {
			if err := xcClient.DeleteHTTPLoadBalancer(r.Context(), xcNs, lbName); err != nil {
				slog.Warn("failed to delete XC HTTP LB on cleanup", "name", lbName, "error", err)
			} else {
				slog.Info("deleted XC HTTP LB", "name", lbName)
			}
		}

		poolName, _, _ := unstructured.NestedString(status, "xcOriginPoolName")
		if poolName != "" {
			if err := xcClient.DeleteOriginPool(r.Context(), xcNs, poolName); err != nil {
				slog.Warn("failed to delete XC origin pool on cleanup", "name", poolName, "error", err)
			} else {
				slog.Info("deleted XC origin pool", "name", poolName)
			}
		}
	}

	// Fall back to name-based convention if status fields not set.
	spec, _, _ := unstructured.NestedMap(obj.Object, "spec")
	if spec != nil {
		httpRouteRef, _, _ := unstructured.NestedString(spec, "httpRouteRef")
		if httpRouteRef != "" {
			lbName := "ngf-" + httpRouteRef
			poolName := "ngf-" + httpRouteRef + "-pool"
			_ = xcClient.DeleteHTTPLoadBalancer(r.Context(), xcNs, lbName)
			_ = xcClient.DeleteOriginPool(r.Context(), xcNs, poolName)
		}
	}
}

// --- WAF ---

// ListWAFPolicies returns available WAF policies from the XC tenant.
func (h *XCHandler) ListWAFPolicies(w http.ResponseWriter, r *http.Request) {
	xcClient, err := h.getXCClient(r)
	if err != nil {
		writeJSON(w, http.StatusOK, []WAFPolicyResponse{})
		return
	}

	creds, _ := h.Store.GetXCCredentials(r.Context())
	xcNs := "default"
	if creds != nil {
		xcNs = creds.Namespace
	}

	firewalls, err := xcClient.ListAppFirewalls(r.Context(), xcNs)
	if err != nil {
		slog.Warn("failed to list XC WAF policies", "error", err)
		writeJSON(w, http.StatusOK, []WAFPolicyResponse{})
		return
	}

	resp := make([]WAFPolicyResponse, 0, len(firewalls))
	for _, fw := range firewalls {
		resp = append(resp, WAFPolicyResponse{
			Name:        fw.Name,
			Description: fw.Description,
			Mode:        fw.Mode,
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

// --- Helpers ---

// parsePublishID parses an ID string as "namespace/name" or just "name" (defaults to "default" namespace).
func parsePublishID(id string) (namespace, name string) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "default", id
}

// toXCPublishUnstructured converts a publish request into an unstructured object.
func toXCPublishUnstructured(req XCPublishRequest) *unstructured.Unstructured {
	spec := map[string]any{
		"httpRouteRef": req.HTTPRouteRef,
	}

	if req.InferencePoolRef != "" {
		spec["inferencePoolRef"] = req.InferencePoolRef
	}

	if req.DistributedCloud != nil {
		spec["distributedCloud"] = req.DistributedCloud
	}

	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": distributedCloudPublishGVR.Group + "/" + distributedCloudPublishGVR.Version,
			"kind":       "DistributedCloudPublish",
			"metadata": map[string]any{
				"name":      req.Name,
				"namespace": req.Namespace,
			},
			"spec": spec,
		},
	}

	return obj
}

// toXCPublishResponse converts an unstructured DistributedCloudPublish to a response type.
func toXCPublishResponse(obj *unstructured.Unstructured) XCPublishResponse {
	resp := XCPublishResponse{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
		CreatedAt: obj.GetCreationTimestamp().UTC().Format("2006-01-02T15:04:05Z"),
	}

	spec, _, _ := unstructured.NestedMap(obj.Object, "spec")
	if spec != nil {
		resp.HTTPRouteRef, _, _ = unstructured.NestedString(spec, "httpRouteRef")
		resp.InferencePoolRef, _, _ = unstructured.NestedString(spec, "inferencePoolRef")
	}

	status, _, _ := unstructured.NestedMap(obj.Object, "status")
	if status != nil {
		resp.Phase, _, _ = unstructured.NestedString(status, "phase")
		resp.XCLoadBalancerName, _, _ = unstructured.NestedString(status, "xcLoadBalancerName")
		resp.XCOriginPoolName, _, _ = unstructured.NestedString(status, "xcOriginPoolName")
		resp.XCVirtualIP, _, _ = unstructured.NestedString(status, "xcVirtualIP")
		resp.XCDNS, _, _ = unstructured.NestedString(status, "xcDNS")
		resp.WAFPolicyAttached, _, _ = unstructured.NestedString(status, "wafPolicyAttached")
		lastSynced, _, _ := unstructured.NestedString(status, "lastSyncedAt")
		resp.LastSyncedAt = lastSynced
	}

	return resp
}
