package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
)

// RouteHandler handles HTTPRoute, GRPCRoute, TLSRoute, TCPRoute, and UDPRoute API requests.
type RouteHandler struct{}

// List returns all HTTPRoutes, optionally filtered by ?namespace= query param.
func (h *RouteHandler) List(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	ns := r.URL.Query().Get("namespace")
	routes, err := k8s.ListHTTPRoutes(r.Context(), ns)
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
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	ns := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	hr, err := k8s.GetHTTPRoute(r.Context(), ns, name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toHTTPRouteResponse(hr))
}

// Create creates a new HTTPRoute.
func (h *RouteHandler) Create(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	var req CreateHTTPRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.Name == "" || req.Namespace == "" || len(req.ParentRefs) == 0 || len(req.Rules) == 0 {
		writeError(w, http.StatusBadRequest, "name, namespace, at least one parentRef, and at least one rule are required")
		return
	}

	hr := toHTTPRouteObject(req)
	created, err := k8s.CreateHTTPRoute(r.Context(), hr)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toHTTPRouteResponse(created))
}

// Update modifies an existing HTTPRoute.
func (h *RouteHandler) Update(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	ns := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	var req UpdateHTTPRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	existing, err := k8s.GetHTTPRoute(r.Context(), ns, name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	applyUpdateToHTTPRoute(existing, req)

	updated, err := k8s.UpdateHTTPRoute(r.Context(), existing)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toHTTPRouteResponse(updated))
}

// Delete removes an HTTPRoute.
func (h *RouteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	ns := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	if err := k8s.DeleteHTTPRoute(r.Context(), ns, name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "httproute deleted", "name": name, "namespace": ns})
}

// Simulate performs a dry-run simulation of route matching.
func (h *RouteHandler) Simulate(w http.ResponseWriter, r *http.Request) {
	writeNotImplemented(w)
}
