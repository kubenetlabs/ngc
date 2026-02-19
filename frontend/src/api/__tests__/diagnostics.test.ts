import { describe, it, expect, vi, beforeEach } from "vitest";
import { runRouteCheck, simulateRoute } from "../../api/diagnostics";
import apiClient from "../../api/client";

vi.mock("../../api/client", () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}));

describe("diagnostics API", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("runRouteCheck", () => {
    it("sends POST /diagnostics/route-check with request body and returns result", async () => {
      const req = { namespace: "default", routeName: "my-route" };
      const mockResponse = {
        route: "my-route",
        namespace: "default",
        status: "healthy",
        checks: [{ name: "backend-valid", status: "pass", message: "All backends are valid" }],
      };
      vi.mocked(apiClient.post).mockResolvedValue({ data: mockResponse });

      const result = await runRouteCheck(req);

      expect(apiClient.post).toHaveBeenCalledWith("/diagnostics/route-check", req);
      expect(result).toEqual(mockResponse);
    });
  });

  describe("simulateRoute", () => {
    it("sends POST /httproutes/:ns/:name/simulate with request body and returns result", async () => {
      const simulateReq = { method: "GET", path: "/api" };
      const mockResponse = {
        matched: true,
        matchedRule: 0,
        matchDetails: [{ ruleIndex: 0, matched: true, reason: "Path prefix matched" }],
        backends: [{ name: "api-svc", namespace: "default", port: 8080, weight: 1 }],
      };
      vi.mocked(apiClient.post).mockResolvedValue({ data: mockResponse });

      const result = await simulateRoute("default", "my-route", simulateReq);

      expect(apiClient.post).toHaveBeenCalledWith("/httproutes/default/my-route/simulate", simulateReq);
      expect(result).toEqual(mockResponse);
    });
  });
});
