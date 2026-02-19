package handlers

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1alpha2 "sigs.k8s.io/gateway-api/apis/v1alpha2"
)

// ─── GRPCRoute response types ───────────────────────────────────────────────

// GRPCRouteResponse is the JSON representation of a GRPCRoute.
type GRPCRouteResponse struct {
	Name       string                   `json:"name"`
	Namespace  string                   `json:"namespace"`
	ParentRefs []ParentRefResponse      `json:"parentRefs"`
	Hostnames  []string                 `json:"hostnames,omitempty"`
	Rules      []GRPCRouteRuleResponse  `json:"rules"`
	Status     *HTTPRouteStatusResponse `json:"status,omitempty"`
	CreatedAt  string                   `json:"createdAt"`
}

// GRPCRouteRuleResponse is a single rule inside a GRPCRoute.
type GRPCRouteRuleResponse struct {
	Matches     []GRPCRouteMatchResponse `json:"matches,omitempty"`
	BackendRefs []BackendRefResponse     `json:"backendRefs,omitempty"`
}

// GRPCRouteMatchResponse describes a match condition for a gRPC request.
type GRPCRouteMatchResponse struct {
	Method  *GRPCMethodMatchResponse `json:"method,omitempty"`
	Headers []HeaderMatchResp        `json:"headers,omitempty"`
}

// GRPCMethodMatchResponse describes the gRPC service/method match.
type GRPCMethodMatchResponse struct {
	Type    string  `json:"type,omitempty"`
	Service *string `json:"service,omitempty"`
	Method  *string `json:"method,omitempty"`
}

// ─── TLSRoute response types ────────────────────────────────────────────────

// TLSRouteResponse is the JSON representation of a TLSRoute.
type TLSRouteResponse struct {
	Name       string                   `json:"name"`
	Namespace  string                   `json:"namespace"`
	ParentRefs []ParentRefResponse      `json:"parentRefs"`
	Hostnames  []string                 `json:"hostnames,omitempty"`
	Rules      []TLSRouteRuleResponse   `json:"rules"`
	Status     *HTTPRouteStatusResponse `json:"status,omitempty"`
	CreatedAt  string                   `json:"createdAt"`
}

// TLSRouteRuleResponse is a single rule inside a TLSRoute.
type TLSRouteRuleResponse struct {
	BackendRefs []BackendRefResponse `json:"backendRefs,omitempty"`
}

// ─── TCPRoute response types ────────────────────────────────────────────────

// TCPRouteResponse is the JSON representation of a TCPRoute.
type TCPRouteResponse struct {
	Name       string                   `json:"name"`
	Namespace  string                   `json:"namespace"`
	ParentRefs []ParentRefResponse      `json:"parentRefs"`
	Rules      []TCPRouteRuleResponse   `json:"rules"`
	Status     *HTTPRouteStatusResponse `json:"status,omitempty"`
	CreatedAt  string                   `json:"createdAt"`
}

// TCPRouteRuleResponse is a single rule inside a TCPRoute.
type TCPRouteRuleResponse struct {
	BackendRefs []BackendRefResponse `json:"backendRefs,omitempty"`
}

// ─── UDPRoute response types ────────────────────────────────────────────────

// UDPRouteResponse is the JSON representation of a UDPRoute.
type UDPRouteResponse struct {
	Name       string                   `json:"name"`
	Namespace  string                   `json:"namespace"`
	ParentRefs []ParentRefResponse      `json:"parentRefs"`
	Rules      []UDPRouteRuleResponse   `json:"rules"`
	Status     *HTTPRouteStatusResponse `json:"status,omitempty"`
	CreatedAt  string                   `json:"createdAt"`
}

// UDPRouteRuleResponse is a single rule inside a UDPRoute.
type UDPRouteRuleResponse struct {
	BackendRefs []BackendRefResponse `json:"backendRefs,omitempty"`
}

// ─── GRPCRoute request types ────────────────────────────────────────────────

// CreateGRPCRouteRequest is the request body to create a GRPCRoute.
type CreateGRPCRouteRequest struct {
	Name       string             `json:"name"`
	Namespace  string             `json:"namespace"`
	ParentRefs []ParentRefRequest `json:"parentRefs"`
	Hostnames  []string           `json:"hostnames,omitempty"`
	Rules      []GRPCRouteRuleReq `json:"rules"`
}

