package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestToGatewayResponse(t *testing.T) {
	t.Run("full gateway with all fields", func(t *testing.T) {
		hostname := gatewayv1.Hostname("example.com")
		tlsMode := gatewayv1.TLSModeTerminate
		certNamespace := gatewayv1.Namespace("cert-ns")
		certGroup := gatewayv1.Group("")
		certKind := gatewayv1.Kind("Secret")
		addrType := gatewayv1.IPAddressType

		gw := &gatewayv1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-gateway",
				Namespace: "test-ns",
				Labels: map[string]string{
					"app": "nginx",
				},
				Annotations: map[string]string{
					"description": "test gateway",
				},
				CreationTimestamp: metav1.NewTime(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)),
			},
			Spec: gatewayv1.GatewaySpec{
				GatewayClassName: "nginx",
				Listeners: []gatewayv1.Listener{
					{
						Name:     "http",
						Hostname: &hostname,
						Port:     80,
						Protocol: gatewayv1.HTTPProtocolType,
					},
					{
						Name:     "https",
						Port:     443,
						Protocol: gatewayv1.HTTPSProtocolType,
						TLS: &gatewayv1.ListenerTLSConfig{
							Mode: &tlsMode,
							CertificateRefs: []gatewayv1.SecretObjectReference{
								{
									Group:     &certGroup,
									Kind:      &certKind,
									Name:      "tls-cert",
									Namespace: &certNamespace,
								},
							},
						},
					},
				},
			},
			Status: gatewayv1.GatewayStatus{
				Conditions: []metav1.Condition{
					{
						Type:               "Accepted",
						Status:             metav1.ConditionTrue,
						Reason:             "Accepted",
						Message:            "Gateway accepted",
						LastTransitionTime: metav1.NewTime(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)),
					},
				},
				Addresses: []gatewayv1.GatewayStatusAddress{
					{
						Type:  &addrType,
						Value: "192.168.1.1",
					},
				},
				Listeners: []gatewayv1.ListenerStatus{
					{
						Name:           "http",
						AttachedRoutes: 5,
						SupportedKinds: []gatewayv1.RouteGroupKind{
							{
								Group: (*gatewayv1.Group)(stringPtr("gateway.networking.k8s.io")),
								Kind:  "HTTPRoute",
							},
						},
						Conditions: []metav1.Condition{
							{
								Type:               "Ready",
								Status:             metav1.ConditionTrue,
								Reason:             "Ready",
								Message:            "Listener is ready",
								LastTransitionTime: metav1.NewTime(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)),
							},
						},
					},
				},
			},
		}

		resp := toGatewayResponse(gw)

		if resp.Name != "test-gateway" {
			t.Errorf("expected name test-gateway, got %s", resp.Name)
		}
		if resp.Namespace != "test-ns" {
			t.Errorf("expected namespace test-ns, got %s", resp.Namespace)
		}
		if resp.GatewayClassName != "nginx" {
			t.Errorf("expected gatewayClassName nginx, got %s", resp.GatewayClassName)
		}
		if len(resp.Listeners) != 2 {
			t.Fatalf("expected 2 listeners, got %d", len(resp.Listeners))
		}
		if resp.Listeners[0].Name != "http" {
			t.Errorf("expected listener[0].name http, got %s", resp.Listeners[0].Name)
		}
		if resp.Listeners[0].Hostname == nil || *resp.Listeners[0].Hostname != "example.com" {
			t.Errorf("expected listener[0].hostname example.com")
		}
		if resp.Listeners[1].TLS == nil {
			t.Fatal("expected listener[1].tls to be set")
		}
		if resp.Listeners[1].TLS.Mode != "Terminate" {
			t.Errorf("expected TLS mode Terminate, got %s", resp.Listeners[1].TLS.Mode)
		}
		if len(resp.Listeners[1].TLS.CertificateRefs) != 1 {
			t.Fatalf("expected 1 cert ref, got %d", len(resp.Listeners[1].TLS.CertificateRefs))
		}
		if resp.Listeners[1].TLS.CertificateRefs[0].Name != "tls-cert" {
			t.Errorf("expected cert name tls-cert, got %s", resp.Listeners[1].TLS.CertificateRefs[0].Name)
		}
		if resp.Status == nil {
			t.Fatal("expected status to be set")
		}
		if len(resp.Status.Conditions) != 1 {
			t.Errorf("expected 1 condition, got %d", len(resp.Status.Conditions))
		}
		if len(resp.Status.Addresses) != 1 {
			t.Errorf("expected 1 address, got %d", len(resp.Status.Addresses))
		}
		if resp.Status.Addresses[0].Value != "192.168.1.1" {
			t.Errorf("expected address 192.168.1.1, got %s", resp.Status.Addresses[0].Value)
		}
		if len(resp.Status.Listeners) != 1 {
			t.Errorf("expected 1 listener status, got %d", len(resp.Status.Listeners))
		}
		if resp.Status.Listeners[0].AttachedRoutes != 5 {
			t.Errorf("expected 5 attached routes, got %d", resp.Status.Listeners[0].AttachedRoutes)
		}
		if resp.CreatedAt != "2025-01-01T12:00:00Z" {
			t.Errorf("expected createdAt 2025-01-01T12:00:00Z, got %s", resp.CreatedAt)
		}
	})

	t.Run("nil-safe handling", func(t *testing.T) {
		gw := &gatewayv1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "minimal",
				Namespace:         "default",
				CreationTimestamp: metav1.NewTime(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
			},
			Spec: gatewayv1.GatewaySpec{
				GatewayClassName: "nginx",
				Listeners: []gatewayv1.Listener{
					{
						Name:     "http",
						Port:     80,
						Protocol: gatewayv1.HTTPProtocolType,
					},
				},
			},
		}

		resp := toGatewayResponse(gw)

		if resp.Listeners[0].Hostname != nil {
			t.Errorf("expected nil hostname, got %v", resp.Listeners[0].Hostname)
		}
		if resp.Listeners[0].TLS != nil {
			t.Errorf("expected nil TLS, got %v", resp.Listeners[0].TLS)
		}
		if resp.Status != nil {
			t.Errorf("expected nil status, got %v", resp.Status)
		}
	})
}

