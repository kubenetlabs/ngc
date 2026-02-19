package inference

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEPPClient_GetMetrics_Success(t *testing.T) {
	metricsText := `# HELP epp_requests_total Total number of requests
# TYPE epp_requests_total counter
epp_requests_total 1523
# HELP epp_active_connections Active connections
# TYPE epp_active_connections gauge
epp_active_connections 42
# HELP epp_request_duration_seconds_sum Total request duration
# TYPE epp_request_duration_seconds_sum counter
epp_request_duration_seconds_sum 15.23
`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metrics" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(metricsText))
	}))
	defer server.Close()

	client := NewEPPClient(server.URL)
	metrics, err := client.GetMetrics()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if metrics.TotalRequests != 1523 {
		t.Errorf("expected totalRequests=1523, got %d", metrics.TotalRequests)
	}
	if metrics.ActiveConnections != 42 {
		t.Errorf("expected activeConnections=42, got %d", metrics.ActiveConnections)
	}
	if metrics.AvgLatencyMs <= 0 {
		t.Errorf("expected positive avgLatencyMs, got %f", metrics.AvgLatencyMs)
	}
}

func TestEPPClient_GetMetrics_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewEPPClient(server.URL)
	_, err := client.GetMetrics()
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestEPPClient_GetMetrics_ConnectionRefused(t *testing.T) {
	client := NewEPPClient("http://127.0.0.1:1")
	_, err := client.GetMetrics()
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
}

func TestEPPClient_GetRoutingDecisions(t *testing.T) {
	client := NewEPPClient("http://localhost:9090")
	decisions, err := client.GetRoutingDecisions()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if decisions != nil {
		t.Errorf("expected nil decisions, got %v", decisions)
	}
}

func TestParseEPPMetrics_EmptyResponse(t *testing.T) {
	m := parseEPPMetrics("")
	if m.TotalRequests != 0 {
		t.Errorf("expected 0 requests, got %d", m.TotalRequests)
	}
}

func TestParseEPPMetrics_AlternativeMetricNames(t *testing.T) {
	metricsText := `endpoint_picker_requests_total 500
endpoint_picker_active_connections 10
endpoint_picker_request_duration_seconds_sum 5.0
`
	m := parseEPPMetrics(metricsText)
	if m.TotalRequests != 500 {
		t.Errorf("expected totalRequests=500, got %d", m.TotalRequests)
	}
	if m.ActiveConnections != 10 {
		t.Errorf("expected activeConnections=10, got %d", m.ActiveConnections)
	}
}
