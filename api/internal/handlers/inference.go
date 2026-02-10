package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kubenetlabs/ngc/api/internal/inference"
)

// InferenceHandler handles Gateway API inference extension endpoints
// (InferencePools, EPP, autoscaling).
type InferenceHandler struct {
	Provider inference.MetricsProvider
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

// CreatePool creates a new inference pool.
func (h *InferenceHandler) CreatePool(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// UpdatePool modifies an existing inference pool.
func (h *InferenceHandler) UpdatePool(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// DeletePool removes an inference pool.
func (h *InferenceHandler) DeletePool(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// DeployPool triggers deployment of an inference pool.
func (h *InferenceHandler) DeployPool(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// GetEPP returns Endpoint Picker configuration and status.
func (h *InferenceHandler) GetEPP(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// UpdateEPP updates the Endpoint Picker configuration.
func (h *InferenceHandler) UpdateEPP(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// GetAutoscaling returns autoscaling configuration.
func (h *InferenceHandler) GetAutoscaling(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// UpdateAutoscaling updates autoscaling configuration.
func (h *InferenceHandler) UpdateAutoscaling(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
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
