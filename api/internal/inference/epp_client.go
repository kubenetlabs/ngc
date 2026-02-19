package inference

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// EPPMetrics holds scraped metrics from the EPP sidecar.
type EPPMetrics struct {
	TotalRequests     int64   `json:"totalRequests"`
	ActiveConnections int64   `json:"activeConnections"`
	AvgLatencyMs      float64 `json:"avgLatencyMs"`
}

// RoutingDecision represents a single EPP routing decision.
// Note: EPP does not currently expose a decision log endpoint,
// so this type exists for future use when EPP adds decision logging.
type RoutingDecision struct {
	Timestamp   time.Time `json:"timestamp"`
	RequestID   string    `json:"requestId"`
	SelectedPod string    `json:"selectedPod"`
	Reason      string    `json:"reason"`
}

// EPPClient scrapes metrics from the Endpoint Picker (EPP) sidecar
// to surface request routing decisions and load balancing data.
type EPPClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewEPPClient creates a new EPP metrics client.
func NewEPPClient(baseURL string) *EPPClient {
	slog.Info("epp client created", "base_url", baseURL)
	return &EPPClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// GetMetrics scrapes the EPP /metrics endpoint and parses Prometheus text format.
func (c *EPPClient) GetMetrics() (*EPPMetrics, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/metrics")
	if err != nil {
		return nil, fmt.Errorf("fetching EPP metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("EPP metrics endpoint returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading EPP metrics response: %w", err)
	}

	return parseEPPMetrics(string(body)), nil
}

// parseEPPMetrics parses Prometheus text exposition format for EPP-specific metrics.
func parseEPPMetrics(text string) *EPPMetrics {
	m := &EPPMetrics{}

	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse simple metric lines like "metric_name{labels} value"
		// or "metric_name value"
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		metricName := parts[0]
		// Strip labels if present
		if idx := strings.Index(metricName, "{"); idx != -1 {
			metricName = metricName[:idx]
		}

		value := parts[len(parts)-1]

		switch metricName {
		case "epp_requests_total", "endpoint_picker_requests_total":
			if v, err := strconv.ParseInt(value, 10, 64); err == nil {
				m.TotalRequests = v
			} else if vf, err := strconv.ParseFloat(value, 64); err == nil {
				m.TotalRequests = int64(vf)
			}
		case "epp_active_connections", "endpoint_picker_active_connections":
			if v, err := strconv.ParseInt(value, 10, 64); err == nil {
				m.ActiveConnections = v
			} else if vf, err := strconv.ParseFloat(value, 64); err == nil {
				m.ActiveConnections = int64(vf)
			}
		case "epp_request_duration_seconds_sum", "endpoint_picker_request_duration_seconds_sum":
			if vf, err := strconv.ParseFloat(value, 64); err == nil {
				// Convert seconds to milliseconds
				if m.TotalRequests > 0 {
					m.AvgLatencyMs = (vf / float64(m.TotalRequests)) * 1000
				}
			}
		}
	}

	return m
}

// GetRoutingDecisions returns recent routing decisions made by the EPP.
// EPP does not currently expose a decision log endpoint.
// This method returns nil until EPP adds decision logging support.
func (c *EPPClient) GetRoutingDecisions() ([]RoutingDecision, error) {
	return nil, nil
}
