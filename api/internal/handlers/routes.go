package handlers

import "net/http"

// RouteHandler handles HTTPRoute, GRPCRoute, TLSRoute, TCPRoute, and UDPRoute API requests.
type RouteHandler struct{}

// List returns all routes of the given type.
func (h *RouteHandler) List(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// Get returns a single route by name.
func (h *RouteHandler) Get(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// Create creates a new route.
func (h *RouteHandler) Create(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// Update modifies an existing route.
func (h *RouteHandler) Update(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// Delete removes a route.
func (h *RouteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}

// Simulate performs a dry-run simulation of route matching.
func (h *RouteHandler) Simulate(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}
