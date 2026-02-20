package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
	"github.com/kubenetlabs/ngc/api/internal/database"
	"github.com/kubenetlabs/ngc/api/internal/inference"
)

var inferenceStackGVR = schema.GroupVersionResource{
	Group:    "ngf-console.f5.com",
	Version:  "v1alpha1",
	Resource: "inferencestacks",
}

// InferenceStackHandler handles InferenceStack CRD API requests using the dynamic client.
type InferenceStackHandler struct {
	DynamicClient   dynamic.Interface
	MetricsProvider inference.MetricsProvider
	Store           database.Store
}

// getDynamicClient returns the dynamic client from the handler field or falls back
// to the cluster context's dynamic client.
func (h *InferenceStackHandler) getDynamicClient(r *http.Request) dynamic.Interface {
	if h.DynamicClient != nil {
		return h.DynamicClient
	}
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		return nil
	}
	return k8s.DynamicClient()
}

// List returns all InferenceStacks across all namespaces.
func (h *InferenceStackHandler) List(w http.ResponseWriter, r *http.Request) {
	dc := h.getDynamicClient(r)
	if dc == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	list, err := dc.Resource(inferenceStackGVR).Namespace("").List(r.Context(), metav1.ListOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("listing inferencestacks: %v", err))
		return
	}

	resp := make([]InferenceStackResponse, 0, len(list.Items))
	for i := range list.Items {
		resp = append(resp, toInferenceStackResponse(&list.Items[i]))
	}
	writeJSON(w, http.StatusOK, resp)
}

// Get returns a single InferenceStack by namespace and name.
func (h *InferenceStackHandler) Get(w http.ResponseWriter, r *http.Request) {
	dc := h.getDynamicClient(r)
	if dc == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	ns := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	obj, err := dc.Resource(inferenceStackGVR).Namespace(ns).Get(r.Context(), name, metav1.GetOptions{})
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("getting inferencestack %s/%s: %v", ns, name, err))
		return
	}
	writeJSON(w, http.StatusOK, toInferenceStackResponse(obj))
}

// Create creates a new InferenceStack from the JSON request body.
func (h *InferenceStackHandler) Create(w http.ResponseWriter, r *http.Request) {
	dc := h.getDynamicClient(r)
	if dc == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	var req CreateInferenceStackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.Name == "" || req.Namespace == "" || req.ModelName == "" || req.ServingBackend == "" {
		writeError(w, http.StatusBadRequest, "name, namespace, modelName, and servingBackend are required")
		return
	}

	obj := toInferenceStackUnstructured(req)
	created, err := dc.Resource(inferenceStackGVR).Namespace(req.Namespace).Create(r.Context(), obj, metav1.CreateOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("creating inferencestack: %v", err))
		return
	}

	// Sync pool metadata to ClickHouse so it appears in the pool list.
	if h.MetricsProvider != nil {
		_ = h.MetricsProvider.UpsertPool(r.Context(), inference.PoolStatus{
			Name:           req.Name,
			Namespace:      req.Namespace,
			ModelName:      req.ModelName,
			ModelVersion:   req.ModelVersion,
			ServingBackend: req.ServingBackend,
			GPUType:        req.Pool.GPUType,
			GPUCount:       uint32(req.Pool.GPUCount),
			Replicas:       uint32(req.Pool.Replicas),
			MinReplicas:    uint32(req.Pool.MinReplicas),
			MaxReplicas:    uint32(req.Pool.MaxReplicas),
			Status:         "Pending",
			CreatedAt:      time.Now(),
		})
	}

	resp := toInferenceStackResponse(created)
	auditLog(h.Store, r.Context(), "create", "InferenceStack", req.Name, req.Namespace, nil, resp)
	writeJSON(w, http.StatusCreated, resp)
}

