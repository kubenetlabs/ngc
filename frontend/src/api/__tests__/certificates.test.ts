import { describe, it, expect, vi, beforeEach } from "vitest";
import { fetchCertificates, fetchExpiringCertificates, deleteCertificate } from "../../api/certificates";
import apiClient from "../../api/client";

vi.mock("../../api/client", () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}));

describe("certificates API", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("fetchCertificates", () => {
    it("sends GET /certificates and returns data", async () => {
      const mockCerts = [
        { name: "tls-cert", namespace: "default", notAfter: "2026-01-01T00:00:00Z" },
      ];
      vi.mocked(apiClient.get).mockResolvedValue({ data: mockCerts });

      const result = await fetchCertificates();

      expect(apiClient.get).toHaveBeenCalledWith("/certificates");
      expect(result).toEqual(mockCerts);
    });
  });

  describe("fetchExpiringCertificates", () => {
    it("sends GET /certificates/expiring with empty params when no days specified", async () => {
      const mockCerts = [{ name: "expiring-cert", namespace: "prod" }];
      vi.mocked(apiClient.get).mockResolvedValue({ data: mockCerts });

      const result = await fetchExpiringCertificates();

      expect(apiClient.get).toHaveBeenCalledWith("/certificates/expiring", { params: {} });
      expect(result).toEqual(mockCerts);
    });

    it("sends GET /certificates/expiring with days param when specified", async () => {
      const mockCerts = [{ name: "soon-expiring", namespace: "staging" }];
      vi.mocked(apiClient.get).mockResolvedValue({ data: mockCerts });

      const result = await fetchExpiringCertificates(30);

      expect(apiClient.get).toHaveBeenCalledWith("/certificates/expiring", { params: { days: 30 } });
      expect(result).toEqual(mockCerts);
    });
  });

  describe("deleteCertificate", () => {
    it("sends DELETE /certificates/:name", async () => {
      vi.mocked(apiClient.delete).mockResolvedValue({ data: undefined });

      await deleteCertificate("my-cert");

      expect(apiClient.delete).toHaveBeenCalledWith("/certificates/my-cert");
    });
  });
});
