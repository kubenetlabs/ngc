import { useQuery } from "@tanstack/react-query";
import { useSearchParams } from "react-router-dom";
import { fetchHTTPRoutes } from "@/api/routes";
import { StatusBadge } from "@/components/common/StatusBadge";
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

export default function RouteList() {
  const [searchParams, setSearchParams] = useSearchParams();
  const nsFilter = searchParams.get("namespace") ?? "";
  const activeCluster = useActiveCluster();

  const { data: routes, isLoading, error } = useQuery({
    queryKey: ["httproutes", activeCluster, nsFilter],
    queryFn: () => fetchHTTPRoutes(nsFilter || undefined),
  });

  return (
    <div>
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Routes</h1>
          <p className="mt-1 text-muted-foreground">Manage HTTP, gRPC, TLS, TCP, and UDP routes.</p>
        </div>
        <div>
          <input
            type="text"
            placeholder="Filter by namespace..."
            value={nsFilter}
            onChange={(e) => {
              const ns = e.target.value;
              if (ns) setSearchParams({ namespace: ns });
              else setSearchParams({});
            }}
            className="rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring"
          />
        </div>
      </div>

      {isLoading && <p className="mt-6 text-muted-foreground">Loading routes...</p>}
      {error && <p className="mt-6 text-red-400">Failed to load routes: {String(error)}</p>}

      {routes && routes.length === 0 && (
        <p className="mt-6 text-muted-foreground">No routes found.</p>
      )}

      {routes && routes.length > 0 && (
        <div className="mt-6 overflow-x-auto rounded-lg border border-border">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-muted/30">
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Name</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Namespace</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Hostnames</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Parent Gateway</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Rules</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Status</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Age</th>
              </tr>
            </thead>
            <tbody>
              {routes.map((route) => (
                <tr key={`${route.namespace}/${route.name}`} className="border-b border-border last:border-0 hover:bg-muted/20">
                  <td className="px-4 py-3 font-medium text-foreground">{route.name}</td>
                  <td className="px-4 py-3 text-muted-foreground">{route.namespace}</td>
                  <td className="px-4 py-3 font-mono text-muted-foreground">
                    {route.hostnames?.join(", ") || "*"}
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">
                    {route.parentRefs.map((p) => p.name).join(", ")}
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">{route.rules.length}</td>
                  <td className="px-4 py-3">
                    <div className="flex flex-wrap gap-1">
                      {route.status?.parents.flatMap((p) =>
                        p.conditions.map((c) => (
                          <StatusBadge key={`${p.parentRef.name}-${c.type}`} condition={c} />
                        )),
                      )}
                    </div>
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">{timeAgo(route.createdAt)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
