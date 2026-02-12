import apiClient from "./client";
import type { Gateway, GatewayClass, CreateGatewayPayload, UpdateGatewayPayload, GatewayBundle, CreateGatewayBundlePayload } from "@/types/gateway";

export async function fetchGateways(namespace?: string): Promise<Gateway[]> {
  const params = namespace ? { namespace } : {};
  const { data } = await apiClient.get<Gateway[]>("/gateways", { params });
  return data;
}

export async function fetchGateway(namespace: string, name: string): Promise<Gateway> {
  const { data } = await apiClient.get<Gateway>(`/gateways/${namespace}/${name}`);
  return data;
}

export async function fetchGatewayClasses(): Promise<GatewayClass[]> {
  const { data } = await apiClient.get<GatewayClass[]>("/gatewayclasses");
  return data;
}

export async function fetchGatewayClass(name: string): Promise<GatewayClass> {
  const { data } = await apiClient.get<GatewayClass>(`/gatewayclasses/${name}`);
  return data;
}

export async function createGateway(payload: CreateGatewayPayload): Promise<Gateway> {
  const { data } = await apiClient.post<Gateway>("/gateways", payload);
  return data;
}

export async function updateGateway(namespace: string, name: string, payload: UpdateGatewayPayload): Promise<Gateway> {
  const { data } = await apiClient.put<Gateway>(`/gateways/${namespace}/${name}`, payload);
  return data;
}

export async function deleteGateway(namespace: string, name: string): Promise<void> {
  await apiClient.delete(`/gateways/${namespace}/${name}`);
}

// --- GatewayBundle API ---

export async function fetchGatewayBundles(namespace?: string): Promise<GatewayBundle[]> {
  const params = namespace ? { namespace } : {};
  const { data } = await apiClient.get<GatewayBundle[]>("/gatewaybundles", { params });
  return data;
}

export async function fetchGatewayBundle(namespace: string, name: string): Promise<GatewayBundle> {
  const { data } = await apiClient.get<GatewayBundle>(`/gatewaybundles/${namespace}/${name}`);
  return data;
}

export async function createGatewayBundle(payload: CreateGatewayBundlePayload): Promise<GatewayBundle> {
  const { data } = await apiClient.post<GatewayBundle>("/gatewaybundles", payload);
  return data;
}

export async function updateGatewayBundle(namespace: string, name: string, payload: Partial<CreateGatewayBundlePayload>): Promise<GatewayBundle> {
  const { data } = await apiClient.put<GatewayBundle>(`/gatewaybundles/${namespace}/${name}`, payload);
  return data;
}

export async function deleteGatewayBundle(namespace: string, name: string): Promise<void> {
  await apiClient.delete(`/gatewaybundles/${namespace}/${name}`);
}
