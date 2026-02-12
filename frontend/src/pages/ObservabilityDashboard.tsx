import { useQuery } from "@tanstack/react-query";
import { fetchMetricsSummary, fetchMetricsByRoute, fetchMetricsByGateway } from "@/api/metrics";
import { useActiveCluster } from "@/hooks/useActiveCluster";

function formatNumber(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`;
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}K`;
  return n.toFixed(1);
}

function errorRateColor(rate: number): string {
  if (rate > 0.05) return "text-red-400";
  if (rate > 0.01) return "text-yellow-400";
  return "text-green-400";
}

export default function ObservabilityDashboard() {
  const activeCluster = useActiveCluster();

  const { data: summary, isLoading } = useQuery({
    queryKey: ["metrics-summary", activeCluster],
    queryFn: fetchMetricsSummary,
  });

  const { data: routeMetrics } = useQuery({
    queryKey: ["metrics-by-route", activeCluster],
    queryFn: fetchMetricsByRoute,
  });

  const { data: gatewayMetrics } = useQuery({
    queryKey: ["metrics-by-gateway", activeCluster],
    queryFn: fetchMetricsByGateway,
  });

  return (
    <div>
      <h1 className="text-2xl font-bold">Observability</h1>
      <p className="mt-1 text-muted-foreground">RED metrics from Prometheus (Rate, Errors, Duration).</p>

      {isLoading && <p className="mt-6 text-muted-foreground">Loading metrics...</p>}

      {summary && (
        <div className="mt-6 grid grid-cols-2 gap-4 md:grid-cols-4">
          <div className="rounded-lg border border-border p-4">
            <p className="text-sm text-muted-foreground">Requests/sec</p>
            <p className="mt-1 text-2xl font-bold">{formatNumber(summary.requestsPerSec)}</p>
          </div>
          <div className="rounded-lg border border-border p-4">
            <p className="text-sm text-muted-foreground">Error Rate</p>
            <p className={`mt-1 text-2xl font-bold ${errorRateColor(summary.errorRate)}`}>
              {(summary.errorRate * 100).toFixed(2)}%
            </p>
          </div>
          <div className="rounded-lg border border-border p-4">
            <p className="text-sm text-muted-foreground">P95 Latency</p>
            <p className="mt-1 text-2xl font-bold">{summary.p95LatencyMs.toFixed(1)}ms</p>
          </div>
          <div className="rounded-lg border border-border p-4">
            <p className="text-sm text-muted-foreground">Active Connections</p>
            <p className="mt-1 text-2xl font-bold">{formatNumber(summary.activeConnections)}</p>
          </div>
          <div className="rounded-lg border border-border p-4">
            <p className="text-sm text-muted-foreground">P50 Latency</p>
            <p className="mt-1 text-2xl font-bold">{summary.p50LatencyMs.toFixed(1)}ms</p>
          </div>
          <div className="rounded-lg border border-border p-4">
            <p className="text-sm text-muted-foreground">P99 Latency</p>
            <p className="mt-1 text-2xl font-bold">{summary.p99LatencyMs.toFixed(1)}ms</p>
          </div>
          <div className="rounded-lg border border-border p-4">
            <p className="text-sm text-muted-foreground">Avg Latency</p>
            <p className="mt-1 text-2xl font-bold">{summary.avgLatencyMs.toFixed(1)}ms</p>
          </div>
          <div className="rounded-lg border border-border p-4">
            <p className="text-sm text-muted-foreground">Total Requests</p>
            <p className="mt-1 text-2xl font-bold">{formatNumber(summary.totalRequests)}</p>
          </div>
        </div>
      )}

      {routeMetrics && routeMetrics.length > 0 && (
        <div className="mt-8">
          <h2 className="text-lg font-semibold">Metrics by Route</h2>
          <div className="mt-3 overflow-x-auto rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-muted/30">
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">Route</th>
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">Hostname</th>
                  <th className="px-4 py-3 text-right font-medium text-muted-foreground">Req/s</th>
                  <th className="px-4 py-3 text-right font-medium text-muted-foreground">Error Rate</th>
                  <th className="px-4 py-3 text-right font-medium text-muted-foreground">Avg Latency</th>
                  <th className="px-4 py-3 text-right font-medium text-muted-foreground">P95 Latency</th>
                </tr>
              </thead>
              <tbody>
                {routeMetrics.map((rm) => (
                  <tr key={`${rm.namespace}/${rm.name}`} className="border-b border-border last:border-0 hover:bg-muted/20">
                    <td className="px-4 py-3 font-medium">{rm.namespace}/{rm.name}</td>
                    <td className="px-4 py-3 text-muted-foreground">{rm.hostname || "-"}</td>
                    <td className="px-4 py-3 text-right">{rm.requestsPerSec.toFixed(1)}</td>
                    <td className={`px-4 py-3 text-right ${errorRateColor(rm.errorRate)}`}>
                      {(rm.errorRate * 100).toFixed(2)}%
                    </td>
                    <td className="px-4 py-3 text-right">{rm.avgLatencyMs.toFixed(1)}ms</td>
                    <td className="px-4 py-3 text-right">{rm.p95LatencyMs.toFixed(1)}ms</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {gatewayMetrics && gatewayMetrics.length > 0 && (
        <div className="mt-8">
          <h2 className="text-lg font-semibold">Metrics by Gateway</h2>
          <div className="mt-3 overflow-x-auto rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-muted/30">
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">Gateway</th>
                  <th className="px-4 py-3 text-right font-medium text-muted-foreground">Req/s</th>
                  <th className="px-4 py-3 text-right font-medium text-muted-foreground">Error Rate</th>
                  <th className="px-4 py-3 text-right font-medium text-muted-foreground">Avg Latency</th>
                  <th className="px-4 py-3 text-right font-medium text-muted-foreground">Connections</th>
                </tr>
              </thead>
              <tbody>
                {gatewayMetrics.map((gm) => (
                  <tr key={`${gm.namespace}/${gm.name}`} className="border-b border-border last:border-0 hover:bg-muted/20">
                    <td className="px-4 py-3 font-medium">{gm.namespace}/{gm.name}</td>
                    <td className="px-4 py-3 text-right">{gm.requestsPerSec.toFixed(1)}</td>
                    <td className={`px-4 py-3 text-right ${errorRateColor(gm.errorRate)}`}>
                      {(gm.errorRate * 100).toFixed(2)}%
                    </td>
                    <td className="px-4 py-3 text-right">{gm.avgLatencyMs.toFixed(1)}ms</td>
                    <td className="px-4 py-3 text-right">{formatNumber(gm.activeConnections)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}
