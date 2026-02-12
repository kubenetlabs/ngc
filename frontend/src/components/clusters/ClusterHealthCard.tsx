import type { ManagedCluster } from "@/types/cluster";
import { Server, Wifi, WifiOff } from "lucide-react";

interface Props {
  cluster: ManagedCluster;
  onClick?: () => void;
}

export function ClusterHealthCard({ cluster, onClick }: Props) {
  return (
    <div
      onClick={onClick}
      className={`rounded-lg border border-border bg-card p-4 transition-colors ${onClick ? "cursor-pointer hover:bg-accent/50" : ""}`}
    >
      <div className="flex items-start justify-between">
        <div className="flex items-center gap-3">
          <Server className="h-5 w-5 text-muted-foreground" />
          <div>
            <h3 className="text-sm font-medium text-foreground">
              {cluster.displayName || cluster.name}
            </h3>
            <p className="text-xs text-muted-foreground">
              {cluster.region}
              {cluster.environment ? ` · ${cluster.environment}` : ""}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-1.5">
          {cluster.connected ? (
            <Wifi className="h-4 w-4 text-green-500" />
          ) : (
            <WifiOff className="h-4 w-4 text-red-500" />
          )}
          <span
            className={`text-xs font-medium ${cluster.connected ? "text-green-500" : "text-red-500"}`}
          >
            {cluster.connected ? "Ready" : "Unreachable"}
          </span>
        </div>
      </div>

      <div className="mt-3 grid grid-cols-3 gap-2 text-xs">
        <div>
          <span className="text-muted-foreground">K8s</span>
          <p className="font-medium text-foreground">
            {cluster.kubernetesVersion || "—"}
          </p>
        </div>
        <div>
          <span className="text-muted-foreground">NGF</span>
          <p className="font-medium text-foreground">
            {cluster.ngfVersion || "—"}
          </p>
        </div>
        <div>
          <span className="text-muted-foreground">Agent</span>
          <p className="font-medium text-foreground">
            {cluster.isLocal ? "Hub" : cluster.agentInstalled ? "Installed" : "—"}
          </p>
        </div>
      </div>

      {cluster.resourceCounts && (
        <div className="mt-2 flex gap-3 text-[10px] text-muted-foreground">
          <span>{cluster.resourceCounts.gateways} GW</span>
          <span>{cluster.resourceCounts.httpRoutes} Routes</span>
          <span>{cluster.resourceCounts.services} Svc</span>
          {cluster.gpuCapacity && cluster.gpuCapacity.totalGPUs > 0 && (
            <span>{cluster.gpuCapacity.totalGPUs} GPUs</span>
          )}
        </div>
      )}
    </div>
  );
}
