import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { fetchClusters, unregisterCluster, getClusterSummary } from "@/api/clusters";
import { ClusterHealthCard } from "@/components/clusters/ClusterHealthCard";
import { Plus, Trash2 } from "lucide-react";
import { useState } from "react";

export default function ClusterManagement() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [confirmDelete, setConfirmDelete] = useState<string | null>(null);

  const { data: clusters, isLoading } = useQuery({
    queryKey: ["clusters"],
    queryFn: fetchClusters,
  });

  const { data: summary } = useQuery({
    queryKey: ["clusters", "summary"],
    queryFn: getClusterSummary,
  });

  const deleteMutation = useMutation({
    mutationFn: unregisterCluster,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["clusters"] });
      setConfirmDelete(null);
    },
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <p className="text-muted-foreground">Loading clusters...</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">Clusters</h1>
          <p className="text-sm text-muted-foreground">
            Manage connected Kubernetes clusters
          </p>
        </div>
        <button
          onClick={() => navigate("/clusters/register")}
          className="inline-flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
        >
          <Plus className="h-4 w-4" />
          Add Cluster
        </button>
      </div>

      {summary && (
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-5">
          {[
            { label: "Total Clusters", value: summary.totalClusters },
            { label: "Healthy", value: summary.healthyClusters },
            { label: "Gateways", value: summary.totalGateways },
            { label: "Routes", value: summary.totalRoutes },
            { label: "GPUs", value: summary.totalGPUs },
          ].map((stat) => (
            <div
              key={stat.label}
              className="rounded-lg border border-border bg-card p-3 text-center"
            >
              <p className="text-2xl font-bold text-foreground">{stat.value}</p>
              <p className="text-xs text-muted-foreground">{stat.label}</p>
            </div>
          ))}
        </div>
      )}

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {clusters?.map((cluster) => (
          <div key={cluster.name} className="relative">
            <ClusterHealthCard
              cluster={cluster}
              onClick={() => navigate(`/clusters/${cluster.name}`)}
            />
            {!cluster.isLocal && (
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  setConfirmDelete(cluster.name);
                }}
                className="absolute right-2 top-2 rounded p-1 text-muted-foreground opacity-0 transition-opacity hover:bg-destructive/10 hover:text-destructive group-hover:opacity-100 [div:hover>&]:opacity-100"
                title="Remove cluster"
              >
                <Trash2 className="h-3.5 w-3.5" />
              </button>
            )}
          </div>
        ))}
      </div>

      {confirmDelete && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-background/80">
          <div className="rounded-lg border border-border bg-card p-6 shadow-lg">
            <h3 className="text-lg font-semibold text-foreground">
              Remove Cluster
            </h3>
            <p className="mt-2 text-sm text-muted-foreground">
              Are you sure you want to remove{" "}
              <strong>{confirmDelete}</strong>? This will delete the
              ManagedCluster resource and its kubeconfig Secret.
            </p>
            <div className="mt-4 flex justify-end gap-2">
              <button
                onClick={() => setConfirmDelete(null)}
                className="rounded-md border border-border px-3 py-1.5 text-sm text-foreground hover:bg-accent"
              >
                Cancel
              </button>
              <button
                onClick={() => deleteMutation.mutate(confirmDelete)}
                className="rounded-md bg-destructive px-3 py-1.5 text-sm text-destructive-foreground hover:bg-destructive/90"
              >
                Remove
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
