import apiClient from "./client";

export interface AlertRule {
  id: string;
  name: string;
  description?: string;
  resource: "certificate" | "gateway" | "inference";
  metric: string;
  operator: "gt" | "lt" | "eq";
  threshold: number;
  severity: "critical" | "warning" | "info";
  enabled: boolean;
  createdAt: string;
  updatedAt?: string;
}

export interface CreateAlertRuleRequest {
  name: string;
  resource: "certificate" | "gateway" | "inference";
  metric: string;
  operator: "gt" | "lt" | "eq";
  threshold: number;
  severity: "critical" | "warning" | "info";
}

export async function fetchAlertRules(): Promise<AlertRule[]> {
  const { data } = await apiClient.get<AlertRule[]>("/alerts");
  return data;
}

export async function createAlertRule(rule: CreateAlertRuleRequest): Promise<AlertRule> {
  const { data } = await apiClient.post<AlertRule>("/alerts", rule);
  return data;
}

export async function deleteAlertRule(id: string): Promise<void> {
  await apiClient.delete(`/alerts/${id}`);
}

export async function toggleAlertRule(id: string): Promise<AlertRule> {
  const { data } = await apiClient.post<AlertRule>(`/alerts/${id}/toggle`);
  return data;
}
