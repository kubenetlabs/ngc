package handlers

import "net/http"

// MigrationHandler handles NGINX config migration API requests.
type MigrationHandler struct{}

// Import imports an existing NGINX configuration.
func (h *MigrationHandler) Import(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// Analysis analyzes an imported configuration for migration compatibility.
func (h *MigrationHandler) Analysis(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// Generate produces Gateway API resources from the analyzed configuration.
func (h *MigrationHandler) Generate(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// Apply applies generated Gateway API resources to the cluster.
func (h *MigrationHandler) Apply(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// Validate validates migrated resources against the running gateway.
func (h *MigrationHandler) Validate(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}
