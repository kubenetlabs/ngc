import apiClient from "./client";
import type { TopologyResponse } from "@/types/topology";

export async function fetchTopology(): Promise<TopologyResponse> {
  const { data } = await apiClient.get<TopologyResponse>("/topology/full");
  return data;
}

export async function fetchTopologyByGateway(name: string): Promise<TopologyResponse> {
  const { data } = await apiClient.get<TopologyResponse>(`/topology/by-gateway/${name}`);
  return data;
}
