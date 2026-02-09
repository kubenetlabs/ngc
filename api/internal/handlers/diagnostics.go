package handlers

import "net/http"

// DiagnosticsHandler handles diagnostic API requests.
type DiagnosticsHandler struct{}

// RouteCheck validates route configuration and detects issues.
func (h *DiagnosticsHandler) RouteCheck(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// Trace performs a request trace through the gateway routing pipeline.
func (h *DiagnosticsHandler) Trace(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}
