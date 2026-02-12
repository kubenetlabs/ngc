import { useQuery } from "@tanstack/react-query";
import { fetchGlobalRoutes } from "@/api/global";
import { ClusterBadge } from "@/components/clusters/ClusterBadge";
import { useNavigate } from "react-router-dom";
import { useClusterStore } from "@/store/clusterStore";

export function GlobalRouteList() {
  const navigate = useNavigate();
  const setActiveCluster = useClusterStore((s) => s.setActiveCluster);

  const { data: routes, isLoading } = useQuery({
    queryKey: ["global", "routes"],
    queryFn: fetchGlobalRoutes,
  });

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <p className="text-muted-foreground">
          Loading routes from all clusters...
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">
            Routes — All Clusters
          </h1>
          <p className="text-sm text-muted-foreground">
            {routes?.length ?? 0} routes across all clusters
          </p>
        </div>
      </div>
      <div className="overflow-hidden rounded-lg border border-border">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border bg-muted/50">
              <th className="px-4 py-2 text-left font-medium text-muted-foreground">
                Cluster
              </th>
              <th className="px-4 py-2 text-left font-medium text-muted-foreground">
                Name
              </th>
              <th className="px-4 py-2 text-left font-medium text-muted-foreground">
                Namespace
              </th>
              <th className="px-4 py-2 text-left font-medium text-muted-foreground">
                Hostnames
              </th>
              <th className="px-4 py-2 text-left font-medium text-muted-foreground">
                Parent
              </th>
              <th className="px-4 py-2 text-left font-medium text-muted-foreground">
                Status
              </th>
            </tr>
          </thead>
          <tbody>
            {routes?.map((item, i) => (
              <tr
                key={`${item.clusterName}-${item.route.namespace}-${item.route.name}-${i}`}
                className="cursor-pointer border-b border-border last:border-0 hover:bg-accent/50"
                onClick={() => {
                  setActiveCluster(item.clusterName);
                  navigate(
                    `/routes/${item.route.namespace}/${item.route.name}`,
                  );
                }}
              >
                <td className="px-4 py-2">
                  <ClusterBadge
                    name={item.clusterName}
                    region={item.clusterRegion}
                  />
                </td>
                <td className="px-4 py-2 font-medium text-foreground">
                  {item.route.name}
                </td>
                <td className="px-4 py-2 text-muted-foreground">
                  {item.route.namespace}
                </td>
                <td className="px-4 py-2 text-muted-foreground">
                  {item.route.hostnames?.join(", ") || "—"}
                </td>
                <td className="px-4 py-2 text-muted-foreground">
                  {item.route.parentRefs
                    ?.map((p) => p.name)
                    .join(", ") || "—"}
                </td>
                <td className="px-4 py-2">
                  <span
                    className={`rounded-full px-2 py-0.5 text-xs font-medium ${
                      item.route.status === "Accepted"
                        ? "bg-green-500/20 text-green-500"
                        : "bg-yellow-500/20 text-yellow-500"
                    }`}
                  >
                    {item.route.status}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
