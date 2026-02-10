import { useParams, Link, useNavigate } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { fetchHTTPRoute, deleteHTTPRoute } from "@/api/routes";
import { StatusBadge } from "@/components/common/StatusBadge";
import { useActiveCluster } from "@/hooks/useActiveCluster";

export default function RouteDetail() {
  const { ns, name } = useParams<{ ns: string; name: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const activeCluster = useActiveCluster();

  const { data: route, isLoading, error } = useQuery({
    queryKey: ["httproute", activeCluster, ns, name],
    queryFn: () => fetchHTTPRoute(ns!, name!),
    enabled: !!ns && !!name,
  });

  const deleteMutation = useMutation({
    mutationFn: () => deleteHTTPRoute(ns!, name!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["httproutes"] });
      navigate("/routes");
    },
  });

  const handleDelete = () => {
    if (window.confirm(`Delete HTTPRoute "${name}" in namespace "${ns}"?`)) {
      deleteMutation.mutate();
    }
  };

  if (isLoading) return <p className="text-muted-foreground">Loading route...</p>;
  if (error) return <p className="text-red-400">Failed to load route: {String(error)}</p>;
  if (!route) return <p className="text-muted-foreground">Route not found.</p>;

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <Link to="/routes" className="text-sm text-blue-400 hover:underline">
          &larr; Back to Routes
        </Link>
        <div className="flex gap-2">
          <Link
            to={`/routes/${ns}/${name}/edit`}
            className="rounded-md border border-border px-3 py-1.5 text-sm font-medium text-foreground hover:bg-muted/30"
          >
            Edit
          </Link>
          <button
            onClick={handleDelete}
            disabled={deleteMutation.isPending}
            className="rounded-md bg-red-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50"
          >
            {deleteMutation.isPending ? "Deleting..." : "Delete"}
          </button>
        </div>
      </div>

      <h1 className="text-2xl font-bold">{route.name}</h1>
      <p className="mt-1 text-muted-foreground">{route.namespace}</p>

      {/* Parent Gateways */}
      <section className="mt-6">
        <h2 className="text-lg font-semibold">Parent Gateways</h2>
        <div className="mt-2 flex flex-wrap gap-2">
          {route.parentRefs.map((ref, i) => (
            <Link
              key={i}
              to={`/gateways/${ref.namespace ?? route.namespace}/${ref.name}`}
              className="rounded bg-muted px-2 py-0.5 font-mono text-sm text-blue-400 hover:underline"
            >
              {ref.namespace ?? route.namespace}/{ref.name}
              {ref.sectionName ? ` (${ref.sectionName})` : ""}
            </Link>
          ))}
        </div>
      </section>

      {/* Hostnames */}
      {route.hostnames && route.hostnames.length > 0 && (
        <section className="mt-6">
          <h2 className="text-lg font-semibold">Hostnames</h2>
          <div className="mt-2 flex flex-wrap gap-2">
            {route.hostnames.map((h) => (
              <span key={h} className="rounded bg-muted px-2 py-0.5 font-mono text-sm text-muted-foreground">
                {h}
              </span>
            ))}
          </div>
        </section>
      )}

      {/* Rules */}
      <section className="mt-6">
        <h2 className="text-lg font-semibold">Rules</h2>
        <div className="mt-2 space-y-4">
          {route.rules.map((rule, ruleIdx) => (
            <div key={ruleIdx} className="rounded-lg border border-border p-4">
              <span className="text-sm font-medium text-muted-foreground">Rule {ruleIdx + 1}</span>

              {/* Matches */}
              {rule.matches && rule.matches.length > 0 && (
                <div className="mt-3">
                  <span className="text-xs font-medium text-muted-foreground">Matches</span>
                  <div className="mt-1 space-y-1">
                    {rule.matches.map((m, mIdx) => (
                      <div key={mIdx} className="flex items-center gap-2 text-sm">
                        {m.method && (
                          <span className="rounded bg-blue-500/20 px-1.5 py-0.5 text-xs font-medium text-blue-400">
                            {m.method}
                          </span>
                        )}
                        {m.path && (
                          <span className="font-mono text-foreground">
                            {m.path.type}: {m.path.value}
                          </span>
                        )}
                        {m.headers?.map((h, hIdx) => (
                          <span key={hIdx} className="text-muted-foreground">
                            Header {h.name}={h.value} ({h.type})
                          </span>
                        ))}
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Backend Refs */}
              {rule.backendRefs && rule.backendRefs.length > 0 && (
                <div className="mt-3">
                  <span className="text-xs font-medium text-muted-foreground">Backends</span>
                  <div className="mt-1 overflow-x-auto rounded border border-border">
                    <table className="w-full text-sm">
                      <thead>
                        <tr className="border-b border-border bg-muted/30">
                          <th className="px-3 py-2 text-left font-medium text-muted-foreground">Service</th>
                          <th className="px-3 py-2 text-left font-medium text-muted-foreground">Port</th>
                          <th className="px-3 py-2 text-left font-medium text-muted-foreground">Weight</th>
                        </tr>
                      </thead>
                      <tbody>
                        {rule.backendRefs.map((br, brIdx) => (
                          <tr key={brIdx} className="border-b border-border last:border-0">
                            <td className="px-3 py-2 font-mono text-foreground">
                              {br.namespace ? `${br.namespace}/` : ""}{br.name}
                            </td>
                            <td className="px-3 py-2 text-muted-foreground">{br.port ?? "-"}</td>
                            <td className="px-3 py-2 text-muted-foreground">{br.weight ?? 1}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}
            </div>
          ))}
        </div>
      </section>

      {/* Status */}
      {route.status?.parents && route.status.parents.length > 0 && (
        <section className="mt-6">
          <h2 className="text-lg font-semibold">Status</h2>
          <div className="mt-2 space-y-3">
            {route.status.parents.map((p, pIdx) => (
              <div key={pIdx} className="rounded-lg border border-border p-3">
                <div className="text-sm text-muted-foreground">
                  Parent: <span className="font-mono text-foreground">{p.parentRef.name}</span>
                  {" / Controller: "}
                  <span className="font-mono text-foreground">{p.controllerName}</span>
                </div>
                <div className="mt-2 flex flex-wrap gap-2">
                  {p.conditions.map((c) => (
                    <StatusBadge key={c.type} condition={c} />
                  ))}
                </div>
              </div>
            ))}
          </div>
        </section>
      )}
    </div>
  );
}
