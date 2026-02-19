package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
)

// TCPRouteHandler handles TCPRoute API requests.
type TCPRouteHandler struct{}

// List returns all TCPRoutes, optionally filtered by ?namespace= query param.
func (h *TCPRouteHandler) List(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	ns := r.URL.Query().Get("namespace")
	routes, err := k8s.ListTCPRoutes(r.Context(), ns)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := make([]TCPRouteResponse, 0, len(routes))
	for i := range routes {
		resp = append(resp, toTCPRouteResponse(&routes[i]))
	}
	writeJSON(w, http.StatusOK, resp)
}

// Get returns a single TCPRoute by namespace and name.
func (h *TCPRouteHandler) Get(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	ns := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	route, err := k8s.GetTCPRoute(r.Context(), ns, name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toTCPRouteResponse(route))
}

// Create creates a new TCPRoute.
func (h *TCPRouteHandler) Create(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	var req CreateTCPRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.Name == "" || req.Namespace == "" || len(req.ParentRefs) == 0 || len(req.Rules) == 0 {
		writeError(w, http.StatusBadRequest, "name, namespace, at least one parentRef, and at least one rule are required")
		return
	}

	route := toTCPRouteObject(req)
	created, err := k8s.CreateTCPRoute(r.Context(), route)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, toTCPRouteResponse(created))
}

// Update modifies an existing TCPRoute.
func (h *TCPRouteHandler) Update(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	ns := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	var req UpdateTCPRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	existing, err := k8s.GetTCPRoute(r.Context(), ns, name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	applyUpdateToTCPRoute(existing, req)

	updated, err := k8s.UpdateTCPRoute(r.Context(), existing)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toTCPRouteResponse(updated))
}

// Delete removes a TCPRoute.
func (h *TCPRouteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	ns := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	if err := k8s.DeleteTCPRoute(r.Context(), ns, name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "tcproute deleted", "name": name, "namespace": ns})
}
