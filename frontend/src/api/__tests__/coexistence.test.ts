import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  fetchCoexistenceOverview,
  fetchMigrationReadiness,
} from "../../api/coexistence";
import apiClient from "../../api/client";

vi.mock("../../api/client", () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}));

describe("coexistence API", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("fetchCoexistenceOverview calls GET /coexistence/overview", async () => {
    const mockData = {
      kic: {
        installed: true,
        version: "3.4.0",
        resourceCount: 12,
        namespaces: ["default"],
        resources: [{ kind: "Ingress", count: 10 }],
      },
      ngf: {
        installed: true,
        version: "1.2.0",
        resourceCount: 5,
        namespaces: ["default"],
        resources: [{ kind: "HTTPRoute", count: 5 }],
      },
      sharedResources: [],
      conflicts: [],
    };
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchCoexistenceOverview();

    expect(apiClient.get).toHaveBeenCalledWith("/coexistence/overview");
    expect(result).toEqual(mockData);
  });

  it("fetchMigrationReadiness calls GET /coexistence/migration-readiness", async () => {
    const mockData = {
      score: 85,
      status: "partial",
      categories: [
        { name: "Routes", score: 90, status: "pass", details: "All routes convertible" },
      ],
      blockers: [],
      recommendations: ["Review TLS configuration"],
    };
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchMigrationReadiness();

    expect(apiClient.get).toHaveBeenCalledWith("/coexistence/migration-readiness");
    expect(result).toEqual(mockData);
  });
});
