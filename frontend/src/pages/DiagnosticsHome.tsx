import { Link } from "react-router-dom";

interface DiagnosticCard {
  title: string;
  description: string;
  link: string;
  icon: string;
}

const diagnosticCards: DiagnosticCard[] = [
  {
    title: "Route Diagnostic",
    description:
      "Check the health and configuration of HTTPRoutes. Identify misconfigurations, missing backend references, and policy conflicts.",
    link: "/diagnostics/route-check",
    icon: "\u{1F6E4}",
  },
  {
    title: "Inference Diagnostics",
    description:
      "Monitor inference pool health, GPU utilization, EPP decisions, and model serving performance metrics.",
    link: "/inference",
    icon: "\u{1F9E0}",
  },
  {
    title: "Log Explorer",
    description:
      "Search and filter access logs, error logs, and audit trails. Correlate events across gateways and routes.",
    link: "/observability/logs",
    icon: "\u{1F4CB}",
  },
];

export default function DiagnosticsHome() {
  return (
    <div>
      <div>
        <h1 className="text-2xl font-bold">Diagnostics</h1>
        <p className="mt-1 text-muted-foreground">
          Troubleshooting tools and diagnostic wizards.
        </p>
      </div>

      <div className="mt-6 grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
        {diagnosticCards.map((card) => (
          <div
            key={card.title}
            className="flex flex-col rounded-lg border border-border p-6"
          >
            <div className="text-3xl">{card.icon}</div>
            <h2 className="mt-4 text-lg font-semibold text-foreground">
              {card.title}
            </h2>
            <p className="mt-2 flex-1 text-sm text-muted-foreground">
              {card.description}
            </p>
            <div className="mt-4">
              <Link
                to={card.link}
                className="inline-block rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700"
              >
                Launch
              </Link>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