// UpdateGRPCRouteRequest is the request body to update a GRPCRoute.
type UpdateGRPCRouteRequest struct {
	ParentRefs []ParentRefRequest `json:"parentRefs"`
	Hostnames  []string           `json:"hostnames,omitempty"`
	Rules      []GRPCRouteRuleReq `json:"rules"`
}

// GRPCRouteRuleReq is a rule in a create/update GRPCRoute request.
type GRPCRouteRuleReq struct {
	Matches     []GRPCRouteMatchReq `json:"matches,omitempty"`
	BackendRefs []BackendRefRequest `json:"backendRefs,omitempty"`
}

// GRPCRouteMatchReq is a match condition in a GRPCRoute rule request.
type GRPCRouteMatchReq struct {
	Method  *GRPCMethodMatchReq  `json:"method,omitempty"`
	Headers []HeaderMatchRequest `json:"headers,omitempty"`
}

// GRPCMethodMatchReq specifies gRPC method matching in a request.
type GRPCMethodMatchReq struct {
	Type    string  `json:"type,omitempty"`
	Service *string `json:"service,omitempty"`
	Method  *string `json:"method,omitempty"`
}

// ─── TLS/TCP/UDP route request types ────────────────────────────────────────

// CreateTLSRouteRequest is the request body to create a TLSRoute.
type CreateTLSRouteRequest struct {
	Name       string               `json:"name"`
	Namespace  string               `json:"namespace"`
	ParentRefs []ParentRefRequest   `json:"parentRefs"`
	Hostnames  []string             `json:"hostnames,omitempty"`
	Rules      []SimpleRouteRuleReq `json:"rules"`
}

// UpdateTLSRouteRequest is the request body to update a TLSRoute.
type UpdateTLSRouteRequest struct {
	ParentRefs []ParentRefRequest   `json:"parentRefs"`
	Hostnames  []string             `json:"hostnames,omitempty"`
	Rules      []SimpleRouteRuleReq `json:"rules"`
}

// CreateTCPRouteRequest is the request body to create a TCPRoute.
type CreateTCPRouteRequest struct {
	Name       string               `json:"name"`
	Namespace  string               `json:"namespace"`
	ParentRefs []ParentRefRequest   `json:"parentRefs"`
	Rules      []SimpleRouteRuleReq `json:"rules"`
}

// UpdateTCPRouteRequest is the request body to update a TCPRoute.
type UpdateTCPRouteRequest struct {
	ParentRefs []ParentRefRequest   `json:"parentRefs"`
	Rules      []SimpleRouteRuleReq `json:"rules"`
}

// CreateUDPRouteRequest is the request body to create a UDPRoute.
type CreateUDPRouteRequest struct {
	Name       string               `json:"name"`
	Namespace  string               `json:"namespace"`
	ParentRefs []ParentRefRequest   `json:"parentRefs"`
	Rules      []SimpleRouteRuleReq `json:"rules"`
}

// UpdateUDPRouteRequest is the request body to update a UDPRoute.
type UpdateUDPRouteRequest struct {
	ParentRefs []ParentRefRequest   `json:"parentRefs"`
	Rules      []SimpleRouteRuleReq `json:"rules"`
}

// SimpleRouteRuleReq is a rule containing only backendRefs (used by TLS/TCP/UDP routes).
type SimpleRouteRuleReq struct {
	BackendRefs []BackendRefRequest `json:"backendRefs,omitempty"`
}

// ─── Conversion helpers ─────────────────────────────────────────────────────

// convertBackendRefRequests converts a slice of BackendRefRequest to Gateway API BackendRef objects.
func convertBackendRefRequests(refs []BackendRefRequest) []gatewayv1.BackendRef {
	result := make([]gatewayv1.BackendRef, 0, len(refs))
	for _, br := range refs {
		ref := gatewayv1.BackendRef{
			BackendObjectReference: gatewayv1.BackendObjectReference{
				Name: gatewayv1.ObjectName(br.Name),
			},
			Weight: br.Weight,
		}
		if br.Namespace != nil {
			ns := gatewayv1.Namespace(*br.Namespace)
			ref.Namespace = &ns
		}
		if br.Port != nil {
			port := gatewayv1.PortNumber(*br.Port)
			ref.Port = &port
		}
		result = append(result, ref)
	}
	return result
}

