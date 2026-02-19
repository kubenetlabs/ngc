package inference

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTritonClient_GetMetrics_Success(t *testing.T) {
	metricsText := `# HELP nv_inference_count Number of inferences
# TYPE nv_inference_count counter
nv_inference_count{model="llama",version="1"} 5000
# HELP nv_inference_queue_duration_us Queue wait time
# TYPE nv_inference_queue_duration_us counter
nv_inference_queue_duration_us{model="llama",version="1"} 12345.6
# HELP nv_gpu_utilization GPU utilization
# TYPE nv_gpu_utilization gauge
nv_gpu_utilization{gpu_uuid="GPU-abc123"} 0.85
# HELP nv_gpu_memory_used_bytes GPU memory used
# TYPE nv_gpu_memory_used_bytes gauge
nv_gpu_memory_used_bytes{gpu_uuid="GPU-abc123"} 8589934592
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

	client := NewTritonClient(server.URL)
	metrics, err := client.GetMetrics()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if metrics.InferenceCount != 5000 {
		t.Errorf("expected inferenceCount=5000, got %d", metrics.InferenceCount)
	}
	if metrics.QueueDurationUs != 12345.6 {
		t.Errorf("expected queueDurationUs=12345.6, got %f", metrics.QueueDurationUs)
	}
	if metrics.GPUUtilPct != 85 {
		t.Errorf("expected gpuUtilPct=85, got %f", metrics.GPUUtilPct)
	}
	if metrics.GPUMemUsedBytes != 8589934592 {
		t.Errorf("expected gpuMemUsedBytes=8589934592, got %d", metrics.GPUMemUsedBytes)
	}
}

func TestTritonClient_GetMetrics_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewTritonClient(server.URL)
	_, err := client.GetMetrics()
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestTritonClient_GetMetrics_ConnectionRefused(t *testing.T) {
	client := NewTritonClient("http://127.0.0.1:1")
	_, err := client.GetMetrics()
	if err == nil {
		t.Fatal("expected error for connection refused")
	}
}

func TestTritonClient_GetModelStatus_Success(t *testing.T) {
	modelResp := tritonModelResponse{
		Name:     "llama-3",
		Versions: []string{"1", "2"},
		Platform: "tensorrt_llm",
		State:    "READY",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/models/llama-3" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(modelResp)
	}))
	defer server.Close()

	client := NewTritonClient(server.URL)
	status, err := client.GetModelStatus("llama-3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.Name != "llama-3" {
		t.Errorf("expected name=llama-3, got %s", status.Name)
	}
	if status.Version != "2" {
		t.Errorf("expected version=2, got %s", status.Version)
	}
	if !status.Ready {
		t.Error("expected ready=true")
	}
	if status.State != "READY" {
		t.Errorf("expected state=READY, got %s", status.State)
	}
}

func TestTritonClient_GetModelStatus_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewTritonClient(server.URL)
	_, err := client.GetModelStatus("nonexistent")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}

func TestTritonClient_GetModelStatus_NoVersions(t *testing.T) {
	modelResp := tritonModelResponse{
		Name:     "simple",
		Versions: []string{},
		State:    "READY",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(modelResp)
	}))
	defer server.Close()

	client := NewTritonClient(server.URL)
	status, err := client.GetModelStatus("simple")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if status.Version != "" {
		t.Errorf("expected empty version, got %s", status.Version)
	}
}

func TestParseTritonMetrics_EmptyResponse(t *testing.T) {
	m := parseTritonMetrics("")
	if m.InferenceCount != 0 {
		t.Errorf("expected 0 inference count, got %d", m.InferenceCount)
	}
	if m.GPUUtilPct != 0 {
		t.Errorf("expected 0 GPU util, got %f", m.GPUUtilPct)
	}
}
