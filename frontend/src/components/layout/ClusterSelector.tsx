import { useQuery, useQueryClient } from "@tanstack/react-query";
import { fetchClusters } from "@/api/clusters";
import { useClusterStore } from "@/store/clusterStore";

export function ClusterSelector() {
  const queryClient = useQueryClient();
  const { activeCluster, setActiveCluster } = useClusterStore();

  const { data: clusters } = useQuery({
    queryKey: ["clusters"],
    queryFn: fetchClusters,
    staleTime: 30_000,
  });

  // Hide when only one cluster (single-cluster mode)
  if (!clusters || clusters.length <= 1) {
    return null;
  }

  const handleChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const name = e.target.value;
    setActiveCluster(name);
    // Invalidate all queries except "clusters" so data refreshes for the new cluster
    queryClient.invalidateQueries({
      predicate: (query) => query.queryKey[0] !== "clusters",
    });
  };

  // Default to the cluster marked as default if no active cluster is set
  const selected = activeCluster || clusters.find((c) => c.default)?.name || clusters[0].name;

  return (
    <div className="flex items-center gap-2">
      <select
        value={selected}
        onChange={handleChange}
        className="rounded-md border border-border bg-background px-2 py-1 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
      >
        {clusters.map((c) => (
          <option key={c.name} value={c.name}>
            <span>{c.displayName || c.name}</span>
          </option>
        ))}
      </select>
      {clusters.map((c) =>
        c.name === selected ? (
          <span
            key={c.name}
            className={`h-2 w-2 rounded-full ${c.connected ? "bg-green-500" : "bg-red-500"}`}
            title={c.connected ? "Connected" : "Disconnected"}
          />
        ) : null,
      )}
    </div>
  );
}
