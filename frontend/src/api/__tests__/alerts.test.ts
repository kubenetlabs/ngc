import { describe, it, expect, vi, beforeEach } from "vitest";
import { fetchAlertRules, createAlertRule, deleteAlertRule, toggleAlertRule } from "../../api/alerts";
import apiClient from "../../api/client";

vi.mock("../../api/client", () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}));

describe("alerts API", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("fetchAlertRules", () => {
    it("sends GET /alerts and returns data", async () => {
      const mockRules = [
        { id: "1", name: "High latency", resource: "gateway", metric: "latency", operator: "gt", threshold: 500, severity: "critical", enabled: true, createdAt: "2025-01-01T00:00:00Z" },
      ];
      vi.mocked(apiClient.get).mockResolvedValue({ data: mockRules });

      const result = await fetchAlertRules();

      expect(apiClient.get).toHaveBeenCalledWith("/alerts");
      expect(result).toEqual(mockRules);
    });
  });

  describe("createAlertRule", () => {
    it("sends POST /alerts with rule body and returns created rule", async () => {
      const newRule = {
        name: "Cert expiry",
        resource: "certificate" as const,
        metric: "days_until_expiry",
        operator: "lt" as const,
        threshold: 7,
        severity: "warning" as const,
      };
      const createdRule = { ...newRule, id: "alert-456", enabled: true, createdAt: "2025-01-01T00:00:00Z" };
      vi.mocked(apiClient.post).mockResolvedValue({ data: createdRule });

      const result = await createAlertRule(newRule);

      expect(apiClient.post).toHaveBeenCalledWith("/alerts", newRule);
      expect(result).toEqual(createdRule);
    });
  });

  describe("deleteAlertRule", () => {
    it("sends DELETE /alerts/:id", async () => {
      vi.mocked(apiClient.delete).mockResolvedValue({ data: undefined });

      await deleteAlertRule("alert-123");

      expect(apiClient.delete).toHaveBeenCalledWith("/alerts/alert-123");
    });
  });

  describe("toggleAlertRule", () => {
    it("sends POST /alerts/:id/toggle and returns updated rule", async () => {
      const toggledRule = { id: "alert-123", name: "Test", resource: "gateway", metric: "latency", operator: "gt", threshold: 100, severity: "info", enabled: false, createdAt: "2025-01-01T00:00:00Z" };
      vi.mocked(apiClient.post).mockResolvedValue({ data: toggledRule });

      const result = await toggleAlertRule("alert-123");

      expect(apiClient.post).toHaveBeenCalledWith("/alerts/alert-123/toggle");
      expect(result).toEqual(toggledRule);
    });
  });
});
