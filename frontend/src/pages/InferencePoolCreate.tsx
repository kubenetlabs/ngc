import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { Link } from "react-router-dom";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import apiClient from "@/api/client";

interface InferenceStackPayload {
  name: string;
  namespace: string;
  modelName: string;
  servingBackend: string;
  pool: {
    gpuType: string;
    gpuCount: number;
    replicas: number;
    minReplicas: number;
    maxReplicas: number;
    selector?: Record<string, string>;
  };
  epp?: {
    strategy: string;
    weights?: {
      queueDepth: number;
      kvCache: number;
      prefixAffinity: number;
    };
  };
}

const inputClass =
  "mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring";
const selectClass = inputClass;

const GPU_TYPES = [
  { value: "nvidia-t4", label: "NVIDIA T4" },
  { value: "nvidia-l4", label: "NVIDIA L4" },
  { value: "nvidia-l40s", label: "NVIDIA L40S" },
  { value: "nvidia-a100", label: "NVIDIA A100" },
  { value: "nvidia-h100", label: "NVIDIA H100" },
];

const EPP_STRATEGIES = [
  { value: "composite", label: "Composite" },
  { value: "least_queue", label: "Least Queue" },
  { value: "kv_cache", label: "KV Cache" },
  { value: "prefix_affinity", label: "Prefix Affinity" },
];

async function createInferenceStack(payload: InferenceStackPayload) {
  const { data } = await apiClient.post("/inference/stacks", payload);
  return data;
}

