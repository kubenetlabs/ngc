export interface ResourceCounts {
  gateways: number;
  httpRoutes: number;
  inferencePools: number;
  inferenceStacks: number;
  gatewayBundles: number;
  services: number;
  namespaces: number;
}

export interface GPUCapacitySummary {
  totalGPUs: number;
  allocatedGPUs: number;
  gpuTypes?: Record<string, number>;
}

export interface ManagedCluster {
  name: string;
  displayName: string;
  region: string;
  environment: string;
  connected: boolean;
  edition: "oss" | "enterprise" | "unknown";
  default: boolean;
  kubernetesVersion?: string;
  ngfVersion?: string;
  agentInstalled: boolean;
  lastHeartbeat?: string;
  resourceCounts?: ResourceCounts;
  gpuCapacity?: GPUCapacitySummary;
  isLocal: boolean;
}

export interface RegisterClusterPayload {
  name: string;
  displayName: string;
  region: string;
  environment: string;
  kubeconfig: string;
  ngfEdition?: string;
}

export interface ClusterSummary {
  totalClusters: number;
  healthyClusters: number;
  totalGateways: number;
  totalRoutes: number;
  totalGPUs: number;
}

export interface ClusterTestResult {
  connected: boolean;
  kubernetesVersion?: string;
  ngfVersion?: string;
  error?: string;
}

export interface AgentInstallInfo {
  helmCommand: string;
  clusterName: string;
}

export type ClusterPhase =
  | "Pending"
  | "Connecting"
  | "Ready"
  | "Degraded"
  | "Unreachable"
  | "Terminating";
