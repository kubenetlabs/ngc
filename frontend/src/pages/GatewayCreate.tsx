import { useState } from "react";
import { useNavigate, Link } from "react-router-dom";
import { useForm, useFieldArray } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { createGatewayBundle, fetchGatewayClasses } from "@/api/gateways";
import {
  createGatewayBundleSchema,
  type CreateGatewayBundleFormData,
  type CreateGatewayBundlePayload,
  type GatewayClass,
} from "@/types/gateway";
import { useActiveCluster } from "@/hooks/useActiveCluster";

const STEPS = [
  "Class Selection",
  "Basic Config",
  "Listeners",
  "Advanced",
  "Review",
] as const;

const inputClass =
  "mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring";
const selectClass = inputClass;
const smallInputClass =
  "mt-1 w-full rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring";

function StepIndicator({ currentStep }: { currentStep: number }) {
  return (
    <div className="mb-8 flex items-center gap-1">
      {STEPS.map((label, i) => {
        const isActive = i === currentStep;
        const isCompleted = i < currentStep;
        return (
          <div key={label} className="flex items-center gap-1">
            {i > 0 && (
              <div
                className={`mx-1 h-px w-8 ${
                  isCompleted ? "bg-blue-500" : "bg-border"
                }`}
              />
            )}
            <div className="flex items-center gap-2">
              <div
                className={`flex h-7 w-7 items-center justify-center rounded-full text-xs font-medium ${
                  isActive
                    ? "bg-blue-600 text-white"
                    : isCompleted
                      ? "bg-blue-600/20 text-blue-400"
                      : "bg-muted text-muted-foreground"
                }`}
              >
                {isCompleted ? "\u2713" : i + 1}
              </div>
              <span
                className={`hidden text-sm sm:inline ${
                  isActive
                    ? "font-medium text-foreground"
                    : "text-muted-foreground"
                }`}
              >
                {label}
              </span>
            </div>
          </div>
        );
      })}
    </div>
  );
}

function StepClassSelection({
  gatewayClasses,
  selectedClass,
  register,
  error,
}: {
  gatewayClasses: GatewayClass[] | undefined;
  selectedClass: string;
  register: ReturnType<typeof useForm<CreateGatewayBundleFormData>>["register"];
  error?: string;
}) {
  const selected = gatewayClasses?.find((gc) => gc.name === selectedClass);

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-lg font-semibold">Select Gateway Class</h2>
        <p className="mt-1 text-sm text-muted-foreground">
          Choose the GatewayClass that defines the controller for this gateway.
        </p>
      </div>

      <div>
        <label htmlFor="gatewayClassName" className="block text-sm font-medium">
          Gateway Class
        </label>
        <select
          id="gatewayClassName"
          {...register("gatewayClassName")}
          className={selectClass}
        >
          <option value="">Select a gateway class...</option>
          {gatewayClasses?.map((gc) => (
            <option key={gc.name} value={gc.name}>
              {gc.name}
            </option>
          ))}
        </select>
        {error && <p className="mt-1 text-xs text-red-400">{error}</p>}
      </div>

      {selected && (
        <div className="rounded-lg border border-border bg-muted/20 p-4">
          <h3 className="text-sm font-medium">Class Details</h3>
          <dl className="mt-2 space-y-2 text-sm">
            <div>
              <dt className="text-muted-foreground">Controller</dt>
              <dd className="font-mono text-foreground">
                {selected.controllerName}
              </dd>
            </div>
            {selected.description && (
              <div>
                <dt className="text-muted-foreground">Description</dt>
                <dd className="text-foreground">{selected.description}</dd>
              </div>
            )}
            {selected.parametersRef && (
              <div>
                <dt className="text-muted-foreground">Parameters Ref</dt>
                <dd className="font-mono text-foreground">
                  {selected.parametersRef.kind}/{selected.parametersRef.name}
                </dd>
              </div>
            )}
          </dl>
        </div>
      )}
    </div>
  );
}

function StepBasicConfig({
  register,
  errors,
}: {
  register: ReturnType<typeof useForm<CreateGatewayBundleFormData>>["register"];
  errors: Record<string, { message?: string } | undefined>;
}) {
  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-lg font-semibold">Basic Configuration</h2>
        <p className="mt-1 text-sm text-muted-foreground">
          Set the name, namespace, and labels for your gateway.
        </p>
      </div>

      <div>
        <label htmlFor="name" className="block text-sm font-medium">
          Name
        </label>
        <input
          id="name"
          {...register("name")}
          className={inputClass}
          placeholder="my-gateway"
        />
        {errors.name && (
          <p className="mt-1 text-xs text-red-400">{errors.name.message}</p>
        )}
      </div>

      <div>
        <label htmlFor="namespace" className="block text-sm font-medium">
          Namespace
        </label>
        <input
          id="namespace"
          {...register("namespace")}
          className={inputClass}
          placeholder="default"
        />
        {errors.namespace && (
          <p className="mt-1 text-xs text-red-400">
            {errors.namespace.message}
          </p>
        )}
      </div>
    </div>
  );
}