func TestToGatewayClassResponse(t *testing.T) {
	t.Run("with parametersRef and description", func(t *testing.T) {
		desc := "NGINX Gateway Class"
		ns := gatewayv1.Namespace("nginx-gateway")
		gc := &gatewayv1.GatewayClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: "nginx",
			},
			Spec: gatewayv1.GatewayClassSpec{
				ControllerName: "gateway.nginx.org/nginx-gateway-controller",
				Description:    &desc,
				ParametersRef: &gatewayv1.ParametersReference{
					Group:     "gateway.nginx.org",
					Kind:      "GatewayConfig",
					Name:      "nginx-config",
					Namespace: &ns,
				},
			},
		}

		resp := toGatewayClassResponse(gc)

		if resp.Name != "nginx" {
			t.Errorf("expected name nginx, got %s", resp.Name)
		}
		if resp.ControllerName != "gateway.nginx.org/nginx-gateway-controller" {
			t.Errorf("expected controller name, got %s", resp.ControllerName)
		}
		if resp.Description == nil || *resp.Description != "NGINX Gateway Class" {
			t.Errorf("expected description to be set")
		}
		if resp.ParametersRef == nil {
			t.Fatal("expected parametersRef to be set")
		}
		if resp.ParametersRef.Name != "nginx-config" {
			t.Errorf("expected parametersRef.name nginx-config, got %s", resp.ParametersRef.Name)
		}
		if resp.ParametersRef.Namespace == nil || *resp.ParametersRef.Namespace != "nginx-gateway" {
			t.Errorf("expected parametersRef.namespace nginx-gateway")
		}
	})

	t.Run("without optional fields", func(t *testing.T) {
		gc := &gatewayv1.GatewayClass{
			ObjectMeta: metav1.ObjectMeta{
				Name: "nginx",
			},
			Spec: gatewayv1.GatewayClassSpec{
				ControllerName: "gateway.nginx.org/nginx-gateway-controller",
			},
		}

		resp := toGatewayClassResponse(gc)

		if resp.Description != nil {
			t.Errorf("expected nil description, got %v", resp.Description)
		}
		if resp.ParametersRef != nil {
			t.Errorf("expected nil parametersRef, got %v", resp.ParametersRef)
		}
	})
}