// Update modifies an existing InferenceStack by namespace and name.
func (h *InferenceStackHandler) Update(w http.ResponseWriter, r *http.Request) {
	dc := h.getDynamicClient(r)
	if dc == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	ns := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	// Fetch existing to get resourceVersion.
	existing, err := dc.Resource(inferenceStackGVR).Namespace(ns).Get(r.Context(), name, metav1.GetOptions{})
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("getting inferencestack %s/%s: %v", ns, name, err))
		return
	}

	var req CreateInferenceStackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	// Build the updated object, preserving metadata from existing.
	beforeResp := toInferenceStackResponse(existing)
	updated := toInferenceStackUnstructured(req)
	updated.SetNamespace(ns)
	updated.SetName(name)
	updated.SetResourceVersion(existing.GetResourceVersion())
	updated.SetUID(existing.GetUID())
	updated.SetCreationTimestamp(existing.GetCreationTimestamp())

	result, err := dc.Resource(inferenceStackGVR).Namespace(ns).Update(r.Context(), updated, metav1.UpdateOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("updating inferencestack %s/%s: %v", ns, name, err))
		return
	}
	afterResp := toInferenceStackResponse(result)
	auditLog(h.Store, r.Context(), "update", "InferenceStack", name, ns, beforeResp, afterResp)
	writeJSON(w, http.StatusOK, afterResp)
}

// Delete removes an InferenceStack by namespace and name.
func (h *InferenceStackHandler) Delete(w http.ResponseWriter, r *http.Request) {
	dc := h.getDynamicClient(r)
	if dc == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	ns := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	err := dc.Resource(inferenceStackGVR).Namespace(ns).Delete(r.Context(), name, metav1.DeleteOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("deleting inferencestack %s/%s: %v", ns, name, err))
		return
	}

	// Remove pool metadata from ClickHouse.
	if h.MetricsProvider != nil {
		_ = h.MetricsProvider.DeletePool(r.Context(), name, ns)
	}

	auditLog(h.Store, r.Context(), "delete", "InferenceStack", name, ns, map[string]string{"name": name, "namespace": ns}, nil)
	writeJSON(w, http.StatusOK, map[string]string{"message": "inferencestack deleted", "name": name, "namespace": ns})
}

// GetStatus returns just the status sub-resource of an InferenceStack.
func (h *InferenceStackHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	dc := h.getDynamicClient(r)
	if dc == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	ns := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	obj, err := dc.Resource(inferenceStackGVR).Namespace(ns).Get(r.Context(), name, metav1.GetOptions{}, "status")
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("getting inferencestack status %s/%s: %v", ns, name, err))
		return
	}

	resp := toInferenceStackResponse(obj)
	// Return only the status-relevant fields.
	statusResp := map[string]any{
		"name":      resp.Name,
		"namespace": resp.Namespace,
		"phase":     resp.Phase,
		"children":  resp.Children,
	}
	if resp.ObservedSpecHash != "" {
		statusResp["observedSpecHash"] = resp.ObservedSpecHash
	}
	if resp.LastReconciledAt != "" {
		statusResp["lastReconciledAt"] = resp.LastReconciledAt
	}
	writeJSON(w, http.StatusOK, statusResp)
}

