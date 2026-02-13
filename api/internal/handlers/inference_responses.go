package handlers

import "time"

// Inference response types matching frontend/src/types/inference.ts

type InferencePoolResponse struct {
	Name           string            `json:"name"`
	Namespace      string            `json:"namespace"`
	ModelName      string            `json:"modelName"`
	ModelVersion   string            `json:"modelVersion,omitempty"`
	ServingBackend string            `json:"servingBackend"`
	GPUType        string            `json:"gpuType"`
	GPUCount       int               `json:"gpuCount"`
	Replicas       int               `json:"replicas"`
	MinReplicas    int               `json:"minReplicas"`
	MaxReplicas    int               `json:"maxReplicas"`
	Selector       map[string]string `json:"selector"`
	Status         *InferencePoolStatusResponse `json:"status,omitempty"`
	AvgGPUUtil     float64           `json:"avgGpuUtil"`
	CreatedAt      string            `json:"createdAt"`
}

type InferencePoolStatusResponse struct {
	ReadyReplicas  int                 `json:"readyReplicas"`
	TotalReplicas  int                 `json:"totalReplicas"`
	Conditions     []ConditionResponse `json:"conditions"`
}

type EPPDecisionResponse struct {
	Timestamp            string  `json:"timestamp"`
	RequestID            string  `json:"requestId"`
	SelectedPod          string  `json:"selectedPod"`
	Reason               string  `json:"reason"`
	QueueDepth           uint32  `json:"queueDepth"`
	KVCachePct           float64 `json:"kvCachePct"`
	PrefixCacheHit       uint8   `json:"prefixCacheHit"`
	CandidatesConsidered uint32  `json:"candidatesConsidered"`
	DecisionLatencyUs    uint32  `json:"decisionLatencyUs"`
}

type PodGPUMetricsResponse struct {
	PodName          string  `json:"podName"`
	NodeName         string  `json:"nodeName"`
	GPUID            uint8   `json:"gpuId"`
	GPUType          string  `json:"gpuType"`
	QueueDepth       uint16  `json:"queueDepth"`
	KVCacheUtilPct   float64 `json:"kvCacheUtilPct"`
	PrefixCacheState uint8   `json:"prefixCacheState"`
	GPUUtilPct       float64 `json:"gpuUtilPct"`
	GPUMemUsedMB     uint32  `json:"gpuMemUsedMb"`
	GPUMemTotalMB    uint32  `json:"gpuMemTotalMb"`
	GPUTemperatureC  uint16  `json:"gpuTemperatureC"`
	RequestsInFlight uint16  `json:"requestsInFlight"`
}

type InferenceMetricsSummaryResponse struct {
	AvgTTFT            float64 `json:"avgTTFT"`
	P95TTFT            float64 `json:"p95TTFT"`
	P99TTFT            float64 `json:"p99TTFT"`
	AvgTPS             float64 `json:"avgTPS"`
	TotalTokens        uint64  `json:"totalTokens"`
	AvgQueueDepth      float64 `json:"avgQueueDepth"`
	AvgKVCachePct      float64 `json:"avgKVCachePct"`
	PrefixCacheHitRate float64 `json:"prefixCacheHitRate"`
	AvgGPUUtil         float64 `json:"avgGPUUtil"`
}

type HistogramBucketResponse struct {
	RangeStart float64 `json:"rangeStart"`
	RangeEnd   float64 `json:"rangeEnd"`
	Count      uint64  `json:"count"`
}

type TimeseriesPointResponse struct {
	Timestamp string  `json:"timestamp"`
	Value     float64 `json:"value"`
}

type CostEstimateResponse struct {
	GPUType      string  `json:"gpuType"`
	ReplicaCount int     `json:"replicaCount"`
	HourlyRate   float64 `json:"hourlyRate"`
	DailyCost    float64 `json:"dailyCost"`
	MonthlyCost  float64 `json:"monthlyCost"`
}

// Conversion helpers from domain types to response types

func formatTime(t time.Time) string {
	return t.UTC().Format("2006-01-02T15:04:05Z")
}