function StepListeners({
  register,
  fields,
  append,
  remove,
  errors,
}: {
  register: ReturnType<typeof useForm<CreateGatewayBundleFormData>>["register"];
  fields: { id: string }[];
  append: (value: {
    name: string;
    port: number;
    protocol: "HTTP" | "HTTPS" | "TLS" | "TCP" | "UDP";
    hostname?: string;
  }) => void;
  remove: (index: number) => void;
  errors: ReturnType<typeof useForm<CreateGatewayBundleFormData>>["formState"]["errors"];
}) {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold">Listeners</h2>
          <p className="mt-1 text-sm text-muted-foreground">
            Configure the listeners for your gateway. At least one listener is
            required.
          </p>
        </div>
        <button
          type="button"
          onClick={() =>
            append({ name: "", port: 80, protocol: "HTTP", hostname: "" })
          }
          className="text-sm text-blue-400 hover:underline"
        >
          + Add Listener
        </button>
      </div>

      {errors.listeners?.root && (
        <p className="text-xs text-red-400">
          {errors.listeners.root.message}
        </p>
      )}

      <div className="space-y-4">
        {fields.map((field, index) => (
          <div key={field.id} className="rounded-lg border border-border p-4">
            <div className="flex items-center justify-between">
              <span className="text-sm font-medium text-muted-foreground">
                Listener {index + 1}
              </span>
              {fields.length > 1 && (
                <button
                  type="button"
                  onClick={() => remove(index)}
                  className="text-sm text-red-400 hover:underline"
                >
                  Remove
                </button>
              )}
            </div>

            <div className="mt-3 grid grid-cols-2 gap-4">
              <div>
                <label className="block text-xs text-muted-foreground">
                  Name
                </label>
                <input
                  {...register(`listeners.${index}.name`)}
                  className={smallInputClass}
                  placeholder="http"
                />
                {errors.listeners?.[index]?.name && (
                  <p className="mt-1 text-xs text-red-400">
                    {errors.listeners[index].name.message}
                  </p>
                )}
              </div>

              <div>
                <label className="block text-xs text-muted-foreground">
                  Port
                </label>
                <input
                  type="number"
                  {...register(`listeners.${index}.port`, {
                    valueAsNumber: true,
                  })}
                  className={smallInputClass}
                  placeholder="80"
                />
                {errors.listeners?.[index]?.port && (
                  <p className="mt-1 text-xs text-red-400">
                    {errors.listeners[index].port.message}
                  </p>
                )}
              </div>

              <div>
                <label className="block text-xs text-muted-foreground">
                  Protocol
                </label>
                <select
                  {...register(`listeners.${index}.protocol`)}
                  className={smallInputClass}
                >
                  <option value="HTTP">HTTP</option>
                  <option value="HTTPS">HTTPS</option>
                  <option value="TLS">TLS</option>
                  <option value="TCP">TCP</option>
                  <option value="UDP">UDP</option>
                </select>
              </div>

              <div>
                <label className="block text-xs text-muted-foreground">
                  Hostname (optional)
                </label>
                <input
                  {...register(`listeners.${index}.hostname`)}
                  className={smallInputClass}
                  placeholder="example.com"
                />
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function StepAdvanced({
  register,
}: {
  register: ReturnType<typeof useForm<CreateGatewayBundleFormData>>["register"];
}) {
  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-lg font-semibold">Advanced Configuration</h2>
        <p className="mt-1 text-sm text-muted-foreground">
          Configure optional enterprise features for your gateway.
        </p>
      </div>

      {/* NginxProxy */}
      <div className="rounded-lg border border-border p-4 opacity-60">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <input
              type="checkbox"
              {...register("enableNginxProxy")}
              disabled
              className="h-4 w-4 rounded border-border bg-background"
            />
            <div>
              <span className="text-sm font-medium">NginxProxy</span>
              <span className="ml-2 rounded bg-yellow-500/20 px-1.5 py-0.5 text-xs font-medium text-yellow-400">
                Enterprise
              </span>
            </div>
          </div>
        </div>
        <p className="mt-2 text-sm text-muted-foreground">
          Configure NginxProxy settings including IP family, client IP
          rewriting, and telemetry exporters.
        </p>
      </div>

      {/* WAF */}
      <div className="rounded-lg border border-border p-4 opacity-60">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <input
              type="checkbox"
              {...register("enableWAF")}
              disabled
              className="h-4 w-4 rounded border-border bg-background"
            />
            <div>
              <span className="text-sm font-medium">
                Web Application Firewall
              </span>
              <span className="ml-2 rounded bg-yellow-500/20 px-1.5 py-0.5 text-xs font-medium text-yellow-400">
                Enterprise
              </span>
            </div>
          </div>
        </div>
        <p className="mt-2 text-sm text-muted-foreground">
          Enable WAF protection with configurable policy references for
          enhanced security.
        </p>
      </div>

      {/* SnippetsFilter */}
      <div className="rounded-lg border border-border p-4 opacity-60">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <input type="checkbox" disabled className="h-4 w-4 rounded border-border bg-background" />
            <div>
              <span className="text-sm font-medium">Snippets Filter</span>
              <span className="ml-2 rounded bg-yellow-500/20 px-1.5 py-0.5 text-xs font-medium text-yellow-400">
                Enterprise
              </span>
            </div>
          </div>
        </div>
        <p className="mt-2 text-sm text-muted-foreground">
          Inject custom NGINX configuration snippets at the server and
          location block level.
        </p>
      </div>
    </div>
  );
}

function buildPayloadPreview(data: CreateGatewayBundleFormData): CreateGatewayBundlePayload {
  return {
    name: data.name,
    namespace: data.namespace,
    gatewayClassName: data.gatewayClassName,
    listeners: data.listeners.map((l) => ({
      name: l.name,
      port: l.port,
      protocol: l.protocol,
      hostname: l.hostname || undefined,
    })),
  };
}

function StepReview({
  data,
  gatewayClasses,
  showYAML,
  setShowYAML,
}: {
  data: CreateGatewayBundleFormData;
  gatewayClasses: GatewayClass[] | undefined;
  showYAML: boolean;
  setShowYAML: (v: boolean) => void;
}) {
  const selectedClass = gatewayClasses?.find(
    (gc) => gc.name === data.gatewayClassName,
  );

  const payload = buildPayloadPreview(data);

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-lg font-semibold">Review Configuration</h2>
        <p className="mt-1 text-sm text-muted-foreground">
          Verify your gateway configuration before creating.
        </p>
      </div>

      <div className="rounded-lg border border-border bg-muted/20 p-4">
        <dl className="space-y-3 text-sm">
          <div className="flex justify-between">
            <dt className="text-muted-foreground">Name</dt>
            <dd className="font-medium text-foreground">{data.name}</dd>
          </div>
          <div className="flex justify-between">
            <dt className="text-muted-foreground">Namespace</dt>
            <dd className="font-medium text-foreground">{data.namespace}</dd>
          </div>
          <div className="flex justify-between">
            <dt className="text-muted-foreground">Gateway Class</dt>
            <dd className="font-medium text-foreground">
              {data.gatewayClassName}
              {selectedClass?.description && (
                <span className="ml-2 text-muted-foreground">
                  ({selectedClass.description})
                </span>
              )}
            </dd>
          </div>
        </dl>
      </div>

      {/* Listeners Summary */}
      <div className="rounded-lg border border-border bg-muted/20 p-4">
        <h3 className="mb-3 text-sm font-medium">
          Listeners ({data.listeners.length})
        </h3>
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border">
                <th className="pb-2 text-left font-medium text-muted-foreground">
                  Name
                </th>
                <th className="pb-2 text-left font-medium text-muted-foreground">
                  Port
                </th>
                <th className="pb-2 text-left font-medium text-muted-foreground">
                  Protocol
                </th>
                <th className="pb-2 text-left font-medium text-muted-foreground">
                  Hostname
                </th>
              </tr>
            </thead>
            <tbody>
              {data.listeners.map((l, i) => (
                <tr key={i} className="border-b border-border last:border-0">
                  <td className="py-2 font-medium">{l.name}</td>
                  <td className="py-2 text-muted-foreground">{l.port}</td>
                  <td className="py-2 text-muted-foreground">{l.protocol}</td>
                  <td className="py-2 font-mono text-muted-foreground">
                    {l.hostname || "*"}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* YAML Preview Toggle */}
      <div>
        <button
          type="button"
          onClick={() => setShowYAML(!showYAML)}
          className="text-sm text-blue-400 hover:underline"
        >
          {showYAML ? "Hide" : "Show"} YAML Preview
        </button>
        {showYAML && (
          <pre className="mt-3 max-h-80 overflow-auto rounded-lg border border-border bg-muted/30 p-4 text-xs text-foreground">
            {JSON.stringify(payload, null, 2)}
          </pre>
        )}
      </div>
    </div>
  );
}

export default function GatewayCreate() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const activeCluster = useActiveCluster();
  const [currentStep, setCurrentStep] = useState(0);
  const [showYAML, setShowYAML] = useState(false);

  const { data: gatewayClasses } = useQuery({
    queryKey: ["gatewayclasses", activeCluster],
    queryFn: fetchGatewayClasses,
  });

  const {
    register,
    control,
    handleSubmit,
    trigger,
    watch,
    formState: { errors },
  } = useForm<CreateGatewayBundleFormData>({
    resolver: zodResolver(createGatewayBundleSchema),
    defaultValues: {
      name: "",
      namespace: "default",
      gatewayClassName: "",
      listeners: [{ name: "http", port: 80, protocol: "HTTP", hostname: "" }],
      enableNginxProxy: false,
      enableWAF: false,
    },
  });

  const { fields, append, remove } = useFieldArray({
    control,
    name: "listeners",
  });

  const mutation = useMutation({
    mutationFn: createGatewayBundle,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["gateways"] });
      queryClient.invalidateQueries({ queryKey: ["gatewaybundles"] });
      navigate("/gateways");
    },
  });

  const watchedData = watch();

  const onSubmit = (data: CreateGatewayBundleFormData) => {
    const payload = buildPayloadPreview(data);
    mutation.mutate(payload);
  };

  const validateCurrentStep = async (): Promise<boolean> => {
    switch (currentStep) {
      case 0:
        return trigger("gatewayClassName");
      case 1:
        return trigger(["name", "namespace"]);
      case 2:
        return trigger("listeners");
      case 3:
        return true;
      default:
        return true;
    }
  };

  const handleNext = async () => {
    const valid = await validateCurrentStep();
    if (valid) {
      setCurrentStep((prev) => Math.min(prev + 1, STEPS.length - 1));
    }
  };

  const handleBack = () => {
    setCurrentStep((prev) => Math.max(prev - 1, 0));
  };

  return (
    <div>
      <div className="mb-6">
        <Link to="/gateways" className="text-sm text-blue-400 hover:underline">
          &larr; Back to Gateways
        </Link>
      </div>

      <h1 className="text-2xl font-bold">Create Gateway</h1>
      <p className="mt-1 text-muted-foreground">
        Create a new GatewayBundle resource using the wizard below.
      </p>

      <div className="mt-6 max-w-2xl">
        <StepIndicator currentStep={currentStep} />

        {mutation.isError && (
          <div className="mb-6 rounded-md border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-400">
            {String(mutation.error)}
          </div>
        )}

        <form onSubmit={handleSubmit(onSubmit)}>
          {currentStep === 0 && (
            <StepClassSelection
              gatewayClasses={gatewayClasses}
              selectedClass={watchedData.gatewayClassName}
              register={register}
              error={errors.gatewayClassName?.message}
            />
          )}

          {currentStep === 1 && (
            <StepBasicConfig register={register} errors={errors} />
          )}

          {currentStep === 2 && (
            <StepListeners
              register={register}
              fields={fields}
              append={append}
              remove={remove}
              errors={errors}
            />
          )}

          {currentStep === 3 && <StepAdvanced register={register} />}

          {currentStep === 4 && (
            <StepReview
              data={watchedData}
              gatewayClasses={gatewayClasses}
              showYAML={showYAML}
              setShowYAML={setShowYAML}
            />
          )}

          {/* Navigation Buttons */}
          <div className="mt-8 flex items-center justify-between border-t border-border pt-6">
            <div>
              {currentStep > 0 && (
                <button
                  type="button"
                  onClick={handleBack}
                  className="rounded-md border border-border px-4 py-2 text-sm font-medium text-foreground hover:bg-muted/30"
                >
                  Back
                </button>
              )}
            </div>

            <div className="flex gap-3">
              <Link
                to="/gateways"
                className="rounded-md border border-border px-4 py-2 text-sm font-medium text-foreground hover:bg-muted/30"
              >
                Cancel
              </Link>

              {currentStep < STEPS.length - 1 ? (
                <button
                  type="button"
                  onClick={handleNext}
                  className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
                >
                  Next
                </button>
              ) : (
                <button
                  type="submit"
                  disabled={mutation.isPending}
                  className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
                >
                  {mutation.isPending ? "Creating..." : "Create Gateway"}
                </button>
              )}
            </div>
          </div>
        </form>
      </div>
    </div>
  );
}
