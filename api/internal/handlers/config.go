package handlers

import (
	"net/http"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
	"github.com/kubenetlabs/ngc/api/pkg/version"
)

// ConfigHandler serves application configuration info.
type ConfigHandler struct{}

type configResponse struct {
	Edition   string `json:"edition"`
	Version   string `json:"version"`
	Connected bool   `json:"connected"`
	Cluster   string `json:"cluster,omitempty"`
}

// GetConfig returns edition, version, and cluster connection status.
func (h *ConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	resp := configResponse{
		Edition: "unknown",
		Version: version.Version,
		Cluster: cluster.ClusterNameFromContext(r.Context()),
	}

	k8s := cluster.ClientFromContext(r.Context())
	if k8s != nil {
		resp.Connected = true
		resp.Edition = string(k8s.DetectEdition(r.Context()))
	}

	writeJSON(w, http.StatusOK, resp)
}
