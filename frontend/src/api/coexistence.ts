import apiClient from "./client";

export interface ResourceCount {
  kind: string;
  count: number;
}

export interface ControllerSummary {
  installed: boolean;
  version?: string;
  resourceCount: number;
  namespaces: string[];
  resources: ResourceCount[];
}

export interface SharedResource {
  kind: string;
  name: string;
  namespace: string;
  usedBy: string[]; // ["kic", "ngf"]
}

export interface Conflict {
  type: string;
  description: string;
  severity: "high" | "medium" | "low";
  resource: string;
}

export interface CoexistenceOverview {
  kic: ControllerSummary;
  ngf: ControllerSummary;
  sharedResources: SharedResource[];
  conflicts: Conflict[];
}

export interface ReadinessCategory {
  name: string;
  score: number;
  status: "pass" | "warn" | "fail";
  details: string;
}

export interface MigrationReadiness {
  score: number;
  status: "ready" | "partial" | "not-ready";
  categories: ReadinessCategory[];
  blockers: string[];
  recommendations: string[];
}

export async function fetchCoexistenceOverview(): Promise<CoexistenceOverview> {
  const { data } = await apiClient.get<CoexistenceOverview>("/coexistence/overview");
  return data;
}

export async function fetchMigrationReadiness(): Promise<MigrationReadiness> {
  const { data } = await apiClient.get<MigrationReadiness>("/coexistence/migration-readiness");
  return data;
}
