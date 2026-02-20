package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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
	Name               string                 `json:"name"`
	Namespace          string                 `json:"namespace"`
	HTTPRouteRef       string                 `json:"httpRouteRef"`
	InferencePoolRef   string                 `json:"inferencePoolRef,omitempty"`
	PublicHostname     string                 `json:"publicHostname,omitempty"`
	OriginAddress      string                 `json:"originAddress,omitempty"`
	WAFEnabled         bool                   `json:"wafEnabled,omitempty"`
	WAFPolicyName      string                 `json:"wafPolicyName,omitempty"`
	WAFPolicyNamespace string                 `json:"wafPolicyNamespace,omitempty"`
	WebSocketEnabled   bool                   `json:"webSocketEnabled,omitempty"`
	DistributedCloud   map[string]interface{} `json:"distributedCloud,omitempty"`
}

// XCPublishResponse represents a DistributedCloudPublish resource response.
type XCPublishResponse struct {
	Name               string   `json:"name"`
	Namespace          string   `json:"namespace"`
	HTTPRouteRef       string   `json:"httpRouteRef"`
	InferencePoolRef   string   `json:"inferencePoolRef,omitempty"`
	Phase              string   `json:"phase"`
	XCLoadBalancerName string   `json:"xcLoadBalancerName,omitempty"`
	XCOriginPoolName   string   `json:"xcOriginPoolName,omitempty"`
	XCVirtualIP        string   `json:"xcVirtualIP,omitempty"`
	XCDNS              string   `json:"xcDNS,omitempty"`
	WAFPolicyAttached  string   `json:"wafPolicyAttached,omitempty"`
	LastSyncedAt       string   `json:"lastSyncedAt,omitempty"`
	CreatedAt          string   `json:"createdAt"`
	Errors             []string `json:"errors,omitempty"`
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
	Namespace          string `json:"namespace"`
	HTTPRouteRef       string `json:"httpRouteRef"`
	PublicHostname     string `json:"publicHostname,omitempty"`
	OriginAddress      string `json:"originAddress,omitempty"`
	WAFEnabled         bool   `json:"wafEnabled,omitempty"`
	WAFPolicyName      string `json:"wafPolicyName,omitempty"`
	WAFPolicyNamespace string `json:"wafPolicyNamespace,omitempty"`
	WebSocketEnabled   bool   `json:"webSocketEnabled,omitempty"`
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
	Namespace   string `json:"namespace"`
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
	xcTenant := ""
	if creds != nil {
		xcTenant = creds.Tenant
	}

	opts := xc.MapOptions{
		XCNamespace:        xcNamespace,
		Tenant:             xcTenant,
		PublicHostname:     req.PublicHostname,
		WAFEnabled:         req.WAFEnabled,
		WAFPolicyName:      req.WAFPolicyName,
		WAFPolicyNamespace: req.WAFPolicyNamespace,
		WebSocketEnabled:   req.WebSocketEnabled,
		OriginPort:         80,
		OriginTLS:          false,
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
			policyName = "default"
		}
		wafDisplay := policyName
		if req.WAFPolicyNamespace != "" {
			wafDisplay = req.WAFPolicyNamespace + "/" + policyName
		}
		preview.WAFPolicy = &wafDisplay
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
			policyName = "default"
		}
		wafNs := req.WAFPolicyNamespace
		if wafNs != "" {
			req.DistributedCloud["wafPolicy"] = wafNs + "/" + policyName
		} else {
			req.DistributedCloud["wafPolicy"] = policyName
		}
	}

	// Create or update the CRD object in K8s.
	obj := toXCPublishUnstructured(req)
	created, err := dc.Resource(distributedCloudPublishGVR).Namespace(req.Namespace).Create(r.Context(), obj, metav1.CreateOptions{})
	if k8serrors.IsAlreadyExists(err) {
		// CRD already exists — fetch the existing one and re-publish XC resources.
		slog.Info("CRD already exists, re-publishing XC resources", "name", req.Name, "namespace", req.Namespace)
		existing, getErr := dc.Resource(distributedCloudPublishGVR).Namespace(req.Namespace).Get(r.Context(), req.Name, metav1.GetOptions{})
		if getErr != nil {
			writeError(w, http.StatusInternalServerError, fmt.Sprintf("fetching existing distributedcloudpublish: %v", getErr))
			return
		}
		created = existing
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("creating distributedcloudpublish: %v", err))
		return
	}

	resp := toXCPublishResponse(created)

	// Attempt to create XC resources via the XC API.
	var xcErrors []string
	if creds != nil {
		xcClient := xc.New(creds.Tenant, creds.APIToken)

		// Fetch the HTTPRoute to derive config.
		route, routeErr := k8s.GetHTTPRoute(r.Context(), req.Namespace, req.HTTPRouteRef)
		if routeErr != nil {
			xcErrors = append(xcErrors, fmt.Sprintf("Could not fetch HTTPRoute: %v", routeErr))
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
				XCNamespace:        xcNs,
				Tenant:             creds.Tenant,
				PublicHostname:     req.PublicHostname,
				WAFEnabled:         req.WAFEnabled,
				WAFPolicyName:      req.WAFPolicyName,
				WAFPolicyNamespace: req.WAFPolicyNamespace,
				WebSocketEnabled:   req.WebSocketEnabled,
				OriginPort:         originPort,
				OriginTLS:          originTLS,
			}

			// Allow origin address override (e.g. when local hostname differs from public IP).
			publishOriginAddr := gatewayAddress
			if req.OriginAddress != "" {
				publishOriginAddr = req.OriginAddress
			}

			// Create or replace origin pool.
			pool := xc.BuildOriginPool(req.HTTPRouteRef, publishOriginAddr, originPort, originTLS)
			_, poolErr := xcClient.CreateOriginPool(r.Context(), xcNs, *pool)
			if poolErr != nil {
				// Try replace if create failed (likely already exists).
				_, replaceErr := xcClient.ReplaceOriginPool(r.Context(), xcNs, *pool)
				if replaceErr != nil {
					xcErrors = append(xcErrors, fmt.Sprintf("Origin pool: %v (replace also failed: %v)", poolErr, replaceErr))
					slog.Warn("failed to create/replace XC origin pool", "createErr", poolErr, "replaceErr", replaceErr)
				} else {
					resp.XCOriginPoolName = pool.Metadata.Name
					slog.Info("replaced existing XC origin pool", "name", pool.Metadata.Name)
				}
			} else {
				resp.XCOriginPoolName = pool.Metadata.Name
				slog.Info("created XC origin pool", "name", pool.Metadata.Name)
			}

			// Create or replace HTTP LB.
			lb := xc.MapHTTPRouteToLoadBalancer(route, publishOriginAddr, opts)
			_, lbErr := xcClient.CreateHTTPLoadBalancer(r.Context(), xcNs, *lb)
			if lbErr != nil {
				// Try replace if create failed (likely already exists).
				_, replaceErr := xcClient.ReplaceHTTPLoadBalancer(r.Context(), xcNs, *lb)
				if replaceErr != nil {
					xcErrors = append(xcErrors, fmt.Sprintf("HTTP Load Balancer: %v (replace also failed: %v)", lbErr, replaceErr))
					slog.Warn("failed to create/replace XC HTTP load balancer", "createErr", lbErr, "replaceErr", replaceErr)
				} else {
					resp.XCLoadBalancerName = lb.Metadata.Name
					slog.Info("replaced existing XC HTTP load balancer", "name", lb.Metadata.Name)
				}
			} else {
				resp.XCLoadBalancerName = lb.Metadata.Name
				slog.Info("created XC HTTP load balancer", "name", lb.Metadata.Name)
			}

			// After creating/replacing the LB, fetch it back from XC to discover
			// the auto-generated CNAME (ves-io-*.ac.vh.ves.io) and add it to the
			// domains list so the LB responds on that hostname.
			if resp.XCLoadBalancerName != "" {
				h.addXCAutoDomain(r.Context(), xcClient, xcNs, lb)
			}

			if req.WAFEnabled {
				policyName := req.WAFPolicyName
				if policyName == "" {
					policyName = "default"
				}
				if req.WAFPolicyNamespace != "" {
					resp.WAFPolicyAttached = req.WAFPolicyNamespace + "/" + policyName
				} else {
					resp.WAFPolicyAttached = policyName
				}
			}
		}
	}

	// Update CRD status based on XC API results.
	phase := "Published"
	if len(xcErrors) > 0 {
		phase = "Error"
		resp.Errors = xcErrors
	}
	resp.Phase = phase
	resp.LastSyncedAt = time.Now().UTC().Format(time.RFC3339)

	statusPatch := map[string]any{
		"status": map[string]any{
			"phase":              phase,
			"xcLoadBalancerName": resp.XCLoadBalancerName,
			"xcOriginPoolName":   resp.XCOriginPoolName,
			"wafPolicyAttached":  resp.WAFPolicyAttached,
			"lastSyncedAt":       resp.LastSyncedAt,
		},
	}
	patchBytes, _ := json.Marshal(statusPatch)
	_, patchErr := dc.Resource(distributedCloudPublishGVR).Namespace(req.Namespace).Patch(
		r.Context(), req.Name, types.MergePatchType, patchBytes, metav1.PatchOptions{}, "status",
	)
	if patchErr != nil {
		slog.Warn("failed to patch CRD status", "error", patchErr)
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

// GetPublish returns a specific publication by namespace and name.
func (h *XCHandler) GetPublish(w http.ResponseWriter, r *http.Request) {
	dc := h.getDynamicClient(r)
	if dc == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	ns := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

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

	ns := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	// Attempt to clean up XC resources before deleting the CRD.
	var warnings []string
	obj, getErr := dc.Resource(distributedCloudPublishGVR).Namespace(ns).Get(r.Context(), name, metav1.GetOptions{})
	if getErr == nil {
		warnings = h.cleanupXCResources(r, obj)
	}

	err := dc.Resource(distributedCloudPublishGVR).Namespace(ns).Delete(r.Context(), name, metav1.DeleteOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("deleting distributedcloudpublish %s/%s: %v", ns, name, err))
		return
	}

	resp := map[string]any{
		"message":   "distributedcloudpublish deleted",
		"name":      name,
		"namespace": ns,
	}
	if len(warnings) > 0 {
		resp["warnings"] = warnings
	}
	writeJSON(w, http.StatusOK, resp)
}

