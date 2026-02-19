import { describe, it, expect, vi, beforeEach } from "vitest";
import apiClient from "../client";
import {
  fetchHTTPRoutes,
  fetchHTTPRoute,
  createHTTPRoute,
  updateHTTPRoute,
  deleteHTTPRoute,
} from "../routes";

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

describe("routes API", () => {
  it("fetchHTTPRoutes() calls GET /httproutes with no params", async () => {
    const mockData = [{ name: "r1" }];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchHTTPRoutes();

    expect(apiClient.get).toHaveBeenCalledWith("/httproutes", { params: {} });
    expect(result).toEqual(mockData);
  });

  it('fetchHTTPRoutes("ns1") calls GET /httproutes with namespace param', async () => {
    const mockData = [{ name: "r1" }];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchHTTPRoutes("ns1");

    expect(apiClient.get).toHaveBeenCalledWith("/httproutes", {
      params: { namespace: "ns1" },
    });
    expect(result).toEqual(mockData);
  });

  it("fetchHTTPRoute() calls GET /httproutes/:namespace/:name", async () => {
    const mockData = { name: "r1", namespace: "ns1" };
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchHTTPRoute("ns1", "r1");

    expect(apiClient.get).toHaveBeenCalledWith("/httproutes/ns1/r1");
    expect(result).toEqual(mockData);
  });

  it("createHTTPRoute() calls POST /httproutes with payload", async () => {
    const payload = { name: "r1", namespace: "ns1" };
    const mockData = { name: "r1", namespace: "ns1", status: "created" };
    vi.mocked(apiClient.post).mockResolvedValue({ data: mockData });

    const result = await createHTTPRoute(payload as any);

    expect(apiClient.post).toHaveBeenCalledWith("/httproutes", payload);
    expect(result).toEqual(mockData);
  });

  it("updateHTTPRoute() calls PUT /httproutes/:namespace/:name with payload", async () => {
    const payload = { rules: [] };
    const mockData = { name: "r1", namespace: "ns1", rules: [] };
    vi.mocked(apiClient.put).mockResolvedValue({ data: mockData });

    const result = await updateHTTPRoute("ns1", "r1", payload as any);

    expect(apiClient.put).toHaveBeenCalledWith("/httproutes/ns1/r1", payload);
    expect(result).toEqual(mockData);
  });

  it("deleteHTTPRoute() calls DELETE /httproutes/:namespace/:name", async () => {
    vi.mocked(apiClient.delete).mockResolvedValue({ data: undefined });

    await deleteHTTPRoute("ns1", "r1");

    expect(apiClient.delete).toHaveBeenCalledWith("/httproutes/ns1/r1");
  });
});
