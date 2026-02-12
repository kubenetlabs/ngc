import { useMemo, useCallback } from "react";
import {
  ReactFlow,
  Background,
  Controls,
  type Node,
  type Edge,
  type NodeTypes,
  Position,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import type { TopologyNode, TopologyEdge } from "@/types/topology";

interface TopologyGraphProps {
  nodes: TopologyNode[];
  edges: TopologyEdge[];
}

const statusColors: Record<string, string> = {
  healthy: "border-emerald-500 bg-emerald-500/10",
  degraded: "border-yellow-500 bg-yellow-500/10",
  error: "border-red-500 bg-red-500/10",
  unknown: "border-zinc-500 bg-zinc-500/10",
};

const typeIcons: Record<string, string> = {
  gateway: "GW",
  httproute: "RT",
  service: "SVC",
};

const typeColors: Record<string, string> = {
  gateway: "text-blue-400",
  httproute: "text-purple-400",
  service: "text-emerald-400",
};

function TopologyNodeComponent({ data }: { data: Record<string, unknown> }) {
  const nodeType = data.nodeType as string;
  const status = data.status as string;
  return (
    <div
      className={`rounded-lg border-2 px-4 py-3 shadow-md ${statusColors[status] || statusColors.unknown}`}
    >
      <div className="flex items-center gap-2">
        <span
          className={`text-xs font-bold uppercase ${typeColors[nodeType] || "text-zinc-400"}`}
        >
          {typeIcons[nodeType] || "?"}
        </span>
        <div>
          <div className="text-sm font-medium text-foreground">
            {data.label as string}
          </div>
          <div className="text-[10px] text-muted-foreground">
            {data.namespace as string}
          </div>
        </div>
      </div>
    </div>
  );
}

const nodeTypes: NodeTypes = {
  topology: TopologyNodeComponent,
};

export function TopologyGraph({ nodes, edges }: TopologyGraphProps) {
  const layoutNodes = useCallback((topoNodes: TopologyNode[]): Node[] => {
    // Simple layered layout: gateways left, routes middle, services right
    const gateways = topoNodes.filter((n) => n.type === "gateway");
    const routes = topoNodes.filter((n) => n.type === "httproute");
    const services = topoNodes.filter((n) => n.type === "service");

    const result: Node[] = [];
    const xPositions = { gateway: 0, httproute: 350, service: 700 };
    const ySpacing = 120;

    [gateways, routes, services].forEach((group) => {
      group.forEach((node, i) => {
        result.push({
          id: node.id,
          type: "topology",
          position: { x: xPositions[node.type], y: i * ySpacing + 50 },
          data: {
            label: node.name,
            namespace: node.namespace,
            nodeType: node.type,
            status: node.status,
            ...node.metadata,
          },
          sourcePosition: Position.Right,
          targetPosition: Position.Left,
        });
      });
    });

    return result;
  }, []);

  const flowNodes = useMemo(() => layoutNodes(nodes), [nodes, layoutNodes]);

  const flowEdges: Edge[] = useMemo(
    () =>
      edges.map((e) => ({
        id: e.id,
        source: e.source,
        target: e.target,
        type: "smoothstep",
        animated: e.type === "parentRef",
        style: {
          stroke: e.type === "parentRef" ? "#60a5fa" : "#a78bfa",
          strokeWidth: 2,
        },
        label: e.type,
        labelStyle: { fontSize: 10, fill: "#888" },
      })),
    [edges],
  );

  return (
    <div className="h-[500px] w-full rounded-lg border border-border bg-card">
      <ReactFlow
        nodes={flowNodes}
        edges={flowEdges}
        nodeTypes={nodeTypes}
        fitView
        proOptions={{ hideAttribution: true }}
      >
        <Background />
        <Controls />
      </ReactFlow>
    </div>
  );
}