// cleanupXCResources deletes XC HTTP LB and origin pool resources for a publish.
// Returns a list of warnings for any cleanup failures.
func (h *XCHandler) cleanupXCResources(r *http.Request, obj *unstructured.Unstructured) []string {
	var warnings []string

	creds, err := h.Store.GetXCCredentials(r.Context())
	if err != nil || creds == nil {
		return warnings
	}

	xcClient := xc.New(creds.Tenant, creds.APIToken)
	xcNs := creds.Namespace

	deletedLB := false
	deletedPool := false

	// Read the XC resource names from status.
	status, _, _ := unstructured.NestedMap(obj.Object, "status")
	if status != nil {
		lbName, _, _ := unstructured.NestedString(status, "xcLoadBalancerName")
		if lbName != "" {
			if err := xcClient.DeleteHTTPLoadBalancer(r.Context(), xcNs, lbName); err != nil {
				warnings = append(warnings, fmt.Sprintf("Failed to delete XC HTTP LB %q: %v", lbName, err))
				slog.Warn("failed to delete XC HTTP LB on cleanup", "name", lbName, "error", err)
			} else {
				deletedLB = true
				slog.Info("deleted XC HTTP LB", "name", lbName)
			}
		}

		poolName, _, _ := unstructured.NestedString(status, "xcOriginPoolName")
		if poolName != "" {
			if err := xcClient.DeleteOriginPool(r.Context(), xcNs, poolName); err != nil {
				warnings = append(warnings, fmt.Sprintf("Failed to delete XC origin pool %q: %v", poolName, err))
				slog.Warn("failed to delete XC origin pool on cleanup", "name", poolName, "error", err)
			} else {
				deletedPool = true
				slog.Info("deleted XC origin pool", "name", poolName)
			}
		}
	}

	// Fall back to name-based convention if status fields not set.
	if !deletedLB || !deletedPool {
		spec, _, _ := unstructured.NestedMap(obj.Object, "spec")
		if spec != nil {
			httpRouteRef, _, _ := unstructured.NestedString(spec, "httpRouteRef")
			if httpRouteRef != "" {
				if !deletedLB {
					lbName := "ngf-" + httpRouteRef
					if err := xcClient.DeleteHTTPLoadBalancer(r.Context(), xcNs, lbName); err != nil {
						slog.Warn("failed to delete XC HTTP LB by convention", "name", lbName, "error", err)
					}
				}
				if !deletedPool {
					poolName := "ngf-" + httpRouteRef + "-pool"
					if err := xcClient.DeleteOriginPool(r.Context(), xcNs, poolName); err != nil {
						slog.Warn("failed to delete XC origin pool by convention", "name", poolName, "error", err)
					}
				}
			}
		}
	}

	return warnings
}

