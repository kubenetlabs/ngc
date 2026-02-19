import { describe, it, expect, vi, beforeEach } from "vitest";
import apiClient from "../client";
import {
  fetchClusters,
  fetchClusterDetail,
  registerCluster,
  unregisterCluster,
  testClusterConnection,
  getAgentInstallCommand,
  getClusterSummary,
} from "../clusters";

vi.mock("../client", () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}));

beforeEach(() => {
  vi.clearAllMocks();
});

describe("clusters API", () => {
  it("fetchClusters() calls GET /clusters", async () => {
    const mockData = [{ name: "prod" }];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchClusters();

    expect(apiClient.get).toHaveBeenCalledWith("/clusters");
    expect(result).toEqual(mockData);
  });

  it("fetchClusterDetail() calls GET /clusters/:name/detail", async () => {
    const mockData = { name: "prod", status: "connected" };
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchClusterDetail("prod");

    expect(apiClient.get).toHaveBeenCalledWith("/clusters/prod/detail");
    expect(result).toEqual(mockData);
  });

  it("registerCluster() calls POST /clusters with payload", async () => {
    const payload = { name: "prod", kubeconfig: "..." };
    const mockData = { message: "registered", name: "prod" };
    vi.mocked(apiClient.post).mockResolvedValue({ data: mockData });

    const result = await registerCluster(payload as any);

    expect(apiClient.post).toHaveBeenCalledWith("/clusters", payload);
    expect(result).toEqual(mockData);
  });

  it("unregisterCluster() calls DELETE /clusters/:name", async () => {
    const mockData = { message: "unregistered", name: "prod" };
    vi.mocked(apiClient.delete).mockResolvedValue({ data: mockData });

    const result = await unregisterCluster("prod");

    expect(apiClient.delete).toHaveBeenCalledWith("/clusters/prod");
    expect(result).toEqual(mockData);
  });

  it("testClusterConnection() calls POST /clusters/:name/test", async () => {
    const mockData = { success: true, latencyMs: 42 };
    vi.mocked(apiClient.post).mockResolvedValue({ data: mockData });

    const result = await testClusterConnection("prod");

    expect(apiClient.post).toHaveBeenCalledWith("/clusters/prod/test");
    expect(result).toEqual(mockData);
  });

  it("getAgentInstallCommand() calls POST /clusters/:name/install-agent", async () => {
    const mockData = { command: "helm install ...", token: "abc123" };
    vi.mocked(apiClient.post).mockResolvedValue({ data: mockData });

    const result = await getAgentInstallCommand("prod");

    expect(apiClient.post).toHaveBeenCalledWith(
      "/clusters/prod/install-agent",
    );
    expect(result).toEqual(mockData);
  });

  it("getClusterSummary() calls GET /clusters/summary", async () => {
    const mockData = { total: 5, connected: 3, disconnected: 2 };
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await getClusterSummary();

    expect(apiClient.get).toHaveBeenCalledWith("/clusters/summary");
    expect(result).toEqual(mockData);
  });
});
