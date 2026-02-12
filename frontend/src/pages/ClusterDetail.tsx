import { useQuery } from "@tanstack/react-query";
import { useParams, useNavigate } from "react-router-dom";
import { fetchClusterDetail, getAgentInstallCommand } from "@/api/clusters";
import { ArrowLeft, Copy, Check } from "lucide-react";
import { useState } from "react";

export default function ClusterDetail() {
  const { name } = useParams<{ name: string }>();
  const navigate = useNavigate();
  const [copied, setCopied] = useState(false);

  const { data: cluster, isLoading } = useQuery({
    queryKey: ["clusters", name, "detail"],
    queryFn: () => fetchClusterDetail(name!),
    enabled: !!name,
  });

  const { data: installInfo } = useQuery({
    queryKey: ["clusters", name, "install"],
    queryFn: () => getAgentInstallCommand(name!),
    enabled: !!name,
  });

  const copyCommand = () => {
    if (installInfo?.helmCommand) {
      navigator.clipboard.writeText(installInfo.helmCommand);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <p className="text-muted-foreground">Loading cluster details...</p>
      </div>
    );
  }

  if (!cluster) {
    return (
      <div className="flex items-center justify-center py-12">
        <p className="text-muted-foreground">Cluster not found</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <button
          onClick={() => navigate("/clusters")}
          className="rounded-md p-1 text-muted-foreground hover:bg-accent"
        >
          <ArrowLeft className="h-5 w-5" />
        </button>
        <div>
          <h1 className="text-2xl font-bold text-foreground">
            {cluster.displayName || cluster.name}
          </h1>
          <p className="text-sm text-muted-foreground">
            {cluster.region}
            {cluster.environment ? ` · ${cluster.environment}` : ""}
          </p>
        </div>
        <span
          className={`ml-2 rounded-full px-2 py-0.5 text-xs font-medium ${
            cluster.connected
              ? "bg-green-500/20 text-green-500"
              : "bg-red-500/20 text-red-500"
          }`}
        >
          {cluster.connected ? "Ready" : "Unreachable"}
        </span>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {[
          { label: "Kubernetes", value: cluster.kubernetesVersion || "—" },
          { label: "NGF Version", value: cluster.ngfVersion || "—" },
          { label: "Edition", value: cluster.edition || "—" },
          {
            label: "Agent",
            value: cluster.agentInstalled ? "Installed" : "Not installed",
          },
        ].map((item) => (
          <div
            key={item.label}
            className="rounded-lg border border-border bg-card p-4"
          >
            <p className="text-xs text-muted-foreground">{item.label}</p>
            <p className="mt-1 text-lg font-semibold text-foreground">
              {item.value}
            </p>
          </div>
        ))}
      </div>

      {cluster.resourceCounts && (
        <div>
          <h2 className="mb-3 text-lg font-semibold text-foreground">
            Resources
          </h2>
          <div className="grid gap-3 sm:grid-cols-3 lg:grid-cols-7">
            {[
              { label: "Gateways", value: cluster.resourceCounts.gateways },
              { label: "HTTP Routes", value: cluster.resourceCounts.httpRoutes },
              {
                label: "Inference Pools",
                value: cluster.resourceCounts.inferencePools,
              },
              {
                label: "Inference Stacks",
                value: cluster.resourceCounts.inferenceStacks,
              },
              {
                label: "Gateway Bundles",
                value: cluster.resourceCounts.gatewayBundles,
              },
              { label: "Services", value: cluster.resourceCounts.services },
              { label: "Namespaces", value: cluster.resourceCounts.namespaces },
            ].map((item) => (
              <div
                key={item.label}
                className="rounded-lg border border-border bg-card p-3 text-center"
              >
                <p className="text-xl font-bold text-foreground">
                  {item.value}
                </p>
                <p className="text-[10px] text-muted-foreground">
                  {item.label}
                </p>
              </div>
            ))}
          </div>
        </div>
      )}

      {cluster.gpuCapacity && cluster.gpuCapacity.totalGPUs > 0 && (
        <div>
          <h2 className="mb-3 text-lg font-semibold text-foreground">
            GPU Capacity
          </h2>
          <div className="grid gap-3 sm:grid-cols-3">
            <div className="rounded-lg border border-border bg-card p-4">
              <p className="text-xs text-muted-foreground">Total GPUs</p>
              <p className="mt-1 text-2xl font-bold text-foreground">
                {cluster.gpuCapacity.totalGPUs}
              </p>
            </div>
            <div className="rounded-lg border border-border bg-card p-4">
              <p className="text-xs text-muted-foreground">Allocated</p>
              <p className="mt-1 text-2xl font-bold text-foreground">
                {cluster.gpuCapacity.allocatedGPUs}
              </p>
            </div>
            {cluster.gpuCapacity.gpuTypes && (
              <div className="rounded-lg border border-border bg-card p-4">
                <p className="text-xs text-muted-foreground">GPU Types</p>
                <div className="mt-1 space-y-1">
                  {Object.entries(cluster.gpuCapacity.gpuTypes).map(
                    ([type, count]) => (
                      <p key={type} className="text-sm text-foreground">
                        {type}: {count}
                      </p>
                    ),
                  )}
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {!cluster.agentInstalled && installInfo && (
        <div>
          <h2 className="mb-3 text-lg font-semibold text-foreground">
            Install Agent
          </h2>
          <div className="rounded-lg border border-border bg-card p-4">
            <p className="mb-2 text-sm text-muted-foreground">
              Run this command on the workload cluster to install the NGF Console
              agent:
            </p>
            <div className="relative">
              <pre className="overflow-x-auto rounded-md bg-muted p-3 text-xs text-foreground">
                {installInfo.helmCommand}
              </pre>
              <button
                onClick={copyCommand}
                className="absolute right-2 top-2 rounded p-1 text-muted-foreground hover:bg-accent"
                title="Copy command"
              >
                {copied ? (
                  <Check className="h-4 w-4 text-green-500" />
                ) : (
                  <Copy className="h-4 w-4" />
                )}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
