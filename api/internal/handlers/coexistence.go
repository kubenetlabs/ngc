package handlers

import "net/http"

// CoexistenceHandler handles NGINX Plus / NGF coexistence API requests.
type CoexistenceHandler struct{}

// Overview returns the coexistence status overview.
func (h *CoexistenceHandler) Overview(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// MigrationReadiness returns the readiness assessment for migration.
func (h *CoexistenceHandler) MigrationReadiness(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}