// convertBackendRefsToResponse converts a slice of Gateway API BackendRef to response objects.
func convertBackendRefsToResponse(refs []gatewayv1.BackendRef) []BackendRefResponse {
	result := make([]BackendRefResponse, 0, len(refs))
	for _, br := range refs {
		brr := BackendRefResponse{Name: string(br.Name)}
		if br.Group != nil {
			brr.Group = string(*br.Group)
		}
		if br.Kind != nil {
			brr.Kind = string(*br.Kind)
		}
		if br.Namespace != nil {
			ns := string(*br.Namespace)
			brr.Namespace = &ns
		}
		if br.Port != nil {
			p := int32(*br.Port)
			brr.Port = &p
		}
		if br.Weight != nil {
			w := *br.Weight
			brr.Weight = &w
		}
		result = append(result, brr)
	}
	return result
}

// convertParentRefsToResponse converts Gateway API ParentReferences to response objects.
func convertParentRefsToResponse(refs []gatewayv1.ParentReference) []ParentRefResponse {
	result := make([]ParentRefResponse, 0, len(refs))
	for _, pr := range refs {
		pResp := ParentRefResponse{Name: string(pr.Name)}
		if pr.Group != nil {
			pResp.Group = string(*pr.Group)
		}
		if pr.Kind != nil {
			pResp.Kind = string(*pr.Kind)
		}
		if pr.Namespace != nil {
			ns := string(*pr.Namespace)
			pResp.Namespace = &ns
		}
		if pr.SectionName != nil {
			sn := string(*pr.SectionName)
			pResp.SectionName = &sn
		}
		if pr.Port != nil {
			p := int32(*pr.Port)
			pResp.Port = &p
		}
		result = append(result, pResp)
	}
	return result
}

// convertRouteStatusToResponse converts a Gateway API RouteStatus to the shared HTTPRouteStatusResponse.
func convertRouteStatusToResponse(parents []gatewayv1.RouteParentStatus) *HTTPRouteStatusResponse {
	if len(parents) == 0 {
		return nil
	}
	status := &HTTPRouteStatusResponse{}
	for _, ps := range parents {
		psr := RouteParentStatusResponse{
			ControllerName: string(ps.ControllerName),
			Conditions:     convertConditions(ps.Conditions),
		}
		psr.ParentRef = ParentRefResponse{Name: string(ps.ParentRef.Name)}
		if ps.ParentRef.Group != nil {
			psr.ParentRef.Group = string(*ps.ParentRef.Group)
		}
		if ps.ParentRef.Kind != nil {
			psr.ParentRef.Kind = string(*ps.ParentRef.Kind)
		}
		if ps.ParentRef.Namespace != nil {
			ns := string(*ps.ParentRef.Namespace)
			psr.ParentRef.Namespace = &ns
		}
		if ps.ParentRef.SectionName != nil {
			sn := string(*ps.ParentRef.SectionName)
			psr.ParentRef.SectionName = &sn
		}
		status.Parents = append(status.Parents, psr)
	}
	return status
}

// ─── GRPCRoute conversions ──────────────────────────────────────────────────