func TestToHTTPRouteResponse(t *testing.T) {
	t.Run("full route with multiple parentRefs and rules", func(t *testing.T) {
		parentNs := gatewayv1.Namespace("test-ns")
		parentGroup := gatewayv1.Group("gateway.networking.k8s.io")
		parentKind := gatewayv1.Kind("Gateway")
		sectionName := gatewayv1.SectionName("http")
		backendNs := gatewayv1.Namespace("test-ns")
		backendGroup := gatewayv1.Group("")
		backendKind := gatewayv1.Kind("Service")
		port := gatewayv1.PortNumber(8080)
		weight := int32(100)
		pathType := gatewayv1.PathMatchPathPrefix
		method := gatewayv1.HTTPMethodGet

		hr := &gatewayv1.HTTPRoute{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "test-route",
				Namespace:         "test-ns",
				CreationTimestamp: metav1.NewTime(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)),
			},
			Spec: gatewayv1.HTTPRouteSpec{
				CommonRouteSpec: gatewayv1.CommonRouteSpec{
					ParentRefs: []gatewayv1.ParentReference{
						{
							Group:       &parentGroup,
							Kind:        &parentKind,
							Name:        "test-gateway",
							Namespace:   &parentNs,
							SectionName: &sectionName,
						},
					},
				},
				Hostnames: []gatewayv1.Hostname{
					"example.com",
					"www.example.com",
				},
				Rules: []gatewayv1.HTTPRouteRule{
					{
						Matches: []gatewayv1.HTTPRouteMatch{
							{
								Path: &gatewayv1.HTTPPathMatch{
									Type:  &pathType,
									Value: stringPtr("/api"),
								},
								Method: &method,
							},
						},
						BackendRefs: []gatewayv1.HTTPBackendRef{
							{
								BackendRef: gatewayv1.BackendRef{
									BackendObjectReference: gatewayv1.BackendObjectReference{
										Group:     &backendGroup,
										Kind:      &backendKind,
										Name:      "api-service",
										Namespace: &backendNs,
										Port:      &port,
									},
									Weight: &weight,
								},
							},
						},
					},
				},
			},
			Status: gatewayv1.HTTPRouteStatus{
				RouteStatus: gatewayv1.RouteStatus{
					Parents: []gatewayv1.RouteParentStatus{
						{
							ParentRef: gatewayv1.ParentReference{
								Group:     &parentGroup,
								Kind:      &parentKind,
								Name:      "test-gateway",
								Namespace: &parentNs,
							},
							ControllerName: "gateway.nginx.org/nginx-gateway-controller",
							Conditions: []metav1.Condition{
								{
									Type:               "Accepted",
									Status:             metav1.ConditionTrue,
									Reason:             "Accepted",
									Message:            "Route accepted",
									LastTransitionTime: metav1.NewTime(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)),
								},
							},
						},
					},
				},
			},
		}

		resp := toHTTPRouteResponse(hr)

		if resp.Name != "test-route" {
			t.Errorf("expected name test-route, got %s", resp.Name)
		}
		if resp.Namespace != "test-ns" {
			t.Errorf("expected namespace test-ns, got %s", resp.Namespace)
		}
		if len(resp.ParentRefs) != 1 {
			t.Fatalf("expected 1 parentRef, got %d", len(resp.ParentRefs))
		}
		if resp.ParentRefs[0].Name != "test-gateway" {
			t.Errorf("expected parentRef name test-gateway, got %s", resp.ParentRefs[0].Name)
		}
		if resp.ParentRefs[0].SectionName == nil || *resp.ParentRefs[0].SectionName != "http" {
			t.Errorf("expected parentRef sectionName http")
		}
		if len(resp.Hostnames) != 2 {
			t.Errorf("expected 2 hostnames, got %d", len(resp.Hostnames))
		}
		if len(resp.Rules) != 1 {
			t.Fatalf("expected 1 rule, got %d", len(resp.Rules))
		}
		if len(resp.Rules[0].Matches) != 1 {
			t.Errorf("expected 1 match, got %d", len(resp.Rules[0].Matches))
		}
		if resp.Rules[0].Matches[0].Path == nil {
			t.Fatal("expected path match to be set")
		}
		if resp.Rules[0].Matches[0].Path.Type != "PathPrefix" {
			t.Errorf("expected path type PathPrefix, got %s", resp.Rules[0].Matches[0].Path.Type)
		}
		if resp.Rules[0].Matches[0].Path.Value != "/api" {
			t.Errorf("expected path value /api, got %s", resp.Rules[0].Matches[0].Path.Value)
		}
		if resp.Rules[0].Matches[0].Method == nil || *resp.Rules[0].Matches[0].Method != "GET" {
			t.Errorf("expected method GET")
		}
		if len(resp.Rules[0].BackendRefs) != 1 {
			t.Fatalf("expected 1 backendRef, got %d", len(resp.Rules[0].BackendRefs))
		}
		if resp.Rules[0].BackendRefs[0].Name != "api-service" {
			t.Errorf("expected backend name api-service, got %s", resp.Rules[0].BackendRefs[0].Name)
		}
		if resp.Rules[0].BackendRefs[0].Port == nil || *resp.Rules[0].BackendRefs[0].Port != 8080 {
			t.Errorf("expected backend port 8080")
		}
		if resp.Rules[0].BackendRefs[0].Weight == nil || *resp.Rules[0].BackendRefs[0].Weight != 100 {
			t.Errorf("expected backend weight 100")
		}
		if resp.Status == nil {
			t.Fatal("expected status to be set")
		}
		if len(resp.Status.Parents) != 1 {
			t.Errorf("expected 1 parent status, got %d", len(resp.Status.Parents))
		}
		if resp.CreatedAt != "2025-01-01T12:00:00Z" {
			t.Errorf("expected createdAt 2025-01-01T12:00:00Z, got %s", resp.CreatedAt)
		}
	})

	t.Run("edge cases: nil path type and method, empty rules", func(t *testing.T) {
		hr := &gatewayv1.HTTPRoute{
			ObjectMeta: metav1.ObjectMeta{
				Name:              "minimal-route",
				Namespace:         "default",
				CreationTimestamp: metav1.NewTime(time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)),
			},
			Spec: gatewayv1.HTTPRouteSpec{
				Rules: []gatewayv1.HTTPRouteRule{},
			},
		}

		resp := toHTTPRouteResponse(hr)

		if len(resp.Rules) != 0 {
			t.Errorf("expected 0 rules, got %d", len(resp.Rules))
		}
		if resp.Status != nil {
			t.Errorf("expected nil status, got %v", resp.Status)
		}
	})
}

