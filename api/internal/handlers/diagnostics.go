package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
)

// DiagnosticsHandler handles diagnostic API requests.
type DiagnosticsHandler struct{}

// RouteCheck types

type RouteCheckRequest struct {
	Namespace string `json:"namespace"`
	RouteName string `json:"routeName"`
	RouteKind string `json:"routeKind"` // "HTTPRoute", "GRPCRoute", etc
}

type RouteCheckResponse struct {
	Route     string            `json:"route"`
	Namespace string            `json:"namespace"`
	Status    string            `json:"status"` // "healthy", "degraded", "unhealthy"
	Checks    []DiagnosticCheck `json:"checks"`
}

type DiagnosticCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "pass", "fail", "warn", "skip"
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Trace types

type TraceRequest struct {
	Method  string            `json:"method"`
	Host    string            `json:"host"`
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers,omitempty"`
}

type TraceResponse struct {
	Request      TraceRequest `json:"request"`
	Steps        []TraceStep  `json:"steps"`
	Matched      bool         `json:"matched"`
	MatchedRoute *string      `json:"matchedRoute,omitempty"`
}

type TraceStep struct {
	Name       string  `json:"name"`
	Status     string  `json:"status"` // "pass", "fail", "skip"
	Message    string  `json:"message"`
	DurationMs float64 `json:"durationMs"`
}

