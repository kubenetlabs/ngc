import apiClient from "./client";
import type { UDPRoute, CreateUDPRoutePayload, UpdateUDPRoutePayload } from "@/types/route";

export async function fetchUDPRoutes(namespace?: string): Promise<UDPRoute[]> {
  const params = namespace ? { namespace } : {};
  const { data } = await apiClient.get<UDPRoute[]>("/udproutes", { params });
  return data;
}

export async function fetchUDPRoute(namespace: string, name: string): Promise<UDPRoute> {
  const { data } = await apiClient.get<UDPRoute>(`/udproutes/${namespace}/${name}`);
  return data;
}

export async function createUDPRoute(payload: CreateUDPRoutePayload): Promise<UDPRoute> {
  const { data } = await apiClient.post<UDPRoute>("/udproutes", payload);
  return data;
}

export async function updateUDPRoute(namespace: string, name: string, payload: UpdateUDPRoutePayload): Promise<UDPRoute> {
  const { data } = await apiClient.put<UDPRoute>(`/udproutes/${namespace}/${name}`, payload);
  return data;
}

export async function deleteUDPRoute(namespace: string, name: string): Promise<void> {
  await apiClient.delete(`/udproutes/${namespace}/${name}`);
}
