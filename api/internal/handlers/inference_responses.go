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
	QueueDepth           int     `json:"queueDepth"`
	KVCachePct           float64 `json:"kvCachePct"`
	PrefixCacheHit       bool    `json:"prefixCacheHit"`
	CandidatesConsidered int     `json:"candidatesConsidered"`
	DecisionLatencyUs    int     `json:"decisionLatencyUs"`
}

type PodGPUMetricsResponse struct {
	PodName          string  `json:"podName"`
	NodeName         string  `json:"nodeName"`
	GPUID            int     `json:"gpuId"`
	GPUType          string  `json:"gpuType"`
	QueueDepth       int     `json:"queueDepth"`
	KVCacheUtilPct   float64 `json:"kvCacheUtilPct"`
	PrefixCacheState bool    `json:"prefixCacheState"`
	GPUUtilPct       float64 `json:"gpuUtilPct"`
	GPUMemUsedMB     int     `json:"gpuMemUsedMb"`
	GPUMemTotalMB    int     `json:"gpuMemTotalMb"`
	GPUTemperatureC  int     `json:"gpuTemperatureC"`
	RequestsInFlight int     `json:"requestsInFlight"`
}

type InferenceMetricsSummaryResponse struct {
	AvgTTFT            float64 `json:"avgTTFT"`
	P95TTFT            float64 `json:"p95TTFT"`
	P99TTFT            float64 `json:"p99TTFT"`
	AvgTPS             float64 `json:"avgTPS"`
	TotalTokens        int64   `json:"totalTokens"`
	AvgQueueDepth      float64 `json:"avgQueueDepth"`
	AvgKVCachePct      float64 `json:"avgKVCachePct"`
	PrefixCacheHitRate float64 `json:"prefixCacheHitRate"`
	AvgGPUUtil         float64 `json:"avgGPUUtil"`
}

type HistogramBucketResponse struct {
	RangeStart float64 `json:"rangeStart"`
	RangeEnd   float64 `json:"rangeEnd"`
	Count      int     `json:"count"`
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
