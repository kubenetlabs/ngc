import { useState, useEffect, useCallback } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useActiveCluster } from "@/hooks/useActiveCluster";
import { fetchHTTPRoutes } from "@/api/routes";
import type { HTTPRoute } from "@/types/route";
import {
  fetchXCStatus,
  fetchXCPublishes,
  fetchXCMetrics,
  fetchXCCredentials,
  fetchWAFPolicies,
  publishToXC,
  previewXCPublish,
  deleteXCPublish,
  type XCPublish,
  type XCRegion,
  type XCPublishRequest,
  type XCPreviewResponse,
  type WAFPolicy,
} from "@/api/xc";

const inputClass =
  "mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring";

function statusBadge(status: string): string {
  switch (status) {
    case "Published":
      return "bg-emerald-500/10 text-emerald-400 border-emerald-500/30";
    case "Pending":
    case "Publishing":
      return "bg-yellow-500/10 text-yellow-400 border-yellow-500/30";
    case "Degraded":
      return "bg-orange-500/10 text-orange-400 border-orange-500/30";
    case "Error":
      return "bg-red-500/10 text-red-400 border-red-500/30";
    default:
      return "bg-zinc-500/10 text-zinc-400 border-zinc-500/30";
  }
}

type WizardStep = "select" | "configure" | "review";

interface StatusMessage {
  type: "success" | "warning" | "error";
  text: string;
}

