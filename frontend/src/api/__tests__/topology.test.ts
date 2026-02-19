import { describe, it, expect, vi, beforeEach } from "vitest";
import { fetchTopology, fetchTopologyByGateway } from "../../api/topology";
import apiClient from "../../api/client";

vi.mock("../../api/client", () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}));

describe("topology API", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("fetchTopology calls GET /topology/full when no clusterName is provided", async () => {
    const mockData = { nodes: [], edges: [] };
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchTopology();

    expect(apiClient.get).toHaveBeenCalledWith("/topology/full");
    expect(result).toEqual(mockData);
  });

  it("fetchTopology calls GET /clusters/:name/topology/full when clusterName is provided", async () => {
    const mockData = { nodes: [{ id: "gw-1" }], edges: [] };
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchTopology("prod-us");

    expect(apiClient.get).toHaveBeenCalledWith("/clusters/prod-us/topology/full");
    expect(result).toEqual(mockData);
  });

  it("fetchTopologyByGateway calls GET /topology/by-gateway/:name", async () => {
    const mockData = { nodes: [{ id: "gw-1" }], edges: [{ source: "gw-1", target: "route-1" }] };
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchTopologyByGateway("my-gw");

    expect(apiClient.get).toHaveBeenCalledWith("/topology/by-gateway/my-gw");
    expect(result).toEqual(mockData);
  });
});
