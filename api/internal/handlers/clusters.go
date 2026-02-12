package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
	"github.com/kubenetlabs/ngc/api/internal/multicluster"
)

// validClusterName matches valid Kubernetes resource names (RFC 1123 DNS subdomain).
var validClusterName = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

// ClusterHandler handles cluster management API requests.
type ClusterHandler struct {
	Manager cluster.Provider
	Pool    *multicluster.ClientPool // non-nil only in CRD-based multi-cluster mode
}

// ClusterResponse represents a cluster in the API response.
type ClusterResponse struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Connected   bool   `json:"connected"`
	Edition     string `json:"edition"`
	Default     bool   `json:"default"`
}

// ClusterDetailResponse provides extended information about a single cluster.
type ClusterDetailResponse struct {
	Name              string                 `json:"name"`
	DisplayName       string                 `json:"displayName"`
	Region            string                 `json:"region"`
	Environment       string                 `json:"environment"`
	Connected         bool                   `json:"connected"`
	Edition           string                 `json:"edition"`
	Default           bool                   `json:"default"`
	KubernetesVersion string                 `json:"kubernetesVersion,omitempty"`
	NGFVersion        string                 `json:"ngfVersion,omitempty"`
	AgentInstalled    bool                   `json:"agentInstalled"`
	LastHeartbeat     *string                `json:"lastHeartbeat,omitempty"`
	ResourceCounts    *multicluster.ResourceCounts  `json:"resourceCounts,omitempty"`
	GPUCapacity       *multicluster.GPUCapacitySummary `json:"gpuCapacity,omitempty"`
	IsLocal           bool                   `json:"isLocal"`
}

// RegisterClusterRequest is the payload for registering a new cluster.
type RegisterClusterRequest struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Region      string `json:"region"`
	Environment string `json:"environment"`
	Kubeconfig  string `json:"kubeconfig"`
	NGFEdition  string `json:"ngfEdition,omitempty"`
}

// HeartbeatRequest is the payload sent by agents to report cluster health.
type HeartbeatRequest struct {
	KubernetesVersion string                          `json:"kubernetesVersion"`
	NGFVersion        string                          `json:"ngfVersion"`
	ResourceCounts    *multicluster.ResourceCounts     `json:"resourceCounts,omitempty"`
	GPUCapacity       *multicluster.GPUCapacitySummary `json:"gpuCapacity,omitempty"`
}

// ClusterSummaryResponse provides a global summary across all clusters.
type ClusterSummaryResponse struct {
	TotalClusters   int   `json:"totalClusters"`
	HealthyClusters int   `json:"healthyClusters"`
	TotalGateways   int32 `json:"totalGateways"`
	TotalRoutes     int32 `json:"totalRoutes"`
	TotalGPUs       int32 `json:"totalGPUs"`
}

// List returns all registered clusters with their connection status.
func (h *ClusterHandler) List(w http.ResponseWriter, r *http.Request) {
	if h.Pool != nil {
		h.listFromPool(w, r)
		return
	}

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

func (h *ClusterHandler) listFromPool(w http.ResponseWriter, r *http.Request) {
	clients := h.Pool.List()
	resp := make([]ClusterDetailResponse, 0, len(clients))

	for _, cc := range clients {
		detail := ClusterDetailResponse{
			Name:              cc.Name,
			DisplayName:       cc.DisplayName,
			Region:            cc.Region,
			Environment:       cc.Environment,
			Connected:         cc.Healthy,
			Default:           cc.Name == h.Manager.DefaultName(),
			KubernetesVersion: cc.K8sVersion,
			NGFVersion:        cc.NGFVersion,
			AgentInstalled:    cc.AgentInstalled,
			ResourceCounts:    cc.ResourceCounts,
			GPUCapacity:       cc.GPUCapacity,
			IsLocal:           cc.IsLocal,
		}
		if cc.K8sClient != nil {
			detail.Edition = string(cc.K8sClient.DetectEdition(r.Context()))
		}
		resp = append(resp, detail)
	}

	writeJSON(w, http.StatusOK, resp)
}

// Get returns detail for a single cluster.
func (h *ClusterHandler) Get(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "cluster")

	if h.Pool == nil {
		writeError(w, http.StatusNotImplemented, "cluster detail requires CRD-based multi-cluster mode")
		return
	}

	cc, err := h.Pool.Get(name)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("cluster %q not found", name))
		return
	}

	detail := ClusterDetailResponse{
		Name:              cc.Name,
		DisplayName:       cc.DisplayName,
		Region:            cc.Region,
		Environment:       cc.Environment,
		Connected:         cc.Healthy,
		Default:           cc.Name == h.Manager.DefaultName(),
		KubernetesVersion: cc.K8sVersion,
		NGFVersion:        cc.NGFVersion,
		AgentInstalled:    cc.AgentInstalled,
		ResourceCounts:    cc.ResourceCounts,
		GPUCapacity:       cc.GPUCapacity,
		IsLocal:           cc.IsLocal,
	}
	if cc.K8sClient != nil {
		detail.Edition = string(cc.K8sClient.DetectEdition(r.Context()))
	}

	writeJSON(w, http.StatusOK, detail)
}

