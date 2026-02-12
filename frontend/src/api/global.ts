import apiClient from "./client";

export interface ClusterGateway {
  clusterName: string;
  clusterRegion: string;
  gateway: {
    name: string;
    namespace: string;
    className: string;
    listeners: { name: string; port: number; protocol: string }[];
    status: string;
    addresses?: string[];
  };
}

export interface ClusterRoute {
  clusterName: string;
  clusterRegion: string;
  route: {
    name: string;
    namespace: string;
    hostnames?: string[];
    parentRefs: { name: string; namespace?: string }[];
    status: string;
  };
}

export interface GPUClusterCapacity {
  clusterName: string;
  clusterRegion: string;
  totalGPUs: number;
  allocatedGPUs: number;
  gpuTypes?: Record<string, number>;
}

export interface GlobalGPUCapacity {
  totalGPUs: number;
  allocatedGPUs: number;
  clusters: GPUClusterCapacity[];
}

export async function fetchGlobalGateways(): Promise<ClusterGateway[]> {
  const { data } = await apiClient.get("/global/gateways");
  return data;
}

export async function fetchGlobalRoutes(): Promise<ClusterRoute[]> {
  const { data } = await apiClient.get("/global/routes");
  return data;
}

export async function fetchGlobalGPUCapacity(): Promise<GlobalGPUCapacity> {
  const { data } = await apiClient.get("/global/gpu-capacity");
  return data;
}
