import apiClient from "./client";

export interface ImportRequest {
  content: string;
  format: "nginx-conf" | "ingress-yaml" | "virtualserver-yaml";
}

export interface ImportResponse {
  importId: string;
  resourceCount: number;
  format: string;
}

export interface AnalysisResource {
  kind: string;
  name: string;
  namespace: string;
  confidence: "high" | "medium" | "low";
  notes: string[];
}

export interface AnalysisResponse {
  analysisId: string;
  resources: AnalysisResource[];
  warnings: string[];
  errors: string[];
}

export interface GeneratedResource {
  kind: string;
  name: string;
  namespace: string;
  yaml: string;
}

export interface GenerateResponse {
  generateId: string;
  resources: GeneratedResource[];
}

export interface ApplyResult {
  resource: string;
  status: "created" | "updated" | "failed";
  message: string;
}

export interface ApplyResponse {
  results: ApplyResult[];
  successCount: number;
  failureCount: number;
}

export async function importResources(req: ImportRequest): Promise<ImportResponse> {
  const { data } = await apiClient.post<ImportResponse>("/migration/import", req);
  return data;
}

export async function analyzeImport(req: { importId: string }): Promise<AnalysisResponse> {
  const { data } = await apiClient.post<AnalysisResponse>("/migration/analysis", req);
  return data;
}

export async function generateResources(req: { analysisId: string }): Promise<GenerateResponse> {
  const { data } = await apiClient.post<GenerateResponse>("/migration/generate", req);
  return data;
}

export async function applyMigration(req: {
  generateId: string;
  dryRun: boolean;
}): Promise<ApplyResponse> {
  const { data } = await apiClient.post<ApplyResponse>("/migration/apply", req);
  return data;
}
