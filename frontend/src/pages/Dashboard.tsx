import { useQuery } from "@tanstack/react-query";
import { Network, Route, Cpu, Shield } from "lucide-react";
import { fetchGateways } from "@/api/gateways";
import { fetchHTTPRoutes } from "@/api/routes";
import { fetchConfig } from "@/api/config";
import { useEdition } from "@/hooks/useEdition";
import { useActiveCluster } from "@/hooks/useActiveCluster";

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
        {loading ? <span className="animate-pulse text-muted-foreground">--</span> : value}
      </div>
    </div>
  );
}

export default function Dashboard() {
  const { edition } = useEdition();
  const activeCluster = useActiveCluster();

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

  const editionLabel = edition === "enterprise" ? "Enterprise" : edition === "oss" ? "OSS" : "Unknown";

  return (
    <div>
      <h1 className="text-2xl font-bold">Dashboard</h1>
      <p className="mt-2 text-muted-foreground">Overview of your NGINX Gateway Fabric deployment.</p>

      <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
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
    </div>
  );
}
