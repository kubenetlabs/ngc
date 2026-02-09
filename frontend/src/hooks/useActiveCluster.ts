import { useClusterStore } from "@/store/clusterStore";

export function useActiveCluster(): string {
  return useClusterStore((s) => s.activeCluster);
}