export default function InferencePoolCreate() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const [name, setName] = useState("");
  const [namespace, setNamespace] = useState("default");
  const [modelName, setModelName] = useState("");
  const [selectorKey, setSelectorKey] = useState("model");
  const [selectorValue, setSelectorValue] = useState("");
  const [replicas, setReplicas] = useState(1);
  const [gpuType, setGpuType] = useState("nvidia-t4");
  const [gpuCount, setGpuCount] = useState(1);
  const [servingBackend, setServingBackend] = useState("vllm");
  const [minReplicas, setMinReplicas] = useState(1);
  const [maxReplicas, setMaxReplicas] = useState(4);
  const [eppStrategy, setEppStrategy] = useState("composite");
  const [showWeights, setShowWeights] = useState(false);
  const [queueDepthWeight, setQueueDepthWeight] = useState(50);
  const [kvCacheWeight, setKvCacheWeight] = useState(30);
  const [prefixAffinityWeight, setPrefixAffinityWeight] = useState(20);

  const mutation = useMutation({
    mutationFn: createInferenceStack,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["inference-pools"] });
      queryClient.invalidateQueries({ queryKey: ["inference-stacks"] });
      navigate("/inference/pools");
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    const payload: InferenceStackPayload = {
      name,
      namespace,
      modelName,
      servingBackend,
      pool: {
        gpuType,
        gpuCount,
        replicas,
        minReplicas,
        maxReplicas,
        ...(selectorValue && { selector: { [selectorKey]: selectorValue } }),
      },
      epp: {
        strategy: eppStrategy,
        ...(showWeights && {
          weights: {
            queueDepth: queueDepthWeight,
            kvCache: kvCacheWeight,
            prefixAffinity: prefixAffinityWeight,
          },
        }),
      },
    };

    mutation.mutate(payload);
  };

  return (
    <div>
      <div className="mb-6">
        <Link
          to="/inference/pools"
          className="text-sm text-blue-400 hover:underline"
        >
          &larr; Back to Inference Pools
        </Link>
      </div>

      <h1 className="text-2xl font-bold">Create Inference Pool</h1>
      <p className="mt-1 text-muted-foreground">
        Configure a new InferencePool with GPU and EPP settings.
      </p>

      <form onSubmit={handleSubmit} className="mt-6 max-w-2xl space-y-6">
        {/* Basic Info */}
        <div className="rounded-lg border border-border p-4">
          <h2 className="text-lg font-semibold">Basic Information</h2>
          <div className="mt-4 grid gap-4 sm:grid-cols-2">
            <div>
              <label className="block text-sm font-medium">Name</label>
              <input
                value={name}
                onChange={(e) => setName(e.target.value)}
                className={inputClass}
                placeholder="llama3-70b-pool"
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
          </div>
        </div>

        {/* Model Configuration */}
        <div className="rounded-lg border border-border p-4">
          <h2 className="text-lg font-semibold">Model Configuration</h2>
          <div className="mt-4 space-y-4">
            <div>
              <label className="block text-sm font-medium">Model Name</label>
              <input
                value={modelName}
                onChange={(e) => setModelName(e.target.value)}
                className={inputClass}
                placeholder="meta-llama/Llama-3-70B-Instruct"
                required
              />
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <div>
                <label className="block text-sm font-medium">
                  Selector Key
                </label>
                <input
                  value={selectorKey}
                  onChange={(e) => setSelectorKey(e.target.value)}
                  className={inputClass}
                  placeholder="model"
                />
              </div>
              <div>
                <label className="block text-sm font-medium">
                  Selector Value
                </label>
                <input
                  value={selectorValue}
                  onChange={(e) => setSelectorValue(e.target.value)}
                  className={inputClass}
                  placeholder="llama3-70b"
                />
              </div>
            </div>
          </div>
        </div>

        {/* Pool Settings */}
        <div className="rounded-lg border border-border p-4">
          <h2 className="text-lg font-semibold">Pool Settings</h2>
          <div className="mt-4 grid gap-4 sm:grid-cols-2">
            <div>
              <label className="block text-sm font-medium">Serving Backend</label>
              <select
                value={servingBackend}
                onChange={(e) => setServingBackend(e.target.value)}
                className={selectClass}
              >
                <option value="vllm">vLLM</option>
                <option value="triton">Triton</option>
                <option value="tgi">TGI</option>
                <option value="ollama">Ollama</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium">Replicas</label>
              <input
                type="number"
                value={replicas}
                onChange={(e) => setReplicas(Number(e.target.value))}
                className={inputClass}
                min={1}
              />
            </div>
            <div>
              <label className="block text-sm font-medium">Min Replicas</label>
              <input
                type="number"
                value={minReplicas}
                onChange={(e) => setMinReplicas(Number(e.target.value))}
                className={inputClass}
                min={0}
              />
            </div>
            <div>
              <label className="block text-sm font-medium">Max Replicas</label>
              <input
                type="number"
                value={maxReplicas}
                onChange={(e) => setMaxReplicas(Number(e.target.value))}
                className={inputClass}
                min={1}
              />
            </div>
          </div>
        </div>

        {/* GPU Configuration */}
        <div className="rounded-lg border border-border p-4">
          <h2 className="text-lg font-semibold">GPU Configuration</h2>
          <div className="mt-4 grid gap-4 sm:grid-cols-2">
            <div>
              <label className="block text-sm font-medium">GPU Type</label>
              <select
                value={gpuType}
                onChange={(e) => setGpuType(e.target.value)}
                className={selectClass}
              >
                {GPU_TYPES.map((g) => (
                  <option key={g.value} value={g.value}>
                    {g.label}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium">
                GPUs per Node
              </label>
              <input
                type="number"
                value={gpuCount}
                onChange={(e) => setGpuCount(Number(e.target.value))}
                className={inputClass}
                min={1}
                max={8}
              />
            </div>
          </div>
        </div>

        {/* EPP Configuration */}
        <div className="rounded-lg border border-border p-4">
          <h2 className="text-lg font-semibold">EPP Configuration</h2>
          <div className="mt-4 space-y-4">
            <div>
              <label className="block text-sm font-medium">Strategy</label>
              <select
                value={eppStrategy}
                onChange={(e) => setEppStrategy(e.target.value)}
                className={selectClass}
              >
                {EPP_STRATEGIES.map((s) => (
                  <option key={s.value} value={s.value}>
                    {s.label}
                  </option>
                ))}
              </select>
            </div>

            <div>
              <label className="flex items-center gap-2 text-sm">
                <input
                  type="checkbox"
                  checked={showWeights}
                  onChange={(e) => setShowWeights(e.target.checked)}
                  className="h-4 w-4 rounded border-border bg-background"
                />
                Custom Weights
              </label>
            </div>

            {showWeights && (
              <div className="grid gap-4 sm:grid-cols-3">
                <div>
                  <label className="block text-xs text-muted-foreground">
                    Queue Depth
                  </label>
                  <input
                    type="number"
                    value={queueDepthWeight}
                    onChange={(e) =>
                      setQueueDepthWeight(Number(e.target.value))
                    }
                    className={inputClass}
                    min={0}
                    max={100}
                  />
                </div>
                <div>
                  <label className="block text-xs text-muted-foreground">
                    KV Cache
                  </label>
                  <input
                    type="number"
                    value={kvCacheWeight}
                    onChange={(e) =>
                      setKvCacheWeight(Number(e.target.value))
                    }
                    className={inputClass}
                    min={0}
                    max={100}
                  />
                </div>
                <div>
                  <label className="block text-xs text-muted-foreground">
                    Prefix Affinity
                  </label>
                  <input
                    type="number"
                    value={prefixAffinityWeight}
                    onChange={(e) =>
                      setPrefixAffinityWeight(Number(e.target.value))
                    }
                    className={inputClass}
                    min={0}
                    max={100}
                  />
                </div>
              </div>
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
            to="/inference/pools"
            className="rounded-md border border-border px-4 py-2 text-sm font-medium text-foreground hover:bg-muted/30"
          >
            Cancel
          </Link>
          <button
            type="submit"
            disabled={mutation.isPending || !name || !modelName}
            className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {mutation.isPending ? "Creating..." : "Create Inference Pool"}
          </button>
        </div>
      </form>
    </div>
  );
}
