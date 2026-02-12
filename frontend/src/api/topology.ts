import apiClient from "./client";
import type { TopologyResponse } from "@/types/topology";

export async function fetchTopology(clusterName?: string): Promise<TopologyResponse> {
  // If a specific cluster is requested, prefix the URL directly so the
  // interceptor (which skips URLs starting with /clusters) doesn't double-prefix.
  const url = clusterName
    ? `/clusters/${clusterName}/topology/full`
    : "/topology/full";
  const { data } = await apiClient.get<TopologyResponse>(url);
  return data;
}

export async function fetchTopologyByGateway(name: string): Promise<TopologyResponse> {
  const { data } = await apiClient.get<TopologyResponse>(`/topology/by-gateway/${name}`);
  return data;
}
