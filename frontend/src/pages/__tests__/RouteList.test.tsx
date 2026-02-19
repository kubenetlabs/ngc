import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import RouteList from "../RouteList";

// Mock API modules
const mockFetchHTTPRoutes = vi.fn();
vi.mock("@/api/routes", () => ({
  fetchHTTPRoutes: (...args: unknown[]) => mockFetchHTTPRoutes(...args),
}));
vi.mock("@/api/grpcroutes", () => ({
  fetchGRPCRoutes: vi.fn().mockResolvedValue([]),
}));
vi.mock("@/api/tlsroutes", () => ({
  fetchTLSRoutes: vi.fn().mockResolvedValue([]),
}));
vi.mock("@/api/tcproutes", () => ({
  fetchTCPRoutes: vi.fn().mockResolvedValue([]),
}));
vi.mock("@/api/udproutes", () => ({
  fetchUDPRoutes: vi.fn().mockResolvedValue([]),
}));

// Mock useActiveCluster to return a single cluster (not ALL)
vi.mock("@/hooks/useActiveCluster", () => ({
  useActiveCluster: () => "test-cluster",
}));

vi.mock("@/store/clusterStore", () => ({
  ALL_CLUSTERS: "all",
  useClusterStore: () => "test-cluster",
}));

// Mock GlobalRouteList
vi.mock("@/components/global/GlobalRouteList", () => ({
  GlobalRouteList: () => <div>Global Route List</div>,
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

describe("RouteList", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders loading state initially", () => {
    mockFetchHTTPRoutes.mockReturnValue(new Promise(() => {}));
    render(<RouteList />, { wrapper: createWrapper() });

    expect(screen.getByText("Loading routes...")).toBeInTheDocument();
  });

  it("renders route list when data loads", async () => {
    mockFetchHTTPRoutes.mockResolvedValue([
      {
        name: "web-route",
        namespace: "default",
        hostnames: ["example.com"],
        parentRefs: [{ name: "my-gateway" }],
        rules: [{ matches: [], backendRefs: [] }],
        status: {
          parents: [
            {
              parentRef: { name: "my-gateway" },
              conditions: [{ type: "Accepted", status: "True", reason: "Accepted" }],
            },
          ],
        },
        createdAt: new Date().toISOString(),
      },
    ]);

    render(<RouteList />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText("web-route")).toBeInTheDocument();
    });
    expect(screen.getByText("default")).toBeInTheDocument();
    expect(screen.getByText("example.com")).toBeInTheDocument();
  });

  it("renders ErrorState when query fails", async () => {
    mockFetchHTTPRoutes.mockRejectedValue(new Error("Connection refused"));

    render(<RouteList />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText("Failed to load routes")).toBeInTheDocument();
    });
    expect(screen.getByRole("button", { name: "Retry" })).toBeInTheDocument();
  });

  it("renders empty state when no routes", async () => {
    mockFetchHTTPRoutes.mockResolvedValue([]);

    render(<RouteList />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText("No HTTPRoutes found.")).toBeInTheDocument();
    });
  });

  it("renders heading and route type tabs", () => {
    mockFetchHTTPRoutes.mockReturnValue(new Promise(() => {}));
    render(<RouteList />, { wrapper: createWrapper() });

    expect(screen.getByText("Routes")).toBeInTheDocument();
    expect(screen.getByText("HTTP")).toBeInTheDocument();
    expect(screen.getByText("gRPC")).toBeInTheDocument();
    expect(screen.getByText("TLS")).toBeInTheDocument();
    expect(screen.getByText("TCP")).toBeInTheDocument();
    expect(screen.getByText("UDP")).toBeInTheDocument();
  });
});
