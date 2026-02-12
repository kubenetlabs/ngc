export type GPUType = "A100" | "H100" | "L40S" | "T4";
export type EPPStrategy = "least_queue" | "kv_cache" | "prefix_affinity" | "composite";
export type ScalingBackend = "hpa" | "keda";

export interface InferencePool {
  name: string;
  namespace: string;
  modelName: string;
  modelVersion?: string;
  servingBackend: "triton" | "vllm" | "tgi";
  gpuType: GPUType;
  gpuCount: number;
  replicas: number;
  minReplicas: number;
  maxReplicas: number;
  selector: Record<string, string>;
  status?: InferencePoolStatus;
  createdAt: string;
}

export interface InferencePoolStatus {
  readyReplicas: number;
  totalReplicas: number;
  conditions: { type: string; status: string; message: string }[];
}

export interface EPPConfig {
  scrapeInterval: string;
  metricsPath: string;
  strategy: EPPStrategy;
  weights?: { queueDepth: number; kvCache: number; prefixCache: number };
  dcgmEnabled: boolean;
}

export interface EPPDecision {
  timestamp: string;
  requestId: string;
  selectedPod: string;
  reason: EPPStrategy;
  queueDepth: number;
  kvCachePct: number;
  prefixCacheHit: boolean;
  candidatesConsidered: number;
  decisionLatencyUs: number;
}

export interface PodGPUMetrics {
  podName: string;
  nodeName: string;
  gpuId: number;
  gpuType: GPUType;
  queueDepth: number;
  kvCacheUtilPct: number;
  prefixCacheState: boolean;
  gpuUtilPct: number;
  gpuMemUsedMb: number;
  gpuMemTotalMb: number;
  gpuTemperatureC: number;
  requestsInFlight: number;
}

export interface ScalingPolicy {
  backend: ScalingBackend;
  triggers: ScalingTrigger[];
  cooldownPeriodSeconds: number;
  minReplicas: number;
  maxReplicas: number;
}

export interface ScalingTrigger {
  metric: "queue_depth" | "kv_cache_utilization" | "gpu_utilization" | "request_rate";
  threshold: number;
  durationSeconds: number;
}

export interface InferenceMetricsSummary {
  avgTTFT: number;
  p95TTFT: number;
  p99TTFT: number;
  avgTPS: number;
  totalTokens: number;
  avgQueueDepth: number;
  avgKVCachePct: number;
  prefixCacheHitRate: number;
  avgGPUUtil: number;
}

export interface CostEstimate {
  gpuType: GPUType;
  replicaCount: number;
  hourlyRate: number;
  dailyCost: number;
  monthlyCost: number;
}

export interface HistogramBucket {
  rangeStart: number;
  rangeEnd: number;
  count: number;
}

export interface TimeseriesPoint {
  timestamp: string;
  value: number;
}

export interface InferencePoolWithGPU extends InferencePool {
  avgGpuUtil: number;
}

// Pool CRUD payloads

export interface CreatePoolPayload {
  name: string;
  namespace: string;
  modelName: string;
  modelVersion?: string;
  servingBackend: string;
  gpuType: string;
  gpuCount: number;
  replicas: number;
  minReplicas?: number;
  maxReplicas?: number;
  selector?: Record<string, string>;
  epp?: { strategy: string; weights?: { queueDepth: number; kvCache: number; prefixAffinity: number } };
}

export interface UpdatePoolPayload {
  modelName?: string;
  modelVersion?: string;
  servingBackend?: string;
  gpuType?: string;
  gpuCount?: number;
  replicas?: number;
  minReplicas?: number;
  maxReplicas?: number;
  selector?: Record<string, string>;
  epp?: { strategy: string; weights?: { queueDepth: number; kvCache: number; prefixAffinity: number } };
}

export interface EPPConfigPayload {
  pool: string;
  strategy: string;
  weights?: { queueDepth: number; kvCache: number; prefixAffinity: number };
}

export interface AutoscalingPayload {
  pool: string;
  minReplicas: number;
  maxReplicas: number;
  replicas: number;
}
