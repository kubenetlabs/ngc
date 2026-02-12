import { useState } from "react";
import { useParams, useNavigate, Link } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import apiClient from "@/api/client";
import type { PolicyType } from "@/types/policy";

const inputClass =
  "mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring";
const selectClass = inputClass;

const POLICY_TYPE_LABELS: Record<PolicyType, string> = {
  ratelimit: "Rate Limit",
  clientsettings: "Client Settings",
  backendtls: "Backend TLS",
  observability: "Observability",
};

const POLICY_TYPES: PolicyType[] = [
  "ratelimit",
  "clientsettings",
  "backendtls",
  "observability",
];

function RateLimitFields({
  rate,
  setRate,
  burst,
  setBurst,
  unit,
  setUnit,
}: {
  rate: number;
  setRate: (v: number) => void;
  burst: number;
  setBurst: (v: number) => void;
  unit: string;
  setUnit: (v: string) => void;
}) {
  return (
    <>
      <div>
        <label className="block text-sm font-medium">Rate</label>
        <input
          type="number"
          value={rate}
          onChange={(e) => setRate(Number(e.target.value))}
          className={inputClass}
          min={1}
          placeholder="100"
        />
      </div>
      <div>
        <label className="block text-sm font-medium">Burst</label>
        <input
          type="number"
          value={burst}
          onChange={(e) => setBurst(Number(e.target.value))}
          className={inputClass}
          min={0}
          placeholder="50"
        />
      </div>
      <div>
        <label className="block text-sm font-medium">Unit</label>
        <select
          value={unit}
          onChange={(e) => setUnit(e.target.value)}
          className={selectClass}
        >
          <option value="per-second">Per Second</option>
          <option value="per-minute">Per Minute</option>
        </select>
      </div>
    </>
  );
}

function ClientSettingsFields({
  maxRequestBodySize,
  setMaxRequestBodySize,
  keepAliveTimeout,
  setKeepAliveTimeout,
}: {
  maxRequestBodySize: string;
  setMaxRequestBodySize: (v: string) => void;
  keepAliveTimeout: string;
  setKeepAliveTimeout: (v: string) => void;
}) {
  return (
    <>
      <div>
        <label className="block text-sm font-medium">
          Max Request Body Size
        </label>
        <input
          value={maxRequestBodySize}
          onChange={(e) => setMaxRequestBodySize(e.target.value)}
          className={inputClass}
          placeholder="16m"
        />
        <p className="mt-1 text-xs text-muted-foreground">
          e.g., 1m, 16m, 100k
        </p>
      </div>
      <div>
        <label className="block text-sm font-medium">
          Keep-Alive Timeout
        </label>
        <input
          value={keepAliveTimeout}
          onChange={(e) => setKeepAliveTimeout(e.target.value)}
          className={inputClass}
          placeholder="60s"
        />
        <p className="mt-1 text-xs text-muted-foreground">
          e.g., 30s, 60s, 120s
        </p>
      </div>
    </>
  );
}

function BackendTLSFields({
  sni,
  setSni,
  caCertRef,
  setCaCertRef,
}: {
  sni: string;
  setSni: (v: string) => void;
  caCertRef: string;
  setCaCertRef: (v: string) => void;
}) {
  return (
    <>
      <div>
        <label className="block text-sm font-medium">SNI</label>
        <input
          value={sni}
          onChange={(e) => setSni(e.target.value)}
          className={inputClass}
          placeholder="backend.example.com"
        />
      </div>
      <div>
        <label className="block text-sm font-medium">CA Cert Ref</label>
        <input
          value={caCertRef}
          onChange={(e) => setCaCertRef(e.target.value)}
          className={inputClass}
          placeholder="my-ca-cert"
        />
        <p className="mt-1 text-xs text-muted-foreground">
          Name of the Secret containing the CA certificate
        </p>
      </div>
    </>
  );
}

function ObservabilityFields({
  enableTracing,
  setEnableTracing,
  samplingRate,
  setSamplingRate,
}: {
  enableTracing: boolean;
  setEnableTracing: (v: boolean) => void;
  samplingRate: number;
  setSamplingRate: (v: number) => void;
}) {
  return (
    <>
      <div>
        <label className="flex items-center gap-2 text-sm font-medium">
          <input
            type="checkbox"
            checked={enableTracing}
            onChange={(e) => setEnableTracing(e.target.checked)}
            className="h-4 w-4 rounded border-border bg-background"
          />
          Enable Tracing
        </label>
      </div>
      <div>
        <label className="block text-sm font-medium">Sampling Rate</label>
        <input
          type="number"
          value={samplingRate}
          onChange={(e) => setSamplingRate(Number(e.target.value))}
          className={inputClass}
          min={0}
          max={100}
          step={1}
          placeholder="100"
        />
        <p className="mt-1 text-xs text-muted-foreground">
          Percentage of requests to trace (0-100)
        </p>
      </div>
    </>
  );
}

