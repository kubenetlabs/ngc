import apiClient from "./client";
import type { HTTPRoute } from "@/types/route";

export async function fetchHTTPRoutes(namespace?: string): Promise<HTTPRoute[]> {
  const params = namespace ? { namespace } : {};
  const { data } = await apiClient.get<HTTPRoute[]>("/httproutes", { params });
  return data;
}

export async function fetchHTTPRoute(namespace: string, name: string): Promise<HTTPRoute> {
  const { data } = await apiClient.get<HTTPRoute>(`/httproutes/${namespace}/${name}`);
  return data;
}
