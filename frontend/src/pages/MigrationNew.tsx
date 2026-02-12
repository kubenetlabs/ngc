import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import {
  importResources,
  analyzeImport,
  generateResources,
  applyMigration,
  type ImportResponse,
  type AnalysisResponse,
  type AnalysisResource,
  type GenerateResponse,
  type GeneratedResource,
  type ApplyResponse,
  type ApplyResult,
} from "@/api/migration";

const STEPS = ["Import", "Analysis", "Generate", "Apply"] as const;

const inputClass =
  "mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-ring";
const selectClass = inputClass;

function confidenceColor(confidence: string): string {
  switch (confidence) {
    case "high":
      return "bg-emerald-500/10 text-emerald-400 border-emerald-500/30";
    case "medium":
      return "bg-yellow-500/10 text-yellow-400 border-yellow-500/30";
    case "low":
      return "bg-red-500/10 text-red-400 border-red-500/30";
    default:
      return "bg-zinc-500/10 text-zinc-400 border-zinc-500/30";
  }
}

function applyStatusColor(status: string): string {
  switch (status) {
    case "created":
      return "bg-emerald-500/10 text-emerald-400";
    case "updated":
      return "bg-blue-500/10 text-blue-400";
    case "failed":
      return "bg-red-500/10 text-red-400";
    default:
      return "bg-zinc-500/10 text-zinc-400";
  }
}

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

