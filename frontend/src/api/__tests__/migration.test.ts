import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  importResources,
  analyzeImport,
  generateResources,
  applyMigration,
} from "../../api/migration";
import apiClient from "../../api/client";

vi.mock("../../api/client", () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}));

describe("migration API", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("importResources calls POST /migration/import with content and format", async () => {
    const req = { content: "server { listen 80; }", format: "nginx-conf" as const };
    const mockData = { importId: "imp-1", resourceCount: 3, format: "nginx-conf" };
    vi.mocked(apiClient.post).mockResolvedValue({ data: mockData });

    const result = await importResources(req);

    expect(apiClient.post).toHaveBeenCalledWith("/migration/import", req);
    expect(result).toEqual(mockData);
  });

  it("analyzeImport calls POST /migration/analysis", async () => {
    const req = { importId: "imp-1" };
    const mockData = {
      analysisId: "ana-1",
      resources: [
        {
          kind: "HTTPRoute",
          name: "route-1",
          namespace: "default",
          confidence: "high",
          notes: [],
        },
      ],
      warnings: [],
      errors: [],
    };
    vi.mocked(apiClient.post).mockResolvedValue({ data: mockData });

    const result = await analyzeImport(req);

    expect(apiClient.post).toHaveBeenCalledWith("/migration/analysis", req);
    expect(result).toEqual(mockData);
  });

  it("generateResources calls POST /migration/generate", async () => {
    const req = { analysisId: "ana-1" };
    const mockData = {
      generateId: "gen-1",
      resources: [
        { kind: "HTTPRoute", name: "route-1", namespace: "default", yaml: "apiVersion: v1" },
      ],
    };
    vi.mocked(apiClient.post).mockResolvedValue({ data: mockData });

    const result = await generateResources(req);

    expect(apiClient.post).toHaveBeenCalledWith("/migration/generate", req);
    expect(result).toEqual(mockData);
  });

  it("applyMigration calls POST /migration/apply with dryRun flag", async () => {
    const req = { generateId: "gen-1", dryRun: true };
    const mockData = {
      results: [
        { resource: "HTTPRoute/route-1", status: "created", message: "OK" },
      ],
      successCount: 1,
      failureCount: 0,
    };
    vi.mocked(apiClient.post).mockResolvedValue({ data: mockData });

    const result = await applyMigration(req);

    expect(apiClient.post).toHaveBeenCalledWith("/migration/apply", req);
    expect(result).toEqual(mockData);
  });
});