// toInferenceStackResponse converts an unstructured InferenceStack to a response type.
func toInferenceStackResponse(obj *unstructured.Unstructured) InferenceStackResponse {
	resp := InferenceStackResponse{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
		CreatedAt: obj.GetCreationTimestamp().UTC().Format("2006-01-02T15:04:05Z"),
	}

	spec, _, _ := unstructured.NestedMap(obj.Object, "spec")
	if spec != nil {
		resp.ModelName, _, _ = unstructured.NestedString(spec, "modelName")
		resp.ModelVersion, _, _ = unstructured.NestedString(spec, "modelVersion")
		resp.ServingBackend, _, _ = unstructured.NestedString(spec, "servingBackend")

		// Pool
		pool, _, _ := unstructured.NestedMap(spec, "pool")
		if pool != nil {
			resp.Pool.GPUType, _, _ = unstructured.NestedString(pool, "gpuType")
			gpuCount, _, _ := unstructured.NestedInt64(pool, "gpuCount")
			resp.Pool.GPUCount = int(gpuCount)
			replicas, _, _ := unstructured.NestedInt64(pool, "replicas")
			resp.Pool.Replicas = int(replicas)
			minReplicas, _, _ := unstructured.NestedInt64(pool, "minReplicas")
			resp.Pool.MinReplicas = int(minReplicas)
			maxReplicas, _, _ := unstructured.NestedInt64(pool, "maxReplicas")
			resp.Pool.MaxReplicas = int(maxReplicas)
			selector, _, _ := unstructured.NestedStringMap(pool, "selector")
			resp.Pool.Selector = selector
		}

		// EPP
		epp, _, _ := unstructured.NestedMap(spec, "epp")
		if epp != nil {
			eppResp := &InferenceStackEPPResponse{}
			eppResp.Strategy, _, _ = unstructured.NestedString(epp, "strategy")

			weights, _, _ := unstructured.NestedMap(epp, "weights")
			if weights != nil {
				w := &InferenceStackWeightsResp{}
				qd, _, _ := unstructured.NestedInt64(weights, "queueDepth")
				w.QueueDepth = int(qd)
				kv, _, _ := unstructured.NestedInt64(weights, "kvCache")
				w.KVCache = int(kv)
				pa, _, _ := unstructured.NestedInt64(weights, "prefixAffinity")
				w.PrefixAffinity = int(pa)
				eppResp.Weights = w
			}
			resp.EPP = eppResp
		}
	}

	// Status
	status, _, _ := unstructured.NestedMap(obj.Object, "status")
	if status != nil {
		resp.Phase, _, _ = unstructured.NestedString(status, "phase")
		resp.ObservedSpecHash, _, _ = unstructured.NestedString(status, "observedSpecHash")
		resp.LastReconciledAt, _, _ = unstructured.NestedString(status, "lastReconciledAt")

		children, _, _ := unstructured.NestedSlice(status, "children")
		for _, child := range children {
			childMap, ok := child.(map[string]any)
			if !ok {
				continue
			}
			cs := ChildStatusResponse{}
			cs.Kind, _, _ = unstructured.NestedString(childMap, "kind")
			cs.Name, _, _ = unstructured.NestedString(childMap, "name")
			cs.Ready, _, _ = unstructured.NestedBool(childMap, "ready")
			cs.Message, _, _ = unstructured.NestedString(childMap, "message")
			resp.Children = append(resp.Children, cs)
		}
	}

	return resp
}

// toInferenceStackUnstructured converts a create request into an unstructured object.
func toInferenceStackUnstructured(req CreateInferenceStackRequest) *unstructured.Unstructured {
	spec := map[string]any{
		"modelName":      req.ModelName,
		"servingBackend": req.ServingBackend,
		"pool": map[string]any{
			"gpuType":     req.Pool.GPUType,
			"gpuCount":    int64(req.Pool.GPUCount),
			"replicas":    int64(req.Pool.Replicas),
			"minReplicas": int64(req.Pool.MinReplicas),
			"maxReplicas": int64(req.Pool.MaxReplicas),
		},
	}

	if req.ModelVersion != "" {
		spec["modelVersion"] = req.ModelVersion
	}

	if req.Pool.Selector != nil {
		poolMap := spec["pool"].(map[string]any)
		poolMap["selector"] = req.Pool.Selector
	}

	if req.EPP != nil {
		eppMap := map[string]any{
			"strategy": req.EPP.Strategy,
		}
		if req.EPP.Weights != nil {
			eppMap["weights"] = map[string]any{
				"queueDepth":     int64(req.EPP.Weights.QueueDepth),
				"kvCache":        int64(req.EPP.Weights.KVCache),
				"prefixAffinity": int64(req.EPP.Weights.PrefixAffinity),
			}
		}
		spec["epp"] = eppMap
	}

	obj := &unstructured.Unstructured{
		Object: map[string]any{
			"apiVersion": inferenceStackGVR.Group + "/" + inferenceStackGVR.Version,
			"kind":       "InferenceStack",
			"metadata": map[string]any{
				"name":      req.Name,
				"namespace": req.Namespace,
			},
			"spec": spec,
		},
	}

	return obj
}
