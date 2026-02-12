import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { fetchAuditEntries, fetchAuditDiff } from "@/api/audit";
import { useActiveCluster } from "@/hooks/useActiveCluster";
import type { AuditEntry } from "@/types/audit";

function timeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime();
  const days = Math.floor(diff / 86_400_000);
  if (days > 0) return `${days}d ago`;
  const hours = Math.floor(diff / 3_600_000);
  if (hours > 0) return `${hours}h ago`;
  const mins = Math.floor(diff / 60_000);
  return `${mins}m ago`;
}

function actionColor(action: string): string {
  switch (action) {
    case "create": return "bg-green-500/10 text-green-400";
    case "update": return "bg-blue-500/10 text-blue-400";
    case "delete": return "bg-red-500/10 text-red-400";
    default: return "bg-gray-500/10 text-gray-400";
  }
}

export default function AuditLog() {
  const activeCluster = useActiveCluster();
  const [resourceFilter, setResourceFilter] = useState("");
  const [actionFilter, setActionFilter] = useState("");
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [page, setPage] = useState(0);
  const limit = 25;

  const { data, isLoading, error } = useQuery({
    queryKey: ["audit", activeCluster, resourceFilter, actionFilter, page],
    queryFn: () => fetchAuditEntries({
      resource: resourceFilter || undefined,
      action: actionFilter || undefined,
      limit,
      offset: page * limit,
    }),
  });

  const { data: diff } = useQuery({
    queryKey: ["audit-diff", selectedId],
    queryFn: () => fetchAuditDiff(selectedId!),
    enabled: !!selectedId,
  });

  return (
    <div>
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Audit Log</h1>
          <p className="mt-1 text-muted-foreground">Configuration change history and diff viewer.</p>
        </div>
      </div>

      <div className="mt-4 flex gap-3">
        <select
          value={resourceFilter}
          onChange={(e) => { setResourceFilter(e.target.value); setPage(0); }}
          className="rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground"
        >
          <option value="">All resources</option>
          <option value="Gateway">Gateway</option>
          <option value="HTTPRoute">HTTPRoute</option>
          <option value="InferenceStack">InferenceStack</option>
          <option value="GatewayBundle">GatewayBundle</option>
        </select>
        <select
          value={actionFilter}
          onChange={(e) => { setActionFilter(e.target.value); setPage(0); }}
          className="rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground"
        >
          <option value="">All actions</option>
          <option value="create">Create</option>
          <option value="update">Update</option>
          <option value="delete">Delete</option>
        </select>
      </div>

      {isLoading && <p className="mt-6 text-muted-foreground">Loading audit log...</p>}
      {error && <p className="mt-6 text-red-400">Failed to load audit log: {String(error)}</p>}

      {data && (
        <>
          <p className="mt-4 text-sm text-muted-foreground">{data.total} total entries</p>
          <div className="mt-2 overflow-x-auto rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-muted/30">
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">Time</th>
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">Action</th>
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">Resource</th>
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">Name</th>
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">Namespace</th>
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">User</th>
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">Diff</th>
                </tr>
              </thead>
              <tbody>
                {data.entries.map((entry: AuditEntry) => (
                  <tr key={entry.id} className="border-b border-border last:border-0 hover:bg-muted/20">
                    <td className="px-4 py-3 text-muted-foreground">{timeAgo(entry.timestamp)}</td>
                    <td className="px-4 py-3">
                      <span className={`inline-block rounded px-2 py-0.5 text-xs font-medium ${actionColor(entry.action)}`}>
                        {entry.action}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-muted-foreground">{entry.resource}</td>
                    <td className="px-4 py-3 font-medium">{entry.name}</td>
                    <td className="px-4 py-3 text-muted-foreground">{entry.namespace || "-"}</td>
                    <td className="px-4 py-3 text-muted-foreground">{entry.user || "system"}</td>
                    <td className="px-4 py-3">
                      <button
                        onClick={() => setSelectedId(selectedId === entry.id ? null : entry.id)}
                        className="text-xs text-blue-400 hover:underline"
                      >
                        {selectedId === entry.id ? "Hide" : "View"}
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className="mt-4 flex items-center justify-between">
            <button
              onClick={() => setPage(Math.max(0, page - 1))}
              disabled={page === 0}
              className="rounded-md border border-border px-3 py-1.5 text-sm disabled:opacity-50"
            >
              Previous
            </button>
            <span className="text-sm text-muted-foreground">Page {page + 1}</span>
            <button
              onClick={() => setPage(page + 1)}
              disabled={(page + 1) * limit >= data.total}
              className="rounded-md border border-border px-3 py-1.5 text-sm disabled:opacity-50"
            >
              Next
            </button>
          </div>
        </>
      )}

      {selectedId && diff && (
        <div className="mt-4 rounded-lg border border-border p-4">
          <h3 className="font-medium">Diff: {diff.action} {diff.resource}/{diff.name}</h3>
          <div className="mt-3 grid grid-cols-2 gap-4">
            <div>
              <p className="mb-1 text-sm font-medium text-muted-foreground">Before</p>
              <pre className="overflow-auto rounded bg-muted/30 p-3 text-xs">
                {JSON.stringify(diff.beforeJson, null, 2)}
              </pre>
            </div>
            <div>
              <p className="mb-1 text-sm font-medium text-muted-foreground">After</p>
              <pre className="overflow-auto rounded bg-muted/30 p-3 text-xs">
                {JSON.stringify(diff.afterJson, null, 2)}
              </pre>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
