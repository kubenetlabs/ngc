import { Link } from "react-router-dom";

const STEPS = [
  {
    step: 1,
    title: "Import",
    description:
      "Upload your existing NGINX configuration, Ingress resources, or VirtualServer definitions.",
  },
  {
    step: 2,
    title: "Analyze",
    description:
      "Automated analysis identifies resource mappings, confidence levels, and potential issues.",
  },
  {
    step: 3,
    title: "Generate",
    description:
      "Generate equivalent Gateway API resources (Gateways, HTTPRoutes, policies) from your imports.",
  },
  {
    step: 4,
    title: "Apply",
    description:
      "Apply the generated resources to your cluster with dry-run validation before committing.",
  },
];

export default function MigrationList() {
  return (
    <div>
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Migration Projects</h1>
          <p className="mt-1 text-muted-foreground">
            NGINX Ingress Controller migration tool.
          </p>
        </div>
        <Link
          to="/migration/new"
          className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
        >
          New Migration
        </Link>
      </div>

      {/* Workflow overview */}
      <div className="mt-8">
        <h2 className="text-lg font-semibold">Migration Workflow</h2>
        <p className="mt-1 text-sm text-muted-foreground">
          Migrate from NGINX Ingress Controller to NGINX Gateway Fabric in four
          steps.
        </p>

        <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {STEPS.map((s) => (
            <div key={s.step} className="rounded-lg border border-border p-4">
              <div className="flex h-8 w-8 items-center justify-center rounded-full bg-blue-600/20 text-sm font-bold text-blue-400">
                {s.step}
              </div>
              <h3 className="mt-3 text-sm font-semibold text-foreground">
                {s.title}
              </h3>
              <p className="mt-1 text-xs text-muted-foreground">
                {s.description}
              </p>
            </div>
          ))}
        </div>
      </div>

      {/* Empty state */}
      <div className="mt-8 rounded-lg border border-border p-8 text-center">
        <p className="text-lg font-medium text-foreground">
          No migration projects yet
        </p>
        <p className="mt-2 text-sm text-muted-foreground">
          Start a new migration to convert your NGINX Ingress Controller
          configuration to Gateway API resources.
        </p>
        <div className="mt-4">
          <Link
            to="/migration/new"
            className="inline-block rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
          >
            Start Migration
          </Link>
        </div>
      </div>
    </div>
  );
}
