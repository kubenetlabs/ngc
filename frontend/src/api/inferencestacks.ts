import apiClient from "./client";
import type { InferenceStack, CreateInferenceStackPayload } from "@/types/inferencestack";

export async function fetchInferenceStacks(): Promise<InferenceStack[]> {
  const { data } = await apiClient.get<InferenceStack[]>("/inference/stacks");
  return data;
}

export async function fetchInferenceStack(
  namespace: string,
  name: string,
): Promise<InferenceStack> {
  const { data } = await apiClient.get<InferenceStack>(
    `/inference/stacks/${namespace}/${name}`,
  );
  return data;
}

export async function createInferenceStack(
  payload: CreateInferenceStackPayload,
): Promise<InferenceStack> {
  const { data } = await apiClient.post<InferenceStack>("/inference/stacks", payload);
  return data;
}

export async function updateInferenceStack(
  namespace: string,
  name: string,
  payload: Partial<CreateInferenceStackPayload>,
): Promise<InferenceStack> {
  const { data } = await apiClient.put<InferenceStack>(
    `/inference/stacks/${namespace}/${name}`,
    payload,
  );
  return data;
}

export async function deleteInferenceStack(
  namespace: string,
  name: string,
): Promise<void> {
  await apiClient.delete(`/inference/stacks/${namespace}/${name}`);
}
