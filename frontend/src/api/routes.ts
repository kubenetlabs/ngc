import apiClient from "./client";
import type { HTTPRoute, CreateHTTPRoutePayload, UpdateHTTPRoutePayload } from "@/types/route";

export async function fetchHTTPRoutes(namespace?: string): Promise<HTTPRoute[]> {
  const params = namespace ? { namespace } : {};
  const { data } = await apiClient.get<HTTPRoute[]>("/httproutes", { params });
  return data;
}

export async function fetchHTTPRoute(namespace: string, name: string): Promise<HTTPRoute> {
  const { data } = await apiClient.get<HTTPRoute>(`/httproutes/${namespace}/${name}`);
  return data;
}

export async function createHTTPRoute(payload: CreateHTTPRoutePayload): Promise<HTTPRoute> {
  const { data } = await apiClient.post<HTTPRoute>("/httproutes", payload);
  return data;
}

export async function updateHTTPRoute(namespace: string, name: string, payload: UpdateHTTPRoutePayload): Promise<HTTPRoute> {
  const { data } = await apiClient.put<HTTPRoute>(`/httproutes/${namespace}/${name}`, payload);
  return data;
}

export async function deleteHTTPRoute(namespace: string, name: string): Promise<void> {
  await apiClient.delete(`/httproutes/${namespace}/${name}`);
}
