package alerting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// NotificationPayload is the JSON body sent to webhook endpoints
// when an alert fires or is resolved.
type NotificationPayload struct {
	Status    string      `json:"status"` // "firing" or "resolved"
	Alert     FiringAlert `json:"alert"`
	Timestamp time.Time   `json:"timestamp"`
}

// sendWebhook POSTs a NotificationPayload to each configured webhook URL.
// Errors are logged but do not propagate — alerting notifications are best-effort.
func (e *Evaluator) sendWebhook(alert FiringAlert, resolved bool) {
	if len(e.webhooks) == 0 {
		return
	}

	status := "firing"
	if resolved {
		status = "resolved"
	}

	payload := NotificationPayload{
		Status:    status,
		Alert:     alert,
		Timestamp: time.Now().UTC(),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		slog.Error("alert webhook: failed to marshal payload", "error", err)
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}

	for _, wh := range e.webhooks {
		if err := postWebhook(client, wh, body); err != nil {
			slog.Error("alert webhook: delivery failed",
				"url", wh.URL,
				"status", status,
				"rule_id", alert.RuleID,
				"error", err,
			)
		} else {
			slog.Info("alert webhook: delivered",
				"url", wh.URL,
				"status", status,
				"rule_id", alert.RuleID,
				"rule_name", alert.RuleName,
			)
		}
	}
}

// postWebhook sends the JSON body to a single webhook endpoint.
func postWebhook(client *http.Client, wh WebhookConfig, body []byte) error {
	req, err := http.NewRequest(http.MethodPost, wh.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range wh.Headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}
