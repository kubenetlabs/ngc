import { describe, it, expect, vi, beforeEach } from "vitest";
import apiClient from "../client";
import {
  fetchGateways,
  fetchGateway,
  fetchGatewayClasses,
  fetchGatewayClass,
  createGateway,
  updateGateway,
  deleteGateway,
  fetchGatewayBundles,
  fetchGatewayBundle,
  createGatewayBundle,
  updateGatewayBundle,
  deleteGatewayBundle,
} from "../gateways";

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

describe("gateways API", () => {
  // --- Gateways ---

  it("fetchGateways() calls GET /gateways with no params", async () => {
    const mockData = [{ name: "gw1" }];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchGateways();

    expect(apiClient.get).toHaveBeenCalledWith("/gateways", { params: {} });
    expect(result).toEqual(mockData);
  });

  it('fetchGateways("ns1") calls GET /gateways with namespace param', async () => {
    const mockData = [{ name: "gw1" }];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchGateways("ns1");

    expect(apiClient.get).toHaveBeenCalledWith("/gateways", {
      params: { namespace: "ns1" },
    });
    expect(result).toEqual(mockData);
  });

  it("fetchGateway() calls GET /gateways/:namespace/:name", async () => {
    const mockData = { name: "gw1", namespace: "ns1" };
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchGateway("ns1", "gw1");

    expect(apiClient.get).toHaveBeenCalledWith("/gateways/ns1/gw1");
    expect(result).toEqual(mockData);
  });

  it("fetchGatewayClasses() calls GET /gatewayclasses", async () => {
    const mockData = [{ name: "nginx" }];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchGatewayClasses();

    expect(apiClient.get).toHaveBeenCalledWith("/gatewayclasses");
    expect(result).toEqual(mockData);
  });

  it("fetchGatewayClass() calls GET /gatewayclasses/:name", async () => {
    const mockData = { name: "nginx" };
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchGatewayClass("nginx");

    expect(apiClient.get).toHaveBeenCalledWith("/gatewayclasses/nginx");
    expect(result).toEqual(mockData);
  });

  it("createGateway() calls POST /gateways with payload", async () => {
    const payload = { name: "gw1", namespace: "ns1" };
    const mockData = { name: "gw1", namespace: "ns1", status: "created" };
    vi.mocked(apiClient.post).mockResolvedValue({ data: mockData });

    const result = await createGateway(payload as any);

    expect(apiClient.post).toHaveBeenCalledWith("/gateways", payload);
    expect(result).toEqual(mockData);
  });

  it("updateGateway() calls PUT /gateways/:namespace/:name with payload", async () => {
    const payload = { listeners: [] };
    const mockData = { name: "gw1", namespace: "ns1", listeners: [] };
    vi.mocked(apiClient.put).mockResolvedValue({ data: mockData });

    const result = await updateGateway("ns1", "gw1", payload as any);

    expect(apiClient.put).toHaveBeenCalledWith(
      "/gateways/ns1/gw1",
      payload,
    );
    expect(result).toEqual(mockData);
  });

  it("deleteGateway() calls DELETE /gateways/:namespace/:name", async () => {
    vi.mocked(apiClient.delete).mockResolvedValue({ data: undefined });

    await deleteGateway("ns1", "gw1");

    expect(apiClient.delete).toHaveBeenCalledWith("/gateways/ns1/gw1");
  });

  // --- GatewayBundles ---

  it("fetchGatewayBundles() calls GET /gatewaybundles with no params", async () => {
    const mockData = [{ name: "b1" }];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchGatewayBundles();

    expect(apiClient.get).toHaveBeenCalledWith("/gatewaybundles", {
      params: {},
    });
    expect(result).toEqual(mockData);
  });

  it('fetchGatewayBundles("ns1") calls GET /gatewaybundles with namespace param', async () => {
    const mockData = [{ name: "b1" }];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchGatewayBundles("ns1");

    expect(apiClient.get).toHaveBeenCalledWith("/gatewaybundles", {
      params: { namespace: "ns1" },
    });
    expect(result).toEqual(mockData);
  });

  it("fetchGatewayBundle() calls GET /gatewaybundles/:namespace/:name", async () => {
    const mockData = { name: "b1", namespace: "ns1" };
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchGatewayBundle("ns1", "b1");

    expect(apiClient.get).toHaveBeenCalledWith("/gatewaybundles/ns1/b1");
    expect(result).toEqual(mockData);
  });

  it("createGatewayBundle() calls POST /gatewaybundles with payload", async () => {
    const payload = { name: "b1", namespace: "ns1" };
    const mockData = { name: "b1", namespace: "ns1", status: "created" };
    vi.mocked(apiClient.post).mockResolvedValue({ data: mockData });

    const result = await createGatewayBundle(payload as any);

    expect(apiClient.post).toHaveBeenCalledWith("/gatewaybundles", payload);
    expect(result).toEqual(mockData);
  });

  it("updateGatewayBundle() calls PUT /gatewaybundles/:namespace/:name with payload", async () => {
    const payload = { displayName: "Updated" };
    const mockData = { name: "b1", namespace: "ns1", displayName: "Updated" };
    vi.mocked(apiClient.put).mockResolvedValue({ data: mockData });

    const result = await updateGatewayBundle("ns1", "b1", payload as any);

    expect(apiClient.put).toHaveBeenCalledWith(
      "/gatewaybundles/ns1/b1",
      payload,
    );
    expect(result).toEqual(mockData);
  });

  it("deleteGatewayBundle() calls DELETE /gatewaybundles/:namespace/:name", async () => {
    vi.mocked(apiClient.delete).mockResolvedValue({ data: undefined });

    await deleteGatewayBundle("ns1", "b1");

    expect(apiClient.delete).toHaveBeenCalledWith("/gatewaybundles/ns1/b1");
  });
});
