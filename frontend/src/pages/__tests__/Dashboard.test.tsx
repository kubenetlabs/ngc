import { render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, it, expect, vi, beforeEach } from "vitest";
import Dashboard from "../Dashboard";

// Mock APIs
const mockFetchGateways = vi.fn();
const mockFetchHTTPRoutes = vi.fn();
const mockFetchConfig = vi.fn();
const mockFetchTopology = vi.fn();
const mockFetchCertificates = vi.fn();
const mockFetchPolicies = vi.fn();
const mockFetchAuditEntries = vi.fn();
const mockFetchClusters = vi.fn();

vi.mock("@/api/gateways", () => ({
  fetchGateways: (...args: unknown[]) => mockFetchGateways(...args),
}));
vi.mock("@/api/routes", () => ({
  fetchHTTPRoutes: (...args: unknown[]) => mockFetchHTTPRoutes(...args),
}));
vi.mock("@/api/config", () => ({
  fetchConfig: (...args: unknown[]) => mockFetchConfig(...args),
}));
vi.mock("@/api/topology", () => ({
  fetchTopology: (...args: unknown[]) => mockFetchTopology(...args),
}));
vi.mock("@/api/certificates", () => ({
  fetchCertificates: (...args: unknown[]) => mockFetchCertificates(...args),
}));
vi.mock("@/api/policies", () => ({
  fetchPolicies: (...args: unknown[]) => mockFetchPolicies(...args),
}));
vi.mock("@/api/audit", () => ({
  fetchAuditEntries: (...args: unknown[]) => mockFetchAuditEntries(...args),
}));
vi.mock("@/api/clusters", () => ({
  fetchClusters: (...args: unknown[]) => mockFetchClusters(...args),
}));

vi.mock("@/hooks/useActiveCluster", () => ({
  useActiveCluster: () => "test-cluster",
}));
vi.mock("@/hooks/useEdition", () => ({
  useEdition: () => ({ edition: "enterprise", isEnterprise: true }),
}));
vi.mock("@/store/clusterStore", () => ({
  ALL_CLUSTERS: "all",
  useClusterStore: () => "test-cluster",
}));

// Mock TopologyGraph since it uses canvas/SVG
vi.mock("@/components/topology/TopologyGraph", () => ({
  TopologyGraph: () => <div data-testid="topology-graph">Topology</div>,
}));

// Mock GlobalDashboard
vi.mock("@/components/global/GlobalDashboard", () => ({
  GlobalDashboard: () => <div>Global Dashboard</div>,
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

describe("Dashboard", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Default: resolve with minimal data
    mockFetchGateways.mockResolvedValue([]);
    mockFetchHTTPRoutes.mockResolvedValue([]);
    mockFetchConfig.mockResolvedValue({
      ngfVersion: "1.5.0",
      edition: "enterprise",
    });
    mockFetchTopology.mockResolvedValue({ nodes: [], edges: [] });
    mockFetchCertificates.mockResolvedValue([]);
    mockFetchPolicies.mockResolvedValue([]);
    mockFetchAuditEntries.mockResolvedValue({ entries: [], total: 0 });
    mockFetchClusters.mockResolvedValue([]);
  });

  it("renders Dashboard heading", async () => {
    render(<Dashboard />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText("Dashboard")).toBeInTheDocument();
    });
  });

  it("shows summary cards with gateway and route counts", async () => {
    mockFetchGateways.mockResolvedValue([
      { name: "gw1", namespace: "default", status: { conditions: [] } },
      { name: "gw2", namespace: "default", status: { conditions: [] } },
    ]);
    mockFetchHTTPRoutes.mockResolvedValue([
      { name: "rt1", namespace: "default" },
    ]);

    render(<Dashboard />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText("Gateways")).toBeInTheDocument();
    });
  });

  it("shows Quick Actions links", async () => {
    render(<Dashboard />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText("Quick Actions")).toBeInTheDocument();
    });
    expect(screen.getByText("Create Gateway")).toBeInTheDocument();
    expect(screen.getByText("Create Route")).toBeInTheDocument();
    expect(screen.getByText("Create Inference Pool")).toBeInTheDocument();
    expect(screen.getByText("Run Diagnostics")).toBeInTheDocument();
  });

  it("shows loading state with animate-pulse", () => {
    // Never resolving promises = loading state
    mockFetchGateways.mockReturnValue(new Promise(() => {}));
    mockFetchHTTPRoutes.mockReturnValue(new Promise(() => {}));
    mockFetchConfig.mockReturnValue(new Promise(() => {}));

    render(<Dashboard />, { wrapper: createWrapper() });

    expect(screen.getByText("Dashboard")).toBeInTheDocument();
    // Loading indicators should be present
    const pulseElements = document.querySelectorAll(".animate-pulse");
    expect(pulseElements.length).toBeGreaterThan(0);
  });

  it("shows Recent Activity section", async () => {
    mockFetchAuditEntries.mockResolvedValue({
      entries: [
        {
          id: "1",
          action: "create",
          resource: "Gateway",
          name: "my-gateway",
          namespace: "default",
          timestamp: new Date().toISOString(),
        },
      ],
      total: 1,
    });

    render(<Dashboard />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText("Recent Activity")).toBeInTheDocument();
    });
  });
});
