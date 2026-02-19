package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// DistributedCloudPublishSpec defines the desired state of DistributedCloudPublish.
type DistributedCloudPublishSpec struct {
	// HTTPRouteRef is the name of the HTTPRoute to publish.
	HTTPRouteRef string `json:"httpRouteRef"`
	// InferencePoolRef is the optional name of an InferencePool to publish.
	InferencePoolRef string `json:"inferencePoolRef,omitempty"`
	// DistributedCloud holds the F5 Distributed Cloud configuration.
	DistributedCloud DistributedCloudConfig `json:"distributedCloud"`
}

// DistributedCloudConfig holds the Distributed Cloud configuration.
type DistributedCloudConfig struct {
	Tenant         string         `json:"tenant"`
	Namespace      string         `json:"namespace"`
	PublicHostname string         `json:"publicHostname"`
	WAFPolicy      string         `json:"wafPolicy,omitempty"`
	BotDefense     BotDefense     `json:"botDefense,omitempty"`
	DDoSProtection DDoSProtection `json:"ddosProtection,omitempty"`
	TLS            XCTLSConfig    `json:"tls,omitempty"`
	OriginPool     OriginPool     `json:"originPool,omitempty"`
	RateLimiting   RateLimiting   `json:"rateLimiting,omitempty"`
	MultiRegion    MultiRegion    `json:"multiRegion,omitempty"`
}

// BotDefense configures bot defense settings.
type BotDefense struct {
	Enabled bool `json:"enabled,omitempty"`
}

// DDoSProtection configures DDoS protection settings.
type DDoSProtection struct {
	Enabled bool `json:"enabled,omitempty"`
}

// XCTLSConfig configures TLS settings for XC.
type XCTLSConfig struct {
	SecretRef string `json:"secretRef,omitempty"`
}

// OriginPool configures origin pool settings.
type OriginPool struct {
	Name string `json:"name,omitempty"`
}

// RateLimiting configures rate limiting settings.
type RateLimiting struct {
	Enabled         bool   `json:"enabled,omitempty"`
	RequestsPerUnit int32  `json:"requestsPerUnit,omitempty"`
	Unit            string `json:"unit,omitempty"`
}

// MultiRegion configures multi-region settings.
type MultiRegion struct {
	Enabled bool     `json:"enabled,omitempty"`
	Regions []string `json:"regions,omitempty"`
}

// DistributedCloudPublishStatus defines the observed state of DistributedCloudPublish.
type DistributedCloudPublishStatus struct {
	// Conditions are the standard Kubernetes conditions.
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// Phase is the lifecycle phase.
	Phase string `json:"phase,omitempty"`
	// XCLoadBalancerName is the name of the HTTP LB created in XC.
	XCLoadBalancerName string `json:"xcLoadBalancerName,omitempty"`
	// XCOriginPoolName is the name of the origin pool created in XC.
	XCOriginPoolName string `json:"xcOriginPoolName,omitempty"`
	// XCVirtualIP is the virtual IP assigned by XC.
	XCVirtualIP string `json:"xcVirtualIP,omitempty"`
	// XCDNS is the DNS name assigned by XC.
	XCDNS string `json:"xcDNS,omitempty"`
	// WAFPolicyAttached is the name of the WAF policy attached to the LB.
	WAFPolicyAttached string `json:"wafPolicyAttached,omitempty"`
	// LastSyncedAt is the last time the XC resources were verified.
	LastSyncedAt *metav1.Time `json:"lastSyncedAt,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="HTTPRoute",type=string,JSONPath=`.spec.httpRouteRef`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// DistributedCloudPublish is the Schema for the distributedcloudpublishes API.
type DistributedCloudPublish struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DistributedCloudPublishSpec   `json:"spec,omitempty"`
	Status DistributedCloudPublishStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DistributedCloudPublishList contains a list of DistributedCloudPublish.
type DistributedCloudPublishList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DistributedCloudPublish `json:"items"`
}
