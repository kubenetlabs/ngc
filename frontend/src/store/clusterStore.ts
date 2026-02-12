import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { ManagedCluster } from "@/types/cluster";

export const ALL_CLUSTERS = "__all__";

interface ClusterState {
  activeCluster: string; // empty = use default/legacy routes, "__all__" = global view
  clusters: ManagedCluster[];
  setClusters: (clusters: ManagedCluster[]) => void;
  setActiveCluster: (name: string) => void;
  isAllClusters: () => boolean;
}

export const useClusterStore = create<ClusterState>()(
  persist(
    (set, get) => ({
      activeCluster: "",
      clusters: [],
      setClusters: (clusters) => {
        const state = get();
        const names = clusters.map((c) => c.name);
        // Reset stale activeCluster if it no longer exists (but keep "" and ALL_CLUSTERS).
        if (
          state.activeCluster &&
          state.activeCluster !== ALL_CLUSTERS &&
          !names.includes(state.activeCluster)
        ) {
          set({ clusters, activeCluster: names[0] ?? "" });
        } else {
          set({ clusters });
        }
      },
      setActiveCluster: (name) => set({ activeCluster: name }),
      isAllClusters: () => get().activeCluster === ALL_CLUSTERS,
    }),
    { name: "ngf-console-cluster" },
  ),
);
