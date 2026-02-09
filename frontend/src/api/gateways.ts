import apiClient from "./client";
import type { Gateway, GatewayClass, CreateGatewayPayload, UpdateGatewayPayload } from "@/types/gateway";

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
