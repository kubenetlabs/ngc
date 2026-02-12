package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
	"github.com/kubenetlabs/ngc/api/internal/inference"
)

// InferenceHandler handles Gateway API inference extension endpoints
// (InferencePools, EPP, autoscaling).
// Pool CRUD operations are routed through InferenceStack CRDs.
type InferenceHandler struct {
	Provider      inference.MetricsProvider
	DynamicClient dynamic.Interface
}

// getDynamicClient returns the dynamic client from the handler field or falls back
// to the cluster context's dynamic client.
func (h *InferenceHandler) getDynamicClient(r *http.Request) dynamic.Interface {
	if h.DynamicClient != nil {
		return h.DynamicClient
	}
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		return nil
	}
	return k8s.DynamicClient()
}

// findInferenceStackByName searches all namespaces for an InferenceStack with the given name.
func (h *InferenceHandler) findInferenceStackByName(r *http.Request, name string) (*unstructured.Unstructured, error) {
	dc := h.getDynamicClient(r)
	if dc == nil {
		return nil, fmt.Errorf("no cluster context")
	}

	list, err := dc.Resource(inferenceStackGVR).Namespace("").List(r.Context(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing inferencestacks: %w", err)
	}

	for i := range list.Items {
		if list.Items[i].GetName() == name {
			return &list.Items[i], nil
		}
	}
	return nil, fmt.Errorf("inferencestack %q not found", name)
}

// ListPools returns all inference pools.
func (h *InferenceHandler) ListPools(w http.ResponseWriter, r *http.Request) {
	pools, err := h.Provider.ListPools(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := make([]InferencePoolResponse, 0, len(pools))
	for _, p := range pools {
		resp = append(resp, toInferencePoolResponse(p))
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetPool returns a single inference pool by name.
func (h *InferenceHandler) GetPool(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	pool, err := h.Provider.GetPool(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toInferencePoolResponse(*pool))
}

// CreatePoolRequest is the request body for creating a pool via the pool-oriented API.
type CreatePoolRequest struct {
	Name           string            `json:"name"`
	Namespace      string            `json:"namespace"`
	ModelName      string            `json:"modelName"`
	ModelVersion   string            `json:"modelVersion,omitempty"`
	ServingBackend string            `json:"servingBackend"`
	GPUType        string            `json:"gpuType"`
	GPUCount       int               `json:"gpuCount"`
	Replicas       int               `json:"replicas"`
	MinReplicas    int               `json:"minReplicas,omitempty"`
	MaxReplicas    int               `json:"maxReplicas,omitempty"`
	Selector       map[string]string `json:"selector,omitempty"`
	EPP            *CreateInferenceStackEPPReq `json:"epp,omitempty"`
}

// CreatePool creates a new inference pool by creating an InferenceStack CRD.
func (h *InferenceHandler) CreatePool(w http.ResponseWriter, r *http.Request) {
	dc := h.getDynamicClient(r)
	if dc == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	var req CreatePoolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.Name == "" || req.Namespace == "" || req.ModelName == "" || req.ServingBackend == "" {
		writeError(w, http.StatusBadRequest, "name, namespace, modelName, and servingBackend are required")
		return
	}

	// Convert pool request to InferenceStack request.
	stackReq := CreateInferenceStackRequest{
		Name:           req.Name,
		Namespace:      req.Namespace,
		ModelName:      req.ModelName,
		ModelVersion:   req.ModelVersion,
		ServingBackend: req.ServingBackend,
		Pool: CreateInferenceStackPoolReq{
			GPUType:     req.GPUType,
			GPUCount:    req.GPUCount,
			Replicas:    req.Replicas,
			MinReplicas: req.MinReplicas,
			MaxReplicas: req.MaxReplicas,
			Selector:    req.Selector,
		},
		EPP: req.EPP,
	}

	obj := toInferenceStackUnstructured(stackReq)
	created, err := dc.Resource(inferenceStackGVR).Namespace(req.Namespace).Create(r.Context(), obj, metav1.CreateOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("creating inferencestack: %v", err))
		return
	}
	writeJSON(w, http.StatusCreated, toInferenceStackResponse(created))
}

// UpdatePoolRequest is the request body for updating a pool.
type UpdatePoolRequest struct {
	ModelName      string            `json:"modelName,omitempty"`
	ModelVersion   string            `json:"modelVersion,omitempty"`
	ServingBackend string            `json:"servingBackend,omitempty"`
	GPUType        string            `json:"gpuType,omitempty"`
	GPUCount       *int              `json:"gpuCount,omitempty"`
	Replicas       *int              `json:"replicas,omitempty"`
	MinReplicas    *int              `json:"minReplicas,omitempty"`
	MaxReplicas    *int              `json:"maxReplicas,omitempty"`
	Selector       map[string]string `json:"selector,omitempty"`
	EPP            *CreateInferenceStackEPPReq `json:"epp,omitempty"`
}

// UpdatePool modifies an existing inference pool by updating its InferenceStack CRD.
func (h *InferenceHandler) UpdatePool(w http.ResponseWriter, r *http.Request) {
	dc := h.getDynamicClient(r)
	if dc == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	name := chi.URLParam(r, "name")

	existing, err := h.findInferenceStackByName(r, name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	var req UpdatePoolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	// Read current spec and apply updates.
	spec, _, _ := unstructured.NestedMap(existing.Object, "spec")
	if spec == nil {
		spec = map[string]any{}
	}

	if req.ModelName != "" {
		spec["modelName"] = req.ModelName
	}
	if req.ModelVersion != "" {
		spec["modelVersion"] = req.ModelVersion
	}
	if req.ServingBackend != "" {
		spec["servingBackend"] = req.ServingBackend
	}

	pool, _, _ := unstructured.NestedMap(spec, "pool")
	if pool == nil {
		pool = map[string]any{}
	}
	if req.GPUType != "" {
		pool["gpuType"] = req.GPUType
	}
	if req.GPUCount != nil {
		pool["gpuCount"] = int64(*req.GPUCount)
	}
	if req.Replicas != nil {
		pool["replicas"] = int64(*req.Replicas)
	}
	if req.MinReplicas != nil {
		pool["minReplicas"] = int64(*req.MinReplicas)
	}
	if req.MaxReplicas != nil {
		pool["maxReplicas"] = int64(*req.MaxReplicas)
	}
	if req.Selector != nil {
		pool["selector"] = req.Selector
	}
	spec["pool"] = pool

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

	existing.Object["spec"] = spec

	result, err := dc.Resource(inferenceStackGVR).Namespace(existing.GetNamespace()).Update(r.Context(), existing, metav1.UpdateOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("updating inferencestack %s: %v", name, err))
		return
	}
	writeJSON(w, http.StatusOK, toInferenceStackResponse(result))
}

// DeletePool removes an inference pool by deleting its InferenceStack CRD.
func (h *InferenceHandler) DeletePool(w http.ResponseWriter, r *http.Request) {
	dc := h.getDynamicClient(r)
	if dc == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	name := chi.URLParam(r, "name")

	existing, err := h.findInferenceStackByName(r, name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	ns := existing.GetNamespace()
	err = dc.Resource(inferenceStackGVR).Namespace(ns).Delete(r.Context(), name, metav1.DeleteOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("deleting inferencestack %s/%s: %v", ns, name, err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "inference pool deleted", "name": name, "namespace": ns})
}

// DeployPool triggers reconciliation of an inference pool by annotating its InferenceStack.
func (h *InferenceHandler) DeployPool(w http.ResponseWriter, r *http.Request) {
	dc := h.getDynamicClient(r)
	if dc == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	name := chi.URLParam(r, "name")

	existing, err := h.findInferenceStackByName(r, name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	// Set a reconcile annotation to trigger the operator.
	annotations := existing.GetAnnotations()
	if annotations == nil {
		annotations = map[string]string{}
	}
	annotations["ngf-console.f5.com/reconcile-requested"] = time.Now().UTC().Format(time.RFC3339)
	existing.SetAnnotations(annotations)

	result, err := dc.Resource(inferenceStackGVR).Namespace(existing.GetNamespace()).Update(r.Context(), existing, metav1.UpdateOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("deploying inferencestack %s: %v", name, err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"message": "reconciliation triggered",
		"name":    result.GetName(),
		"namespace": result.GetNamespace(),
	})
}

// EPPConfigResponse represents the EPP configuration for a pool.
type EPPConfigResponse struct {
	Pool     string                      `json:"pool"`
	Strategy string                      `json:"strategy"`
	Weights  *InferenceStackWeightsResp  `json:"weights,omitempty"`
}

// GetEPP returns Endpoint Picker configuration for a pool.
func (h *InferenceHandler) GetEPP(w http.ResponseWriter, r *http.Request) {
	poolName := r.URL.Query().Get("pool")
	if poolName == "" {
		writeError(w, http.StatusBadRequest, "pool query parameter is required")
		return
	}

	existing, err := h.findInferenceStackByName(r, poolName)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	resp := EPPConfigResponse{Pool: poolName}

	epp, _, _ := unstructured.NestedMap(existing.Object, "spec", "epp")
	if epp != nil {
		resp.Strategy, _, _ = unstructured.NestedString(epp, "strategy")
		weights, _, _ := unstructured.NestedMap(epp, "weights")
		if weights != nil {
			w := &InferenceStackWeightsResp{}
			qd, _, _ := unstructured.NestedInt64(weights, "queueDepth")
			w.QueueDepth = int(qd)
			kv, _, _ := unstructured.NestedInt64(weights, "kvCache")
			w.KVCache = int(kv)
			pa, _, _ := unstructured.NestedInt64(weights, "prefixAffinity")
			w.PrefixAffinity = int(pa)
			resp.Weights = w
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// UpdateEPPRequest is the request body for updating EPP configuration.
type UpdateEPPRequest struct {
	Pool     string                      `json:"pool"`
	Strategy string                      `json:"strategy"`
	Weights  *InferenceStackWeightsResp  `json:"weights,omitempty"`
}

// UpdateEPP updates the Endpoint Picker configuration for a pool's InferenceStack.
func (h *InferenceHandler) UpdateEPP(w http.ResponseWriter, r *http.Request) {
	dc := h.getDynamicClient(r)
	if dc == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	var req UpdateEPPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.Pool == "" || req.Strategy == "" {
		writeError(w, http.StatusBadRequest, "pool and strategy are required")
		return
	}

	existing, err := h.findInferenceStackByName(r, req.Pool)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	eppMap := map[string]any{
		"strategy": req.Strategy,
	}
	if req.Weights != nil {
		eppMap["weights"] = map[string]any{
			"queueDepth":     int64(req.Weights.QueueDepth),
			"kvCache":        int64(req.Weights.KVCache),
			"prefixAffinity": int64(req.Weights.PrefixAffinity),
		}
	}

	if err := unstructured.SetNestedField(existing.Object, eppMap, "spec", "epp"); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("setting epp: %v", err))
		return
	}

	result, err := dc.Resource(inferenceStackGVR).Namespace(existing.GetNamespace()).Update(r.Context(), existing, metav1.UpdateOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("updating epp for %s: %v", req.Pool, err))
		return
	}

	resp := toInferenceStackResponse(result)
	eppResp := EPPConfigResponse{Pool: req.Pool}
	if resp.EPP != nil {
		eppResp.Strategy = resp.EPP.Strategy
		eppResp.Weights = resp.EPP.Weights
	}
	writeJSON(w, http.StatusOK, eppResp)
}

// AutoscalingConfigResponse represents the autoscaling configuration for a pool.
type AutoscalingConfigResponse struct {
	Pool        string `json:"pool"`
	MinReplicas int    `json:"minReplicas"`
	MaxReplicas int    `json:"maxReplicas"`
	Replicas    int    `json:"replicas"`
}

// GetAutoscaling returns autoscaling configuration for a pool.
func (h *InferenceHandler) GetAutoscaling(w http.ResponseWriter, r *http.Request) {
	poolName := r.URL.Query().Get("pool")
	if poolName == "" {
		writeError(w, http.StatusBadRequest, "pool query parameter is required")
		return
	}

	existing, err := h.findInferenceStackByName(r, poolName)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	resp := AutoscalingConfigResponse{Pool: poolName}

	pool, _, _ := unstructured.NestedMap(existing.Object, "spec", "pool")
	if pool != nil {
		replicas, _, _ := unstructured.NestedInt64(pool, "replicas")
		resp.Replicas = int(replicas)
		minReplicas, _, _ := unstructured.NestedInt64(pool, "minReplicas")
		resp.MinReplicas = int(minReplicas)
		maxReplicas, _, _ := unstructured.NestedInt64(pool, "maxReplicas")
		resp.MaxReplicas = int(maxReplicas)
	}

	writeJSON(w, http.StatusOK, resp)
}

// UpdateAutoscalingRequest is the request body for updating autoscaling.
type UpdateAutoscalingRequest struct {
	Pool        string `json:"pool"`
	MinReplicas *int   `json:"minReplicas,omitempty"`
	MaxReplicas *int   `json:"maxReplicas,omitempty"`
	Replicas    *int   `json:"replicas,omitempty"`
}

// UpdateAutoscaling updates autoscaling configuration for a pool's InferenceStack.
func (h *InferenceHandler) UpdateAutoscaling(w http.ResponseWriter, r *http.Request) {
	dc := h.getDynamicClient(r)
	if dc == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	var req UpdateAutoscalingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.Pool == "" {
		writeError(w, http.StatusBadRequest, "pool is required")
		return
	}

	existing, err := h.findInferenceStackByName(r, req.Pool)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	pool, _, _ := unstructured.NestedMap(existing.Object, "spec", "pool")
	if pool == nil {
		pool = map[string]any{}
	}

	if req.Replicas != nil {
		pool["replicas"] = int64(*req.Replicas)
	}
	if req.MinReplicas != nil {
		pool["minReplicas"] = int64(*req.MinReplicas)
	}
	if req.MaxReplicas != nil {
		pool["maxReplicas"] = int64(*req.MaxReplicas)
	}

	if err := unstructured.SetNestedField(existing.Object, pool, "spec", "pool"); err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("setting pool: %v", err))
		return
	}

	result, err := dc.Resource(inferenceStackGVR).Namespace(existing.GetNamespace()).Update(r.Context(), existing, metav1.UpdateOptions{})
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("updating autoscaling for %s: %v", req.Pool, err))
		return
	}

	stackResp := toInferenceStackResponse(result)
	writeJSON(w, http.StatusOK, AutoscalingConfigResponse{
		Pool:        req.Pool,
		MinReplicas: stackResp.Pool.MinReplicas,
		MaxReplicas: stackResp.Pool.MaxReplicas,
		Replicas:    stackResp.Pool.Replicas,
	})
}

func toInferencePoolResponse(p inference.PoolStatus) InferencePoolResponse {
	resp := InferencePoolResponse{
		Name:           p.Name,
		Namespace:      p.Namespace,
		ModelName:      p.ModelName,
		ModelVersion:   p.ModelVersion,
		ServingBackend: p.ServingBackend,
		GPUType:        p.GPUType,
		GPUCount:       p.GPUCount,
		Replicas:       p.Replicas,
		MinReplicas:    p.MinReplicas,
		MaxReplicas:    p.MaxReplicas,
		Selector:       p.Selector,
		AvgGPUUtil:     p.AvgGPUUtil,
		CreatedAt:      formatTime(p.CreatedAt),
	}
	resp.Status = &InferencePoolStatusResponse{
		ReadyReplicas: p.ReadyReplicas,
		TotalReplicas: p.Replicas,
		Conditions: []ConditionResponse{
			{Type: "Ready", Status: p.Status, Reason: p.Status, Message: ""},
		},
	}
	return resp
}
