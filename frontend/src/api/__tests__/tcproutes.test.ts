import { describe, it, expect, vi, beforeEach } from "vitest";
import apiClient from "../client";
import {
  fetchTCPRoutes,
  fetchTCPRoute,
  createTCPRoute,
  updateTCPRoute,
  deleteTCPRoute,
} from "../tcproutes";

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

describe("tcproutes API", () => {
  it("fetchTCPRoutes() calls GET /tcproutes with no params", async () => {
    const mockData = [{ name: "tcp-1", namespace: "default" }];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchTCPRoutes();

    expect(apiClient.get).toHaveBeenCalledWith("/tcproutes", { params: {} });
    expect(result).toEqual(mockData);
  });

  it('fetchTCPRoutes("ns1") calls GET /tcproutes with namespace param', async () => {
    const mockData = [{ name: "tcp-1", namespace: "ns1" }];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchTCPRoutes("ns1");

    expect(apiClient.get).toHaveBeenCalledWith("/tcproutes", {
      params: { namespace: "ns1" },
    });
    expect(result).toEqual(mockData);
  });

  it("fetchTCPRoute() calls GET /tcproutes/:namespace/:name", async () => {
    const mockData = { name: "tcp-1", namespace: "default" };
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchTCPRoute("default", "tcp-1");

    expect(apiClient.get).toHaveBeenCalledWith("/tcproutes/default/tcp-1");
    expect(result).toEqual(mockData);
  });

  it("createTCPRoute() calls POST /tcproutes with payload", async () => {
    const payload = { name: "new-tcp", namespace: "default", parentRefs: [{ name: "gw" }], rules: [] };
    const mockData = { name: "new-tcp", namespace: "default", status: "created" };
    vi.mocked(apiClient.post).mockResolvedValue({ data: mockData });

    const result = await createTCPRoute(payload as any);

    expect(apiClient.post).toHaveBeenCalledWith("/tcproutes", payload);
    expect(result).toEqual(mockData);
  });

  it("updateTCPRoute() calls PUT /tcproutes/:namespace/:name with payload", async () => {
    const payload = { parentRefs: [{ name: "gw" }], rules: [] };
    const mockData = { name: "tcp-1", namespace: "default", rules: [] };
    vi.mocked(apiClient.put).mockResolvedValue({ data: mockData });

    const result = await updateTCPRoute("default", "tcp-1", payload as any);

    expect(apiClient.put).toHaveBeenCalledWith("/tcproutes/default/tcp-1", payload);
    expect(result).toEqual(mockData);
  });

  it("deleteTCPRoute() calls DELETE /tcproutes/:namespace/:name", async () => {
    vi.mocked(apiClient.delete).mockResolvedValue({ data: undefined });

    await deleteTCPRoute("default", "tcp-1");

    expect(apiClient.delete).toHaveBeenCalledWith("/tcproutes/default/tcp-1");
  });
});
