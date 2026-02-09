package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kubenetlabs/ngc/api/internal/kubernetes"
)

func setupScheme(t *testing.T) *runtime.Scheme {
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatalf("failed to add client-go scheme: %v", err)
	}
	if err := gatewayv1.Install(scheme); err != nil {
		t.Fatalf("failed to add gateway-api scheme: %v", err)
	}
	return scheme
}

func setupTestServer(t *testing.T) *httptest.Server {
	scheme := setupScheme(t)

	// Seed test data
	gw := &gatewayv1.Gateway{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-gw",
			Namespace: "test-ns",
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

	gc := &gatewayv1.GatewayClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "nginx",
		},
		Spec: gatewayv1.GatewayClassSpec{
			ControllerName: "gateway.nginx.org/nginx-gateway-controller",
		},
	}

	route := &gatewayv1.HTTPRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-route",
			Namespace: "test-ns",
		},
		Spec: gatewayv1.HTTPRouteSpec{
			Rules: []gatewayv1.HTTPRouteRule{
				{
					BackendRefs: []gatewayv1.HTTPBackendRef{
						{
							BackendRef: gatewayv1.BackendRef{
								BackendObjectReference: gatewayv1.BackendObjectReference{
									Name: "test-service",
								},
							},
						},
					},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(gw, gc, route).
		Build()

	k8sClient := kubernetes.NewForTest(fakeClient)
	srv := New(Config{KubeClient: k8sClient})

	return httptest.NewServer(srv.Router)
}

func TestServer_Integration(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		checkJSON      bool
		checkCORS      bool
	}{
		{
			name:           "GET /api/v1/gateways",
			path:           "/api/v1/gateways",
			expectedStatus: http.StatusOK,
			checkJSON:      true,
			checkCORS:      true,
		},
		{
			name:           "GET /api/v1/gatewayclasses",
			path:           "/api/v1/gatewayclasses",
			expectedStatus: http.StatusOK,
			checkJSON:      true,
			checkCORS:      true,
		},
		{
			name:           "GET /api/v1/httproutes",
			path:           "/api/v1/httproutes",
			expectedStatus: http.StatusOK,
			checkJSON:      true,
			checkCORS:      true,
		},
		{
			name:           "GET /api/v1/config",
			path:           "/api/v1/config",
			expectedStatus: http.StatusOK,
			checkJSON:      true,
			checkCORS:      true,
		},
		{
			name:           "GET /api/v1/gateways/test-ns/test-gw",
			path:           "/api/v1/gateways/test-ns/test-gw",
			expectedStatus: http.StatusOK,
			checkJSON:      true,
			checkCORS:      true,
		},
		{
			name:           "GET /api/v1/gateways/bad-ns/bad-name (404)",
			path:           "/api/v1/gateways/bad-ns/bad-name",
			expectedStatus: http.StatusNotFound,
			checkJSON:      true,
			checkCORS:      true,
		},
		{
			name:           "GET /api/v1/certificates (not implemented)",
			path:           "/api/v1/certificates",
			expectedStatus: http.StatusNotImplemented,
			checkJSON:      true,
			checkCORS:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := http.Get(ts.URL + tt.path)
			if err != nil {
				t.Fatalf("failed to make request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.checkJSON {
				contentType := resp.Header.Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("expected Content-Type application/json, got %s", contentType)
				}

				// Verify we can decode JSON
				var result interface{}
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Errorf("failed to decode JSON response: %v", err)
				}
			}

			if tt.checkCORS {
				cors := resp.Header.Get("Access-Control-Allow-Origin")
				if cors != "*" {
					t.Errorf("expected CORS header *, got %s", cors)
				}
			}
		})
	}
}

func TestServer_ListFiltering(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	t.Run("filter gateways by namespace", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/gateways?namespace=test-ns")
		if err != nil {
			t.Fatalf("failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var gateways []interface{}
		if err := json.NewDecoder(resp.Body).Decode(&gateways); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(gateways) != 1 {
			t.Errorf("expected 1 gateway in test-ns, got %d", len(gateways))
		}
	})

	t.Run("filter httproutes by namespace", func(t *testing.T) {
		resp, err := http.Get(ts.URL + "/api/v1/httproutes?namespace=test-ns")
		if err != nil {
			t.Fatalf("failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		var routes []interface{}
		if err := json.NewDecoder(resp.Body).Decode(&routes); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if len(routes) != 1 {
			t.Errorf("expected 1 route in test-ns, got %d", len(routes))
		}
	})
}

func TestServer_ConfigEndpoint(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/v1/config")
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	var config map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if config["connected"] != true {
		t.Error("expected connected to be true")
	}

	if config["edition"] != "oss" {
		t.Errorf("expected edition oss, got %v", config["edition"])
	}

	if config["version"] == nil || config["version"] == "" {
		t.Error("expected version to be set")
	}
}

func TestServer_CORSMiddleware(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	t.Run("OPTIONS request", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodOptions, ts.URL+"/api/v1/gateways", nil)
		if err != nil {
			t.Fatalf("failed to create request: %v", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("failed to make request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNoContent {
			t.Errorf("expected status 204, got %d", resp.StatusCode)
		}

		if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
			t.Error("expected Access-Control-Allow-Origin header")
		}

		if resp.Header.Get("Access-Control-Allow-Methods") == "" {
			t.Error("expected Access-Control-Allow-Methods header")
		}

		if resp.Header.Get("Access-Control-Allow-Headers") == "" {
			t.Error("expected Access-Control-Allow-Headers header")
		}
	})
}

func TestServer_NotImplementedEndpoints(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	tests := []struct {
		path   string
		method string
	}{
		{"/api/v1/certificates", http.MethodGet},
		{"/api/v1/metrics/summary", http.MethodGet},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, ts.URL+tt.path, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("failed to make request: %v", err)
			}
			defer resp.Body.Close()

			// These endpoints should return 501 Not Implemented
			if resp.StatusCode != http.StatusNotImplemented {
				t.Errorf("expected status 501, got %d", resp.StatusCode)
			}

			var result map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				t.Errorf("failed to decode JSON response: %v", err)
			}
		})
	}
}
