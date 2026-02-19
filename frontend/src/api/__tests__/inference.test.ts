import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  fetchInferencePools,
  fetchInferencePool,
  fetchInferenceMetricsSummary,
  fetchPodMetrics,
  fetchEPPDecisions,
  fetchTTFTHistogram,
  fetchTPSThroughput,
  fetchQueueDepthSeries,
  fetchActiveRequestsSeries,
  fetchGPUUtilSeries,
  fetchKVCacheSeries,
  fetchCostEstimate,
  createInferencePool,
  updateInferencePool,
  deleteInferencePool,
  deployInferencePool,
  fetchEPPConfig,
  updateEPPConfig,
  fetchAutoscaling,
  updateAutoscaling,
} from "../../api/inference";
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

describe("fetchInferencePools", () => {
  it("calls GET /inference/pools and returns data", async () => {
    const mockPools = [{ name: "pool1" }, { name: "pool2" }];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockPools });

    const result = await fetchInferencePools();

    expect(apiClient.get).toHaveBeenCalledWith("/inference/pools");
    expect(result).toEqual(mockPools);
  });
});

describe("fetchInferencePool", () => {
  it("calls GET /inference/pools/:name and returns data", async () => {
    const mockPool = { name: "pool1" };
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockPool });

    const result = await fetchInferencePool("pool1");

    expect(apiClient.get).toHaveBeenCalledWith("/inference/pools/pool1");
    expect(result).toEqual(mockPool);
  });
});

describe("fetchInferenceMetricsSummary", () => {
  it("calls GET /inference/metrics/summary with no params when pool is omitted", async () => {
    const mockSummary = { totalRequests: 100 };
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockSummary });

    const result = await fetchInferenceMetricsSummary();

    expect(apiClient.get).toHaveBeenCalledWith("/inference/metrics/summary", { params: {} });
    expect(result).toEqual(mockSummary);
  });

  it("calls GET /inference/metrics/summary with pool param when provided", async () => {
    const mockSummary = { totalRequests: 50 };
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockSummary });

    const result = await fetchInferenceMetricsSummary("pool1");

    expect(apiClient.get).toHaveBeenCalledWith("/inference/metrics/summary", { params: { pool: "pool1" } });
    expect(result).toEqual(mockSummary);
  });
});

describe("fetchPodMetrics", () => {
  it("calls GET /inference/metrics/pods with pool param", async () => {
    const mockPods = [{ pod: "pod1", gpu: 80 }];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockPods });

    const result = await fetchPodMetrics("pool1");

    expect(apiClient.get).toHaveBeenCalledWith("/inference/metrics/pods", { params: { pool: "pool1" } });
    expect(result).toEqual(mockPods);
  });
});

describe("fetchEPPDecisions", () => {
  it("calls GET /inference/metrics/epp-decisions with pool and limit params", async () => {
    const mockDecisions = [{ id: "d1" }];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockDecisions });

    const result = await fetchEPPDecisions("pool1", 10);

    expect(apiClient.get).toHaveBeenCalledWith("/inference/metrics/epp-decisions", {
      params: { pool: "pool1", limit: 10 },
    });
    expect(result).toEqual(mockDecisions);
  });
});

describe("fetchTTFTHistogram", () => {
  it("calls GET /inference/metrics/ttft-histogram/:pool", async () => {
    const mockBuckets = [{ le: 0.1, count: 5 }];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockBuckets });

    const result = await fetchTTFTHistogram("pool1");

    expect(apiClient.get).toHaveBeenCalledWith("/inference/metrics/ttft-histogram/pool1");
    expect(result).toEqual(mockBuckets);
  });
});

describe("fetchTPSThroughput", () => {
  it("calls GET /inference/metrics/tps-throughput/:pool", async () => {
    const mockPoints = [{ ts: "2024-01-01", value: 42 }];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockPoints });

    const result = await fetchTPSThroughput("pool1");

    expect(apiClient.get).toHaveBeenCalledWith("/inference/metrics/tps-throughput/pool1");
    expect(result).toEqual(mockPoints);
  });
});

describe("fetchQueueDepthSeries", () => {
  it("calls GET /inference/metrics/queue-depth/:pool", async () => {
    const mockPoints = [{ ts: "2024-01-01", value: 3 }];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockPoints });

    const result = await fetchQueueDepthSeries("pool1");

    expect(apiClient.get).toHaveBeenCalledWith("/inference/metrics/queue-depth/pool1");
    expect(result).toEqual(mockPoints);
  });
});

describe("fetchActiveRequestsSeries", () => {
  it("calls GET /inference/metrics/active-requests/:pool", async () => {
    const mockPoints = [{ ts: "2024-01-01", value: 7 }];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockPoints });

    const result = await fetchActiveRequestsSeries("pool1");

    expect(apiClient.get).toHaveBeenCalledWith("/inference/metrics/active-requests/pool1");
    expect(result).toEqual(mockPoints);
  });
});

