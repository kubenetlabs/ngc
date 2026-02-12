import apiClient from "./client";
import type { LogEntry, LogQueryRequest, TopNEntry } from "@/types/logs";

export async function queryLogs(query: LogQueryRequest): Promise<LogEntry[]> {
  const { data } = await apiClient.post<LogEntry[]>("/logs/query", query);
  return data;
}

export async function fetchTopNLogs(metric?: string, limit?: number): Promise<TopNEntry[]> {
  const params: Record<string, string | number> = {};
  if (metric) params.field = metric;
  if (limit) params.n = limit;
  const { data } = await apiClient.get<TopNEntry[]>("/logs/topn", { params });
  return data;
}
