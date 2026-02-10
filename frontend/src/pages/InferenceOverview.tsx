import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { fetchInferencePools, fetchInferenceMetricsSummary, fetchEPPDecisions } from "@/api/inference";
import { MetricCard } from "@/components/inference/MetricCard";
import { GPUUtilizationBar } from "@/components/inference/GPUUtilizationBar";
import { useActiveCluster } from "@/hooks/useActiveCluster";

export default function InferenceOverview() {
  const activeCluster = useActiveCluster();

  const { data: pools } = useQuery({
    queryKey: ["inference-pools", activeCluster],
    queryFn: fetchInferencePools,
    refetchInterval: 10000,
  });

  const { data: summary } = useQuery({
    queryKey: ["inference-metrics-summary", activeCluster],
    queryFn: () => fetchInferenceMetricsSummary(),
    refetchInterval: 10000,
  });

  const defaultPool = pools?.[0]?.name ?? "llama3-70b-prod";

  const { data: decisions } = useQuery({
    queryKey: ["epp-decisions-overview", activeCluster, defaultPool],
    queryFn: () => fetchEPPDecisions(defaultPool, 5),
    enabled: !!pools && pools.length > 0,
    refetchInterval: 5000,
  });

  const totalGPUs = pools?.reduce((sum, p) => sum + p.gpuCount * p.replicas, 0) ?? 0;

  return (
    <div>
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Inference</h1>
          <p className="mt-1 text-muted-foreground">GPU-aware inference pool management and observability.</p>
        </div>
        <Link
          to="/inference/pools"
          className="rounded-md bg-blue-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-blue-700"
        >
          View Pools
        </Link>
      </div>

      {/* Summary cards */}
      <div className="mt-6 grid grid-cols-2 gap-4 lg:grid-cols-4">
        <MetricCard
          title="Total Pools"
          value={pools?.length ?? 0}
          subtitle={`${pools?.filter((p) => p.status?.conditions[0]?.status === "Ready").length ?? 0} ready`}
        />
        <MetricCard
          title="Total GPUs"
          value={totalGPUs}
          subtitle={pools?.map((p) => p.gpuType).filter((v, i, a) => a.indexOf(v) === i).join(", ") ?? ""}
        />
        <MetricCard
          title="Avg GPU Utilization"
          value={summary ? `${summary.avgGPUUtil.toFixed(1)}%` : "--"}
          subtitle={summary ? `P95 TTFT: ${summary.p95TTFT.toFixed(0)}ms` : ""}
          trend={summary && summary.avgGPUUtil > 80 ? "up" : "neutral"}
        />
        <MetricCard
          title="Avg TTFT"
          value={summary ? `${summary.avgTTFT.toFixed(0)}ms` : "--"}
          subtitle={summary ? `${summary.avgTPS.toFixed(0)} tok/s throughput` : ""}
        />
      </div>

      <div className="mt-6 grid gap-6 lg:grid-cols-2">
        {/* Pool quick-list */}
        <div className="rounded-lg border border-border bg-card p-4">
          <h3 className="mb-3 text-sm font-medium text-muted-foreground">Pools</h3>
          <div className="space-y-2">
            {pools?.map((pool) => (
              <Link
                key={pool.name}
                to={`/inference/pools/${pool.namespace}/${pool.name}`}
                className="flex items-center justify-between rounded-md px-3 py-2 hover:bg-muted/30"
              >
                <div>
                  <span className="text-sm font-medium text-foreground">{pool.name}</span>
                  <span className="ml-2 text-xs text-muted-foreground">{pool.modelName.split("/").pop()}</span>
                </div>
                <div className="flex items-center gap-3">
                  <span className="text-xs text-muted-foreground">
                    {pool.gpuType} x{pool.gpuCount}
                  </span>
                  <GPUUtilizationBar value={pool.avgGpuUtil} />
                </div>
              </Link>
            ))}
          </div>
        </div>

        {/* Recent EPP decisions */}
        <div className="rounded-lg border border-border bg-card p-4">
          <h3 className="mb-3 text-sm font-medium text-muted-foreground">Recent EPP Decisions</h3>
          <div className="space-y-1">
            {decisions?.map((d, i) => (
              <div
                key={d.requestId || i}
                className="flex items-center justify-between rounded px-2 py-1.5 text-xs hover:bg-muted/20"
              >
                <div className="flex items-center gap-2">
                  <span className="font-mono text-muted-foreground">{d.requestId}</span>
                  <span className="text-foreground">&rarr; {d.selectedPod}</span>
                </div>
                <span
                  className={`rounded px-1.5 py-0.5 text-[10px] font-medium ${
                    d.reason === "least_queue"
                      ? "bg-blue-500/15 text-blue-400"
                      : d.reason === "kv_cache"
                        ? "bg-purple-500/15 text-purple-400"
                        : "bg-zinc-500/15 text-zinc-400"
                  }`}
                >
                  {d.reason}
                </span>
              </div>
            ))}
            {(!decisions || decisions.length === 0) && (
              <p className="text-xs text-muted-foreground">No recent decisions.</p>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
