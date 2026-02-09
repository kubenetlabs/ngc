package handlers

import (
	"net/http"

	"github.com/kubenetlabs/ngc/api/internal/kubernetes"
	"github.com/kubenetlabs/ngc/api/pkg/version"
)

// ConfigHandler serves application configuration info.
type ConfigHandler struct {
	KubeClient *kubernetes.Client
}

type configResponse struct {
	Edition   string `json:"edition"`
	Version   string `json:"version"`
	Connected bool   `json:"connected"`
}

// GetConfig returns edition, version, and cluster connection status.
func (h *ConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	resp := configResponse{
		Edition:   "unknown",
		Version:   version.Version,
		Connected: false,
	}

	if h.KubeClient != nil {
		resp.Connected = true
		resp.Edition = string(h.KubeClient.DetectEdition(r.Context()))
	}

	writeJSON(w, http.StatusOK, resp)
}
