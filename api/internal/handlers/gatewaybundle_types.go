package handlers

// GatewayBundle request types

// CreateGatewayBundleRequest is the request body for creating a GatewayBundle.
type CreateGatewayBundleRequest struct {
	Name             string                       `json:"name"`
	Namespace        string                       `json:"namespace"`
	GatewayClassName string                       `json:"gatewayClassName"`
	Listeners        []GatewayBundleListenerReq   `json:"listeners"`
	Labels           map[string]string            `json:"labels,omitempty"`
	Annotations      map[string]string            `json:"annotations,omitempty"`
	NginxProxy       *NginxProxyReq               `json:"nginxProxy,omitempty"`
	WAF              *WAFReq                      `json:"waf,omitempty"`
	SnippetsFilter   *SnippetsFilterReq           `json:"snippetsFilter,omitempty"`
	TLS              *GatewayTLSReq               `json:"tls,omitempty"`
}

// UpdateGatewayBundleRequest is the request body for updating a GatewayBundle.
type UpdateGatewayBundleRequest struct {
	GatewayClassName string                       `json:"gatewayClassName"`
	Listeners        []GatewayBundleListenerReq   `json:"listeners"`
	Labels           map[string]string            `json:"labels,omitempty"`
	Annotations      map[string]string            `json:"annotations,omitempty"`
	NginxProxy       *NginxProxyReq               `json:"nginxProxy,omitempty"`
	WAF              *WAFReq                      `json:"waf,omitempty"`
	SnippetsFilter   *SnippetsFilterReq           `json:"snippetsFilter,omitempty"`
	TLS              *GatewayTLSReq               `json:"tls,omitempty"`
}

// GatewayBundleListenerReq represents a listener in a GatewayBundle request.
type GatewayBundleListenerReq struct {
	Name          string                `json:"name"`
	Port          int32                 `json:"port"`
	Protocol      string                `json:"protocol"`
	Hostname      string                `json:"hostname,omitempty"`
	TLS           *ListenerTLSReq       `json:"tls,omitempty"`
	AllowedRoutes *AllowedRoutesReq     `json:"allowedRoutes,omitempty"`
}

// ListenerTLSReq configures TLS on a listener.
type ListenerTLSReq struct {
	Mode            string        `json:"mode,omitempty"`
	CertificateRefs []CertRefReq `json:"certificateRefs,omitempty"`
}

// CertRefReq references a TLS certificate.
type CertRefReq struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

// AllowedRoutesReq restricts which routes can attach.
type AllowedRoutesReq struct {
	Namespaces *RouteNamespacesReq `json:"namespaces,omitempty"`
}

// RouteNamespacesReq defines namespace selection for routes.
type RouteNamespacesReq struct {
	From     string            `json:"from,omitempty"`
	Selector map[string]string `json:"selector,omitempty"`
}

// NginxProxyReq configures the NginxProxy resource.
type NginxProxyReq struct {
	Enabled         bool                 `json:"enabled,omitempty"`
	IPFamily        string               `json:"ipFamily,omitempty"`
	RewriteClientIP *RewriteClientIPReq  `json:"rewriteClientIP,omitempty"`
	Telemetry       *NginxTelemetryReq   `json:"telemetry,omitempty"`
}

// RewriteClientIPReq configures client IP rewriting.
type RewriteClientIPReq struct {
	Mode             string `json:"mode,omitempty"`
	SetIPRecursively bool   `json:"setIPRecursively,omitempty"`
}

// NginxTelemetryReq configures NGINX telemetry.
type NginxTelemetryReq struct {
	Exporter *OTelExporterReq `json:"exporter,omitempty"`
}

// OTelExporterReq configures the OTel exporter.
type OTelExporterReq struct {
	Endpoint string `json:"endpoint,omitempty"`
}

// WAFReq configures Web Application Firewall.
type WAFReq struct {
	Enabled   bool   `json:"enabled,omitempty"`
	PolicyRef string `json:"policyRef,omitempty"`
}

// SnippetsFilterReq configures NGINX SnippetsFilter.
type SnippetsFilterReq struct {
	Enabled         bool   `json:"enabled,omitempty"`
	ServerSnippet   string `json:"serverSnippet,omitempty"`
	LocationSnippet string `json:"locationSnippet,omitempty"`
}

