import apiClient from "./client";
import type { TCPRoute, CreateTCPRoutePayload, UpdateTCPRoutePayload } from "@/types/route";

export async function fetchTCPRoutes(namespace?: string): Promise<TCPRoute[]> {
  const params = namespace ? { namespace } : {};
  const { data } = await apiClient.get<TCPRoute[]>("/tcproutes", { params });
  return data;
}

export async function fetchTCPRoute(namespace: string, name: string): Promise<TCPRoute> {
  const { data } = await apiClient.get<TCPRoute>(`/tcproutes/${namespace}/${name}`);
  return data;
}

export async function createTCPRoute(payload: CreateTCPRoutePayload): Promise<TCPRoute> {
  const { data } = await apiClient.post<TCPRoute>("/tcproutes", payload);
  return data;
}

export async function updateTCPRoute(namespace: string, name: string, payload: UpdateTCPRoutePayload): Promise<TCPRoute> {
  const { data } = await apiClient.put<TCPRoute>(`/tcproutes/${namespace}/${name}`, payload);
  return data;
}

export async function deleteTCPRoute(namespace: string, name: string): Promise<void> {
  await apiClient.delete(`/tcproutes/${namespace}/${name}`);
}