// RouteCheck validates route configuration and detects issues.
func (h *DiagnosticsHandler) RouteCheck(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	var req RouteCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.Namespace == "" || req.RouteName == "" {
		writeError(w, http.StatusBadRequest, "namespace and routeName are required")
		return
	}

	if req.RouteKind == "" {
		req.RouteKind = "HTTPRoute"
	}

	ctx := r.Context()
	checks := make([]DiagnosticCheck, 0, 6)

	// Currently only HTTPRoute is supported for detailed checks.
	if req.RouteKind != "HTTPRoute" {
		checks = append(checks, DiagnosticCheck{
			Name:    "Route Exists",
			Status:  "skip",
			Message: fmt.Sprintf("Route kind %q is not yet supported for diagnostics", req.RouteKind),
		})
		writeJSON(w, http.StatusOK, RouteCheckResponse{
			Route:     req.RouteName,
			Namespace: req.Namespace,
			Status:    "unhealthy",
			Checks:    checks,
		})
		return
	}

	// Check 1: Route Exists
	route, err := k8s.GetHTTPRoute(ctx, req.Namespace, req.RouteName)
	if err != nil {
		checks = append(checks, DiagnosticCheck{
			Name:    "Route Exists",
			Status:  "fail",
			Message: fmt.Sprintf("HTTPRoute %s/%s not found", req.Namespace, req.RouteName),
			Details: err.Error(),
		})
		// All remaining checks are skip
		for _, name := range []string{"Parent Gateway Attached", "Listener Match", "Backend Health", "Route Accepted", "Route Resolved"} {
			checks = append(checks, DiagnosticCheck{
				Name:    name,
				Status:  "skip",
				Message: "Skipped because route does not exist",
			})
		}
		writeJSON(w, http.StatusOK, RouteCheckResponse{
			Route:     req.RouteName,
			Namespace: req.Namespace,
			Status:    "unhealthy",
			Checks:    checks,
		})
		return
	}

	checks = append(checks, DiagnosticCheck{
		Name:    "Route Exists",
		Status:  "pass",
		Message: fmt.Sprintf("HTTPRoute %s/%s exists", req.Namespace, req.RouteName),
	})

	// Check 2: Parent Gateway Attached
	parentGatewayCheck := DiagnosticCheck{
		Name: "Parent Gateway Attached",
	}
	var parentGateway *gatewayv1.Gateway
	if len(route.Spec.ParentRefs) == 0 {
		parentGatewayCheck.Status = "fail"
		parentGatewayCheck.Message = "No parentRefs defined on route"
	} else {
		allFound := true
		var details []string
		for _, ref := range route.Spec.ParentRefs {
			gwName := string(ref.Name)
			gwNamespace := req.Namespace
			if ref.Namespace != nil {
				gwNamespace = string(*ref.Namespace)
			}
			gw, gwErr := k8s.GetGateway(ctx, gwNamespace, gwName)
			if gwErr != nil {
				allFound = false
				details = append(details, fmt.Sprintf("Gateway %s/%s not found: %v", gwNamespace, gwName, gwErr))
			} else if parentGateway == nil {
				parentGateway = gw
			}
		}
		if allFound {
			parentGatewayCheck.Status = "pass"
			parentGatewayCheck.Message = fmt.Sprintf("All %d parent gateway(s) found", len(route.Spec.ParentRefs))
		} else {
			parentGatewayCheck.Status = "fail"
			parentGatewayCheck.Message = "One or more parent gateways not found"
			parentGatewayCheck.Details = strings.Join(details, "; ")
		}
	}
	checks = append(checks, parentGatewayCheck)

	// Check 3: Listener Match
	listenerCheck := DiagnosticCheck{
		Name: "Listener Match",
	}
	if parentGateway == nil {
		listenerCheck.Status = "skip"
		listenerCheck.Message = "Skipped because no parent gateway was found"
	} else {
		matched := false
		for _, listener := range parentGateway.Spec.Listeners {
			// Check protocol compatibility (HTTP/HTTPS for HTTPRoute)
			if listener.Protocol == gatewayv1.HTTPProtocolType || listener.Protocol == gatewayv1.HTTPSProtocolType {
				// Check hostname overlap if both specify hostnames
				if listener.Hostname != nil && len(route.Spec.Hostnames) > 0 {
					for _, routeHostname := range route.Spec.Hostnames {
						if hostnamesMatch(string(*listener.Hostname), string(routeHostname)) {
							matched = true
							break
						}
					}
				} else {
					// No hostname constraint or route has no hostnames — matches
					matched = true
				}
			}
			if matched {
				break
			}
		}
		if matched {
			listenerCheck.Status = "pass"
			listenerCheck.Message = "Gateway has a compatible listener for this route"
		} else {
			listenerCheck.Status = "warn"
			listenerCheck.Message = "No gateway listener matches the route's protocol/hostname"
			listenerCheck.Details = fmt.Sprintf("Gateway %s/%s has %d listener(s), route has %d hostname(s)",
				parentGateway.Namespace, parentGateway.Name,
				len(parentGateway.Spec.Listeners), len(route.Spec.Hostnames))
		}
	}
	checks = append(checks, listenerCheck)

	// Check 4: Backend Health
	backendCheck := DiagnosticCheck{
		Name: "Backend Health",
	}
	backendNames := collectBackendNames(route)
	if len(backendNames) == 0 {
		backendCheck.Status = "warn"
		backendCheck.Message = "No backendRefs defined in route rules"
	} else {
		services, svcErr := k8s.ListServices(ctx, req.Namespace)
		if svcErr != nil {
			backendCheck.Status = "fail"
			backendCheck.Message = "Failed to list services"
			backendCheck.Details = svcErr.Error()
		} else {
			svcSet := make(map[string]bool, len(services))
			for _, svc := range services {
				svcSet[svc.Name] = true
			}
			var missing []string
			for _, name := range backendNames {
				if !svcSet[name] {
					missing = append(missing, name)
				}
			}
			if len(missing) == 0 {
				backendCheck.Status = "pass"
				backendCheck.Message = fmt.Sprintf("All %d backend service(s) found", len(backendNames))
			} else {
				backendCheck.Status = "fail"
				backendCheck.Message = fmt.Sprintf("%d of %d backend service(s) missing", len(missing), len(backendNames))
				backendCheck.Details = "Missing: " + strings.Join(missing, ", ")
			}
		}
	}
	checks = append(checks, backendCheck)

	// Check 5: Route Accepted
	acceptedCheck := DiagnosticCheck{
		Name: "Route Accepted",
	}
	acceptedStatus := findRouteCondition(route, "Accepted")
	if acceptedStatus == "" {
		acceptedCheck.Status = "warn"
		acceptedCheck.Message = "No Accepted condition found in route status"
	} else if acceptedStatus == string(metav1.ConditionTrue) {
		acceptedCheck.Status = "pass"
		acceptedCheck.Message = "Route has Accepted=True condition"
	} else {
		acceptedCheck.Status = "fail"
		acceptedCheck.Message = fmt.Sprintf("Route Accepted condition is %s", acceptedStatus)
	}
	checks = append(checks, acceptedCheck)

	// Check 6: Route Resolved
	resolvedCheck := DiagnosticCheck{
		Name: "Route Resolved",
	}
	resolvedStatus := findRouteCondition(route, "ResolvedRefs")
	if resolvedStatus == "" {
		resolvedCheck.Status = "warn"
		resolvedCheck.Message = "No ResolvedRefs condition found in route status"
	} else if resolvedStatus == string(metav1.ConditionTrue) {
		resolvedCheck.Status = "pass"
		resolvedCheck.Message = "Route has ResolvedRefs=True condition"
	} else {
		resolvedCheck.Status = "fail"
		resolvedCheck.Message = fmt.Sprintf("Route ResolvedRefs condition is %s", resolvedStatus)
	}
	checks = append(checks, resolvedCheck)

	// Determine overall status
	overallStatus := "healthy"
	for _, c := range checks {
		if c.Status == "fail" {
			overallStatus = "unhealthy"
			break
		}
		if c.Status == "warn" {
			overallStatus = "degraded"
		}
	}

	writeJSON(w, http.StatusOK, RouteCheckResponse{
		Route:     req.RouteName,
		Namespace: req.Namespace,
		Status:    overallStatus,
		Checks:    checks,
	})
}

