package inference

import "log/slog"

// EPPClient scrapes metrics from the Endpoint Picker (EPP) sidecar
// to surface request routing decisions and load balancing data.
type EPPClient struct {
	// baseURL string
}

// NewEPPClient creates a new EPP metrics client.
func NewEPPClient(baseURL string) *EPPClient {
	slog.Info("epp client created (stub)", "base_url", baseURL)
	return &EPPClient{}
}

// GetMetrics scrapes the EPP /metrics endpoint.
func (c *EPPClient) GetMetrics() (interface{}, error) {
	// TODO: implement HTTP GET to EPP metrics endpoint and parse Prometheus exposition format
	slog.Info("get epp metrics (stub)")
	return nil, nil
}

// GetRoutingDecisions returns recent routing decisions made by the EPP.
func (c *EPPClient) GetRoutingDecisions() ([]interface{}, error) {
	// TODO: implement
	slog.Info("get epp routing decisions (stub)")
	return nil, nil
}
