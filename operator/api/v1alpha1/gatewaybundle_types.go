package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// GatewayBundleSpec defines the desired state of a GatewayBundle.
type GatewayBundleSpec struct {
	// GatewayClassName is the name of the GatewayClass to use.
	GatewayClassName string `json:"gatewayClassName"`
	// Listeners is the list of listeners for the Gateway child.
	Listeners []GatewayListenerSpec `json:"listeners"`
	// Labels to apply to the Gateway child.
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations to apply to the Gateway child.
	Annotations map[string]string `json:"annotations,omitempty"`

	// NginxProxy configures the NginxProxy child (Enterprise only).
	NginxProxy *NginxProxySpec `json:"nginxProxy,omitempty"`
	// WAF configures Web Application Firewall (Enterprise only).
	WAF *WAFSpec `json:"waf,omitempty"`
	// SnippetsFilter configures NGINX SnippetsFilter (Enterprise only).
	SnippetsFilter *SnippetsFilterSpec `json:"snippetsFilter,omitempty"`
	// TLS configures TLS certificate management.
	TLS *GatewayTLSSpec `json:"tls,omitempty"`
}

// GatewayListenerSpec defines a Gateway listener.
type GatewayListenerSpec struct {
	// Name is the listener name.
	Name string `json:"name"`
	// Port is the listener port number.
	Port int32 `json:"port"`
	// Protocol is the listener protocol: HTTP, HTTPS, TLS, TCP, UDP.
	Protocol string `json:"protocol"`
	// Hostname is the optional hostname for the listener.
	Hostname string `json:"hostname,omitempty"`
	// TLS configures TLS for this listener (when protocol is HTTPS or TLS).
	TLS *ListenerTLSSpec `json:"tls,omitempty"`
	// AllowedRoutes restricts which routes can attach to this listener.
	AllowedRoutes *AllowedRoutesSpec `json:"allowedRoutes,omitempty"`
}

// ListenerTLSSpec configures TLS on a listener.
type ListenerTLSSpec struct {
	// Mode is the TLS mode: Terminate or Passthrough.
	Mode string `json:"mode,omitempty"`
	// CertificateRefs are references to TLS certificates.
	CertificateRefs []CertRefSpec `json:"certificateRefs,omitempty"`
}

// CertRefSpec references a TLS certificate.
type CertRefSpec struct {
	// Name of the certificate Secret.
	Name string `json:"name"`
	// Namespace of the certificate Secret.
	Namespace string `json:"namespace,omitempty"`
}

// AllowedRoutesSpec restricts which routes can attach.
type AllowedRoutesSpec struct {
	// Namespaces controls which namespaces' routes can attach.
	Namespaces *RouteNamespacesSpec `json:"namespaces,omitempty"`
}

// RouteNamespacesSpec defines namespace selection for routes.
type RouteNamespacesSpec struct {
	// From determines the namespace scope: Same, All, or Selector.
	From string `json:"from,omitempty"`
	// Selector is a label selector when From is "Selector".
	Selector map[string]string `json:"selector,omitempty"`
}

// NginxProxySpec configures the NginxProxy resource (Enterprise).
type NginxProxySpec struct {
	// Enabled controls whether NginxProxy is deployed.
	Enabled bool `json:"enabled,omitempty"`
	// IPFamily sets the IP family: "dual", "ipv4", "ipv6".
	IPFamily string `json:"ipFamily,omitempty"`
	// RewriteClientIP configures client IP rewriting.
	RewriteClientIP *RewriteClientIPSpec `json:"rewriteClientIP,omitempty"`
	// Telemetry configures NGINX telemetry.
	Telemetry *NginxTelemetrySpec `json:"telemetry,omitempty"`
}

// RewriteClientIPSpec configures client IP rewriting.
type RewriteClientIPSpec struct {
	// Mode is the rewrite mode: "ProxyProtocol" or "XForwardedFor".
	Mode string `json:"mode,omitempty"`
	// SetIPRecursively enables recursive IP resolution.
	SetIPRecursively bool `json:"setIPRecursively,omitempty"`
}

// NginxTelemetrySpec configures NGINX telemetry.
type NginxTelemetrySpec struct {
	// Exporter configures the OpenTelemetry exporter.
	Exporter *OTelExporterSpec `json:"exporter,omitempty"`
}

// OTelExporterSpec configures the OTel exporter.
type OTelExporterSpec struct {
	// Endpoint is the OTLP endpoint.
	Endpoint string `json:"endpoint,omitempty"`
}

// WAFSpec configures Web Application Firewall (Enterprise).
type WAFSpec struct {
	// Enabled controls whether WAF is deployed.
	Enabled bool `json:"enabled,omitempty"`
	// PolicyRef references a WAF policy.
	PolicyRef string `json:"policyRef,omitempty"`
}

// SnippetsFilterSpec configures NGINX SnippetsFilter (Enterprise).
type SnippetsFilterSpec struct {
	// Enabled controls whether SnippetsFilter is created.
	Enabled bool `json:"enabled,omitempty"`
	// ServerSnippet is the NGINX server-context snippet.
	ServerSnippet string `json:"serverSnippet,omitempty"`
	// LocationSnippet is the NGINX location-context snippet.
	LocationSnippet string `json:"locationSnippet,omitempty"`
}

// GatewayTLSSpec configures TLS certificate management.
type GatewayTLSSpec struct {
	// SecretRefs lists TLS secret references to manage.
	SecretRefs []TLSSecretRef `json:"secretRefs,omitempty"`
}

// TLSSecretRef references a TLS secret.
type TLSSecretRef struct {
	// Name of the TLS Secret.
	Name string `json:"name"`
	// Namespace of the TLS Secret.
	Namespace string `json:"namespace,omitempty"`
}

// GatewayBundleStatus defines the observed state of a GatewayBundle.
type GatewayBundleStatus struct {
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
	// GatewayAddress is the externally-reachable address if available.
	GatewayAddress string `json:"gatewayAddress,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Class",type=string,JSONPath=`.spec.gatewayClassName`
// +kubebuilder:printcolumn:name="Address",type=string,JSONPath=`.status.gatewayAddress`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// GatewayBundle is the Schema for the gatewaybundles API.
// It declares a complete Gateway deployment and the operator reconciles
// the Gateway child along with optional NginxProxy, WAF, SnippetsFilter, and TLS children.
type GatewayBundle struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GatewayBundleSpec   `json:"spec,omitempty"`
	Status GatewayBundleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GatewayBundleList contains a list of GatewayBundle.
type GatewayBundleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GatewayBundle `json:"items"`
}

// Finalizer constant for GatewayBundle.
const GatewayBundleFinalizer = "ngf-console.f5.com/gatewaybundle-finalizer"
