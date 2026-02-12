import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { runRouteCheck, type RouteCheckResponse, type RouteCheckResult } from "@/api/diagnostics";

const inputClass =
  "mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring";

function statusIcon(status: string): { icon: string; color: string } {
  switch (status) {
    case "pass":
      return { icon: "\u2713", color: "text-emerald-400" };
    case "fail":
      return { icon: "\u2717", color: "text-red-400" };
    case "warn":
      return { icon: "!", color: "text-yellow-400" };
    case "skip":
      return { icon: "-", color: "text-muted-foreground" };
    default:
      return { icon: "?", color: "text-muted-foreground" };
  }
}

function overallStatusBadge(status: string): string {
  switch (status) {
    case "pass":
      return "bg-emerald-500/10 text-emerald-400 border-emerald-500/30";
    case "fail":
      return "bg-red-500/10 text-red-400 border-red-500/30";
    case "warn":
      return "bg-yellow-500/10 text-yellow-400 border-yellow-500/30";
    default:
      return "bg-zinc-500/10 text-zinc-400 border-zinc-500/30";
  }
}

export default function RouteCheck() {
  const [namespace, setNamespace] = useState("default");
  const [routeName, setRouteName] = useState("");

  const mutation = useMutation({
    mutationFn: runRouteCheck,
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!routeName.trim()) return;
    mutation.mutate({ namespace, routeName });
  };

  const result: RouteCheckResponse | undefined = mutation.data;

  return (
    <div>
      <div className="mb-6">
        <Link
          to="/diagnostics"
          className="text-sm text-blue-400 hover:underline"
        >
          &larr; Back to Diagnostics
        </Link>
      </div>

      <h1 className="text-2xl font-bold">Route Diagnostic</h1>
      <p className="mt-1 text-muted-foreground">
        Check the configuration and health of an HTTPRoute resource.
      </p>

      <form
        onSubmit={handleSubmit}
        className="mt-6 max-w-lg rounded-lg border border-border p-4"
      >
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium">Namespace</label>
            <input
              value={namespace}
              onChange={(e) => setNamespace(e.target.value)}
              className={inputClass}
              placeholder="default"
            />
          </div>
          <div>
            <label className="block text-sm font-medium">Route Name</label>
            <input
              value={routeName}
              onChange={(e) => setRouteName(e.target.value)}
              className={inputClass}
              placeholder="my-http-route"
            />
          </div>
        </div>

        {mutation.isError && (
          <p className="mt-3 text-sm text-red-400">
            {String(mutation.error)}
          </p>
        )}

        <div className="mt-4">
          <button
            type="submit"
            disabled={mutation.isPending || !routeName.trim()}
            className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {mutation.isPending ? "Running..." : "Run Check"}
          </button>
        </div>
      </form>

      {/* Results */}
      {result && (
        <div className="mt-6">
          <div className="flex items-center gap-3">
            <h2 className="text-lg font-semibold">Results</h2>
            <span
              className={`inline-flex rounded-md border px-2 py-0.5 text-xs font-medium ${overallStatusBadge(result.status)}`}
            >
              {result.status}
            </span>
          </div>
          <p className="mt-1 text-sm text-muted-foreground">
            {result.route} in {result.namespace}
          </p>

          <div className="mt-4 space-y-3">
            {result.checks.map((check: RouteCheckResult, i: number) => {
              const { icon, color } = statusIcon(check.status);
              return (
                <div
                  key={i}
                  className="rounded-lg border border-border p-4"
                >
                  <div className="flex items-center gap-3">
                    <span
                      className={`flex h-6 w-6 items-center justify-center rounded-full text-sm font-bold ${color} ${
                        check.status === "pass"
                          ? "bg-emerald-500/10"
                          : check.status === "fail"
                            ? "bg-red-500/10"
                            : check.status === "warn"
                              ? "bg-yellow-500/10"
                              : "bg-zinc-500/10"
                      }`}
                    >
                      {icon}
                    </span>
                    <span className="text-sm font-medium text-foreground">
                      {check.name}
                    </span>
                  </div>
                  <p className="mt-2 text-sm text-muted-foreground">
                    {check.message}
                  </p>
                  {check.details && (
                    <pre className="mt-2 overflow-auto rounded bg-muted/30 p-3 text-xs text-muted-foreground">
                      {check.details}
                    </pre>
                  )}
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
