package handlers

import (
	"net/http"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
)

// ClusterHandler handles cluster management API requests.
type ClusterHandler struct {
	Manager *cluster.Manager
}

// ClusterResponse represents a cluster in the API response.
type ClusterResponse struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Connected   bool   `json:"connected"`
	Edition     string `json:"edition"`
	Default     bool   `json:"default"`
}

// List returns all registered clusters with their connection status.
func (h *ClusterHandler) List(w http.ResponseWriter, r *http.Request) {
	infos := h.Manager.List(r.Context())

	resp := make([]ClusterResponse, 0, len(infos))
	for _, info := range infos {
		resp = append(resp, ClusterResponse{
			Name:        info.Name,
			DisplayName: info.DisplayName,
			Connected:   info.Connected,
			Edition:     string(info.Edition),
			Default:     info.Default,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}
