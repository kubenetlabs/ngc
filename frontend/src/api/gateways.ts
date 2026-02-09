import apiClient from "./client";
import type { Gateway, GatewayClass } from "@/types/gateway";

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
