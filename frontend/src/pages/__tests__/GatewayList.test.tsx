import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import GatewayList from "../GatewayList";

// Mock API
const mockFetchGateways = vi.fn();
vi.mock("@/api/gateways", () => ({
  fetchGateways: (...args: unknown[]) => mockFetchGateways(...args),
}));

// Mock useActiveCluster to return a single cluster (not ALL)
vi.mock("@/hooks/useActiveCluster", () => ({
  useActiveCluster: () => "test-cluster",
}));

vi.mock("@/store/clusterStore", () => ({
  ALL_CLUSTERS: "all",
  useClusterStore: () => "test-cluster",
}));

// Mock GlobalGatewayList since it's used when ALL_CLUSTERS is selected
vi.mock("@/components/global/GlobalGatewayList", () => ({
  GlobalGatewayList: () => <div>Global Gateway List</div>,
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

describe("GatewayList", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders loading state initially", () => {
    mockFetchGateways.mockReturnValue(new Promise(() => {})); // never resolves
    render(<GatewayList />, { wrapper: createWrapper() });

    expect(screen.getByText("Loading gateways...")).toBeInTheDocument();
  });

  it("renders gateway list when data loads", async () => {
    mockFetchGateways.mockResolvedValue([
      {
        name: "my-gateway",
        namespace: "default",
        gatewayClassName: "nginx",
        listeners: [{ name: "http", port: 80, protocol: "HTTP" }],
        status: {
          conditions: [{ type: "Accepted", status: "True", reason: "Accepted" }],
          listeners: [{ name: "http", attachedRoutes: 3 }],
        },
        createdAt: new Date().toISOString(),
      },
    ]);

    render(<GatewayList />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText("my-gateway")).toBeInTheDocument();
    });
    expect(screen.getByText("default")).toBeInTheDocument();
    expect(screen.getByText("nginx")).toBeInTheDocument();
  });

  it("renders ErrorState when query fails", async () => {
    mockFetchGateways.mockRejectedValue(new Error("Network error"));

    render(<GatewayList />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText("Failed to load gateways")).toBeInTheDocument();
    });
    expect(screen.getByRole("button", { name: "Retry" })).toBeInTheDocument();
  });

  it("renders empty state when no gateways", async () => {
    mockFetchGateways.mockResolvedValue([]);

    render(<GatewayList />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText("No gateways found.")).toBeInTheDocument();
    });
  });

  it("renders heading and create button", () => {
    mockFetchGateways.mockReturnValue(new Promise(() => {}));
    render(<GatewayList />, { wrapper: createWrapper() });

    expect(screen.getByText("Gateways")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Create Gateway" })).toBeInTheDocument();
  });
});
