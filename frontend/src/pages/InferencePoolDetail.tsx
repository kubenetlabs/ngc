import { useState } from "react";
import { useParams, Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import {
  fetchInferencePool,
  fetchInferenceMetricsSummary,
  fetchPodMetrics,
  fetchTTFTHistogram,
  fetchTPSThroughput,
  fetchQueueDepthSeries,
  fetchActiveRequestsSeries,
  fetchGPUUtilSeries,
  fetchKVCacheSeries,
  fetchCostEstimate,
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
  const { name } = useParams<{ ns: string; name: string }>();
  const activeCluster = useActiveCluster();
  const [activeTab, setActiveTab] = useState<Tab>("overview");
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

  const { data: activeReqData } = useQuery({
    queryKey: ["active-requests-series", activeCluster, poolName],
    queryFn: () => fetchActiveRequestsSeries(poolName),
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
              {activeReqData && (
                <TimeseriesChart title="Active Requests" data={activeReqData} unit="" color="#f59e0b" variant="area" />
              )}
              {gpuUtilData && (
                <TimeseriesChart title="GPU Utilization" data={gpuUtilData} unit="%" color="#10b981" variant="area" />
              )}
              {kvCacheData && (
                <TimeseriesChart title="KV Cache Utilization" data={kvCacheData} unit="%" color="#8b5cf6" variant="area" />
              )}
              {queueData && (
                <TimeseriesChart title="Queue Depth (Waiting)" data={queueData} unit="" color="#ef4444" variant="area" />
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
    </div>
  );
}
