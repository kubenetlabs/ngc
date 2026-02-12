import { useQuery } from "@tanstack/react-query";
import { useActiveCluster } from "@/hooks/useActiveCluster";
import {
  fetchCoexistenceOverview,
  fetchMigrationReadiness,
  type ResourceCount,
  type SharedResource,
  type Conflict,
  type ReadinessCategory,
} from "@/api/coexistence";

function severityBadge(severity: string): string {
  switch (severity) {
    case "high":
      return "bg-red-500/10 text-red-400 border-red-500/30";
    case "medium":
      return "bg-yellow-500/10 text-yellow-400 border-yellow-500/30";
    case "low":
      return "bg-blue-500/10 text-blue-400 border-blue-500/30";
    default:
      return "bg-zinc-500/10 text-zinc-400 border-zinc-500/30";
  }
}

function statusColor(status: string): string {
  switch (status) {
    case "pass":
      return "text-emerald-400";
    case "warn":
      return "text-yellow-400";
    case "fail":
      return "text-red-400";
    default:
      return "text-muted-foreground";
  }
}

function ReadinessBar({ score }: { score: number }) {
  const pct = Math.min(100, (score / 25) * 100);
  const color =
    pct >= 80 ? "bg-emerald-500" : pct >= 50 ? "bg-yellow-500" : "bg-red-500";
  return (
    <div className="flex items-center gap-3">
      <div className="h-2 flex-1 rounded-full bg-muted/30">
        <div
          className={`h-2 rounded-full ${color}`}
          style={{ width: `${pct}%` }}
        />
      </div>
      <span className="text-sm font-medium text-foreground">
        {score}/25
      </span>
    </div>
  );
}