// GatewayTLSReq configures TLS certificate management.
type GatewayTLSReq struct {
	SecretRefs []TLSSecretRefReq `json:"secretRefs,omitempty"`
}

// TLSSecretRefReq references a TLS secret.
type TLSSecretRefReq struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

// GatewayBundle response types

// GatewayBundleResponse represents a GatewayBundle resource in the API response.
type GatewayBundleResponse struct {
	Name             string                          `json:"name"`
	Namespace        string                          `json:"namespace"`
	GatewayClassName string                          `json:"gatewayClassName"`
	Listeners        []GatewayBundleListenerResp     `json:"listeners"`
	Labels           map[string]string               `json:"labels,omitempty"`
	Annotations      map[string]string               `json:"annotations,omitempty"`
	NginxProxy       *NginxProxyResp                 `json:"nginxProxy,omitempty"`
	WAF              *WAFResp                        `json:"waf,omitempty"`
	SnippetsFilter   *SnippetsFilterResp             `json:"snippetsFilter,omitempty"`
	TLS              *GatewayTLSResp                 `json:"tls,omitempty"`
	Phase            string                          `json:"phase,omitempty"`
	Children         []ChildStatusResponse           `json:"children,omitempty"`
	Conditions       []ConditionResponse             `json:"conditions,omitempty"`
	GatewayAddress   string                          `json:"gatewayAddress,omitempty"`
	ObservedSpecHash string                          `json:"observedSpecHash,omitempty"`
	LastReconciledAt string                          `json:"lastReconciledAt,omitempty"`
	CreatedAt        string                          `json:"createdAt"`
}

// GatewayBundleListenerResp represents a listener in a GatewayBundle response.
type GatewayBundleListenerResp struct {
	Name          string                 `json:"name"`
	Port          int32                  `json:"port"`
	Protocol      string                 `json:"protocol"`
	Hostname      string                 `json:"hostname,omitempty"`
	TLS           *ListenerTLSResp       `json:"tls,omitempty"`
	AllowedRoutes *AllowedRoutesResp     `json:"allowedRoutes,omitempty"`
}

// ListenerTLSResp represents TLS configuration in a response.
type ListenerTLSResp struct {
	Mode            string         `json:"mode,omitempty"`
	CertificateRefs []CertRefResp `json:"certificateRefs,omitempty"`
}

// CertRefResp represents a certificate reference in a response.
type CertRefResp struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

// NginxProxyResp represents the NginxProxy configuration in a response.
type NginxProxyResp struct {
	Enabled         bool                  `json:"enabled,omitempty"`
	IPFamily        string                `json:"ipFamily,omitempty"`
	RewriteClientIP *RewriteClientIPResp  `json:"rewriteClientIP,omitempty"`
	Telemetry       *NginxTelemetryResp   `json:"telemetry,omitempty"`
}

// RewriteClientIPResp represents client IP rewriting in a response.
type RewriteClientIPResp struct {
	Mode             string `json:"mode,omitempty"`
	SetIPRecursively bool   `json:"setIPRecursively,omitempty"`
}

// NginxTelemetryResp represents NGINX telemetry in a response.
type NginxTelemetryResp struct {
	Exporter *OTelExporterResp `json:"exporter,omitempty"`
}

// OTelExporterResp represents the OTel exporter in a response.
type OTelExporterResp struct {
	Endpoint string `json:"endpoint,omitempty"`
}

// WAFResp represents WAF configuration in a response.
type WAFResp struct {
	Enabled   bool   `json:"enabled,omitempty"`
	PolicyRef string `json:"policyRef,omitempty"`
}

// SnippetsFilterResp represents SnippetsFilter configuration in a response.
type SnippetsFilterResp struct {
	Enabled         bool   `json:"enabled,omitempty"`
	ServerSnippet   string `json:"serverSnippet,omitempty"`
	LocationSnippet string `json:"locationSnippet,omitempty"`
}

// GatewayTLSResp represents TLS configuration in a response.
type GatewayTLSResp struct {
	SecretRefs []TLSSecretRefResp `json:"secretRefs,omitempty"`
}

// TLSSecretRefResp represents a TLS secret reference in a response.
type TLSSecretRefResp struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}
