package inference

import "log/slog"

// TritonClient scrapes metrics from NVIDIA Triton Inference Server
// pods to surface model-level inference performance data.
type TritonClient struct {
	// baseURL string
}

// NewTritonClient creates a new Triton metrics client.
func NewTritonClient(baseURL string) *TritonClient {
	slog.Info("triton client created (stub)", "base_url", baseURL)
	return &TritonClient{}
}

// GetMetrics scrapes the Triton /metrics endpoint.
func (c *TritonClient) GetMetrics() (interface{}, error) {
	// TODO: implement HTTP GET to Triton /metrics and parse Prometheus exposition format
	// Key metrics: nv_inference_request_duration, nv_inference_queue_duration,
	// nv_inference_count, nv_gpu_utilization, etc.
	slog.Info("get triton metrics (stub)")
	return nil, nil
}

// GetModelStatus returns the load status of a specific model.
func (c *TritonClient) GetModelStatus(modelName string) (interface{}, error) {
	// TODO: implement using Triton HTTP API /v2/models/{modelName}
	slog.Info("get triton model status (stub)", "model", modelName)
	return nil, nil
}