// toGRPCRouteResponse converts a Gateway API GRPCRoute to the JSON response type.
func toGRPCRouteResponse(route *gatewayv1.GRPCRoute) GRPCRouteResponse {
	resp := GRPCRouteResponse{
		Name:      route.Name,
		Namespace: route.Namespace,
		CreatedAt: route.CreationTimestamp.UTC().Format("2006-01-02T15:04:05Z"),
	}

	// Parent refs
	resp.ParentRefs = convertParentRefsToResponse(route.Spec.ParentRefs)

	// Hostnames
	for _, h := range route.Spec.Hostnames {
		resp.Hostnames = append(resp.Hostnames, string(h))
	}

	// Rules
	resp.Rules = make([]GRPCRouteRuleResponse, 0, len(route.Spec.Rules))
	for _, rule := range route.Spec.Rules {
		rr := GRPCRouteRuleResponse{}

		// Matches
		for _, m := range rule.Matches {
			mr := GRPCRouteMatchResponse{}
			if m.Method != nil {
				mm := &GRPCMethodMatchResponse{}
				if m.Method.Type != nil {
					mm.Type = string(*m.Method.Type)
				}
				mm.Service = m.Method.Service
				mm.Method = m.Method.Method
				mr.Method = mm
			}
			for _, h := range m.Headers {
				hm := HeaderMatchResp{Name: string(h.Name), Value: h.Value}
				if h.Type != nil {
					hm.Type = string(*h.Type)
				}
				mr.Headers = append(mr.Headers, hm)
			}
			rr.Matches = append(rr.Matches, mr)
		}

		// BackendRefs — GRPCBackendRef embeds BackendRef
		for _, br := range rule.BackendRefs {
			brr := BackendRefResponse{Name: string(br.Name)}
			if br.Group != nil {
				brr.Group = string(*br.Group)
			}
			if br.Kind != nil {
				brr.Kind = string(*br.Kind)
			}
			if br.Namespace != nil {
				ns := string(*br.Namespace)
				brr.Namespace = &ns
			}
			if br.Port != nil {
				p := int32(*br.Port)
				brr.Port = &p
			}
			if br.Weight != nil {
				w := *br.Weight
				brr.Weight = &w
			}
			rr.BackendRefs = append(rr.BackendRefs, brr)
		}

		resp.Rules = append(resp.Rules, rr)
	}

	// Status
	resp.Status = convertRouteStatusToResponse(route.Status.Parents)

	return resp
}

// toGRPCRouteObject converts a create request into a Gateway API GRPCRoute object.
func toGRPCRouteObject(req CreateGRPCRouteRequest) *gatewayv1.GRPCRoute {
	route := &gatewayv1.GRPCRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
		Spec: gatewayv1.GRPCRouteSpec{},
	}
	route.Spec.ParentRefs = convertParentRefRequests(req.ParentRefs)

	for _, h := range req.Hostnames {
		route.Spec.Hostnames = append(route.Spec.Hostnames, gatewayv1.Hostname(h))
	}

	route.Spec.Rules = convertGRPCRouteRuleRequests(req.Rules)
	return route
}

// applyUpdateToGRPCRoute applies an update request to an existing GRPCRoute.
func applyUpdateToGRPCRoute(route *gatewayv1.GRPCRoute, req UpdateGRPCRouteRequest) {
	route.Spec.ParentRefs = convertParentRefRequests(req.ParentRefs)

	route.Spec.Hostnames = nil
	for _, h := range req.Hostnames {
		route.Spec.Hostnames = append(route.Spec.Hostnames, gatewayv1.Hostname(h))
	}

	route.Spec.Rules = convertGRPCRouteRuleRequests(req.Rules)
}

// convertGRPCRouteRuleRequests converts request rules to Gateway API GRPCRouteRule objects.
func convertGRPCRouteRuleRequests(rules []GRPCRouteRuleReq) []gatewayv1.GRPCRouteRule {
	result := make([]gatewayv1.GRPCRouteRule, 0, len(rules))
	for _, rule := range rules {
		r := gatewayv1.GRPCRouteRule{}

		// Matches
		for _, m := range rule.Matches {
			match := gatewayv1.GRPCRouteMatch{}
			if m.Method != nil {
				methodMatch := &gatewayv1.GRPCMethodMatch{
					Service: m.Method.Service,
					Method:  m.Method.Method,
				}
				if m.Method.Type != "" {
					t := gatewayv1.GRPCMethodMatchType(m.Method.Type)
					methodMatch.Type = &t
				}
				match.Method = methodMatch
			}
			for _, h := range m.Headers {
				headerType := gatewayv1.GRPCHeaderMatchType(h.Type)
				match.Headers = append(match.Headers, gatewayv1.GRPCHeaderMatch{
					Type:  &headerType,
					Name:  gatewayv1.GRPCHeaderName(h.Name),
					Value: h.Value,
				})
			}
			r.Matches = append(r.Matches, match)
		}

		// BackendRefs — wrap BackendRef into GRPCBackendRef
		for _, br := range rule.BackendRefs {
			backendRef := gatewayv1.GRPCBackendRef{
				BackendRef: gatewayv1.BackendRef{
					BackendObjectReference: gatewayv1.BackendObjectReference{
						Name: gatewayv1.ObjectName(br.Name),
					},
					Weight: br.Weight,
				},
			}
			if br.Namespace != nil {
				ns := gatewayv1.Namespace(*br.Namespace)
				backendRef.BackendRef.Namespace = &ns
			}
			if br.Port != nil {
				port := gatewayv1.PortNumber(*br.Port)
				backendRef.BackendRef.Port = &port
			}
			r.BackendRefs = append(r.BackendRefs, backendRef)
		}

		result = append(result, r)
	}
	return result
}

