import { describe, it, expect, vi, beforeEach } from "vitest";
import apiClient from "../client";
import {
  fetchGRPCRoutes,
  fetchGRPCRoute,
  createGRPCRoute,
  updateGRPCRoute,
  deleteGRPCRoute,
} from "../grpcroutes";

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

describe("grpcroutes API", () => {
  it("fetchGRPCRoutes() calls GET /grpcroutes with no params", async () => {
    const mockData = [{ name: "grpc-1", namespace: "default" }];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchGRPCRoutes();

    expect(apiClient.get).toHaveBeenCalledWith("/grpcroutes", { params: {} });
    expect(result).toEqual(mockData);
  });

  it('fetchGRPCRoutes("ns1") calls GET /grpcroutes with namespace param', async () => {
    const mockData = [{ name: "grpc-1", namespace: "ns1" }];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchGRPCRoutes("ns1");

    expect(apiClient.get).toHaveBeenCalledWith("/grpcroutes", {
      params: { namespace: "ns1" },
    });
    expect(result).toEqual(mockData);
  });

  it("fetchGRPCRoute() calls GET /grpcroutes/:namespace/:name", async () => {
    const mockData = { name: "grpc-1", namespace: "default" };
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchGRPCRoute("default", "grpc-1");

    expect(apiClient.get).toHaveBeenCalledWith("/grpcroutes/default/grpc-1");
    expect(result).toEqual(mockData);
  });

  it("createGRPCRoute() calls POST /grpcroutes with payload", async () => {
    const payload = { name: "new-grpc", namespace: "default", parentRefs: [{ name: "gw" }], rules: [] };
    const mockData = { name: "new-grpc", namespace: "default", status: "created" };
    vi.mocked(apiClient.post).mockResolvedValue({ data: mockData });

    const result = await createGRPCRoute(payload as any);

    expect(apiClient.post).toHaveBeenCalledWith("/grpcroutes", payload);
    expect(result).toEqual(mockData);
  });

  it("updateGRPCRoute() calls PUT /grpcroutes/:namespace/:name with payload", async () => {
    const payload = { parentRefs: [{ name: "gw" }], rules: [] };
    const mockData = { name: "grpc-1", namespace: "default", rules: [] };
    vi.mocked(apiClient.put).mockResolvedValue({ data: mockData });

    const result = await updateGRPCRoute("default", "grpc-1", payload as any);

    expect(apiClient.put).toHaveBeenCalledWith("/grpcroutes/default/grpc-1", payload);
    expect(result).toEqual(mockData);
  });

  it("deleteGRPCRoute() calls DELETE /grpcroutes/:namespace/:name", async () => {
    vi.mocked(apiClient.delete).mockResolvedValue({ data: undefined });

    await deleteGRPCRoute("default", "grpc-1");

    expect(apiClient.delete).toHaveBeenCalledWith("/grpcroutes/default/grpc-1");
  });
});
