import apiClient from "./client";

export interface RouteCheckRequest {
  namespace: string;
  routeName: string;
}

export interface RouteCheckResult {
  name: string;
  status: "pass" | "fail" | "warn" | "skip";
  message: string;
  details?: string;
}

export interface RouteCheckResponse {
  route: string;
  namespace: string;
  status: "healthy" | "degraded" | "unhealthy";
  checks: RouteCheckResult[];
}

export interface SimulateRouteRequest {
  method: string;
  path: string;
  host?: string;
  headers?: Record<string, string>;
}

export interface SimulateMatchDetail {
  ruleIndex: number;
  matched: boolean;
  reason: string;
}

export interface SimulateRouteResponse {
  matched: boolean;
  matchedRule: number;
  matchDetails: SimulateMatchDetail[];
  backends?: { name: string; group?: string; kind?: string; namespace?: string; port?: number; weight?: number }[];
}

export async function runRouteCheck(req: RouteCheckRequest): Promise<RouteCheckResponse> {
  const { data } = await apiClient.post<RouteCheckResponse>("/diagnostics/route-check", req);
  return data;
}

export async function simulateRoute(
  ns: string,
  name: string,
  req: SimulateRouteRequest,
): Promise<SimulateRouteResponse> {
  const { data } = await apiClient.post<SimulateRouteResponse>(
    `/httproutes/${ns}/${name}/simulate`,
    req,
  );
  return data;
}
