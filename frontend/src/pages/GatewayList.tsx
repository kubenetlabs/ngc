import { useQuery } from "@tanstack/react-query";
import { Link, useSearchParams } from "react-router-dom";
import { fetchGateways } from "@/api/gateways";
import { StatusBadge } from "@/components/common/StatusBadge";
import { ErrorState } from "@/components/common/ErrorState";
import { useActiveCluster } from "@/hooks/useActiveCluster";
import { ALL_CLUSTERS } from "@/store/clusterStore";
import { GlobalGatewayList } from "@/components/global/GlobalGatewayList";
import type { Gateway } from "@/types/gateway";

function timeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime();
  const days = Math.floor(diff / 86_400_000);
  if (days > 0) return `${days}d ago`;
  const hours = Math.floor(diff / 3_600_000);
  if (hours > 0) return `${hours}h ago`;
  const mins = Math.floor(diff / 60_000);
  return `${mins}m ago`;
}

function attachedRouteCount(gw: Gateway): number {
  return gw.status?.listeners.reduce((sum, l) => sum + l.attachedRoutes, 0) ?? 0;
}

function SingleClusterGatewayList() {
  const [searchParams, setSearchParams] = useSearchParams();
  const nsFilter = searchParams.get("namespace") ?? "";
  const activeCluster = useActiveCluster();

  const { data: gateways, isLoading, error, refetch } = useQuery({
    queryKey: ["gateways", activeCluster, nsFilter],
    queryFn: () => fetchGateways(nsFilter || undefined),
  });

  return (
    <div>
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Gateways</h1>
          <p className="mt-1 text-muted-foreground">Manage Gateway resources.</p>
        </div>
        <div className="flex items-center gap-3">
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
          <Link
            to="/gateways/create"
            className="rounded-md bg-blue-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-blue-700"
          >
            Create Gateway
          </Link>
        </div>
      </div>

      {isLoading && <p className="mt-6 text-muted-foreground">Loading gateways...</p>}
      {error && <ErrorState error={error as Error} onRetry={() => refetch()} message="Failed to load gateways" />}

      {gateways && gateways.length === 0 && (
        <p className="mt-6 text-muted-foreground">No gateways found.</p>
      )}

      {gateways && gateways.length > 0 && (
        <div className="mt-6 overflow-x-auto rounded-lg border border-border">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-muted/30">
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Name</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Namespace</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Class</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Listeners</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Routes</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Status</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Age</th>
              </tr>
            </thead>
            <tbody>
              {gateways.map((gw) => (
                <tr key={`${gw.namespace}/${gw.name}`} className="border-b border-border last:border-0 hover:bg-muted/20">
                  <td className="px-4 py-3">
                    <Link
                      to={`/gateways/${gw.namespace}/${gw.name}`}
                      className="font-medium text-blue-400 hover:underline"
                    >
                      {gw.name}
                    </Link>
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">{gw.namespace}</td>
                  <td className="px-4 py-3 text-muted-foreground">{gw.gatewayClassName}</td>
                  <td className="px-4 py-3 text-muted-foreground">{gw.listeners.length}</td>
                  <td className="px-4 py-3 text-muted-foreground">{attachedRouteCount(gw)}</td>
                  <td className="px-4 py-3">
                    <div className="flex flex-wrap gap-1">
                      {gw.status?.conditions.map((c) => (
                        <StatusBadge key={c.type} condition={c} />
                      ))}
                    </div>
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">{timeAgo(gw.createdAt)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

export default function GatewayList() {
  const activeCluster = useActiveCluster();

  if (activeCluster === ALL_CLUSTERS) {
    return <GlobalGatewayList />;
  }

  return <SingleClusterGatewayList />;
}
