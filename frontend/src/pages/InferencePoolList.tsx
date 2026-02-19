import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { fetchInferencePools } from "@/api/inference";
import { GPUUtilizationBar } from "@/components/inference/GPUUtilizationBar";
import { ErrorState } from "@/components/common/ErrorState";
import { useActiveCluster } from "@/hooks/useActiveCluster";

function timeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime();
  const days = Math.floor(diff / 86_400_000);
  if (days > 0) return `${days}d ago`;
  const hours = Math.floor(diff / 3_600_000);
  if (hours > 0) return `${hours}h ago`;
  const mins = Math.floor(diff / 60_000);
  return `${mins}m ago`;
}

export default function InferencePoolList() {
  const activeCluster = useActiveCluster();

  const { data: pools, isLoading, error, refetch } = useQuery({
    queryKey: ["inference-pools", activeCluster],
    queryFn: fetchInferencePools,
    refetchInterval: 10000,
  });

  return (
    <div>
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Inference Pools</h1>
          <p className="mt-1 text-muted-foreground">GPU-aware inference pool management.</p>
        </div>
        <div className="flex items-center gap-2">
          <Link
            to="/inference"
            className="rounded-md border border-border px-3 py-1.5 text-sm text-muted-foreground hover:bg-muted/30"
          >
            Overview
          </Link>
          <Link
            to="create"
            className="rounded-md bg-blue-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-blue-500"
          >
            Create Pool
          </Link>
        </div>
      </div>

      {isLoading && <p className="mt-6 text-muted-foreground">Loading pools...</p>}
      {error && <ErrorState error={error as Error} onRetry={() => refetch()} message="Failed to load inference pools" />}

      {pools && pools.length === 0 && (
        <p className="mt-6 text-muted-foreground">No inference pools found.</p>
      )}

      {pools && pools.length > 0 && (
        <div className="mt-6 overflow-x-auto rounded-lg border border-border">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-muted/30">
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Name</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Namespace</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Model</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">GPU</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Replicas</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">GPU Util</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Status</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Age</th>
              </tr>
            </thead>
            <tbody>
              {pools.map((pool) => {
                const statusColor =
                  pool.status?.conditions[0]?.status === "Ready"
                    ? "bg-emerald-500/15 text-emerald-400 border-emerald-500/30"
                    : pool.status?.conditions[0]?.status === "Degraded"
                      ? "bg-yellow-500/15 text-yellow-400 border-yellow-500/30"
                      : "bg-zinc-500/15 text-zinc-400 border-zinc-500/30";

                return (
                  <tr
                    key={`${pool.namespace}/${pool.name}`}
                    className="border-b border-border last:border-0 hover:bg-muted/20"
                  >
                    <td className="px-4 py-3">
                      <Link
                        to={`/inference/pools/${pool.namespace}/${pool.name}`}
                        className="font-medium text-blue-400 hover:underline"
                      >
                        {pool.name}
                      </Link>
                    </td>
                    <td className="px-4 py-3 text-muted-foreground">{pool.namespace}</td>
                    <td className="px-4 py-3">
                      <div className="max-w-[200px] truncate text-muted-foreground" title={pool.modelName}>
                        {pool.modelName.split("/").pop()}
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <span className="rounded bg-muted px-1.5 py-0.5 text-xs font-medium text-foreground">
                        {pool.gpuType} x{pool.gpuCount}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-muted-foreground">
                      {pool.status?.readyReplicas ?? 0}/{pool.replicas}
                    </td>
                    <td className="px-4 py-3">
                      <GPUUtilizationBar value={pool.avgGpuUtil} />
                    </td>
                    <td className="px-4 py-3">
                      <span className={`inline-flex rounded-md border px-2 py-0.5 text-xs font-medium ${statusColor}`}>
                        {pool.status?.conditions[0]?.status ?? "Unknown"}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-muted-foreground">{timeAgo(pool.createdAt)}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
