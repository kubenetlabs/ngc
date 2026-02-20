import { useState } from "react";
import { useParams, Link, useNavigate } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import {
  fetchInferencePool,
  fetchInferenceMetricsSummary,
  fetchPodMetrics,
  fetchTTFTHistogram,
  fetchTPSThroughput,
  fetchQueueDepthSeries,
  fetchGPUUtilSeries,
  fetchKVCacheSeries,
  fetchCostEstimate,
  deleteInferencePool,
} from "@/api/inference";
import { MetricCard } from "@/components/inference/MetricCard";
import { GPUHeatmap } from "@/components/inference/GPUHeatmap";
import { EPPDecisionVisualizer } from "@/components/inference/EPPDecisionVisualizer";
import { TTFTHistogram } from "@/components/inference/TTFTHistogram";
import { TimeseriesChart } from "@/components/inference/TimeseriesChart";
import { CostEstimateCard } from "@/components/inference/CostEstimateCard";
import { useActiveCluster } from "@/hooks/useActiveCluster";

type Tab = "overview" | "epp" | "metrics" | "cost";

export default function InferencePoolDetail() {
  const { ns, name } = useParams<{ ns: string; name: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const activeCluster = useActiveCluster();
  const [activeTab, setActiveTab] = useState<Tab>("overview");
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [deleteError, setDeleteError] = useState<string | null>(null);
  const poolName = name ?? "";

  const { data: pool, isLoading, error } = useQuery({
    queryKey: ["inference-pool", activeCluster, poolName],
    queryFn: () => fetchInferencePool(poolName),
    enabled: !!poolName,
  });

  const { data: summary } = useQuery({
    queryKey: ["inference-metrics-summary", activeCluster, poolName],
    queryFn: () => fetchInferenceMetricsSummary(poolName),
    enabled: !!poolName,
    refetchInterval: 10000,
  });

  const { data: pods } = useQuery({
    queryKey: ["pod-metrics", activeCluster, poolName],
    queryFn: () => fetchPodMetrics(poolName),
    enabled: !!poolName && (activeTab === "overview" || activeTab === "epp"),
    refetchInterval: 5000,
  });

  const { data: ttftData } = useQuery({
    queryKey: ["ttft-histogram", activeCluster, poolName],
    queryFn: () => fetchTTFTHistogram(poolName),
    enabled: !!poolName && activeTab === "metrics",
    refetchInterval: 15000,
  });

  const { data: tpsData } = useQuery({
    queryKey: ["tps-throughput", activeCluster, poolName],
    queryFn: () => fetchTPSThroughput(poolName),
    enabled: !!poolName && activeTab === "metrics",
    refetchInterval: 15000,
  });

  const { data: queueData } = useQuery({
    queryKey: ["queue-depth", activeCluster, poolName],
    queryFn: () => fetchQueueDepthSeries(poolName),
    enabled: !!poolName && activeTab === "metrics",
    refetchInterval: 15000,
  });

  const { data: gpuUtilData } = useQuery({
    queryKey: ["gpu-util-series", activeCluster, poolName],
    queryFn: () => fetchGPUUtilSeries(poolName),
    enabled: !!poolName && activeTab === "metrics",
    refetchInterval: 15000,
  });

  const { data: kvCacheData } = useQuery({
    queryKey: ["kv-cache-series", activeCluster, poolName],
    queryFn: () => fetchKVCacheSeries(poolName),
    enabled: !!poolName && activeTab === "metrics",
    refetchInterval: 15000,
  });

  const { data: cost } = useQuery({
    queryKey: ["cost-estimate", activeCluster, poolName],
    queryFn: () => fetchCostEstimate(poolName),
    enabled: !!poolName && activeTab === "cost",
  });

  const deleteMutation = useMutation({
    mutationFn: () => deleteInferencePool(poolName),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["inference-pools"] });
      navigate("/inference/pools");
    },
    onError: (err: any) => {
      setDeleteError(err?.response?.data?.error || String(err));
    },
  });

  if (isLoading) return <p className="text-muted-foreground">Loading pool...</p>;
  if (error) return <p className="text-red-400">Failed to load pool: {String(error)}</p>;
  if (!pool) return <p className="text-muted-foreground">Pool not found.</p>;

  const tabs: { id: Tab; label: string }[] = [
    { id: "overview", label: "Overview" },
    { id: "epp", label: "EPP Decisions" },
    { id: "metrics", label: "Metrics" },
    { id: "cost", label: "Cost" },
  ];

  return (
    <div>
      {/* Header */}
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Link to="/inference/pools" className="hover:text-foreground">
          Pools
        </Link>
        <span>/</span>
        <span className="text-foreground">{pool.name}</span>
      </div>

      <div className="mt-3 flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">{pool.name}</h1>
          <p className="mt-1 text-sm text-muted-foreground">
            {pool.modelName} &middot; {pool.gpuType} x{pool.gpuCount} &middot; {pool.servingBackend}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Link
            to={`/inference/pools/${ns}/${name}/edit`}
            className="rounded-md border border-border px-3 py-1.5 text-sm font-medium text-foreground hover:bg-muted/30"
          >
            Edit
          </Link>
          <button
            onClick={() => { setShowDeleteConfirm(true); setDeleteError(null); }}
            className="rounded-md border border-red-500/30 bg-red-500/10 px-3 py-1.5 text-sm font-medium text-red-400 hover:bg-red-500/20"
          >
            Delete
          </button>
          <span className="rounded bg-muted px-2 py-1 text-xs text-muted-foreground">{pool.namespace}</span>
          <span
            className={`rounded-md border px-2 py-0.5 text-xs font-medium ${
              pool.status?.conditions[0]?.status === "Ready"
                ? "bg-emerald-500/15 text-emerald-400 border-emerald-500/30"
                : "bg-yellow-500/15 text-yellow-400 border-yellow-500/30"
            }`}
          >
            {pool.status?.conditions[0]?.status ?? "Unknown"}
          </span>
        </div>
      </div>

      {/* Tabs */}
      <div className="mt-6 flex gap-1 border-b border-border">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={`px-4 py-2 text-sm font-medium transition-colors ${
              activeTab === tab.id
                ? "border-b-2 border-blue-500 text-foreground"
                : "text-muted-foreground hover:text-foreground"
            }`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Tab content */}
      <div className="mt-6">
        {activeTab === "overview" && (
          <div className="space-y-6">
            {/* Summary cards */}
            <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
              <MetricCard title="Replicas" value={`${pool.status?.readyReplicas ?? 0}/${pool.replicas}`} />
              <MetricCard
                title="Avg GPU Util"
                value={summary ? `${summary.avgGPUUtil.toFixed(1)}%` : "--"}
                trend={summary && summary.avgGPUUtil > 80 ? "up" : "neutral"}
              />
              <MetricCard title="Avg TTFT" value={summary ? `${summary.avgTTFT.toFixed(0)}ms` : "--"} />
              <MetricCard title="Throughput" value={summary ? `${summary.avgTPS.toFixed(0)} tok/s` : "--"} />
            </div>

            {/* GPU Heatmap */}
            {pods && pods.length > 0 && <GPUHeatmap pods={pods} />}
          </div>
        )}

        {activeTab === "epp" && <EPPDecisionVisualizer pool={poolName} />}

        {activeTab === "metrics" && (
          <div className="space-y-6">
            {ttftData && <TTFTHistogram data={ttftData} />}
            <div className="grid gap-6 lg:grid-cols-2">
              {tpsData && <TimeseriesChart title="Tokens per Second" data={tpsData} unit=" tok/s" color="#3b82f6" />}
              {queueData && (
                <TimeseriesChart title="Queue Depth" data={queueData} unit="" color="#f59e0b" variant="area" />
              )}
              {gpuUtilData && (
                <TimeseriesChart title="GPU Utilization" data={gpuUtilData} unit="%" color="#10b981" variant="area" />
              )}
              {kvCacheData && (
                <TimeseriesChart title="KV Cache Utilization" data={kvCacheData} unit="%" color="#8b5cf6" variant="area" />
              )}
            </div>
          </div>
        )}

        {activeTab === "cost" && cost && (
          <div className="max-w-md">
            <CostEstimateCard cost={cost} />
          </div>
        )}
      </div>

      {/* Delete confirmation dialog */}
      {showDeleteConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="w-full max-w-md rounded-lg border border-border bg-background p-6 shadow-xl">
            <h3 className="text-lg font-semibold">Delete Inference Pool</h3>
            <p className="mt-2 text-sm text-muted-foreground">
              Are you sure you want to delete pool{" "}
              <span className="font-mono font-medium text-foreground">{pool.name}</span> in namespace{" "}
              <span className="font-mono font-medium text-foreground">{pool.namespace}</span>?
              This action cannot be undone.
            </p>

            {deleteError && (
              <div className="mt-3 rounded-md border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400">
                {deleteError}
              </div>
            )}

            <div className="mt-6 flex justify-end gap-3">
              <button
                onClick={() => { setShowDeleteConfirm(false); setDeleteError(null); }}
                className="rounded-md border border-border px-4 py-2 text-sm font-medium text-foreground hover:bg-muted/30"
              >
                Cancel
              </button>
              <button
                onClick={() => deleteMutation.mutate()}
                disabled={deleteMutation.isPending}
                className="rounded-md bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50"
              >
                {deleteMutation.isPending ? "Deleting..." : "Delete"}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
