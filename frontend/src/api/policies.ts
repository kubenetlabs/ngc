import apiClient from "./client";
import type { Policy, PolicyType } from "@/types/policy";

export async function fetchPolicies(type: PolicyType): Promise<Policy[]> {
  const { data } = await apiClient.get<Policy[]>(`/policies/${type}`);
  return data;
}

export async function fetchPolicy(type: PolicyType, name: string, namespace?: string): Promise<Policy> {
  const params = namespace ? { namespace } : {};
  const { data } = await apiClient.get<Policy>(`/policies/${type}/${name}`, { params });
  return data;
}

export async function createPolicy(type: PolicyType, policy: Partial<Policy>): Promise<Policy> {
  const { data } = await apiClient.post<Policy>(`/policies/${type}`, policy);
  return data;
}

export async function deletePolicy(type: PolicyType, name: string, namespace?: string): Promise<void> {
  const params = namespace ? { namespace } : {};
  await apiClient.delete(`/policies/${type}/${name}`, { params });
}
