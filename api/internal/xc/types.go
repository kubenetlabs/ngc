package xc

// HTTPLoadBalancer represents an F5 Distributed Cloud HTTP Load Balancer configuration.
type HTTPLoadBalancer struct {
	Metadata ObjectMeta          `json:"metadata"`
	Spec     HTTPLoadBalancerSpec `json:"spec"`
}

// ObjectMeta holds standard XC object metadata.
type ObjectMeta struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

// HTTPLoadBalancerSpec defines the HTTP LB configuration.
type HTTPLoadBalancerSpec struct {
	Domains            []string            `json:"domains"`
	HTTPListenPort     *uint32             `json:"http_listen_port,omitempty"`
	HTTPSAutoType      *HTTPSAutoType      `json:"https_auto_cert,omitempty"`
	HTTP               *HTTPConfig         `json:"http,omitempty"`
	DefaultRoutePools  []RoutePool         `json:"default_route_pools,omitempty"`
	Routes             []Route             `json:"routes,omitempty"`
	AppFirewall                    *AppFirewallRef       `json:"app_firewall,omitempty"`
	DisableWAF                     *EmptyObject          `json:"disable_waf,omitempty"`
	AdvertiseOnPublicDefaultVIP    *EmptyObject          `json:"advertise_on_public_default_vip,omitempty"`
	DefaultPool                    *OriginPoolWithWeight `json:"default_pool,omitempty"`
}

// EmptyObject is used for XC API fields that take an empty object to indicate a setting.
type EmptyObject struct{}

// HTTPSAutoType configures automatic TLS certificate provisioning.
type HTTPSAutoType struct {
	HTTPRedirect bool `json:"http_redirect,omitempty"`
}

// HTTPConfig configures HTTP (non-TLS) listener.
type HTTPConfig struct {
	DNSVolterraManaged bool `json:"dns_volterra_managed,omitempty"`
	Port               uint32 `json:"port,omitempty"`
}

// RoutePool references an origin pool with weight for routing.
type RoutePool struct {
	Pool     PoolRef `json:"pool"`
	Weight   uint32  `json:"weight,omitempty"`
	Priority uint32  `json:"priority,omitempty"`
}

// PoolRef references an origin pool by tenant and namespace.
type PoolRef struct {
	Tenant    string `json:"tenant,omitempty"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// Route defines an XC HTTP LB route with match conditions and actions.
type Route struct {
	SimpleRoute *SimpleRoute `json:"simple_route,omitempty"`
}

// SimpleRoute is a basic route with path match and origin pools.
type SimpleRoute struct {
	HTTPMethod      string                      `json:"http_method,omitempty"`
	Path            PathMatch                   `json:"path"`
	OriginPools     []RoutePool                 `json:"origin_pools,omitempty"`
	HostRewrite     string                      `json:"host_rewrite,omitempty"`
	AutoHostRewrite *EmptyObject                `json:"auto_host_rewrite,omitempty"`
	AdvancedOptions *RouteSimpleAdvancedOptions `json:"advanced_options,omitempty"`
}

// RouteSimpleAdvancedOptions holds advanced route settings including WebSocket config.
type RouteSimpleAdvancedOptions struct {
	WebSocketConfig        *WebSocketConfig `json:"web_socket_config,omitempty"`
	DisableWebSocketConfig *EmptyObject     `json:"disable_web_socket_config,omitempty"`
}

// WebSocketConfig enables WebSocket protocol upgrade on a route.
type WebSocketConfig struct {
	UseWebSocket bool `json:"use_websocket,omitempty"`
}

// PathMatch defines path matching criteria.
type PathMatch struct {
	Prefix string `json:"prefix,omitempty"`
	Exact  string `json:"exact,omitempty"`
	Regex  string `json:"regex,omitempty"`
}

// AppFirewallRef references an XC App Firewall (WAF) policy.
type AppFirewallRef struct {
	Tenant    string `json:"tenant,omitempty"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

// OriginPoolWithWeight references origin pools in the default route.
type OriginPoolWithWeight struct {
	OriginPools []RoutePool `json:"origin_pools"`
}

// OriginPoolConfig represents an F5 Distributed Cloud origin pool.
type OriginPoolConfig struct {
	Metadata ObjectMeta       `json:"metadata"`
	Spec     OriginPoolSpec   `json:"spec"`
}

// OriginPoolSpec defines the origin pool configuration.
type OriginPoolSpec struct {
	OriginServers    []OriginServer `json:"origin_servers"`
	Port             uint32         `json:"port"`
	NoTLS            *EmptyObject   `json:"no_tls,omitempty"`
	UseTLS           *OriginTLS     `json:"use_tls,omitempty"`
	LoadbalancerAlgo string         `json:"loadbalancer_algorithm,omitempty"`
	HealthCheck      []HealthCheck  `json:"healthcheck,omitempty"`
}

// OriginServer defines a single origin server in a pool.
type OriginServer struct {
	PublicIP   *PublicIP   `json:"public_ip,omitempty"`
	PublicName *PublicName `json:"public_name,omitempty"`
}

// PublicIP references an origin by public IP address.
type PublicIP struct {
	IP string `json:"ip"`
}

// PublicName references an origin by DNS name.
type PublicName struct {
	DNSName string `json:"dns_name"`
}

// OriginTLS configures TLS to origin servers.
type OriginTLS struct {
	UseHostHeaderAsSNI bool `json:"use_host_header_as_sni,omitempty"`
}

// HealthCheck defines a health check configuration.
type HealthCheck struct {
	Tenant    string `json:"tenant,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
}

// AppFirewall represents an XC WAF policy.
type AppFirewall struct {
	Name        string `json:"name"`
	Namespace   string `json:"namespace,omitempty"`
	Description string `json:"description,omitempty"`
	Mode        string `json:"mode,omitempty"` // "blocking" or "monitoring"
}

// XCListResponse is the generic list response envelope from the XC API.
type XCListResponse[T any] struct {
	Items []T `json:"items"`
}

// XCError represents an error response from the XC API.
type XCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *XCError) Error() string {
	return e.Message
}
