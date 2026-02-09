import type { Condition } from "./gateway";

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
