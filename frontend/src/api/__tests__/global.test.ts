import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  fetchGlobalGateways,
  fetchGlobalRoutes,
  fetchGlobalGPUCapacity,
} from "../../api/global";
import apiClient from "../../api/client";

vi.mock("../../api/client", () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}));

describe("global API", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("fetchGlobalGateways calls GET /global/gateways", async () => {
    const mockData = [
      {
        clusterName: "prod-us",
        clusterRegion: "us-east-1",
        gateway: {
          name: "main-gw",
          namespace: "default",
          className: "nginx",
          listeners: [{ name: "http", port: 80, protocol: "HTTP" }],
          status: "Accepted",
          addresses: ["10.0.0.1"],
        },
      },
    ];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchGlobalGateways();

    expect(apiClient.get).toHaveBeenCalledWith("/global/gateways");
    expect(result).toEqual(mockData);
  });

  it("fetchGlobalRoutes calls GET /global/routes", async () => {
    const mockData = [
      {
        clusterName: "prod-us",
        clusterRegion: "us-east-1",
        route: {
          name: "my-route",
          namespace: "default",
          hostnames: ["example.com"],
          parentRefs: [{ name: "main-gw", namespace: "default" }],
          status: "Accepted",
        },
      },
    ];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchGlobalRoutes();

    expect(apiClient.get).toHaveBeenCalledWith("/global/routes");
    expect(result).toEqual(mockData);
  });

  it("fetchGlobalGPUCapacity calls GET /global/gpu-capacity", async () => {
    const mockData = {
      totalGPUs: 16,
      allocatedGPUs: 10,
      clusters: [
        {
          clusterName: "gpu-west",
          clusterRegion: "us-west-2",
          totalGPUs: 8,
          allocatedGPUs: 5,
          gpuTypes: { "A100": 8 },
        },
      ],
    };
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchGlobalGPUCapacity();

    expect(apiClient.get).toHaveBeenCalledWith("/global/gpu-capacity");
    expect(result).toEqual(mockData);
  });
});