export default function XCOverview() {
  const activeCluster = useActiveCluster();
  const queryClient = useQueryClient();

  // Wizard state
  const [showWizard, setShowWizard] = useState(false);
  const [wizardStep, setWizardStep] = useState<WizardStep>("select");
  const [selectedRoute, setSelectedRoute] = useState<HTTPRoute | null>(null);
  const [publishName, setPublishName] = useState("");
  const [publicHostname, setPublicHostname] = useState("");
  const [originAddress, setOriginAddress] = useState("");
  const [wafEnabled, setWafEnabled] = useState(false);
  const [wafPolicyName, setWafPolicyName] = useState("");
  const [webSocketEnabled, setWebSocketEnabled] = useState(false);
  const [preview, setPreview] = useState<XCPreviewResponse | null>(null);

  // Status message state
  const [statusMessage, setStatusMessage] = useState<StatusMessage | null>(
    null
  );

  // Auto-dismiss success messages after 8 seconds
  useEffect(() => {
    if (statusMessage && statusMessage.type === "success") {
      const timer = setTimeout(() => setStatusMessage(null), 8000);
      return () => clearTimeout(timer);
    }
  }, [statusMessage]);

  const dismissMessage = useCallback(() => setStatusMessage(null), []);

  // Queries
  const { data: statusData } = useQuery({
    queryKey: ["xc-status", activeCluster],
    queryFn: fetchXCStatus,
  });

  const { data: publishes, isLoading: publishesLoading } = useQuery({
    queryKey: ["xc-publishes", activeCluster],
    queryFn: fetchXCPublishes,
  });

  const { data: metrics } = useQuery({
    queryKey: ["xc-metrics", activeCluster],
    queryFn: fetchXCMetrics,
  });

  const { data: creds } = useQuery({
    queryKey: ["xc-credentials"],
    queryFn: fetchXCCredentials,
  });

  const { data: routes } = useQuery({
    queryKey: ["httproutes", activeCluster],
    queryFn: () => fetchHTTPRoutes(),
    enabled: showWizard,
  });

  const { data: wafPolicies } = useQuery({
    queryKey: ["xc-waf-policies", activeCluster],
    queryFn: fetchWAFPolicies,
    enabled: showWizard && wafEnabled,
  });

  // Mutations
  const previewMutation = useMutation({
    mutationFn: previewXCPublish,
    onSuccess: (data) => {
      setPreview(data);
      setWizardStep("review");
    },
    onError: (error) => {
      setStatusMessage({
        type: "error",
        text: `Preview failed: ${String(error)}`,
      });
    },
  });

  const publishMutation = useMutation({
    mutationFn: publishToXC,
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ["xc-publishes"] });
      queryClient.invalidateQueries({ queryKey: ["xc-status"] });
      resetWizard();
      if (data.errors && data.errors.length > 0) {
        setStatusMessage({
          type: "warning",
          text: `Published with warnings: ${data.errors.join("; ")}`,
        });
      } else {
        setStatusMessage({
          type: "success",
          text: "Successfully published to F5 Distributed Cloud.",
        });
      }
    },
    onError: (error) => {
      setStatusMessage({
        type: "error",
        text: `Publish failed: ${String(error)}`,
      });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteXCPublish,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["xc-publishes"] });
      queryClient.invalidateQueries({ queryKey: ["xc-status"] });
      setStatusMessage({
        type: "success",
        text: "Publish deleted successfully. XC resources have been cleaned up.",
      });
    },
    onError: (error) => {
      setStatusMessage({
        type: "error",
        text: `Delete failed: ${String(error)}`,
      });
    },
  });

  const resetWizard = () => {
    setShowWizard(false);
    setWizardStep("select");
    setSelectedRoute(null);
    setPublishName("");
    setPublicHostname("");
    setOriginAddress("");
    setWafEnabled(false);
    setWafPolicyName("");
    setWebSocketEnabled(false);
    setPreview(null);
  };

  const handleSelectRoute = (route: HTTPRoute) => {
    setSelectedRoute(route);
    setPublishName(`xc-${route.name}`);
    if (route.hostnames && route.hostnames.length > 0) {
      setPublicHostname(route.hostnames[0]);
    }
    setWizardStep("configure");
  };

  const handlePreview = () => {
    if (!selectedRoute) return;
    setStatusMessage(null);
    previewMutation.mutate({
      namespace: selectedRoute.namespace,
      httpRouteRef: selectedRoute.name,
      publicHostname: publicHostname || undefined,
      originAddress: originAddress || undefined,
      wafEnabled,
      wafPolicyName: wafEnabled ? wafPolicyName || undefined : undefined,
      webSocketEnabled,
    });
  };

  const handlePublish = () => {
    if (!selectedRoute) return;
    setStatusMessage(null);
    const req: XCPublishRequest = {
      name: publishName,
      namespace: selectedRoute.namespace,
      httpRouteRef: selectedRoute.name,
      publicHostname: publicHostname || undefined,
      originAddress: originAddress || undefined,
      wafEnabled,
      wafPolicyName: wafEnabled ? wafPolicyName || undefined : undefined,
      webSocketEnabled,
    };
    publishMutation.mutate(req);
  };

  const handleDelete = (id: string, name: string) => {
    if (!confirm(`Delete publish "${name}"? This will also remove XC resources.`))
      return;
    setStatusMessage(null);
    deleteMutation.mutate(id);
  };

  const isConfigured = creds?.configured;

  return (
    <div>
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">F5 Distributed Cloud</h1>
          <p className="mt-1 text-muted-foreground">
            Publish HTTPRoutes to XC as HTTP Load Balancers with optional WAF
            protection.
          </p>
        </div>
        <button
          onClick={() => (showWizard ? resetWizard() : setShowWizard(true))}
          disabled={!isConfigured}
          className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {showWizard ? "Cancel" : "Publish Route"}
        </button>
      </div>

      {/* Status Message Banner */}
      {statusMessage && (
        <div
          className={`mt-4 rounded-lg border p-4 ${
            statusMessage.type === "success"
              ? "border-emerald-500/30 bg-emerald-500/10"
              : statusMessage.type === "warning"
                ? "border-yellow-500/30 bg-yellow-500/10"
                : "border-red-500/30 bg-red-500/10"
          }`}
        >
          <div className="flex items-start justify-between gap-3">
            <p
              className={`text-sm ${
                statusMessage.type === "success"
                  ? "text-emerald-400"
                  : statusMessage.type === "warning"
                    ? "text-yellow-400"
                    : "text-red-400"
              }`}
            >
              {statusMessage.type === "success" && "✓ "}
              {statusMessage.type === "warning" && "⚠ "}
              {statusMessage.type === "error" && "✗ "}
              {statusMessage.text}
            </p>
            <button
              onClick={dismissMessage}
              className="shrink-0 text-muted-foreground hover:text-foreground"
            >
              ✕
            </button>
          </div>
        </div>
      )}

      {/* Not configured warning */}
      {!isConfigured && (
        <div className="mt-6 rounded-lg border border-yellow-500/30 bg-yellow-500/5 p-4">
          <p className="text-sm text-yellow-400">
            XC credentials not configured. Go to Settings &gt; Distributed Cloud
            to connect your XC tenant.
          </p>
        </div>
      )}

      {/* Publish Wizard */}
      {showWizard && (
        <div className="mt-6 rounded-lg border border-border">
          {/* Wizard Steps Header */}
          <div className="flex border-b border-border">
            {(["select", "configure", "review"] as WizardStep[]).map(
              (step, i) => {
                const labels = [
                  "1. Select Route",
                  "2. Configure",
                  "3. Review & Publish",
                ];
                const isActive = wizardStep === step;
                const isPast =
                  (step === "select" && wizardStep !== "select") ||
                  (step === "configure" && wizardStep === "review");
                return (
                  <div
                    key={step}
                    className={`flex-1 px-4 py-3 text-center text-sm font-medium ${
                      isActive
                        ? "border-b-2 border-blue-500 text-foreground"
                        : isPast
                          ? "text-muted-foreground"
                          : "text-muted-foreground/50"
                    }`}
                  >
                    {labels[i]}
                  </div>
                );
              }
            )}
          </div>

          <div className="p-4">
            {/* Step 1: Select Route */}
            {wizardStep === "select" && (
              <div>
                <h3 className="text-sm font-semibold">
                  Select an HTTPRoute to publish
                </h3>
                {!routes || routes.length === 0 ? (
                  <p className="mt-3 text-sm text-muted-foreground">
                    No HTTPRoutes found. Create one first.
                  </p>
                ) : (
                  <div className="mt-3 space-y-2">
                    {routes.map((route: HTTPRoute) => (
                      <button
                        key={`${route.namespace}/${route.name}`}
                        onClick={() => handleSelectRoute(route)}
                        className="w-full rounded-lg border border-border p-3 text-left hover:border-blue-500/50 hover:bg-blue-500/5"
                      >
                        <div className="flex items-center justify-between">
                          <span className="font-medium">{route.name}</span>
                          <span className="text-xs text-muted-foreground">
                            {route.namespace}
                          </span>
                        </div>
                        {route.hostnames && route.hostnames.length > 0 && (
                          <p className="mt-1 text-xs text-muted-foreground">
                            Hostnames: {route.hostnames.join(", ")}
                          </p>
                        )}
                        {route.rules && (
                          <p className="mt-0.5 text-xs text-muted-foreground">
                            {route.rules.length} rule(s) |{" "}
                            {route.parentRefs?.length || 0} parent ref(s)
                          </p>
                        )}
                      </button>
                    ))}
                  </div>
                )}
              </div>
            )}

            {/* Step 2: Configure */}
            {wizardStep === "configure" && selectedRoute && (
              <div>
                <h3 className="text-sm font-semibold">
                  Configure XC Load Balancer
                </h3>
                <p className="mt-1 text-xs text-muted-foreground">
                  Settings derived from{" "}
                  <span className="font-mono">{selectedRoute.name}</span> in{" "}
                  {selectedRoute.namespace}
                </p>

                <div className="mt-4 grid gap-4 sm:grid-cols-2">
                  <div>
                    <label className="block text-sm font-medium">
                      Publish Name
                    </label>
                    <input
                      value={publishName}
                      onChange={(e) => setPublishName(e.target.value)}
                      className={inputClass}
                      placeholder="xc-my-route"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium">
                      Public Hostname
                    </label>
                    <input
                      value={publicHostname}
                      onChange={(e) => setPublicHostname(e.target.value)}
                      className={inputClass}
                      placeholder="app.example.com"
                    />
                    <p className="mt-1 text-xs text-muted-foreground">
                      Edge-facing hostname for the XC Load Balancer
                    </p>
                  </div>
                  <div>
                    <label className="block text-sm font-medium">
                      Origin Address Override
                    </label>
                    <input
                      value={originAddress}
                      onChange={(e) => setOriginAddress(e.target.value)}
                      className={inputClass}
                      placeholder="52.73.106.230"
                    />
                    <p className="mt-1 text-xs text-muted-foreground">
                      IP or hostname XC will use to reach the Gateway. Leave
                      blank to auto-detect from the Gateway&apos;s external
                      address.
                    </p>
                  </div>
                </div>

                {/* WAF Toggle */}
                <div className="mt-6">
                  <div className="flex items-center gap-3">
                    <button
                      type="button"
                      onClick={() => setWafEnabled(!wafEnabled)}
                      className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                        wafEnabled ? "bg-blue-600" : "bg-muted"
                      }`}
                    >
                      <span
                        className={`inline-block h-4 w-4 rounded-full bg-white transition-transform ${
                          wafEnabled
                            ? "translate-x-[22px]"
                            : "translate-x-[3px]"
                        }`}
                      />
                    </button>
                    <span className="text-sm font-medium">
                      Enable WAF Protection
                    </span>
                  </div>

                  {wafEnabled && (
                    <div className="mt-3">
                      <label className="block text-sm font-medium">
                        WAF Policy
                      </label>
                      <select
                        value={wafPolicyName}
                        onChange={(e) => setWafPolicyName(e.target.value)}
                        className={inputClass}
                      >
                        <option value="">Default WAF Policy</option>
                        {wafPolicies?.map((p: WAFPolicy) => (
                          <option key={p.name} value={p.name}>
                            {p.name}
                            {p.mode ? ` (${p.mode})` : ""}
                          </option>
                        ))}
                      </select>
                    </div>
                  )}
                </div>

                {/* WebSocket Toggle */}
                <div className="mt-6">
                  <div className="flex items-center gap-3">
                    <input
                      type="checkbox"
                      id="websocket-toggle"
                      checked={webSocketEnabled}
                      onChange={(e) => setWebSocketEnabled(e.target.checked)}
                      className="h-4 w-4 rounded border-border bg-background text-blue-600 focus:ring-blue-500"
                    />
                    <label
                      htmlFor="websocket-toggle"
                      className="text-sm font-medium"
                    >
                      Enable WebSocket Support
                    </label>
                  </div>
                  <p className="mt-1 ml-7 text-xs text-muted-foreground">
                    Enable WebSocket protocol upgrade on routes. Required for
                    applications that use real-time connections (e.g. chat,
                    streaming).
                  </p>
                </div>

                {/* Navigation */}
                <div className="mt-6 flex items-center justify-between">
                  <button
                    onClick={() => setWizardStep("select")}
                    className="rounded-md border border-border px-4 py-2 text-sm font-medium text-foreground hover:bg-muted"
                  >
                    Back
                  </button>
                  <button
                    onClick={handlePreview}
                    disabled={!publishName || previewMutation.isPending}
                    className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
                  >
                    {previewMutation.isPending
                      ? "Generating Preview..."
                      : "Preview Configuration"}
                  </button>
                </div>

                {previewMutation.isError && (
                  <div className="mt-3 rounded-lg border border-red-500/30 bg-red-500/10 p-3">
                    <p className="text-sm text-red-400">
                      ✗ Preview failed: {String(previewMutation.error)}
                    </p>
                  </div>
                )}
              </div>
            )}

            {/* Step 3: Review & Publish */}
            {wizardStep === "review" && preview && (
              <div>
                <h3 className="text-sm font-semibold">
                  Review XC Configuration
                </h3>
                <p className="mt-1 text-xs text-muted-foreground">
                  The following resources will be created in your XC tenant.
                </p>

                {/* HTTP Load Balancer Preview */}
                <div className="mt-4 rounded-lg border border-border p-4">
                  <h4 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                    HTTP Load Balancer
                  </h4>
                  <pre className="mt-2 max-h-64 overflow-auto rounded bg-muted/30 p-3 font-mono text-xs text-foreground">
                    {JSON.stringify(preview.loadBalancer, null, 2)}
                  </pre>
                </div>

                {/* Origin Pool Preview */}
                <div className="mt-4 rounded-lg border border-border p-4">
                  <h4 className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                    Origin Pool
                  </h4>
                  <pre className="mt-2 max-h-48 overflow-auto rounded bg-muted/30 p-3 font-mono text-xs text-foreground">
                    {JSON.stringify(preview.originPool, null, 2)}
                  </pre>
                </div>

                {/* WAF Preview */}
                {preview.wafPolicy && (
                  <div className="mt-4 rounded-lg border border-blue-500/20 bg-blue-500/5 p-4">
                    <h4 className="text-xs font-semibold uppercase tracking-wider text-blue-400">
                      WAF Policy
                    </h4>
                    <p className="mt-1 text-sm text-foreground">
                      {preview.wafPolicy}
                    </p>
                  </div>
                )}

                {/* Navigation */}
                <div className="mt-6 flex items-center justify-between">
                  <button
                    onClick={() => setWizardStep("configure")}
                    className="rounded-md border border-border px-4 py-2 text-sm font-medium text-foreground hover:bg-muted"
                  >
                    Back
                  </button>
                  <button
                    onClick={handlePublish}
                    disabled={publishMutation.isPending}
                    className="rounded-md bg-emerald-600 px-6 py-2 text-sm font-medium text-white hover:bg-emerald-700 disabled:opacity-50"
                  >
                    {publishMutation.isPending
                      ? "Publishing..."
                      : "Publish to XC"}
                  </button>
                </div>

                {publishMutation.isError && (
                  <div className="mt-3 rounded-lg border border-red-500/30 bg-red-500/10 p-3">
                    <p className="text-sm text-red-400">
                      ✗ Publish failed: {String(publishMutation.error)}
                    </p>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
      )}

      {/* Metrics Cards */}
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
            <div className="flex items-center gap-2">
              <span
                className={`inline-flex h-2.5 w-2.5 rounded-full ${statusData.xcConnected ? "bg-emerald-400" : "bg-red-400"}`}
              />
              <span className="text-sm font-medium">
                {statusData.xcConnected
                  ? `Connected to ${statusData.tenant}`
                  : "XC Not Connected"}
              </span>
            </div>
            <span className="text-sm text-muted-foreground">
              {statusData.publishCount} publish(es)
            </span>
          </div>
        </div>
      )}

      {/* Publishes Table */}
      <div className="mt-6">
        <h2 className="text-lg font-semibold">Publishes</h2>
        {publishesLoading && (
          <div className="mt-3 flex items-center gap-2 text-muted-foreground">
            <div className="h-4 w-4 animate-spin rounded-full border-2 border-muted-foreground border-t-transparent" />
            <p>Loading publishes...</p>
          </div>
        )}
        {publishes && publishes.length === 0 && (
          <p className="mt-3 text-muted-foreground">
            No publishes configured. Click &quot;Publish Route&quot; to create
            one.
          </p>
        )}
        {publishes && publishes.length > 0 && (
          <div className="mt-3 overflow-x-auto rounded-lg border border-border">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-border bg-muted/30">
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                    Name
                  </th>
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                    HTTPRoute
                  </th>
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                    Phase
                  </th>
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                    XC LB
                  </th>
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                    WAF
                  </th>
                  <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                    Synced
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
                    <td className="px-4 py-3">
                      <div className="font-medium">{pub.name}</div>
                      <div className="text-xs text-muted-foreground">
                        {pub.namespace}
                      </div>
                    </td>
                    <td className="px-4 py-3 text-muted-foreground">
                      {pub.httpRouteRef}
                    </td>
                    <td className="px-4 py-3">
                      <span
                        className={`inline-flex rounded-md border px-2 py-0.5 text-xs font-medium ${statusBadge(pub.phase)}`}
                      >
                        {pub.phase || "Pending"}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-xs text-muted-foreground">
                      {pub.xcLoadBalancerName || "-"}
                      {pub.xcVirtualIP && (
                        <div className="text-xs">{pub.xcVirtualIP}</div>
                      )}
                    </td>
                    <td className="px-4 py-3 text-xs text-muted-foreground">
                      {pub.wafPolicyAttached || "-"}
                    </td>
                    <td className="px-4 py-3 text-xs text-muted-foreground">
                      {pub.lastSyncedAt
                        ? new Date(pub.lastSyncedAt).toLocaleString()
                        : "-"}
                    </td>
                    <td className="px-4 py-3">
                      <button
                        onClick={() =>
                          handleDelete(
                            `${pub.namespace}/${pub.name}`,
                            pub.name
                          )
                        }
                        disabled={deleteMutation.isPending}
                        className="text-xs text-red-400 hover:underline disabled:opacity-50"
                      >
                        {deleteMutation.isPending ? "Deleting..." : "Delete"}
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
