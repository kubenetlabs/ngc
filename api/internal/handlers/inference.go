package handlers

import "net/http"

// InferenceHandler handles Gateway API inference extension endpoints
// (InferencePools, EPP, autoscaling).
type InferenceHandler struct{}

// ListPools returns all inference pools.
func (h *InferenceHandler) ListPools(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// GetPool returns a single inference pool by name.
func (h *InferenceHandler) GetPool(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
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
