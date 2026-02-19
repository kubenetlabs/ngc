import { describe, it, expect, vi, beforeEach } from "vitest";
import apiClient from "../client";
import {
  fetchTLSRoutes,
  fetchTLSRoute,
  createTLSRoute,
  updateTLSRoute,
  deleteTLSRoute,
} from "../tlsroutes";

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

describe("tlsroutes API", () => {
  it("fetchTLSRoutes() calls GET /tlsroutes with no params", async () => {
    const mockData = [{ name: "tls-1", namespace: "default" }];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchTLSRoutes();

    expect(apiClient.get).toHaveBeenCalledWith("/tlsroutes", { params: {} });
    expect(result).toEqual(mockData);
  });

  it('fetchTLSRoutes("ns1") calls GET /tlsroutes with namespace param', async () => {
    const mockData = [{ name: "tls-1", namespace: "ns1" }];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchTLSRoutes("ns1");

    expect(apiClient.get).toHaveBeenCalledWith("/tlsroutes", {
      params: { namespace: "ns1" },
    });
    expect(result).toEqual(mockData);
  });

  it("fetchTLSRoute() calls GET /tlsroutes/:namespace/:name", async () => {
    const mockData = { name: "tls-1", namespace: "default" };
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchTLSRoute("default", "tls-1");

    expect(apiClient.get).toHaveBeenCalledWith("/tlsroutes/default/tls-1");
    expect(result).toEqual(mockData);
  });

  it("createTLSRoute() calls POST /tlsroutes with payload", async () => {
    const payload = { name: "new-tls", namespace: "default", parentRefs: [{ name: "gw" }], rules: [] };
    const mockData = { name: "new-tls", namespace: "default", status: "created" };
    vi.mocked(apiClient.post).mockResolvedValue({ data: mockData });

    const result = await createTLSRoute(payload as any);

    expect(apiClient.post).toHaveBeenCalledWith("/tlsroutes", payload);
    expect(result).toEqual(mockData);
  });

  it("updateTLSRoute() calls PUT /tlsroutes/:namespace/:name with payload", async () => {
    const payload = { parentRefs: [{ name: "gw" }], rules: [] };
    const mockData = { name: "tls-1", namespace: "default", rules: [] };
    vi.mocked(apiClient.put).mockResolvedValue({ data: mockData });

    const result = await updateTLSRoute("default", "tls-1", payload as any);

    expect(apiClient.put).toHaveBeenCalledWith("/tlsroutes/default/tls-1", payload);
    expect(result).toEqual(mockData);
  });

  it("deleteTLSRoute() calls DELETE /tlsroutes/:namespace/:name", async () => {
    vi.mocked(apiClient.delete).mockResolvedValue({ data: undefined });

    await deleteTLSRoute("default", "tls-1");

    expect(apiClient.delete).toHaveBeenCalledWith("/tlsroutes/default/tls-1");
  });
});
