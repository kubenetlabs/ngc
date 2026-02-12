import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { fetchPolicies, deletePolicy } from "@/api/policies";
import { useActiveCluster } from "@/hooks/useActiveCluster";
import type { PolicyType } from "@/types/policy";

const POLICY_TYPES: { value: PolicyType; label: string }[] = [
  { value: "ratelimit", label: "Rate Limit" },
  { value: "clientsettings", label: "Client Settings" },
  { value: "backendtls", label: "Backend TLS" },
  { value: "observability", label: "Observability" },
];

function timeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime();
  const days = Math.floor(diff / 86_400_000);
  if (days > 0) return `${days}d ago`;
  const hours = Math.floor(diff / 3_600_000);
  if (hours > 0) return `${hours}h ago`;
  const mins = Math.floor(diff / 60_000);
  return `${mins}m ago`;
}

export default function PolicyList() {
  const activeCluster = useActiveCluster();
  const [policyType, setPolicyType] = useState<PolicyType>("ratelimit");

  const { data: policies, isLoading, error, refetch } = useQuery({
    queryKey: ["policies", activeCluster, policyType],
    queryFn: () => fetchPolicies(policyType),
  });

  const handleDelete = async (name: string, namespace?: string) => {
    if (!confirm(`Delete policy ${name}?`)) return;
    try {
      await deletePolicy(policyType, name, namespace);
      refetch();
    } catch (err) {
      alert(`Failed to delete policy: ${err instanceof Error ? err.message : String(err)}`);
    }
  };

  return (
    <div>
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Policies</h1>
          <p className="mt-1 text-muted-foreground">Manage rate limiting, TLS, and observability policies.</p>
        </div>
        <Link
          to="/policies/create"
          className="rounded-md bg-blue-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-blue-700"
        >
          Create Policy
        </Link>
      </div>

      <div className="mt-4 flex gap-2">
        {POLICY_TYPES.map((pt) => (
          <button
            key={pt.value}
            onClick={() => setPolicyType(pt.value)}
            className={`rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
              policyType === pt.value
                ? "bg-blue-600 text-white"
                : "border border-border text-muted-foreground hover:text-foreground"
            }`}
          >
            {pt.label}
          </button>
        ))}
      </div>

      {isLoading && <p className="mt-6 text-muted-foreground">Loading policies...</p>}
      {error && <p className="mt-6 text-red-400">Failed to load policies: {String(error)}</p>}

      {policies && policies.length === 0 && (
        <p className="mt-6 text-muted-foreground">No {policyType} policies found.</p>
      )}

      {policies && policies.length > 0 && (
        <div className="mt-4 overflow-x-auto rounded-lg border border-border">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-muted/30">
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Name</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Namespace</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Type</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Age</th>
                <th className="px-4 py-3 text-left font-medium text-muted-foreground">Actions</th>
              </tr>
            </thead>
            <tbody>
              {policies.map((p) => (
                <tr key={`${p.namespace}/${p.name}`} className="border-b border-border last:border-0 hover:bg-muted/20">
                  <td className="px-4 py-3 font-medium">{p.name}</td>
                  <td className="px-4 py-3 text-muted-foreground">{p.namespace}</td>
                  <td className="px-4 py-3 text-muted-foreground">{p.policyType}</td>
                  <td className="px-4 py-3 text-muted-foreground">{timeAgo(p.createdAt)}</td>
                  <td className="px-4 py-3">
                    <button
                      onClick={() => handleDelete(p.name, p.namespace)}
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
