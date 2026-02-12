package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// InferenceStackSpec defines the desired state of an InferenceStack.
type InferenceStackSpec struct {
	// ModelName is the HuggingFace model identifier (e.g., "meta-llama/Llama-3-70B-Instruct").
	ModelName string `json:"modelName"`
	// ModelVersion is an optional version tag for the model.
	ModelVersion string `json:"modelVersion,omitempty"`
	// ServingBackend is the inference server type: "vllm", "triton", or "tgi".
	ServingBackend string `json:"servingBackend"`

	// Pool configures the InferencePool child resource.
	Pool InferencePoolSpec `json:"pool"`
	// EPP configures the Endpoint Picker Plugin.
	EPP EPPSpec `json:"epp,omitempty"`
	// Autoscaling configures KEDA-based autoscaling (Phase 2).
	Autoscaling *AutoscalingSpec `json:"autoscaling,omitempty"`
	// HTTPRoute configures the HTTPRoute child resource (Phase 2).
	HTTPRoute *HTTPRouteSpec `json:"httpRoute,omitempty"`
	// DCGM configures the DCGM GPU metrics exporter (Phase 2).
	DCGM *DCGMSpec `json:"dcgm,omitempty"`
	// DistributedCloud configures XC publishing (Phase 3).
	DistributedCloud *DistributedCloudConfig `json:"distributedCloud,omitempty"`
}

// InferencePoolSpec defines the pool parameters.
type InferencePoolSpec struct {
	// GPUType is the GPU accelerator type (e.g., "H100", "A100", "L40S", "T4").
	GPUType string `json:"gpuType"`
	// GPUCount is the number of GPUs per replica.
	GPUCount int32 `json:"gpuCount"`
	// Replicas is the desired number of replicas.
	Replicas int32 `json:"replicas"`
	// MinReplicas is the minimum replica count for autoscaling.
	MinReplicas int32 `json:"minReplicas"`
	// MaxReplicas is the maximum replica count for autoscaling.
	MaxReplicas int32 `json:"maxReplicas"`
	// Selector is the pod label selector for the inference pool.
	Selector map[string]string `json:"selector,omitempty"`
}

// EPPSpec configures the Endpoint Picker Plugin (EPP).
type EPPSpec struct {
	// Strategy is the routing strategy: "least_queue", "kv_cache", "prefix_affinity", "composite".
	Strategy string `json:"strategy,omitempty"`
	// Weights configures per-strategy weights when using composite strategy.
	Weights *EPPWeights `json:"weights,omitempty"`
}

// EPPWeights defines the strategy weights for composite routing.
type EPPWeights struct {
	QueueDepth     int32 `json:"queueDepth,omitempty"`
	KVCache        int32 `json:"kvCache,omitempty"`
	PrefixAffinity int32 `json:"prefixAffinity,omitempty"`
}

// AutoscalingSpec configures KEDA-based autoscaling.
type AutoscalingSpec struct {
	// Backend is the autoscaler backend: "keda" or "hpa".
	Backend string `json:"backend,omitempty"`
	// Thresholds are the scaling thresholds.
	Thresholds []ThresholdSpec `json:"thresholds,omitempty"`
	// CooldownSeconds is the cooldown period after a scaling event.
	CooldownSeconds int32 `json:"cooldownSeconds,omitempty"`
}

// HTTPRouteSpec configures the HTTPRoute child resource.
type HTTPRouteSpec struct {
	// Hostnames are the hostnames to match.
	Hostnames []string `json:"hostnames,omitempty"`
	// GatewayRef is the name of the Gateway to attach to.
	GatewayRef string `json:"gatewayRef,omitempty"`
	// GatewayNamespace is the namespace of the Gateway.
	GatewayNamespace string `json:"gatewayNamespace,omitempty"`
}

// DCGMSpec configures the DCGM exporter DaemonSet.
type DCGMSpec struct {
	// Enabled controls whether the DCGM exporter is deployed.
	Enabled bool `json:"enabled,omitempty"`
	// Image is the DCGM exporter container image.
	Image string `json:"image,omitempty"`
}

// InferenceStackStatus defines the observed state of an InferenceStack.
type InferenceStackStatus struct {
	// Phase is the aggregate lifecycle phase: Ready, Pending, Degraded, Error, Terminating.
	Phase string `json:"phase,omitempty"`
	// Children tracks the status of each reconciled child resource.
	Children []ChildStatus `json:"children,omitempty"`
	// Conditions are the standard Kubernetes conditions.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// ObservedSpecHash is the SHA256 hash of the last reconciled spec.
	ObservedSpecHash string `json:"observedSpecHash,omitempty"`
	// LastReconciledAt is the timestamp of the last successful reconciliation.
	LastReconciledAt *metav1.Time `json:"lastReconciledAt,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Model",type=string,JSONPath=`.spec.modelName`
// +kubebuilder:printcolumn:name="Backend",type=string,JSONPath=`.spec.servingBackend`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// InferenceStack is the Schema for the inferencestacks API.
// It declares a complete inference serving stack and the operator reconciles
// all child resources (InferencePool, EPP ConfigMap, KEDA ScaledObject, etc.).
type InferenceStack struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InferenceStackSpec   `json:"spec,omitempty"`
	Status InferenceStackStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// InferenceStackList contains a list of InferenceStack.
type InferenceStackList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InferenceStack `json:"items"`
}
