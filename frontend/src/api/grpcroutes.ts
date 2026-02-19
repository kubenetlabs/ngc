import apiClient from "./client";
import type { GRPCRoute, CreateGRPCRoutePayload, UpdateGRPCRoutePayload } from "@/types/route";

export async function fetchGRPCRoutes(namespace?: string): Promise<GRPCRoute[]> {
  const params = namespace ? { namespace } : {};
  const { data } = await apiClient.get<GRPCRoute[]>("/grpcroutes", { params });
  return data;
}

export async function fetchGRPCRoute(namespace: string, name: string): Promise<GRPCRoute> {
  const { data } = await apiClient.get<GRPCRoute>(`/grpcroutes/${namespace}/${name}`);
  return data;
}

export async function createGRPCRoute(payload: CreateGRPCRoutePayload): Promise<GRPCRoute> {
  const { data } = await apiClient.post<GRPCRoute>("/grpcroutes", payload);
  return data;
}

export async function updateGRPCRoute(namespace: string, name: string, payload: UpdateGRPCRoutePayload): Promise<GRPCRoute> {
  const { data } = await apiClient.put<GRPCRoute>(`/grpcroutes/${namespace}/${name}`, payload);
  return data;
}

export async function deleteGRPCRoute(namespace: string, name: string): Promise<void> {
  await apiClient.delete(`/grpcroutes/${namespace}/${name}`);
}