// Register creates a new ManagedCluster CRD and kubeconfig Secret.
func (h *ClusterHandler) Register(w http.ResponseWriter, r *http.Request) {
	if h.Pool == nil {
		writeError(w, http.StatusNotImplemented, "cluster registration requires CRD-based multi-cluster mode")
		return
	}

	var req RegisterClusterRequest
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20)) // 1MB limit for kubeconfig
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "name and displayName are required")
		return
	}

	if !validClusterName.MatchString(req.Name) {
		writeError(w, http.StatusBadRequest, "name must be a valid DNS subdomain (lowercase alphanumeric and hyphens, 1-63 chars)")
		return
	}

	// Create kubeconfig Secret.
	if req.Kubeconfig != "" {
		secretObj := map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]interface{}{
				"name":      req.Name + "-kubeconfig",
				"namespace": h.Pool.Namespace(),
			},
			"type": "Opaque",
			"stringData": map[string]interface{}{
				"kubeconfig": req.Kubeconfig,
			},
		}
		if err := h.Pool.CreateRaw(r.Context(), "v1", "secrets", secretObj); err != nil {
			slog.Error("failed to create kubeconfig secret", "cluster", req.Name, "error", err)
			writeError(w, http.StatusInternalServerError, "failed to create kubeconfig secret")
			return
		}
	}

	// Create ManagedCluster CRD.
	mcObj := map[string]interface{}{
		"apiVersion": "ngf-console.f5.com/v1alpha1",
		"kind":       "ManagedCluster",
		"metadata": map[string]interface{}{
			"name":      req.Name,
			"namespace": h.Pool.Namespace(),
		},
		"spec": map[string]interface{}{
			"displayName": req.DisplayName,
			"region":      req.Region,
			"environment": req.Environment,
			"ngfEdition":  req.NGFEdition,
			"kubeconfigSecretRef": map[string]interface{}{
				"name": req.Name + "-kubeconfig",
			},
		},
	}
	if err := h.Pool.CreateManagedCluster(r.Context(), mcObj); err != nil {
		slog.Error("failed to create ManagedCluster", "cluster", req.Name, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to register cluster")
		return
	}

	// Trigger pool sync to pick up the new cluster.
	if err := h.Pool.Sync(r.Context()); err != nil {
		slog.Error("failed to sync pool after registration", "cluster", req.Name, "error", err)
		writeError(w, http.StatusInternalServerError, "cluster registered but sync failed")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"message": "cluster registered",
		"name":    req.Name,
	})
}

// Unregister removes a ManagedCluster CRD and its kubeconfig Secret.
func (h *ClusterHandler) Unregister(w http.ResponseWriter, r *http.Request) {
	if h.Pool == nil {
		writeError(w, http.StatusNotImplemented, "cluster management requires CRD-based multi-cluster mode")
		return
	}

	name := chi.URLParam(r, "cluster")

	if !validClusterName.MatchString(name) {
		writeError(w, http.StatusBadRequest, "invalid cluster name")
		return
	}

	if err := h.Pool.DeleteManagedCluster(r.Context(), name); err != nil {
		slog.Error("failed to delete ManagedCluster", "cluster", name, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to unregister cluster")
		return
	}

	// Also delete the kubeconfig secret.
	_ = h.Pool.DeleteRaw(r.Context(), "v1", "secrets", name+"-kubeconfig")

	// Re-sync pool to remove the client.
	_ = h.Pool.Sync(r.Context())

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "cluster unregistered",
		"name":    name,
	})
}

