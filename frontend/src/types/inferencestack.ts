import type { GPUType, EPPStrategy } from "./inference";

export interface InferenceStack {
  name: string;
  namespace: string;
  modelName: string;
  modelVersion?: string;
  servingBackend: "triton" | "vllm" | "tgi";
  pool: InferenceStackPool;
  epp?: InferenceStackEPP;
  phase?: string;
  children?: ChildStatus[];
  observedSpecHash?: string;
  lastReconciledAt?: string;
  createdAt: string;
}

export interface InferenceStackPool {
  gpuType: GPUType;
  gpuCount: number;
  replicas: number;
  minReplicas: number;
  maxReplicas: number;
  selector?: Record<string, string>;
}

export interface InferenceStackEPP {
  strategy: EPPStrategy;
  weights?: {
    queueDepth: number;
    kvCache: number;
    prefixAffinity: number;
  };
}

export interface ChildStatus {
  kind: string;
  name: string;
  ready: boolean;
  message?: string;
}

export interface CreateInferenceStackPayload {
  name: string;
  namespace: string;
  modelName: string;
  modelVersion?: string;
  servingBackend: "triton" | "vllm" | "tgi";
  pool: InferenceStackPool;
  epp?: InferenceStackEPP;
}
