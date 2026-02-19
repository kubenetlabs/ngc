import { describe, it, expect, beforeEach } from "vitest";
import { useClusterStore, ALL_CLUSTERS } from "@/store/clusterStore";
import apiClient from "../client";

describe("apiClient cluster routing interceptor", () => {
  beforeEach(() => {
    useClusterStore.setState({ activeCluster: "", clusters: [] });
  });

  it("does not prefix when activeCluster is empty", () => {
    const config = { url: "/gateways", headers: {} };
    const interceptor = apiClient.interceptors.request.handlers[0]?.fulfilled;
    expect(interceptor).toBeDefined();
    const result = interceptor!(config);
    expect(result.url).toBe("/gateways");
  });

  it("prefixes URL with cluster name when activeCluster is set", () => {
    useClusterStore.setState({ activeCluster: "prod-us" });
    const config = { url: "/gateways", headers: {} };
    const interceptor = apiClient.interceptors.request.handlers[0]?.fulfilled;
    const result = interceptor!(config);
    expect(result.url).toBe("/clusters/prod-us/gateways");
  });

  it("does not prefix when activeCluster is ALL_CLUSTERS", () => {
    useClusterStore.setState({ activeCluster: ALL_CLUSTERS });
    const config = { url: "/gateways", headers: {} };
    const interceptor = apiClient.interceptors.request.handlers[0]?.fulfilled;
    const result = interceptor!(config);
    expect(result.url).toBe("/gateways");
  });

  it("does not prefix /clusters URLs", () => {
    useClusterStore.setState({ activeCluster: "prod-us" });
    const config = { url: "/clusters/summary", headers: {} };
    const interceptor = apiClient.interceptors.request.handlers[0]?.fulfilled;
    const result = interceptor!(config);
    expect(result.url).toBe("/clusters/summary");
  });

  it("does not prefix /global URLs", () => {
    useClusterStore.setState({ activeCluster: "prod-us" });
    const config = { url: "/global/gateways", headers: {} };
    const interceptor = apiClient.interceptors.request.handlers[0]?.fulfilled;
    const result = interceptor!(config);
    expect(result.url).toBe("/global/gateways");
  });

  it("prefixes inference URLs with cluster name", () => {
    useClusterStore.setState({ activeCluster: "gpu-west" });
    const config = { url: "/inference/pools", headers: {} };
    const interceptor = apiClient.interceptors.request.handlers[0]?.fulfilled;
    const result = interceptor!(config);
    expect(result.url).toBe("/clusters/gpu-west/inference/pools");
  });
});