// --- WAF ---

// ListWAFPolicies returns available WAF policies from the XC tenant.
// Queries both the user's namespace and the "shared" namespace.
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

	resp := make([]WAFPolicyResponse, 0)
	seen := make(map[string]bool)

	// List from shared namespace first (most WAF policies live here).
	sharedFW, err := xcClient.ListAppFirewalls(r.Context(), "shared")
	if err != nil {
		slog.Warn("failed to list shared XC WAF policies", "error", err)
	}
	for _, fw := range sharedFW {
		ns := fw.Namespace
		if ns == "" {
			ns = "shared"
		}
		key := ns + "/" + fw.Name
		if !seen[key] {
			seen[key] = true
			resp = append(resp, WAFPolicyResponse{
				Name:        fw.Name,
				Namespace:   ns,
				Description: fw.Description,
				Mode:        fw.Mode,
			})
		}
	}

	// Also list from user's namespace if different from shared.
	if xcNs != "shared" {
		userFW, err := xcClient.ListAppFirewalls(r.Context(), xcNs)
		if err != nil {
			slog.Warn("failed to list user XC WAF policies", "namespace", xcNs, "error", err)
		}
		for _, fw := range userFW {
			ns := fw.Namespace
			if ns == "" {
				ns = xcNs
			}
			key := ns + "/" + fw.Name
			if !seen[key] {
				seen[key] = true
				resp = append(resp, WAFPolicyResponse{
					Name:        fw.Name,
					Namespace:   ns,
					Description: fw.Description,
					Mode:        fw.Mode,
				})
			}
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// addXCAutoDomain fetches the LB from XC, looks for the auto-generated CNAME
// (e.g. ves-io-{uuid}.ac.vh.ves.io), and adds it to the LB's domains list
// so the LB responds on that hostname. This is a best-effort operation.
func (h *XCHandler) addXCAutoDomain(ctx context.Context, xcClient *xc.Client, xcNs string, lb *xc.HTTPLoadBalancer) {
	raw, err := xcClient.GetHTTPLoadBalancerRaw(ctx, xcNs, lb.Metadata.Name)
	if err != nil {
		slog.Warn("could not fetch LB to discover auto CNAME", "error", err)
		return
	}

	// The auto-generated CNAME is typically in spec.auto_cert_info.dns_records
	// or in the status/system_metadata. Search the raw JSON for ves.io hostnames.
	autoDomain := findVesDomain(raw)
	if autoDomain == "" {
		slog.Info("no auto-generated ves.io domain found in LB response")
		return
	}

	// Check if already in domains list.
	for _, d := range lb.Spec.Domains {
		if d == autoDomain {
			return
		}
	}

	// Add the auto domain and update the LB.
	lb.Spec.Domains = append(lb.Spec.Domains, autoDomain)
	_, err = xcClient.ReplaceHTTPLoadBalancer(ctx, xcNs, *lb)
	if err != nil {
		slog.Warn("could not update LB with auto CNAME domain", "domain", autoDomain, "error", err)
	} else {
		slog.Info("added XC auto CNAME to LB domains", "domain", autoDomain)
	}
}

// findVesDomain recursively searches a map for a string value containing ".vh.ves.io".
func findVesDomain(data map[string]any) string {
	for _, v := range data {
		switch val := v.(type) {
		case string:
			if strings.Contains(val, ".vh.ves.io") {
				return val
			}
		case map[string]any:
			if result := findVesDomain(val); result != "" {
				return result
			}
		case []any:
			for _, item := range val {
				if m, ok := item.(map[string]any); ok {
					if result := findVesDomain(m); result != "" {
						return result
					}
				}
				if s, ok := item.(string); ok {
					if strings.Contains(s, ".vh.ves.io") {
						return s
					}
				}
			}
		}
	}
	return ""
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
