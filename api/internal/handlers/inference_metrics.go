package handlers

import "net/http"

// InferenceMetricsHandler handles inference-specific metrics endpoints.
type InferenceMetricsHandler struct{}

// Summary returns an aggregated inference metrics summary.
func (h *InferenceMetricsHandler) Summary(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// ByPool returns inference metrics grouped by pool.
func (h *InferenceMetricsHandler) ByPool(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// PodMetrics returns per-pod inference metrics.
func (h *InferenceMetricsHandler) PodMetrics(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// Cost returns cost estimation for inference workloads.
func (h *InferenceMetricsHandler) Cost(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}
