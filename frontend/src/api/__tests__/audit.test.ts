import { describe, it, expect, vi, beforeEach } from "vitest";
import { fetchAuditEntries, fetchAuditDiff } from "../../api/audit";
import apiClient from "../../api/client";

vi.mock("../../api/client", () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}));

describe("audit API", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("fetchAuditEntries", () => {
    it("sends GET /audit with no params when called without arguments", async () => {
      const mockResponse = { entries: [], total: 0 };
      vi.mocked(apiClient.get).mockResolvedValue({ data: mockResponse });

      const result = await fetchAuditEntries();

      expect(apiClient.get).toHaveBeenCalledWith("/audit", { params: undefined });
      expect(result).toEqual(mockResponse);
    });

    it("sends GET /audit with params when provided", async () => {
      const mockResponse = { entries: [{ id: "1", resource: "gateway", action: "create" }], total: 1 };
      vi.mocked(apiClient.get).mockResolvedValue({ data: mockResponse });

      const result = await fetchAuditEntries({ resource: "gateway", limit: 10 });

      expect(apiClient.get).toHaveBeenCalledWith("/audit", { params: { resource: "gateway", limit: 10 } });
      expect(result).toEqual(mockResponse);
    });
  });

  describe("fetchAuditDiff", () => {
    it("sends GET /audit/diff/:id and returns diff data", async () => {
      const mockDiff = { id: "diff-123", before: "{}", after: "{}", diff: "..." };
      vi.mocked(apiClient.get).mockResolvedValue({ data: mockDiff });

      const result = await fetchAuditDiff("diff-123");

      expect(apiClient.get).toHaveBeenCalledWith("/audit/diff/diff-123");
      expect(result).toEqual(mockDiff);
    });
  });
});
