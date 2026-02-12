export interface Policy {
  name: string;
  namespace: string;
  policyType: string;
  spec: Record<string, unknown>;
  createdAt: string;
}

export type PolicyType = "ratelimit" | "clientsettings" | "backendtls" | "observability";
