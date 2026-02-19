import { describe, it, expect, vi, beforeEach } from "vitest";
import { queryLogs, fetchTopNLogs } from "../../api/logs";
import apiClient from "../../api/client";

vi.mock("../../api/client", () => ({
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

describe("queryLogs", () => {
  it("calls POST /logs/query with query body and returns data", async () => {
    const query = { gateway: "gw1" };
    const mockLogs = [{ timestamp: "2024-01-01", message: "request received" }];
    vi.mocked(apiClient.post).mockResolvedValue({ data: mockLogs });

    const result = await queryLogs(query as any);

    expect(apiClient.post).toHaveBeenCalledWith("/logs/query", query);
    expect(result).toEqual(mockLogs);
  });
});

describe("fetchTopNLogs", () => {
  it("calls GET /logs/topn with no params when arguments are omitted", async () => {
    const mockTopN = [{ field: "path", value: "/api", count: 100 }];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockTopN });

    const result = await fetchTopNLogs();

    expect(apiClient.get).toHaveBeenCalledWith("/logs/topn", { params: {} });
    expect(result).toEqual(mockTopN);
  });

  it("calls GET /logs/topn with field and n params when provided", async () => {
    const mockTopN = [{ field: "status_code", value: "200", count: 50 }];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockTopN });

    const result = await fetchTopNLogs("status_code", 5);

    expect(apiClient.get).toHaveBeenCalledWith("/logs/topn", { params: { field: "status_code", n: 5 } });
    expect(result).toEqual(mockTopN);
  });
});
