import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  fetchMetricsSummary,
  fetchMetricsByRoute,
  fetchMetricsByGateway,
} from "../../api/metrics";
import apiClient from "../../api/client";

vi.mock("../../api/client", () => ({
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

describe("fetchMetricsSummary", () => {
  it("calls GET /metrics/summary and returns data", async () => {
    const mockSummary = { totalRequests: 1000, errorRate: 0.02 };
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockSummary });

    const result = await fetchMetricsSummary();

    expect(apiClient.get).toHaveBeenCalledWith("/metrics/summary");
    expect(result).toEqual(mockSummary);
  });
});

describe("fetchMetricsByRoute", () => {
  it("calls GET /metrics/by-route and returns data", async () => {
    const mockRoutes = [{ route: "/api/v1", requests: 500 }];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockRoutes });

    const result = await fetchMetricsByRoute();

    expect(apiClient.get).toHaveBeenCalledWith("/metrics/by-route");
    expect(result).toEqual(mockRoutes);
  });
});

describe("fetchMetricsByGateway", () => {
  it("calls GET /metrics/by-gateway and returns data", async () => {
    const mockGateways = [{ gateway: "gw1", requests: 300 }];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockGateways });

    const result = await fetchMetricsByGateway();

    expect(apiClient.get).toHaveBeenCalledWith("/metrics/by-gateway");
    expect(result).toEqual(mockGateways);
  });
});
