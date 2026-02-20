import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { fetchInferencePools, deleteInferencePool } from "@/api/inference";
import { GPUUtilizationBar } from "@/components/inference/GPUUtilizationBar";
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
  const queryClient = useQueryClient();
  const [deletingPool, setDeletingPool] = useState<{ name: string; namespace: string } | null>(null);
  const [deleteError, setDeleteError] = useState<string | null>(null);

  const { data: pools, isLoading, error } = useQuery({
    queryKey: ["inference-pools", activeCluster],
    queryFn: fetchInferencePools,
    refetchInterval: 10000,
  });

  const deleteMutation = useMutation({
    mutationFn: (name: string) => deleteInferencePool(name),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["inference-pools"] });
      setDeletingPool(null);
      setDeleteError(null);
    },
    onError: (err: any) => {
      setDeleteError(err?.response?.data?.error || String(err));
    },
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
      {error && <p className="mt-6 text-red-400">Failed to load pools: {String(error)}</p>}

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
                <th className="px-4 py-3 text-right font-medium text-muted-foreground">Actions</th>
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
                    <td className="px-4 py-3 text-right">
                      <div className="flex items-center justify-end gap-1">
                        <Link
                          to={`/inference/pools/${pool.namespace}/${pool.name}/edit`}
                          className="rounded p-1.5 text-muted-foreground hover:bg-muted/50 hover:text-foreground"
                          title="Edit pool"
                        >
                          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                            <path d="M17 3a2.85 2.83 0 1 1 4 4L7.5 20.5 2 22l1.5-5.5Z" />
                            <path d="m15 5 4 4" />
                          </svg>
                        </Link>
                        <button
                          onClick={() => { setDeletingPool({ name: pool.name, namespace: pool.namespace }); setDeleteError(null); }}
                          className="rounded p-1.5 text-muted-foreground hover:bg-red-500/10 hover:text-red-400"
                          title="Delete pool"
                        >
                          <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                            <path d="M3 6h18" />
                            <path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6" />
                            <path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2" />
                            <line x1="10" x2="10" y1="11" y2="17" />
                            <line x1="14" x2="14" y1="11" y2="17" />
                          </svg>
                        </button>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {/* Delete confirmation dialog */}
      {deletingPool && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="w-full max-w-md rounded-lg border border-border bg-background p-6 shadow-xl">
            <h3 className="text-lg font-semibold">Delete Inference Pool</h3>
            <p className="mt-2 text-sm text-muted-foreground">
              Are you sure you want to delete pool{" "}
              <span className="font-mono font-medium text-foreground">{deletingPool.name}</span> in namespace{" "}
              <span className="font-mono font-medium text-foreground">{deletingPool.namespace}</span>?
              This action cannot be undone.
            </p>

            {deleteError && (
              <div className="mt-3 rounded-md border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-400">
                {deleteError}
              </div>
            )}

            <div className="mt-6 flex justify-end gap-3">
              <button
                onClick={() => { setDeletingPool(null); setDeleteError(null); }}
                className="rounded-md border border-border px-4 py-2 text-sm font-medium text-foreground hover:bg-muted/30"
              >
                Cancel
              </button>
              <button
                onClick={() => deleteMutation.mutate(deletingPool.name)}
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