export default function PolicyCreate() {
  const { type: urlType } = useParams<{ type: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [policyType, setPolicyType] = useState<PolicyType>(
    (urlType as PolicyType) || "ratelimit",
  );
  const [name, setName] = useState("");
  const [namespace, setNamespace] = useState("default");
  const [targetGateway, setTargetGateway] = useState("");

  // Rate limit fields
  const [rate, setRate] = useState(100);
  const [burst, setBurst] = useState(50);
  const [unit, setUnit] = useState("per-second");

  // Client settings fields
  const [maxRequestBodySize, setMaxRequestBodySize] = useState("16m");
  const [keepAliveTimeout, setKeepAliveTimeout] = useState("60s");

  // Backend TLS fields
  const [sni, setSni] = useState("");
  const [caCertRef, setCaCertRef] = useState("");

  // Observability fields
  const [enableTracing, setEnableTracing] = useState(true);
  const [samplingRate, setSamplingRate] = useState(100);

  const mutation = useMutation({
    mutationFn: async (payload: {
      name: string;
      namespace: string;
      spec: Record<string, unknown>;
    }) => {
      const { data } = await apiClient.post(
        `/policies/${policyType}`,
        payload,
      );
      return data;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["policies"] });
      navigate("/policies");
    },
  });

  const buildSpec = (): Record<string, unknown> => {
    const base: Record<string, unknown> = {};
    if (targetGateway) {
      base.targetRef = { name: targetGateway };
    }

    switch (policyType) {
      case "ratelimit":
        return { ...base, rate, burst, unit };
      case "clientsettings":
        return { ...base, maxRequestBodySize, keepAliveTimeout };
      case "backendtls":
        return { ...base, sni, caCertRef };
      case "observability":
        return { ...base, enableTracing, samplingRate };
      default:
        return base;
    }
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    mutation.mutate({ name, namespace, spec: buildSpec() });
  };

  // If type is not in the URL, show a type selector
  const showTypeSelector = !urlType;

  return (
    <div>
      <div className="mb-6">
        <Link to="/policies" className="text-sm text-blue-400 hover:underline">
          &larr; Back to Policies
        </Link>
      </div>

      <h1 className="text-2xl font-bold">Create Policy</h1>
      <p className="mt-1 text-muted-foreground">
        Create a new {POLICY_TYPE_LABELS[policyType]} policy resource.
      </p>

      <form onSubmit={handleSubmit} className="mt-6 max-w-2xl space-y-6">
        {/* Policy Type Selector */}
        {showTypeSelector && (
          <div className="rounded-lg border border-border p-4">
            <h2 className="text-lg font-semibold">Policy Type</h2>
            <div className="mt-3 flex gap-2">
              {POLICY_TYPES.map((pt) => (
                <button
                  key={pt}
                  type="button"
                  onClick={() => setPolicyType(pt)}
                  className={`rounded-md px-3 py-1.5 text-sm font-medium transition-colors ${
                    policyType === pt
                      ? "bg-blue-600 text-white"
                      : "border border-border text-muted-foreground hover:text-foreground"
                  }`}
                >
                  {POLICY_TYPE_LABELS[pt]}
                </button>
              ))}
            </div>
          </div>
        )}

        {/* Common Fields */}
        <div className="rounded-lg border border-border p-4">
          <h2 className="text-lg font-semibold">Basic Information</h2>
          <div className="mt-4 grid gap-4 sm:grid-cols-2">
            <div>
              <label className="block text-sm font-medium">Name</label>
              <input
                value={name}
                onChange={(e) => setName(e.target.value)}
                className={inputClass}
                placeholder="my-policy"
                required
              />
            </div>
            <div>
              <label className="block text-sm font-medium">Namespace</label>
              <input
                value={namespace}
                onChange={(e) => setNamespace(e.target.value)}
                className={inputClass}
                placeholder="default"
                required
              />
            </div>
            <div className="sm:col-span-2">
              <label className="block text-sm font-medium">
                Target Gateway
              </label>
              <input
                value={targetGateway}
                onChange={(e) => setTargetGateway(e.target.value)}
                className={inputClass}
                placeholder="my-gateway (optional)"
              />
              <p className="mt-1 text-xs text-muted-foreground">
                Name of the Gateway this policy targets
              </p>
            </div>
          </div>
        </div>

        {/* Type-specific fields */}
        <div className="rounded-lg border border-border p-4">
          <h2 className="text-lg font-semibold">
            {POLICY_TYPE_LABELS[policyType]} Settings
          </h2>
          <div className="mt-4 grid gap-4 sm:grid-cols-2">
            {policyType === "ratelimit" && (
              <RateLimitFields
                rate={rate}
                setRate={setRate}
                burst={burst}
                setBurst={setBurst}
                unit={unit}
                setUnit={setUnit}
              />
            )}
            {policyType === "clientsettings" && (
              <ClientSettingsFields
                maxRequestBodySize={maxRequestBodySize}
                setMaxRequestBodySize={setMaxRequestBodySize}
                keepAliveTimeout={keepAliveTimeout}
                setKeepAliveTimeout={setKeepAliveTimeout}
              />
            )}
            {policyType === "backendtls" && (
              <BackendTLSFields
                sni={sni}
                setSni={setSni}
                caCertRef={caCertRef}
                setCaCertRef={setCaCertRef}
              />
            )}
            {policyType === "observability" && (
              <ObservabilityFields
                enableTracing={enableTracing}
                setEnableTracing={setEnableTracing}
                samplingRate={samplingRate}
                setSamplingRate={setSamplingRate}
              />
            )}
          </div>
        </div>

        {/* Error display */}
        {mutation.isError && (
          <div className="rounded-md border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-400">
            {String(mutation.error)}
          </div>
        )}

        {/* Submit */}
        <div className="flex items-center gap-3 border-t border-border pt-6">
          <Link
            to="/policies"
            className="rounded-md border border-border px-4 py-2 text-sm font-medium text-foreground hover:bg-muted/30"
          >
            Cancel
          </Link>
          <button
            type="submit"
            disabled={mutation.isPending || !name}
            className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {mutation.isPending
              ? "Creating..."
              : `Create ${POLICY_TYPE_LABELS[policyType]} Policy`}
          </button>
        </div>
      </form>
    </div>
  );
}
