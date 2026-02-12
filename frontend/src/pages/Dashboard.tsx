import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import {
  Network,
  Route,
  Cpu,
  Shield,
  Lock,
  FileText,
  Zap,
  Plus,
  Search,
} from "lucide-react";
import { Link } from "react-router-dom";
import { fetchGateways } from "@/api/gateways";
import { fetchHTTPRoutes } from "@/api/routes";
import { fetchConfig } from "@/api/config";
import { fetchTopology } from "@/api/topology";
import { fetchCertificates } from "@/api/certificates";
import { fetchPolicies } from "@/api/policies";
import { fetchAuditEntries } from "@/api/audit";
import { fetchClusters } from "@/api/clusters";
import { TopologyGraph } from "@/components/topology/TopologyGraph";
import { useEdition } from "@/hooks/useEdition";
import { useActiveCluster } from "@/hooks/useActiveCluster";
import { ALL_CLUSTERS } from "@/store/clusterStore";
import { GlobalDashboard } from "@/components/global/GlobalDashboard";
import type { Gateway } from "@/types/gateway";

function SummaryCard({
  label,
  value,
  icon: Icon,
  loading,
}: {
  label: string;
  value: string | number;
  icon: React.ElementType;
  loading?: boolean;
}) {
  return (
    <div className="rounded-lg border border-border bg-card p-5">
      <div className="flex items-center justify-between">
        <span className="text-sm text-muted-foreground">{label}</span>
        <Icon className="h-5 w-5 text-muted-foreground" />
      </div>
      <div className="mt-2 text-3xl font-bold text-foreground">
        {loading ? (
          <span className="animate-pulse text-muted-foreground">--</span>
        ) : (
          value
        )}
      </div>
    </div>
  );
}

function QuickActions() {
  const actions = [
    {
      label: "Create Gateway",
      href: "/gateways/create",
      icon: Plus,
      description: "Deploy a new gateway",
    },
    {
      label: "Create Route",
      href: "/routes/create/HTTPRoute",
      icon: Route,
      description: "Add an HTTP route",
    },
    {
      label: "Create Inference Pool",
      href: "/inference",
      icon: Zap,
      description: "Set up AI inference",
    },
    {
      label: "Run Diagnostics",
      href: "/diagnostics",
      icon: Search,
      description: "Check cluster health",
    },
  ];

  return (
    <div>
      <h2 className="mb-3 text-lg font-semibold">Quick Actions</h2>
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        {actions.map((action) => (
          <Link
            key={action.label}
            to={action.href}
            className="group flex items-center gap-3 rounded-lg border border-border bg-card p-4 transition-colors hover:border-blue-500/50 hover:bg-muted/30"
          >
            <div className="flex h-9 w-9 items-center justify-center rounded-md bg-blue-600/10 text-blue-400 transition-colors group-hover:bg-blue-600/20">
              <action.icon className="h-4 w-4" />
            </div>
            <div>
              <p className="text-sm font-medium text-foreground">
                {action.label}
              </p>
              <p className="text-xs text-muted-foreground">
                {action.description}
              </p>
            </div>
          </Link>
        ))}
      </div>
    </div>
  );
}

