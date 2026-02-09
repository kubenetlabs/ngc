package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
)

// GatewayHandler handles Gateway and GatewayClass API requests.
type GatewayHandler struct{}

// List returns all gateways, optionally filtered by ?namespace= query param.
func (h *GatewayHandler) List(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	ns := r.URL.Query().Get("namespace")
	gateways, err := k8s.ListGateways(r.Context(), ns)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := make([]GatewayResponse, 0, len(gateways))
	for i := range gateways {
		resp = append(resp, toGatewayResponse(&gateways[i]))
	}
	writeJSON(w, http.StatusOK, resp)
}

// Get returns a single gateway by namespace and name.
func (h *GatewayHandler) Get(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	ns := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	gw, err := k8s.GetGateway(r.Context(), ns, name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toGatewayResponse(gw))
}

// ListClasses returns all GatewayClasses.
func (h *GatewayHandler) ListClasses(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	classes, err := k8s.ListGatewayClasses(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := make([]GatewayClassResponse, 0, len(classes))
	for i := range classes {
		resp = append(resp, toGatewayClassResponse(&classes[i]))
	}
	writeJSON(w, http.StatusOK, resp)
}

// GetClass returns a single GatewayClass by name.
func (h *GatewayHandler) GetClass(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	name := chi.URLParam(r, "name")

	gc, err := k8s.GetGatewayClass(r.Context(), name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toGatewayClassResponse(gc))
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
