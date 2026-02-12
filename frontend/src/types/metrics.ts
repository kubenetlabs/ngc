export interface MetricsSummary {
  totalRequests: number;
  errorRate: number;
  avgLatencyMs: number;
  p50LatencyMs: number;
  p95LatencyMs: number;
  p99LatencyMs: number;
  requestsPerSec: number;
  activeConnections: number;
}

export interface RouteMetrics {
  namespace: string;
  name: string;
  hostname: string;
  requestsPerSec: number;
  errorRate: number;
  avgLatencyMs: number;
  p95LatencyMs: number;
}

export interface GatewayMetrics {
  namespace: string;
  name: string;
  requestsPerSec: number;
  errorRate: number;
  avgLatencyMs: number;
  activeConnections: number;
}
