package xc

import (
	"fmt"
	"strings"

	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// MapOptions controls how an HTTPRoute is mapped to an XC HTTP Load Balancer.
type MapOptions struct {
	XCNamespace    string // target XC namespace
	PublicHostname string // override hostname for edge (optional)
	WAFEnabled     bool   // whether to attach WAF
	WAFPolicyName  string // specific WAF policy to use (or empty for default)
	OriginPort     int32  // port to use for origin pool
	OriginTLS      bool   // whether origin uses TLS
}

// MapHTTPRouteToLoadBalancer derives an XC HTTP Load Balancer configuration from a Gateway API HTTPRoute.
func MapHTTPRouteToLoadBalancer(route *gatewayv1.HTTPRoute, gatewayAddress string, opts MapOptions) *HTTPLoadBalancer {
	name := "ngf-" + route.Name

	// Derive domains from HTTPRoute hostnames.
	domains := make([]string, 0)
	for _, h := range route.Spec.Hostnames {
		domains = append(domains, string(h))
	}
	if opts.PublicHostname != "" {
		// Add or replace with the public hostname.
		found := false
		for _, d := range domains {
			if d == opts.PublicHostname {
				found = true
				break
			}
		}
		if !found {
			domains = append([]string{opts.PublicHostname}, domains...)
		}
	}
	if len(domains) == 0 {
		domains = append(domains, name+".example.com")
	}

	// Build origin pool reference.
	poolName := name + "-pool"
	poolRef := PoolRef{
		Namespace: opts.XCNamespace,
		Name:      poolName,
	}

	// Build routes from HTTPRoute rules.
	routes := make([]Route, 0)
	for _, rule := range route.Spec.Rules {
		for _, match := range rule.Matches {
			pathMatch := PathMatch{}
			if match.Path != nil && match.Path.Value != nil {
				pathType := "PathPrefix"
				if match.Path.Type != nil {
					pathType = string(*match.Path.Type)
				}
				switch pathType {
				case "Exact":
					pathMatch.Exact = *match.Path.Value
				case "RegularExpression":
					pathMatch.Regex = *match.Path.Value
				default: // PathPrefix
					pathMatch.Prefix = *match.Path.Value
				}
			} else {
				pathMatch.Prefix = "/"
			}

			method := ""
			if match.Method != nil {
				method = string(*match.Method)
			}

			routes = append(routes, Route{
				SimpleRoute: &SimpleRoute{
					HTTPMethod: method,
					Path:       pathMatch,
					OriginPools: []RoutePool{
						{Pool: poolRef, Weight: 1},
					},
				},
			})
		}

		// If a rule has no matches, it matches everything.
		if len(rule.Matches) == 0 {
			routes = append(routes, Route{
				SimpleRoute: &SimpleRoute{
					Path: PathMatch{Prefix: "/"},
					OriginPools: []RoutePool{
						{Pool: poolRef, Weight: 1},
					},
				},
			})
		}
	}

	lb := &HTTPLoadBalancer{
		Metadata: ObjectMeta{
			Name:      name,
			Namespace: opts.XCNamespace,
		},
		Spec: HTTPLoadBalancerSpec{
			Domains: domains,
			AdvertiseOnPublic: &AdvertiseOnPublic{
				DefaultVIP: true,
			},
			DefaultRoutePools: []RoutePool{
				{Pool: poolRef, Weight: 1},
			},
		},
	}

	// Add explicit routes if there are path-specific rules.
	if len(routes) > 0 {
		hasNonDefault := false
		for _, rt := range routes {
			if rt.SimpleRoute != nil && (rt.SimpleRoute.Path.Exact != "" || rt.SimpleRoute.Path.Regex != "" ||
				(rt.SimpleRoute.Path.Prefix != "" && rt.SimpleRoute.Path.Prefix != "/") ||
				rt.SimpleRoute.HTTPMethod != "") {
				hasNonDefault = true
				break
			}
		}
		if hasNonDefault {
			lb.Spec.Routes = routes
		}
	}

	// Configure TLS: if origin uses TLS, set up HTTPS auto-cert; otherwise use HTTP.
	if opts.OriginTLS {
		lb.Spec.HTTPSAutoType = &HTTPSAutoType{HTTPRedirect: true}
	} else {
		port := uint32(80)
		lb.Spec.HTTPListenPort = &port
		lb.Spec.HTTP = &HTTPConfig{
			DNSVolterraManaged: true,
			Port:               port,
		}
	}

	// Attach WAF if enabled.
	if opts.WAFEnabled {
		policyName := opts.WAFPolicyName
		if policyName == "" {
			policyName = "ngf-default-waf"
		}
		lb.Spec.AppFirewall = &AppFirewallRef{
			Namespace: opts.XCNamespace,
			Name:      policyName,
		}
	} else {
		lb.Spec.DisableWAF = &EmptyObject{}
	}

	return lb
}

// BuildOriginPool creates an origin pool configuration that points back to the NGF Gateway.
func BuildOriginPool(routeName, gatewayAddress string, port int32, useTLS bool) *OriginPoolConfig {
	poolName := fmt.Sprintf("ngf-%s-pool", routeName)

	var originServer OriginServer
	if isIPAddress(gatewayAddress) {
		originServer.PublicIP = &PublicIP{IP: gatewayAddress}
	} else {
		originServer.PublicName = &PublicName{DNSName: gatewayAddress}
	}

	pool := &OriginPoolConfig{
		Metadata: ObjectMeta{
			Name: poolName,
		},
		Spec: OriginPoolSpec{
			OriginServers:    []OriginServer{originServer},
			Port:             uint32(port),
			LoadbalancerAlgo: "ROUND_ROBIN",
		},
	}

	if useTLS {
		pool.Spec.UseTLS = &OriginTLS{UseHostHeaderAsSNI: true}
	} else {
		pool.Spec.NoTLS = &EmptyObject{}
	}

	return pool
}

// isIPAddress returns true if the given string looks like an IP address.
func isIPAddress(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) != 4 {
		return false
	}
	for _, p := range parts {
		if len(p) == 0 || len(p) > 3 {
			return false
		}
		for _, c := range p {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}
