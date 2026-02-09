import { useNavigate, useParams, Link } from "react-router-dom";
import { useForm, useFieldArray } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { fetchGateway, fetchGatewayClasses, updateGateway } from "@/api/gateways";
import { listenerSchema, type UpdateGatewayPayload } from "@/types/gateway";
import { useActiveCluster } from "@/hooks/useActiveCluster";
import { z } from "zod";

const updateGatewaySchema = z.object({
  gatewayClassName: z.string().min(1, "Gateway class is required"),
  listeners: z.array(listenerSchema).min(1, "At least one listener is required"),
});

type UpdateGatewayFormData = z.infer<typeof updateGatewaySchema>;

export default function GatewayEdit() {
  const { ns, name } = useParams<{ ns: string; name: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const activeCluster = useActiveCluster();

  const { data: gw, isLoading } = useQuery({
    queryKey: ["gateway", activeCluster, ns, name],
    queryFn: () => fetchGateway(ns!, name!),
    enabled: !!ns && !!name,
  });

  const { data: gatewayClasses } = useQuery({
    queryKey: ["gatewayclasses", activeCluster],
    queryFn: fetchGatewayClasses,
  });

  const {
    register,
    control,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<UpdateGatewayFormData>({
    resolver: zodResolver(updateGatewaySchema),
    values: gw
      ? {
          gatewayClassName: gw.gatewayClassName,
          listeners: gw.listeners.map((l) => ({
            name: l.name,
            port: l.port,
            protocol: l.protocol,
            hostname: l.hostname ?? "",
          })),
        }
      : undefined,
  });

  const { fields, append, remove } = useFieldArray({ control, name: "listeners" });

  const mutation = useMutation({
    mutationFn: (payload: UpdateGatewayPayload) => updateGateway(ns!, name!, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["gateways"] });
      queryClient.invalidateQueries({ queryKey: ["gateway", activeCluster, ns, name] });
      navigate(`/gateways/${ns}/${name}`);
    },
  });

  const onSubmit = (data: UpdateGatewayFormData) => {
    const payload: UpdateGatewayPayload = {
      ...data,
      listeners: data.listeners.map((l) => ({
        ...l,
        hostname: l.hostname || undefined,
      })),
    };
    mutation.mutate(payload);
  };

  if (isLoading) return <p className="text-muted-foreground">Loading gateway...</p>;
  if (!gw) return <p className="text-muted-foreground">Gateway not found.</p>;

  return (
    <div>
      <div className="mb-6">
        <Link to={`/gateways/${ns}/${name}`} className="text-sm text-blue-400 hover:underline">
          &larr; Back to {name}
        </Link>
      </div>

      <h1 className="text-2xl font-bold">Edit Gateway</h1>
      <p className="mt-1 text-muted-foreground">
        {ns}/{name}
      </p>

      {mutation.isError && (
        <div className="mt-4 rounded-md border border-red-500/30 bg-red-500/10 px-4 py-3 text-sm text-red-400">
          {String(mutation.error)}
        </div>
      )}

      <form onSubmit={handleSubmit(onSubmit)} className="mt-6 max-w-2xl space-y-6">
        {/* Name (read-only) */}
        <div>
          <label className="block text-sm font-medium">Name</label>
          <p className="mt-1 font-mono text-sm text-muted-foreground">{name}</p>
        </div>

        {/* Namespace (read-only) */}
        <div>
          <label className="block text-sm font-medium">Namespace</label>
          <p className="mt-1 font-mono text-sm text-muted-foreground">{ns}</p>
        </div>

        {/* Gateway Class */}
        <div>
          <label htmlFor="gatewayClassName" className="block text-sm font-medium">
            Gateway Class
          </label>
          <select
            id="gatewayClassName"
            {...register("gatewayClassName")}
            className="mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
          >
            <option value="">Select a gateway class...</option>
            {gatewayClasses?.map((gc) => (
              <option key={gc.name} value={gc.name}>
                {gc.name}
              </option>
            ))}
          </select>
          {errors.gatewayClassName && (
            <p className="mt-1 text-xs text-red-400">{errors.gatewayClassName.message}</p>
          )}
        </div>

        {/* Listeners */}
        <div>
          <div className="flex items-center justify-between">
            <label className="block text-sm font-medium">Listeners</label>
            <button
              type="button"
              onClick={() => append({ name: "", port: 80, protocol: "HTTP", hostname: "" })}
              className="text-sm text-blue-400 hover:underline"
            >
              + Add Listener
            </button>
          </div>
          {errors.listeners?.root && (
            <p className="mt-1 text-xs text-red-400">{errors.listeners.root.message}</p>
          )}

          <div className="mt-2 space-y-4">
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
                    <label className="block text-xs text-muted-foreground">Name</label>
                    <input
                      {...register(`listeners.${index}.name`)}
                      className="mt-1 w-full rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
                      placeholder="http"
                    />
                    {errors.listeners?.[index]?.name && (
                      <p className="mt-1 text-xs text-red-400">{errors.listeners[index].name.message}</p>
                    )}
                  </div>

                  <div>
                    <label className="block text-xs text-muted-foreground">Port</label>
                    <input
                      type="number"
                      {...register(`listeners.${index}.port`, { valueAsNumber: true })}
                      className="mt-1 w-full rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
                      placeholder="80"
                    />
                    {errors.listeners?.[index]?.port && (
                      <p className="mt-1 text-xs text-red-400">{errors.listeners[index].port.message}</p>
                    )}
                  </div>

                  <div>
                    <label className="block text-xs text-muted-foreground">Protocol</label>
                    <select
                      {...register(`listeners.${index}.protocol`)}
                      className="mt-1 w-full rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
                    >
                      <option value="HTTP">HTTP</option>
                      <option value="HTTPS">HTTPS</option>
                      <option value="TLS">TLS</option>
                      <option value="TCP">TCP</option>
                      <option value="UDP">UDP</option>
                    </select>
                  </div>

                  <div>
                    <label className="block text-xs text-muted-foreground">Hostname (optional)</label>
                    <input
                      {...register(`listeners.${index}.hostname`)}
                      className="mt-1 w-full rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
                      placeholder="example.com"
                    />
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>

        {/* Submit */}
        <div className="flex gap-3">
          <button
            type="submit"
            disabled={isSubmitting || mutation.isPending}
            className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {mutation.isPending ? "Saving..." : "Save Changes"}
          </button>
          <Link
            to={`/gateways/${ns}/${name}`}
            className="rounded-md border border-border px-4 py-2 text-sm font-medium text-foreground hover:bg-muted/30"
          >
            Cancel
          </Link>
        </div>
      </form>
    </div>
  );
}
