package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kubenetlabs/ngc/api/internal/kubernetes"
)

func TestConfigHandler_GetConfig(t *testing.T) {
	t.Run("with connected kube client (OSS)", func(t *testing.T) {
		scheme := setupScheme(t)
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		k8sClient := kubernetes.NewForTest(fakeClient)
		handler := &ConfigHandler{KubeClient: k8sClient}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/config", nil)
		w := httptest.NewRecorder()

		handler.GetConfig(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp configResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if !resp.Connected {
			t.Error("expected connected to be true")
		}
		// Without enterprise CRDs in fake, should return OSS
		if resp.Edition != "oss" {
			t.Errorf("expected edition oss, got %s", resp.Edition)
		}
		if resp.Version == "" {
			t.Error("expected version to be set")
		}
	})

	t.Run("with nil kube client", func(t *testing.T) {
		handler := &ConfigHandler{KubeClient: nil}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/config", nil)
		w := httptest.NewRecorder()

		handler.GetConfig(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		var resp configResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}

		if resp.Connected {
			t.Error("expected connected to be false")
		}
		if resp.Edition != "unknown" {
			t.Errorf("expected edition unknown, got %s", resp.Edition)
		}
		if resp.Version == "" {
			t.Error("expected version to be set")
		}
	})
}
