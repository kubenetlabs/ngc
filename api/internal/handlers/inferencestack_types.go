package handlers

// InferenceStack response types

// InferenceStackResponse represents an InferenceStack resource in the API response.
type InferenceStackResponse struct {
	Name             string                     `json:"name"`
	Namespace        string                     `json:"namespace"`
	ModelName        string                     `json:"modelName"`
	ModelVersion     string                     `json:"modelVersion,omitempty"`
	ServingBackend   string                     `json:"servingBackend"`
	Pool             InferenceStackPoolResponse `json:"pool"`
	EPP              *InferenceStackEPPResponse `json:"epp,omitempty"`
	Phase            string                     `json:"phase,omitempty"`
	Children         []ChildStatusResponse      `json:"children,omitempty"`
	ObservedSpecHash string                     `json:"observedSpecHash,omitempty"`
	LastReconciledAt string                     `json:"lastReconciledAt,omitempty"`
	CreatedAt        string                     `json:"createdAt"`
}

// InferenceStackPoolResponse represents the pool configuration within an InferenceStack.
type InferenceStackPoolResponse struct {
	GPUType     string            `json:"gpuType"`
	GPUCount    int               `json:"gpuCount"`
	Replicas    int               `json:"replicas"`
	MinReplicas int               `json:"minReplicas"`
	MaxReplicas int               `json:"maxReplicas"`
	Selector    map[string]string `json:"selector,omitempty"`
}

// InferenceStackEPPResponse represents the Endpoint Picker configuration within an InferenceStack.
type InferenceStackEPPResponse struct {
	Strategy string                     `json:"strategy"`
	Weights  *InferenceStackWeightsResp `json:"weights,omitempty"`
}

// InferenceStackWeightsResp represents EPP scheduling weights.
type InferenceStackWeightsResp struct {
	QueueDepth     int `json:"queueDepth"`
	KVCache        int `json:"kvCache"`
	PrefixAffinity int `json:"prefixAffinity"`
}

// ChildStatusResponse represents the status of a child resource managed by the InferenceStack.
type ChildStatusResponse struct {
	Kind    string `json:"kind"`
	Name    string `json:"name"`
	Ready   bool   `json:"ready"`
	Message string `json:"message,omitempty"`
}

// InferenceStack request types

// CreateInferenceStackRequest is the request body for creating an InferenceStack.
type CreateInferenceStackRequest struct {
	Name           string                       `json:"name"`
	Namespace      string                       `json:"namespace"`
	ModelName      string                       `json:"modelName"`
	ModelVersion   string                       `json:"modelVersion,omitempty"`
	ServingBackend string                       `json:"servingBackend"`
	Pool           CreateInferenceStackPoolReq  `json:"pool"`
	EPP            *CreateInferenceStackEPPReq  `json:"epp,omitempty"`
}

// CreateInferenceStackPoolReq represents the pool configuration in a create request.
type CreateInferenceStackPoolReq struct {
	GPUType     string            `json:"gpuType"`
	GPUCount    int               `json:"gpuCount"`
	Replicas    int               `json:"replicas"`
	MinReplicas int               `json:"minReplicas"`
	MaxReplicas int               `json:"maxReplicas"`
	Selector    map[string]string `json:"selector,omitempty"`
}

// CreateInferenceStackEPPReq represents the EPP configuration in a create request.
type CreateInferenceStackEPPReq struct {
	Strategy string                     `json:"strategy"`
	Weights  *InferenceStackWeightsResp `json:"weights,omitempty"`
}

// Topology response types

// TopologyResponse represents the full cluster topology graph.
type TopologyResponse struct {
	Nodes []TopologyNode `json:"nodes"`
	Edges []TopologyEdge `json:"edges"`
}

// TopologyNode represents a node in the topology graph.
type TopologyNode struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"` // "gateway", "httproute", "service"
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Status    string            `json:"status"` // "healthy", "degraded", "error"
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// TopologyEdge represents an edge in the topology graph.
type TopologyEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"` // "parentRef", "backendRef"
}
