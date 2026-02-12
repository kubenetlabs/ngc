package handlers

import (
	"net/http"
	"time"

	prom "github.com/kubenetlabs/ngc/api/internal/prometheus"
)

// MetricsHandler handles metrics API requests using Prometheus.
type MetricsHandler struct {
	Prom *prom.Client
}

// Summary returns an aggregated RED metrics summary.
func (h *MetricsHandler) Summary(w http.ResponseWriter, r *http.Request) {
	if h.Prom == nil {
		writeError(w, http.StatusServiceUnavailable, "prometheus not configured")
		return
	}

	summary, err := h.Prom.Summary(r.Context(), time.Now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

// ByRoute returns metrics grouped by HTTPRoute.
func (h *MetricsHandler) ByRoute(w http.ResponseWriter, r *http.Request) {
	if h.Prom == nil {
		writeError(w, http.StatusServiceUnavailable, "prometheus not configured")
		return
	}

	routes, err := h.Prom.ByRoute(r.Context(), time.Now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if routes == nil {
		routes = []prom.RouteMetrics{}
	}
	writeJSON(w, http.StatusOK, routes)
}

// ByGateway returns metrics grouped by gateway.
func (h *MetricsHandler) ByGateway(w http.ResponseWriter, r *http.Request) {
	if h.Prom == nil {
		writeError(w, http.StatusServiceUnavailable, "prometheus not configured")
		return
	}

	gateways, err := h.Prom.ByGateway(r.Context(), time.Now())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if gateways == nil {
		gateways = []prom.GatewayMetrics{}
	}
	writeJSON(w, http.StatusOK, gateways)
}
