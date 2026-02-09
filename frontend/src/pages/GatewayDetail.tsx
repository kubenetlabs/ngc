import { useParams, Link, useNavigate } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { fetchGateway, deleteGateway } from "@/api/gateways";
import { fetchHTTPRoutes } from "@/api/routes";
import { StatusBadge } from "@/components/common/StatusBadge";
import { useActiveCluster } from "@/hooks/useActiveCluster";

export default function GatewayDetail() {
  const { ns, name } = useParams<{ ns: string; name: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const activeCluster = useActiveCluster();

  const { data: gw, isLoading, error } = useQuery({
    queryKey: ["gateway", activeCluster, ns, name],
    queryFn: () => fetchGateway(ns!, name!),
    enabled: !!ns && !!name,
  });

  const { data: allRoutes } = useQuery({
    queryKey: ["httproutes", activeCluster],
    queryFn: () => fetchHTTPRoutes(),
  });

  // Filter routes attached to this gateway
  const attachedRoutes = allRoutes?.filter((route) =>
    route.parentRefs.some(
      (ref) => ref.name === name && (ref.namespace === ns || (!ref.namespace && route.namespace === ns)),
    ),
  );

  const deleteMutation = useMutation({
    mutationFn: () => deleteGateway(ns!, name!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["gateways"] });
      navigate("/gateways");
    },
  });

  const handleDelete = () => {
    if (window.confirm(`Delete gateway "${name}" in namespace "${ns}"?`)) {
      deleteMutation.mutate();
    }
  };

  if (isLoading) return <p className="text-muted-foreground">Loading gateway...</p>;
  if (error) return <p className="text-red-400">Failed to load gateway: {String(error)}</p>;
  if (!gw) return <p className="text-muted-foreground">Gateway not found.</p>;

  return (
    <div>
      <div className="mb-6 flex items-center justify-between">
        <Link to="/gateways" className="text-sm text-blue-400 hover:underline">
          &larr; Back to Gateways
        </Link>
        <div className="flex gap-2">
          <Link
            to={`/gateways/${ns}/${name}/edit`}
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

      <h1 className="text-2xl font-bold">{gw.name}</h1>
      <p className="mt-1 text-muted-foreground">
        {gw.namespace} &middot; Class: {gw.gatewayClassName}
      </p>

      {/* Status Conditions */}
      {gw.status?.conditions && gw.status.conditions.length > 0 && (
        <section className="mt-6">
          <h2 className="text-lg font-semibold">Status</h2>
          <div className="mt-2 flex flex-wrap gap-2">
            {gw.status.conditions.map((c) => (
              <StatusBadge key={c.type} condition={c} />
            ))}
          </div>
        </section>
      )}

      {/* Addresses */}
      {gw.status?.addresses && gw.status.addresses.length > 0 && (
        <section className="mt-6">
          <h2 className="text-lg font-semibold">Addresses</h2>
          <div className="mt-2 space-y-1">
            {gw.status.addresses.map((a, i) => (
              <div key={i} className="text-sm text-muted-foreground">
                {a.type}: <span className="font-mono text-foreground">{a.value}</span>
              </div>
            ))}
          </div>
        </section>
      )}

      {/* Listeners */}
      <section className="mt-6">
        <h2 className="text-lg font-semibold">Listeners</h2>
        <div className="mt-2 overflow-x-auto rounded-lg border border-border">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-muted/30">
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Name</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Port</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Protocol</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Hostname</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Attached Routes</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Status</th>
              </tr>
            </thead>
            <tbody>
              {gw.listeners.map((l) => {
                const ls = gw.status?.listeners.find((s) => s.name === l.name);
                return (
                  <tr key={l.name} className="border-b border-border last:border-0">
                    <td className="px-4 py-3 font-medium">{l.name}</td>
                    <td className="px-4 py-3 text-muted-foreground">{l.port}</td>
                    <td className="px-4 py-3 text-muted-foreground">{l.protocol}</td>
                    <td className="px-4 py-3 font-mono text-muted-foreground">{l.hostname ?? "*"}</td>
                    <td className="px-4 py-3 text-muted-foreground">{ls?.attachedRoutes ?? 0}</td>
                    <td className="px-4 py-3">
                      <div className="flex flex-wrap gap-1">
                        {ls?.conditions.map((c) => <StatusBadge key={c.type} condition={c} />)}
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </section>

      {/* Attached Routes */}
      {attachedRoutes && attachedRoutes.length > 0 && (
        <section className="mt-6">
          <h2 className="text-lg font-semibold">Attached Routes</h2>
          <div className="mt-2 overflow-x-auto rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-muted/30">
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">Name</th>
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">Namespace</th>
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">Hostnames</th>
                </tr>
              </thead>
              <tbody>
                {attachedRoutes.map((route) => (
                  <tr key={`${route.namespace}/${route.name}`} className="border-b border-border last:border-0">
                    <td className="px-4 py-3 font-medium text-foreground">{route.name}</td>
                    <td className="px-4 py-3 text-muted-foreground">{route.namespace}</td>
                    <td className="px-4 py-3 font-mono text-muted-foreground">
                      {route.hostnames?.join(", ") || "*"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      )}

      {/* Labels & Annotations */}
      {gw.labels && Object.keys(gw.labels).length > 0 && (
        <section className="mt-6">
          <h2 className="text-lg font-semibold">Labels</h2>
          <div className="mt-2 flex flex-wrap gap-2">
            {Object.entries(gw.labels).map(([k, v]) => (
              <span key={k} className="rounded bg-muted px-2 py-0.5 font-mono text-xs text-muted-foreground">
                {k}={v}
              </span>
            ))}
          </div>
        </section>
      )}
    </div>
  );
}