describe("fetchGPUUtilSeries", () => {
  it("calls GET /inference/metrics/gpu-util/:pool", async () => {
    const mockPoints = [{ ts: "2024-01-01", value: 85 }];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockPoints });

    const result = await fetchGPUUtilSeries("pool1");

    expect(apiClient.get).toHaveBeenCalledWith("/inference/metrics/gpu-util/pool1");
    expect(result).toEqual(mockPoints);
  });
});

describe("fetchKVCacheSeries", () => {
  it("calls GET /inference/metrics/kv-cache/:pool", async () => {
    const mockPoints = [{ ts: "2024-01-01", value: 60 }];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockPoints });

    const result = await fetchKVCacheSeries("pool1");

    expect(apiClient.get).toHaveBeenCalledWith("/inference/metrics/kv-cache/pool1");
    expect(result).toEqual(mockPoints);
  });
});

describe("fetchCostEstimate", () => {
  it("calls GET /inference/metrics/cost with pool param", async () => {
    const mockCost = { hourly: 1.5 };
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockCost });

    const result = await fetchCostEstimate("pool1");

    expect(apiClient.get).toHaveBeenCalledWith("/inference/metrics/cost", { params: { pool: "pool1" } });
    expect(result).toEqual(mockCost);
  });
});

describe("createInferencePool", () => {
  it("calls POST /inference/pools with payload", async () => {
    const payload = { name: "new-pool", model: "llama2" };
    const mockResponse = { name: "new-pool" };
    vi.mocked(apiClient.post).mockResolvedValue({ data: mockResponse });

    const result = await createInferencePool(payload as any);

    expect(apiClient.post).toHaveBeenCalledWith("/inference/pools", payload);
    expect(result).toEqual(mockResponse);
  });
});

describe("updateInferencePool", () => {
  it("calls PUT /inference/pools/:name with payload", async () => {
    const payload = { model: "llama3" };
    const mockResponse = { name: "pool1", model: "llama3" };
    vi.mocked(apiClient.put).mockResolvedValue({ data: mockResponse });

    const result = await updateInferencePool("pool1", payload as any);

    expect(apiClient.put).toHaveBeenCalledWith("/inference/pools/pool1", payload);
    expect(result).toEqual(mockResponse);
  });
});

describe("deleteInferencePool", () => {
  it("calls DELETE /inference/pools/:name", async () => {
    const mockResponse = { success: true };
    vi.mocked(apiClient.delete).mockResolvedValue({ data: mockResponse });

    const result = await deleteInferencePool("pool1");

    expect(apiClient.delete).toHaveBeenCalledWith("/inference/pools/pool1");
    expect(result).toEqual(mockResponse);
  });
});

describe("deployInferencePool", () => {
  it("calls POST /inference/pools/:name/deploy", async () => {
    const mockResponse = { status: "deploying" };
    vi.mocked(apiClient.post).mockResolvedValue({ data: mockResponse });

    const result = await deployInferencePool("pool1");

    expect(apiClient.post).toHaveBeenCalledWith("/inference/pools/pool1/deploy");
    expect(result).toEqual(mockResponse);
  });
});

describe("fetchEPPConfig", () => {
  it("calls GET /inference/epp with pool param", async () => {
    const mockConfig = { scheduler: "round-robin" };
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockConfig });

    const result = await fetchEPPConfig("pool1");

    expect(apiClient.get).toHaveBeenCalledWith("/inference/epp", { params: { pool: "pool1" } });
    expect(result).toEqual(mockConfig);
  });
});

describe("updateEPPConfig", () => {
  it("calls PUT /inference/epp with payload", async () => {
    const payload = { scheduler: "least-load" };
    const mockResponse = { scheduler: "least-load" };
    vi.mocked(apiClient.put).mockResolvedValue({ data: mockResponse });

    const result = await updateEPPConfig(payload as any);

    expect(apiClient.put).toHaveBeenCalledWith("/inference/epp", payload);
    expect(result).toEqual(mockResponse);
  });
});

describe("fetchAutoscaling", () => {
  it("calls GET /inference/autoscaling with pool param", async () => {
    const mockConfig = { minReplicas: 1, maxReplicas: 10 };
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockConfig });

    const result = await fetchAutoscaling("pool1");

    expect(apiClient.get).toHaveBeenCalledWith("/inference/autoscaling", { params: { pool: "pool1" } });
    expect(result).toEqual(mockConfig);
  });
});

describe("updateAutoscaling", () => {
  it("calls PUT /inference/autoscaling with payload", async () => {
    const payload = { minReplicas: 2, maxReplicas: 20 };
    const mockResponse = { minReplicas: 2, maxReplicas: 20 };
    vi.mocked(apiClient.put).mockResolvedValue({ data: mockResponse });

    const result = await updateAutoscaling(payload as any);

    expect(apiClient.put).toHaveBeenCalledWith("/inference/autoscaling", payload);
    expect(result).toEqual(mockResponse);
  });
});