// Trace performs a request trace through the gateway routing pipeline.
func (h *DiagnosticsHandler) Trace(w http.ResponseWriter, r *http.Request) {
	k8s := cluster.ClientFromContext(r.Context())
	if k8s == nil {
		writeError(w, http.StatusServiceUnavailable, "no cluster context")
		return
	}

	var req TraceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	if req.Host == "" || req.Path == "" {
		writeError(w, http.StatusBadRequest, "host and path are required")
		return
	}

	if req.Method == "" {
		req.Method = "GET"
	}

	ctx := r.Context()
	steps := make([]TraceStep, 0, 4)
	matched := false
	var matchedRouteName *string

	// Step 1: Gateway Listener Selection
	gateways, err := k8s.ListGateways(ctx, "")
	step1 := TraceStep{
		Name:       "Gateway Listener Selection",
		DurationMs: 0.5,
	}
	if err != nil {
		step1.Status = "fail"
		step1.Message = "Failed to list gateways: " + err.Error()
		steps = append(steps, step1)
		writeJSON(w, http.StatusOK, TraceResponse{
			Request: req,
			Steps:   steps,
			Matched: false,
		})
		return
	}

	var matchedGateways []gatewayv1.Gateway
	for _, gw := range gateways {
		for _, listener := range gw.Spec.Listeners {
			if listener.Protocol != gatewayv1.HTTPProtocolType && listener.Protocol != gatewayv1.HTTPSProtocolType {
				continue
			}
			if listener.Hostname == nil || hostnamesMatch(string(*listener.Hostname), req.Host) {
				matchedGateways = append(matchedGateways, gw)
				break
			}
		}
	}

	if len(matchedGateways) == 0 {
		step1.Status = "fail"
		step1.Message = fmt.Sprintf("No gateway listener matches host %q (checked %d gateway(s))", req.Host, len(gateways))
		steps = append(steps, step1)
		// Remaining steps are skip
		steps = append(steps,
			TraceStep{Name: "Route Matching", Status: "skip", Message: "Skipped: no matching gateway", DurationMs: 0},
			TraceStep{Name: "Rule Matching", Status: "skip", Message: "Skipped: no matching gateway", DurationMs: 0},
			TraceStep{Name: "Backend Selection", Status: "skip", Message: "Skipped: no matching gateway", DurationMs: 0},
		)
		writeJSON(w, http.StatusOK, TraceResponse{
			Request: req,
			Steps:   steps,
			Matched: false,
		})
		return
	}

	gwNames := make([]string, 0, len(matchedGateways))
	for _, gw := range matchedGateways {
		gwNames = append(gwNames, gw.Namespace+"/"+gw.Name)
	}
	step1.Status = "pass"
	step1.Message = fmt.Sprintf("Matched %d gateway(s): %s", len(matchedGateways), strings.Join(gwNames, ", "))
	steps = append(steps, step1)

	// Step 2: Route Matching
	allRoutes, err := k8s.ListHTTPRoutes(ctx, "")
	step2 := TraceStep{
		Name:       "Route Matching",
		DurationMs: 0.3,
	}
	if err != nil {
		step2.Status = "fail"
		step2.Message = "Failed to list HTTPRoutes: " + err.Error()
		steps = append(steps, step2)
		steps = append(steps,
			TraceStep{Name: "Rule Matching", Status: "skip", Message: "Skipped: route listing failed", DurationMs: 0},
			TraceStep{Name: "Backend Selection", Status: "skip", Message: "Skipped: route listing failed", DurationMs: 0},
		)
		writeJSON(w, http.StatusOK, TraceResponse{
			Request: req,
			Steps:   steps,
			Matched: false,
		})
		return
	}

	// Build a set of matched gateway keys for parent ref checking
	gwKeySet := make(map[string]bool, len(matchedGateways))
	for _, gw := range matchedGateways {
		gwKeySet[gw.Namespace+"/"+gw.Name] = true
	}

	var matchedRoutes []gatewayv1.HTTPRoute
	for _, route := range allRoutes {
		// Check that the route references one of the matched gateways
		parentMatches := false
		for _, ref := range route.Spec.ParentRefs {
			refNS := route.Namespace
			if ref.Namespace != nil {
				refNS = string(*ref.Namespace)
			}
			if gwKeySet[refNS+"/"+string(ref.Name)] {
				parentMatches = true
				break
			}
		}
		if !parentMatches {
			continue
		}

		// Check hostname match
		if len(route.Spec.Hostnames) > 0 {
			hostMatch := false
			for _, h := range route.Spec.Hostnames {
				if hostnamesMatch(string(h), req.Host) {
					hostMatch = true
					break
				}
			}
			if !hostMatch {
				continue
			}
		}

		matchedRoutes = append(matchedRoutes, route)
	}

	if len(matchedRoutes) == 0 {
		step2.Status = "fail"
		step2.Message = fmt.Sprintf("No HTTPRoute matches host %q (checked %d route(s))", req.Host, len(allRoutes))
		steps = append(steps, step2)
		steps = append(steps,
			TraceStep{Name: "Rule Matching", Status: "skip", Message: "Skipped: no matching route", DurationMs: 0},
			TraceStep{Name: "Backend Selection", Status: "skip", Message: "Skipped: no matching route", DurationMs: 0},
		)
		writeJSON(w, http.StatusOK, TraceResponse{
			Request: req,
			Steps:   steps,
			Matched: false,
		})
		return
	}

	routeNames := make([]string, 0, len(matchedRoutes))
	for _, rt := range matchedRoutes {
		routeNames = append(routeNames, rt.Namespace+"/"+rt.Name)
	}
	step2.Status = "pass"
	step2.Message = fmt.Sprintf("Matched %d route(s): %s", len(matchedRoutes), strings.Join(routeNames, ", "))
	steps = append(steps, step2)

	// Step 3: Rule Matching
	step3 := TraceStep{
		Name:       "Rule Matching",
		DurationMs: 0.2,
	}

	type ruleMatch struct {
		routeKey string
		rule     gatewayv1.HTTPRouteRule
	}
	var ruleMatches []ruleMatch

	for _, route := range matchedRoutes {
		for _, rule := range route.Spec.Rules {
			if ruleMatchesRequest(rule, req) {
				ruleMatches = append(ruleMatches, ruleMatch{
					routeKey: route.Namespace + "/" + route.Name,
					rule:     rule,
				})
			}
		}
	}

	if len(ruleMatches) == 0 {
		step3.Status = "fail"
		step3.Message = fmt.Sprintf("No rule matches %s %s (checked rules across %d route(s))", req.Method, req.Path, len(matchedRoutes))
		steps = append(steps, step3)
		steps = append(steps,
			TraceStep{Name: "Backend Selection", Status: "skip", Message: "Skipped: no matching rule", DurationMs: 0},
		)
		writeJSON(w, http.StatusOK, TraceResponse{
			Request: req,
			Steps:   steps,
			Matched: false,
		})
		return
	}

	step3.Status = "pass"
	step3.Message = fmt.Sprintf("Matched %d rule(s) in route %s for %s %s", len(ruleMatches), ruleMatches[0].routeKey, req.Method, req.Path)
	steps = append(steps, step3)

	// Step 4: Backend Selection
	step4 := TraceStep{
		Name:       "Backend Selection",
		DurationMs: 0.1,
	}

	bestMatch := ruleMatches[0]
	if len(bestMatch.rule.BackendRefs) == 0 {
		step4.Status = "fail"
		step4.Message = "Matched rule has no backendRefs"
		steps = append(steps, step4)
		routeName := bestMatch.routeKey
		writeJSON(w, http.StatusOK, TraceResponse{
			Request:      req,
			Steps:        steps,
			Matched:      false,
			MatchedRoute: &routeName,
		})
		return
	}

	backendNames := make([]string, 0, len(bestMatch.rule.BackendRefs))
	for _, br := range bestMatch.rule.BackendRefs {
		name := string(br.Name)
		if br.Port != nil {
			name = fmt.Sprintf("%s:%d", name, *br.Port)
		}
		if br.Weight != nil {
			name = fmt.Sprintf("%s (weight=%d)", name, *br.Weight)
		}
		backendNames = append(backendNames, name)
	}

	step4.Status = "pass"
	step4.Message = fmt.Sprintf("Selected backend(s): %s", strings.Join(backendNames, ", "))
	steps = append(steps, step4)

	matched = true
	routeName := bestMatch.routeKey
	matchedRouteName = &routeName

	writeJSON(w, http.StatusOK, TraceResponse{
		Request:      req,
		Steps:        steps,
		Matched:      matched,
		MatchedRoute: matchedRouteName,
	})
}

