import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import InferenceOverview from "../InferenceOverview";

// Mock APIs
const mockFetchInferencePools = vi.fn();
const mockFetchInferenceMetricsSummary = vi.fn();
const mockFetchEPPDecisions = vi.fn();
const mockFetchInferenceStacks = vi.fn();

vi.mock("@/api/inference", () => ({
  fetchInferencePools: (...args: unknown[]) => mockFetchInferencePools(...args),
  fetchInferenceMetricsSummary: (...args: unknown[]) => mockFetchInferenceMetricsSummary(...args),
  fetchEPPDecisions: (...args: unknown[]) => mockFetchEPPDecisions(...args),
}));
vi.mock("@/api/inferencestacks", () => ({
  fetchInferenceStacks: (...args: unknown[]) => mockFetchInferenceStacks(...args),
}));

vi.mock("@/hooks/useActiveCluster", () => ({
  useActiveCluster: () => "test-cluster",
}));

// Mock MetricCard so we can assert text
vi.mock("@/components/inference/MetricCard", () => ({
  MetricCard: ({ title, value }: { title: string; value: string | number }) => (
    <div data-testid={`metric-${title}`}>
      <span>{title}</span>
      <span>{value}</span>
    </div>
  ),
}));

// Mock GPUUtilizationBar
vi.mock("@/components/inference/GPUUtilizationBar", () => ({
  GPUUtilizationBar: () => <div data-testid="gpu-bar">GPU Bar</div>,
}));

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return (
      <QueryClientProvider client={queryClient}>
        <MemoryRouter>{children}</MemoryRouter>
      </QueryClientProvider>
    );
  };
}

describe("InferenceOverview", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockFetchInferencePools.mockResolvedValue([]);
    mockFetchInferenceMetricsSummary.mockResolvedValue({
      avgTTFT: 0,
      p95TTFT: 0,
      p99TTFT: 0,
      avgTPS: 0,
      totalTokens: 0,
      avgQueueDepth: 0,
      avgKVCachePct: 0,
      prefixCacheHitRate: 0,
      avgGPUUtil: 0,
    });
    mockFetchEPPDecisions.mockResolvedValue([]);
    mockFetchInferenceStacks.mockResolvedValue([]);
  });

  it("renders Inference heading", async () => {
    render(<InferenceOverview />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText("Inference")).toBeInTheDocument();
    });
  });

  it("renders View Pools link", async () => {
    render(<InferenceOverview />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByRole("link", { name: "View Pools" })).toBeInTheDocument();
    });
  });

  it("shows metric cards", async () => {
    mockFetchInferencePools.mockResolvedValue([
      {
        name: "llama3-70b",
        namespace: "default",
        modelName: "meta/llama3-70b",
        gpuType: "H100",
        gpuCount: 8,
        replicas: 4,
        readyReplicas: 4,
        avgGpuUtil: 72,
        status: { conditions: [{ status: "Ready" }] },
      },
    ]);
    mockFetchInferenceStacks.mockResolvedValue([
      { name: "stack-1", phase: "Ready" },
    ]);

    render(<InferenceOverview />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByTestId("metric-Total Pools")).toBeInTheDocument();
      expect(screen.getByTestId("metric-Total GPUs")).toBeInTheDocument();
      expect(screen.getByTestId("metric-Stacks")).toBeInTheDocument();
    });
  });

  it("shows pool data when loaded", async () => {
    mockFetchInferencePools.mockResolvedValue([
      {
        name: "llama3-70b",
        namespace: "default",
        modelName: "meta-llama/Llama-3-70B",
        gpuType: "H100",
        gpuCount: 8,
        replicas: 4,
        readyReplicas: 4,
        avgGpuUtil: 72,
        status: { conditions: [{ status: "Ready" }] },
      },
    ]);

    render(<InferenceOverview />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText("llama3-70b")).toBeInTheDocument();
      expect(screen.getByText("Llama-3-70B")).toBeInTheDocument();
    });
  });
});
