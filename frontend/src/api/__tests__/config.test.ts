import { describe, it, expect, vi, beforeEach } from "vitest";
import { fetchConfig } from "../../api/config";
import apiClient from "../../api/client";

vi.mock("../../api/client", () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}));

describe("config API", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("fetchConfig calls GET /config", async () => {
    const mockData = {
      edition: "enterprise",
      version: "1.5.0",
      connected: true,
      cluster: "prod-us",
    };
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockData });

    const result = await fetchConfig();

    expect(apiClient.get).toHaveBeenCalledWith("/config");
    expect(result).toEqual(mockData);
  });
});