// hostnamesMatch checks if a listener hostname pattern matches a request hostname.
// Supports wildcard prefixes like *.example.com.
func hostnamesMatch(pattern, hostname string) bool {
	pattern = strings.ToLower(pattern)
	hostname = strings.ToLower(hostname)

	if pattern == hostname {
		return true
	}

	// Wildcard match: *.example.com matches foo.example.com
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[1:] // ".example.com"
		if strings.HasSuffix(hostname, suffix) && strings.Count(hostname, ".") >= strings.Count(suffix, ".") {
			return true
		}
	}

	return false
}

// collectBackendNames extracts unique backend service names from all route rules.
func collectBackendNames(route *gatewayv1.HTTPRoute) []string {
	seen := make(map[string]bool)
	var names []string
	for _, rule := range route.Spec.Rules {
		for _, br := range rule.BackendRefs {
			name := string(br.Name)
			if !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}
	return names
}

// findRouteCondition looks through route status parents for a condition of the given type
// and returns its status string. Returns empty string if not found.
func findRouteCondition(route *gatewayv1.HTTPRoute, condType string) string {
	for _, parent := range route.Status.Parents {
		for _, cond := range parent.Conditions {
			if cond.Type == condType {
				return string(cond.Status)
			}
		}
	}
	return ""
}

// ruleMatchesRequest checks whether an HTTPRouteRule matches a trace request.
func ruleMatchesRequest(rule gatewayv1.HTTPRouteRule, req TraceRequest) bool {
	// If the rule has no matches, it matches everything (catch-all).
	if len(rule.Matches) == 0 {
		return true
	}

	for _, match := range rule.Matches {
		if singleMatchMatches(match, req) {
			return true
		}
	}
	return false
}

// singleMatchMatches checks a single HTTPRouteMatch against the trace request.
func singleMatchMatches(match gatewayv1.HTTPRouteMatch, req TraceRequest) bool {
	// Check path
	if match.Path != nil && match.Path.Value != nil {
		pathType := gatewayv1.PathMatchPathPrefix
		if match.Path.Type != nil {
			pathType = *match.Path.Type
		}
		switch pathType {
		case gatewayv1.PathMatchExact:
			if req.Path != *match.Path.Value {
				return false
			}
		case gatewayv1.PathMatchPathPrefix:
			if !strings.HasPrefix(req.Path, *match.Path.Value) {
				return false
			}
		case gatewayv1.PathMatchRegularExpression:
			// Simplified: just check if it's a substring for tracing purposes
			if !strings.Contains(req.Path, *match.Path.Value) {
				return false
			}
		}
	}

	// Check method
	if match.Method != nil {
		if strings.ToUpper(req.Method) != string(*match.Method) {
			return false
		}
	}

	// Check headers
	for _, headerMatch := range match.Headers {
		headerType := gatewayv1.HeaderMatchExact
		if headerMatch.Type != nil {
			headerType = *headerMatch.Type
		}
		reqValue, exists := req.Headers[string(headerMatch.Name)]
		if !exists {
			return false
		}
		switch headerType {
		case gatewayv1.HeaderMatchExact:
			if reqValue != headerMatch.Value {
				return false
			}
		case gatewayv1.HeaderMatchRegularExpression:
			if !strings.Contains(reqValue, headerMatch.Value) {
				return false
			}
		}
	}

	return true
}
