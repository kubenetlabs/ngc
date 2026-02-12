import { useQuery } from "@tanstack/react-query";
import { fetchClusters, getClusterSummary } from "@/api/clusters";
import { fetchGlobalGPUCapacity } from "@/api/global";
import { ClusterHealthCard } from "@/components/clusters/ClusterHealthCard";
import { useNavigate } from "react-router-dom";

export function GlobalDashboard() {
  const navigate = useNavigate();

  const { data: clusters } = useQuery({
    queryKey: ["clusters"],
    queryFn: fetchClusters,
  });

  const { data: summary } = useQuery({
    queryKey: ["clusters", "summary"],
    queryFn: getClusterSummary,
  });

  const { data: gpuCapacity } = useQuery({
    queryKey: ["global", "gpu-capacity"],
    queryFn: fetchGlobalGPUCapacity,
  });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">
          Multi-Cluster Overview
        </h1>
        <p className="text-sm text-muted-foreground">
          Aggregated view across all connected clusters
        </p>
      </div>

      {summary && (
        <div className="grid grid-cols-2 gap-4 sm:grid-cols-5">
          {[
            { label: "Total Clusters", value: summary.totalClusters },
            { label: "Healthy", value: summary.healthyClusters },
            { label: "Gateways", value: summary.totalGateways },
            { label: "Routes", value: summary.totalRoutes },
            { label: "GPUs", value: gpuCapacity?.totalGPUs ?? summary.totalGPUs },
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

      {gpuCapacity && gpuCapacity.clusters.length > 0 && (
        <div>
          <h2 className="mb-3 text-lg font-semibold text-foreground">
            GPU Capacity by Cluster
          </h2>
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {gpuCapacity.clusters.map((cluster) => (
              <div
                key={cluster.clusterName}
                className="rounded-lg border border-border bg-card p-4"
              >
                <div className="flex items-center justify-between">
                  <p className="text-sm font-medium text-foreground">
                    {cluster.clusterName}
                  </p>
                  <span className="text-xs text-muted-foreground">
                    {cluster.clusterRegion}
                  </span>
                </div>
                <div className="mt-2 flex items-end gap-4">
                  <div>
                    <p className="text-2xl font-bold text-foreground">
                      {cluster.totalGPUs}
                    </p>
                    <p className="text-[10px] text-muted-foreground">
                      Total GPUs
                    </p>
                  </div>
                  <div>
                    <p className="text-lg font-semibold text-foreground">
                      {cluster.allocatedGPUs}
                    </p>
                    <p className="text-[10px] text-muted-foreground">
                      Allocated
                    </p>
                  </div>
                  {cluster.totalGPUs > 0 && (
                    <div className="flex-1">
                      <div className="h-2 w-full rounded-full bg-muted">
                        <div
                          className="h-2 rounded-full bg-primary"
                          style={{
                            width: `${Math.min(100, (cluster.allocatedGPUs / cluster.totalGPUs) * 100)}%`,
                          }}
                        />
                      </div>
                      <p className="mt-0.5 text-right text-[10px] text-muted-foreground">
                        {Math.round(
                          (cluster.allocatedGPUs / cluster.totalGPUs) * 100,
                        )}
                        % used
                      </p>
                    </div>
                  )}
                </div>
                {cluster.gpuTypes && (
                  <div className="mt-2 flex flex-wrap gap-1">
                    {Object.entries(cluster.gpuTypes).map(([type_, count]) => (
                      <span
                        key={type_}
                        className="rounded bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground"
                      >
                        {type_}: {count}
                      </span>
                    ))}
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      <div>
        <h2 className="mb-3 text-lg font-semibold text-foreground">
          Cluster Health
        </h2>
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {clusters?.map((cluster) => (
            <ClusterHealthCard
              key={cluster.name}
              cluster={cluster}
              onClick={() => navigate(`/clusters/${cluster.name}`)}
            />
          ))}
        </div>
      </div>
    </div>
  );
}