export default function MigrationNew() {
  const [step, setStep] = useState(0);
  const [yamlContent, setYamlContent] = useState("");
  const [format, setFormat] = useState<
    "nginx-conf" | "ingress-yaml" | "virtualserver-yaml"
  >("ingress-yaml");

  const [importResult, setImportResult] = useState<ImportResponse | null>(null);
  const [analysisResult, setAnalysisResult] =
    useState<AnalysisResponse | null>(null);
  const [generateResult, setGenerateResult] =
    useState<GenerateResponse | null>(null);
  const [applyResult, setApplyResult] = useState<ApplyResponse | null>(null);

  const importMutation = useMutation({
    mutationFn: importResources,
    onSuccess: (data) => {
      setImportResult(data);
      setStep(1);
    },
  });

  const analysisMutation = useMutation({
    mutationFn: analyzeImport,
    onSuccess: (data) => {
      setAnalysisResult(data);
      setStep(2);
    },
  });

  const generateMutation = useMutation({
    mutationFn: generateResources,
    onSuccess: (data) => {
      setGenerateResult(data);
      setStep(3);
    },
  });

  const applyMutation = useMutation({
    mutationFn: applyMigration,
    onSuccess: (data) => {
      setApplyResult(data);
    },
  });

  const handleImport = (e: React.FormEvent) => {
    e.preventDefault();
    if (!yamlContent.trim()) return;
    importMutation.mutate({ content: yamlContent, format });
  };

  const handleAnalyze = () => {
    if (!importResult) return;
    analysisMutation.mutate({ importId: importResult.importId });
  };

  const handleGenerate = () => {
    if (!analysisResult) return;
    generateMutation.mutate({ analysisId: analysisResult.analysisId });
  };

  const handleApply = () => {
    if (!generateResult) return;
    applyMutation.mutate({ generateId: generateResult.generateId, dryRun: false });
  };

  return (
    <div>
      <div className="mb-6">
        <Link
          to="/migration"
          className="text-sm text-blue-400 hover:underline"
        >
          &larr; Back to Migration
        </Link>
      </div>

      <h1 className="text-2xl font-bold">New Migration</h1>
      <p className="mt-1 text-muted-foreground">
        Import and migrate NGINX Ingress Controller resources.
      </p>

      <div className="mt-6 max-w-3xl">
        <StepIndicator currentStep={step} />

        {/* Step 0: Import */}
        {step === 0 && (
          <form onSubmit={handleImport}>
            <div>
              <h2 className="text-lg font-semibold">Import Configuration</h2>
              <p className="mt-1 text-sm text-muted-foreground">
                Paste your NGINX configuration, Ingress YAML, or VirtualServer
                YAML below.
              </p>
            </div>

            <div className="mt-4">
              <label className="block text-sm font-medium">Format</label>
              <select
                value={format}
                onChange={(e) =>
                  setFormat(
                    e.target.value as
                      | "nginx-conf"
                      | "ingress-yaml"
                      | "virtualserver-yaml",
                  )
                }
                className={selectClass}
              >
                <option value="nginx-conf">NGINX Configuration</option>
                <option value="ingress-yaml">Ingress YAML</option>
                <option value="virtualserver-yaml">VirtualServer YAML</option>
              </select>
            </div>

            <div className="mt-4">
              <label className="block text-sm font-medium">Content</label>
              <textarea
                value={yamlContent}
                onChange={(e) => setYamlContent(e.target.value)}
                rows={15}
                className={`${inputClass} font-mono`}
                placeholder="Paste your YAML or configuration here..."
              />
            </div>

            {importMutation.isError && (
              <p className="mt-3 text-sm text-red-400">
                {String(importMutation.error)}
              </p>
            )}

            <div className="mt-4">
              <button
                type="submit"
                disabled={importMutation.isPending || !yamlContent.trim()}
                className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
              >
                {importMutation.isPending ? "Importing..." : "Import"}
              </button>
            </div>
          </form>
        )}

        {/* Step 1: Analysis */}
        {step === 1 && (
          <div>
            <h2 className="text-lg font-semibold">Analysis</h2>
            <p className="mt-1 text-sm text-muted-foreground">
              Imported {importResult?.resourceCount} resources in{" "}
              {importResult?.format} format. Click "Analyze" to assess migration
              compatibility.
            </p>

            {analysisResult && (
              <div className="mt-4">
                {analysisResult.warnings.length > 0 && (
                  <div className="mb-4 rounded-md border border-yellow-500/30 bg-yellow-500/10 p-3">
                    <p className="text-sm font-medium text-yellow-400">
                      Warnings
                    </p>
                    <ul className="mt-1 space-y-1">
                      {analysisResult.warnings.map((w, i) => (
                        <li key={i} className="text-sm text-muted-foreground">
                          - {w}
                        </li>
                      ))}
                    </ul>
                  </div>
                )}

                {analysisResult.errors.length > 0 && (
                  <div className="mb-4 rounded-md border border-red-500/30 bg-red-500/10 p-3">
                    <p className="text-sm font-medium text-red-400">Errors</p>
                    <ul className="mt-1 space-y-1">
                      {analysisResult.errors.map((e, i) => (
                        <li key={i} className="text-sm text-muted-foreground">
                          - {e}
                        </li>
                      ))}
                    </ul>
                  </div>
                )}

                <div className="overflow-x-auto rounded-lg border border-border">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b border-border bg-muted/30">
                        <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                          Kind
                        </th>
                        <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                          Name
                        </th>
                        <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                          Namespace
                        </th>
                        <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                          Confidence
                        </th>
                        <th className="px-4 py-3 text-left font-medium text-muted-foreground">
                          Notes
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      {analysisResult.resources.map(
                        (r: AnalysisResource, i: number) => (
                          <tr
                            key={i}
                            className="border-b border-border last:border-0 hover:bg-muted/20"
                          >
                            <td className="px-4 py-3 font-medium">{r.kind}</td>
                            <td className="px-4 py-3 text-muted-foreground">
                              {r.name}
                            </td>
                            <td className="px-4 py-3 text-muted-foreground">
                              {r.namespace}
                            </td>
                            <td className="px-4 py-3">
                              <span
                                className={`inline-flex rounded-md border px-2 py-0.5 text-xs font-medium ${confidenceColor(r.confidence)}`}
                              >
                                {r.confidence}
                              </span>
                            </td>
                            <td className="px-4 py-3 text-xs text-muted-foreground">
                              {r.notes.join("; ")}
                            </td>
                          </tr>
                        ),
                      )}
                    </tbody>
                  </table>
                </div>
              </div>
            )}

            {analysisMutation.isError && (
              <p className="mt-3 text-sm text-red-400">
                {String(analysisMutation.error)}
              </p>
            )}

            <div className="mt-6 flex items-center gap-3">
              <button
                onClick={() => setStep(0)}
                className="rounded-md border border-border px-4 py-2 text-sm font-medium text-foreground hover:bg-muted/30"
              >
                Back
              </button>
              {!analysisResult && (
                <button
                  onClick={handleAnalyze}
                  disabled={analysisMutation.isPending}
                  className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
                >
                  {analysisMutation.isPending ? "Analyzing..." : "Analyze"}
                </button>
              )}
              {analysisResult && (
                <button
                  onClick={handleGenerate}
                  disabled={generateMutation.isPending}
                  className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
                >
                  {generateMutation.isPending
                    ? "Generating..."
                    : "Generate Resources"}
                </button>
              )}
            </div>
          </div>
        )}

        {/* Step 2: Generate */}
        {step === 2 && (
          <div>
            <h2 className="text-lg font-semibold">Generated Resources</h2>
            <p className="mt-1 text-sm text-muted-foreground">
              Review the generated Gateway API resources before applying.
            </p>

            {generateResult && (
              <div className="mt-4 space-y-4">
                {generateResult.resources.map(
                  (r: GeneratedResource, i: number) => (
                    <div
                      key={i}
                      className="rounded-lg border border-border p-4"
                    >
                      <div className="flex items-center gap-2">
                        <span className="rounded bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">
                          {r.kind}
                        </span>
                        <span className="text-sm font-medium text-foreground">
                          {r.name}
                        </span>
                        <span className="text-xs text-muted-foreground">
                          ({r.namespace})
                        </span>
                      </div>
                      <pre className="mt-3 max-h-60 overflow-auto rounded bg-muted/30 p-3 text-xs text-foreground">
                        {r.yaml}
                      </pre>
                    </div>
                  ),
                )}
              </div>
            )}

            {generateMutation.isError && (
              <p className="mt-3 text-sm text-red-400">
                {String(generateMutation.error)}
              </p>
            )}

            <div className="mt-6 flex items-center gap-3">
              <button
                onClick={() => setStep(1)}
                className="rounded-md border border-border px-4 py-2 text-sm font-medium text-foreground hover:bg-muted/30"
              >
                Back
              </button>
              {generateResult && (
                <button
                  onClick={handleApply}
                  disabled={applyMutation.isPending}
                  className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
                >
                  {applyMutation.isPending ? "Applying..." : "Apply to Cluster"}
                </button>
              )}
            </div>
          </div>
        )}

        {/* Step 3: Apply */}
        {step === 3 && (
          <div>
            <h2 className="text-lg font-semibold">Apply Results</h2>
            <p className="mt-1 text-sm text-muted-foreground">
              Migration resources applied to the cluster.
            </p>

            {applyResult && (
              <div className="mt-4">
                <div className="mb-4 grid gap-4 sm:grid-cols-2">
                  <div className="rounded-lg border border-emerald-500/30 bg-emerald-500/10 p-4">
                    <p className="text-sm text-emerald-400">Succeeded</p>
                    <p className="mt-1 text-2xl font-bold text-emerald-400">
                      {applyResult.successCount}
                    </p>
                  </div>
                  <div className="rounded-lg border border-red-500/30 bg-red-500/10 p-4">
                    <p className="text-sm text-red-400">Failed</p>
                    <p className="mt-1 text-2xl font-bold text-red-400">
                      {applyResult.failureCount}
                    </p>
                  </div>
                </div>

                <div className="space-y-2">
                  {applyResult.results.map(
                    (r: ApplyResult, i: number) => (
                      <div
                        key={i}
                        className="flex items-center justify-between rounded-lg border border-border px-4 py-3"
                      >
                        <span className="text-sm font-medium text-foreground">
                          {r.resource}
                        </span>
                        <div className="flex items-center gap-3">
                          <span
                            className={`inline-flex rounded px-2 py-0.5 text-xs font-medium ${applyStatusColor(r.status)}`}
                          >
                            {r.status}
                          </span>
                          {r.message && (
                            <span className="text-xs text-muted-foreground">
                              {r.message}
                            </span>
                          )}
                        </div>
                      </div>
                    ),
                  )}
                </div>
              </div>
            )}

            {!applyResult && (
              <div className="mt-4">
                <p className="text-sm text-muted-foreground">
                  Click "Apply to Cluster" to apply the generated resources.
                </p>
                <div className="mt-4 flex items-center gap-3">
                  <button
                    onClick={() => setStep(2)}
                    className="rounded-md border border-border px-4 py-2 text-sm font-medium text-foreground hover:bg-muted/30"
                  >
                    Back
                  </button>
                  <button
                    onClick={handleApply}
                    disabled={applyMutation.isPending}
                    className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
                  >
                    {applyMutation.isPending
                      ? "Applying..."
                      : "Apply to Cluster"}
                  </button>
                </div>
              </div>
            )}

            {applyMutation.isError && (
              <p className="mt-3 text-sm text-red-400">
                {String(applyMutation.error)}
              </p>
            )}

            {applyResult && (
              <div className="mt-6">
                <Link
                  to="/migration"
                  className="rounded-md border border-border px-4 py-2 text-sm font-medium text-foreground hover:bg-muted/30"
                >
                  Back to Migrations
                </Link>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
