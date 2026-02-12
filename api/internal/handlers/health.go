package handlers

import "net/http"

// HealthCheck returns a simple 200 OK for liveness/readiness probes.
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