// TestConnection tests connectivity to a managed cluster.
func (h *ClusterHandler) TestConnection(w http.ResponseWriter, r *http.Request) {
	if h.Pool == nil {
		writeError(w, http.StatusNotImplemented, "cluster management requires CRD-based multi-cluster mode")
		return
	}

	name := chi.URLParam(r, "cluster")

	cc, err := h.Pool.Get(name)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"connected": false,
			"error":     fmt.Sprintf("cluster %q not found or circuit breaker open", name),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"connected":         cc.Healthy,
		"kubernetesVersion": cc.K8sVersion,
		"ngfVersion":        cc.NGFVersion,
	})
}

// InstallAgent generates the Helm install command for the agent chart.
func (h *ClusterHandler) InstallAgent(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "cluster")

	// Validate cluster name to prevent shell injection in generated command.
	if !validClusterName.MatchString(name) {
		writeError(w, http.StatusBadRequest, "invalid cluster name")
		return
	}

	hubAPIEndpoint := r.Host
	if hubAPIEndpoint == "" {
		hubAPIEndpoint = "https://<hub-api-endpoint>"
	}

	helmCmd := fmt.Sprintf(
		"helm install ngf-console-agent charts/ngf-console-agent"+
			" --namespace ngf-system --create-namespace"+
			" --set cluster.name=%q"+
			" --set hub.apiEndpoint=%q"+
			" --set hub.otelEndpoint=%q",
		name, hubAPIEndpoint, hubAPIEndpoint+":4317",
	)

	writeJSON(w, http.StatusOK, map[string]string{
		"helmCommand": helmCmd,
		"clusterName": name,
	})
}

// Heartbeat receives a health report from a cluster agent.
func (h *ClusterHandler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	if h.Pool == nil {
		writeError(w, http.StatusNotImplemented, "heartbeat requires CRD-based multi-cluster mode")
		return
	}

	name := chi.URLParam(r, "cluster")

	var req HeartbeatRequest
	decoder := json.NewDecoder(io.LimitReader(r.Body, 64*1024)) // 64KB limit for heartbeat
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cc, err := h.Pool.Get(name)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("cluster %q not found", name))
		return
	}

	// Update in-memory state (protected by ClusterClient mutex).
	cc.SetHeartbeat(req.KubernetesVersion, req.NGFVersion, req.ResourceCounts, req.GPUCapacity)

	// Update CRD status on hub.
	status := map[string]interface{}{
		"phase":             "Ready",
		"kubernetesVersion": req.KubernetesVersion,
		"ngfVersion":        req.NGFVersion,
		"agentInstalled":    true,
		"lastHeartbeat":     time.Now().UTC().Format(time.RFC3339),
	}
	if req.ResourceCounts != nil {
		status["resourceCounts"] = req.ResourceCounts
	}
	if req.GPUCapacity != nil {
		status["gpuCapacity"] = req.GPUCapacity
	}

	if err := h.Pool.UpdateStatus(r.Context(), name, status); err != nil {
		slog.Error("failed to update cluster status", "cluster", name, "error", err)
		writeError(w, http.StatusInternalServerError, "failed to update cluster status")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Summary returns a global summary across all clusters.
func (h *ClusterHandler) Summary(w http.ResponseWriter, r *http.Request) {
	if h.Pool == nil {
		writeError(w, http.StatusNotImplemented, "summary requires CRD-based multi-cluster mode")
		return
	}

	clients := h.Pool.List()
	summary := ClusterSummaryResponse{
		TotalClusters: len(clients),
	}

	for _, cc := range clients {
		if cc.Healthy {
			summary.HealthyClusters++
		}
		if cc.ResourceCounts != nil {
			summary.TotalGateways += cc.ResourceCounts.Gateways
			summary.TotalRoutes += cc.ResourceCounts.HTTPRoutes
		}
		if cc.GPUCapacity != nil {
			summary.TotalGPUs += cc.GPUCapacity.TotalGPUs
		}
	}

	writeJSON(w, http.StatusOK, summary)
}
