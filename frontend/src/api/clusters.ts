import apiClient from "./client";
import type {
  ManagedCluster,
  RegisterClusterPayload,
  ClusterSummary,
  ClusterTestResult,
  AgentInstallInfo,
} from "@/types/cluster";

// Legacy simple cluster type for backward compatibility.
export interface Cluster {
  name: string;
  displayName: string;
  connected: boolean;
  edition: "oss" | "enterprise" | "unknown";
  default: boolean;
}

export async function fetchClusters(): Promise<ManagedCluster[]> {
  const { data } = await apiClient.get<ManagedCluster[]>("/clusters");
  return data;
}

export async function fetchClusterDetail(
  name: string,
): Promise<ManagedCluster> {
  const { data } = await apiClient.get<ManagedCluster>(
    `/clusters/${name}/detail`,
  );
  return data;
}

export async function registerCluster(
  payload: RegisterClusterPayload,
): Promise<{ message: string; name: string }> {
  const { data } = await apiClient.post("/clusters", payload);
  return data;
}

export async function unregisterCluster(
  name: string,
): Promise<{ message: string; name: string }> {
  const { data } = await apiClient.delete(`/clusters/${name}`);
  return data;
}

export async function testClusterConnection(
  name: string,
): Promise<ClusterTestResult> {
  const { data } = await apiClient.post<ClusterTestResult>(
    `/clusters/${name}/test`,
  );
  return data;
}

export async function getAgentInstallCommand(
  name: string,
): Promise<AgentInstallInfo> {
  const { data } = await apiClient.post<AgentInstallInfo>(
    `/clusters/${name}/install-agent`,
  );
  return data;
}

export async function getClusterSummary(): Promise<ClusterSummary> {
  const { data } = await apiClient.get<ClusterSummary>("/clusters/summary");
  return data;
}
