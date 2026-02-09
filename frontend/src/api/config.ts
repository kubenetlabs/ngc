import apiClient from "./client";

export interface AppConfig {
  edition: "oss" | "enterprise" | "unknown";
  version: string;
  connected: boolean;
  cluster?: string;
}

export async function fetchConfig(): Promise<AppConfig> {
  const { data } = await apiClient.get<AppConfig>("/config");
  return data;
}
