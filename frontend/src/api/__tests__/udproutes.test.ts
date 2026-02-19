import { describe, it, expect, vi, beforeEach } from "vitest";
import apiClient from "../client";
import {
  fetchUDPRoutes,
  fetchUDPRoute,
  createUDPRoute,
  updateUDPRoute,
  deleteUDPRoute,
} from "../udproutes";

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

describe("udproutes API", () => {
  it("fetchUDPRoutes() calls GET /udproutes with no params", async () => {
    const mockData = [{ name: "udp-1", namespace: "default" }];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchUDPRoutes();

    expect(apiClient.get).toHaveBeenCalledWith("/udproutes", { params: {} });
    expect(result).toEqual(mockData);
  });

  it('fetchUDPRoutes("ns1") calls GET /udproutes with namespace param', async () => {
    const mockData = [{ name: "udp-1", namespace: "ns1" }];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchUDPRoutes("ns1");

    expect(apiClient.get).toHaveBeenCalledWith("/udproutes", {
      params: { namespace: "ns1" },
    });
    expect(result).toEqual(mockData);
  });

  it("fetchUDPRoute() calls GET /udproutes/:namespace/:name", async () => {
    const mockData = { name: "udp-1", namespace: "default" };
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchUDPRoute("default", "udp-1");

    expect(apiClient.get).toHaveBeenCalledWith("/udproutes/default/udp-1");
    expect(result).toEqual(mockData);
  });

  it("createUDPRoute() calls POST /udproutes with payload", async () => {
    const payload = { name: "new-udp", namespace: "default", parentRefs: [{ name: "gw" }], rules: [] };
    const mockData = { name: "new-udp", namespace: "default", status: "created" };
    vi.mocked(apiClient.post).mockResolvedValue({ data: mockData });

    const result = await createUDPRoute(payload as any);

    expect(apiClient.post).toHaveBeenCalledWith("/udproutes", payload);
    expect(result).toEqual(mockData);
  });

  it("updateUDPRoute() calls PUT /udproutes/:namespace/:name with payload", async () => {
    const payload = { parentRefs: [{ name: "gw" }], rules: [] };
    const mockData = { name: "udp-1", namespace: "default", rules: [] };
    vi.mocked(apiClient.put).mockResolvedValue({ data: mockData });

    const result = await updateUDPRoute("default", "udp-1", payload as any);

    expect(apiClient.put).toHaveBeenCalledWith("/udproutes/default/udp-1", payload);
    expect(result).toEqual(mockData);
  });

  it("deleteUDPRoute() calls DELETE /udproutes/:namespace/:name", async () => {
    vi.mocked(apiClient.delete).mockResolvedValue({ data: undefined });

    await deleteUDPRoute("default", "udp-1");

    expect(apiClient.delete).toHaveBeenCalledWith("/udproutes/default/udp-1");
  });
});
