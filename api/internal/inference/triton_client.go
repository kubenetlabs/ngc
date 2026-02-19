package inference

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// TritonMetrics holds scraped metrics from NVIDIA Triton Inference Server.
type TritonMetrics struct {
	InferenceCount  int64   `json:"inferenceCount"`
	QueueDurationUs float64 `json:"queueDurationUs"`
	GPUUtilPct      float64 `json:"gpuUtilPct"`
	GPUMemUsedBytes int64   `json:"gpuMemUsedBytes"`
}

// ModelStatus represents the load status of a Triton model.
type ModelStatus struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	State   string `json:"state"`
	Ready   bool   `json:"ready"`
}

// tritonModelResponse is the JSON response from /v2/models/{name}
type tritonModelResponse struct {
	Name     string   `json:"name"`
	Versions []string `json:"versions"`
	Platform string   `json:"platform"`
	State    string   `json:"state"`
}

// TritonClient scrapes metrics from NVIDIA Triton Inference Server
// pods to surface model-level inference performance data.
type TritonClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewTritonClient creates a new Triton metrics client.
func NewTritonClient(baseURL string) *TritonClient {
	slog.Info("triton client created", "base_url", baseURL)
	return &TritonClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// GetMetrics scrapes the Triton /metrics endpoint and parses Prometheus text format.
func (c *TritonClient) GetMetrics() (*TritonMetrics, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/metrics")
	if err != nil {
		return nil, fmt.Errorf("fetching Triton metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Triton metrics endpoint returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading Triton metrics response: %w", err)
	}

	return parseTritonMetrics(string(body)), nil
}

// parseTritonMetrics parses Prometheus text exposition format for Triton-specific metrics.
func parseTritonMetrics(text string) *TritonMetrics {
	m := &TritonMetrics{}

	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		metricName := parts[0]
		if idx := strings.Index(metricName, "{"); idx != -1 {
			metricName = metricName[:idx]
		}

		value := parts[len(parts)-1]

		switch metricName {
		case "nv_inference_count":
			if v, err := strconv.ParseFloat(value, 64); err == nil {
				m.InferenceCount += int64(v)
			}
		case "nv_inference_queue_duration_us":
			if v, err := strconv.ParseFloat(value, 64); err == nil {
				m.QueueDurationUs = v
			}
		case "nv_gpu_utilization":
			if v, err := strconv.ParseFloat(value, 64); err == nil {
				m.GPUUtilPct = v * 100 // Triton reports as 0-1 fraction
			}
		case "nv_gpu_memory_used_bytes":
			if v, err := strconv.ParseFloat(value, 64); err == nil {
				m.GPUMemUsedBytes = int64(v)
			}
		}
	}

	return m
}

// GetModelStatus returns the load status of a specific model.
func (c *TritonClient) GetModelStatus(modelName string) (*ModelStatus, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/v2/models/" + modelName)
	if err != nil {
		return nil, fmt.Errorf("fetching model status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("model %q not found", modelName)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("model status endpoint returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading model status response: %w", err)
	}

	var tritonResp tritonModelResponse
	if err := json.Unmarshal(body, &tritonResp); err != nil {
		return nil, fmt.Errorf("parsing model status JSON: %w", err)
	}

	version := ""
	if len(tritonResp.Versions) > 0 {
		version = tritonResp.Versions[len(tritonResp.Versions)-1]
	}

	state := tritonResp.State
	if state == "" {
		state = "READY"
	}

	return &ModelStatus{
		Name:    tritonResp.Name,
		Version: version,
		State:   state,
		Ready:   state == "READY",
	}, nil
}
