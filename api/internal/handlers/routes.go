package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kubenetlabs/ngc/api/internal/kubernetes"
)

// RouteHandler handles HTTPRoute, GRPCRoute, TLSRoute, TCPRoute, and UDPRoute API requests.
type RouteHandler struct {
	KubeClient *kubernetes.Client
}

// List returns all HTTPRoutes, optionally filtered by ?namespace= query param.
func (h *RouteHandler) List(w http.ResponseWriter, r *http.Request) {
	ns := r.URL.Query().Get("namespace")
	routes, err := h.KubeClient.ListHTTPRoutes(r.Context(), ns)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := make([]HTTPRouteResponse, 0, len(routes))
	for i := range routes {
		resp = append(resp, toHTTPRouteResponse(&routes[i]))
	}
	writeJSON(w, http.StatusOK, resp)
}

// Get returns a single HTTPRoute by namespace and name.
func (h *RouteHandler) Get(w http.ResponseWriter, r *http.Request) {
	ns := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	hr, err := h.KubeClient.GetHTTPRoute(r.Context(), ns, name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toHTTPRouteResponse(hr))
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
