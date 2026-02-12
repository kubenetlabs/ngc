import apiClient from "./client";

export interface XCPublish {
  name: string;
  namespace: string;
  httpRouteRef: string;
  inferencePoolRef?: string;
  phase: string;
  createdAt: string;
}

export interface XCStatusResponse {
  connected: boolean;
  publishCount: number;
}

export interface XCRegion {
  name: string;
  requests: number;
  latencyMs: number;
}

export interface XCMetrics {
  totalRequests: number;
  avgLatencyMs: number;
  errorRate: number;
  regions: XCRegion[];
}

export interface XCPublishRequest {
  name: string;
  namespace: string;
  httpRouteRef: string;
  inferencePoolRef?: string;
  distributedCloud?: Record<string, unknown>;
}

export async function fetchXCStatus(): Promise<XCStatusResponse> {
  const { data } = await apiClient.get<XCStatusResponse>("/xc/status");
  return data;
}

export async function fetchXCPublishes(): Promise<XCPublish[]> {
  // Status endpoint doesn't return publishes — fetch via separate list if available,
  // or return empty array as a fallback
  try {
    const { data } = await apiClient.get<XCPublish[]>("/xc/publishes");
    return data;
  } catch {
    return [];
  }
}

export async function fetchXCMetrics(): Promise<XCMetrics> {
  const { data } = await apiClient.get<XCMetrics>("/xc/metrics");
  return data;
}

export async function publishToXC(req: XCPublishRequest): Promise<XCPublish> {
  const { data } = await apiClient.post<XCPublish>("/xc/publish", req);
  return data;
}

export async function deleteXCPublish(id: string): Promise<void> {
  await apiClient.delete(`/xc/publish/${id}`);
}
