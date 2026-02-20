package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
	"github.com/kubenetlabs/ngc/api/internal/database"
)

// RouteHandler handles HTTPRoute, GRPCRoute, TLSRoute, TCPRoute, and UDPRoute API requests.
type RouteHandler struct {
	Store database.Store
}

// List returns all HTTPRoutes, optionally filtered by ?namespace= query param.
func (h *RouteHandler) List(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	ns := r.URL.Query().Get("namespace")
	routes, err := k8s.ListHTTPRoutes(r.Context(), ns)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp := make([]HTTPRouteResponse, 0, len(routes))
	for i := range routes {
		resp = append(resp, toHTTPRouteResponse(&routes[i]))
	}
	writeJSON(w, http.StatusOK, resp)
}

// Get returns a single HTTPRoute by namespace and name.
func (h *RouteHandler) Get(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	ns := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	hr, err := k8s.GetHTTPRoute(r.Context(), ns, name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toHTTPRouteResponse(hr))
}

// Create creates a new HTTPRoute.
func (h *RouteHandler) Create(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	var req CreateHTTPRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.Name == "" || req.Namespace == "" || len(req.ParentRefs) == 0 || len(req.Rules) == 0 {
		writeError(w, http.StatusBadRequest, "name, namespace, at least one parentRef, and at least one rule are required")
		return
	}

	hr := toHTTPRouteObject(req)
	created, err := k8s.CreateHTTPRoute(r.Context(), hr)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	resp := toHTTPRouteResponse(created)
	auditLog(h.Store, r.Context(), "create", "HTTPRoute", req.Name, req.Namespace, nil, resp)
	writeJSON(w, http.StatusCreated, resp)
}

// Update modifies an existing HTTPRoute.
func (h *RouteHandler) Update(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	ns := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	var req UpdateHTTPRouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	existing, err := k8s.GetHTTPRoute(r.Context(), ns, name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	beforeResp := toHTTPRouteResponse(existing)
	applyUpdateToHTTPRoute(existing, req)

	updated, err := k8s.UpdateHTTPRoute(r.Context(), existing)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	afterResp := toHTTPRouteResponse(updated)
	auditLog(h.Store, r.Context(), "update", "HTTPRoute", name, ns, beforeResp, afterResp)
	writeJSON(w, http.StatusOK, afterResp)
}

// Delete removes an HTTPRoute.
func (h *RouteHandler) Delete(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	ns := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	if err := k8s.DeleteHTTPRoute(r.Context(), ns, name); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	auditLog(h.Store, r.Context(), "delete", "HTTPRoute", name, ns, map[string]string{"name": name, "namespace": ns}, nil)
	writeJSON(w, http.StatusOK, map[string]string{"message": "httproute deleted", "name": name, "namespace": ns})
}

// Simulate performs a dry-run simulation of route matching.
func (h *RouteHandler) Simulate(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	ns := chi.URLParam(r, "namespace")
	name := chi.URLParam(r, "name")

	hr, err := k8s.GetHTTPRoute(r.Context(), ns, name)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}

	var req SimulateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	resp := SimulateResponse{
		Matched:      false,
		MatchedRule:  -1,
		MatchDetails: make([]SimulateMatchDetail, 0, len(hr.Spec.Rules)),
	}

	for ruleIdx, rule := range hr.Spec.Rules {
		detail := SimulateMatchDetail{
			RuleIndex: ruleIdx,
			Matched:   false,
		}

		// A rule with no matches matches everything.
		if len(rule.Matches) == 0 {
			detail.Matched = true
			detail.Reason = "rule has no match conditions (matches all requests)"
		} else {
			// OR logic: any single match block satisfying all its conditions is enough.
			var reasons []string
			for _, m := range rule.Matches {
				matched, reason := evaluateMatch(m, req)
				if matched {
					detail.Matched = true
					detail.Reason = reason
					break
				}
				reasons = append(reasons, reason)
			}
			if !detail.Matched {
				detail.Reason = strings.Join(reasons, "; ")
			}
		}

		resp.MatchDetails = append(resp.MatchDetails, detail)

		// Return the first matching rule's backends.
		if detail.Matched && !resp.Matched {
			resp.Matched = true
			resp.MatchedRule = ruleIdx
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
				resp.Backends = append(resp.Backends, brr)
			}
		}
	}

	writeJSON(w, http.StatusOK, resp)
}

