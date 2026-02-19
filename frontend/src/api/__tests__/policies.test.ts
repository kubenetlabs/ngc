import { describe, it, expect, vi, beforeEach } from "vitest";
import apiClient from "../client";
import {
  fetchPolicies,
  fetchPolicy,
  createPolicy,
  deletePolicy,
} from "../policies";

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

describe("policies API", () => {
  it("fetchPolicies() calls GET /policies/:type", async () => {
    const mockData = [{ name: "my-policy" }];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchPolicies("ratelimit" as any);

    expect(apiClient.get).toHaveBeenCalledWith("/policies/ratelimit");
    expect(result).toEqual(mockData);
  });

  it("fetchPolicy() calls GET /policies/:type/:name with no namespace", async () => {
    const mockData = { name: "my-policy", type: "ratelimit" };
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchPolicy("ratelimit" as any, "my-policy");

    expect(apiClient.get).toHaveBeenCalledWith(
      "/policies/ratelimit/my-policy",
      { params: {} },
    );
    expect(result).toEqual(mockData);
  });

  it("fetchPolicy() calls GET /policies/:type/:name with namespace param", async () => {
    const mockData = { name: "my-policy", type: "ratelimit", namespace: "ns1" };
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchPolicy("ratelimit" as any, "my-policy", "ns1");

    expect(apiClient.get).toHaveBeenCalledWith(
      "/policies/ratelimit/my-policy",
      { params: { namespace: "ns1" } },
    );
    expect(result).toEqual(mockData);
  });

  it("createPolicy() calls POST /policies/:type with body", async () => {
    const policyObj = { name: "my-policy", spec: {} };
    const mockData = { name: "my-policy", spec: {}, status: "created" };
    vi.mocked(apiClient.post).mockResolvedValue({ data: mockData });

    const result = await createPolicy("ratelimit" as any, policyObj as any);

    expect(apiClient.post).toHaveBeenCalledWith(
      "/policies/ratelimit",
      policyObj,
    );
    expect(result).toEqual(mockData);
  });

  it("deletePolicy() calls DELETE /policies/:type/:name with no namespace", async () => {
    vi.mocked(apiClient.delete).mockResolvedValue({ data: undefined });

    await deletePolicy("ratelimit" as any, "my-policy");

    expect(apiClient.delete).toHaveBeenCalledWith(
      "/policies/ratelimit/my-policy",
      { params: {} },
    );
  });

  it("deletePolicy() calls DELETE /policies/:type/:name with namespace param", async () => {
    vi.mocked(apiClient.delete).mockResolvedValue({ data: undefined });

    await deletePolicy("ratelimit" as any, "my-policy", "ns1");

    expect(apiClient.delete).toHaveBeenCalledWith(
      "/policies/ratelimit/my-policy",
      { params: { namespace: "ns1" } },
    );
  });
});
