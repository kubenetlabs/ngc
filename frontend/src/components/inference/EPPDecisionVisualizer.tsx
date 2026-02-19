import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { PodCard } from "./PodCard";
import { fetchPodMetrics, fetchEPPDecisions } from "@/api/inference";
import { useActiveCluster } from "@/hooks/useActiveCluster";
import { useWebSocket } from "@/hooks/useWebSocket";
import type { EPPDecision } from "@/types/inference";

interface EPPDecisionVisualizerProps {
  pool: string;
}

export function EPPDecisionVisualizer({ pool }: EPPDecisionVisualizerProps) {
  const activeCluster = useActiveCluster();
  const [highlightedPod, setHighlightedPod] = useState<string | null>(null);
  const [wsDecisions, setWsDecisions] = useState<EPPDecision[]>([]);

  const { data: pods } = useQuery({
    queryKey: ["pod-metrics", activeCluster, pool],
    queryFn: () => fetchPodMetrics(pool),
    refetchInterval: 5000,
  });

  const { data: initialDecisions } = useQuery({
    queryKey: ["epp-decisions", activeCluster, pool],
    queryFn: () => fetchEPPDecisions(pool, 10),
  });

  const { connected } = useWebSocket({
    url: "/api/v1/ws/inference/epp-decisions",
    onMessage: (msg: unknown) => {
      const data = msg as { topic?: string; data?: EPPDecision };
      if (data.topic === "epp-decisions" && data.data?.requestId) {
        const decision = data.data;
        setWsDecisions((prev) => [decision, ...prev].slice(0, 20));
        setHighlightedPod(decision.selectedPod);
        setTimeout(() => setHighlightedPod(null), 800);
      }
    },
  });

  const decisions = wsDecisions.length > 0 ? wsDecisions : initialDecisions ?? [];

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-medium text-foreground">Live EPP Routing</h3>
        <div className="flex items-center gap-1.5">
          <div
            className={`h-2 w-2 rounded-full ${connected ? "bg-emerald-500 animate-pulse" : "bg-zinc-500"}`}
          />
          <span className="text-xs text-muted-foreground">
            {connected ? "Live" : "Connecting..."}
          </span>
        </div>
      </div>

      {pods && pods.length > 0 && (
        <div className="grid grid-cols-2 gap-2 lg:grid-cols-3">
          {pods.map((pod) => (
            <PodCard
              key={pod.podName}
              podName={pod.podName}
              gpuUtilPct={pod.gpuUtilPct}
              kvCacheUtilPct={pod.kvCacheUtilPct}
              queueDepth={pod.queueDepth}
              requestsInFlight={pod.requestsInFlight}
              highlighted={highlightedPod === pod.podName}
            />
          ))}
        </div>
      )}

      <div className="rounded-lg border border-border bg-card p-3">
        <h4 className="mb-2 text-xs font-medium text-muted-foreground">Recent Decisions</h4>
        <div className="max-h-48 space-y-1 overflow-y-auto">
          {decisions.slice(0, 10).map((d, i) => (
            <div
              key={d.requestId || i}
              className="flex items-center justify-between rounded px-2 py-1 text-xs hover:bg-muted/30"
            >
              <div className="flex items-center gap-2">
                <span className="font-mono text-muted-foreground">{d.requestId}</span>
                <span className="text-foreground">&rarr; {d.selectedPod}</span>
              </div>
              <div className="flex items-center gap-2">
                <span
                  className={`rounded px-1.5 py-0.5 text-[10px] font-medium ${
                    d.reason === "least_queue"
                      ? "bg-blue-500/15 text-blue-400"
                      : d.reason === "kv_cache"
                        ? "bg-purple-500/15 text-purple-400"
                        : d.reason === "prefix_affinity"
                          ? "bg-amber-500/15 text-amber-400"
                          : "bg-zinc-500/15 text-zinc-400"
                  }`}
                >
                  {d.reason}
                </span>
                <span className="text-muted-foreground">{d.decisionLatencyUs}us</span>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