func TestConvertConditions(t *testing.T) {
	conditions := []metav1.Condition{
		{
			Type:               "Ready",
			Status:             metav1.ConditionTrue,
			Reason:             "AllListenersReady",
			Message:            "All listeners are ready",
			LastTransitionTime: metav1.NewTime(time.Date(2025, 1, 15, 10, 30, 45, 0, time.UTC)),
		},
		{
			Type:               "Accepted",
			Status:             metav1.ConditionFalse,
			Reason:             "InvalidConfiguration",
			Message:            "Configuration is invalid",
			LastTransitionTime: metav1.NewTime(time.Date(2025, 1, 15, 10, 30, 45, 0, time.UTC)),
		},
	}

	result := convertConditions(conditions)

	if len(result) != 2 {
		t.Fatalf("expected 2 conditions, got %d", len(result))
	}
	if result[0].Type != "Ready" {
		t.Errorf("expected type Ready, got %s", result[0].Type)
	}
	if result[0].Status != "True" {
		t.Errorf("expected status True, got %s", result[0].Status)
	}
	if result[0].Reason != "AllListenersReady" {
		t.Errorf("expected reason AllListenersReady, got %s", result[0].Reason)
	}
	if result[0].LastTransitionTime != "2025-01-15T10:30:45Z" {
		t.Errorf("expected ISO 8601 UTC time 2025-01-15T10:30:45Z, got %s", result[0].LastTransitionTime)
	}
}

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	data := map[string]string{"message": "success"}

	writeJSON(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
	}

	var result map[string]string
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result["message"] != "success" {
		t.Errorf("expected message success, got %s", result["message"])
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()

	writeError(w, http.StatusBadRequest, "invalid request")

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", w.Header().Get("Content-Type"))
	}

	var result map[string]string
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if result["error"] != "invalid request" {
		t.Errorf("expected error message 'invalid request', got %s", result["error"])
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
