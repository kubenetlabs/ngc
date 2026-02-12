export interface TopologyNode {
  id: string;
  type: "gateway" | "httproute" | "service";
  name: string;
  namespace: string;
  status: "healthy" | "degraded" | "error" | "unknown";
  metadata?: Record<string, string>;
}

export interface TopologyEdge {
  id: string;
  source: string;
  target: string;
  type: "parentRef" | "backendRef";
}

export interface TopologyResponse {
  nodes: TopologyNode[];
  edges: TopologyEdge[];
}
