import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  fetchXCStatus,
  fetchXCPublishes,
  fetchXCMetrics,
  publishToXC,
  deleteXCPublish,
} from "../../api/xc";
import apiClient from "../../api/client";

vi.mock("../../api/client", () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}));

describe("xc API", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("fetchXCStatus calls GET /xc/status", async () => {
    const mockData = { connected: true, publishCount: 3 };
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchXCStatus();

    expect(apiClient.get).toHaveBeenCalledWith("/xc/status");
    expect(result).toEqual(mockData);
  });

  it("fetchXCPublishes calls GET /xc/publishes", async () => {
    const mockData = [
      {
        name: "pub-1",
        namespace: "default",
        httpRouteRef: "route-1",
        phase: "published",
        createdAt: "2025-01-01T00:00:00Z",
      },
    ];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchXCPublishes();

    expect(apiClient.get).toHaveBeenCalledWith("/xc/publishes");
    expect(result).toEqual(mockData);
  });

  it("fetchXCPublishes returns [] when API fails", async () => {
    vi.mocked(apiClient.get).mockRejectedValue(new Error("Network error"));

    const result = await fetchXCPublishes();

    expect(apiClient.get).toHaveBeenCalledWith("/xc/publishes");
    expect(result).toEqual([]);
  });

  it("fetchXCMetrics calls GET /xc/metrics", async () => {
    const mockData = {
      totalRequests: 1000,
      avgLatencyMs: 42,
      errorRate: 0.01,
      regions: [{ name: "us-east", requests: 500, latencyMs: 40 }],
    };
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchXCMetrics();

    expect(apiClient.get).toHaveBeenCalledWith("/xc/metrics");
    expect(result).toEqual(mockData);
  });

  it("publishToXC calls POST /xc/publish with request body", async () => {
    const req = {
      name: "my-publish",
      namespace: "default",
      httpRouteRef: "route-1",
    };
    const mockData = { ...req, phase: "published", createdAt: "2025-01-01T00:00:00Z" };
    vi.mocked(apiClient.post).mockResolvedValue({ data: mockData });

    const result = await publishToXC(req);

    expect(apiClient.post).toHaveBeenCalledWith("/xc/publish", req);
    expect(result).toEqual(mockData);
  });

  it("deleteXCPublish calls DELETE /xc/publish/:id", async () => {
    vi.mocked(apiClient.delete).mockResolvedValue({ data: undefined });

    await deleteXCPublish("pub-123");

    expect(apiClient.delete).toHaveBeenCalledWith("/xc/publish/pub-123");
  });
});
