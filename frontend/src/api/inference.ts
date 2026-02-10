import apiClient from "./client";
import type {
  InferencePoolWithGPU,
  InferenceMetricsSummary,
  PodGPUMetrics,
  EPPDecision,
  HistogramBucket,
  TimeseriesPoint,
  CostEstimate,
} from "@/types/inference";

export async function fetchInferencePools(): Promise<InferencePoolWithGPU[]> {
  const { data } = await apiClient.get<InferencePoolWithGPU[]>("/inference/pools");
  return data;
}

export async function fetchInferencePool(name: string): Promise<InferencePoolWithGPU> {
  const { data } = await apiClient.get<InferencePoolWithGPU>(`/inference/pools/${name}`);
  return data;
}

export async function fetchInferenceMetricsSummary(pool?: string): Promise<InferenceMetricsSummary> {
  const params = pool ? { pool } : {};
  const { data } = await apiClient.get<InferenceMetricsSummary>("/inference/metrics/summary", { params });
  return data;
}

export async function fetchPodMetrics(pool: string): Promise<PodGPUMetrics[]> {
  const { data } = await apiClient.get<PodGPUMetrics[]>("/inference/metrics/pods", { params: { pool } });
  return data;
}

export async function fetchEPPDecisions(pool: string, limit = 20): Promise<EPPDecision[]> {
  const { data } = await apiClient.get<EPPDecision[]>("/inference/metrics/epp-decisions", {
    params: { pool, limit },
  });
  return data;
}

export async function fetchTTFTHistogram(pool: string): Promise<HistogramBucket[]> {
  const { data } = await apiClient.get<HistogramBucket[]>(`/inference/metrics/ttft-histogram/${pool}`);
  return data;
}

export async function fetchTPSThroughput(pool: string): Promise<TimeseriesPoint[]> {
  const { data } = await apiClient.get<TimeseriesPoint[]>(`/inference/metrics/tps-throughput/${pool}`);
  return data;
}

export async function fetchQueueDepthSeries(pool: string): Promise<TimeseriesPoint[]> {
  const { data } = await apiClient.get<TimeseriesPoint[]>(`/inference/metrics/queue-depth/${pool}`);
  return data;
}

export async function fetchGPUUtilSeries(pool: string): Promise<TimeseriesPoint[]> {
  const { data } = await apiClient.get<TimeseriesPoint[]>(`/inference/metrics/gpu-util/${pool}`);
  return data;
}

export async function fetchKVCacheSeries(pool: string): Promise<TimeseriesPoint[]> {
  const { data } = await apiClient.get<TimeseriesPoint[]>(`/inference/metrics/kv-cache/${pool}`);
  return data;
}

export async function fetchCostEstimate(pool: string): Promise<CostEstimate> {
  const { data } = await apiClient.get<CostEstimate>("/inference/metrics/cost", { params: { pool } });
  return data;
}
