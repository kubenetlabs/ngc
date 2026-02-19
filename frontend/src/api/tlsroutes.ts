import apiClient from "./client";
import type { TLSRoute, CreateTLSRoutePayload, UpdateTLSRoutePayload } from "@/types/route";

export async function fetchTLSRoutes(namespace?: string): Promise<TLSRoute[]> {
  const params = namespace ? { namespace } : {};
  const { data } = await apiClient.get<TLSRoute[]>("/tlsroutes", { params });
  return data;
}

export async function fetchTLSRoute(namespace: string, name: string): Promise<TLSRoute> {
  const { data } = await apiClient.get<TLSRoute>(`/tlsroutes/${namespace}/${name}`);
  return data;
}

export async function createTLSRoute(payload: CreateTLSRoutePayload): Promise<TLSRoute> {
  const { data } = await apiClient.post<TLSRoute>("/tlsroutes", payload);
  return data;
}

export async function updateTLSRoute(namespace: string, name: string, payload: UpdateTLSRoutePayload): Promise<TLSRoute> {
  const { data } = await apiClient.put<TLSRoute>(`/tlsroutes/${namespace}/${name}`, payload);
  return data;
}

export async function deleteTLSRoute(namespace: string, name: string): Promise<void> {
  await apiClient.delete(`/tlsroutes/${namespace}/${name}`);
}
