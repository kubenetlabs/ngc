package handlers

import "net/http"

// MetricsHandler handles metrics API requests.
type MetricsHandler struct{}

// Summary returns an aggregated metrics summary.
func (h *MetricsHandler) Summary(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// ByRoute returns metrics grouped by route.
func (h *MetricsHandler) ByRoute(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// ByGateway returns metrics grouped by gateway.
func (h *MetricsHandler) ByGateway(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}