// ─── TLSRoute conversions ───────────────────────────────────────────────────

// toTLSRouteResponse converts a Gateway API TLSRoute to the JSON response type.
func toTLSRouteResponse(route *gatewayv1alpha2.TLSRoute) TLSRouteResponse {
	resp := TLSRouteResponse{
		Name:      route.Name,
		Namespace: route.Namespace,
		CreatedAt: route.CreationTimestamp.UTC().Format("2006-01-02T15:04:05Z"),
	}

	resp.ParentRefs = convertParentRefsToResponse(route.Spec.ParentRefs)

	for _, h := range route.Spec.Hostnames {
		resp.Hostnames = append(resp.Hostnames, string(h))
	}

	resp.Rules = make([]TLSRouteRuleResponse, 0, len(route.Spec.Rules))
	for _, rule := range route.Spec.Rules {
		rr := TLSRouteRuleResponse{
			BackendRefs: convertBackendRefsToResponse(rule.BackendRefs),
		}
		resp.Rules = append(resp.Rules, rr)
	}

	resp.Status = convertRouteStatusToResponse(route.Status.Parents)

	return resp
}

// toTLSRouteObject converts a create request into a Gateway API TLSRoute object.
func toTLSRouteObject(req CreateTLSRouteRequest) *gatewayv1alpha2.TLSRoute {
	route := &gatewayv1alpha2.TLSRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
		Spec: gatewayv1alpha2.TLSRouteSpec{},
	}
	route.Spec.ParentRefs = convertParentRefRequests(req.ParentRefs)

	for _, h := range req.Hostnames {
		route.Spec.Hostnames = append(route.Spec.Hostnames, gatewayv1alpha2.Hostname(h))
	}

	route.Spec.Rules = make([]gatewayv1alpha2.TLSRouteRule, 0, len(req.Rules))
	for _, rule := range req.Rules {
		route.Spec.Rules = append(route.Spec.Rules, gatewayv1alpha2.TLSRouteRule{
			BackendRefs: convertBackendRefRequests(rule.BackendRefs),
		})
	}

	return route
}

// applyUpdateToTLSRoute applies an update request to an existing TLSRoute.
func applyUpdateToTLSRoute(route *gatewayv1alpha2.TLSRoute, req UpdateTLSRouteRequest) {
	route.Spec.ParentRefs = convertParentRefRequests(req.ParentRefs)

	route.Spec.Hostnames = nil
	for _, h := range req.Hostnames {
		route.Spec.Hostnames = append(route.Spec.Hostnames, gatewayv1alpha2.Hostname(h))
	}

	route.Spec.Rules = make([]gatewayv1alpha2.TLSRouteRule, 0, len(req.Rules))
	for _, rule := range req.Rules {
		route.Spec.Rules = append(route.Spec.Rules, gatewayv1alpha2.TLSRouteRule{
			BackendRefs: convertBackendRefRequests(rule.BackendRefs),
		})
	}
}

// ─── TCPRoute conversions ───────────────────────────────────────────────────

// toTCPRouteResponse converts a Gateway API TCPRoute to the JSON response type.
func toTCPRouteResponse(route *gatewayv1alpha2.TCPRoute) TCPRouteResponse {
	resp := TCPRouteResponse{
		Name:      route.Name,
		Namespace: route.Namespace,
		CreatedAt: route.CreationTimestamp.UTC().Format("2006-01-02T15:04:05Z"),
	}

	resp.ParentRefs = convertParentRefsToResponse(route.Spec.ParentRefs)

	resp.Rules = make([]TCPRouteRuleResponse, 0, len(route.Spec.Rules))
	for _, rule := range route.Spec.Rules {
		rr := TCPRouteRuleResponse{
			BackendRefs: convertBackendRefsToResponse(rule.BackendRefs),
		}
		resp.Rules = append(resp.Rules, rr)
	}

	resp.Status = convertRouteStatusToResponse(route.Status.Parents)

	return resp
}

