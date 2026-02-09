package handlers

import (
	"encoding/json"
	"net/http"
)

// GatewayHandler handles Gateway and GatewayClass API requests.
type GatewayHandler struct{}

// List returns all gateways or gateway classes.
func (h *GatewayHandler) List(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// Get returns a single gateway or gateway class by name.
func (h *GatewayHandler) Get(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// Create creates a new gateway.
func (h *GatewayHandler) Create(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// Update modifies an existing gateway.
func (h *GatewayHandler) Update(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// Delete removes a gateway.
func (h *GatewayHandler) Delete(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// Deploy triggers a gateway deployment.
func (h *GatewayHandler) Deploy(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// writeNotImplemented sends a 501 JSON response.
func writeNotImplemented(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{"error": "not implemented"})
}
