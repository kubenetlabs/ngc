package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
)

var distributedCloudPublishGVR = schema.GroupVersionResource{
	Group:    "ngf-console.f5.com",
	Version:  "v1alpha1",
	Resource: "distributedcloudpublishes",
}

// XC request/response types

// XCStatusResponse represents XC connectivity status.
type XCStatusResponse struct {
	Connected    bool `json:"connected"`
	PublishCount int  `json:"publishCount"`
}

// XCPublishRequest represents a request to create a DistributedCloudPublish resource.
type XCPublishRequest struct {
	Name             string                 `json:"name"`
	Namespace        string                 `json:"namespace"`
	HTTPRouteRef     string                 `json:"httpRouteRef"`
	InferencePoolRef string                 `json:"inferencePoolRef,omitempty"`
	DistributedCloud map[string]interface{} `json:"distributedCloud"`
}

// XCPublishResponse represents a DistributedCloudPublish resource response.
type XCPublishResponse struct {
	Name             string `json:"name"`
	Namespace        string `json:"namespace"`
	HTTPRouteRef     string `json:"httpRouteRef"`
	InferencePoolRef string `json:"inferencePoolRef,omitempty"`
	Phase            string `json:"phase"`
	CreatedAt        string `json:"createdAt"`
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

// XCHandler handles cross-cluster (XC) API requests.
type XCHandler struct{}

// getDynamicClient returns the dynamic client from the cluster context.
func (h *XCHandler) getDynamicClient(r *http.Request) dynamic.Interface {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		return nil
	}
	return k8s.DynamicClient()
}

// Status returns the cross-cluster connectivity status.
func (h *XCHandler) Status(w http.ResponseWriter, r *http.Request) {
	dc := h.getDynamicClient(r)
	if dc == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	list, err := dc.Resource(distributedCloudPublishGVR).Namespace("").List(r.Context(), metav1.ListOptions{})
	if err != nil {
		// If the CRD doesn't exist or listing fails, report as not connected.
		writeJSON(w, http.StatusOK, XCStatusResponse{
			Connected:    false,
			PublishCount: 0,
		})
		return
	}

	writeJSON(w, http.StatusOK, XCStatusResponse{
		Connected:    true,
		PublishCount: len(list.Items),
	})
}

// Publish creates a new cross-cluster service publication.
func (h *XCHandler) Publish(w http.ResponseWriter, r *http.Request) {
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

	obj := toXCPublishUnstructured(req)
	created, err := dc.Resource(distributedCloudPublishGVR).Namespace(req.Namespace).Create(r.Context(), obj, metav1.CreateOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("creating distributedcloudpublish: %v", err))
		return
	}

	writeJSON(w, http.StatusCreated, toXCPublishResponse(created))
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

// DeletePublish removes a cross-cluster service publication.
func (h *XCHandler) DeletePublish(w http.ResponseWriter, r *http.Request) {
	dc := h.getDynamicClient(r)
	if dc == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	ns, name := parsePublishID(chi.URLParam(r, "id"))

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

// Metrics returns cross-cluster traffic metrics (mock data).
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
	}

	return resp
}
