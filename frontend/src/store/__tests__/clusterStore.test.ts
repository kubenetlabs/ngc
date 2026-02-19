import { describe, it, expect, beforeEach } from "vitest";
import { useClusterStore, ALL_CLUSTERS } from "../clusterStore";
import type { ManagedCluster } from "@/types/cluster";

const makeCluster = (name: string): ManagedCluster => ({
  name,
  displayName: name,
  region: "us-east-1",
  environment: "prod",
  connected: true,
  edition: "oss",
  default: false,
  agentInstalled: true,
  isLocal: false,
});

describe("clusterStore", () => {
  beforeEach(() => {
    useClusterStore.setState({
      activeCluster: "",
      clusters: [],
    });
  });

  it("initializes with empty state", () => {
    const state = useClusterStore.getState();
    expect(state.activeCluster).toBe("");
    expect(state.clusters).toEqual([]);
  });

  it("sets active cluster", () => {
    useClusterStore.getState().setActiveCluster("cluster-a");
    expect(useClusterStore.getState().activeCluster).toBe("cluster-a");
  });

  it("sets clusters list", () => {
    const clusters = [makeCluster("alpha"), makeCluster("beta")];
    useClusterStore.getState().setClusters(clusters);
    expect(useClusterStore.getState().clusters).toHaveLength(2);
    expect(useClusterStore.getState().clusters[0].name).toBe("alpha");
  });

  it("resets stale activeCluster when cluster no longer exists", () => {
    useClusterStore.setState({ activeCluster: "gone-cluster" });
    const clusters = [makeCluster("alpha"), makeCluster("beta")];
    useClusterStore.getState().setClusters(clusters);
    expect(useClusterStore.getState().activeCluster).toBe("alpha");
  });

  it("keeps activeCluster if it still exists in new list", () => {
    useClusterStore.setState({ activeCluster: "beta" });
    const clusters = [makeCluster("alpha"), makeCluster("beta")];
    useClusterStore.getState().setClusters(clusters);
    expect(useClusterStore.getState().activeCluster).toBe("beta");
  });

  it("keeps ALL_CLUSTERS even when not in cluster list", () => {
    useClusterStore.setState({ activeCluster: ALL_CLUSTERS });
    const clusters = [makeCluster("alpha")];
    useClusterStore.getState().setClusters(clusters);
    expect(useClusterStore.getState().activeCluster).toBe(ALL_CLUSTERS);
  });

  it("keeps empty activeCluster on setClusters", () => {
    useClusterStore.setState({ activeCluster: "" });
    const clusters = [makeCluster("alpha")];
    useClusterStore.getState().setClusters(clusters);
    expect(useClusterStore.getState().activeCluster).toBe("");
  });

  it("isAllClusters returns true when activeCluster is ALL_CLUSTERS", () => {
    useClusterStore.setState({ activeCluster: ALL_CLUSTERS });
    expect(useClusterStore.getState().isAllClusters()).toBe(true);
  });

  it("isAllClusters returns false for a specific cluster", () => {
    useClusterStore.setState({ activeCluster: "my-cluster" });
    expect(useClusterStore.getState().isAllClusters()).toBe(false);
  });

  it("resets to first cluster when list becomes empty and active was set", () => {
    useClusterStore.setState({ activeCluster: "old-cluster" });
    useClusterStore.getState().setClusters([]);
    expect(useClusterStore.getState().activeCluster).toBe("");
  });
});
