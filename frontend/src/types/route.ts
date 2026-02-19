import type { Condition } from "./gateway";
import { z } from "zod";

export type RouteType = "HTTPRoute" | "GRPCRoute" | "TLSRoute" | "TCPRoute" | "UDPRoute";

export interface ParentRef {
  group?: string;
  kind?: string;
  name: string;
  namespace?: string;
  sectionName?: string;
  port?: number;
}

export interface HTTPRouteMatch {
  path?: { type: "Exact" | "PathPrefix" | "RegularExpression"; value: string };
  headers?: { type: "Exact" | "RegularExpression"; name: string; value: string }[];
  queryParams?: { type: "Exact" | "RegularExpression"; name: string; value: string }[];
  method?: string;
}

export interface BackendRef {
  group?: string;
  kind?: string;
  name: string;
  namespace?: string;
  port?: number;
  weight?: number;
}

export interface HTTPRouteFilter {
  type: "RequestHeaderModifier" | "ResponseHeaderModifier" | "URLRewrite" | "RequestRedirect" | "RequestMirror" | "ExtensionRef";
  requestHeaderModifier?: { set?: HeaderValue[]; add?: HeaderValue[]; remove?: string[] };
  responseHeaderModifier?: { set?: HeaderValue[]; add?: HeaderValue[]; remove?: string[] };
  urlRewrite?: { hostname?: string; path?: { type: string; value: string } };
  requestRedirect?: { scheme?: string; hostname?: string; port?: number; statusCode?: number };
}

export interface HeaderValue {
  name: string;
  value: string;
}

export interface HTTPRouteRule {
  matches?: HTTPRouteMatch[];
  filters?: HTTPRouteFilter[];
  backendRefs?: BackendRef[];
}

export interface HTTPRoute {
  name: string;
  namespace: string;
  parentRefs: ParentRef[];
  hostnames?: string[];
  rules: HTTPRouteRule[];
  status?: { parents: { parentRef: ParentRef; controllerName: string; conditions: Condition[] }[] };
  createdAt: string;
}

// --- CRUD payload types ---

export interface CreateHTTPRoutePayload {
  name: string;
  namespace: string;
  parentRefs: { name: string; namespace?: string; sectionName?: string }[];
  hostnames?: string[];
  rules: {
    matches?: { path?: { type: string; value: string }; headers?: { type: string; name: string; value: string }[]; method?: string }[];
    backendRefs?: { name: string; namespace?: string; port?: number; weight?: number }[];
  }[];
}

export interface UpdateHTTPRoutePayload {
  parentRefs: { name: string; namespace?: string; sectionName?: string }[];
  hostnames?: string[];
  rules: {
    matches?: { path?: { type: string; value: string }; headers?: { type: string; name: string; value: string }[]; method?: string }[];
    backendRefs?: { name: string; namespace?: string; port?: number; weight?: number }[];
  }[];
}

export const parentRefSchema = z.object({
  name: z.string().min(1, "Gateway name is required"),
  namespace: z.string().optional(),
  sectionName: z.string().optional(),
});

export const pathMatchSchema = z.object({
  type: z.enum(["Exact", "PathPrefix", "RegularExpression"]),
  value: z.string().min(1, "Path value is required"),
});

export const backendRefSchema = z.object({
  name: z.string().min(1, "Service name is required"),
  namespace: z.string().optional(),
  port: z.number().int().min(1).max(65535).optional(),
  weight: z.number().int().min(0).optional(),
});

export const httpRouteMatchSchema = z.object({
  path: pathMatchSchema.optional(),
  method: z.string().optional(),
});

export const httpRouteRuleSchema = z.object({
  matches: z.array(httpRouteMatchSchema).optional(),
  backendRefs: z.array(backendRefSchema).min(1, "At least one backend ref is required"),
});

export const createHTTPRouteSchema = z.object({
  name: z
    .string()
    .min(1, "Name is required")
    .max(253)
    .regex(/^[a-z0-9]([a-z0-9-]*[a-z0-9])?$/, "Must be a valid Kubernetes name"),
  namespace: z.string().min(1, "Namespace is required"),
  parentRefs: z.array(parentRefSchema).min(1, "At least one parent gateway is required"),
  hostnames: z.string().optional(),
  rules: z.array(httpRouteRuleSchema).min(1, "At least one rule is required"),
});

