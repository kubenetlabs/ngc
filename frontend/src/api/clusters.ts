import apiClient from "./client";

export interface Cluster {
  name: string;
  displayName: string;
  connected: boolean;
  edition: "oss" | "enterprise" | "unknown";
  default: boolean;
}

export async function fetchClusters(): Promise<Cluster[]> {
  const { data } = await apiClient.get<Cluster[]>("/clusters");
  return data;
}