export default function CoexistenceDashboard() {
  const activeCluster = useActiveCluster();

  const {
    data: overview,
    isLoading: overviewLoading,
    error: overviewError,
  } = useQuery({
    queryKey: ["coexistence-overview", activeCluster],
    queryFn: fetchCoexistenceOverview,
  });

  const {
    data: readiness,
    isLoading: readinessLoading,
    error: readinessError,
  } = useQuery({
    queryKey: ["migration-readiness", activeCluster],
    queryFn: fetchMigrationReadiness,
  });

  const isLoading = overviewLoading || readinessLoading;
  const error = overviewError || readinessError;

  return (
    <div>
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Coexistence</h1>
          <p className="mt-1 text-muted-foreground">
            KIC and NGF side-by-side deployment view.
          </p>
        </div>
      </div>

      {isLoading && (
        <p className="mt-6 text-muted-foreground">Loading coexistence data...</p>
      )}
      {error && (
        <p className="mt-6 text-red-400">
          Failed to load coexistence data: {String(error)}
        </p>
      )}

      {overview && (
        <>
          {/* KIC vs NGF side-by-side */}
          <div className="mt-6 grid gap-6 lg:grid-cols-2">
            {/* KIC column */}
            <div className="rounded-lg border border-border p-4">
              <div className="flex items-center justify-between">
                <h2 className="text-lg font-semibold">
                  NGINX Ingress Controller
                </h2>
                <span className="rounded bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">
                  v{overview.kic.version}
                </span>
              </div>
              <div className="mt-4 space-y-2">
                {overview.kic.resources.map((r: ResourceCount) => (
                  <div
                    key={r.kind}
                    className="flex items-center justify-between rounded-md px-3 py-2 hover:bg-muted/20"
                  >
                    <span className="text-sm text-foreground">{r.kind}</span>
                    <span className="rounded bg-muted/30 px-2 py-0.5 text-xs font-medium text-muted-foreground">
                      {r.count}
                    </span>
                  </div>
                ))}
                {overview.kic.resources.length === 0 && (
                  <p className="text-sm text-muted-foreground">
                    No KIC resources detected.
                  </p>
                )}
              </div>
            </div>

            {/* NGF column */}
            <div className="rounded-lg border border-border p-4">
              <div className="flex items-center justify-between">
                <h2 className="text-lg font-semibold">
                  NGINX Gateway Fabric
                </h2>
                <span className="rounded bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">
                  v{overview.ngf.version}
                </span>
              </div>
              <div className="mt-4 space-y-2">
                {overview.ngf.resources.map((r: ResourceCount) => (
                  <div
                    key={r.kind}
                    className="flex items-center justify-between rounded-md px-3 py-2 hover:bg-muted/20"
                  >
                    <span className="text-sm text-foreground">{r.kind}</span>
                    <span className="rounded bg-muted/30 px-2 py-0.5 text-xs font-medium text-muted-foreground">
                      {r.count}
                    </span>
                  </div>
                ))}
                {overview.ngf.resources.length === 0 && (
                  <p className="text-sm text-muted-foreground">
                    No NGF resources detected.
                  </p>
                )}
              </div>
            </div>
          </div>

          {/* Shared Resources */}
          {overview.sharedResources.length > 0 && (
            <div className="mt-6">
              <h2 className="text-lg font-semibold">Shared Resources</h2>
              <p className="mt-1 text-sm text-muted-foreground">
                Resources referenced by both KIC and NGF deployments.
              </p>
              <div className="mt-3 overflow-x-auto rounded-lg border border-border">
                <table className="w-full text-sm">
                  <thead>
                    <tr className="border-b border-border bg-muted/30">
                      <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                        Name
                      </th>
                      <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                        Namespace
                      </th>
                      <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                        Kind
                      </th>
                      <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                        Used By
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {overview.sharedResources.map((r: SharedResource) => (
                      <tr
                        key={`${r.namespace}/${r.name}`}
                        className="border-b border-border last:border-0 hover:bg-muted/20"
                      >
                        <td className="px-4 py-3 font-medium">{r.name}</td>
                        <td className="px-4 py-3 text-muted-foreground">
                          {r.namespace}
                        </td>
                        <td className="px-4 py-3 text-muted-foreground">
                          {r.kind}
                        </td>
                        <td className="px-4 py-3 text-muted-foreground">
                          {r.usedBy.join(", ")}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}

          {/* Conflicts */}
          {overview.conflicts.length > 0 && (
            <div className="mt-6">
              <h2 className="text-lg font-semibold">Conflicts</h2>
              <p className="mt-1 text-sm text-muted-foreground">
                Detected conflicts between KIC and NGF configurations.
              </p>
              <div className="mt-3 space-y-3">
                {overview.conflicts.map((c: Conflict, i: number) => (
                  <div
                    key={i}
                    className="rounded-lg border border-border p-4"
                  >
                    <div className="flex items-center gap-3">
                      <span
                        className={`inline-flex rounded-md border px-2 py-0.5 text-xs font-medium ${severityBadge(c.severity)}`}
                      >
                        {c.severity}
                      </span>
                      <span className="text-sm font-medium text-foreground">
                        {c.type}
                      </span>
                    </div>
                    <p className="mt-2 text-sm text-muted-foreground">
                      {c.description}
                    </p>
                    {c.resource && (
                      <div className="mt-2">
                        <span className="rounded bg-muted/30 px-2 py-0.5 text-xs text-muted-foreground">
                          {c.resource}
                        </span>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </div>
          )}

          {overview.conflicts.length === 0 && (
            <div className="mt-6 rounded-lg border border-emerald-500/30 bg-emerald-500/10 p-4">
              <p className="text-sm text-emerald-400">
                No conflicts detected between KIC and NGF deployments.
              </p>
            </div>
          )}
        </>
      )}

      {/* Migration Readiness */}
      {readiness && (
        <div className="mt-6">
          <h2 className="text-lg font-semibold">Migration Readiness</h2>
          <p className="mt-1 text-sm text-muted-foreground">
            Assessment of readiness to migrate from KIC to NGF.
          </p>

          {/* Overall Score */}
          <div className="mt-4 rounded-lg border border-border p-6">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium text-muted-foreground">
                Overall Readiness Score
              </span>
              <span className="text-2xl font-bold text-foreground">
                {readiness.score}%
              </span>
            </div>
            <div className="mt-3 h-3 rounded-full bg-muted/30">
              <div
                className={`h-3 rounded-full ${
                  readiness.score >= 80
                    ? "bg-emerald-500"
                    : readiness.score >= 50
                      ? "bg-yellow-500"
                      : "bg-red-500"
                }`}
                style={{ width: `${readiness.score}%` }}
              />
            </div>
          </div>

          {/* Categories */}
          <div className="mt-4 space-y-3">
            {readiness.categories.map((cat: ReadinessCategory) => (
              <div key={cat.name} className="rounded-lg border border-border p-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <span className={`text-sm font-medium ${statusColor(cat.status)}`}>
                      {cat.status === "pass"
                        ? "\u2713"
                        : cat.status === "warn"
                          ? "!"
                          : "\u2717"}
                    </span>
                    <span className="text-sm font-medium text-foreground">
                      {cat.name}
                    </span>
                  </div>
                </div>
                <div className="mt-2">
                  <ReadinessBar score={cat.score} />
                </div>
                <p className="mt-2 text-xs text-muted-foreground">
                  {cat.details}
                </p>
              </div>
            ))}
          </div>

          {/* Blockers */}
          {readiness.blockers.length > 0 && (
            <div className="mt-4">
              <h3 className="text-sm font-medium text-red-400">Blockers</h3>
              <ul className="mt-2 space-y-1">
                {readiness.blockers.map((b: string, i: number) => (
                  <li
                    key={i}
                    className="text-sm text-muted-foreground"
                  >
                    - {b}
                  </li>
                ))}
              </ul>
            </div>
          )}

          {/* Recommendations */}
          {readiness.recommendations.length > 0 && (
            <div className="mt-4">
              <h3 className="text-sm font-medium text-blue-400">
                Recommendations
              </h3>
              <ul className="mt-2 space-y-1">
                {readiness.recommendations.map((r: string, i: number) => (
                  <li
                    key={i}
                    className="text-sm text-muted-foreground"
                  >
                    - {r}
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