export type CreateHTTPRouteFormData = z.infer<typeof createHTTPRouteSchema>;

// --- GRPCRoute types ---

export interface GRPCMethodMatch {
  type?: "Exact" | "RegularExpression";
  service?: string;
  method?: string;
}

export interface GRPCRouteMatch {
  method?: GRPCMethodMatch;
  headers?: { type: "Exact" | "RegularExpression"; name: string; value: string }[];
}

export interface GRPCRouteRule {
  matches?: GRPCRouteMatch[];
  backendRefs?: BackendRef[];
}

export interface GRPCRoute {
  name: string;
  namespace: string;
  parentRefs: ParentRef[];
  hostnames?: string[];
  rules: GRPCRouteRule[];
  status?: { parents: { parentRef: ParentRef; controllerName: string; conditions: Condition[] }[] };
  createdAt: string;
}

export interface CreateGRPCRoutePayload {
  name: string;
  namespace: string;
  parentRefs: { name: string; namespace?: string; sectionName?: string }[];
  hostnames?: string[];
  rules: {
    matches?: { method?: { type?: string; service?: string; method?: string }; headers?: { type: string; name: string; value: string }[] }[];
    backendRefs?: { name: string; namespace?: string; port?: number; weight?: number }[];
  }[];
}

export interface UpdateGRPCRoutePayload {
  parentRefs: { name: string; namespace?: string; sectionName?: string }[];
  hostnames?: string[];
  rules: {
    matches?: { method?: { type?: string; service?: string; method?: string }; headers?: { type: string; name: string; value: string }[] }[];
    backendRefs?: { name: string; namespace?: string; port?: number; weight?: number }[];
  }[];
}

// --- TLSRoute types ---

export interface TLSRouteRule {
  backendRefs?: BackendRef[];
}

export interface TLSRoute {
  name: string;
  namespace: string;
  parentRefs: ParentRef[];
  hostnames?: string[];
  rules: TLSRouteRule[];
  status?: { parents: { parentRef: ParentRef; controllerName: string; conditions: Condition[] }[] };
  createdAt: string;
}

export interface CreateTLSRoutePayload {
  name: string;
  namespace: string;
  parentRefs: { name: string; namespace?: string; sectionName?: string }[];
  hostnames?: string[];
  rules: { backendRefs?: { name: string; namespace?: string; port?: number; weight?: number }[] }[];
}

export interface UpdateTLSRoutePayload {
  parentRefs: { name: string; namespace?: string; sectionName?: string }[];
  hostnames?: string[];
  rules: { backendRefs?: { name: string; namespace?: string; port?: number; weight?: number }[] }[];
}

// --- TCPRoute types ---

export interface TCPRouteRule {
  backendRefs?: BackendRef[];
}

export interface TCPRoute {
  name: string;
  namespace: string;
  parentRefs: ParentRef[];
  rules: TCPRouteRule[];
  status?: { parents: { parentRef: ParentRef; controllerName: string; conditions: Condition[] }[] };
  createdAt: string;
}

export interface CreateTCPRoutePayload {
  name: string;
  namespace: string;
  parentRefs: { name: string; namespace?: string; sectionName?: string }[];
  rules: { backendRefs?: { name: string; namespace?: string; port?: number; weight?: number }[] }[];
}

export interface UpdateTCPRoutePayload {
  parentRefs: { name: string; namespace?: string; sectionName?: string }[];
  rules: { backendRefs?: { name: string; namespace?: string; port?: number; weight?: number }[] }[];
}

// --- UDPRoute types ---

export interface UDPRouteRule {
  backendRefs?: BackendRef[];
}

export interface UDPRoute {
  name: string;
  namespace: string;
  parentRefs: ParentRef[];
  rules: UDPRouteRule[];
  status?: { parents: { parentRef: ParentRef; controllerName: string; conditions: Condition[] }[] };
  createdAt: string;
}

export interface CreateUDPRoutePayload {
  name: string;
  namespace: string;
  parentRefs: { name: string; namespace?: string; sectionName?: string }[];
  rules: { backendRefs?: { name: string; namespace?: string; port?: number; weight?: number }[] }[];
}

export interface UpdateUDPRoutePayload {
  parentRefs: { name: string; namespace?: string; sectionName?: string }[];
  rules: { backendRefs?: { name: string; namespace?: string; port?: number; weight?: number }[] }[];
}
