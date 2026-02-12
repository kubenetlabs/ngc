export interface GatewayClass {
  name: string;
  controllerName: string;
  description?: string;
  parametersRef?: {
    group: string;
    kind: string;
    name: string;
    namespace?: string;
  };
}

export interface Listener {
  name: string;
  hostname?: string;
  port: number;
  protocol: "HTTP" | "HTTPS" | "TLS" | "TCP" | "UDP";
  tls?: {
    mode: "Terminate" | "Passthrough";
    certificateRefs: CertificateRef[];
  };
  allowedRoutes?: {
    namespaces?: { from: "Same" | "All" | "Selector"; selector?: Record<string, string> };
    kinds?: { group: string; kind: string }[];
  };
}

export interface CertificateRef {
  group?: string;
  kind?: string;
  name: string;
  namespace?: string;
}

export interface GatewayStatus {
  conditions: Condition[];
  listeners: ListenerStatus[];
  addresses: GatewayAddress[];
}

export interface ListenerStatus {
  name: string;
  supportedKinds: { group: string; kind: string }[];
  attachedRoutes: number;
  conditions: Condition[];
}

export interface GatewayAddress {
  type: "IPAddress" | "Hostname";
  value: string;
}

export interface Condition {
  type: string;
  status: "True" | "False" | "Unknown";
  reason: string;
  message: string;
  lastTransitionTime: string;
}

export interface Gateway {
  name: string;
  namespace: string;
  gatewayClassName: string;
  listeners: Listener[];
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  status?: GatewayStatus;
  createdAt: string;
}

// --- CRUD payload types ---

import { z } from "zod";

export interface CreateGatewayPayload {
  name: string;
  namespace: string;
  gatewayClassName: string;
  listeners: { name: string; port: number; protocol: string; hostname?: string }[];
}

export interface UpdateGatewayPayload {
  gatewayClassName: string;
  listeners: { name: string; port: number; protocol: string; hostname?: string }[];
}

export const listenerSchema = z.object({
  name: z.string().min(1, "Listener name is required"),
  port: z.number().int().min(1).max(65535),
  protocol: z.enum(["HTTP", "HTTPS", "TLS", "TCP", "UDP"]),
  hostname: z.string().optional(),
});

export const createGatewaySchema = z.object({
  name: z
    .string()
    .min(1, "Name is required")
    .max(253)
    .regex(/^[a-z0-9]([a-z0-9-]*[a-z0-9])?$/, "Must be a valid Kubernetes name"),
  namespace: z.string().min(1, "Namespace is required"),
  gatewayClassName: z.string().min(1, "Gateway class is required"),
  listeners: z.array(listenerSchema).min(1, "At least one listener is required"),
});

export type CreateGatewayFormData = z.infer<typeof createGatewaySchema>;

// --- GatewayBundle types ---

export interface GatewayBundleListener {
  name: string;
  port: number;
  protocol: "HTTP" | "HTTPS" | "TLS" | "TCP" | "UDP";
  hostname?: string;
  tls?: {
    mode: "Terminate" | "Passthrough";
    certificateRefs: { name: string; namespace?: string }[];
  };
  allowedRoutes?: {
    namespaces?: { from: "Same" | "All" | "Selector"; selector?: Record<string, string> };
  };
}

export interface NginxProxyConfig {
  enabled: boolean;
  ipFamily?: "dual" | "ipv4" | "ipv6";
  rewriteClientIP?: { mode: string; setIPRecursively: boolean };
  telemetry?: { exporter?: { endpoint: string } };
}

export interface WAFConfig {
  enabled: boolean;
  policyRef?: string;
}

export interface SnippetsFilterConfig {
  enabled: boolean;
  serverSnippet?: string;
  locationSnippet?: string;
}

export interface GatewayBundleChildStatus {
  kind: string;
  name: string;
  ready: boolean;
  message?: string;
}

export interface GatewayBundleStatus {
  phase: string;
  children: GatewayBundleChildStatus[];
  conditions: Condition[];
  observedSpecHash: string;
  lastReconciledAt: string;
  gatewayAddress: string;
}

export interface GatewayBundle {
  name: string;
  namespace: string;
  gatewayClassName: string;
  listeners: GatewayBundleListener[];
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  nginxProxy?: NginxProxyConfig;
  waf?: WAFConfig;
  snippetsFilter?: SnippetsFilterConfig;
  status?: GatewayBundleStatus;
  createdAt: string;
}

export interface CreateGatewayBundlePayload {
  name: string;
  namespace: string;
  gatewayClassName: string;
  listeners: GatewayBundleListener[];
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  nginxProxy?: NginxProxyConfig;
  waf?: WAFConfig;
  snippetsFilter?: SnippetsFilterConfig;
}

export const createGatewayBundleSchema = z.object({
  name: z.string().min(1, "Name is required").max(253).regex(/^[a-z0-9]([a-z0-9-]*[a-z0-9])?$/, "Must be a valid Kubernetes name"),
  namespace: z.string().min(1, "Namespace is required"),
  gatewayClassName: z.string().min(1, "Gateway class is required"),
  listeners: z.array(listenerSchema).min(1, "At least one listener is required"),
  enableNginxProxy: z.boolean().optional(),
  enableWAF: z.boolean().optional(),
});

export type CreateGatewayBundleFormData = z.infer<typeof createGatewayBundleSchema>;
