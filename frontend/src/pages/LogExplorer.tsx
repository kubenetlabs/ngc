import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { queryLogs, fetchTopNLogs } from "@/api/logs";
import { useActiveCluster } from "@/hooks/useActiveCluster";
import type { LogQueryRequest } from "@/types/logs";

function statusColor(code: number): string {
  if (code >= 500) return "text-red-400";
  if (code >= 400) return "text-yellow-400";
  if (code >= 300) return "text-blue-400";
  return "text-green-400";
}

export default function LogExplorer() {
  const activeCluster = useActiveCluster();
  const [namespace, setNamespace] = useState("");
  const [hostname, setHostname] = useState("");
  const [search, setSearch] = useState("");
  const [limit, setLimit] = useState(50);

  const query: LogQueryRequest = {
    namespace: namespace || undefined,
    hostname: hostname || undefined,
    search: search || undefined,
    limit,
  };

  const { data: logs, isLoading, error } = useQuery({
    queryKey: ["logs", activeCluster, query],
    queryFn: () => queryLogs(query),
  });

  const { data: topN } = useQuery({
    queryKey: ["logs-topn", activeCluster],
    queryFn: () => fetchTopNLogs("path", 10),
  });

  return (
    <div>
      <h1 className="text-2xl font-bold">Log Explorer</h1>
      <p className="mt-1 text-muted-foreground">Search and analyze access logs.</p>

      <div className="mt-4 flex flex-wrap gap-3">
        <input
          type="text"
          placeholder="Namespace..."
          value={namespace}
          onChange={(e) => setNamespace(e.target.value)}
          className="rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring"
        />
        <input
          type="text"
          placeholder="Hostname..."
          value={hostname}
          onChange={(e) => setHostname(e.target.value)}
          className="rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring"
        />
        <input
          type="text"
          placeholder="Search path..."
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="flex-1 rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring"
        />
        <select
          value={limit}
          onChange={(e) => setLimit(Number(e.target.value))}
          className="rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground"
        >
          <option value={25}>25 rows</option>
          <option value={50}>50 rows</option>
          <option value={100}>100 rows</option>
        </select>
      </div>

      <div className="mt-6 grid gap-6 lg:grid-cols-3">
        <div className="lg:col-span-2">
          {isLoading && <p className="text-muted-foreground">Loading logs...</p>}
          {error && <p className="text-red-400">Failed to load logs: {String(error)}</p>}

          {logs && (
            <div className="overflow-x-auto rounded-lg border border-border">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-muted/30">
                    <th className="px-3 py-2 text-left font-medium text-muted-foreground">Time</th>
                    <th className="px-3 py-2 text-left font-medium text-muted-foreground">Method</th>
                    <th className="px-3 py-2 text-left font-medium text-muted-foreground">Path</th>
                    <th className="px-3 py-2 text-left font-medium text-muted-foreground">Status</th>
                    <th className="px-3 py-2 text-right font-medium text-muted-foreground">Latency</th>
                    <th className="px-3 py-2 text-left font-medium text-muted-foreground">Upstream</th>
                  </tr>
                </thead>
                <tbody>
                  {logs.map((entry, i) => (
                    <tr key={i} className="border-b border-border last:border-0 hover:bg-muted/20 font-mono text-xs">
                      <td className="px-3 py-2 text-muted-foreground whitespace-nowrap">
                        {new Date(entry.timestamp).toLocaleTimeString()}
                      </td>
                      <td className="px-3 py-2">{entry.method}</td>
                      <td className="px-3 py-2 max-w-[200px] truncate" title={entry.path}>{entry.path}</td>
                      <td className={`px-3 py-2 font-medium ${statusColor(entry.statusCode)}`}>
                        {entry.statusCode}
                      </td>
                      <td className="px-3 py-2 text-right">{entry.latency.toFixed(1)}ms</td>
                      <td className="px-3 py-2 text-muted-foreground">{entry.upstreamService}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>

        <div>
          <h2 className="text-lg font-semibold">Top Paths</h2>
          {topN && topN.length > 0 ? (
            <div className="mt-3 space-y-2">
              {topN.map((entry, i) => (
                <div key={i} className="rounded-lg border border-border p-3">
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-mono truncate max-w-[180px]" title={entry.key}>{entry.key}</span>
                    <span className="text-xs text-muted-foreground">{entry.count.toLocaleString()}</span>
                  </div>
                  <div className="mt-2 h-1.5 rounded-full bg-muted/30">
                    <div
                      className="h-1.5 rounded-full bg-blue-500"
                      style={{ width: `${entry.percentage}%` }}
                    />
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <p className="mt-3 text-sm text-muted-foreground">No data available.</p>
          )}
        </div>
      </div>
    </div>
  );
}
