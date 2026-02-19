import { render, screen, waitFor, act } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import { EPPDecisionVisualizer } from "../EPPDecisionVisualizer";

// Mock APIs
const mockFetchPodMetrics = vi.fn();
const mockFetchEPPDecisions = vi.fn();

vi.mock("@/api/inference", () => ({
  fetchPodMetrics: (...args: unknown[]) => mockFetchPodMetrics(...args),
  fetchEPPDecisions: (...args: unknown[]) => mockFetchEPPDecisions(...args),
}));

vi.mock("@/hooks/useActiveCluster", () => ({
  useActiveCluster: () => "test-cluster",
}));

// Capture the useWebSocket onMessage callback so we can trigger it in tests
let capturedOnMessage: ((msg: unknown) => void) | null = null;
const mockUseWebSocket = vi.fn().mockImplementation((opts: { onMessage?: (msg: unknown) => void }) => {
  capturedOnMessage = opts.onMessage ?? null;
  return { connected: true, send: vi.fn() };
});

vi.mock("@/hooks/useWebSocket", () => ({
  useWebSocket: (opts: { onMessage?: (msg: unknown) => void }) => mockUseWebSocket(opts),
}));

// Mock PodCard to simplify assertions
vi.mock("../PodCard", () => ({
  PodCard: ({ podName, highlighted }: { podName: string; highlighted: boolean }) => (
    <div data-testid={`pod-${podName}`} data-highlighted={highlighted}>
      {podName}
    </div>
  ),
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

describe("EPPDecisionVisualizer", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    capturedOnMessage = null;
    mockFetchPodMetrics.mockResolvedValue([]);
    mockFetchEPPDecisions.mockResolvedValue([]);
  });

  it("renders Live EPP Routing heading", async () => {
    render(<EPPDecisionVisualizer pool="test-pool" />, {
      wrapper: createWrapper(),
    });

    expect(screen.getByText("Live EPP Routing")).toBeInTheDocument();
  });

  it("shows connected status when WebSocket is connected", () => {
    render(<EPPDecisionVisualizer pool="test-pool" />, {
      wrapper: createWrapper(),
    });

    expect(screen.getByText("Live")).toBeInTheDocument();
  });

  it("shows Connecting status when WebSocket is disconnected", () => {
    mockUseWebSocket.mockImplementation((opts: { onMessage?: (msg: unknown) => void }) => {
      capturedOnMessage = opts.onMessage ?? null;
      return { connected: false, send: vi.fn() };
    });

    render(<EPPDecisionVisualizer pool="test-pool" />, {
      wrapper: createWrapper(),
    });

    expect(screen.getByText("Connecting...")).toBeInTheDocument();
  });

  it("renders pod cards when pod data loads", async () => {
    mockFetchPodMetrics.mockResolvedValue([
      {
        podName: "pod-a",
        gpuUtilPct: 85,
        kvCacheUtilPct: 45,
        queueDepth: 3,
        requestsInFlight: 2,
      },
      {
        podName: "pod-b",
        gpuUtilPct: 60,
        kvCacheUtilPct: 30,
        queueDepth: 1,
        requestsInFlight: 1,
      },
    ]);

    render(<EPPDecisionVisualizer pool="test-pool" />, {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(screen.getByTestId("pod-pod-a")).toBeInTheDocument();
      expect(screen.getByTestId("pod-pod-b")).toBeInTheDocument();
    });
  });

  it("shows Recent Decisions section", async () => {
    mockFetchEPPDecisions.mockResolvedValue([
      {
        requestId: "req-001",
        selectedPod: "pod-a",
        reason: "least-load",
        queueDepth: 3,
        kvCachePct: 0.45,
        timestamp: new Date().toISOString(),
      },
    ]);

    render(<EPPDecisionVisualizer pool="test-pool" />, {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(screen.getByText("Recent Decisions")).toBeInTheDocument();
    });
  });

  it("updates decisions when WebSocket message arrives", async () => {
    mockFetchPodMetrics.mockResolvedValue([
      {
        podName: "pod-a",
        gpuUtilPct: 85,
        kvCacheUtilPct: 45,
        queueDepth: 3,
        requestsInFlight: 2,
      },
    ]);
    mockFetchEPPDecisions.mockResolvedValue([]);

    render(<EPPDecisionVisualizer pool="test-pool" />, {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(screen.getByTestId("pod-pod-a")).toBeInTheDocument();
    });

    // Simulate WebSocket message
    act(() => {
      capturedOnMessage?.({
        topic: "epp-decisions",
        data: {
          requestId: "req-ws-001",
          selectedPod: "pod-a",
          reason: "prefix-cache",
          queueDepth: 2,
          kvCachePct: 0.5,
          timestamp: new Date().toISOString(),
        },
      });
    });

    // The pod should be highlighted after the WS message
    await waitFor(() => {
      const podEl = screen.getByTestId("pod-pod-a");
      expect(podEl.getAttribute("data-highlighted")).toBe("true");
    });
  });
});