// evaluateMatch checks whether a single HTTPRouteMatch block matches the simulated request.
// All conditions within the match must be satisfied (AND logic).
func evaluateMatch(m gatewayv1.HTTPRouteMatch, req SimulateRequest) (bool, string) {
	// Path matching
	if m.Path != nil && m.Path.Value != nil {
		pathType := "PathPrefix"
		if m.Path.Type != nil {
			pathType = string(*m.Path.Type)
		}
		pathValue := *m.Path.Value

		switch pathType {
		case "Exact":
			if req.Path != pathValue {
				return false, fmt.Sprintf("path %q does not exactly match %q", req.Path, pathValue)
			}
		case "PathPrefix":
			if !strings.HasPrefix(req.Path, pathValue) {
				return false, fmt.Sprintf("path %q does not have prefix %q", req.Path, pathValue)
			}
		case "RegularExpression":
			matched, err := regexp.MatchString(pathValue, req.Path)
			if err != nil {
				return false, fmt.Sprintf("invalid path regex %q: %v", pathValue, err)
			}
			if !matched {
				return false, fmt.Sprintf("path %q does not match regex %q", req.Path, pathValue)
			}
		}
	}

	// Method matching
	if m.Method != nil {
		if req.Method != string(*m.Method) {
			return false, fmt.Sprintf("method %q does not match %q", req.Method, string(*m.Method))
		}
	}

	// Header matching
	for _, hm := range m.Headers {
		headerType := "Exact"
		if hm.Type != nil {
			headerType = string(*hm.Type)
		}
		headerName := string(hm.Name)
		expectedValue := hm.Value
		actualValue, ok := req.Headers[headerName]
		if !ok {
			// Also try case-insensitive lookup since HTTP headers are case-insensitive.
			for k, v := range req.Headers {
				if strings.EqualFold(k, headerName) {
					actualValue = v
					ok = true
					break
				}
			}
		}
		if !ok {
			return false, fmt.Sprintf("header %q not present in request", headerName)
		}

		switch headerType {
		case "Exact":
			if actualValue != expectedValue {
				return false, fmt.Sprintf("header %q value %q does not match %q", headerName, actualValue, expectedValue)
			}
		case "RegularExpression":
			matched, err := regexp.MatchString(expectedValue, actualValue)
			if err != nil {
				return false, fmt.Sprintf("invalid header regex %q: %v", expectedValue, err)
			}
			if !matched {
				return false, fmt.Sprintf("header %q value %q does not match regex %q", headerName, actualValue, expectedValue)
			}
		}
	}

	return true, "all conditions matched"
}

// SimulateRequest is the request body for route simulation.
type SimulateRequest struct {
	Method  string            `json:"method"`
	Path    string            `json:"path"`
	Host    string            `json:"host"`
	Headers map[string]string `json:"headers,omitempty"`
}

// SimulateResponse is the response for route simulation.
type SimulateResponse struct {
	Matched      bool                  `json:"matched"`
	MatchedRule  int                   `json:"matchedRule"`
	MatchDetails []SimulateMatchDetail `json:"matchDetails"`
	Backends     []BackendRefResponse  `json:"backends,omitempty"`
}

// SimulateMatchDetail describes the match result of a single rule.
type SimulateMatchDetail struct {
	RuleIndex int    `json:"ruleIndex"`
	Matched   bool   `json:"matched"`
	Reason    string `json:"reason"`
}