function ResourceHealth({ gateways }: { gateways: Gateway[] }) {
  let healthy = 0;
  let degraded = 0;
  let errored = 0;

  for (const gw of gateways) {
    if (!gw.status?.conditions || gw.status.conditions.length === 0) {
      degraded++;
      continue;
    }

    const acceptedCondition = gw.status.conditions.find(
      (c) => c.type === "Accepted" || c.type === "Programmed",
    );

    if (acceptedCondition?.status === "True") {
      healthy++;
    } else if (acceptedCondition?.status === "False") {
      errored++;
    } else {
      degraded++;
    }
  }

  const total = gateways.length;

  return (
    <div>
      <h2 className="mb-3 text-lg font-semibold">Resource Health</h2>
      <div className="rounded-lg border border-border bg-card p-5">
        <div className="mb-4 flex items-center justify-between">
          <span className="text-sm text-muted-foreground">
            Gateway Health Overview
          </span>
          <span className="text-sm text-muted-foreground">
            {total} total
          </span>
        </div>

        {total === 0 ? (
          <p className="text-sm text-muted-foreground">
            No gateways deployed yet.
          </p>
        ) : (
          <>
            {/* Health bar */}
            <div className="mb-4 flex h-3 overflow-hidden rounded-full bg-muted">
              {healthy > 0 && (
                <div
                  className="bg-emerald-500 transition-all"
                  style={{ width: `${(healthy / total) * 100}%` }}
                />
              )}
              {degraded > 0 && (
                <div
                  className="bg-yellow-500 transition-all"
                  style={{ width: `${(degraded / total) * 100}%` }}
                />
              )}
              {errored > 0 && (
                <div
                  className="bg-red-500 transition-all"
                  style={{ width: `${(errored / total) * 100}%` }}
                />
              )}
            </div>

            {/* Legend */}
            <div className="flex gap-6">
              <div className="flex items-center gap-2">
                <span className="h-2.5 w-2.5 rounded-full bg-emerald-500" />
                <span className="text-sm text-muted-foreground">
                  Healthy ({healthy})
                </span>
              </div>
              <div className="flex items-center gap-2">
                <span className="h-2.5 w-2.5 rounded-full bg-yellow-500" />
                <span className="text-sm text-muted-foreground">
                  Degraded ({degraded})
                </span>
              </div>
              <div className="flex items-center gap-2">
                <span className="h-2.5 w-2.5 rounded-full bg-red-500" />
                <span className="text-sm text-muted-foreground">
                  Error ({errored})
                </span>
              </div>
            </div>
          </>
        )}
      </div>
    </div>
  );
}

