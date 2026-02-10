import { useNavigate, useParams, Link } from "react-router-dom";
import { useForm, useFieldArray } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { fetchHTTPRoute, updateHTTPRoute } from "@/api/routes";
import { fetchGateways } from "@/api/gateways";
import { httpRouteRuleSchema, parentRefSchema, type UpdateHTTPRoutePayload } from "@/types/route";
import { useActiveCluster } from "@/hooks/useActiveCluster";
import { z } from "zod";

const updateHTTPRouteSchema = z.object({
  parentRefs: z.array(parentRefSchema).min(1, "At least one parent gateway is required"),
  hostnames: z.string().optional(),
  rules: z.array(httpRouteRuleSchema).min(1, "At least one rule is required"),
});

type UpdateHTTPRouteFormData = z.infer<typeof updateHTTPRouteSchema>;

export default function RouteEdit() {
  const { ns, name } = useParams<{ ns: string; name: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const activeCluster = useActiveCluster();

  const { data: route, isLoading } = useQuery({
    queryKey: ["httproute", activeCluster, ns, name],
    queryFn: () => fetchHTTPRoute(ns!, name!),
    enabled: !!ns && !!name,
  });

  const { data: gateways } = useQuery({
    queryKey: ["gateways", activeCluster],
    queryFn: () => fetchGateways(),
  });

  const {
    register,
    control,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<UpdateHTTPRouteFormData>({
    resolver: zodResolver(updateHTTPRouteSchema),
    values: route
      ? {
          parentRefs: route.parentRefs.map((p) => ({
            name: p.name,
            namespace: p.namespace ?? "",
            sectionName: p.sectionName ?? "",
          })),
          hostnames: route.hostnames?.join(", ") ?? "",
          rules: route.rules.map((rule) => ({
            matches: rule.matches?.map((m) => ({
              path: m.path ? { type: m.path.type, value: m.path.value } : undefined,
              method: m.method ?? "",
            })) ?? [{ path: { type: "PathPrefix" as const, value: "/" }, method: "" }],
            backendRefs: rule.backendRefs?.map((br) => ({
              name: br.name,
              namespace: br.namespace ?? "",
              port: br.port,
              weight: br.weight,
            })) ?? [{ name: "", port: 80, weight: 1, namespace: "" }],
          })),
        }
      : undefined,
  });

  const { fields: parentRefFields, append: appendParentRef, remove: removeParentRef } = useFieldArray({ control, name: "parentRefs" });
  const { fields: ruleFields, append: appendRule, remove: removeRule } = useFieldArray({ control, name: "rules" });

  const mutation = useMutation({
    mutationFn: (payload: UpdateHTTPRoutePayload) => updateHTTPRoute(ns!, name!, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["httproutes"] });
      queryClient.invalidateQueries({ queryKey: ["httproute", activeCluster, ns, name] });
      navigate(`/routes/${ns}/${name}`);
    },
  });

  const onSubmit = (data: UpdateHTTPRouteFormData) => {
    const payload: UpdateHTTPRoutePayload = {
      parentRefs: data.parentRefs.map((p) => ({
        name: p.name,
        namespace: p.namespace || undefined,
        sectionName: p.sectionName || undefined,
      })),
      hostnames: data.hostnames ? data.hostnames.split(",").map((h) => h.trim()).filter(Boolean) : undefined,
      rules: data.rules.map((rule) => ({
        matches: rule.matches
          ?.filter((m) => m.path?.value)
          .map((m) => ({
            path: m.path ? { type: m.path.type, value: m.path.value } : undefined,
            method: m.method || undefined,
          })),
        backendRefs: rule.backendRefs.map((br) => ({
          name: br.name,
          namespace: br.namespace || undefined,
          port: br.port,
          weight: br.weight,
        })),
      })),
    };
    mutation.mutate(payload);
  };

  if (isLoading) return <p className="text-muted-foreground">Loading route...</p>;
  if (!route) return <p className="text-muted-foreground">Route not found.</p>;

  return (
    <div>
      <div className="mb-6">
        <Link to={`/routes/${ns}/${name}`} className="text-sm text-blue-400 hover:underline">
          &larr; Back to {name}
        </Link>
      </div>

      <h1 className="text-2xl font-bold">Edit HTTPRoute</h1>
      <p className="mt-1 text-muted-foreground">{ns}/{name}</p>

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

        {/* Parent Refs */}
        <div>
          <div className="flex items-center justify-between">
            <label className="block text-sm font-medium">Parent Gateways</label>
            <button
              type="button"
              onClick={() => appendParentRef({ name: "", namespace: "", sectionName: "" })}
              className="text-sm text-blue-400 hover:underline"
            >
              + Add Gateway
            </button>
          </div>
          {errors.parentRefs?.root && (
            <p className="mt-1 text-xs text-red-400">{errors.parentRefs.root.message}</p>
          )}
          <div className="mt-2 space-y-3">
            {parentRefFields.map((field, index) => (
              <div key={field.id} className="flex items-start gap-3 rounded-lg border border-border p-3">
                <div className="flex-1">
                  <label className="block text-xs text-muted-foreground">Gateway</label>
                  <select
                    {...register(`parentRefs.${index}.name`)}
                    className="mt-1 w-full rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
                  >
                    <option value="">Select a gateway...</option>
                    {gateways?.map((gw) => (
                      <option key={`${gw.namespace}/${gw.name}`} value={gw.name}>
                        {gw.namespace}/{gw.name}
                      </option>
                    ))}
                  </select>
                  {errors.parentRefs?.[index]?.name && (
                    <p className="mt-1 text-xs text-red-400">{errors.parentRefs[index].name.message}</p>
                  )}
                </div>
                <div className="w-32">
                  <label className="block text-xs text-muted-foreground">Section (opt)</label>
                  <input
                    {...register(`parentRefs.${index}.sectionName`)}
                    className="mt-1 w-full rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
                    placeholder="http"
                  />
                </div>
                {parentRefFields.length > 1 && (
                  <button type="button" onClick={() => removeParentRef(index)} className="mt-5 text-sm text-red-400 hover:underline">
                    Remove
                  </button>
                )}
              </div>
            ))}
          </div>
        </div>

        {/* Hostnames */}
        <div>
          <label htmlFor="hostnames" className="block text-sm font-medium">Hostnames (optional, comma-separated)</label>
          <input
            id="hostnames"
            {...register("hostnames")}
            className="mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
            placeholder="example.com, api.example.com"
          />
        </div>

        {/* Rules */}
        <div>
          <div className="flex items-center justify-between">
            <label className="block text-sm font-medium">Rules</label>
            <button
              type="button"
              onClick={() =>
                appendRule({
                  matches: [{ path: { type: "PathPrefix", value: "/" }, method: "" }],
                  backendRefs: [{ name: "", port: 80, weight: 1, namespace: "" }],
                })
              }
              className="text-sm text-blue-400 hover:underline"
            >
              + Add Rule
            </button>
          </div>
          {errors.rules?.root && (
            <p className="mt-1 text-xs text-red-400">{errors.rules.root.message}</p>
          )}

          <div className="mt-2 space-y-4">
            {ruleFields.map((ruleField, ruleIndex) => (
              <RuleEditor
                key={ruleField.id}
                index={ruleIndex}
                control={control}
                register={register}
                errors={errors}
                canRemove={ruleFields.length > 1}
                onRemove={() => removeRule(ruleIndex)}
              />
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
            to={`/routes/${ns}/${name}`}
            className="rounded-md border border-border px-4 py-2 text-sm font-medium text-foreground hover:bg-muted/30"
          >
            Cancel
          </Link>
        </div>
      </form>
    </div>
  );
}

function RuleEditor({
  index,
  control,
  register,
  errors,
  canRemove,
  onRemove,
}: {
  index: number;
  control: any;
  register: any;
  errors: any;
  canRemove: boolean;
  onRemove: () => void;
}) {
  const { fields: backendRefFields, append: appendBackendRef, remove: removeBackendRef } = useFieldArray({
    control,
    name: `rules.${index}.backendRefs`,
  });

  return (
    <div className="rounded-lg border border-border p-4">
      <div className="flex items-center justify-between">
        <span className="text-sm font-medium text-muted-foreground">Rule {index + 1}</span>
        {canRemove && (
          <button type="button" onClick={onRemove} className="text-sm text-red-400 hover:underline">
            Remove Rule
          </button>
        )}
      </div>

      {/* Match: Path */}
      <div className="mt-3">
        <label className="block text-xs font-medium text-muted-foreground">Path Match</label>
        <div className="mt-1 flex gap-2">
          <select
            {...register(`rules.${index}.matches.0.path.type`)}
            className="w-40 rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
          >
            <option value="PathPrefix">PathPrefix</option>
            <option value="Exact">Exact</option>
            <option value="RegularExpression">RegularExpression</option>
          </select>
          <input
            {...register(`rules.${index}.matches.0.path.value`)}
            className="flex-1 rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
            placeholder="/"
          />
        </div>
      </div>

      {/* Match: Method */}
      <div className="mt-3">
        <label className="block text-xs font-medium text-muted-foreground">Method (optional)</label>
        <select
          {...register(`rules.${index}.matches.0.method`)}
          className="mt-1 w-40 rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
        >
          <option value="">Any</option>
          <option value="GET">GET</option>
          <option value="POST">POST</option>
          <option value="PUT">PUT</option>
          <option value="DELETE">DELETE</option>
          <option value="PATCH">PATCH</option>
          <option value="HEAD">HEAD</option>
          <option value="OPTIONS">OPTIONS</option>
        </select>
      </div>

      {/* Backend Refs */}
      <div className="mt-4">
        <div className="flex items-center justify-between">
          <label className="block text-xs font-medium text-muted-foreground">Backend Services</label>
          <button
            type="button"
            onClick={() => appendBackendRef({ name: "", port: 80, weight: 1, namespace: "" })}
            className="text-xs text-blue-400 hover:underline"
          >
            + Add Backend
          </button>
        </div>
        {errors.rules?.[index]?.backendRefs?.root && (
          <p className="mt-1 text-xs text-red-400">{errors.rules[index].backendRefs.root.message}</p>
        )}

        <div className="mt-2 space-y-2">
          {backendRefFields.map((brField, brIndex) => (
            <div key={brField.id} className="flex items-end gap-2">
              <div className="flex-1">
                <label className="block text-xs text-muted-foreground">Service Name</label>
                <input
                  {...register(`rules.${index}.backendRefs.${brIndex}.name`)}
                  className="mt-1 w-full rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
                  placeholder="my-service"
                />
                {errors.rules?.[index]?.backendRefs?.[brIndex]?.name && (
                  <p className="mt-1 text-xs text-red-400">{errors.rules[index].backendRefs[brIndex].name.message}</p>
                )}
              </div>
              <div className="w-24">
                <label className="block text-xs text-muted-foreground">Port</label>
                <input
                  type="number"
                  {...register(`rules.${index}.backendRefs.${brIndex}.port`, { valueAsNumber: true })}
                  className="mt-1 w-full rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
                  placeholder="80"
                />
              </div>
              <div className="w-20">
                <label className="block text-xs text-muted-foreground">Weight</label>
                <input
                  type="number"
                  {...register(`rules.${index}.backendRefs.${brIndex}.weight`, { valueAsNumber: true })}
                  className="mt-1 w-full rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring"
                  placeholder="1"
                />
              </div>
              {backendRefFields.length > 1 && (
                <button type="button" onClick={() => removeBackendRef(brIndex)} className="mb-1 text-sm text-red-400 hover:underline">
                  Remove
                </button>
              )}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
