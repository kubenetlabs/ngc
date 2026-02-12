import apiClient from "./client";
import type { AuditListResponse, AuditDiffResponse } from "@/types/audit";

export interface AuditListParams {
  resource?: string;
  action?: string;
  user?: string;
  namespace?: string;
  since?: string;
  limit?: number;
  offset?: number;
}

export async function fetchAuditEntries(params?: AuditListParams): Promise<AuditListResponse> {
  const { data } = await apiClient.get<AuditListResponse>("/audit", { params });
  return data;
}

export async function fetchAuditDiff(id: string): Promise<AuditDiffResponse> {
  const { data } = await apiClient.get<AuditDiffResponse>(`/audit/diff/${id}`);
  return data;
}
