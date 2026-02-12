import apiClient from "./client";
import type { MetricsSummary, RouteMetrics, GatewayMetrics } from "@/types/metrics";

export async function fetchMetricsSummary(): Promise<MetricsSummary> {
  const { data } = await apiClient.get<MetricsSummary>("/metrics/summary");
  return data;
}

export async function fetchMetricsByRoute(): Promise<RouteMetrics[]> {
  const { data } = await apiClient.get<RouteMetrics[]>("/metrics/by-route");
  return data;
}

export async function fetchMetricsByGateway(): Promise<GatewayMetrics[]> {
  const { data } = await apiClient.get<GatewayMetrics[]>("/metrics/by-gateway");
  return data;
}
