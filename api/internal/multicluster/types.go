package multicluster

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// SecretReference is a reference to a Secret in the same namespace.
type SecretReference struct {
	// Name of the Secret.
	Name string `json:"name"`
}

// AgentConfigOverrides allows the hub to push configuration to the agent.
type AgentConfigOverrides struct {
	// HeartbeatIntervalSeconds overrides the default 30s heartbeat interval.
	HeartbeatIntervalSeconds *int32 `json:"heartbeatIntervalSeconds,omitempty"`
	// OTelEndpoint overrides the OTel Collector endpoint on the hub.
	OTelEndpoint string `json:"otelEndpoint,omitempty"`
}

// ResourceCounts tracks the number of key K8s/Gateway API resources.
type ResourceCounts struct {
	Gateways        int32 `json:"gateways"`
	HTTPRoutes      int32 `json:"httpRoutes"`
	InferencePools  int32 `json:"inferencePools"`
	InferenceStacks int32 `json:"inferenceStacks"`
	GatewayBundles  int32 `json:"gatewayBundles"`
	Services        int32 `json:"services"`
	Namespaces      int32 `json:"namespaces"`
}

// GPUCapacitySummary provides aggregate GPU information for the cluster.
type GPUCapacitySummary struct {
	TotalGPUs     int32             `json:"totalGPUs"`
	AllocatedGPUs int32             `json:"allocatedGPUs"`
	GPUTypes      map[string]int32  `json:"gpuTypes,omitempty"`
}

// ManagedClusterSpec defines the desired state of a ManagedCluster.
type ManagedClusterSpec struct {
	// DisplayName is a human-readable name shown in the UI.
	DisplayName string `json:"displayName"`
	// Region is the cloud provider region (e.g., "us-east-1").
	Region string `json:"region,omitempty"`
	// Environment classifies the cluster (e.g., "production", "staging", "gpu").
	Environment string `json:"environment,omitempty"`
	// Labels are additional key-value labels for filtering.
	Labels map[string]string `json:"labels,omitempty"`

	// KubeconfigSecretRef references a Secret containing the kubeconfig for
	// connecting to this cluster. Ignored when IsLocal is true.
	KubeconfigSecretRef *SecretReference `json:"kubeconfigSecretRef,omitempty"`
	// PrometheusSecretRef references a Secret containing the Prometheus URL
	// for this cluster.
	PrometheusSecretRef *SecretReference `json:"prometheusSecretRef,omitempty"`

	// IsLocal indicates this cluster is the hub cluster itself (use in-cluster config).
	IsLocal bool `json:"isLocal,omitempty"`
	// NGFEdition indicates the expected NGF edition: "oss" or "enterprise".
	NGFEdition string `json:"ngfEdition,omitempty"`
	// AgentConfig provides optional overrides for the agent running in this cluster.
	AgentConfig *AgentConfigOverrides `json:"agentConfig,omitempty"`
}

// ManagedClusterPhase represents the lifecycle phase of a ManagedCluster.
type ManagedClusterPhase string

const (
	ClusterPhasePending      ManagedClusterPhase = "Pending"
	ClusterPhaseConnecting   ManagedClusterPhase = "Connecting"
	ClusterPhaseReady        ManagedClusterPhase = "Ready"
	ClusterPhaseDegraded     ManagedClusterPhase = "Degraded"
	ClusterPhaseUnreachable  ManagedClusterPhase = "Unreachable"
	ClusterPhaseTerminating  ManagedClusterPhase = "Terminating"
)

// ManagedClusterStatus defines the observed state of a ManagedCluster.
type ManagedClusterStatus struct {
	// Phase is the aggregate lifecycle phase.
	Phase ManagedClusterPhase `json:"phase,omitempty"`
	// KubernetesVersion is the cluster's K8s server version (e.g., "v1.31.2").
	KubernetesVersion string `json:"kubernetesVersion,omitempty"`
	// NGFVersion is the version of NGINX Gateway Fabric installed.
	NGFVersion string `json:"ngfVersion,omitempty"`
	// AgentInstalled indicates whether the NGF Console agent is running.
	AgentInstalled bool `json:"agentInstalled,omitempty"`
	// LastHeartbeat is the timestamp of the last successful agent heartbeat.
	LastHeartbeat *metav1.Time `json:"lastHeartbeat,omitempty"`
	// ResourceCounts tracks key resource counts in this cluster.
	ResourceCounts *ResourceCounts `json:"resourceCounts,omitempty"`
	// GPUCapacity summarizes GPU availability.
	GPUCapacity *GPUCapacitySummary `json:"gpuCapacity,omitempty"`
	// Conditions are standard K8s conditions for the cluster.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Display Name",type=string,JSONPath=`.spec.displayName`
// +kubebuilder:printcolumn:name="Region",type=string,JSONPath=`.spec.region`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="NGF",type=string,JSONPath=`.status.ngfVersion`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ManagedCluster represents a Kubernetes cluster managed by the NGF Console hub.
type ManagedCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ManagedClusterSpec   `json:"spec,omitempty"`
	Status ManagedClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ManagedClusterList contains a list of ManagedCluster.
type ManagedClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedCluster `json:"items"`
}