function RecentActivityFeed() {
  const activeCluster = useActiveCluster();

  const { data: auditResponse, isLoading } = useQuery({
    queryKey: ["audit-recent", activeCluster],
    queryFn: () => fetchAuditEntries({ limit: 8 }),
  });

  const entries = auditResponse?.entries ?? [];

  function actionBadgeClass(action: string): string {
    switch (action.toLowerCase()) {
      case "create":
        return "bg-emerald-500/10 text-emerald-400";
      case "delete":
        return "bg-red-500/10 text-red-400";
      case "update":
        return "bg-blue-500/10 text-blue-400";
      default:
        return "bg-zinc-500/10 text-zinc-400";
    }
  }

  function formatTimestamp(ts: string): string {
    try {
      const date = new Date(ts);
      const now = new Date();
      const diffMs = now.getTime() - date.getTime();
      const diffMins = Math.floor(diffMs / 60000);

      if (diffMins < 1) return "just now";
      if (diffMins < 60) return `${diffMins}m ago`;
      const diffHours = Math.floor(diffMins / 60);
      if (diffHours < 24) return `${diffHours}h ago`;
      const diffDays = Math.floor(diffHours / 24);
      return `${diffDays}d ago`;
    } catch {
      return ts;
    }
  }

  return (
    <div>
      <div className="mb-3 flex items-center justify-between">
        <h2 className="text-lg font-semibold">Recent Activity</h2>
        <Link
          to="/audit"
          className="text-sm text-blue-400 hover:text-blue-300"
        >
          View all
        </Link>
      </div>
      <div className="rounded-lg border border-border bg-card">
        {isLoading ? (
          <div className="space-y-0">
            {Array.from({ length: 4 }).map((_, i) => (
              <div
                key={i}
                className="flex items-center gap-3 border-b border-border px-4 py-3 last:border-0"
              >
                <div className="h-4 w-14 animate-pulse rounded bg-muted" />
                <div className="h-4 w-32 animate-pulse rounded bg-muted" />
                <div className="ml-auto h-4 w-12 animate-pulse rounded bg-muted" />
              </div>
            ))}
          </div>
        ) : entries.length === 0 ? (
          <div className="px-4 py-8 text-center text-sm text-muted-foreground">
            No recent activity recorded.
          </div>
        ) : (
          <div>
            {entries.map((entry) => (
              <div
                key={entry.id}
                className="flex items-center gap-3 border-b border-border px-4 py-3 last:border-0"
              >
                <span
                  className={`inline-flex rounded px-2 py-0.5 text-xs font-medium ${actionBadgeClass(entry.action)}`}
                >
                  {entry.action}
                </span>
                <span className="text-sm text-foreground">
                  <span className="font-medium">{entry.resource}</span>
                  <span className="text-muted-foreground">/</span>
                  {entry.name}
                </span>
                {entry.namespace && (
                  <span className="hidden text-xs text-muted-foreground sm:inline">
                    in {entry.namespace}
                  </span>
                )}
                <span className="ml-auto shrink-0 text-xs text-muted-foreground">
                  {formatTimestamp(entry.timestamp)}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

function SingleClusterDashboard() {
  const { edition } = useEdition();
  const activeCluster = useActiveCluster();
  const [topologyCluster, setTopologyCluster] = useState<string>("");

  const { data: clusters } = useQuery({
    queryKey: ["clusters"],
    queryFn: fetchClusters,
  });

  // Resolve which cluster to use for topology: local override or active cluster.
  const resolvedTopologyCluster = topologyCluster || activeCluster || undefined;

  const { data: gateways, isLoading: gwLoading } = useQuery({
    queryKey: ["gateways", activeCluster],
    queryFn: () => fetchGateways(),
  });

  const { data: routes, isLoading: rtLoading } = useQuery({
    queryKey: ["httproutes", activeCluster],
    queryFn: () => fetchHTTPRoutes(),
  });

  const { data: config, isLoading: cfgLoading } = useQuery({
    queryKey: ["config", activeCluster],
    queryFn: fetchConfig,
  });

  const { data: topology } = useQuery({
    queryKey: ["topology", resolvedTopologyCluster],
    queryFn: () => fetchTopology(resolvedTopologyCluster),
    refetchInterval: 30000,
  });

  const { data: certificates, isLoading: certLoading } = useQuery({
    queryKey: ["certificates", activeCluster],
    queryFn: () => fetchCertificates(),
  });

  const { data: policies, isLoading: polLoading } = useQuery({
    queryKey: ["policies-ratelimit", activeCluster],
    queryFn: () => fetchPolicies("ratelimit"),
  });

  const editionLabel =
    edition === "enterprise"
      ? "Enterprise"
      : edition === "oss"
        ? "OSS"
        : "Unknown";

  return (
    <div>
      <h1 className="text-2xl font-bold">Dashboard</h1>
      <p className="mt-2 text-muted-foreground">
        Overview of your NGINX Gateway Fabric deployment.
      </p>

      {/* Summary Cards - 2 rows of 3 */}
      <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <SummaryCard
          label="Gateways"
          value={gateways?.length ?? 0}
          icon={Network}
          loading={gwLoading}
        />
        <SummaryCard
          label="HTTP Routes"
          value={routes?.length ?? 0}
          icon={Route}
          loading={rtLoading}
        />
        <SummaryCard
          label="Certificates"
          value={certificates?.length ?? 0}
          icon={Lock}
          loading={certLoading}
        />
        <SummaryCard
          label="Policies"
          value={policies?.length ?? 0}
          icon={FileText}
          loading={polLoading}
        />
        <SummaryCard
          label="Cluster"
          value={config?.connected ? "Connected" : "Disconnected"}
          icon={Cpu}
          loading={cfgLoading}
        />
        <SummaryCard
          label="Edition"
          value={editionLabel}
          icon={Shield}
          loading={cfgLoading}
        />
      </div>

      {/* Resource Health */}
      {gateways && (
        <div className="mt-6">
          <ResourceHealth gateways={gateways} />
        </div>
      )}

      {/* Quick Actions */}
      <div className="mt-6">
        <QuickActions />
      </div>

      {/* Recent Activity */}
      <div className="mt-6">
        <RecentActivityFeed />
      </div>

      {/* Topology Graph */}
      {topology && topology.nodes.length > 0 && (
        <div className="mt-6">
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-lg font-semibold">Topology</h2>
            {clusters && clusters.length > 1 && (
              <select
                value={topologyCluster}
                onChange={(e) => setTopologyCluster(e.target.value)}
                className="rounded-md border border-border bg-card px-3 py-1.5 text-sm text-foreground focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              >
                <option value="">Current cluster</option>
                {clusters.map((c) => (
                  <option key={c.name} value={c.name}>
                    {c.displayName || c.name}
                  </option>
                ))}
              </select>
            )}
          </div>
          <TopologyGraph nodes={topology.nodes} edges={topology.edges} />
        </div>
      )}
    </div>
  );
}

export default function Dashboard() {
  const activeCluster = useActiveCluster();

  if (activeCluster === ALL_CLUSTERS) {
    return <GlobalDashboard />;
  }

  return <SingleClusterDashboard />;
}