// toTCPRouteObject converts a create request into a Gateway API TCPRoute object.
func toTCPRouteObject(req CreateTCPRouteRequest) *gatewayv1alpha2.TCPRoute {
	route := &gatewayv1alpha2.TCPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
		Spec: gatewayv1alpha2.TCPRouteSpec{},
	}
	route.Spec.ParentRefs = convertParentRefRequests(req.ParentRefs)

	route.Spec.Rules = make([]gatewayv1alpha2.TCPRouteRule, 0, len(req.Rules))
	for _, rule := range req.Rules {
		route.Spec.Rules = append(route.Spec.Rules, gatewayv1alpha2.TCPRouteRule{
			BackendRefs: convertBackendRefRequests(rule.BackendRefs),
		})
	}

	return route
}

// applyUpdateToTCPRoute applies an update request to an existing TCPRoute.
func applyUpdateToTCPRoute(route *gatewayv1alpha2.TCPRoute, req UpdateTCPRouteRequest) {
	route.Spec.ParentRefs = convertParentRefRequests(req.ParentRefs)

	route.Spec.Rules = make([]gatewayv1alpha2.TCPRouteRule, 0, len(req.Rules))
	for _, rule := range req.Rules {
		route.Spec.Rules = append(route.Spec.Rules, gatewayv1alpha2.TCPRouteRule{
			BackendRefs: convertBackendRefRequests(rule.BackendRefs),
		})
	}
}

// ─── UDPRoute conversions ───────────────────────────────────────────────────

// toUDPRouteResponse converts a Gateway API UDPRoute to the JSON response type.
func toUDPRouteResponse(route *gatewayv1alpha2.UDPRoute) UDPRouteResponse {
	resp := UDPRouteResponse{
		Name:      route.Name,
		Namespace: route.Namespace,
		CreatedAt: route.CreationTimestamp.UTC().Format("2006-01-02T15:04:05Z"),
	}

	resp.ParentRefs = convertParentRefsToResponse(route.Spec.ParentRefs)

	resp.Rules = make([]UDPRouteRuleResponse, 0, len(route.Spec.Rules))
	for _, rule := range route.Spec.Rules {
		rr := UDPRouteRuleResponse{
			BackendRefs: convertBackendRefsToResponse(rule.BackendRefs),
		}
		resp.Rules = append(resp.Rules, rr)
	}

	resp.Status = convertRouteStatusToResponse(route.Status.Parents)

	return resp
}

// toUDPRouteObject converts a create request into a Gateway API UDPRoute object.
func toUDPRouteObject(req CreateUDPRouteRequest) *gatewayv1alpha2.UDPRoute {
	route := &gatewayv1alpha2.UDPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
		Spec: gatewayv1alpha2.UDPRouteSpec{},
	}
	route.Spec.ParentRefs = convertParentRefRequests(req.ParentRefs)

	route.Spec.Rules = make([]gatewayv1alpha2.UDPRouteRule, 0, len(req.Rules))
	for _, rule := range req.Rules {
		route.Spec.Rules = append(route.Spec.Rules, gatewayv1alpha2.UDPRouteRule{
			BackendRefs: convertBackendRefRequests(rule.BackendRefs),
		})
	}

	return route
}

// applyUpdateToUDPRoute applies an update request to an existing UDPRoute.
func applyUpdateToUDPRoute(route *gatewayv1alpha2.UDPRoute, req UpdateUDPRouteRequest) {
	route.Spec.ParentRefs = convertParentRefRequests(req.ParentRefs)

	route.Spec.Rules = make([]gatewayv1alpha2.UDPRouteRule, 0, len(req.Rules))
	for _, rule := range req.Rules {
		route.Spec.Rules = append(route.Spec.Rules, gatewayv1alpha2.UDPRouteRule{
			BackendRefs: convertBackendRefRequests(rule.BackendRefs),
		})
	}
}
