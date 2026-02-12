import { useState } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import {
  registerCluster,
  testClusterConnection,
  getAgentInstallCommand,
} from "@/api/clusters";
import { ArrowLeft, ArrowRight, Check, Copy, Loader2 } from "lucide-react";

type Step = "info" | "kubeconfig" | "test" | "agent";

export default function ClusterRegister() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [step, setStep] = useState<Step>("info");
  const [name, setName] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [region, setRegion] = useState("");
  const [environment, setEnvironment] = useState("production");
  const [kubeconfig, setKubeconfig] = useState("");
  const [testResult, setTestResult] = useState<{
    connected: boolean;
    error?: string;
  } | null>(null);
  const [helmCommand, setHelmCommand] = useState("");
  const [copied, setCopied] = useState(false);

  const registerMutation = useMutation({
    mutationFn: registerCluster,
    onSuccess: async () => {
      queryClient.invalidateQueries({ queryKey: ["clusters"] });
      // Test connection.
      try {
        const result = await testClusterConnection(name);
        setTestResult(result);
      } catch {
        setTestResult({ connected: false, error: "Failed to test connection" });
      }
      setStep("test");
    },
  });

  const handleRegister = () => {
    registerMutation.mutate({
      name,
      displayName,
      region,
      environment,
      kubeconfig,
    });
  };

  const handleTestPassed = async () => {
    try {
      const info = await getAgentInstallCommand(name);
      setHelmCommand(info.helmCommand);
    } catch {
      setHelmCommand(`helm install ngf-console-agent charts/ngf-console-agent \\
  --namespace ngf-system --create-namespace \\
  --set cluster.name=${name}`);
    }
    setStep("agent");
  };

  const copyCommand = () => {
    navigator.clipboard.writeText(helmCommand);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      <div className="flex items-center gap-3">
        <button
          onClick={() => navigate("/clusters")}
          className="rounded-md p-1 text-muted-foreground hover:bg-accent"
        >
          <ArrowLeft className="h-5 w-5" />
        </button>
        <div>
          <h1 className="text-2xl font-bold text-foreground">
            Register Cluster
          </h1>
          <p className="text-sm text-muted-foreground">
            Add a workload cluster to the hub
          </p>
        </div>
      </div>

      {/* Step indicator */}
      <div className="flex items-center gap-2 text-xs">
        {(["info", "kubeconfig", "test", "agent"] as Step[]).map((s, i) => (
          <div key={s} className="flex items-center gap-2">
            {i > 0 && <div className="h-px w-6 bg-border" />}
            <span
              className={`rounded-full px-2 py-0.5 ${
                step === s
                  ? "bg-primary text-primary-foreground"
                  : "bg-muted text-muted-foreground"
              }`}
            >
              {i + 1}. {s.charAt(0).toUpperCase() + s.slice(1)}
            </span>
          </div>
        ))}
      </div>

      {step === "info" && (
        <div className="space-y-4 rounded-lg border border-border bg-card p-6">
          <div>
            <label className="mb-1 block text-sm font-medium text-foreground">
              Cluster Name
            </label>
            <input
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g., ai-inference"
              className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
            />
            <p className="mt-1 text-xs text-muted-foreground">
              Lowercase alphanumeric with hyphens
            </p>
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-foreground">
              Display Name
            </label>
            <input
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              placeholder="e.g., AI Inference (US-West-2)"
              className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="mb-1 block text-sm font-medium text-foreground">
                Region
              </label>
              <input
                value={region}
                onChange={(e) => setRegion(e.target.value)}
                placeholder="e.g., us-west-2"
                className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
              />
            </div>
            <div>
              <label className="mb-1 block text-sm font-medium text-foreground">
                Environment
              </label>
              <select
                value={environment}
                onChange={(e) => setEnvironment(e.target.value)}
                className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
              >
                <option value="production">Production</option>
                <option value="staging">Staging</option>
                <option value="dev">Development</option>
                <option value="gpu">GPU</option>
              </select>
            </div>
          </div>
          <div className="flex justify-end">
            <button
              onClick={() => setStep("kubeconfig")}
              disabled={!name || !displayName}
              className="inline-flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
            >
              Next <ArrowRight className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}

      {step === "kubeconfig" && (
        <div className="space-y-4 rounded-lg border border-border bg-card p-6">
          <div>
            <label className="mb-1 block text-sm font-medium text-foreground">
              Kubeconfig
            </label>
            <textarea
              value={kubeconfig}
              onChange={(e) => setKubeconfig(e.target.value)}
              rows={12}
              placeholder="Paste the kubeconfig YAML for the workload cluster..."
              className="w-full rounded-md border border-border bg-background px-3 py-2 font-mono text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
            />
            <p className="mt-1 text-xs text-muted-foreground">
              This will be stored as a Kubernetes Secret on the hub cluster
            </p>
          </div>
          <div className="flex justify-between">
            <button
              onClick={() => setStep("info")}
              className="rounded-md border border-border px-4 py-2 text-sm text-foreground hover:bg-accent"
            >
              Back
            </button>
            <button
              onClick={handleRegister}
              disabled={!kubeconfig || registerMutation.isPending}
              className="inline-flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50"
            >
              {registerMutation.isPending ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Registering...
                </>
              ) : (
                <>
                  Register & Test <ArrowRight className="h-4 w-4" />
                </>
              )}
            </button>
          </div>
        </div>
      )}

      {step === "test" && (
        <div className="space-y-4 rounded-lg border border-border bg-card p-6">
          <h2 className="text-lg font-semibold text-foreground">
            Connection Test
          </h2>
          {testResult ? (
            <div
              className={`rounded-md p-4 ${
                testResult.connected
                  ? "bg-green-500/10 text-green-500"
                  : "bg-red-500/10 text-red-500"
              }`}
            >
              <p className="font-medium">
                {testResult.connected
                  ? "Connection successful!"
                  : `Connection failed: ${testResult.error || "Unknown error"}`}
              </p>
            </div>
          ) : (
            <div className="flex items-center gap-2 text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" />
              Testing connection...
            </div>
          )}
          <div className="flex justify-between">
            <button
              onClick={() => setStep("kubeconfig")}
              className="rounded-md border border-border px-4 py-2 text-sm text-foreground hover:bg-accent"
            >
              Back
            </button>
            <button
              onClick={handleTestPassed}
              className="inline-flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
            >
              Continue to Agent Install <ArrowRight className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}

      {step === "agent" && (
        <div className="space-y-4 rounded-lg border border-border bg-card p-6">
          <h2 className="text-lg font-semibold text-foreground">
            Install Agent
          </h2>
          <p className="text-sm text-muted-foreground">
            Run this command on the workload cluster to install the NGF Console
            agent. The agent will send heartbeats and telemetry to the hub.
          </p>
          <div className="relative">
            <pre className="overflow-x-auto rounded-md bg-muted p-3 text-xs text-foreground">
              {helmCommand}
            </pre>
            <button
              onClick={copyCommand}
              className="absolute right-2 top-2 rounded p-1 text-muted-foreground hover:bg-accent"
            >
              {copied ? (
                <Check className="h-4 w-4 text-green-500" />
              ) : (
                <Copy className="h-4 w-4" />
              )}
            </button>
          </div>
          <div className="flex justify-end">
            <button
              onClick={() => navigate(`/clusters/${name}`)}
              className="inline-flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90"
            >
              <Check className="h-4 w-4" /> Done
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
