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
	GPUCount       uint32            `json:"gpuCount"`
	Replicas       uint32            `json:"replicas"`
	ReadyReplicas  uint32            `json:"readyReplicas"`
	MinReplicas    uint32            `json:"minReplicas"`
	MaxReplicas    uint32            `json:"maxReplicas"`
	Selector       map[string]string `json:"selector"`
	AvgGPUUtil     float64           `json:"avgGpuUtil"`
	Status         string            `json:"status"`
	CreatedAt      time.Time         `json:"createdAt"`
}

// PodMetrics holds GPU and inference metrics for a single pod.
type PodMetrics struct {
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

// EPPDecision represents a single endpoint picker routing decision.
type EPPDecision struct {
	Timestamp            time.Time `json:"timestamp"`
	RequestID            string    `json:"requestId"`
	SelectedPod          string    `json:"selectedPod"`
	Reason               string    `json:"reason"`
	QueueDepth           uint32    `json:"queueDepth"`
	KVCachePct           float64   `json:"kvCachePct"`
	PrefixCacheHit       uint8     `json:"prefixCacheHit"`
	CandidatesConsidered uint32    `json:"candidatesConsidered"`
	DecisionLatencyUs    uint32    `json:"decisionLatencyUs"`
}

// MetricsSummary holds aggregate inference metrics.
type MetricsSummary struct {
	AvgTTFT           float64 `json:"avgTTFT"`
	P95TTFT           float64 `json:"p95TTFT"`
	P99TTFT           float64 `json:"p99TTFT"`
	AvgTPS            float64 `json:"avgTPS"`
	TotalTokens       uint64  `json:"totalTokens"`
	AvgQueueDepth     float64 `json:"avgQueueDepth"`
	AvgKVCachePct     float64 `json:"avgKVCachePct"`
	PrefixCacheHitRate float64 `json:"prefixCacheHitRate"`
	AvgGPUUtil        float64 `json:"avgGPUUtil"`
}

// HistogramBucket is one bar in a TTFT distribution histogram.
type HistogramBucket struct {
	RangeStart float64 `json:"rangeStart"`
	RangeEnd   float64 `json:"rangeEnd"`
	Count      uint64  `json:"count"`
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
