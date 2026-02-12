import { useQuery, useQueryClient } from "@tanstack/react-query";
import { fetchClusters } from "@/api/clusters";
import { useClusterStore, ALL_CLUSTERS } from "@/store/clusterStore";

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
  const selected =
    activeCluster ||
    clusters.find((c) => c.default)?.name ||
    clusters[0].name;

  // Group clusters by environment
  const envGroups = new Map<string, typeof clusters>();
  for (const c of clusters) {
    const env = c.environment || "other";
    if (!envGroups.has(env)) {
      envGroups.set(env, []);
    }
    envGroups.get(env)!.push(c);
  }

  const selectedCluster = clusters.find((c) => c.name === selected);

  return (
    <div className="flex items-center gap-2">
      <select
        value={selected}
        onChange={handleChange}
        className="rounded-md border border-border bg-background px-2 py-1 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
      >
        <option value={ALL_CLUSTERS}>All Clusters</option>
        {envGroups.size > 1 ? (
          Array.from(envGroups.entries()).map(([env, group]) => (
            <optgroup
              key={env}
              label={env.charAt(0).toUpperCase() + env.slice(1)}
            >
              {group.map((c) => (
                <option key={c.name} value={c.name}>
                  {c.displayName || c.name}
                  {c.region ? ` (${c.region})` : ""}
                </option>
              ))}
            </optgroup>
          ))
        ) : (
          clusters.map((c) => (
            <option key={c.name} value={c.name}>
              {c.displayName || c.name}
              {c.region ? ` (${c.region})` : ""}
            </option>
          ))
        )}
      </select>
      {selected !== ALL_CLUSTERS && selectedCluster && (
        <span
          className={`h-2 w-2 rounded-full ${selectedCluster.connected ? "bg-green-500" : "bg-red-500"}`}
          title={selectedCluster.connected ? "Connected" : "Disconnected"}
        />
      )}
      {selected === ALL_CLUSTERS && (
        <span className="text-[10px] text-muted-foreground">
          {clusters.filter((c) => c.connected).length}/{clusters.length}
        </span>
      )}
    </div>
  );
}
