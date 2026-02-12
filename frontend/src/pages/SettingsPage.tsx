import { useState, useEffect, useCallback } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useActiveCluster } from "@/hooks/useActiveCluster";
import {
  fetchAlertRules,
  createAlertRule,
  deleteAlertRule,
  toggleAlertRule,
  type AlertRule,
  type CreateAlertRuleRequest,
} from "@/api/alerts";
import { Bell, Trash2, ExternalLink } from "lucide-react";

const inputClass =
  "mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring";
const selectClass = inputClass;

function severityBadge(severity: string): string {
  switch (severity) {
    case "critical":
      return "bg-red-500/10 text-red-400";
    case "warning":
      return "bg-yellow-500/10 text-yellow-400";
    case "info":
      return "bg-blue-500/10 text-blue-400";
    default:
      return "bg-zinc-500/10 text-zinc-400";
  }
}

function operatorLabel(op: string): string {
  switch (op) {
    case "gt":
      return ">";
    case "lt":
      return "<";
    case "eq":
      return "=";
    default:
      return op;
  }
}

// --- Webhook / Notification Channel types ---

const WEBHOOKS_KEY = "ngf-notification-webhooks";

interface WebhookConfig {
  id: string;
  url: string;
  description: string;
  createdAt: string;
}

function loadWebhooks(): WebhookConfig[] {
  try {
    const raw = localStorage.getItem(WEBHOOKS_KEY);
    if (!raw) return [];
    return JSON.parse(raw) as WebhookConfig[];
  } catch {
    return [];
  }
}

function saveWebhooks(webhooks: WebhookConfig[]): void {
  localStorage.setItem(WEBHOOKS_KEY, JSON.stringify(webhooks));
}

// --- Notifications Tab ---

