package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kubenetlabs/ngc/api/internal/cluster"
	"github.com/kubenetlabs/ngc/api/internal/kubernetes"
)

func TestHealthCheck(t *testing.T) {
	r := chi.NewRouter()
	r.Get("/health", HealthCheck)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("expected status ok, got %s", resp["status"])
	}
}

func newSingleClusterManager(t *testing.T) *cluster.Manager {
	t.Helper()
	scheme := setupScheme(t)
	fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	k8s := kubernetes.NewForTest(fakeClient)
	return cluster.NewSingleCluster(k8s)
}

func TestClusterHandler_List_SingleCluster(t *testing.T) {
	manager := newSingleClusterManager(t)
	handler := &ClusterHandler{Manager: manager}

	r := chi.NewRouter()
	r.Get("/clusters", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/clusters", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []ClusterResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp) != 1 {
		t.Fatalf("expected 1 cluster, got %d", len(resp))
	}
	if resp[0].Name != "default" {
		t.Errorf("expected name default, got %s", resp[0].Name)
	}
	if !resp[0].Connected {
		t.Error("expected connected=true")
	}
}

func TestClusterHandler_Get_NoPool(t *testing.T) {
	manager := newSingleClusterManager(t)
	handler := &ClusterHandler{Manager: manager}

	r := chi.NewRouter()
	r.Get("/clusters/{cluster}", handler.Get)

	req := httptest.NewRequest(http.MethodGet, "/clusters/default", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Fatalf("expected status 501, got %d: %s", w.Code, w.Body.String())
	}
}

func TestClusterHandler_Register_NoPool(t *testing.T) {
	manager := newSingleClusterManager(t)
	handler := &ClusterHandler{Manager: manager}

	r := chi.NewRouter()
	r.Post("/clusters", handler.Register)

	req := httptest.NewRequest(http.MethodPost, "/clusters", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Fatalf("expected status 501, got %d: %s", w.Code, w.Body.String())
	}
}

func TestClusterHandler_Unregister_NoPool(t *testing.T) {
	manager := newSingleClusterManager(t)
	handler := &ClusterHandler{Manager: manager}

	r := chi.NewRouter()
	r.Delete("/clusters/{cluster}", handler.Unregister)

	req := httptest.NewRequest(http.MethodDelete, "/clusters/default", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Fatalf("expected status 501, got %d: %s", w.Code, w.Body.String())
	}
}

func TestClusterHandler_TestConnection_NoPool(t *testing.T) {
	manager := newSingleClusterManager(t)
	handler := &ClusterHandler{Manager: manager}

	r := chi.NewRouter()
	r.Post("/clusters/{cluster}/test", handler.TestConnection)

	req := httptest.NewRequest(http.MethodPost, "/clusters/default/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Fatalf("expected status 501, got %d: %s", w.Code, w.Body.String())
	}
}

func TestClusterHandler_Heartbeat_NoPool(t *testing.T) {
	manager := newSingleClusterManager(t)
	handler := &ClusterHandler{Manager: manager}

	r := chi.NewRouter()
	r.Post("/clusters/{cluster}/heartbeat", handler.Heartbeat)

	req := httptest.NewRequest(http.MethodPost, "/clusters/default/heartbeat", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Fatalf("expected status 501, got %d: %s", w.Code, w.Body.String())
	}
}

func TestClusterHandler_Summary_NoPool(t *testing.T) {
	manager := newSingleClusterManager(t)
	handler := &ClusterHandler{Manager: manager}

	r := chi.NewRouter()
	r.Get("/clusters/summary", handler.Summary)

	req := httptest.NewRequest(http.MethodGet, "/clusters/summary", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Fatalf("expected status 501, got %d: %s", w.Code, w.Body.String())
	}
}

func TestClusterHandler_InstallAgent_ValidName(t *testing.T) {
	manager := newSingleClusterManager(t)
	handler := &ClusterHandler{Manager: manager}

	r := chi.NewRouter()
	r.Get("/clusters/{cluster}/install-agent", handler.InstallAgent)

	req := httptest.NewRequest(http.MethodGet, "/clusters/my-cluster-1/install-agent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp["clusterName"] != "my-cluster-1" {
		t.Errorf("expected clusterName my-cluster-1, got %s", resp["clusterName"])
	}
	if resp["helmCommand"] == "" {
		t.Error("expected non-empty helmCommand")
	}
}

func TestClusterHandler_InstallAgent_InvalidName(t *testing.T) {
	manager := newSingleClusterManager(t)
	handler := &ClusterHandler{Manager: manager}

	r := chi.NewRouter()
	r.Get("/clusters/{cluster}/install-agent", handler.InstallAgent)

	req := httptest.NewRequest(http.MethodGet, "/clusters/INVALID_NAME!!/install-agent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGlobalHandler_Gateways_EmptyCluster(t *testing.T) {
	manager := newSingleClusterManager(t)
	handler := &GlobalHandler{Manager: manager}

	r := chi.NewRouter()
	r.Get("/global/gateways", handler.Gateways)

	req := httptest.NewRequest(http.MethodGet, "/global/gateways", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []clusterGateway
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp) != 0 {
		t.Errorf("expected 0 gateways, got %d", len(resp))
	}
}

func TestGlobalHandler_Routes_EmptyCluster(t *testing.T) {
	manager := newSingleClusterManager(t)
	handler := &GlobalHandler{Manager: manager}

	r := chi.NewRouter()
	r.Get("/global/routes", handler.Routes)

	req := httptest.NewRequest(http.MethodGet, "/global/routes", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp []clusterRoute
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp) != 0 {
		t.Errorf("expected 0 routes, got %d", len(resp))
	}
}

func TestGlobalHandler_GPUCapacity_NoPool(t *testing.T) {
	manager := newSingleClusterManager(t)
	handler := &GlobalHandler{Manager: manager}

	r := chi.NewRouter()
	r.Get("/global/gpu-capacity", handler.GPUCapacity)

	req := httptest.NewRequest(http.MethodGet, "/global/gpu-capacity", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp globalGPUCapacity
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.TotalGPUs != 0 {
		t.Errorf("expected 0 total GPUs, got %d", resp.TotalGPUs)
	}
}
