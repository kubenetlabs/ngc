import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useActiveCluster } from "@/hooks/useActiveCluster";
import {
  fetchXCStatus,
  fetchXCPublishes,
  fetchXCMetrics,
  publishToXC,
  deleteXCPublish,
  type XCPublish,
  type XCRegion,
  type XCPublishRequest,
} from "@/api/xc";

const inputClass =
  "mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring";

function statusBadge(status: string): string {
  switch (status) {
    case "Published":
      return "bg-emerald-500/10 text-emerald-400 border-emerald-500/30";
    case "Pending":
      return "bg-yellow-500/10 text-yellow-400 border-yellow-500/30";
    case "Error":
      return "bg-red-500/10 text-red-400 border-red-500/30";
    default:
      return "bg-zinc-500/10 text-zinc-400 border-zinc-500/30";
  }
}

export default function XCOverview() {
  const activeCluster = useActiveCluster();
  const queryClient = useQueryClient();
  const [showForm, setShowForm] = useState(false);
  const [formData, setFormData] = useState<XCPublishRequest>({
    namespace: "default",
    name: "",
    httpRouteRef: "",
  });

  const {
    data: statusData,
    isLoading: statusLoading,
    error: statusError,
  } = useQuery({
    queryKey: ["xc-status", activeCluster],
    queryFn: fetchXCStatus,
  });

  const {
    data: publishes,
    isLoading: publishesLoading,
    error: publishesError,
  } = useQuery({
    queryKey: ["xc-publishes", activeCluster],
    queryFn: fetchXCPublishes,
  });

  const {
    data: metrics,
    isLoading: metricsLoading,
    error: metricsError,
  } = useQuery({
    queryKey: ["xc-metrics", activeCluster],
    queryFn: fetchXCMetrics,
  });

  const publishMutation = useMutation({
    mutationFn: publishToXC,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["xc-publishes"] });
      queryClient.invalidateQueries({ queryKey: ["xc-status"] });
      setShowForm(false);
      setFormData({
        namespace: "default",
        name: "",
        httpRouteRef: "",
      });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteXCPublish,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["xc-publishes"] });
      queryClient.invalidateQueries({ queryKey: ["xc-status"] });
    },
  });

  const handlePublish = (e: React.FormEvent) => {
    e.preventDefault();
    publishMutation.mutate(formData);
  };

  const handleDelete = (id: string, name: string) => {
    if (!confirm(`Delete publish "${name}"?`)) return;
    deleteMutation.mutate(id);
  };

  const isLoading = statusLoading || publishesLoading || metricsLoading;
  const error = statusError || publishesError || metricsError;

  return (
    <div>
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">F5 Distributed Cloud</h1>
          <p className="mt-1 text-muted-foreground">
            Manage XC publishing and edge security.
          </p>
        </div>
        <button
          onClick={() => setShowForm(!showForm)}
          className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
        >
          {showForm ? "Cancel" : "Publish"}
        </button>
      </div>

      {isLoading && (
        <p className="mt-6 text-muted-foreground">Loading XC data...</p>
      )}
      {error && (
        <p className="mt-6 text-red-400">
          Failed to load XC data: {String(error)}
        </p>
      )}

      {/* Publish Form */}
      {showForm && (
        <form
          onSubmit={handlePublish}
          className="mt-6 rounded-lg border border-border p-4"
        >
          <h2 className="text-lg font-semibold">Publish to XC</h2>
          <div className="mt-4 grid gap-4 sm:grid-cols-2">
            <div>
              <label className="block text-sm font-medium">Namespace</label>
              <input
                value={formData.namespace}
                onChange={(e) =>
                  setFormData({ ...formData, namespace: e.target.value })
                }
                className={inputClass}
                placeholder="default"
              />
            </div>
            <div>
              <label className="block text-sm font-medium">Name</label>
              <input
                value={formData.name}
                onChange={(e) =>
                  setFormData({ ...formData, name: e.target.value })
                }
                className={inputClass}
                placeholder="my-publish"
              />
            </div>
            <div className="sm:col-span-2">
              <label className="block text-sm font-medium">HTTPRoute Ref</label>
              <input
                value={formData.httpRouteRef}
                onChange={(e) =>
                  setFormData({ ...formData, httpRouteRef: e.target.value })
                }
                className={inputClass}
                placeholder="my-http-route"
              />
            </div>
          </div>

          {publishMutation.isError && (
            <p className="mt-3 text-sm text-red-400">
              {String(publishMutation.error)}
            </p>
          )}

          <div className="mt-4 flex justify-end">
            <button
              type="submit"
              disabled={publishMutation.isPending}
              className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
            >
              {publishMutation.isPending ? "Publishing..." : "Publish"}
            </button>
          </div>
        </form>
      )}

      {/* Regional Metrics */}
      {metrics && (
        <div className="mt-6">
          <div className="grid gap-4 sm:grid-cols-3">
            <div className="rounded-lg border border-border p-4">
              <p className="text-sm text-muted-foreground">Total Requests</p>
              <p className="mt-1 text-2xl font-bold text-foreground">
                {metrics.totalRequests.toLocaleString()}
              </p>
            </div>
            <div className="rounded-lg border border-border p-4">
              <p className="text-sm text-muted-foreground">Avg Latency</p>
              <p className="mt-1 text-2xl font-bold text-foreground">
                {metrics.avgLatencyMs.toFixed(1)}ms
              </p>
            </div>
            <div className="rounded-lg border border-border p-4">
              <p className="text-sm text-muted-foreground">Regions</p>
              <p className="mt-1 text-2xl font-bold text-foreground">
                {metrics.regions.length}
              </p>
            </div>
          </div>

          {metrics.regions.length > 0 && (
            <div className="mt-4 grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {metrics.regions.map((region: XCRegion) => (
                <div
                  key={region.name}
                  className="rounded-lg border border-border p-4"
                >
                  <h3 className="text-sm font-medium text-foreground">
                    {region.name}
                  </h3>
                  <div className="mt-3 space-y-1 text-sm text-muted-foreground">
                    <div className="flex justify-between">
                      <span>Requests</span>
                      <span>{region.requests.toLocaleString()}</span>
                    </div>
                    <div className="flex justify-between">
                      <span>Latency</span>
                      <span>{region.latencyMs.toFixed(1)}ms</span>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Status Summary */}
      {statusData && (
        <div className="mt-6 rounded-lg border border-border p-4">
          <div className="flex items-center gap-4">
            <span className={`text-sm font-medium ${statusData.connected ? "text-emerald-400" : "text-red-400"}`}>
              {statusData.connected ? "Connected" : "Disconnected"}
            </span>
            <span className="text-sm text-muted-foreground">
              {statusData.publishCount} publish(es)
            </span>
          </div>
        </div>
      )}

      {/* Publishes Table */}
      {publishes && (
        <div className="mt-6">
          <h2 className="text-lg font-semibold">Publishes</h2>
          {publishes.length === 0 && (
            <p className="mt-3 text-muted-foreground">
              No publishes configured. Click "Publish" to create one.
            </p>
          )}
          {publishes.length > 0 && (
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
                      HTTPRoute Ref
                    </th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                      Phase
                    </th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                      Created
                    </th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {publishes.map((pub: XCPublish) => (
                    <tr
                      key={`${pub.namespace}/${pub.name}`}
                      className="border-b border-border last:border-0 hover:bg-muted/20"
                    >
                      <td className="px-4 py-3 font-medium">{pub.name}</td>
                      <td className="px-4 py-3 text-muted-foreground">
                        {pub.namespace}
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">
                        {pub.httpRouteRef}
                      </td>
                      <td className="px-4 py-3">
                        <span
                          className={`inline-flex rounded-md border px-2 py-0.5 text-xs font-medium ${statusBadge(pub.phase)}`}
                        >
                          {pub.phase}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-muted-foreground">
                        {pub.createdAt}
                      </td>
                      <td className="px-4 py-3">
                        <button
                          onClick={() => handleDelete(`${pub.namespace}/${pub.name}`, pub.name)}
                          disabled={deleteMutation.isPending}
                          className="text-xs text-red-400 hover:underline disabled:opacity-50"
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
      )}
    </div>
  );
}
