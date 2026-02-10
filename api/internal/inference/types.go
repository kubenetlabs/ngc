package inference

import "time"

// PoolStatus represents the current state of an inference pool.
type PoolStatus struct {
	Name           string            `json:"name"`
	Namespace      string            `json:"namespace"`
	ModelName      string            `json:"modelName"`
	ModelVersion   string            `json:"modelVersion,omitempty"`
	ServingBackend string            `json:"servingBackend"`
	GPUType        string            `json:"gpuType"`
	GPUCount       int               `json:"gpuCount"`
	Replicas       int               `json:"replicas"`
	ReadyReplicas  int               `json:"readyReplicas"`
	MinReplicas    int               `json:"minReplicas"`
	MaxReplicas    int               `json:"maxReplicas"`
	Selector       map[string]string `json:"selector"`
	AvgGPUUtil     float64           `json:"avgGpuUtil"`
	Status         string            `json:"status"`
	CreatedAt      time.Time         `json:"createdAt"`
}

// PodMetrics holds GPU and inference metrics for a single pod.
type PodMetrics struct {
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

// EPPDecision represents a single endpoint picker routing decision.
type EPPDecision struct {
	Timestamp            time.Time `json:"timestamp"`
	RequestID            string    `json:"requestId"`
	SelectedPod          string    `json:"selectedPod"`
	Reason               string    `json:"reason"`
	QueueDepth           int       `json:"queueDepth"`
	KVCachePct           float64   `json:"kvCachePct"`
	PrefixCacheHit       bool      `json:"prefixCacheHit"`
	CandidatesConsidered int       `json:"candidatesConsidered"`
	DecisionLatencyUs    int       `json:"decisionLatencyUs"`
}

// MetricsSummary holds aggregate inference metrics.
type MetricsSummary struct {
	AvgTTFT           float64 `json:"avgTTFT"`
	P95TTFT           float64 `json:"p95TTFT"`
	P99TTFT           float64 `json:"p99TTFT"`
	AvgTPS            float64 `json:"avgTPS"`
	TotalTokens       int64   `json:"totalTokens"`
	AvgQueueDepth     float64 `json:"avgQueueDepth"`
	AvgKVCachePct     float64 `json:"avgKVCachePct"`
	PrefixCacheHitRate float64 `json:"prefixCacheHitRate"`
	AvgGPUUtil        float64 `json:"avgGPUUtil"`
}

// HistogramBucket is one bar in a TTFT distribution histogram.
type HistogramBucket struct {
	RangeStart float64 `json:"rangeStart"`
	RangeEnd   float64 `json:"rangeEnd"`
	Count      int     `json:"count"`
}

// TimeseriesPoint is a single time+value pair for line/area charts.
type TimeseriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// CostEstimate provides GPU cost projections.
type CostEstimate struct {
	GPUType      string  `json:"gpuType"`
	ReplicaCount int     `json:"replicaCount"`
	HourlyRate   float64 `json:"hourlyRate"`
	DailyCost    float64 `json:"dailyCost"`
	MonthlyCost  float64 `json:"monthlyCost"`
}
