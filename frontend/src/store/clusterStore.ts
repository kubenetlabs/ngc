import { create } from "zustand";
import { persist } from "zustand/middleware";

interface ClusterState {
  activeCluster: string; // empty = use default/legacy routes
  setActiveCluster: (name: string) => void;
}

export const useClusterStore = create<ClusterState>()(
  persist(
    (set) => ({
      activeCluster: "",
      setActiveCluster: (name) => set({ activeCluster: name }),
    }),
    { name: "ngf-console-cluster" },
  ),
);
