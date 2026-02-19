import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  fetchInferenceStacks,
  fetchInferenceStack,
  createInferenceStack,
  updateInferenceStack,
  deleteInferenceStack,
} from "../../api/inferencestacks";
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

describe("fetchInferenceStacks", () => {
  it("calls GET /inference/stacks and returns data", async () => {
    const mockStacks = [{ name: "stack1", namespace: "ns1" }];
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockStacks });

    const result = await fetchInferenceStacks();

    expect(apiClient.get).toHaveBeenCalledWith("/inference/stacks");
    expect(result).toEqual(mockStacks);
  });
});

describe("fetchInferenceStack", () => {
  it("calls GET /inference/stacks/:namespace/:name and returns data", async () => {
    const mockStack = { name: "stack1", namespace: "ns1" };
    vi.mocked(apiClient.get).mockResolvedValue({ data: mockStack });

    const result = await fetchInferenceStack("ns1", "stack1");

    expect(apiClient.get).toHaveBeenCalledWith("/inference/stacks/ns1/stack1");
    expect(result).toEqual(mockStack);
  });
});

describe("createInferenceStack", () => {
  it("calls POST /inference/stacks with payload and returns data", async () => {
    const payload = { name: "stack1", namespace: "ns1", model: "llama2" };
    const mockResponse = { name: "stack1", namespace: "ns1" };
    vi.mocked(apiClient.post).mockResolvedValue({ data: mockResponse });

    const result = await createInferenceStack(payload as any);

    expect(apiClient.post).toHaveBeenCalledWith("/inference/stacks", payload);
    expect(result).toEqual(mockResponse);
  });
});

describe("updateInferenceStack", () => {
  it("calls PUT /inference/stacks/:namespace/:name with payload and returns data", async () => {
    const payload = { model: "llama3" };
    const mockResponse = { name: "stack1", namespace: "ns1", model: "llama3" };
    vi.mocked(apiClient.put).mockResolvedValue({ data: mockResponse });

    const result = await updateInferenceStack("ns1", "stack1", payload as any);

    expect(apiClient.put).toHaveBeenCalledWith("/inference/stacks/ns1/stack1", payload);
    expect(result).toEqual(mockResponse);
  });
});

describe("deleteInferenceStack", () => {
  it("calls DELETE /inference/stacks/:namespace/:name", async () => {
    vi.mocked(apiClient.delete).mockResolvedValue({ data: undefined });

    await deleteInferenceStack("ns1", "stack1");

    expect(apiClient.delete).toHaveBeenCalledWith("/inference/stacks/ns1/stack1");
  });
});
