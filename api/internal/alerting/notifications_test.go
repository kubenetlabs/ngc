package alerting

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSendWebhook_NoWebhooks(t *testing.T) {
	eval := &Evaluator{
		webhooks: nil,
		firing:   make(map[string]*FiringAlert),
	}

	// Should not panic with empty webhooks.
	eval.sendWebhook(FiringAlert{RuleID: "test"}, false)
}

func TestSendWebhook_FiringDelivery(t *testing.T) {
	var received NotificationPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatalf("failed to decode payload: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	eval := &Evaluator{
		webhooks: []WebhookConfig{{URL: server.URL}},
		firing:   make(map[string]*FiringAlert),
	}

	alert := FiringAlert{
		RuleID:   "rule-1",
		RuleName: "Test Alert",
		Severity: "critical",
		Metric:   "error_rate",
		Value:    12.5,
		FiredAt:  time.Now().UTC(),
	}

	eval.sendWebhook(alert, false)

	if received.Status != "firing" {
		t.Errorf("status = %q, want %q", received.Status, "firing")
	}
	if received.Alert.RuleID != "rule-1" {
		t.Errorf("alert rule ID = %q, want %q", received.Alert.RuleID, "rule-1")
	}
	if received.Alert.Severity != "critical" {
		t.Errorf("severity = %q, want %q", received.Alert.Severity, "critical")
	}
}

func TestSendWebhook_ResolvedStatus(t *testing.T) {
	var received NotificationPayload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	eval := &Evaluator{
		webhooks: []WebhookConfig{{URL: server.URL}},
		firing:   make(map[string]*FiringAlert),
	}

	eval.sendWebhook(FiringAlert{RuleID: "rule-2"}, true)

	if received.Status != "resolved" {
		t.Errorf("status = %q, want %q", received.Status, "resolved")
	}
}

func TestSendWebhook_CustomHeaders(t *testing.T) {
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	eval := &Evaluator{
		webhooks: []WebhookConfig{{
			URL:     server.URL,
			Headers: map[string]string{"Authorization": "Bearer secret-token"},
		}},
		firing: make(map[string]*FiringAlert),
	}

	eval.sendWebhook(FiringAlert{RuleID: "rule-3"}, false)

	if receivedAuth != "Bearer secret-token" {
		t.Errorf("Authorization = %q, want %q", receivedAuth, "Bearer secret-token")
	}
}

func TestPostWebhook_ErrorOnNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	wh := WebhookConfig{URL: server.URL}

	err := postWebhook(client, wh, []byte(`{}`))
	if err == nil {
		t.Error("expected error for 500 status, got nil")
	}
}

func TestPostWebhook_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	wh := WebhookConfig{URL: server.URL}

	err := postWebhook(client, wh, []byte(`{"test":true}`))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
