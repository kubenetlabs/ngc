export interface GatewayClass {
  name: string;
  controllerName: string;
  description?: string;
  parametersRef?: {
    group: string;
    kind: string;
    name: string;
    namespace?: string;
  };
}

export interface Listener {
  name: string;
  hostname?: string;
  port: number;
  protocol: "HTTP" | "HTTPS" | "TLS" | "TCP" | "UDP";
  tls?: {
    mode: "Terminate" | "Passthrough";
    certificateRefs: CertificateRef[];
  };
  allowedRoutes?: {
    namespaces?: { from: "Same" | "All" | "Selector"; selector?: Record<string, string> };
    kinds?: { group: string; kind: string }[];
  };
}

export interface CertificateRef {
  group?: string;
  kind?: string;
  name: string;
  namespace?: string;
}

export interface GatewayStatus {
  conditions: Condition[];
  listeners: ListenerStatus[];
  addresses: GatewayAddress[];
}

export interface ListenerStatus {
  name: string;
  supportedKinds: { group: string; kind: string }[];
  attachedRoutes: number;
  conditions: Condition[];
}

export interface GatewayAddress {
  type: "IPAddress" | "Hostname";
  value: string;
}

export interface Condition {
  type: string;
  status: "True" | "False" | "Unknown";
  reason: string;
  message: string;
  lastTransitionTime: string;
}

export interface Gateway {
  name: string;
  namespace: string;
  gatewayClassName: string;
  listeners: Listener[];
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  status?: GatewayStatus;
  createdAt: string;
}
