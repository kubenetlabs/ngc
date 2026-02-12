import { useQuery } from "@tanstack/react-query";
import { fetchCertificates, deleteCertificate } from "@/api/certificates";
import { useActiveCluster } from "@/hooks/useActiveCluster";

function daysLeftColor(days: number): string {
  if (days <= 7) return "text-red-400";
  if (days <= 30) return "text-yellow-400";
  return "text-green-400";
}

function daysLeftBg(days: number): string {
  if (days <= 7) return "bg-red-500/10";
  if (days <= 30) return "bg-yellow-500/10";
  return "bg-green-500/10";
}

export default function CertificateList() {
  const activeCluster = useActiveCluster();

  const { data: certs, isLoading, error, refetch } = useQuery({
    queryKey: ["certificates", activeCluster],
    queryFn: fetchCertificates,
  });

  const handleDelete = async (name: string) => {
    if (!confirm(`Delete certificate ${name}?`)) return;
    try {
      await deleteCertificate(name);
      refetch();
    } catch (err) {
      alert(`Failed to delete certificate: ${err instanceof Error ? err.message : String(err)}`);
    }
  };

  const expiringSoon = certs?.filter((c) => c.daysLeft <= 30) ?? [];

  return (
    <div>
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Certificates</h1>
          <p className="mt-1 text-muted-foreground">TLS certificate inventory and lifecycle management.</p>
        </div>
      </div>

      {isLoading && <p className="mt-6 text-muted-foreground">Loading certificates...</p>}
      {error && <p className="mt-6 text-red-400">Failed to load certificates: {String(error)}</p>}

      {expiringSoon.length > 0 && (
        <div className="mt-4 rounded-lg border border-yellow-500/30 bg-yellow-500/5 p-4">
          <p className="font-medium text-yellow-400">
            {expiringSoon.length} certificate{expiringSoon.length > 1 ? "s" : ""} expiring within 30 days
          </p>
          <ul className="mt-2 space-y-1 text-sm text-yellow-300">
            {expiringSoon.map((c) => (
              <li key={c.name}>
                {c.name} — {c.daysLeft} days left ({c.domains.join(", ")})
              </li>
            ))}
          </ul>
        </div>
      )}

      {certs && certs.length === 0 && (
        <p className="mt-6 text-muted-foreground">No TLS certificates found.</p>
      )}

      {certs && certs.length > 0 && (
        <div className="mt-4 overflow-x-auto rounded-lg border border-border">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-muted/30">
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Name</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Namespace</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Domains</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Issuer</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Expires</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Days Left</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Actions</th>
              </tr>
            </thead>
            <tbody>
              {certs.map((c) => (
                <tr key={`${c.namespace}/${c.name}`} className="border-b border-border last:border-0 hover:bg-muted/20">
                  <td className="px-4 py-3 font-medium">{c.name}</td>
                  <td className="px-4 py-3 text-muted-foreground">{c.namespace}</td>
                  <td className="px-4 py-3 text-muted-foreground">
                    {c.domains.length > 2
                      ? `${c.domains.slice(0, 2).join(", ")} +${c.domains.length - 2}`
                      : c.domains.join(", ")}
                  </td>
                  <td className="px-4 py-3 text-muted-foreground">{c.issuer || "unknown"}</td>
                  <td className="px-4 py-3 text-muted-foreground">{c.notAfter ? new Date(c.notAfter).toLocaleDateString() : "-"}</td>
                  <td className="px-4 py-3">
                    <span className={`inline-block rounded px-2 py-0.5 text-xs font-medium ${daysLeftColor(c.daysLeft)} ${daysLeftBg(c.daysLeft)}`}>
                      {c.daysLeft}d
                    </span>
                  </td>
                  <td className="px-4 py-3">
                    <button
                      onClick={() => handleDelete(c.name)}
                      className="text-xs text-red-400 hover:underline"
                    >
                      Delete
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
