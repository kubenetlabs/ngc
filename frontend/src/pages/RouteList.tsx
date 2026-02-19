import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useSearchParams, Link } from "react-router-dom";
import { fetchHTTPRoutes } from "@/api/routes";
import { fetchGRPCRoutes } from "@/api/grpcroutes";
import { fetchTLSRoutes } from "@/api/tlsroutes";
import { fetchTCPRoutes } from "@/api/tcproutes";
import { fetchUDPRoutes } from "@/api/udproutes";
import { StatusBadge } from "@/components/common/StatusBadge";
import { ErrorState } from "@/components/common/ErrorState";
import { useActiveCluster } from "@/hooks/useActiveCluster";
import { ALL_CLUSTERS } from "@/store/clusterStore";
import { GlobalRouteList } from "@/components/global/GlobalRouteList";
import type { RouteType } from "@/types/route";

const ROUTE_TYPES: { label: string; value: RouteType }[] = [
  { label: "HTTP", value: "HTTPRoute" },
  { label: "gRPC", value: "GRPCRoute" },
  { label: "TLS", value: "TLSRoute" },
  { label: "TCP", value: "TCPRoute" },
  { label: "UDP", value: "UDPRoute" },
];

function timeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime();
  const days = Math.floor(diff / 86_400_000);
  if (days > 0) return `${days}d ago`;
  const hours = Math.floor(diff / 3_600_000);
  if (hours > 0) return `${hours}h ago`;
  const mins = Math.floor(diff / 60_000);
  return `${mins}m ago`;
}

function useFetchRoutes(routeType: RouteType, namespace: string | undefined, activeCluster: string) {
  return useQuery({
    queryKey: [routeType.toLowerCase() + "s", activeCluster, namespace ?? ""],
    queryFn: () => {
      const ns = namespace || undefined;
      switch (routeType) {
        case "HTTPRoute": return fetchHTTPRoutes(ns);
        case "GRPCRoute": return fetchGRPCRoutes(ns);
        case "TLSRoute": return fetchTLSRoutes(ns);
        case "TCPRoute": return fetchTCPRoutes(ns);
        case "UDPRoute": return fetchUDPRoutes(ns);
      }
    },
  });
}

function SingleClusterRouteList() {
  const [searchParams, setSearchParams] = useSearchParams();
  const nsFilter = searchParams.get("namespace") ?? "";
  const activeCluster = useActiveCluster();
  const [routeType, setRouteType] = useState<RouteType>("HTTPRoute");

  const { data: routes, isLoading, error, refetch } = useFetchRoutes(routeType, nsFilter || undefined, activeCluster);

  return (
    <div>
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Routes</h1>
          <p className="mt-1 text-muted-foreground">Manage HTTP, gRPC, TLS, TCP, and UDP routes.</p>
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
            to={`/routes/create/${routeType}`}
            className="rounded-md bg-blue-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-blue-700"
          >
            Create Route
          </Link>
        </div>
      </div>

      {/* Route Type Tabs */}
      <div className="mt-4 flex gap-1 rounded-lg border border-border bg-muted/20 p-1">
        {ROUTE_TYPES.map((rt) => (
          <button
            key={rt.value}
            onClick={() => setRouteType(rt.value)}
            className={`rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
              routeType === rt.value
                ? "bg-background text-foreground shadow-sm"
                : "text-muted-foreground hover:text-foreground"
            }`}
          >
            {rt.label}
          </button>
        ))}
      </div>

      {isLoading && <p className="mt-6 text-muted-foreground">Loading routes...</p>}
      {error && <ErrorState error={error as Error} onRetry={() => refetch()} message="Failed to load routes" />}

      {routes && routes.length === 0 && (
        <p className="mt-6 text-muted-foreground">No {routeType}s found.</p>
      )}

      {routes && routes.length > 0 && (
        <div className="mt-6 overflow-x-auto rounded-lg border border-border">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-muted/30">
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Name</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Namespace</th>
                {(routeType === "HTTPRoute" || routeType === "GRPCRoute" || routeType === "TLSRoute") && (
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">Hostnames</th>
                )}
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Parent Gateway</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Rules</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Status</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Age</th>
              </tr>
            </thead>
            <tbody>
              {routes.map((route: any) => (
                <tr key={`${route.namespace}/${route.name}`} className="border-b border-border last:border-0 hover:bg-muted/20">
                  <td className="px-4 py-3 font-medium">
                    <Link to={`/routes/${route.namespace}/${route.name}`} className="text-blue-400 hover:underline">
                      {route.name}
                    </Link>
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">{route.namespace}</td>
                  {(routeType === "HTTPRoute" || routeType === "GRPCRoute" || routeType === "TLSRoute") && (
                    <td className="px-4 py-3 font-mono text-muted-foreground">
                      {route.hostnames?.join(", ") || "*"}
                    </td>
                  )}
                  <td className="px-4 py-3 text-muted-foreground">
                    {route.parentRefs?.map((p: any) => p.name).join(", ")}
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">{route.rules?.length ?? 0}</td>
                  <td className="px-4 py-3">
                    <div className="flex flex-wrap gap-1">
                      {route.status?.parents?.flatMap((p: any) =>
                        p.conditions?.map((c: any) => (
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

export default function RouteList() {
  const activeCluster = useActiveCluster();

  if (activeCluster === ALL_CLUSTERS) {
    return <GlobalRouteList />;
  }

  return <SingleClusterRouteList />;
}
