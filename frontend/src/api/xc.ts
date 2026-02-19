import apiClient from "./client";

// --- Publish Types ---

export interface XCPublish {
  name: string;
  namespace: string;
  httpRouteRef: string;
  inferencePoolRef?: string;
  phase: string;
  xcLoadBalancerName?: string;
  xcOriginPoolName?: string;
  xcVirtualIP?: string;
  xcDNS?: string;
  wafPolicyAttached?: string;
  lastSyncedAt?: string;
  createdAt: string;
}

export interface XCStatusResponse {
  connected: boolean;
  publishCount: number;
  xcConnected: boolean;
  tenant?: string;
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
  publicHostname?: string;
  originAddress?: string;
  wafEnabled?: boolean;
  wafPolicyName?: string;
  distributedCloud?: Record<string, unknown>;
}

// --- Credential Types ---

export interface XCCredentials {
  tenant: string;
  namespace: string;
  configured: boolean;
}

export interface XCCredentialsSaveRequest {
  tenant: string;
  apiToken: string;
  namespace: string;
}

export interface XCTestConnectionResponse {
  connected: boolean;
  message: string;
}

// --- Preview Types ---

export interface XCPreviewRequest {
  namespace: string;
  httpRouteRef: string;
  publicHostname?: string;
  originAddress?: string;
  wafEnabled?: boolean;
  wafPolicyName?: string;
}

export interface XCPreviewResponse {
  loadBalancer: Record<string, unknown>;
  originPool: Record<string, unknown>;
  wafPolicy?: string;
}

// --- WAF Types ---

export interface WAFPolicy {
  name: string;
  description?: string;
  mode?: string;
}

// --- API Functions ---

// Status & Metrics
export async function fetchXCStatus(): Promise<XCStatusResponse> {
  const { data } = await apiClient.get<XCStatusResponse>("/xc/status");
  return data;
}

export async function fetchXCMetrics(): Promise<XCMetrics> {
  const { data } = await apiClient.get<XCMetrics>("/xc/metrics");
  return data;
}

// Credentials
export async function fetchXCCredentials(): Promise<XCCredentials> {
  const { data } = await apiClient.get<XCCredentials>("/xc/credentials");
  return data;
}

export async function saveXCCredentials(
  req: XCCredentialsSaveRequest
): Promise<XCCredentials> {
  const { data } = await apiClient.post<XCCredentials>("/xc/credentials", req);
  return data;
}

export async function deleteXCCredentials(): Promise<void> {
  await apiClient.delete("/xc/credentials");
}

export async function testXCConnection(): Promise<XCTestConnectionResponse> {
  const { data } = await apiClient.post<XCTestConnectionResponse>(
    "/xc/test-connection"
  );
  return data;
}

// Publishes
export async function fetchXCPublishes(): Promise<XCPublish[]> {
  try {
    const { data } = await apiClient.get<XCPublish[]>("/xc/publishes");
    return data;
  } catch {
    return [];
  }
}

export async function publishToXC(req: XCPublishRequest): Promise<XCPublish> {
  const { data } = await apiClient.post<XCPublish>("/xc/publish", req);
  return data;
}

export async function deleteXCPublish(id: string): Promise<void> {
  await apiClient.delete(`/xc/publish/${id}`);
}

// Preview
export async function previewXCPublish(
  req: XCPreviewRequest
): Promise<XCPreviewResponse> {
  const { data } = await apiClient.post<XCPreviewResponse>("/xc/preview", req);
  return data;
}

// WAF
export async function fetchWAFPolicies(): Promise<WAFPolicy[]> {
  try {
    const { data } = await apiClient.get<WAFPolicy[]>("/xc/waf-policies");
    return data;
  } catch {
    return [];
  }
}
