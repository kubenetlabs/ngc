package handlers

import (
	"encoding/json"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

// Response types matching frontend/src/types/*.ts

type ConditionResponse struct {
	Type               string `json:"type"`
	Status             string `json:"status"`
	Reason             string `json:"reason"`
	Message            string `json:"message"`
	LastTransitionTime string `json:"lastTransitionTime"`
}

type ListenerResponse struct {
	Name          string              `json:"name"`
	Hostname      *string             `json:"hostname,omitempty"`
	Port          int32               `json:"port"`
	Protocol      string              `json:"protocol"`
	TLS           *TLSResponse        `json:"tls,omitempty"`
	AllowedRoutes *AllowedRoutesResp  `json:"allowedRoutes,omitempty"`
}

type TLSResponse struct {
	Mode            string               `json:"mode"`
	CertificateRefs []CertificateRefResp `json:"certificateRefs"`
}

type CertificateRefResp struct {
	Group     string  `json:"group,omitempty"`
	Kind      string  `json:"kind,omitempty"`
	Name      string  `json:"name"`
	Namespace *string `json:"namespace,omitempty"`
}

type AllowedRoutesResp struct {
	Namespaces *NamespaceResp     `json:"namespaces,omitempty"`
	Kinds      []RouteGroupKind   `json:"kinds,omitempty"`
}

type NamespaceResp struct {
	From     string            `json:"from"`
	Selector map[string]string `json:"selector,omitempty"`
}

type RouteGroupKind struct {
	Group string `json:"group"`
	Kind  string `json:"kind"`
}

type ListenerStatusResponse struct {
	Name           string           `json:"name"`
	SupportedKinds []RouteGroupKind `json:"supportedKinds"`
	AttachedRoutes int32            `json:"attachedRoutes"`
	Conditions     []ConditionResponse `json:"conditions"`
}

type GatewayAddressResponse struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type GatewayStatusResponse struct {
	Conditions []ConditionResponse      `json:"conditions"`
	Listeners  []ListenerStatusResponse `json:"listeners"`
	Addresses  []GatewayAddressResponse `json:"addresses"`
}

type GatewayResponse struct {
	Name             string                 `json:"name"`
	Namespace        string                 `json:"namespace"`
	GatewayClassName string                 `json:"gatewayClassName"`
	Listeners        []ListenerResponse     `json:"listeners"`
	Labels           map[string]string      `json:"labels,omitempty"`
	Annotations      map[string]string      `json:"annotations,omitempty"`
	Status           *GatewayStatusResponse `json:"status,omitempty"`
	CreatedAt        string                 `json:"createdAt"`
}

type GatewayClassResponse struct {
	Name           string         `json:"name"`
	ControllerName string         `json:"controllerName"`
	Description    *string        `json:"description,omitempty"`
	ParametersRef  *ParamRefResp  `json:"parametersRef,omitempty"`
}

type ParamRefResp struct {
	Group     string  `json:"group"`
	Kind      string  `json:"kind"`
	Name      string  `json:"name"`
	Namespace *string `json:"namespace,omitempty"`
}

// HTTPRoute response types

type ParentRefResponse struct {
	Group       string  `json:"group,omitempty"`
	Kind        string  `json:"kind,omitempty"`
	Name        string  `json:"name"`
	Namespace   *string `json:"namespace,omitempty"`
	SectionName *string `json:"sectionName,omitempty"`
	Port        *int32  `json:"port,omitempty"`
}

type HTTPRouteMatchResponse struct {
	Path        *PathMatchResp          `json:"path,omitempty"`
	Headers     []HeaderMatchResp       `json:"headers,omitempty"`
	QueryParams []QueryParamMatchResp   `json:"queryParams,omitempty"`
	Method      *string                 `json:"method,omitempty"`
}

type PathMatchResp struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type HeaderMatchResp struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type QueryParamMatchResp struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type BackendRefResponse struct {
	Group     string  `json:"group,omitempty"`
	Kind      string  `json:"kind,omitempty"`
	Name      string  `json:"name"`
	Namespace *string `json:"namespace,omitempty"`
	Port      *int32  `json:"port,omitempty"`
	Weight    *int32  `json:"weight,omitempty"`
}

type HTTPRouteRuleResponse struct {
	Matches     []HTTPRouteMatchResponse `json:"matches,omitempty"`
	BackendRefs []BackendRefResponse     `json:"backendRefs,omitempty"`
}

type RouteParentStatusResponse struct {
	ParentRef      ParentRefResponse   `json:"parentRef"`
	ControllerName string              `json:"controllerName"`
	Conditions     []ConditionResponse `json:"conditions"`
}

type HTTPRouteStatusResponse struct {
	Parents []RouteParentStatusResponse `json:"parents"`
}

type HTTPRouteResponse struct {
	Name      string                    `json:"name"`
	Namespace string                    `json:"namespace"`
	ParentRefs []ParentRefResponse      `json:"parentRefs"`
	Hostnames  []string                `json:"hostnames,omitempty"`
	Rules      []HTTPRouteRuleResponse  `json:"rules"`
	Status     *HTTPRouteStatusResponse `json:"status,omitempty"`
	CreatedAt  string                   `json:"createdAt"`
}

// Conversion functions

func toGatewayResponse(gw *gatewayv1.Gateway) GatewayResponse {
	resp := GatewayResponse{
		Name:             gw.Name,
		Namespace:        gw.Namespace,
		GatewayClassName: string(gw.Spec.GatewayClassName),
		Listeners:        make([]ListenerResponse, 0, len(gw.Spec.Listeners)),
		Labels:           gw.Labels,
		Annotations:      gw.Annotations,
		CreatedAt:        gw.CreationTimestamp.UTC().Format("2006-01-02T15:04:05Z"),
	}

	for _, l := range gw.Spec.Listeners {
		lr := ListenerResponse{
			Name:     string(l.Name),
			Port:     int32(l.Port),
			Protocol: string(l.Protocol),
		}
		if l.Hostname != nil {
			h := string(*l.Hostname)
			lr.Hostname = &h
		}
		if l.TLS != nil {
			tls := &TLSResponse{
				CertificateRefs: make([]CertificateRefResp, 0, len(l.TLS.CertificateRefs)),
			}
			if l.TLS.Mode != nil {
				tls.Mode = string(*l.TLS.Mode)
			}
			for _, ref := range l.TLS.CertificateRefs {
				cr := CertificateRefResp{Name: string(ref.Name)}
				if ref.Group != nil {
					cr.Group = string(*ref.Group)
				}
				if ref.Kind != nil {
					cr.Kind = string(*ref.Kind)
				}
				if ref.Namespace != nil {
					ns := string(*ref.Namespace)
					cr.Namespace = &ns
				}
				tls.CertificateRefs = append(tls.CertificateRefs, cr)
			}
			lr.TLS = tls
		}
		resp.Listeners = append(resp.Listeners, lr)
	}

	if gw.Status.Conditions != nil || len(gw.Status.Listeners) > 0 || len(gw.Status.Addresses) > 0 {
		status := &GatewayStatusResponse{
			Conditions: convertConditions(gw.Status.Conditions),
			Listeners:  make([]ListenerStatusResponse, 0, len(gw.Status.Listeners)),
			Addresses:  make([]GatewayAddressResponse, 0, len(gw.Status.Addresses)),
		}
		for _, ls := range gw.Status.Listeners {
			lsr := ListenerStatusResponse{
				Name:           string(ls.Name),
				AttachedRoutes: ls.AttachedRoutes,
				SupportedKinds: make([]RouteGroupKind, 0, len(ls.SupportedKinds)),
				Conditions:     convertConditions(ls.Conditions),
			}
			for _, sk := range ls.SupportedKinds {
				rgk := RouteGroupKind{Kind: string(sk.Kind)}
				if sk.Group != nil {
					rgk.Group = string(*sk.Group)
				}
				lsr.SupportedKinds = append(lsr.SupportedKinds, rgk)
			}
			status.Listeners = append(status.Listeners, lsr)
		}
		for _, a := range gw.Status.Addresses {
			ar := GatewayAddressResponse{Value: a.Value}
			if a.Type != nil {
				ar.Type = string(*a.Type)
			}
			status.Addresses = append(status.Addresses, ar)
		}
		resp.Status = status
	}

	return resp
}

func toGatewayClassResponse(gc *gatewayv1.GatewayClass) GatewayClassResponse {
	resp := GatewayClassResponse{
		Name:           gc.Name,
		ControllerName: string(gc.Spec.ControllerName),
	}
	if gc.Spec.Description != nil {
		resp.Description = gc.Spec.Description
	}
	if gc.Spec.ParametersRef != nil {
		pr := &ParamRefResp{
			Group: string(gc.Spec.ParametersRef.Group),
			Kind:  string(gc.Spec.ParametersRef.Kind),
			Name:  gc.Spec.ParametersRef.Name,
		}
		if gc.Spec.ParametersRef.Namespace != nil {
			ns := string(*gc.Spec.ParametersRef.Namespace)
			pr.Namespace = &ns
		}
		resp.ParametersRef = pr
	}
	return resp
}

func toHTTPRouteResponse(hr *gatewayv1.HTTPRoute) HTTPRouteResponse {
	resp := HTTPRouteResponse{
		Name:      hr.Name,
		Namespace: hr.Namespace,
		CreatedAt: hr.CreationTimestamp.UTC().Format("2006-01-02T15:04:05Z"),
	}

	// Parent refs
	for _, pr := range hr.Spec.ParentRefs {
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
		resp.ParentRefs = append(resp.ParentRefs, pResp)
	}

	// Hostnames
	for _, h := range hr.Spec.Hostnames {
		resp.Hostnames = append(resp.Hostnames, string(h))
	}

	// Rules
	for _, rule := range hr.Spec.Rules {
		rr := HTTPRouteRuleResponse{}
		for _, m := range rule.Matches {
			mr := HTTPRouteMatchResponse{}
			if m.Path != nil {
				pm := &PathMatchResp{Value: *m.Path.Value}
				if m.Path.Type != nil {
					pm.Type = string(*m.Path.Type)
				}
				mr.Path = pm
			}
			if m.Method != nil {
				method := string(*m.Method)
				mr.Method = &method
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
	if hr.Status.Parents != nil {
		status := &HTTPRouteStatusResponse{}
		for _, ps := range hr.Status.Parents {
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
		resp.Status = status
	}

	return resp
}

func convertConditions(conditions []metav1.Condition) []ConditionResponse {
	result := make([]ConditionResponse, 0, len(conditions))
	for _, c := range conditions {
		result = append(result, ConditionResponse{
			Type:               c.Type,
			Status:             string(c.Status),
			Reason:             c.Reason,
			Message:            c.Message,
			LastTransitionTime: c.LastTransitionTime.UTC().Format("2006-01-02T15:04:05Z"),
		})
	}
	return result
}

// Request types for Gateway CRUD

type CreateGatewayRequest struct {
	Name             string            `json:"name"`
	Namespace        string            `json:"namespace"`
	GatewayClassName string            `json:"gatewayClassName"`
	Listeners        []ListenerRequest `json:"listeners"`
	Labels           map[string]string `json:"labels,omitempty"`
	Annotations      map[string]string `json:"annotations,omitempty"`
}

type UpdateGatewayRequest struct {
	GatewayClassName string            `json:"gatewayClassName"`
	Listeners        []ListenerRequest `json:"listeners"`
	Labels           map[string]string `json:"labels,omitempty"`
	Annotations      map[string]string `json:"annotations,omitempty"`
}

type ListenerRequest struct {
	Name     string  `json:"name"`
	Port     int32   `json:"port"`
	Protocol string  `json:"protocol"`
	Hostname *string `json:"hostname,omitempty"`
}

func toGatewayObject(req CreateGatewayRequest) *gatewayv1.Gateway {
	gw := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:        req.Name,
			Namespace:   req.Namespace,
			Labels:      req.Labels,
			Annotations: req.Annotations,
		},
		Spec: gatewayv1.GatewaySpec{
			GatewayClassName: gatewayv1.ObjectName(req.GatewayClassName),
			Listeners:        make([]gatewayv1.Listener, 0, len(req.Listeners)),
		},
	}
	for _, l := range req.Listeners {
		listener := gatewayv1.Listener{
			Name:     gatewayv1.SectionName(l.Name),
			Port:     gatewayv1.PortNumber(l.Port),
			Protocol: gatewayv1.ProtocolType(l.Protocol),
		}
		if l.Hostname != nil {
			h := gatewayv1.Hostname(*l.Hostname)
			listener.Hostname = &h
		}
		gw.Spec.Listeners = append(gw.Spec.Listeners, listener)
	}
	return gw
}

func applyUpdateToGateway(gw *gatewayv1.Gateway, req UpdateGatewayRequest) {
	gw.Spec.GatewayClassName = gatewayv1.ObjectName(req.GatewayClassName)
	gw.Labels = req.Labels
	gw.Annotations = req.Annotations
	gw.Spec.Listeners = make([]gatewayv1.Listener, 0, len(req.Listeners))
	for _, l := range req.Listeners {
		listener := gatewayv1.Listener{
			Name:     gatewayv1.SectionName(l.Name),
			Port:     gatewayv1.PortNumber(l.Port),
			Protocol: gatewayv1.ProtocolType(l.Protocol),
		}
		if l.Hostname != nil {
			h := gatewayv1.Hostname(*l.Hostname)
			listener.Hostname = &h
		}
		gw.Spec.Listeners = append(gw.Spec.Listeners, listener)
	}
}

// Request types for HTTPRoute CRUD

type CreateHTTPRouteRequest struct {
	Name       string              `json:"name"`
	Namespace  string              `json:"namespace"`
	ParentRefs []ParentRefRequest  `json:"parentRefs"`
	Hostnames  []string            `json:"hostnames,omitempty"`
	Rules      []HTTPRouteRuleReq  `json:"rules"`
}

type UpdateHTTPRouteRequest struct {
	ParentRefs []ParentRefRequest  `json:"parentRefs"`
	Hostnames  []string            `json:"hostnames,omitempty"`
	Rules      []HTTPRouteRuleReq  `json:"rules"`
}

type ParentRefRequest struct {
	Name        string  `json:"name"`
	Namespace   *string `json:"namespace,omitempty"`
	SectionName *string `json:"sectionName,omitempty"`
}

type HTTPRouteRuleReq struct {
	Matches     []HTTPRouteMatchRequest `json:"matches,omitempty"`
	BackendRefs []BackendRefRequest     `json:"backendRefs,omitempty"`
}

type HTTPRouteMatchRequest struct {
	Path    *PathMatchRequest    `json:"path,omitempty"`
	Headers []HeaderMatchRequest `json:"headers,omitempty"`
	Method  *string              `json:"method,omitempty"`
}

type PathMatchRequest struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type HeaderMatchRequest struct {
	Type  string `json:"type"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type BackendRefRequest struct {
	Name      string  `json:"name"`
	Namespace *string `json:"namespace,omitempty"`
	Port      *int32  `json:"port,omitempty"`
	Weight    *int32  `json:"weight,omitempty"`
}

func toHTTPRouteObject(req CreateHTTPRouteRequest) *gatewayv1.HTTPRoute {
	hr := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
		Spec: gatewayv1.HTTPRouteSpec{},
	}
	hr.Spec.ParentRefs = convertParentRefRequests(req.ParentRefs)
	for _, h := range req.Hostnames {
		hr.Spec.Hostnames = append(hr.Spec.Hostnames, gatewayv1.Hostname(h))
	}
	hr.Spec.Rules = convertHTTPRouteRuleRequests(req.Rules)
	return hr
}

func applyUpdateToHTTPRoute(hr *gatewayv1.HTTPRoute, req UpdateHTTPRouteRequest) {
	hr.Spec.ParentRefs = convertParentRefRequests(req.ParentRefs)
	hr.Spec.Hostnames = nil
	for _, h := range req.Hostnames {
		hr.Spec.Hostnames = append(hr.Spec.Hostnames, gatewayv1.Hostname(h))
	}
	hr.Spec.Rules = convertHTTPRouteRuleRequests(req.Rules)
}

func convertParentRefRequests(refs []ParentRefRequest) []gatewayv1.ParentReference {
	group := gatewayv1.Group("gateway.networking.k8s.io")
	kind := gatewayv1.Kind("Gateway")
	result := make([]gatewayv1.ParentReference, 0, len(refs))
	for _, r := range refs {
		pr := gatewayv1.ParentReference{
			Group: &group,
			Kind:  &kind,
			Name:  gatewayv1.ObjectName(r.Name),
		}
		if r.Namespace != nil {
			ns := gatewayv1.Namespace(*r.Namespace)
			pr.Namespace = &ns
		}
		if r.SectionName != nil {
			sn := gatewayv1.SectionName(*r.SectionName)
			pr.SectionName = &sn
		}
		result = append(result, pr)
	}
	return result
}

func convertHTTPRouteRuleRequests(rules []HTTPRouteRuleReq) []gatewayv1.HTTPRouteRule {
	result := make([]gatewayv1.HTTPRouteRule, 0, len(rules))
	for _, rule := range rules {
		r := gatewayv1.HTTPRouteRule{}
		for _, m := range rule.Matches {
			match := gatewayv1.HTTPRouteMatch{}
			if m.Path != nil {
				pathType := gatewayv1.PathMatchType(m.Path.Type)
				match.Path = &gatewayv1.HTTPPathMatch{
					Type:  &pathType,
					Value: &m.Path.Value,
				}
			}
			if m.Method != nil {
				method := gatewayv1.HTTPMethod(*m.Method)
				match.Method = &method
			}
			for _, h := range m.Headers {
				headerType := gatewayv1.HeaderMatchType(h.Type)
				match.Headers = append(match.Headers, gatewayv1.HTTPHeaderMatch{
					Type:  &headerType,
					Name:  gatewayv1.HTTPHeaderName(h.Name),
					Value: h.Value,
				})
			}
			r.Matches = append(r.Matches, match)
		}
		for _, br := range rule.BackendRefs {
			backendRef := gatewayv1.HTTPBackendRef{
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

// Shared response helpers

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