function NotificationsTab() {
  const [webhooks, setWebhooks] = useState<WebhookConfig[]>([]);
  const [newUrl, setNewUrl] = useState("");
  const [newDescription, setNewDescription] = useState("");
  const [urlError, setUrlError] = useState("");

  useEffect(() => {
    setWebhooks(loadWebhooks());
  }, []);

  const handleAdd = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault();
      setUrlError("");

      const trimmedUrl = newUrl.trim();
      if (!trimmedUrl) return;

      try {
        new URL(trimmedUrl);
      } catch {
        setUrlError("Please enter a valid URL.");
        return;
      }

      const webhook: WebhookConfig = {
        id: crypto.randomUUID(),
        url: trimmedUrl,
        description: newDescription.trim(),
        createdAt: new Date().toISOString(),
      };

      const updated = [...webhooks, webhook];
      setWebhooks(updated);
      saveWebhooks(updated);
      setNewUrl("");
      setNewDescription("");
    },
    [newUrl, newDescription, webhooks],
  );

  const handleDelete = useCallback(
    (id: string) => {
      const updated = webhooks.filter((w) => w.id !== id);
      setWebhooks(updated);
      saveWebhooks(updated);
    },
    [webhooks],
  );

  return (
    <div className="mt-6 space-y-6">
      {/* Info section */}
      <div className="rounded-lg border border-blue-500/20 bg-blue-500/5 p-4">
        <div className="flex items-start gap-3">
          <Bell className="mt-0.5 h-5 w-5 text-blue-400" />
          <div>
            <p className="text-sm font-medium text-foreground">
              Notification Channels
            </p>
            <p className="mt-1 text-sm text-muted-foreground">
              Configure webhook endpoints to receive alert notifications.
              Supported integrations include Slack incoming webhooks, PagerDuty
              Events API, and generic HTTP webhooks.
            </p>
          </div>
        </div>
      </div>

      {/* Add webhook form */}
      <div>
        <h3 className="text-sm font-semibold text-foreground">
          Add Webhook
        </h3>
        <form onSubmit={handleAdd} className="mt-3 space-y-3">
          <div>
            <label className="block text-sm font-medium text-muted-foreground">
              Webhook URL
            </label>
            <input
              type="url"
              value={newUrl}
              onChange={(e) => {
                setNewUrl(e.target.value);
                setUrlError("");
              }}
              className={inputClass}
              placeholder="https://hooks.slack.com/services/..."
              required
            />
            {urlError && (
              <p className="mt-1 text-xs text-red-400">{urlError}</p>
            )}
          </div>
          <div>
            <label className="block text-sm font-medium text-muted-foreground">
              Description (optional)
            </label>
            <input
              type="text"
              value={newDescription}
              onChange={(e) => setNewDescription(e.target.value)}
              className={inputClass}
              placeholder="e.g., Slack #alerts channel"
            />
          </div>
          <div className="flex justify-end">
            <button
              type="submit"
              className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
            >
              Add Webhook
            </button>
          </div>
        </form>
      </div>

      {/* Webhook list */}
      <div>
        <h3 className="text-sm font-semibold text-foreground">
          Configured Webhooks
        </h3>
        {webhooks.length === 0 ? (
          <div className="mt-3 rounded-lg border border-border p-8 text-center">
            <Bell className="mx-auto h-8 w-8 text-muted-foreground" />
            <p className="mt-2 text-sm text-muted-foreground">
              No webhooks configured yet. Add one above to start receiving
              notifications.
            </p>
          </div>
        ) : (
          <div className="mt-3 space-y-2">
            {webhooks.map((webhook) => (
              <div
                key={webhook.id}
                className="flex items-center justify-between rounded-lg border border-border bg-card p-4"
              >
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <ExternalLink className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
                    <span className="truncate font-mono text-sm text-foreground">
                      {webhook.url}
                    </span>
                  </div>
                  {webhook.description && (
                    <p className="mt-1 text-xs text-muted-foreground">
                      {webhook.description}
                    </p>
                  )}
                  <p className="mt-1 text-xs text-muted-foreground">
                    Added{" "}
                    {new Date(webhook.createdAt).toLocaleDateString(undefined, {
                      year: "numeric",
                      month: "short",
                      day: "numeric",
                    })}
                  </p>
                </div>
                <button
                  onClick={() => handleDelete(webhook.id)}
                  className="ml-4 shrink-0 rounded p-1.5 text-muted-foreground hover:bg-red-500/10 hover:text-red-400"
                  title="Remove webhook"
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

// --- Main Settings Page ---

const TABS = ["Alert Rules", "Notifications", "Preferences"] as const;

export default function SettingsPage() {
  const activeCluster = useActiveCluster();
  const queryClient = useQueryClient();
  const [activeTab, setActiveTab] = useState<(typeof TABS)[number]>(
    "Alert Rules",
  );
  const [showCreateForm, setShowCreateForm] = useState(false);

  // Alert rule form state
  const [newRule, setNewRule] = useState<CreateAlertRuleRequest>({
    name: "",
    resource: "gateway",
    metric: "",
    operator: "gt",
    threshold: 0,
    severity: "warning",
  });

  const {
    data: alertRules,
    isLoading,
    error,
  } = useQuery({
    queryKey: ["alert-rules", activeCluster],
    queryFn: fetchAlertRules,
    enabled: activeTab === "Alert Rules",
  });

  const createMutation = useMutation({
    mutationFn: createAlertRule,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["alert-rules"] });
      setShowCreateForm(false);
      setNewRule({
        name: "",
        resource: "gateway",
        metric: "",
        operator: "gt",
        threshold: 0,
        severity: "warning",
      });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteAlertRule,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["alert-rules"] });
    },
  });

  const toggleMutation = useMutation({
    mutationFn: toggleAlertRule,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["alert-rules"] });
    },
  });

  const handleCreateSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newRule.name.trim() || !newRule.metric.trim()) return;
    createMutation.mutate(newRule);
  };

  const handleDelete = (id: string, name: string) => {
    if (!confirm(`Delete alert rule "${name}"?`)) return;
    deleteMutation.mutate(id);
  };

  return (
    <div>
      <div>
        <h1 className="text-2xl font-bold">Settings</h1>
        <p className="mt-1 text-muted-foreground">
          User preferences, alert rules, and system configuration.
        </p>
      </div>

      {/* Tabs */}
      <div className="mt-6 flex gap-2 border-b border-border">
        {TABS.map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`border-b-2 px-4 py-2 text-sm font-medium transition-colors ${
              activeTab === tab
                ? "border-blue-500 text-foreground"
                : "border-transparent text-muted-foreground hover:text-foreground"
            }`}
          >
            {tab}
          </button>
        ))}
      </div>

      {/* Tab: Alert Rules */}
      {activeTab === "Alert Rules" && (
        <div className="mt-6">
          <div className="flex items-center justify-between">
            <h2 className="text-lg font-semibold">Alert Rules</h2>
            <button
              onClick={() => setShowCreateForm(!showCreateForm)}
              className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
            >
              {showCreateForm ? "Cancel" : "Create Alert Rule"}
            </button>
          </div>

          {/* Create Form */}
          {showCreateForm && (
            <form
              onSubmit={handleCreateSubmit}
              className="mt-4 rounded-lg border border-border p-4"
            >
              <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
                <div>
                  <label className="block text-sm font-medium">Name</label>
                  <input
                    value={newRule.name}
                    onChange={(e) =>
                      setNewRule({ ...newRule, name: e.target.value })
                    }
                    className={inputClass}
                    placeholder="high-error-rate"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium">Resource</label>
                  <select
                    value={newRule.resource}
                    onChange={(e) =>
                      setNewRule({
                        ...newRule,
                        resource: e.target.value as
                          | "certificate"
                          | "gateway"
                          | "inference",
                      })
                    }
                    className={selectClass}
                  >
                    <option value="certificate">Certificate</option>
                    <option value="gateway">Gateway</option>
                    <option value="inference">Inference</option>
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium">Metric</label>
                  <input
                    value={newRule.metric}
                    onChange={(e) =>
                      setNewRule({ ...newRule, metric: e.target.value })
                    }
                    className={inputClass}
                    placeholder="error_rate"
                    required
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium">Operator</label>
                  <select
                    value={newRule.operator}
                    onChange={(e) =>
                      setNewRule({
                        ...newRule,
                        operator: e.target.value as "gt" | "lt" | "eq",
                      })
                    }
                    className={selectClass}
                  >
                    <option value="gt">Greater than (&gt;)</option>
                    <option value="lt">Less than (&lt;)</option>
                    <option value="eq">Equal to (=)</option>
                  </select>
                </div>
                <div>
                  <label className="block text-sm font-medium">Threshold</label>
                  <input
                    type="number"
                    value={newRule.threshold}
                    onChange={(e) =>
                      setNewRule({
                        ...newRule,
                        threshold: Number(e.target.value),
                      })
                    }
                    className={inputClass}
                    placeholder="90"
                  />
                </div>
                <div>
                  <label className="block text-sm font-medium">Severity</label>
                  <select
                    value={newRule.severity}
                    onChange={(e) =>
                      setNewRule({
                        ...newRule,
                        severity: e.target.value as
                          | "critical"
                          | "warning"
                          | "info",
                      })
                    }
                    className={selectClass}
                  >
                    <option value="critical">Critical</option>
                    <option value="warning">Warning</option>
                    <option value="info">Info</option>
                  </select>
                </div>
              </div>

              {createMutation.isError && (
                <p className="mt-3 text-sm text-red-400">
                  {String(createMutation.error)}
                </p>
              )}

              <div className="mt-4 flex justify-end">
                <button
                  type="submit"
                  disabled={createMutation.isPending}
                  className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
                >
                  {createMutation.isPending ? "Creating..." : "Create Rule"}
                </button>
              </div>
            </form>
          )}

          {isLoading && (
            <p className="mt-4 text-muted-foreground">
              Loading alert rules...
            </p>
          )}
          {error && (
            <p className="mt-4 text-red-400">
              Failed to load alert rules: {String(error)}
            </p>
          )}

          {alertRules && alertRules.length === 0 && !showCreateForm && (
            <p className="mt-4 text-muted-foreground">
              No alert rules configured. Create one to get started.
            </p>
          )}

          {alertRules && alertRules.length > 0 && (
            <div className="mt-4 overflow-x-auto rounded-lg border border-border">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border bg-muted/30">
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                      Name
                    </th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                      Resource
                    </th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                      Condition
                    </th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                      Severity
                    </th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                      Enabled
                    </th>
                    <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                      Actions
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {alertRules.map((rule: AlertRule) => (
                    <tr
                      key={rule.id}
                      className="border-b border-border last:border-0 hover:bg-muted/20"
                    >
                      <td className="px-4 py-3 font-medium">{rule.name}</td>
                      <td className="px-4 py-3 text-muted-foreground">
                        {rule.resource}
                      </td>
                      <td className="px-4 py-3 font-mono text-xs text-muted-foreground">
                        {rule.metric} {operatorLabel(rule.operator)}{" "}
                        {rule.threshold}
                      </td>
                      <td className="px-4 py-3">
                        <span
                          className={`inline-flex rounded px-2 py-0.5 text-xs font-medium ${severityBadge(rule.severity)}`}
                        >
                          {rule.severity}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <button
                          onClick={() => toggleMutation.mutate(rule.id)}
                          disabled={toggleMutation.isPending}
                          className={`relative inline-flex h-5 w-9 items-center rounded-full transition-colors ${
                            rule.enabled ? "bg-blue-600" : "bg-muted"
                          }`}
                        >
                          <span
                            className={`inline-block h-3.5 w-3.5 rounded-full bg-white transition-transform ${
                              rule.enabled
                                ? "translate-x-[18px]"
                                : "translate-x-[3px]"
                            }`}
                          />
                        </button>
                      </td>
                      <td className="px-4 py-3">
                        <button
                          onClick={() => handleDelete(rule.id, rule.name)}
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

      {/* Tab: Notifications */}
      {activeTab === "Notifications" && <NotificationsTab />}

      {/* Tab: Preferences */}
      {activeTab === "Preferences" && (
        <div className="mt-6">
          <div className="rounded-lg border border-border p-8 text-center">
            <p className="text-lg font-medium text-foreground">
              Preferences
            </p>
            <p className="mt-2 text-sm text-muted-foreground">
              Coming soon. User preferences and theme settings will be
              available in a future update.
            </p>
          </div>
        </div>
      )}
    </div>
  );
}
