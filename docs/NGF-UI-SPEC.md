# NGINX Gateway Fabric UI — Technical Specification

## Project: NGF Console

### Version: 0.1.0-alpha
### Author: Dan / F5 Product Incubation
### Date: February 2026

---

## 1. Executive Summary

NGF Console is a web-based management platform and Kubernetes operator for NGINX Gateway Fabric — purpose-built for the AI inference era. It is the first gateway management UI with native support for Kubernetes Gateway Inference Extensions, providing GPU-aware observability, InferencePool lifecycle management, Endpoint Picker (EPP) decision visualization, and intelligent autoscaling configuration for LLM serving infrastructure.

**NGF Console is NOT a kubectl proxy.** It is a declarative control plane built on the Kubernetes operator pattern using controller-runtime. The UI writes parent Custom Resources (InferenceStack, GatewayBundle, MigrationPlan, DistributedCloudPublish), and the operator continuously reconciles all child resources — detecting drift, self-healing, and reporting aggregated status. This makes NGF Console GitOps-native: parent CRDs go into Git, ArgoCD/Flux deploys them, and the operator handles everything downstream.

Beyond inference, NGF Console provides comprehensive traffic management, observability, policy configuration, certificate lifecycle management, troubleshooting workflows, NGINX Ingress Controller migration tooling, and automated F5 Distributed Cloud integration. It supports both OSS and Enterprise NGF editions, with enterprise features gracefully degraded (greyed out) when running against OSS.

The project is containerized, installable via Helm, kubectl manifests, and docker-compose, and designed for production Kubernetes deployments.

---

## 2. Architecture Overview

### 2.1 High-Level Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                         NGF Console UI                            │
│                  (React + TypeScript + Tailwind)                   │
├──────────────────────────────────────────────────────────────────┤
│                    API Server (Thin BFF Layer)                     │
│           (Go REST + WebSocket — CRD CRUD + Read Aggregation)     │
│     Reads: Prometheus, ClickHouse, Triton/metrics, DCGM, K8s     │
│     Writes: Parent CRDs only (InferenceStack, GatewayBundle, etc) │
├──────────────────────────────────────────────────────────────────┤
│                                                                    │
│  ┌─────────────────────────────────────────────────────────────┐  │
│  │                NGF Console Operator (controller-runtime)     │  │
│  │                                                              │  │
│  │  ┌──────────────┐ ┌──────────────┐ ┌──────────────────────┐ │  │
│  │  │ Inference     │ │ Gateway      │ │ DistributedCloud     │ │  │
│  │  │ Stack         │ │ Bundle       │ │ Publish              │ │  │
│  │  │ Controller    │ │ Controller   │ │ Controller           │ │  │
│  │  └──────┬───────┘ └──────┬───────┘ └──────────┬───────────┘ │  │
│  │         │                │                     │              │  │
│  │  ┌──────┴───────┐ ┌──────┴───────┐ ┌──────────┴───────────┐ │  │
│  │  │ Migration    │ │ Certificate  │ │ Drift Detection      │ │  │
│  │  │ Plan         │ │ Bundle       │ │ + Self-Healing       │ │  │
│  │  │ Controller   │ │ Controller   │ │ Engine               │ │  │
│  │  └──────────────┘ └──────────────┘ └──────────────────────┘ │  │
│  └─────────────────────────────────────────────────────────────┘  │
│         │ Reconciles Child Resources                               │
│         ▼                                                          │
│  ┌─────────────────────────────────────────────────────────────┐  │
│  │                  Kubernetes API Server                        │  │
│  │   Gateway, HTTPRoute, InferencePool, HPA, KEDA ScaledObject  │  │
│  │   Services, Secrets, ConfigMaps, DaemonSets, Deployments     │  │
│  └─────────────────────────────────────────────────────────────┘  │
│                                                                    │
├──────────┬──────────┬───────────┬──────────┬────────┬────────────┤
│ K8s API  │ClickHouse│ Postgres/ │Prometheus │ F5 XC  │ Triton     │
│ (read)   │  Client  │ SQLite    │ + DCGM   │ API    │ /metrics   │
└──────────┴──────────┴───────────┴──────────┴────────┴────────────┘
```

**Key Architectural Principle: The operator IS the control plane. The API server is a thin read/write layer.**

```
WRITE PATH (User makes a change):
  UI Form → API Server → Create/Update parent CRD → Done
  (API server does NOT directly create child K8s resources)

RECONCILIATION PATH (Operator manages state):
  Parent CRD created/updated
    → Operator detects change via Watch
    → Generates desired child resources
    → Compares with actual cluster state
    → Creates/updates/deletes child resources
    → Aggregates status from all children
    → Updates parent CRD .status

READ PATH (UI displays data):
  UI → API Server → Read parent CRD .status (aggregated health)
  UI → API Server → Read Prometheus/ClickHouse/DCGM (metrics)
  UI → API Server → Read K8s resources directly (topology, events)
```

### 2.2 Component Inventory

| Component | Technology | Purpose |
|-----------|-----------|---------|
| **Frontend** | React 18, TypeScript, Tailwind CSS, Vite | SPA UI — displays state, writes parent CRDs via API |
| **API Server** | Go | Thin BFF: REST + WebSocket, CRD CRUD, read aggregation from Prometheus/ClickHouse/DCGM. **Does NOT directly manage child K8s resources.** |
| **NGF Console Operator** | Go, controller-runtime v0.17+ | **The brain.** Watches parent CRDs, reconciles all child K8s resources, detects drift, self-heals, reports aggregated status. Runs as a Deployment with leader election. |
| **Configuration DB** | PostgreSQL (production) / SQLite (single-node) | User prefs, audit log, alert rules, saved views. NOT for K8s state — that lives in CRDs. |
| **Analytics DB** | ClickHouse | Access log + inference telemetry analytics |
| **Telemetry Pipeline** | OpenTelemetry Collector | NGINX → OTel → ClickHouse; Triton → OTel → ClickHouse |
| **Inference Metrics Collector** | Go sidecar/agent | Scrapes Triton /metrics, EPP state, DCGM; publishes to OTel + Custom Metrics API |
| **Ingress Migration Tool** | Go CLI + UI integration | Converts NGINX Ingress Controller configs → Gateway API |

### 2.3 Container Images

```
ngf-console-frontend:0.1.0     # Nginx-served React SPA
ngf-console-api:0.1.0          # Thin API server (BFF)
ngf-console-operator:0.1.0     # Kubernetes operator (controller-runtime) — THE CONTROL PLANE
ngf-console-migration:0.1.0    # Ingress migration tool (init container or Job)
```

> **Note for Claude Code:** The old `ngf-console-controller` image is replaced by `ngf-console-operator`. This is not just a rename — it's an architectural shift. The operator owns ALL resource lifecycle management. The API server is deliberately thin.

### 2.4 Installation Methods

**Helm Chart** (primary):
```bash
helm install ngf-console oci://registry.f5.com/ngf-console \
  --namespace ngf-system \
  --set database.type=postgresql \
  --set clickhouse.enabled=true \
  --set xc.enabled=true \
  --set xc.tenantUrl=https://tenant.console.ves.volterra.io \
  --set xc.apiTokenSecretRef=xc-api-token \
  --set ngf.edition=enterprise  # or "oss"
```

**kubectl manifests**:
```bash
kubectl apply -f https://raw.githubusercontent.com/f5/ngf-console/main/deploy/manifests/install.yaml
```

**docker-compose** (dev/demo):
```bash
docker-compose -f deploy/docker-compose/docker-compose.yaml up -d
```

---

## 2.5 Operator Architecture — Declarative Control Plane

> **CRITICAL FOR CLAUDE CODE:** This section defines the core architectural pattern for NGF Console. Every write operation from the UI flows through parent CRDs, NOT through direct Kubernetes API calls. The operator is the single source of truth for cluster state management. If you find yourself writing code in the API server that does `client.Create()` for a child resource like an HTTPRoute, Gateway, HPA, or KEDA ScaledObject — STOP. That belongs in the operator.

### 2.5.1 Why an Operator (Not a kubectl Proxy)

The imperative pattern (UI → API → direct K8s resource creation) fails for this product:

**Problem 1: Partial State Failures.** The InferencePool wizard generates 6 resources. If the API server creates 4 and crashes, you have an inconsistent cluster with no automatic recovery. The operator pattern creates ONE parent CRD and continuously reconciles all children — if it crashes mid-reconciliation, it picks up where it left off on restart.

**Problem 2: Drift Detection.** If someone uses kubectl to delete a KEDA ScaledObject that the console created, the imperative API has no idea. The console dashboard still shows "autoscaling configured" while nothing is actually scaling. The operator watches for changes and self-heals.

**Problem 3: Status Aggregation.** With the imperative pattern, the UI must individually query every child resource to determine composite health. With the operator, the parent CRD's `.status` is the single source of truth with aggregated conditions.

**Problem 4: GitOps Compatibility.** Platform teams use ArgoCD/Flux. They want to commit a YAML file and have the cluster converge. Parent CRDs are the ideal GitOps primitive — ArgoCD manages the CRDs, the operator manages everything else.

### 2.5.2 CRD Hierarchy

```
ngf-console.f5.com/v1alpha1
├── InferenceStack          # Parent for inference workloads
│   ├── InferencePool       (child — Gateway Inference Extension CRD)
│   ├── EPP Configuration   (child — ConfigMap or dedicated CRD)
│   ├── HTTPRoute           (child — Gateway API)
│   ├── KEDA ScaledObject   (child — or HPA)
│   ├── PrometheusAdapter   (child — ConfigMap, if using HPA)
│   └── DCGM DaemonSet     (child — if not already present)
│
├── GatewayBundle           # Parent for gateway + policies
│   ├── Gateway             (child — Gateway API)
│   ├── NginxProxy          (child — NGF Enterprise)
│   ├── APPolicy            (child — NGF Enterprise WAF)
│   ├── SnippetsFilter      (child — NGF Enterprise)
│   └── Secret (TLS)        (child — if TLS inline cert provided)
│
├── DistributedCloudPublish # XC integration (already defined, extend)
│   ├── XC HTTP Load Balancer (external — F5 XC API)
│   ├── XC Origin Pool       (external — F5 XC API)
│   └── XC WAF/Bot/DDoS      (external — F5 XC API)
│
├── MigrationPlan           # Parent for KIC → NGF migration
│   ├── Gateway             (child)
│   ├── HTTPRoute[]         (children)
│   ├── Policy[]            (children — rate limit, TLS, etc.)
│   └── Secret[]            (children — TLS certs)
│
└── CertificateBundle       # Parent for cert lifecycle
    ├── Secret (TLS)        (child)
    ├── cert-manager Certificate (child — if cert-manager present)
    └── AlertRule            (child — in config DB, not K8s)
```

### 2.5.3 CRD Definitions — Go Types

> **Claude Code Implementation Note:** Use `controller-gen` from controller-tools to generate CRD YAML from these Go types. All types go in `operator/api/v1alpha1/`. Run `make generate-crds` to produce the YAML in `operator/config/crd/bases/`.

#### InferenceStack CRD

```go
// operator/api/v1alpha1/inferencestack_types.go
package v1alpha1

import (
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    corev1 "k8s.io/api/core/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Model",type=string,JSONPath=`.spec.model.name`
// +kubebuilder:printcolumn:name="Backend",type=string,JSONPath=`.spec.model.backend`
// +kubebuilder:printcolumn:name="Replicas",type=string,JSONPath=`.status.currentReplicas`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type InferenceStack struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`
    Spec              InferenceStackSpec   `json:"spec,omitempty"`
    Status            InferenceStackStatus `json:"status,omitempty"`
}

type InferenceStackSpec struct {
    // Model serving configuration
    Model ModelSpec `json:"model"`

    // InferencePool configuration
    Pool PoolSpec `json:"pool"`

    // EPP routing configuration
    EPP EPPSpec `json:"epp"`

    // Autoscaling configuration
    Autoscaling *AutoscalingSpec `json:"autoscaling,omitempty"`

    // Gateway attachment
    Gateway GatewayAttachmentSpec `json:"gateway"`

    // Optional: Publish to F5 Distributed Cloud
    // +optional
    DistributedCloud *XCInferenceSpec `json:"distributedCloud,omitempty"`
}

type ModelSpec struct {
    // Serving backend: triton, vllm, tgi
    Backend string `json:"backend"`
    // Model identifier (e.g., "llama-3-70b")
    Name string `json:"name"`
    // Container image for the model server
    Image string `json:"image"`
    // GPU type (A100, H100, L40S, T4)
    GPUType string `json:"gpuType"`
    // GPUs per pod
    GPUsPerPod int `json:"gpusPerPod"`
}

type PoolSpec struct {
    // Number of initial replicas
    Replicas int32 `json:"replicas"`
    // Min replicas for autoscaling
    MinReplicas int32 `json:"minReplicas"`
    // Max replicas for autoscaling
    MaxReplicas int32 `json:"maxReplicas"`
    // Pod selector (match existing deployment)
    // +optional
    Selector *metav1.LabelSelector `json:"selector,omitempty"`
    // Node affinity for GPU nodes
    // +optional
    NodeSelector map[string]string `json:"nodeSelector,omitempty"`
    // Tolerations for GPU taints
    // +optional
    Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
    // Resource requests per pod
    Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

type EPPSpec struct {
    // Metric scraping interval
    // +kubebuilder:default="5s"
    ScrapeInterval string `json:"scrapeInterval,omitempty"`
    // Triton /metrics endpoint path
    // +kubebuilder:default="/metrics"
    MetricsPath string `json:"metricsPath,omitempty"`
    // Routing strategy: least_queue, best_kv_cache, prefix_affinity, composite
    // +kubebuilder:default="least_queue"
    // +kubebuilder:validation:Enum=least_queue;best_kv_cache;prefix_affinity;composite
    Strategy string `json:"strategy,omitempty"`
    // Weights for composite strategy (must sum to 1.0)
    // +optional
    CompositeWeights *CompositeWeights `json:"compositeWeights,omitempty"`
    // Enable DCGM GPU metrics integration
    // +kubebuilder:default=true
    DCGMEnabled bool `json:"dcgmEnabled,omitempty"`
}

type CompositeWeights struct {
    QueueDepth  float64 `json:"queueDepth"`
    KVCache     float64 `json:"kvCache"`
    PrefixCache float64 `json:"prefixCache"`
}

type AutoscalingSpec struct {
    // Backend: keda or hpa
    // +kubebuilder:default="keda"
    // +kubebuilder:validation:Enum=keda;hpa
    Backend string `json:"backend,omitempty"`
    // Queue depth threshold for scale-up
    QueueDepthThreshold *ThresholdSpec `json:"queueDepthThreshold,omitempty"`
    // KV-cache utilization threshold for scale-up
    KVCacheThreshold *ThresholdSpec `json:"kvCacheThreshold,omitempty"`
    // GPU utilization threshold (DCGM safety net)
    GPUUtilThreshold *ThresholdSpec `json:"gpuUtilThreshold,omitempty"`
    // Cool-down period after scale event
    // +kubebuilder:default="300s"
    CooldownPeriod string `json:"cooldownPeriod,omitempty"`
}

type ThresholdSpec struct {
    Value    float64 `json:"value"`
    Duration string  `json:"duration"` // e.g., "60s"
}

type GatewayAttachmentSpec struct {
    // Reference to existing Gateway, or create new
    // +optional
    GatewayRef *GatewayReference `json:"gatewayRef,omitempty"`
    // If no gatewayRef, create a new listener on this gateway
    // +optional
    CreateListener *ListenerSpec `json:"createListener,omitempty"`
    // HTTPRoute matching configuration
    Route RouteSpec `json:"route"`
}

type GatewayReference struct {
    Name      string `json:"name"`
    Namespace string `json:"namespace,omitempty"`
}

type ListenerSpec struct {
    Port     int32  `json:"port"`
    Protocol string `json:"protocol,omitempty"` // HTTP, HTTPS
    Hostname string `json:"hostname,omitempty"`
}

type RouteSpec struct {
    // Path prefix for inference endpoint
    PathPrefix string `json:"pathPrefix"`
    // Request timeout (inference is long-running)
    // +kubebuilder:default="300s"
    Timeout string `json:"timeout,omitempty"`
    // Enable SSE/streaming for token-by-token output
    // +kubebuilder:default=true
    StreamingEnabled bool `json:"streamingEnabled,omitempty"`
}

type XCInferenceSpec struct {
    Tenant           string `json:"tenant"`
    Namespace        string `json:"namespace"`
    WafPolicy        string `json:"wafPolicy,omitempty"`
    BotDefense       bool   `json:"botDefense,omitempty"`
    DDoSProtection   bool   `json:"ddosProtection,omitempty"`
    PerTenantRateLimit *RateLimitSpec `json:"perTenantRateLimit,omitempty"`
    PublicHostname   string `json:"publicHostname"`
    TLSMode          string `json:"tlsMode,omitempty"` // managed, bringYourOwn
}

type RateLimitSpec struct {
    RequestsPerMinute int `json:"requestsPerMinute"`
    TokensPerMinute   int `json:"tokensPerMinute,omitempty"`
}

// --- STATUS ---

type InferenceStackStatus struct {
    // High-level phase: Pending, Provisioning, Ready, Degraded, Error
    // +kubebuilder:validation:Enum=Pending;Provisioning;Ready;Degraded;Error
    Phase string `json:"phase,omitempty"`

    // Current replica count
    CurrentReplicas int32 `json:"currentReplicas,omitempty"`
    // Desired replica count (from autoscaler)
    DesiredReplicas int32 `json:"desiredReplicas,omitempty"`

    // Child resource statuses
    Children ChildResourceStatuses `json:"children,omitempty"`

    // Conditions (standard K8s pattern)
    // +listType=map
    // +listMapKey=type
    Conditions []metav1.Condition `json:"conditions,omitempty"`

    // Hash of the spec used to generate current children (for drift detection)
    ObservedSpecHash string `json:"observedSpecHash,omitempty"`

    // Last time the operator successfully reconciled all children
    LastReconciledAt *metav1.Time `json:"lastReconciledAt,omitempty"`

    // XC publish status (if distributedCloud is configured)
    XCStatus *XCPublishStatus `json:"xcStatus,omitempty"`
}

type ChildResourceStatuses struct {
    InferencePool ChildStatus `json:"inferencePool,omitempty"`
    EPP           ChildStatus `json:"epp,omitempty"`
    HTTPRoute     ChildStatus `json:"httpRoute,omitempty"`
    Autoscaler    ChildStatus `json:"autoscaler,omitempty"`
    DCGMExporter  ChildStatus `json:"dcgmExporter,omitempty"`
}

type ChildStatus struct {
    // Name of the child resource
    Name string `json:"name,omitempty"`
    // Namespace of the child resource
    Namespace string `json:"namespace,omitempty"`
    // Ready or not
    Ready bool `json:"ready"`
    // Human-readable message
    Message string `json:"message,omitempty"`
    // Last time this child was synced
    LastSyncedAt *metav1.Time `json:"lastSyncedAt,omitempty"`
}

type XCPublishStatus struct {
    State           string `json:"state,omitempty"` // Pending, Published, Error
    PublicEndpoint  string `json:"publicEndpoint,omitempty"`
    LoadBalancerName string `json:"loadBalancerName,omitempty"`
    LastSyncTime    *metav1.Time `json:"lastSyncTime,omitempty"`
}
```

**Example InferenceStack YAML (what a user or ArgoCD would deploy):**

```yaml
apiVersion: ngf-console.f5.com/v1alpha1
kind: InferenceStack
metadata:
  name: llama3-production
  namespace: ml-serving
spec:
  model:
    backend: triton
    name: llama-3-70b
    image: nvcr.io/nvidia/tritonserver:24.01-trtllm-python-py3
    gpuType: H100
    gpusPerPod: 1
  pool:
    replicas: 3
    minReplicas: 2
    maxReplicas: 10
    nodeSelector:
      nvidia.com/gpu.product: H100
    tolerations:
      - key: nvidia.com/gpu
        operator: Exists
        effect: NoSchedule
    resources:
      requests:
        nvidia.com/gpu: "1"
        memory: 80Gi
      limits:
        nvidia.com/gpu: "1"
        memory: 80Gi
  epp:
    strategy: composite
    compositeWeights:
      queueDepth: 0.4
      kvCache: 0.35
      prefixCache: 0.25
    dcgmEnabled: true
  autoscaling:
    backend: keda
    queueDepthThreshold:
      value: 5
      duration: "60s"
    kvCacheThreshold:
      value: 80
      duration: "120s"
    gpuUtilThreshold:
      value: 90
      duration: "120s"
    cooldownPeriod: "300s"
  gateway:
    gatewayRef:
      name: inference-gateway
      namespace: ml-serving
    route:
      pathPrefix: /v1/models/llama3
      timeout: "300s"
      streamingEnabled: true
  distributedCloud:
    tenant: mycompany
    namespace: production
    wafPolicy: llm-waf-policy
    botDefense: true
    ddosProtection: true
    perTenantRateLimit:
      requestsPerMinute: 100
      tokensPerMinute: 50000
    publicHostname: llm.example.com
    tlsMode: managed
```

**What the operator creates from this single CRD:**

```
InferenceStack "llama3-production" (parent)
  │
  ├─ InferencePool "llama3-production-pool"
  │    (Gateway Inference Extension CRD — the core routing primitive)
  │
  ├─ EPP ConfigMap "llama3-production-epp-config"
  │    (routing strategy, weights, scrape interval)
  │
  ├─ HTTPRoute "llama3-production-route"
  │    (parentRef → inference-gateway, path → /v1/models/llama3)
  │
  ├─ KEDA ScaledObject "llama3-production-scaler"
  │    (triggers: queue_depth > 5, kv_cache > 80%, gpu_util > 90%)
  │
  ├─ Prometheus Adapter ConfigMap (if needed for custom metrics bridge)
  │
  ├─ DCGM DaemonSet (if not already present on GPU nodes)
  │    (only created once per cluster, shared across InferenceStacks)
  │
  └─ DistributedCloudPublish "llama3-production-xc"
       (XC HTTP LB + Origin Pool + WAF + rate limiting)
```

#### GatewayBundle CRD

```go
// operator/api/v1alpha1/gatewaybundle_types.go

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Class",type=string,JSONPath=`.spec.gatewayClassName`
// +kubebuilder:printcolumn:name="Listeners",type=integer,JSONPath=`.status.listenerCount`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
type GatewayBundle struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`
    Spec              GatewayBundleSpec   `json:"spec,omitempty"`
    Status            GatewayBundleStatus `json:"status,omitempty"`
}

type GatewayBundleSpec struct {
    // GatewayClass to use
    GatewayClassName string `json:"gatewayClassName"`

    // Gateway listeners
    Listeners []BundleListenerSpec `json:"listeners"`

    // Allowed route namespaces
    // +optional
    AllowedRoutes *AllowedRoutesSpec `json:"allowedRoutes,omitempty"`

    // NginxProxy configuration (Enterprise)
    // +optional
    NginxProxy *NginxProxySpec `json:"nginxProxy,omitempty"`

    // WAF policy (Enterprise)
    // +optional
    WAF *WAFPolicySpec `json:"waf,omitempty"`

    // SnippetsFilters (Enterprise)
    // +optional
    Snippets []SnippetSpec `json:"snippets,omitempty"`

    // Infrastructure settings
    // +optional
    Infrastructure *InfrastructureSpec `json:"infrastructure,omitempty"`
}

type BundleListenerSpec struct {
    Name     string `json:"name"`
    Port     int32  `json:"port"`
    Protocol string `json:"protocol"`
    Hostname string `json:"hostname,omitempty"`
    // Inline TLS configuration
    // +optional
    TLS *TLSSpec `json:"tls,omitempty"`
}

type TLSSpec struct {
    Mode string `json:"mode"` // Terminate, Passthrough
    // Reference to existing secret
    // +optional
    CertificateRef *SecretReference `json:"certificateRef,omitempty"`
    // Inline certificate (operator creates Secret)
    // +optional
    InlineCert *InlineCertSpec `json:"inlineCert,omitempty"`
}

type SecretReference struct {
    Name      string `json:"name"`
    Namespace string `json:"namespace,omitempty"`
}

type InlineCertSpec struct {
    Certificate string `json:"certificate"` // PEM
    PrivateKey  string `json:"privateKey"`  // PEM
}

type NginxProxySpec struct {
    // Worker processes
    WorkerProcesses *int `json:"workerProcesses,omitempty"`
    // Worker connections
    WorkerConnections *int `json:"workerConnections,omitempty"`
    // Additional NginxProxy fields as needed
    // Stored as JSON to support evolving NGF NginxProxy API
    AdditionalConfig map[string]interface{} `json:"additionalConfig,omitempty"`
}

type WAFPolicySpec struct {
    // APPolicy name (reference existing) or inline
    PolicyRef string `json:"policyRef,omitempty"`
    // Violation rating threshold (1-5)
    ViolationRatingThreshold *int `json:"violationRatingThreshold,omitempty"`
    // Enforcement mode
    // +kubebuilder:validation:Enum=enforce;log-only
    Mode string `json:"mode,omitempty"`
    // Bot defense
    BotDefense bool `json:"botDefense,omitempty"`
}

type SnippetSpec struct {
    // Context: http, server, location
    Context string `json:"context"`
    // The NGINX config snippet
    Content string `json:"content"`
}

type InfrastructureSpec struct {
    Replicas  *int32                      `json:"replicas,omitempty"`
    Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}

type AllowedRoutesSpec struct {
    Namespaces string `json:"namespaces"` // Same, All, Selector
    // +optional
    Selector *metav1.LabelSelector `json:"selector,omitempty"`
}

type GatewayBundleStatus struct {
    Phase         string             `json:"phase,omitempty"` // Pending, Ready, Degraded, Error
    ListenerCount int                `json:"listenerCount,omitempty"`
    Children      GatewayBundleChildren `json:"children,omitempty"`
    Conditions    []metav1.Condition `json:"conditions,omitempty"`
    ObservedSpecHash string          `json:"observedSpecHash,omitempty"`
    LastReconciledAt *metav1.Time    `json:"lastReconciledAt,omitempty"`
}

type GatewayBundleChildren struct {
    Gateway      ChildStatus `json:"gateway,omitempty"`
    NginxProxy   ChildStatus `json:"nginxProxy,omitempty"`
    WAFPolicy    ChildStatus `json:"wafPolicy,omitempty"`
    Snippets     []ChildStatus `json:"snippets,omitempty"`
    TLSSecrets   []ChildStatus `json:"tlsSecrets,omitempty"`
}
```

**Example GatewayBundle YAML:**

```yaml
apiVersion: ngf-console.f5.com/v1alpha1
kind: GatewayBundle
metadata:
  name: production-gateway
  namespace: gateway-system
spec:
  gatewayClassName: nginx
  listeners:
    - name: http
      port: 80
      protocol: HTTP
    - name: https
      port: 443
      protocol: HTTPS
      hostname: "*.example.com"
      tls:
        mode: Terminate
        certificateRef:
          name: wildcard-tls
  allowedRoutes:
    namespaces: Selector
    selector:
      matchLabels:
        gateway-access: "true"
  waf:
    mode: enforce
    violationRatingThreshold: 3
    botDefense: true
  nginxProxy:
    workerProcesses: 4
    workerConnections: 8192
  infrastructure:
    replicas: 3
```

#### MigrationPlan CRD

```go
// operator/api/v1alpha1/migrationplan_types.go

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Resources",type=integer,JSONPath=`.status.totalResources`
// +kubebuilder:printcolumn:name="Applied",type=integer,JSONPath=`.status.appliedResources`
type MigrationPlan struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`
    Spec              MigrationPlanSpec   `json:"spec,omitempty"`
    Status            MigrationPlanStatus `json:"status,omitempty"`
}

type MigrationPlanSpec struct {
    // What to migrate
    Source MigrationSource `json:"source"`

    // Generated Gateway API resources (populated by analysis, editable by user)
    Resources []MigrationResource `json:"resources,omitempty"`

    // Migration strategy
    Strategy MigrationStrategy `json:"strategy,omitempty"`

    // Whether to actually apply (set to true when user confirms)
    // +kubebuilder:default=false
    Apply bool `json:"apply"`

    // Rollback: set to true to undo the migration
    // +kubebuilder:default=false
    Rollback bool `json:"rollback"`
}

type MigrationSource struct {
    // Source type: ingress, virtualserver, nginx-conf
    Type string `json:"type"`
    // Source resources as YAML (embedded)
    SourceYAML string `json:"sourceYAML,omitempty"`
    // Or reference to source cluster
    SourceCluster *ClusterReference `json:"sourceCluster,omitempty"`
    // Namespace filter
    Namespaces []string `json:"namespaces,omitempty"`
}

type ClusterReference struct {
    KubeconfigSecretRef string `json:"kubeconfigSecretRef"`
}

type MigrationResource struct {
    // Original source resource identifier
    SourceRef string `json:"sourceRef"`
    // Generated Gateway API resource (as YAML string)
    GeneratedYAML string `json:"generatedYAML"`
    // Kind: Gateway, HTTPRoute, RateLimitPolicy, etc.
    Kind string `json:"kind"`
    // Confidence: high, medium, low
    Confidence string `json:"confidence"`
    // Warnings
    Warnings []string `json:"warnings,omitempty"`
    // Whether this resource is enabled for migration
    // +kubebuilder:default=true
    Enabled bool `json:"enabled"`
}

type MigrationStrategy struct {
    // Phased: apply one resource at a time with validation
    // +kubebuilder:default=true
    Phased bool `json:"phased,omitempty"`
    // Pause between resources (for phased)
    PauseBetween string `json:"pauseBetween,omitempty"` // e.g., "30s"
    // Validate each resource after apply
    ValidateAfterApply bool `json:"validateAfterApply,omitempty"`
}

type MigrationPlanStatus struct {
    Phase            string `json:"phase,omitempty"` // Analyzing, Reviewed, Applying, Applied, RollingBack, Completed, Failed
    TotalResources   int    `json:"totalResources,omitempty"`
    AppliedResources int    `json:"appliedResources,omitempty"`
    FailedResources  int    `json:"failedResources,omitempty"`
    // Analysis report (populated when source is processed)
    Analysis *MigrationAnalysis `json:"analysis,omitempty"`
    // Per-resource status
    ResourceStatuses []MigrationResourceStatus `json:"resourceStatuses,omitempty"`
    Conditions       []metav1.Condition        `json:"conditions,omitempty"`
}

type MigrationAnalysis struct {
    TotalSourceResources int `json:"totalSourceResources"`
    DirectMappings       int `json:"directMappings"`
    RequiresReview       int `json:"requiresReview"`
    Unsupported          int `json:"unsupported"`
    RequiresEnterprise   int `json:"requiresEnterprise"`
    OverallConfidence    string `json:"overallConfidence"` // high, medium, low
}

type MigrationResourceStatus struct {
    SourceRef string `json:"sourceRef"`
    Kind      string `json:"kind"`
    Name      string `json:"name"`
    Applied   bool   `json:"applied"`
    Healthy   bool   `json:"healthy"`
    Error     string `json:"error,omitempty"`
}
```

#### CertificateBundle CRD

```go
// operator/api/v1alpha1/certificatebundle_types.go

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Hostname",type=string,JSONPath=`.spec.hostname`
// +kubebuilder:printcolumn:name="Expires",type=date,JSONPath=`.status.expiresAt`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
type CertificateBundle struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`
    Spec              CertificateBundleSpec   `json:"spec,omitempty"`
    Status            CertificateBundleStatus `json:"status,omitempty"`
}

type CertificateBundleSpec struct {
    // Hostname(s) the certificate covers
    Hostnames []string `json:"hostnames"`

    // Source: inline, cert-manager, upload
    Source CertSource `json:"source"`

    // Alert when cert expires within this duration
    // +kubebuilder:default="30d"
    ExpiryAlertThreshold string `json:"expiryAlertThreshold,omitempty"`

    // Auto-renewal via cert-manager
    AutoRenew bool `json:"autoRenew,omitempty"`

    // Gateway listeners to attach this cert to
    AttachTo []GatewayListenerRef `json:"attachTo,omitempty"`
}

type CertSource struct {
    // Inline PEM
    // +optional
    InlineCert *InlineCertSpec `json:"inline,omitempty"`
    // cert-manager issuer reference
    // +optional
    CertManager *CertManagerSpec `json:"certManager,omitempty"`
}

type CertManagerSpec struct {
    IssuerRef   string `json:"issuerRef"`
    IssuerKind  string `json:"issuerKind"` // Issuer, ClusterIssuer
}

type GatewayListenerRef struct {
    GatewayName      string `json:"gatewayName"`
    GatewayNamespace string `json:"gatewayNamespace,omitempty"`
    ListenerName     string `json:"listenerName"`
}

type CertificateBundleStatus struct {
    Phase     string `json:"phase,omitempty"` // Pending, Active, Expiring, Expired, Error
    ExpiresAt *metav1.Time `json:"expiresAt,omitempty"`
    IssuedAt  *metav1.Time `json:"issuedAt,omitempty"`
    Issuer    string `json:"issuer,omitempty"`
    Children  CertBundleChildren `json:"children,omitempty"`
    Conditions []metav1.Condition `json:"conditions,omitempty"`
}

type CertBundleChildren struct {
    Secret      ChildStatus `json:"secret,omitempty"`
    Certificate ChildStatus `json:"certificate,omitempty"` // cert-manager Certificate
}
```

### 2.5.4 Controller Implementation — Reconciliation Loops

> **Claude Code: This is the most important implementation section.** Each controller follows the same pattern. Use `controller-runtime` v0.17+ with the `Reconcile(ctx, req)` pattern. Every controller MUST implement drift detection, status aggregation, and owner references.

#### Common Patterns (All Controllers Must Follow)

```go
// operator/internal/controller/common.go

// Every child resource MUST have owner references pointing to the parent CRD.
// This ensures K8s garbage collection cleans up children when parent is deleted.
func setOwnerReference(parent, child client.Object, scheme *runtime.Scheme) error {
    return controllerutil.SetControllerReference(parent, child, scheme)
}

// Hash the spec to detect changes. Store in status.observedSpecHash.
// Only reconcile children when the spec hash changes OR when drift is detected.
func hashSpec(spec interface{}) string {
    data, _ := json.Marshal(spec)
    hash := sha256.Sum256(data)
    return hex.EncodeToString(hash[:8])
}

// Standard condition helpers
func setCondition(conditions *[]metav1.Condition, condType string, status metav1.ConditionStatus, reason, message string) {
    meta.SetStatusCondition(conditions, metav1.Condition{
        Type:               condType,
        Status:             status,
        Reason:             reason,
        Message:            message,
        LastTransitionTime: metav1.Now(),
    })
}

// Condition types used across all controllers:
const (
    ConditionReady       = "Ready"
    ConditionReconciled  = "Reconciled"
    ConditionDegraded    = "Degraded"
    ConditionDriftDetected = "DriftDetected"
    ConditionXCPublished = "XCPublished"
)
```

#### InferenceStack Controller — Full Reconciliation Loop

```go
// operator/internal/controller/inferencestack_controller.go

package controller

import (
    "context"
    "fmt"
    "time"

    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
    "sigs.k8s.io/controller-runtime/pkg/log"

    v1alpha1 "ngf-console/operator/api/v1alpha1"
    // Import Gateway API types, KEDA types, etc.
)

const inferenceStackFinalizer = "ngf-console.f5.com/inferencestack-finalizer"

type InferenceStackReconciler struct {
    client.Client
    Scheme *runtime.Scheme
    // XC client for DistributedCloud integration
    XCClient *xc.Client
}

// +kubebuilder:rbac:groups=ngf-console.f5.com,resources=inferencestacks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ngf-console.f5.com,resources=inferencestacks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=ngf-console.f5.com,resources=inferencestacks/finalizers,verbs=update
// +kubebuilder:rbac:groups=inference.networking.x-k8s.io,resources=inferencepools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=keda.sh,resources=scaledobjects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps;secrets;services,verbs=get;list;watch;create;update;patch;delete

func (r *InferenceStackReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    logger := log.FromContext(ctx)

    // 1. FETCH the parent CRD
    var stack v1alpha1.InferenceStack
    if err := r.Get(ctx, req.NamespacedName, &stack); err != nil {
        // CRD deleted — children cleaned up via owner references
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // 2. HANDLE DELETION (finalizer pattern)
    if !stack.DeletionTimestamp.IsZero() {
        if controllerutil.ContainsFinalizer(&stack, inferenceStackFinalizer) {
            // Clean up external resources (XC objects)
            if stack.Spec.DistributedCloud != nil {
                if err := r.cleanupXCResources(ctx, &stack); err != nil {
                    logger.Error(err, "failed to cleanup XC resources")
                    // Don't block deletion, but log the error
                }
            }
            controllerutil.RemoveFinalizer(&stack, inferenceStackFinalizer)
            if err := r.Update(ctx, &stack); err != nil {
                return ctrl.Result{}, err
            }
        }
        return ctrl.Result{}, nil
    }

    // 3. ADD FINALIZER if not present
    if !controllerutil.ContainsFinalizer(&stack, inferenceStackFinalizer) {
        controllerutil.AddFinalizer(&stack, inferenceStackFinalizer)
        if err := r.Update(ctx, &stack); err != nil {
            return ctrl.Result{}, err
        }
    }

    // 4. CHECK IF SPEC CHANGED (hash comparison)
    currentHash := hashSpec(stack.Spec)
    specChanged := currentHash != stack.Status.ObservedSpecHash

    // 5. RECONCILE EACH CHILD RESOURCE
    //    Order matters: InferencePool → EPP → Autoscaler → HTTPRoute → XC
    //    Each step: generate desired state → compare with actual → create/update/delete
    
    var childStatuses v1alpha1.ChildResourceStatuses
    var reconcileErrors []error

    // 5a. RECONCILE InferencePool
    poolStatus, err := r.reconcileInferencePool(ctx, &stack, specChanged)
    childStatuses.InferencePool = poolStatus
    if err != nil {
        reconcileErrors = append(reconcileErrors, fmt.Errorf("InferencePool: %w", err))
    }

    // 5b. RECONCILE EPP Configuration
    eppStatus, err := r.reconcileEPPConfig(ctx, &stack, specChanged)
    childStatuses.EPP = eppStatus
    if err != nil {
        reconcileErrors = append(reconcileErrors, fmt.Errorf("EPP: %w", err))
    }

    // 5c. RECONCILE Autoscaler (KEDA ScaledObject or HPA)
    scalerStatus, err := r.reconcileAutoscaler(ctx, &stack, specChanged)
    childStatuses.Autoscaler = scalerStatus
    if err != nil {
        reconcileErrors = append(reconcileErrors, fmt.Errorf("Autoscaler: %w", err))
    }

    // 5d. RECONCILE HTTPRoute
    routeStatus, err := r.reconcileHTTPRoute(ctx, &stack, specChanged)
    childStatuses.HTTPRoute = routeStatus
    if err != nil {
        reconcileErrors = append(reconcileErrors, fmt.Errorf("HTTPRoute: %w", err))
    }

    // 5e. RECONCILE DCGM DaemonSet (if enabled, shared across stacks)
    dcgmStatus, err := r.reconcileDCGMExporter(ctx, &stack)
    childStatuses.DCGMExporter = dcgmStatus
    if err != nil {
        reconcileErrors = append(reconcileErrors, fmt.Errorf("DCGM: %w", err))
    }

    // 5f. RECONCILE XC publish (if configured)
    var xcStatus *v1alpha1.XCPublishStatus
    if stack.Spec.DistributedCloud != nil {
        xcStatus, err = r.reconcileXCPublish(ctx, &stack, specChanged)
        if err != nil {
            reconcileErrors = append(reconcileErrors, fmt.Errorf("XC: %w", err))
        }
    }

    // 6. COMPUTE AGGREGATE PHASE
    phase := computePhase(childStatuses, reconcileErrors)

    // 7. UPDATE STATUS
    stack.Status.Phase = phase
    stack.Status.Children = childStatuses
    stack.Status.XCStatus = xcStatus
    stack.Status.ObservedSpecHash = currentHash
    stack.Status.LastReconciledAt = &metav1.Time{Time: time.Now()}

    // Set conditions
    if len(reconcileErrors) == 0 {
        setCondition(&stack.Status.Conditions, ConditionReconciled, metav1.ConditionTrue,
            "AllChildrenReconciled", "All child resources are reconciled")
        setCondition(&stack.Status.Conditions, ConditionReady, metav1.ConditionTrue,
            "StackReady", "InferenceStack is fully operational")
    } else {
        setCondition(&stack.Status.Conditions, ConditionDegraded, metav1.ConditionTrue,
            "ReconcileErrors", fmt.Sprintf("%d errors during reconciliation", len(reconcileErrors)))
    }

    if err := r.Status().Update(ctx, &stack); err != nil {
        logger.Error(err, "failed to update InferenceStack status")
        return ctrl.Result{}, err
    }

    // 8. REQUEUE for periodic drift detection (every 60s)
    return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
}

// --- CHILD RECONCILIATION METHODS ---
// Each follows the same pattern:
//   1. Generate desired child resource from parent spec
//   2. Fetch actual child resource from cluster (by owner reference)
//   3. If not found → Create
//   4. If found and differs → Update
//   5. If found and matches → No-op
//   6. Return ChildStatus

func (r *InferenceStackReconciler) reconcileInferencePool(
    ctx context.Context, stack *v1alpha1.InferenceStack, specChanged bool,
) (v1alpha1.ChildStatus, error) {
    desired := r.buildDesiredInferencePool(stack)

    // Set owner reference (critical for garbage collection)
    if err := setOwnerReference(stack, desired, r.Scheme); err != nil {
        return v1alpha1.ChildStatus{Ready: false, Message: err.Error()}, err
    }

    // CreateOrUpdate pattern from controller-runtime
    result, err := controllerutil.CreateOrUpdate(ctx, r.Client, desired, func() error {
        // Mutate the existing object to match desired state
        // This is called both on create and update
        r.mutateInferencePool(desired, stack)
        return nil
    })

    if err != nil {
        return v1alpha1.ChildStatus{
            Name: desired.Name, Namespace: desired.Namespace,
            Ready: false, Message: fmt.Sprintf("reconcile failed: %v", err),
        }, err
    }

    now := metav1.Now()
    return v1alpha1.ChildStatus{
        Name: desired.Name, Namespace: desired.Namespace,
        Ready: true,
        Message: fmt.Sprintf("InferencePool %s (%s)", desired.Name, result),
        LastSyncedAt: &now,
    }, nil
}

// buildDesiredInferencePool generates the InferencePool CRD from InferenceStack spec
func (r *InferenceStackReconciler) buildDesiredInferencePool(stack *v1alpha1.InferenceStack) *inferenceapi.InferencePool {
    // Naming convention: {stack-name}-pool
    pool := &inferenceapi.InferencePool{
        ObjectMeta: metav1.ObjectMeta{
            Name:      fmt.Sprintf("%s-pool", stack.Name),
            Namespace: stack.Namespace,
            Labels: map[string]string{
                "app.kubernetes.io/managed-by": "ngf-console",
                "ngf-console.f5.com/stack":     stack.Name,
            },
        },
    }
    // Populate spec from stack.Spec.Pool
    // ... (model backend, selector, resources, etc.)
    return pool
}

// Same pattern for reconcileEPPConfig, reconcileAutoscaler, reconcileHTTPRoute, etc.
// Each builds desired state → CreateOrUpdate → return status

// --- PHASE COMPUTATION ---

func computePhase(children v1alpha1.ChildResourceStatuses, errors []error) string {
    if len(errors) > 0 {
        // Check if all children failed or just some
        allFailed := !children.InferencePool.Ready && !children.HTTPRoute.Ready
        if allFailed {
            return "Error"
        }
        return "Degraded"
    }
    allReady := children.InferencePool.Ready &&
        children.EPP.Ready &&
        children.HTTPRoute.Ready &&
        children.Autoscaler.Ready
    if allReady {
        return "Ready"
    }
    return "Provisioning"
}

// --- SETUP ---

func (r *InferenceStackReconciler) SetupWithManager(mgr ctrl.Manager) error {
    return ctrl.NewControllerManagedBy(mgr).
        For(&v1alpha1.InferenceStack{}).
        // Watch child resources — trigger reconcile of parent when children change
        // This is how drift detection works: if someone edits/deletes a child,
        // the parent is re-reconciled and the operator restores desired state
        Owns(&inferenceapi.InferencePool{}).
        Owns(&gatewayapi.HTTPRoute{}).
        Owns(&corev1.ConfigMap{}).
        // For KEDA ScaledObject — if KEDA types aren't available at compile time,
        // use Watches() with an unstructured.Unstructured instead
        Complete(r)
}
```

#### GatewayBundle Controller — Reconciliation Loop (Abbreviated)

```go
// operator/internal/controller/gatewaybundle_controller.go

func (r *GatewayBundleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    var bundle v1alpha1.GatewayBundle
    if err := r.Get(ctx, req.NamespacedName, &bundle); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    // Handle deletion (finalizer cleanup for NginxProxy, WAF, etc.)
    // ...

    // Reconcile children in order:
    // 1. TLS Secrets (must exist before Gateway references them)
    // 2. Gateway
    // 3. NginxProxy (Enterprise — parametersRef on Gateway)
    // 4. APPolicy (Enterprise — attached to Gateway)
    // 5. SnippetsFilter (Enterprise — attached to Gateway or routes)

    gatewayStatus, err := r.reconcileGateway(ctx, &bundle)
    // Enterprise features: only reconcile if edition is enterprise
    // Use the edition detection from section 3.1
    if isEnterprise {
        r.reconcileNginxProxy(ctx, &bundle)
        r.reconcileWAFPolicy(ctx, &bundle)
        r.reconcileSnippets(ctx, &bundle)
    }

    // Aggregate status, update CRD, requeue for drift detection
    return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
}
```

#### MigrationPlan Controller — Phased Apply Logic

```go
// operator/internal/controller/migrationplan_controller.go

func (r *MigrationPlanReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    var plan v1alpha1.MigrationPlan
    if err := r.Get(ctx, req.NamespacedName, &plan); err != nil {
        return ctrl.Result{}, client.IgnoreNotFound(err)
    }

    switch {
    case plan.Status.Phase == "" || plan.Status.Phase == "Analyzing":
        // Run analysis: parse source, generate resources, compute confidence
        analysis := r.analyzeSource(ctx, &plan)
        plan.Status.Analysis = analysis
        plan.Status.Phase = "Reviewed"
        // Populate plan.Spec.Resources with generated resources
        // User reviews and edits via UI before setting plan.Spec.Apply = true

    case plan.Spec.Apply && plan.Status.Phase == "Reviewed":
        // Begin phased apply
        plan.Status.Phase = "Applying"
        // Fall through to apply logic

    case plan.Status.Phase == "Applying":
        // Apply next resource in the list
        nextIdx := plan.Status.AppliedResources
        if nextIdx >= len(plan.Spec.Resources) {
            plan.Status.Phase = "Completed"
            break
        }
        
        resource := plan.Spec.Resources[nextIdx]
        if !resource.Enabled {
            plan.Status.AppliedResources++
            // Requeue immediately for next resource
            return ctrl.Result{Requeue: true}, nil
        }

        // Apply the resource
        err := r.applyResource(ctx, &plan, resource)
        if err != nil {
            plan.Status.FailedResources++
            plan.Status.ResourceStatuses = append(plan.Status.ResourceStatuses,
                v1alpha1.MigrationResourceStatus{
                    SourceRef: resource.SourceRef, Kind: resource.Kind,
                    Applied: false, Error: err.Error(),
                })
        } else {
            plan.Status.AppliedResources++
            plan.Status.ResourceStatuses = append(plan.Status.ResourceStatuses,
                v1alpha1.MigrationResourceStatus{
                    SourceRef: resource.SourceRef, Kind: resource.Kind,
                    Applied: true, Healthy: true,
                })
        }

        // If phased, wait before next resource
        if plan.Spec.Strategy.Phased {
            pause, _ := time.ParseDuration(plan.Spec.Strategy.PauseBetween)
            if pause == 0 { pause = 30 * time.Second }
            return ctrl.Result{RequeueAfter: pause}, nil
        }
        return ctrl.Result{Requeue: true}, nil

    case plan.Spec.Rollback && plan.Status.Phase != "RollingBack":
        // Rollback: delete all applied resources in reverse order
        plan.Status.Phase = "RollingBack"
        for i := len(plan.Status.ResourceStatuses) - 1; i >= 0; i-- {
            if plan.Status.ResourceStatuses[i].Applied {
                r.deleteResource(ctx, plan.Status.ResourceStatuses[i])
            }
        }
        plan.Status.Phase = "Reviewed" // Reset to allow re-apply
    }

    r.Status().Update(ctx, &plan)
    return ctrl.Result{}, nil
}
```

### 2.5.5 Drift Detection & Self-Healing Engine

```go
// operator/internal/controller/drift.go

// Drift detection runs on every periodic reconciliation (60s requeue).
// For each parent CRD, the controller:
//
// 1. Computes desired state from parent spec
// 2. Fetches actual state from cluster
// 3. Compares (using resource hash or deep equal)
// 4. If different:
//    a. Log the drift with details (who changed what)
//    b. Set DriftDetected condition on parent
//    c. Restore desired state (overwrite the child)
//    d. Record in audit log (via ConfigDB)
//
// IMPORTANT: Owner references handle deletion drift automatically.
// If a child is deleted, K8s fires an event, the Owns() watch triggers,
// and the controller re-creates it. This drift detection handles
// MODIFICATION drift (someone edited the child's spec).

type DriftReport struct {
    ChildKind      string
    ChildName      string
    ChildNamespace string
    DriftType      string // "modified", "deleted", "extra" (orphaned child)
    Details        string // what specifically changed
    DetectedAt     time.Time
}

func detectDrift(desired, actual client.Object) *DriftReport {
    // Compare spec fields (ignore status, metadata.resourceVersion, etc.)
    // Use a normalized comparison that strips server-set fields
    desiredSpec := extractSpec(desired)
    actualSpec := extractSpec(actual)
    
    if !reflect.DeepEqual(desiredSpec, actualSpec) {
        diff := computeDiff(desiredSpec, actualSpec)
        return &DriftReport{
            ChildKind: actual.GetObjectKind().GroupVersionKind().Kind,
            ChildName: actual.GetName(),
            ChildNamespace: actual.GetNamespace(),
            DriftType: "modified",
            Details: diff,
            DetectedAt: time.Now(),
        }
    }
    return nil
}
```

### 2.5.6 Operator Manager Setup

```go
// operator/cmd/main.go

package main

import (
    "os"

    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/healthz"
    "sigs.k8s.io/controller-runtime/pkg/log/zap"

    v1alpha1 "ngf-console/operator/api/v1alpha1"
    "ngf-console/operator/internal/controller"
)

func main() {
    opts := zap.Options{Development: true}
    ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

    mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
        Scheme:                 scheme,
        MetricsBindAddress:     ":8081",
        HealthProbeBindAddress: ":8082",
        LeaderElection:         true,
        LeaderElectionID:       "ngf-console-operator",
        // Leader election prevents split-brain when running multiple replicas
    })
    if err != nil {
        os.Exit(1)
    }

    // Register all controllers
    if err := (&controller.InferenceStackReconciler{
        Client: mgr.GetClient(),
        Scheme: mgr.GetScheme(),
    }).SetupWithManager(mgr); err != nil {
        os.Exit(1)
    }

    if err := (&controller.GatewayBundleReconciler{
        Client: mgr.GetClient(),
        Scheme: mgr.GetScheme(),
    }).SetupWithManager(mgr); err != nil {
        os.Exit(1)
    }

    if err := (&controller.MigrationPlanReconciler{
        Client: mgr.GetClient(),
        Scheme: mgr.GetScheme(),
    }).SetupWithManager(mgr); err != nil {
        os.Exit(1)
    }

    if err := (&controller.DistributedCloudPublishReconciler{
        Client: mgr.GetClient(),
        Scheme: mgr.GetScheme(),
    }).SetupWithManager(mgr); err != nil {
        os.Exit(1)
    }

    if err := (&controller.CertificateBundleReconciler{
        Client: mgr.GetClient(),
        Scheme: mgr.GetScheme(),
    }).SetupWithManager(mgr); err != nil {
        os.Exit(1)
    }

    // Health checks
    mgr.AddHealthzCheck("healthz", healthz.Ping)
    mgr.AddReadyzCheck("readyz", healthz.Ping)

    if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
        os.Exit(1)
    }
}
```

### 2.5.7 API Server Changes — Thin BFF Pattern

> **Claude Code: The API server is DELIBERATELY thin.** Its job is:
> 1. Accept REST/WebSocket requests from the frontend
> 2. For WRITES: Create/update/delete parent CRDs (NOT child resources)
> 3. For READS: Aggregate data from K8s API (CRD status), Prometheus, ClickHouse, DCGM
> 4. Maintain the config DB (audit logs, user prefs, saved views)
> 5. Proxy WebSocket streams (events, metrics, EPP decisions)

**What the API server DOES:**

```go
// api/internal/handlers/inference.go

// POST /api/v1/inference/stacks
func (h *InferenceHandler) CreateInferenceStack(w http.ResponseWriter, r *http.Request) {
    var req CreateInferenceStackRequest
    json.NewDecoder(r.Body).Decode(&req)

    // Build the InferenceStack CRD from the request
    stack := &v1alpha1.InferenceStack{
        ObjectMeta: metav1.ObjectMeta{
            Name:      req.Name,
            Namespace: req.Namespace,
        },
        Spec: req.ToSpec(), // Convert API request to CRD spec
    }

    // Create the parent CRD — the operator handles everything else
    err := h.k8sClient.Create(r.Context(), stack)
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }

    // Record in audit log
    h.auditLog.Record(r.Context(), AuditEntry{
        Action:       "create",
        ResourceKind: "InferenceStack",
        ResourceName: stack.Name,
        NewConfig:    stack.Spec,
    })

    json.NewEncoder(w).Encode(stack)
}

// GET /api/v1/inference/stacks/:ns/:name
func (h *InferenceHandler) GetInferenceStack(w http.ResponseWriter, r *http.Request) {
    var stack v1alpha1.InferenceStack
    err := h.k8sClient.Get(r.Context(), types.NamespacedName{
        Name: chi.URLParam(r, "name"),
        Namespace: chi.URLParam(r, "namespace"),
    }, &stack)

    // The status already contains aggregated health from the operator
    // No need to individually query InferencePool, EPP, HPA, etc.
    json.NewEncoder(w).Encode(stack)
}
```

**What the API server does NOT do:**

```go
// ❌ WRONG — DO NOT DO THIS IN THE API SERVER
func (h *InferenceHandler) CreateInferencePool(w http.ResponseWriter, r *http.Request) {
    pool := buildInferencePool(req)
    h.k8sClient.Create(ctx, pool)      // ❌ Direct child resource creation
    epp := buildEPPConfig(req)
    h.k8sClient.Create(ctx, epp)        // ❌ Direct child resource creation
    hpa := buildHPA(req)
    h.k8sClient.Create(ctx, hpa)        // ❌ Direct child resource creation
    // This is the imperative pattern we're REPLACING
}
```

### 2.5.8 GitOps Compatibility

The CRD-based architecture is inherently GitOps-friendly:

```
# In a Git repository managed by ArgoCD or Flux:
infrastructure/
├── inference-stacks/
│   ├── llama3-production.yaml        # InferenceStack CRD
│   ├── mixtral-staging.yaml          # InferenceStack CRD
│   └── embedding-service.yaml        # InferenceStack CRD
├── gateways/
│   ├── production-gateway.yaml       # GatewayBundle CRD
│   └── staging-gateway.yaml          # GatewayBundle CRD
├── xc-publish/
│   └── llm-api-public.yaml          # DistributedCloudPublish CRD
└── certificates/
    ├── wildcard-prod.yaml            # CertificateBundle CRD
    └── api-cert.yaml                 # CertificateBundle CRD
```

**ArgoCD deploys these CRDs → NGF Console Operator reconciles all child resources → Zero kubectl required.**

The UI and ArgoCD are two interfaces to the same CRDs. Changes made in the UI create/update CRDs, which ArgoCD can then sync to Git. Changes pushed to Git are applied by ArgoCD, and the UI reflects the new state via its CRD watches.

### 2.5.9 RBAC Requirements

```yaml
# The operator needs broad permissions to manage child resources
# operator/config/rbac/role.yaml

apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: ngf-console-operator
rules:
  # Parent CRDs (full control)
  - apiGroups: ["ngf-console.f5.com"]
    resources: ["inferencestacks", "gatewaybundles", "migrationplans",
                "distributedcloudpublishes", "certificatebundles"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  - apiGroups: ["ngf-console.f5.com"]
    resources: ["inferencestacks/status", "gatewaybundles/status",
                "migrationplans/status", "distributedcloudpublishes/status",
                "certificatebundles/status"]
    verbs: ["get", "update", "patch"]
  - apiGroups: ["ngf-console.f5.com"]
    resources: ["inferencestacks/finalizers", "gatewaybundles/finalizers",
                "distributedcloudpublishes/finalizers"]
    verbs: ["update"]

  # Gateway API resources (child management)
  - apiGroups: ["gateway.networking.k8s.io"]
    resources: ["gateways", "gatewayclasses", "httproutes", "grpcroutes",
                "tlsroutes", "tcproutes", "udproutes", "referencegrants"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

  # Inference Extension resources (child management)
  - apiGroups: ["inference.networking.x-k8s.io"]
    resources: ["inferencepools"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

  # NGF Enterprise resources
  - apiGroups: ["gateway.nginx.org"]
    resources: ["nginxproxies", "snippetsfilters", "clientsettingspolicies",
                "observabilitypolicies", "ratelimitpolicies"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

  # Core resources
  - apiGroups: [""]
    resources: ["configmaps", "secrets", "services", "pods", "events"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

  # Autoscaling
  - apiGroups: ["autoscaling"]
    resources: ["horizontalpodautoscalers"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  - apiGroups: ["keda.sh"]
    resources: ["scaledobjects", "triggerauthentications"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

  # DCGM Exporter DaemonSet
  - apiGroups: ["apps"]
    resources: ["daemonsets", "deployments"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

  # Cert-manager (if present)
  - apiGroups: ["cert-manager.io"]
    resources: ["certificates", "issuers", "clusterissuers"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

  # Leader election
  - apiGroups: ["coordination.k8s.io"]
    resources: ["leases"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

---
# The API server needs much narrower permissions
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: ngf-console-api
rules:
  # Parent CRDs only (CRUD for UI writes)
  - apiGroups: ["ngf-console.f5.com"]
    resources: ["inferencestacks", "gatewaybundles", "migrationplans",
                "distributedcloudpublishes", "certificatebundles"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]

  # Read-only access to K8s resources (for topology, metrics, status)
  - apiGroups: ["gateway.networking.k8s.io"]
    resources: ["gateways", "gatewayclasses", "httproutes", "grpcroutes",
                "tlsroutes", "tcproutes", "udproutes"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["inference.networking.x-k8s.io"]
    resources: ["inferencepools"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["pods", "services", "events", "namespaces", "secrets"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["apps"]
    resources: ["deployments", "daemonsets", "replicasets"]
    verbs: ["get", "list", "watch"]
```

### 2.5.10 Testing Strategy for the Operator

> **Claude Code: Tests are critical for controllers. A bug in a reconciliation loop can break entire clusters.**

```go
// Use envtest (controller-runtime's test framework) for integration tests.
// It spins up a real etcd + API server without needing a full cluster.

// operator/internal/controller/inferencestack_controller_test.go

import (
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
    "sigs.k8s.io/controller-runtime/pkg/envtest"
)

var _ = Describe("InferenceStack Controller", func() {
    Context("When creating an InferenceStack", func() {
        It("Should create all child resources", func() {
            stack := &v1alpha1.InferenceStack{
                ObjectMeta: metav1.ObjectMeta{Name: "test-stack", Namespace: "default"},
                Spec: testInferenceStackSpec(),
            }
            Expect(k8sClient.Create(ctx, stack)).To(Succeed())

            // Wait for reconciliation
            Eventually(func() string {
                var s v1alpha1.InferenceStack
                k8sClient.Get(ctx, types.NamespacedName{Name: "test-stack", Namespace: "default"}, &s)
                return s.Status.Phase
            }, timeout, interval).Should(Equal("Ready"))

            // Verify child resources exist
            var pool inferenceapi.InferencePool
            Expect(k8sClient.Get(ctx, types.NamespacedName{
                Name: "test-stack-pool", Namespace: "default",
            }, &pool)).To(Succeed())

            var route gatewayapi.HTTPRoute
            Expect(k8sClient.Get(ctx, types.NamespacedName{
                Name: "test-stack-route", Namespace: "default",
            }, &route)).To(Succeed())

            // Verify owner references
            Expect(pool.OwnerReferences).To(HaveLen(1))
            Expect(pool.OwnerReferences[0].Name).To(Equal("test-stack"))
        })

        It("Should self-heal when a child is deleted", func() {
            // Delete the HTTPRoute (simulating someone using kubectl delete)
            var route gatewayapi.HTTPRoute
            k8sClient.Get(ctx, types.NamespacedName{
                Name: "test-stack-route", Namespace: "default",
            }, &route)
            Expect(k8sClient.Delete(ctx, &route)).To(Succeed())

            // Wait for operator to detect and recreate
            Eventually(func() error {
                return k8sClient.Get(ctx, types.NamespacedName{
                    Name: "test-stack-route", Namespace: "default",
                }, &route)
            }, timeout, interval).Should(Succeed())
        })

        It("Should detect spec drift and correct", func() {
            // Modify the HTTPRoute directly (simulating kubectl edit)
            var route gatewayapi.HTTPRoute
            k8sClient.Get(ctx, types.NamespacedName{
                Name: "test-stack-route", Namespace: "default",
            }, &route)
            route.Spec.Rules[0].Matches[0].Path.Value = ptr.To("/hacked")
            k8sClient.Update(ctx, &route)

            // Wait for operator to restore correct path
            Eventually(func() string {
                k8sClient.Get(ctx, types.NamespacedName{
                    Name: "test-stack-route", Namespace: "default",
                }, &route)
                return *route.Spec.Rules[0].Matches[0].Path.Value
            }, timeout, interval).Should(Equal("/v1/models/test"))
        })
    })

    Context("When deleting an InferenceStack", func() {
        It("Should clean up all child resources via owner references", func() {
            var stack v1alpha1.InferenceStack
            k8sClient.Get(ctx, types.NamespacedName{Name: "test-stack", Namespace: "default"}, &stack)
            Expect(k8sClient.Delete(ctx, &stack)).To(Succeed())

            // All children should be garbage collected
            Eventually(func() bool {
                var pool inferenceapi.InferencePool
                err := k8sClient.Get(ctx, types.NamespacedName{
                    Name: "test-stack-pool", Namespace: "default",
                }, &pool)
                return errors.IsNotFound(err)
            }, timeout, interval).Should(BeTrue())
        })
    })
})
```

---

## 3. Feature Specifications

### 3.1 Edition Detection & Feature Gating

The UI must auto-detect whether the connected NGF instance is OSS or Enterprise.

**Detection Method:**
1. Query the GatewayClass for `parametersRef` pointing to NginxProxy with enterprise fields
2. Check for presence of enterprise CRDs: `APPolicies`, `SnippetsFilters`, etc.
3. Check NGF controller deployment labels/annotations for edition info

**Behavior:**
- Enterprise features show as greyed-out cards/buttons with a tooltip: "Requires NGINX Gateway Fabric Enterprise. Contact F5 for a trial."
- Link to F5 trial page from greyed-out features
- No enterprise API calls made when running against OSS

**Enterprise-only features:**
- App Protect WAF policy management
- SnippetsFilter builder
- Bot defense configuration
- Advanced NGINX tuning via NginxProxy
- F5 Distributed Cloud integration

---

### 3.2 Gateway Creation Workflow

A multi-step wizard that collects gateway configuration and creates a single **GatewayBundle CRD**. The operator reconciles all child resources (Gateway, NginxProxy, APPolicy, SnippetsFilter, TLS Secrets).

> **Operator Pattern:** The wizard does NOT create Gateway, NginxProxy, APPolicy, or SnippetsFilter resources directly. It builds a GatewayBundle CRD spec from wizard inputs. The API server creates the CRD. The operator reconciles all children. Status is read from the CRD's `.status` field.

**Step 1 — GatewayClass Selection**
- List available GatewayClasses with feature comparison table
- Show parametersRef availability
- Highlight enterprise capabilities with edition badge
- API: `GET /api/v1/gatewayclasses` (read-only K8s query)

**Step 2 — Gateway Configuration**
- Form fields: name, namespace, labels, annotations
- Listener builder (dynamic list):
  - Protocol selector: HTTP, HTTPS, TLS, TCP, UDP
  - Port number with conflict detection (real-time validation against cluster)
  - Hostname (with wildcard support and validation)
  - TLS configuration: mode (Terminate, Passthrough), certificateRef selector
  - Allowed routes: namespaces (Same, All, Selector), kinds
- Infrastructure settings via parametersRef (Enterprise):
  - Replica count, resource requests/limits
  - NGINX worker tuning

**Step 3 — WAF Policy (Enterprise)**
- APPolicy selector or inline creation
- Threat signature browser with search/filter
- Violation rating threshold slider with plain-English labels
- Bot defense profile selector
- Log-only / Enforce toggle

**Step 4 — Advanced Configuration (Enterprise)**
- NginxProxy resource builder
- SnippetsFilter builder with:
  - Context selector: http, server, location
  - Syntax highlighting editor (Monaco)
  - NGINX config validation (client-side + server-side)
  - "What this does" documentation panel

**Step 5 — Review & Deploy**
- Complete GatewayBundle YAML preview (read-only, copyable)
- Client-side validation of spec completeness
- Diff view for modifications (against existing GatewayBundle if editing)
- Deploy button creates/updates the GatewayBundle CRD
- Operator reconciles children — UI polls CRD `.status` for progress
- API: `POST /api/v1/gatewaybundles` (creates GatewayBundle CRD)
- API: `GET /api/v1/gatewaybundles/{ns}/{name}` (polls reconciliation status)

**Post-Deploy — Topology View**
- Interactive DAG: GatewayBundle → Gateway → Listeners → HTTPRoutes → BackendRefs → Services → Pods
- Status badges from CRD `.status.children` (Gateway Ready, WAF Active, etc.)
- Drift detection alerts shown inline (operator detected/corrected changes)
- Click-to-drill on any node
- Real-time status updates via WebSocket (watches CRD status changes)
- API: `GET /api/v1/topology?gateway=<n>`

---

### 3.3 Traffic Management

#### 3.3.1 Visual Route Configuration

**HTTPRoute Builder:**
- Form-based with live YAML preview panel
- Fields: parentRefs (gateway selector), hostnames, rules
- Rule builder:
  - Match conditions: path (Exact, PathPrefix, RegularExpression), headers, query params, method
  - Backend refs with weight sliders
  - Filters: RequestHeaderModifier, ResponseHeaderModifier, URLRewrite, RequestRedirect, RequestMirror, ExtensionRef
- Validation: real-time against cluster state
- API: `POST /api/v1/httproutes` (creates HTTPRoute directly — lightweight resource, no parent CRD wrapper needed)

**GRPCRoute Builder:**
- Similar to HTTPRoute but with gRPC-specific matching (service, method)
- API: `POST /api/v1/grpcroutes` (creates GRPCRoute directly)

**TLSRoute / TCPRoute / UDPRoute Builders:**
- Simplified forms for L4 routing
- API: `POST /api/v1/{tls,tcp,udp}routes` (creates route directly)

#### 3.3.2 Traffic Splitting Visualization

- Animated traffic flow diagram showing request distribution across backends
- Weight adjustment via drag sliders with real-time preview
- Historical traffic distribution chart (from ClickHouse)
- Canary deployment helper: auto-configure progressive weight shifts with timers
- API: `GET /api/v1/traffic-split?route=<name>`

#### 3.3.3 Header/Query Parameter Matching Rules Builder

- Visual match condition builder (AND/OR logic tree)
- Test request simulator:
  - Input: method, path, headers, query params
  - Output: which route/rule would match, which backend would receive the request
  - Highlighting of the matching conditions
- API: `POST /api/v1/routes/simulate` (read-only simulation — does not modify cluster)

#### 3.3.4 Gateway & GatewayClass Lifecycle

- List view with status, listener count, route count, age
- Inline editing of existing Gateways
- Scale operations (update replica count)
- Delete with dependency check ("This gateway has 5 routes attached. Delete anyway?")
- GatewayClass management (view, not create — those are cluster-scoped and admin-managed)

---

### 3.4 Observability Dashboard

#### 3.4.1 Real-Time Metrics (Prometheus)

**Sources:** Prometheus endpoint exposed by NGF (existing)

**Dashboard Panels:**
- Request rate (RPS) — global, per gateway, per route, per backend
- Error rate (4xx, 5xx) — with breakdown by status code
- Latency (p50, p95, p99) — histograms per route
- Active connections — per gateway/listener
- Upstream health — per backend service
- SSL handshake rate and errors
- Connection pool utilization

**Implementation:**
- Prometheus client in backend API queries Prometheus directly
- WebSocket push for real-time updates (1s interval configurable)
- Time range selector: 5m, 15m, 1h, 6h, 24h, custom
- Auto-refresh toggle

**Grafana Integration:**
- "Open in Grafana" button per panel (configurable Grafana URL)
- Prometheus endpoint remains the source of truth for Grafana users
- No duplication of data — both UI and Grafana read from same Prometheus

#### 3.4.2 Log Analytics (ClickHouse)

**Data Model:**

```sql
CREATE TABLE ngf_access_logs (
    timestamp DateTime64(3),
    gateway String,
    listener String,
    route String,
    namespace String,
    method LowCardinality(String),
    path String,
    status UInt16,
    latency_ms Float64,
    upstream_latency_ms Float64,
    request_size UInt64,
    response_size UInt64,
    upstream_name String,
    upstream_addr String,
    client_ip String,
    user_agent String,
    request_id String,
    trace_id String,
    tls_version LowCardinality(String),
    tls_cipher LowCardinality(String),
    waf_action LowCardinality(String),    -- Enterprise: pass/block/monitor
    waf_violation_rating Float32,          -- Enterprise
    waf_signatures Array(String),          -- Enterprise
    bot_classification LowCardinality(String), -- Enterprise
    xc_edge_latency_ms Float64            -- When XC-published
) ENGINE = MergeTree()
PARTITION BY toYYYYMMDD(timestamp)
ORDER BY (gateway, route, timestamp)
TTL timestamp + INTERVAL 7 DAY;
```

**Materialized Views for Rollups:**

```sql
-- 1-minute rollups
CREATE MATERIALIZED VIEW ngf_metrics_1m
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMMDD(window_start)
ORDER BY (gateway, route, status_class, window_start)
TTL window_start + INTERVAL 90 DAY
AS SELECT
    toStartOfMinute(timestamp) AS window_start,
    gateway, route,
    multiIf(status < 200, '1xx', status < 300, '2xx', status < 400, '3xx',
            status < 500, '4xx', '5xx') AS status_class,
    count() AS request_count,
    avg(latency_ms) AS avg_latency,
    quantile(0.95)(latency_ms) AS p95_latency,
    quantile(0.99)(latency_ms) AS p99_latency,
    sum(request_size) AS total_request_bytes,
    sum(response_size) AS total_response_bytes
FROM ngf_access_logs
GROUP BY window_start, gateway, route, status_class;

-- 1-hour rollups (same structure, longer TTL)
-- 1-day rollups (same structure, 1 year TTL)
```

**UI Panels Powered by ClickHouse:**
- Access log explorer with full-text search and filters
- Top-N: busiest routes, highest error routes, slowest backends, top user agents
- Request size distribution
- Geographic traffic distribution (via client IP → GeoIP)
- WAF event timeline (Enterprise)
- Custom SQL query editor for power users

**Telemetry Pipeline:**

```
NGINX → OTel Sidecar/DaemonSet → OTel Collector → ClickHouse
                                       ↓
                               Prometheus (metrics)
```

OTel Collector config (deployed as part of Helm chart):
- Receivers: otlp (from NGINX OTel module)
- Processors: batch, resource attributes
- Exporters: clickhouse (logs), prometheus (metrics)

---

### 3.5 Policy Management

#### 3.5.1 Policy Types

| Policy | Scope | Edition |
|--------|-------|---------|
| RateLimitPolicy | Route, Gateway | OSS + Enterprise |
| BackendTLSPolicy | BackendRef | OSS + Enterprise |
| ClientSettingsPolicy | Route, Gateway | OSS + Enterprise |
| ObservabilityPolicy | Route, Gateway | OSS + Enterprise |
| APPolicy (WAF) | Route, Gateway | Enterprise |
| SnippetsFilter | Route | Enterprise |

#### 3.5.2 Policy Builder UI

- Template gallery: pre-built policies for common scenarios
  - "Rate limit API to 100 req/min"
  - "Enable mTLS to backend"
  - "Block SQL injection"
  - "Add CORS headers"
- Visual policy builder: form-based with YAML preview
- Policy attachment: drag-and-drop onto routes/gateways in topology view
- Conflict detection: show which policies are winning at each attachment point
  - Gateway API hierarchy: GatewayClass → Gateway → Route
  - Visual indicators: green (active), yellow (overridden), red (conflicting)
- API: `POST /api/v1/policies/{type}` (policies are standalone K8s resources attached to routes/gateways — created directly, not via parent CRD)

#### 3.5.3 WAF Management (Enterprise)

- Threat signature browser with categories, CVE references
- Custom rule builder with ModSecurity-compatible syntax
- WAF event dashboard: blocked requests timeline, top violations, attack source map
- Policy testing: replay captured requests against policy to preview enforcement
- API: `GET/POST /api/v1/policies/waf` (standalone policy; also embeddable in GatewayBundle CRD for wizard-created gateways)

---

### 3.6 Certificate & TLS Management

**Inventory View:**
- All TLS Secrets in monitored namespaces
- cert-manager Certificate resources (if installed)
- Fields: common name, SANs, issuer, expiry date, key type, attached-to
- Expiry status: green (>30d), yellow (7-30d), red (<7d), expired

**Certificate Lifecycle:**
- Upload new cert/key pair
- Generate CSR (for enterprise CA integration)
- Renew: trigger cert-manager renewal or manual replacement
- cert-to-listener mapping visualization

**Alerting:**
- Configurable expiry threshold alerts (email, Slack webhook, PagerDuty)
- Alert rules stored in configuration DB

**mTLS Configuration:**
- Backend TLS policy builder
- CA bundle management
- Client certificate rotation workflow

**API:** `POST /api/v1/certificatebundles` (creates CertificateBundle CRD — operator reconciles TLS Secret + cert-manager Certificate + expiry alerting)
**Read:** `GET /api/v1/certificatebundles` (reads CRD `.status` for cert health, expiry, renewal status)

---

### 3.7 Troubleshooting & Debugging

#### 3.7.1 Diagnostic Wizard — "Why Isn't My Route Working?"

Interactive wizard that checks (in order):
1. **Route status**: Is the HTTPRoute accepted? Are conditions met?
2. **Gateway attachment**: Is the route attached to a programmed gateway?
3. **Listener match**: Does the route's parentRef match a listener? (hostname, port, protocol, allowed namespaces)
4. **Route precedence**: Is another route taking priority? (longest path match, creation timestamp)
5. **Backend health**: Are backend services healthy? Endpoints present?
6. **Policy blocks**: Is a WAF, rate limit, or other policy blocking requests?
7. **TLS issues**: Certificate valid? TLS version mismatch?
8. **Network connectivity**: Can the gateway pod reach the backend? (optional kubectl exec probe)

Output: step-by-step diagnostic report with pass/fail/warning per check and remediation suggestions.

API: `GET /api/v1/diagnostics/route-check` (read-only analysis of cluster state)

#### 3.7.2 Request Tracing

- Input: request method, path, headers (or paste a curl command)
- Trace the request through: gateway → listener → route matching → policy evaluation → upstream selection → backend response
- Correlate with access logs in ClickHouse via request_id or trace_id
- Display as waterfall timeline

API: `POST /api/v1/diagnostics/trace` (read-only trace — does not modify cluster state)

#### 3.7.3 Configuration Diff Viewer

- Compare current vs. previous configuration for any resource
- Side-by-side diff with syntax highlighting
- Historical versions stored in configuration DB (audit log)
- "Revert to this version" button

API: `GET /api/v1/audit/diff?resource=<kind>/<namespace>/<name>&from=<version>&to=<version>`

#### 3.7.4 Event Stream

- Real-time Kubernetes events for NGF-related resources
- Filterable by: gateway, route, namespace, event type, severity
- Searchable full-text
- WebSocket-based streaming

API: `WS /api/v1/events/stream`

---

### 3.8 F5 Distributed Cloud Integration

#### 3.8.1 XC Auto-Registration Controller

**CRD Definition:**

```yaml
apiVersion: ngf-console.f5.com/v1alpha1
kind: DistributedCloudPublish
metadata:
  name: my-app-xc
  namespace: default
spec:
  httpRouteRef:
    name: my-app-route
    namespace: default
  distributedCloud:
    tenant: my-tenant
    namespace: production
    wafPolicy: default-waf        # XC App Firewall name
    botDefense:
      enabled: true
      profile: standard
    ddosProtection:
      enabled: true
      profile: standard
    publicHostname: app.example.com
    tls:
      mode: managed               # managed (XC Let's Encrypt) | bringYourOwn
      certificateRef:              # only if mode=bringYourOwn
        name: my-tls-cert
        namespace: default
    originPool:
      autoDetect: true             # auto-detect NGF external IP/LB
      overrideAddress: ""          # manual override
      port: 443
      useTLS: true
      tlsConfig:
        sni: internal.example.com
        skipVerify: false
status:
  state: Published                 # Pending | Published | Error | Degraded
  xcHttpLoadBalancerName: "ves-io-http-lb-my-app-xc"
  xcOriginPoolName: "ves-io-op-my-app-xc"
  publicEndpoint: "https://app.example.com"
  lastSyncTime: "2026-02-09T12:00:00Z"
  conditions:
    - type: Synced
      status: "True"
      message: "XC HTTP Load Balancer is active"
```

**Controller Reconciliation Loop:**

```
Watch DistributedCloudPublish CRDs
  │
  ├─ On Create/Update:
  │   ├─ Resolve HTTPRoute → get hostnames, paths, backends
  │   ├─ Resolve NGF Gateway external IP/LB address
  │   ├─ Call XC API: Create/Update Origin Pool
  │   │   └─ Origin server = NGF external address
  │   ├─ Call XC API: Create/Update HTTP Load Balancer
  │   │   ├─ Attach WAF policy
  │   │   ├─ Attach bot defense
  │   │   ├─ Configure TLS
  │   │   └─ Set routes to match HTTPRoute paths
  │   ├─ Update CRD status with XC object references
  │   └─ Store sync metadata in annotations
  │
  ├─ On Delete:
  │   ├─ Call XC API: Delete HTTP Load Balancer
  │   ├─ Call XC API: Delete Origin Pool
  │   └─ Clean up annotations
  │
  └─ Periodic Reconciliation (every 60s):
      ├─ Check XC object health
      ├─ Update CRD status
      └─ Detect drift (XC config changed outside controller)
```

**XC API Integration:**
- Authentication: API token stored in Kubernetes Secret
- Base URL: `https://<tenant>.console.ves.volterra.io/api`
- Key endpoints:
  - `POST /config/namespaces/<ns>/http_loadbalancers` — create HTTP LB
  - `POST /config/namespaces/<ns>/origin_pools` — create origin pool
  - `POST /config/namespaces/<ns>/app_firewalls` — reference WAF policy
  - `GET /config/namespaces/<ns>/http_loadbalancers/<name>` — check status

#### 3.8.2 UI for XC Publishing

In the HTTPRoute creation/edit form:

```
┌─────────────────────────────────────────────────────┐
│ ☑ Publish via F5 Distributed Cloud                  │
│                                                      │
│   WAF Profile:       [Default ▼]                    │
│   Bot Defense:       [● Enabled  ○ Disabled]        │
│   DDoS Protection:   [● Standard ○ Advanced]        │
│   Public Hostname:   [app.example.com        ]      │
│   TLS:               [● Managed by XC  ○ BYOC]     │
│   XC Namespace:      [production ▼]                 │
│                                                      │
│   Status: ● Published — https://app.example.com     │
└─────────────────────────────────────────────────────┘
```

**Dashboard Integration:**
- Unified metrics: NGF (internal) + XC (edge) in single view
- WAF event correlation: XC block events shown on route dashboard
- End-to-end latency breakdown: Client → XC → NGF → Backend
- Security posture score per route: WAF ✓, Bot ✓, DDoS ✓, TLS ✓

---

### 3.9 NGINX Ingress Controller Migration Tool

#### 3.9.1 Purpose

Provide an on-ramp for existing F5/NGINX customers running NGINX Ingress Controller (the commercial KIC product) to migrate to NGINX Gateway Fabric.

#### 3.9.2 Supported Input Formats

- Kubernetes Ingress resources (with NGINX-specific annotations)
- NGINX Ingress Controller ConfigMap (global settings)
- VirtualServer / VirtualServerRoute CRDs (NGINX KIC custom resources)
- TransportServer CRDs
- Policy CRDs (rate limiting, JWT, OIDC, WAF, etc.)
- Exported NGINX config files (nginx.conf)

#### 3.9.3 Migration Workflow

**Step 1 — Import**
- Upload YAML files or paste content
- Or: connect to source cluster and auto-discover Ingress/VirtualServer resources
- Or: import from a Git repository URL
- API: `POST /api/v1/migrationplans` (creates MigrationPlan CRD; operator analyzes source and populates `.status.analysis`)

**Step 2 — Analysis**
- Parse all resources and build a dependency graph
- Identify:
  - Direct mappings: Ingress path → HTTPRoute
  - Annotation translations: `nginx.org/proxy-read-timeout` → BackendSettingsPolicy
  - Unsupported features: features with no Gateway API equivalent
  - Enterprise requirements: features needing enterprise NGF
- Generate a migration report with:
  - Confidence score per resource (high/medium/low)
  - Warnings and manual review items
  - Feature gap analysis
- API: `GET /api/v1/migrationplans/{ns}/{name}` (reads `.status.analysis` from MigrationPlan CRD)

**Step 3 — Review & Edit**
- Side-by-side view: source Ingress/VS ↔ generated Gateway API resources
- Inline editing of generated resources
- Toggle individual migrations on/off
- Resolve warnings interactively

**Step 4 — Apply**
- User sets `.spec.apply = true` on MigrationPlan CRD (via UI toggle)
- Operator applies resources phased: one at a time, validates, pauses (configurable `pauseBetween`), continues
- UI polls `.status.resourceStatuses[]` for per-resource progress
- Rollback: user sets `.spec.rollback = true` — operator deletes applied resources in reverse order

**Annotation Translation Table (partial):**

| NGINX KIC Annotation | Gateway API Equivalent |
|---|---|
| `nginx.org/proxy-connect-timeout` | ClientSettingsPolicy or BackendTLSPolicy |
| `nginx.org/proxy-read-timeout` | ClientSettingsPolicy |
| `nginx.org/server-snippets` | SnippetsFilter (Enterprise) |
| `nginx.org/location-snippets` | SnippetsFilter (Enterprise) |
| `nginx.org/lb-method` | HTTPRoute backendRef weight + custom policy |
| `nginx.org/rate-limiting` | RateLimitPolicy |
| `nginx.org/jwt-*` | External auth integration |
| `nginx.org/waf-*` | APPolicy (Enterprise) |

**VirtualServer CRD Translation:**

| VS Field | Gateway API Equivalent |
|---|---|
| `upstreams` | Service backendRefs |
| `routes[].path` | HTTPRoute rules[].matches[].path |
| `routes[].splits` | HTTPRoute backendRefs with weights |
| `routes[].matches.headers` | HTTPRoute rules[].matches[].headers |
| `routes[].action.redirect` | HTTPRouteFilter (RequestRedirect) |
| `routes[].action.proxy.rewritePath` | HTTPRouteFilter (URLRewrite) |
| `routes[].errorPages` | No direct equivalent (SnippetsFilter) |
| `policies[].rateLimit` | RateLimitPolicy |
| `tls.secret` | Gateway listener TLS certificateRef |

#### 3.9.4 CLI Mode

The migration tool also runs as a standalone CLI for CI/CD integration:

```bash
# Scan source cluster
ngf-migrate scan --kubeconfig=source.yaml --namespace=production

# Generate migration plan
ngf-migrate plan --input=scan-output.yaml --output=migration-plan/

# Apply migration
ngf-migrate apply --plan=migration-plan.yaml --kubeconfig=target.yaml --dry-run  # CLI creates MigrationPlan CRD

# Validate migration
ngf-migrate validate --kubeconfig=target.yaml --namespace=production
```

---

### 3.10 Inference Management (Gateway Inference Extensions)

This is the centerpiece capability of NGF Console — the first gateway management UI built for AI/ML inference workloads.

#### 3.10.1 Overview

NGF Console provides native support for the Kubernetes Gateway Inference Extensions, which introduce the InferencePool resource as a GPU/LLM-aware replacement for standard Kubernetes Services. Instead of round-robin load balancing, an Endpoint Picker (EPP) routes inference requests based on real-time backend telemetry from model serving infrastructure (NVIDIA Triton, vLLM, TGI).

**Key Concepts:**
- **InferencePool**: A CRD defining a pool of model-serving pods (e.g., Triton) with GPU-aware routing
- **Endpoint Picker (EPP)**: A component that scrapes backend metrics and makes LLM-aware routing decisions
- **GPU-Aware Metrics**: Queue depth, KV-cache utilization, prefix cache state, GPU memory/utilization (DCGM)

#### 3.10.2 InferencePool Creation Wizard

A guided workflow oriented toward ML platform engineers who understand GPUs and models but may not be Kubernetes networking experts.

**Step 1 — Model Backend Selection**
- Select serving backend: NVIDIA Triton (primary), vLLM, TGI (future)
- Model name/identifier (e.g., "llama-3-70b", "mixtral-8x7b")
- Backend version and container image
- GPU type and count per pod (A100, H100, L40S, etc.)
- API: `GET /api/v1/inference/backends`

**Step 2 — Pool Configuration**
- InferencePool name, namespace, labels
- Pod selector (match existing Triton deployment or create new)
- GPU affinity / node selector / tolerations
- Min/max replicas
- Resource requests: GPU count, GPU memory, CPU, RAM per pod
- NVIDIA MPS (Multi-Process Service) configuration if sharing GPUs
- API: (collected into InferenceStack CRD spec — no direct pool creation)

**Step 3 — EPP Configuration**
- Metric scraping interval (default: 5s)
- Triton /metrics endpoint path
- Routing strategy selector with visual explanation:
  - **Least Queue Depth**: Route to pod with shortest request queue (default)
  - **Best KV-Cache Available**: Route to pod with most available KV-cache memory
  - **Prefix Cache Affinity**: Route similar prompts to same pod for cache hits
  - **Composite**: Weighted combination of all signals (advanced)
- Strategy weights (for composite): queue_depth=0.4, kv_cache=0.3, prefix_cache=0.3
- DCGM integration toggle (enables GPU utilization safety-net metrics)
- API: (EPP config is part of InferenceStack CRD `.spec.epp`)

**Step 4 — Autoscaling Policy**
- Scaling backend selector: HPA (native), KEDA (recommended for inference)
- Scaling triggers with visual threshold configuration:
  - Queue depth threshold (e.g., "Scale up when avg queue > 5 for 60s")
  - KV-cache utilization threshold (e.g., "Scale up when avg KV-cache > 80%")
  - GPU utilization threshold via DCGM (e.g., "Safety net: scale up when GPU util > 90% for 120s")
  - Request rate / tokens-per-second
- Cool-down period configuration
- Min/max replica bounds
- Preview: "What-if" simulator showing how current traffic would trigger scaling under these rules
- Generates: HPA resource or KEDA ScaledObject + Prometheus Adapter config or KEDA trigger
- API: (autoscaling config is part of InferenceStack CRD `.spec.autoscaling`)

**Step 5 — Gateway Attachment**
- Select or create Gateway listener for inference traffic
- HTTPRoute generation with path/header matching for model endpoints
- Request timeout configuration (inference requests are long-running)
- Streaming response support (SSE/chunked for token-by-token output)
- API: (gateway attachment is part of InferenceStack CRD `.spec.gatewayAttachment`)

**Step 6 — Review & Deploy**
- Complete YAML preview of all generated resources:
  - InferencePool CRD
  - EPP configuration
  - HTTPRoute
  - HPA or KEDA ScaledObject
  - DCGM Exporter DaemonSet (if not already present)
  - Prometheus Adapter ConfigMap (if using HPA)
- Dry-run validation
- Deploy with progress tracking
- API: `POST /api/v1/inferencestacks` (creates InferenceStack CRD — operator reconciles InferencePool + EPP + HTTPRoute + KEDA + DCGM)
- API: `GET /api/v1/inferencestacks/{ns}/{name}` (polls reconciliation status from CRD `.status`)

#### 3.10.3 Inference Observability Dashboard

**Real-Time EPP Decision Visualization**

A live view showing how EPP distributes requests across the pool:
- Each pod rendered as a card showing:
  - Pod name, GPU type, node
  - Queue depth (bar graph, color-coded)
  - KV-cache utilization (percentage ring)
  - Prefix cache state indicator
  - GPU utilization + memory (from DCGM)
  - Current requests in-flight
- Animated request flow showing incoming requests being routed to selected pods
- EPP decision overlay: click any routed request to see WHY that pod was chosen
  - "Selected pod-3: queue_depth=2 (lowest), kv_cache=45% (available), prefix_hit=true"
- WebSocket-driven, 1-second refresh

**Inference Metrics Dashboard Panels:**

| Panel | Source | Description |
|-------|--------|-------------|
| Time-to-First-Token (TTFT) | ClickHouse | Distribution histogram per model/pool — the #1 metric for LLM teams |
| Tokens-per-Second (TPS) | ClickHouse + Prometheus | Throughput per pod and aggregate |
| KV-Cache Heatmap | Triton /metrics via EPP | Pool-wide KV-cache utilization — see hot spots instantly |
| Queue Depth Over Time | Triton /metrics via EPP | Correlate with request rate to validate scaling thresholds |
| Prefix Cache Hit Rate | Triton /metrics via EPP | Effectiveness of prefix-affinity routing |
| GPU Utilization | DCGM Exporter | Per-GPU utilization and memory across all pool pods |
| GPU Memory Pressure | DCGM Exporter | Track memory headroom, predict OOM events |
| Scaling Events Timeline | K8s Events + HPA status | When HPA/KEDA scaled up/down, which metric triggered it |
| Request Latency Breakdown | ClickHouse | Queue wait → inference time → token generation → response |
| Model Throughput Comparison | ClickHouse | Compare TTFT and TPS across different models/pools |
| Cost Estimation | Computed | Estimated hourly/daily GPU cost based on pool size + cloud pricing |

**ClickHouse Schema for Inference Telemetry:**

```sql
CREATE TABLE ngf_inference_logs (
    timestamp DateTime64(3),
    inference_pool String,
    model_name String,
    model_version String,
    pod_name String,
    node_name String,
    gpu_id UInt8,
    gpu_type LowCardinality(String),      -- A100, H100, L40S
    request_id String,
    trace_id String,

    -- Inference performance
    time_to_first_token_ms Float64,
    total_inference_time_ms Float64,
    tokens_generated UInt32,
    input_tokens UInt32,
    output_tokens UInt32,
    tokens_per_second Float32,

    -- EPP decision context
    epp_selected_reason LowCardinality(String),  -- least_queue, prefix_hit, kv_available, composite
    epp_decision_latency_us Float32,              -- time EPP took to decide
    queue_depth_at_selection UInt16,
    kv_cache_pct_at_selection Float32,
    prefix_cache_hit Boolean,
    candidate_pods_considered UInt8,

    -- GPU state at routing decision
    gpu_utilization_pct Float32,
    gpu_memory_used_mb UInt32,
    gpu_memory_total_mb UInt32,
    gpu_temperature_c UInt16,

    -- Scaling context
    pool_replica_count UInt16,
    pool_target_replica_count UInt16,

    -- Standard HTTP fields
    status UInt16,
    client_ip String,
    path String,
    method LowCardinality(String),
    request_size UInt64,
    response_size UInt64,

    -- XC fields (when published via Distributed Cloud)
    xc_edge_latency_ms Float64,
    xc_waf_action LowCardinality(String),
    xc_bot_classification LowCardinality(String)

) ENGINE = MergeTree()
PARTITION BY toYYYYMMDD(timestamp)
ORDER BY (inference_pool, model_name, timestamp)
TTL timestamp + INTERVAL 14 DAY;

-- Inference-specific rollups
CREATE MATERIALIZED VIEW ngf_inference_metrics_1m
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMMDD(window_start)
ORDER BY (inference_pool, model_name, window_start)
TTL window_start + INTERVAL 90 DAY
AS SELECT
    toStartOfMinute(timestamp) AS window_start,
    inference_pool,
    model_name,
    count() AS request_count,
    avg(time_to_first_token_ms) AS avg_ttft,
    quantile(0.50)(time_to_first_token_ms) AS p50_ttft,
    quantile(0.95)(time_to_first_token_ms) AS p95_ttft,
    quantile(0.99)(time_to_first_token_ms) AS p99_ttft,
    avg(tokens_per_second) AS avg_tps,
    sum(tokens_generated) AS total_tokens,
    avg(queue_depth_at_selection) AS avg_queue_depth,
    avg(kv_cache_pct_at_selection) AS avg_kv_cache_pct,
    countIf(prefix_cache_hit = true) / count() AS prefix_cache_hit_rate,
    avg(gpu_utilization_pct) AS avg_gpu_util,
    avg(gpu_memory_used_mb) AS avg_gpu_mem_used,
    max(gpu_memory_used_mb) AS max_gpu_mem_used,
    avg(epp_decision_latency_us) AS avg_epp_latency
FROM ngf_inference_logs
GROUP BY window_start, inference_pool, model_name;
```

#### 3.10.4 Autoscaling Management & Visualization

**Scaling Policy Builder UI:**
- Visual threshold editor with sliders and real-time preview
- Multi-signal scaling rules (combine queue depth + KV-cache + GPU utilization)
- Preview mode: overlay current metrics against proposed thresholds to show when scaling would trigger
- Cost impact estimation: "Adding 1 replica at current GPU pricing = +$X/hour"
- Generates appropriate K8s resources (HPA or KEDA ScaledObject)

**Scaling Timeline View:**
- Combined visualization showing:
  - Request rate (RPS) overlay
  - Queue depth across pool
  - GPU utilization aggregate
  - Replica count changes (scale-up / scale-down events)
  - Cost per hour (computed from replica count × GPU pricing)
- Annotations on scaling events showing which metric triggered the change
- "Missed scaling" detection: highlight periods where metrics exceeded thresholds but scaling didn't occur (misconfigured HPA, cool-down too long, etc.)

**Cost Optimization Insights:**
- Over-provisioning detection: "Your pool averaged 2 active pods but maintained 5 replicas overnight"
- Recommendation engine: "Based on 7-day patterns, scheduled scaling to 2 replicas at 10pm and 6 replicas at 8am would save ~$X/month"
- GPU type comparison: "Switching from A100 to H100 would increase TPS by ~40% — break-even in 3 months at current usage"

#### 3.10.5 Inference-Aware Troubleshooting

**"Why Is My Inference Slow?" Diagnostic Wizard**

Interactive wizard checking (in order):
1. **EPP Health**: Is the EPP running and scraping Triton /metrics?
2. **Queue Depth**: Are any pods showing high queue depth? (>10 = warning, >25 = critical)
3. **KV-Cache Exhaustion**: Is KV-cache utilization >90% on any pod? (causes context swapping delays)
4. **Prefix Cache Effectiveness**: Is prefix routing working? Low hit rate may indicate scattered request distribution
5. **GPU Saturation**: GPU utilization >95%? Memory >90%? (DCGM metrics)
6. **Thermal Throttling**: GPU temperature >85°C? (DCGM)
7. **Scaling Responsiveness**: Is HPA/KEDA responding to demand? Time since last scale event?
8. **Gateway Overhead**: Is NGF itself adding latency? (compare total latency vs inference time)
9. **Network Issues**: MTU mismatches, DNS resolution delays for pod-to-pod communication?

Output: Prioritized findings with specific remediation steps.

API: `GET /api/v1/inference/diagnostics/slow-inference` (read-only analysis from Prometheus + ClickHouse)

**Request Replay Tool:**
- Capture an inference request (prompt, parameters, headers)
- Show what EPP would do with it NOW vs what it did at the original time
- Compare: "At 2:05pm, pod-3 was chosen (queue=2, kv=45%). Right now, pod-1 would be chosen (queue=0, kv=12%)."
- Useful for debugging intermittent latency issues

API: `POST /api/v1/inference/diagnostics/replay` (read-only replay from ClickHouse logs — does not modify cluster)

**Model Performance Benchmarking:**
- Run standardized benchmark prompts against an InferencePool
- Measure TTFT, TPS, total latency across pool sizes
- Generate performance profile: "At 1 replica: p95 TTFT=280ms, at 3 replicas: p95 TTFT=95ms"
- Compare models: "Llama-3-70b vs Mixtral-8x7b on your current GPU fleet"

API: `POST /api/v1/inference/diagnostics/benchmark` (creates ephemeral benchmark Job — not CRD-managed)

#### 3.10.6 Inference + F5 Distributed Cloud

The XC auto-publish workflow extends to inference endpoints with additional capabilities:

**Global Inference Routing:**
- Publish InferencePool endpoints to XC for global distribution
- XC routes inference requests to the nearest cluster with available GPU capacity
- Multi-region failover: if GPU pool in region A is saturated, route to region B
- Geographic compliance: keep certain model inference in specific regions

**LLM Security at the Edge:**
- WAF rules tuned for LLM endpoints: prompt injection detection, output filtering
- Per-tenant / per-API-key rate limiting at XC edge (critical for multi-tenant LLM platforms)
- DDoS protection specifically for inference endpoints (GPU resources are expensive — don't waste them on attack traffic)
- Token-based billing integration: track token consumption per tenant

**Inference-Specific XC Dashboard:**
- Global inference traffic map: which regions are sending requests, latency by region
- Edge-to-inference latency breakdown: Client → XC PoP → Origin cluster → NGF → EPP → Triton pod
- Cost per region: inference cost varies by GPU availability and pricing per region

**CRD Extension for Inference Publishing:**

```yaml
apiVersion: ngf-console.f5.com/v1alpha1
kind: DistributedCloudPublish
metadata:
  name: llm-inference-xc
spec:
  # Can reference an HTTPRoute OR an InferencePool directly
  inferencePoolRef:
    name: llama3-pool
    namespace: ml-serving
  distributedCloud:
    tenant: my-tenant
    namespace: production
    wafPolicy: llm-waf-policy       # LLM-tuned WAF rules
    botDefense:
      enabled: true
    rateLimiting:
      perTenant: true                # Rate limit per API key
      defaultRate: 100               # requests per minute
      defaultTokenBudget: 50000      # tokens per minute
    publicHostname: inference.example.com
    tls:
      mode: managed
    multiRegion:
      enabled: true
      preferredRegions: ["us-west-2", "eu-west-1"]
      failoverPolicy: nearest-available
```

#### 3.10.7 Coexistence Dashboard (KIC + NGF)

Visualize the side-by-side deployment of NGINX Ingress Controller and NGINX Gateway Fabric:

- **Split View**: Left panel shows KIC-managed Ingress/VirtualServer resources, right panel shows NGF-managed Gateway API resources
- **Traffic Split**: Percentage of cluster traffic handled by each controller
- **Route Mapping**: Which routes have equivalents on both controllers (for migration tracking)
- **Migration Readiness Score**: "74% of your KIC routes have Gateway API equivalents ready — 12 routes remaining"
- **Recommendation Engine**: "These 5 KIC VirtualServer routes use only basic path matching — they can be migrated to NGF immediately"
- **Workload Segmentation**: Clearly show which workloads are best suited for each controller
  - Standard L7 / API traffic → KIC (today) → NGF (migration target)
  - LLM inference traffic → NGF + Gateway Inference Extensions (today)

API: `GET /api/v1/coexistence/overview`

#### 3.10.8 Inference API Endpoints

```
# InferencePool Management
GET    /api/v1/inference/pools
POST   /api/v1/inference/pools
GET    /api/v1/inference/pools/:namespace/:name
PUT    /api/v1/inference/pools/:namespace/:name
DELETE /api/v1/inference/pools/:namespace/:name
POST   /api/v1/inferencestacks                  # create InferenceStack CRD (operator reconciles children)
GET    /api/v1/inferencestacks/{ns}/{name}       # read InferenceStack status

# EPP Configuration
GET    /api/v1/inference/pools/:namespace/:name/epp
PUT    /api/v1/inference/pools/:namespace/:name/epp
GET    /api/v1/inference/pools/:namespace/:name/epp/decisions  # live EPP decision stream

# Autoscaling
GET    /api/v1/inference/pools/:namespace/:name/autoscaling
PUT    /api/v1/inference/pools/:namespace/:name/autoscaling
GET    /api/v1/inference/pools/:namespace/:name/autoscaling/simulate  # what-if

# Inference Metrics
GET    /api/v1/inference/metrics/summary                       # aggregate inference metrics
GET    /api/v1/inference/metrics/pool/:namespace/:name         # per-pool metrics
GET    /api/v1/inference/metrics/pool/:namespace/:name/pods    # per-pod GPU state
GET    /api/v1/inference/metrics/cost                          # cost estimation

# Inference Diagnostics
POST   /api/v1/inference/diagnostics/slow-inference            # "why is it slow" wizard
POST   /api/v1/inference/diagnostics/replay                    # request replay
POST   /api/v1/inference/diagnostics/benchmark                 # performance benchmark

# Coexistence
GET    /api/v1/coexistence/overview                            # KIC + NGF split view
GET    /api/v1/coexistence/migration-readiness                 # migration score

# WebSocket
WS     /api/v1/ws/inference/epp-decisions                      # live EPP decision stream
WS     /api/v1/ws/inference/gpu-metrics                        # live GPU metrics stream
WS     /api/v1/ws/inference/scaling-events                     # scaling event stream
```

---

## 4. Configuration Database

### 4.1 Schema Overview

**Tables:**

```sql
-- User preferences and UI state
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email VARCHAR(255),
    display_name VARCHAR(255),
    preferences JSONB,              -- theme, default namespace, dashboard layout
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Audit log of all configuration changes
CREATE TABLE audit_log (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    action VARCHAR(50),             -- create, update, delete, deploy, rollback
    resource_kind VARCHAR(100),     -- Gateway, HTTPRoute, Policy, etc.
    resource_namespace VARCHAR(255),
    resource_name VARCHAR(255),
    previous_config JSONB,          -- full resource before change
    new_config JSONB,               -- full resource after change
    diff TEXT,                      -- unified diff
    metadata JSONB,                 -- extra context (dry-run result, etc.)
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_audit_resource ON audit_log(resource_kind, resource_namespace, resource_name);
CREATE INDEX idx_audit_time ON audit_log(created_at DESC);

-- Alert rules
CREATE TABLE alert_rules (
    id UUID PRIMARY KEY,
    name VARCHAR(255),
    description TEXT,
    rule_type VARCHAR(50),          -- cert_expiry, error_rate, latency_threshold, etc.
    config JSONB,                   -- thresholds, conditions
    notification_channels JSONB,    -- [{type: "slack", webhook: "..."}, ...]
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Saved dashboard views and custom queries
CREATE TABLE saved_views (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    name VARCHAR(255),
    view_type VARCHAR(50),          -- dashboard, query, filter
    config JSONB,                   -- panel layout, filters, time range
    shared BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Migration projects
CREATE TABLE migration_projects (
    id UUID PRIMARY KEY,
    name VARCHAR(255),
    source_type VARCHAR(50),        -- cluster, file, git
    source_config JSONB,
    analysis_result JSONB,
    generated_resources JSONB,
    status VARCHAR(50),             -- importing, analyzed, reviewed, applied, completed
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- XC publish state tracking
CREATE TABLE xc_publish_state (
    id UUID PRIMARY KEY,
    httproute_namespace VARCHAR(255),
    httproute_name VARCHAR(255),
    xc_tenant VARCHAR(255),
    xc_namespace VARCHAR(255),
    xc_http_lb_name VARCHAR(255),
    xc_origin_pool_name VARCHAR(255),
    public_endpoint VARCHAR(255),
    last_sync_status VARCHAR(50),
    last_sync_time TIMESTAMPTZ,
    config_hash VARCHAR(64),        -- detect drift
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

### 4.2 SQLite Compatibility

For single-node / docker-compose deployments:
- Replace JSONB with TEXT (JSON stored as string)
- Replace UUID with TEXT
- Replace TIMESTAMPTZ with TEXT (ISO 8601)
- Replace BIGSERIAL with INTEGER PRIMARY KEY AUTOINCREMENT
- Use application-level JSON parsing

The backend API uses an ORM/query builder with adapter pattern to support both.

---

## 5. API Specification

### 5.1 REST API Routes

```
# ═══════════════════════════════════════════════════════════════════
# PARENT CRD ENDPOINTS — Operator reconciles child resources
# These are the primary write paths. UI wizards create these CRDs.
# ═══════════════════════════════════════════════════════════════════

# InferenceStack (AI Inference — the centerpiece)
GET    /api/v1/inferencestacks                          # list all InferenceStack CRDs
POST   /api/v1/inferencestacks                          # create InferenceStack CRD (operator reconciles 6+ children)
GET    /api/v1/inferencestacks/:namespace/:name          # read CRD spec + .status (aggregated health)
PUT    /api/v1/inferencestacks/:namespace/:name          # update CRD spec (operator re-reconciles)
DELETE /api/v1/inferencestacks/:namespace/:name          # delete CRD (owner refs garbage-collect children)

# GatewayBundle (Gateway + NginxProxy + WAF + Snippets)
GET    /api/v1/gatewaybundles                            # list all GatewayBundle CRDs
POST   /api/v1/gatewaybundles                            # create GatewayBundle CRD
GET    /api/v1/gatewaybundles/:namespace/:name            # read CRD spec + .status
PUT    /api/v1/gatewaybundles/:namespace/:name            # update CRD spec
DELETE /api/v1/gatewaybundles/:namespace/:name            # delete CRD

# CertificateBundle (TLS Secret + cert-manager + expiry alerting)
GET    /api/v1/certificatebundles                        # list all CertificateBundle CRDs
POST   /api/v1/certificatebundles                        # create CertificateBundle CRD
GET    /api/v1/certificatebundles/:namespace/:name        # read CRD spec + .status (expiry, renewal)
PUT    /api/v1/certificatebundles/:namespace/:name        # update CRD spec
DELETE /api/v1/certificatebundles/:namespace/:name        # delete CRD
GET    /api/v1/certificatebundles/expiring                # certs within expiry threshold

# DistributedCloudPublish (F5 XC integration)
GET    /api/v1/distributedcloudpublishes                 # list all XC publish CRDs
POST   /api/v1/distributedcloudpublishes                 # create DistributedCloudPublish CRD
GET    /api/v1/distributedcloudpublishes/:namespace/:name # read CRD spec + .status
DELETE /api/v1/distributedcloudpublishes/:namespace/:name # unpublish (delete CRD)
GET    /api/v1/xc/status                                 # XC connection status
GET    /api/v1/xc/metrics/:namespace/:name               # XC-side metrics

# MigrationPlan (KIC → NGF migration)
GET    /api/v1/migrationplans                            # list all MigrationPlan CRDs
POST   /api/v1/migrationplans                            # create MigrationPlan CRD (operator analyzes)
GET    /api/v1/migrationplans/:namespace/:name            # read CRD spec + .status.analysis
PUT    /api/v1/migrationplans/:namespace/:name            # update (set .spec.apply=true to trigger phased apply)
DELETE /api/v1/migrationplans/:namespace/:name            # delete CRD

# ═══════════════════════════════════════════════════════════════════
# LIGHTWEIGHT K8s RESOURCES — Created directly (no parent CRD)
# Routes and policies are simple enough to not need CRD wrapping.
# ═══════════════════════════════════════════════════════════════════

# Route Management (HTTPRoute, GRPCRoute, etc.)
GET    /api/v1/httproutes                                # list routes
POST   /api/v1/httproutes                                # create HTTPRoute directly
GET    /api/v1/httproutes/:namespace/:name
PUT    /api/v1/httproutes/:namespace/:name
DELETE /api/v1/httproutes/:namespace/:name
POST   /api/v1/routes/simulate                           # test request simulation (read-only)
# Same pattern for grpcroutes, tlsroutes, tcproutes, udproutes

# Policy Management (standalone K8s policy resources)
GET    /api/v1/policies/:type                            # type = ratelimit|backendtls|waf|...
POST   /api/v1/policies/:type                            # create policy resource directly
GET    /api/v1/policies/:type/:namespace/:name
PUT    /api/v1/policies/:type/:namespace/:name
DELETE /api/v1/policies/:type/:namespace/:name
GET    /api/v1/policies/conflicts                        # policy conflict analysis

# ═══════════════════════════════════════════════════════════════════
# READ-ONLY ENDPOINTS — K8s queries, metrics, diagnostics
# API server reads from K8s API, Prometheus, ClickHouse, DCGM.
# ═══════════════════════════════════════════════════════════════════

# K8s Read-Only
GET    /api/v1/gatewayclasses                            # list GatewayClasses
GET    /api/v1/gateways                                  # list Gateway resources (children of GatewayBundles)
GET    /api/v1/gateways/:namespace/:name                 # read Gateway status

# Observability (Prometheus + ClickHouse)
GET    /api/v1/metrics/summary                           # RED metrics overview
GET    /api/v1/metrics/route/:namespace/:name            # per-route metrics
GET    /api/v1/metrics/gateway/:namespace/:name          # per-gateway metrics
GET    /api/v1/metrics/inference/:namespace/:name        # per-InferenceStack GPU metrics
GET    /api/v1/logs/query                                # ClickHouse log query
GET    /api/v1/logs/topn                                 # top-N analytics

# Inference Observability (Prometheus + DCGM + ClickHouse)
GET    /api/v1/inference/metrics/gpu                     # GPU heatmap data (DCGM)
GET    /api/v1/inference/metrics/epp/:namespace/:name    # EPP routing decisions
GET    /api/v1/inference/metrics/ttft/:namespace/:name   # time-to-first-token
GET    /api/v1/inference/metrics/cost/:namespace/:name   # GPU cost estimation

# Topology
GET    /api/v1/topology                                  # full topology graph
GET    /api/v1/topology/gateway/:namespace/:name

# Diagnostics (read-only analysis)
GET    /api/v1/diagnostics/route-check                   # "Why isn't my route working?"
POST   /api/v1/diagnostics/trace                         # request tracing (read-only)
GET    /api/v1/events                                    # filtered K8s event stream
GET    /api/v1/inference/diagnostics/slow-inference       # slow inference analysis
POST   /api/v1/inference/diagnostics/replay              # replay from ClickHouse (read-only)
POST   /api/v1/inference/diagnostics/benchmark           # ephemeral benchmark Job

# Audit
GET    /api/v1/audit                                     # audit log entries (from config DB)
GET    /api/v1/audit/diff                                # config diff

# WebSocket Endpoints
WS     /api/v1/ws/events                                 # real-time K8s event stream
WS     /api/v1/ws/metrics                                # real-time metrics push
WS     /api/v1/ws/topology                               # topology status updates
WS     /api/v1/ws/inference/epp/:namespace/:name         # live EPP routing decisions
WS     /api/v1/ws/crd-status/:namespace/:name            # CRD reconciliation status updates
```

### 5.2 Authentication & Authorization

- Kubernetes ServiceAccount token-based auth (when running in-cluster)
- OIDC integration for user-facing auth (Keycloak, Okta, etc.)
- RBAC mapped to Kubernetes namespaces
- Role model: Admin, Operator, Viewer
- API: Bearer token in Authorization header

---

## 6. Frontend Architecture

### 6.1 Technology Stack

| Layer | Choice | Rationale |
|-------|--------|-----------|
| Framework | React 18 + TypeScript | Industry standard, strong ecosystem |
| Build | Vite | Fast dev server, optimized builds |
| Styling | Tailwind CSS | Utility-first, consistent design |
| Component Library | shadcn/ui | Accessible, customizable |
| State Management | Zustand | Lightweight, TypeScript-native |
| Data Fetching | TanStack Query (React Query) | Caching, real-time refetching |
| Routing | React Router v6 | Standard SPA routing |
| Charts | Recharts + D3 | Metrics dashboards |
| Code Editor | Monaco Editor | YAML editing, NGINX config |
| Topology Graph | React Flow | Interactive node graph |
| Forms | React Hook Form + Zod | Validation, type safety |

### 6.2 Page Structure

```
/                           → Dashboard (overview metrics + topology + inference summary)
/gateways                   → Gateway list
/gateways/create            → Gateway creation wizard
/gateways/:ns/:name         → Gateway detail + topology
/inference                  → Inference Pool overview dashboard
/inference/pools            → InferencePool list
/inference/pools/create     → InferencePool creation wizard
/inference/pools/:ns/:name  → Pool detail + EPP decisions + GPU metrics
/inference/pools/:ns/:name/epp → Live EPP decision visualizer
/inference/pools/:ns/:name/scaling → Autoscaling config + timeline
/inference/pools/:ns/:name/benchmark → Performance benchmarking
/inference/diagnostics      → "Why is my inference slow?" wizard
/inference/cost             → Cost estimation + optimization
/routes                     → Route list (all types)
/routes/create/:type        → Route creation form
/routes/:type/:ns/:name     → Route detail + metrics
/policies                   → Policy list
/policies/create/:type      → Policy builder
/policies/:type/:ns/:name   → Policy detail
/certificates               → Certificate inventory
/observability              → Main observability dashboard
/observability/logs         → ClickHouse log explorer
/observability/metrics      → Prometheus metrics browser
/diagnostics                → Troubleshooting home
/diagnostics/route-check    → Route diagnostic wizard
/diagnostics/trace          → Request tracing
/xc                         → F5 Distributed Cloud overview
/xc/publish                 → XC publish management
/coexistence                → KIC + NGF coexistence dashboard
/migration                  → Migration project list
/migration/new              → New migration wizard
/migration/:id              → Migration project detail
/settings                   → User preferences, alert rules, DB config
/audit                      → Audit log viewer
```

### 6.3 Design System

- Dark mode primary (infrastructure tools convention)
- Light mode toggle
- F5 brand colors as accents (configurable)
- Consistent status colors: green (healthy), yellow (warning), red (error), grey (unknown)
- Enterprise feature badge: purple/gold accent on enterprise-only elements

---

## 7. Deployment Architecture

### 7.1 Helm Chart Structure

```
charts/ngf-console/
├── Chart.yaml
├── values.yaml
├── templates/
│   ├── _helpers.tpl
│   ├── deployment-frontend.yaml
│   ├── deployment-api.yaml
│   ├── deployment-operator.yaml              # NGF Console Operator (replaces controller)
│   ├── service-frontend.yaml
│   ├── service-api.yaml
│   ├── ingress.yaml                          # or HTTPRoute for self-hosting on NGF
│   ├── configmap.yaml
│   ├── secret.yaml
│   ├── serviceaccount-api.yaml
│   ├── serviceaccount-operator.yaml
│   ├── clusterrole-api.yaml                  # Narrow: CRD CRUD + read-only K8s
│   ├── clusterrole-operator.yaml             # Broad: manage all child resources
│   ├── clusterrolebinding-api.yaml
│   ├── clusterrolebinding-operator.yaml
│   ├── crds/                                 # ALL parent CRDs
│   │   ├── ngf-console.f5.com_inferencestacks.yaml
│   │   ├── ngf-console.f5.com_gatewaybundles.yaml
│   │   ├── ngf-console.f5.com_migrationplans.yaml
│   │   ├── ngf-console.f5.com_distributedcloudpublishes.yaml
│   │   └── ngf-console.f5.com_certificatebundles.yaml
│   ├── clickhouse/
│   │   ├── statefulset.yaml
│   │   ├── service.yaml
│   │   ├── configmap-schema.yaml
│   │   └── pvc.yaml
│   ├── postgresql/                            # optional, when database.type=postgresql
│   │   ├── statefulset.yaml
│   │   ├── service.yaml
│   │   └── pvc.yaml
│   └── otel-collector/
│       ├── deployment.yaml
│       ├── service.yaml
│       └── configmap.yaml
└── charts/                                    # subcharts for clickhouse, postgresql
```

### 7.2 values.yaml Key Configuration

```yaml
# NGF Edition
ngf:
  edition: enterprise          # enterprise | oss
  controllerNamespace: nginx-gateway

# Frontend
frontend:
  replicas: 2
  image:
    repository: registry.f5.com/ngf-console/frontend
    tag: "0.1.0"

# API Server (Thin BFF — reads + CRD CRUD only)
api:
  replicas: 2
  image:
    repository: registry.f5.com/ngf-console/api
    tag: "0.1.0"

# NGF Console Operator (THE CONTROL PLANE)
operator:
  replicas: 2                   # leader election ensures only 1 active
  image:
    repository: registry.f5.com/ngf-console/operator
    tag: "0.1.0"
  leaderElection:
    enabled: true
    id: ngf-console-operator
  reconcileInterval: 60s        # drift detection requeue interval
  resources:
    requests:
      cpu: 100m
      memory: 256Mi
    limits:
      cpu: 500m
      memory: 512Mi

# Database
database:
  type: postgresql             # postgresql | sqlite
  postgresql:
    host: ""                   # external PG, or leave empty for bundled
    port: 5432
    database: ngf_console
    existingSecret: ""
  sqlite:
    path: /data/ngf-console.db
    persistence:
      enabled: true
      size: 1Gi

# ClickHouse
clickhouse:
  enabled: true
  replicas: 1                  # single-node for small deployments
  persistence:
    size: 50Gi
  retention:
    rawLogs: 7d
    rollups1m: 90d
    rollups1h: 365d

# OpenTelemetry Collector
otelCollector:
  enabled: true
  mode: deployment             # deployment | daemonset

# Prometheus
prometheus:
  url: http://prometheus:9090  # existing Prometheus URL

# Grafana
grafana:
  enabled: false
  url: http://grafana:3000     # link to existing Grafana

# Inference / GPU Metrics
inference:
  enabled: true
  dcgmExporter:
    enabled: true              # deploy DCGM Exporter DaemonSet for GPU metrics
    image: nvcr.io/nvidia/k8s/dcgm-exporter:3.3.5-3.4.0-ubuntu22.04
  tritonMetrics:
    scrapeInterval: 5s         # EPP scraping interval for Triton /metrics
    metricsPath: /metrics
  scaling:
    backend: keda              # keda | hpa
    kedaNamespace: keda        # namespace where KEDA is installed
  costEstimation:
    enabled: true
    gpuPricing:                # per-hour pricing by GPU type (configurable)
      A100: 3.67
      H100: 8.10
      L40S: 1.84
      T4: 0.53

# F5 Distributed Cloud
xc:
  enabled: false
  tenantUrl: ""
  apiTokenSecretRef: ""
  defaultNamespace: default
  defaultWafPolicy: ""

# Authentication
auth:
  type: kubernetes             # kubernetes | oidc
  oidc:
    issuerUrl: ""
    clientId: ""
    clientSecretRef: ""

# Ingress / HTTPRoute for the console itself
ingress:
  enabled: true
  className: nginx-gateway-fabric
  hostname: ngf-console.example.com
  tls:
    enabled: true
    secretName: ngf-console-tls
```

### 7.3 Docker-Compose (Development / Demo)

```yaml
# deploy/docker-compose/docker-compose.yaml
version: '3.8'
services:
  frontend:
    build: ../../frontend
    ports: ["3000:80"]
    depends_on: [api]

  api:
    build: ../../api
    ports: ["8080:8080"]
    environment:
      - DATABASE_TYPE=sqlite
      - DATABASE_PATH=/data/ngf-console.db
      - CLICKHOUSE_URL=http://clickhouse:8123
      - KUBECONFIG=/root/.kube/config
    volumes:
      - api-data:/data
      - ~/.kube:/root/.kube:ro
    depends_on: [clickhouse]

  clickhouse:
    image: clickhouse/clickhouse-server:24.1
    ports: ["8123:8123", "9000:9000"]
    volumes:
      - clickhouse-data:/var/lib/clickhouse
      - ./clickhouse/init.sql:/docker-entrypoint-initdb.d/init.sql

  otel-collector:
    image: otel/opentelemetry-collector-contrib:latest
    ports: ["4317:4317", "4318:4318"]
    volumes:
      - ./otel/config.yaml:/etc/otelcol-contrib/config.yaml

volumes:
  api-data:
  clickhouse-data:
```

---

## 8. Project Structure for Claude Code

```
ngf-console/
├── README.md
├── Makefile                           # build, test, lint, docker targets
├── .github/
│   └── workflows/
│       ├── ci.yaml                    # lint, test, build
│       └── release.yaml               # build + push images
│
├── frontend/
│   ├── Dockerfile
│   ├── package.json
│   ├── tsconfig.json
│   ├── vite.config.ts
│   ├── tailwind.config.js
│   ├── index.html
│   ├── public/
│   │   └── favicon.ico
│   └── src/
│       ├── main.tsx
│       ├── App.tsx
│       ├── routes.tsx
│       ├── api/                       # API client layer
│       │   ├── client.ts              # axios/fetch wrapper
│       │   ├── gateways.ts
│       │   ├── routes.ts
│       │   ├── policies.ts
│       │   ├── certificates.ts
│       │   ├── metrics.ts
│       │   ├── logs.ts
│       │   ├── xc.ts
│       │   ├── migration.ts
│       │   └── diagnostics.ts
│       ├── components/
│       │   ├── layout/
│       │   │   ├── Sidebar.tsx
│       │   │   ├── Header.tsx
│       │   │   ├── MainLayout.tsx
│       │   │   └── EnterpriseBadge.tsx
│       │   ├── gateway/
│       │   │   ├── GatewayList.tsx
│       │   │   ├── GatewayCreateWizard.tsx
│       │   │   ├── GatewayDetail.tsx
│       │   │   ├── ListenerBuilder.tsx
│       │   │   └── GatewayClassSelector.tsx
│       │   ├── routes/
│       │   │   ├── RouteList.tsx
│       │   │   ├── HTTPRouteBuilder.tsx
│       │   │   ├── MatchConditionBuilder.tsx
│       │   │   ├── TrafficSplitVisualizer.tsx
│       │   │   └── RequestSimulator.tsx
│       │   ├── policies/
│       │   │   ├── PolicyList.tsx
│       │   │   ├── PolicyBuilder.tsx
│       │   │   ├── PolicyTemplateGallery.tsx
│       │   │   ├── WafManager.tsx
│       │   │   ├── ConflictDetector.tsx
│       │   │   └── SnippetsEditor.tsx
│       │   ├── certificates/
│       │   │   ├── CertificateInventory.tsx
│       │   │   ├── CertificateDetail.tsx
│       │   │   └── ExpiryTimeline.tsx
│       │   ├── observability/
│       │   │   ├── Dashboard.tsx
│       │   │   ├── MetricsPanel.tsx
│       │   │   ├── LogExplorer.tsx
│       │   │   ├── TopNView.tsx
│       │   │   └── GrafanaLink.tsx
│       │   ├── topology/
│       │   │   ├── TopologyGraph.tsx
│       │   │   ├── TopologyNode.tsx
│       │   │   └── TopologyEdge.tsx
│       │   ├── diagnostics/
│       │   │   ├── RouteCheckWizard.tsx
│       │   │   ├── RequestTracer.tsx
│       │   │   ├── ConfigDiffViewer.tsx
│       │   │   └── EventStream.tsx
│       │   ├── xc/
│       │   │   ├── XCOverview.tsx
│       │   │   ├── XCPublishForm.tsx
│       │   │   ├── XCStatusDashboard.tsx
│       │   │   └── SecurityPostureScore.tsx
│       │   ├── inference/
│       │   │   ├── InferencePoolList.tsx
│       │   │   ├── InferencePoolCreateWizard.tsx
│       │   │   ├── InferencePoolDetail.tsx
│       │   │   ├── EPPDecisionVisualizer.tsx
│       │   │   ├── EPPConfigEditor.tsx
│       │   │   ├── GPUMetricsHeatmap.tsx
│       │   │   ├── KVCacheUtilization.tsx
│       │   │   ├── PrefixCacheMonitor.tsx
│       │   │   ├── InferenceDashboard.tsx
│       │   │   ├── TTFTDistribution.tsx
│       │   │   ├── TokenThroughputChart.tsx
│       │   │   ├── ScalingPolicyBuilder.tsx
│       │   │   ├── ScalingTimeline.tsx
│       │   │   ├── CostEstimator.tsx
│       │   │   ├── InferenceSlowDiagnostic.tsx
│       │   │   ├── RequestReplayTool.tsx
│       │   │   ├── ModelBenchmark.tsx
│       │   │   └── QueueDepthMonitor.tsx
│       │   ├── coexistence/
│       │   │   ├── CoexistenceDashboard.tsx
│       │   │   ├── ControllerSplitView.tsx
│       │   │   └── MigrationReadinessScore.tsx
│       │   ├── migration/
│       │   │   ├── MigrationWizard.tsx
│       │   │   ├── ImportStep.tsx
│       │   │   ├── AnalysisReport.tsx
│       │   │   ├── SideBySideDiff.tsx
│       │   │   └── MigrationProgress.tsx
│       │   ├── common/
│       │   │   ├── YAMLPreview.tsx
│       │   │   ├── StatusBadge.tsx
│       │   │   ├── NamespaceSelector.tsx
│       │   │   ├── TimeRangeSelector.tsx
│       │   │   ├── ConfirmDialog.tsx
│       │   │   └── EnterpriseGate.tsx  # wraps enterprise features
│       │   └── audit/
│       │       ├── AuditLog.tsx
│       │       └── AuditDiffView.tsx
│       ├── hooks/
│       │   ├── useWebSocket.ts
│       │   ├── useEdition.ts          # OSS vs Enterprise detection
│       │   ├── useMetrics.ts
│       │   └── useAuth.ts
│       ├── store/
│       │   ├── index.ts
│       │   ├── gatewayStore.ts
│       │   ├── metricsStore.ts
│       │   └── settingsStore.ts
│       ├── types/
│       │   ├── gateway.ts
│       │   ├── route.ts
│       │   ├── policy.ts
│       │   ├── metrics.ts
│       │   ├── xc.ts
│       │   └── migration.ts
│       └── utils/
│           ├── yaml.ts
│           ├── validation.ts
│           └── formatting.ts
│
├── api/
│   ├── Dockerfile
│   ├── go.mod
│   ├── go.sum
│   ├── main.go
│   ├── cmd/
│   │   └── server/
│   │       └── main.go
│   ├── internal/
│   │   ├── server/
│   │   │   ├── server.go              # HTTP server setup
│   │   │   ├── middleware.go           # auth, logging, CORS
│   │   │   └── websocket.go           # WS handler
│   │   ├── handlers/
│   │   │   ├── gateways.go
│   │   │   ├── routes.go
│   │   │   ├── policies.go
│   │   │   ├── certificates.go
│   │   │   ├── metrics.go
│   │   │   ├── logs.go
│   │   │   ├── topology.go
│   │   │   ├── diagnostics.go
│   │   │   ├── inference.go           # InferencePool CRUD + EPP + scaling
│   │   │   ├── inference_metrics.go   # inference-specific metrics/dashboards
│   │   │   ├── inference_diag.go      # inference diagnostics + benchmark
│   │   │   ├── coexistence.go         # KIC + NGF coexistence view
│   │   │   ├── xc.go
│   │   │   ├── migration.go
│   │   │   └── audit.go
│   │   ├── kubernetes/
│   │   │   ├── client.go              # K8s client wrapper
│   │   │   ├── gateway.go             # Gateway API operations
│   │   │   ├── inference.go           # InferencePool + EPP operations
│   │   │   ├── watcher.go             # informer-based resource watching
│   │   │   └── edition.go             # OSS vs Enterprise detection
│   │   ├── clickhouse/
│   │   │   ├── client.go
│   │   │   ├── queries.go
│   │   │   └── schema.go
│   │   ├── database/
│   │   │   ├── interface.go           # DB adapter interface
│   │   │   ├── postgresql.go
│   │   │   ├── sqlite.go
│   │   │   └── migrations/
│   │   │       ├── 001_initial.sql
│   │   │       └── 002_xc_state.sql
│   │   ├── prometheus/
│   │   │   ├── client.go
│   │   │   └── queries.go
│   │   ├── inference/
│   │   │   ├── pool_manager.go        # InferencePool lifecycle
│   │   │   ├── epp_client.go          # EPP metrics scraping + decision stream
│   │   │   ├── triton_client.go       # Triton /metrics scraper
│   │   │   ├── dcgm_client.go         # DCGM GPU metrics
│   │   │   ├── scaling.go             # HPA/KEDA ScaledObject generation
│   │   │   ├── cost.go                # GPU cost estimation
│   │   │   ├── diagnostics.go         # Inference diagnostics engine
│   │   │   └── benchmark.go           # Performance benchmark runner
│   │   ├── coexistence/
│   │   │   ├── detector.go            # Detect KIC + NGF installations
│   │   │   ├── mapper.go              # Map KIC routes to NGF equivalents
│   │   │   └── readiness.go           # Migration readiness scoring
│   │   ├── xc/
│   │   │   ├── client.go              # F5 XC API client
│   │   │   ├── types.go
│   │   │   └── sync.go
│   │   └── migration/
│   │       ├── parser.go              # Ingress/VS/TS parsers
│   │       ├── analyzer.go            # compatibility analysis
│   │       ├── translator.go          # → Gateway API translation
│   │       └── annotations.go         # annotation mapping table
│   └── pkg/
│       ├── types/
│       │   └── api.go                 # shared API types
│       └── version/
│           └── version.go
│
├── operator/                                  # THE CONTROL PLANE (controller-runtime)
│   ├── Dockerfile
│   ├── go.mod
│   ├── go.sum
│   ├── cmd/
│   │   └── main.go                            # Registers all controllers (see §2.5.6)
│   ├── api/
│   │   └── v1alpha1/                          # CRD Go type definitions
│   │       ├── groupversion_info.go           # SchemeBuilder + GroupVersion
│   │       ├── inferencestack_types.go        # InferenceStack CRD (§2.5.3)
│   │       ├── gatewaybundle_types.go         # GatewayBundle CRD (§2.5.3)
│   │       ├── migrationplan_types.go         # MigrationPlan CRD (§2.5.3)
│   │       ├── distributedcloudpublish_types.go
│   │       ├── certificatebundle_types.go     # CertificateBundle CRD (§2.5.3)
│   │       ├── common_types.go               # ChildStatus, ThresholdSpec, etc.
│   │       └── zz_generated.deepcopy.go      # Generated by controller-gen
│   ├── internal/
│   │   ├── controller/
│   │   │   ├── common.go                      # hashSpec, setOwnerRef, conditions (§2.5.4)
│   │   │   ├── drift.go                       # Drift detection engine (§2.5.5)
│   │   │   ├── inferencestack_controller.go   # InferenceStack reconciler
│   │   │   ├── inferencestack_controller_test.go
│   │   │   ├── inferencestack_children.go     # buildDesiredInferencePool, etc.
│   │   │   ├── gatewaybundle_controller.go
│   │   │   ├── gatewaybundle_controller_test.go
│   │   │   ├── gatewaybundle_children.go
│   │   │   ├── migrationplan_controller.go    # Phased apply logic
│   │   │   ├── migrationplan_controller_test.go
│   │   │   ├── distributedcloudpublish_controller.go
│   │   │   ├── distributedcloudpublish_controller_test.go
│   │   │   ├── certificatebundle_controller.go
│   │   │   └── certificatebundle_controller_test.go
│   │   ├── xc/                                # F5 XC API client
│   │   │   ├── client.go
│   │   │   ├── types.go
│   │   │   └── client_test.go
│   │   ├── migration/                         # Migration analysis engine
│   │   │   ├── parser.go                      # Ingress/VirtualServer/nginx.conf
│   │   │   ├── analyzer.go                    # Confidence scoring
│   │   │   ├── translator.go                  # → Gateway API translation
│   │   │   └── annotations.go                 # Annotation mapping table
│   │   └── edition/
│   │       └── detector.go                    # OSS vs Enterprise detection
│   ├── config/
│   │   ├── crd/
│   │   │   └── bases/                         # Generated by `make generate-crds`
│   │   │       ├── ngf-console.f5.com_inferencestacks.yaml
│   │   │       ├── ngf-console.f5.com_gatewaybundles.yaml
│   │   │       ├── ngf-console.f5.com_migrationplans.yaml
│   │   │       ├── ngf-console.f5.com_distributedcloudpublishes.yaml
│   │   │       └── ngf-console.f5.com_certificatebundles.yaml
│   │   ├── rbac/
│   │   │   ├── operator_role.yaml             # Broad perms (§2.5.9)
│   │   │   ├── operator_role_binding.yaml
│   │   │   ├── api_role.yaml                  # Narrow perms (§2.5.9)
│   │   │   └── api_role_binding.yaml
│   │   ├── manager/
│   │   │   └── manager.yaml
│   │   └── samples/                           # Example CRD instances
│   │       ├── inferencestack_triton.yaml
│   │       ├── inferencestack_vllm.yaml
│   │       ├── gatewaybundle_production.yaml
│   │       ├── migrationplan_from_kic.yaml
│   │       └── distributedcloudpublish_llm.yaml
│   └── hack/
│       └── boilerplate.go.txt
│
├── migration-cli/
│   ├── Dockerfile
│   ├── go.mod
│   ├── main.go
│   └── cmd/
│       ├── scan.go
│       ├── plan.go
│       ├── apply.go
│       └── validate.go
│
├── deploy/
│   ├── helm/
│   │   └── ngf-console/
│   │       ├── Chart.yaml
│   │       ├── values.yaml
│   │       └── templates/
│   │           └── ... (as described in section 7.1)
│   ├── manifests/
│   │   └── install.yaml              # generated from Helm template
│   └── docker-compose/
│       ├── docker-compose.yaml
│       ├── clickhouse/
│       │   └── init.sql
│       └── otel/
│           └── config.yaml
│
└── docs/
    ├── architecture.md                # Overall architecture + operator pattern
    ├── operator-guide.md              # Operator internals: CRDs, reconciliation, drift detection
    ├── crd-reference.md               # Full CRD API reference (InferenceStack, GatewayBundle, etc.)
    ├── gitops-guide.md                # Using NGF Console CRDs with ArgoCD/Flux
    ├── development.md                 # Dev setup, running locally, testing
    ├── api-reference.md               # REST API reference (thin BFF endpoints)
    ├── deployment-guide.md            # Helm, kubectl, docker-compose
    ├── migration-guide.md             # KIC → NGF migration
    └── xc-integration.md             # F5 Distributed Cloud setup
```

---

## 9. Development Priorities (Phased Approach)

### Phase 1 — Foundation + Inference Observability (Weeks 1-6)

**Goal:** Ship the inference dashboard first — it's the differentiator and the demo magnet. Stand up the operator with InferenceStack CRD.

- [ ] Project scaffolding (monorepo, CI/CD, Docker builds)
- [ ] **Operator scaffolding: controller-runtime, scheme registration, leader election**
- [ ] **InferenceStack CRD types (api/v1alpha1/) + controller-gen CRD generation**
- [ ] **InferenceStack controller: reconcile InferencePool + EPP ConfigMap (read/create/update children)**
- [ ] **Owner references, finalizers, basic status aggregation**
- [ ] Backend API server with K8s client — **thin BFF: CRD CRUD + read aggregation only**
- [ ] Frontend shell: layout, routing, sidebar, edition detection
- [ ] InferencePool list view with GPU status (reads from InferenceStack .status)
- [ ] EPP decision visualizer (live WebSocket view)
- [ ] GPU metrics heatmap (DCGM integration)
- [ ] KV-cache utilization + queue depth monitoring
- [ ] Inference dashboard: TTFT, TPS, queue depth, GPU util panels
- [ ] ClickHouse deployment + OTel pipeline (inference telemetry)
- [ ] Gateway + Route list views with status
- [ ] Basic topology graph (read-only, React Flow)
- [ ] Helm chart (basic deployment with operator + ClickHouse + OTel)
- [ ] Docker-compose for local development
- [ ] **envtest integration tests for InferenceStack controller**

### Phase 2 — Inference Management + Configuration (Weeks 7-14)

**Goal:** Enable active InferencePool lifecycle and gateway management via operator-managed CRDs

- [ ] InferencePool creation wizard (full 6-step flow → **creates InferenceStack CRD**)
- [ ] **InferenceStack controller: reconcile KEDA ScaledObject + HTTPRoute + DCGM children**
- [ ] **GatewayBundle CRD types + controller (reconcile Gateway + NginxProxy + WAF + Snippets)**
- [ ] **Drift detection engine: periodic reconciliation + Owns() watches**
- [ ] EPP configuration editor (routing strategy, weights)
- [ ] Autoscaling policy builder (HPA/KEDA with visual threshold config)
- [ ] Scaling timeline visualization
- [ ] Cost estimation dashboard
- [ ] Gateway creation wizard (full 5-step flow → **creates GatewayBundle CRD**)
- [ ] HTTPRoute builder with YAML preview
- [ ] Policy builder (rate limit, backend TLS, client settings)
- [ ] WAF management (Enterprise)
- [ ] Configuration DB (PostgreSQL + SQLite support)
- [ ] Prometheus metrics dashboard (RED metrics for standard traffic)
- [ ] **CertificateBundle CRD types + controller**
- [ ] Certificate inventory with expiry tracking
- [ ] Audit log with config diffing
- [ ] **envtest integration tests for all controllers (drift, self-heal, deletion)**

### Phase 3 — XC Integration + Migration + Diagnostics (Weeks 15-22)

**Goal:** Revenue-driving XC features, customer on-ramp, and operational tooling — all operator-managed

- [ ] **DistributedCloudPublish controller (supports HTTPRoute AND InferencePool refs)**
- [ ] **InferenceStack → auto-creates DistributedCloudPublish child when XC spec present**
- [ ] XC publish UI (route builder + inference pool)
- [ ] XC-specific inference features: multi-region routing, LLM WAF, per-tenant rate limiting
- [ ] Unified NGF + XC metrics view (including inference edge-to-origin latency)
- [ ] Security posture score
- [ ] **MigrationPlan CRD types + controller (phased apply with rollback)**
- [ ] NGINX Ingress Controller migration tool (UI + CLI → **creates MigrationPlan CRD**)
- [ ] VirtualServer/VirtualServerRoute translation
- [ ] Coexistence dashboard (KIC + NGF side-by-side)
- [ ] Migration readiness scoring
- [ ] "Why isn't my route working?" diagnostic wizard
- [ ] "Why is my inference slow?" diagnostic wizard
- [ ] Request tracing + inference request replay
- [ ] **GitOps documentation: using CRDs with ArgoCD/Flux**

### Phase 4 — Advanced Operations (Weeks 23-30)

**Goal:** Enterprise-grade operational tooling and advanced inference capabilities

- [ ] RBAC and multi-user support
- [ ] OIDC authentication
- [ ] Model performance benchmarking tool
- [ ] Cost optimization recommendations engine
- [ ] Traffic splitting with canary automation
- [ ] Request simulation tool
- [ ] Custom ClickHouse query editor
- [ ] SnippetsFilter editor (Enterprise)
- [ ] Multi-cluster inference pool management (future)
- [ ] Capacity planning dashboards
- [ ] Log explorer with full-text search
- [ ] Alert rule configuration (cert expiry, error rate, GPU saturation)

---

## 10. Build & Development Commands

```bash
# Development
make dev                    # Start all services in dev mode
make dev-frontend           # Frontend dev server (Vite, hot reload)
make dev-api               # API server with air (Go hot reload)
make dev-operator          # Operator with air (Go hot reload, needs cluster)
make dev-compose           # Start docker-compose dependencies

# Build
make build                 # Build all images
make build-frontend        # Build frontend (npm run build)
make build-api            # Build API (go build)
make build-operator       # Build operator (go build)
make build-migration-cli  # Build migration CLI

# Docker
make docker-build         # Build all container images
make docker-push          # Push to registry

# Helm
make helm-package         # Package Helm chart
make helm-install         # Install to current cluster
make helm-template        # Render templates (dry run)

# Test
make test                 # Run all tests
make test-frontend        # Jest + React Testing Library
make test-api            # Go test ./...
make test-operator       # Go test with envtest (controller integration tests)
make test-e2e            # Playwright E2E tests

# Lint
make lint                 # Run all linters
make lint-frontend        # ESLint + Prettier
make lint-api            # golangci-lint (api/)
make lint-operator       # golangci-lint (operator/)

# Generate (IMPORTANT — run after changing CRD types)
make generate            # Run all generators
make generate-crds       # controller-gen: Go types → CRD YAML (operator/config/crd/bases/)
make generate-deepcopy   # controller-gen: Generate zz_generated.deepcopy.go
make generate-rbac       # controller-gen: Generate RBAC from +kubebuilder:rbac markers
make generate-api        # Generate OpenAPI spec from API handlers
make generate-manifests  # Generate kubectl manifests from Helm
```

---

## 11. Key Technical Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| **Architecture pattern** | **Kubernetes Operator (controller-runtime)** | **Declarative control plane, not a kubectl proxy. Enables drift detection, self-healing, GitOps compatibility, and status aggregation. This is the most important decision in the project.** |
| Backend language | Go | Same as NGF itself, strong K8s ecosystem, controller-runtime |
| Frontend framework | React + TypeScript | Largest ecosystem, strong typing |
| Config DB | PostgreSQL / SQLite | PG for production scale, SQLite for simplicity. K8s state lives in CRDs, NOT in the DB. |
| Analytics DB | ClickHouse | Columnar, time-series optimized for access logs + inference telemetry |
| Telemetry pipeline | OpenTelemetry | Vendor-neutral, standard, pluggable exporters |
| GPU metrics | NVIDIA DCGM Exporter | Industry standard for GPU telemetry, Prometheus-compatible |
| Inference metrics | Triton /metrics + EPP | Direct scraping for real-time EPP decisions |
| Autoscaling | KEDA (preferred) / HPA | KEDA supports custom metric triggers natively, better for inference |
| Real-time updates | WebSocket | Lower latency than polling for EPP decisions/GPU metrics |
| Container runtime | Multi-stage Docker builds | Small images, reproducible builds |
| Parent CRDs | InferenceStack, GatewayBundle, MigrationPlan, DistributedCloudPublish, CertificateBundle | One parent CRD per workflow. Operator reconciles all children. UI writes CRDs, not child resources. |
| API server role | Thin BFF (CRD CRUD + read aggregation) | API server does NOT manage child K8s resources. Operator does. |
| Drift detection | 60s periodic reconciliation + Owns() watches | Catches both deletion drift (via watches) and modification drift (via periodic comparison) |
| GitOps | CRDs are the GitOps primitive | ArgoCD/Flux manages parent CRDs in Git, operator manages children in cluster |
| Migration tool | Go CLI + MigrationPlan CRD | CLI for CI/CD, CRD for operator-managed phased apply with rollback |

---

## 12. Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| **Inference pool MTTR** | <5 min (from 45+ min) | Inference diagnostic wizard usage |
| **Inference pool creation time** | <3 min via wizard | Wizard completion time |
| **EPP routing visibility** | 100% of decisions traceable | EPP decision log completeness |
| Gateway creation time | <2 min (from 10+ min YAML) | Wizard completion time |
| Route issue MTTR | <5 min (from 30+ min) | Diagnostic wizard usage |
| XC publish adoption | 40% of routes + inference pools XC-published within 6 months | DistributedCloudPublish CRD count |
| Migration conversion | 80% of KIC customers migrated within 12 months | Migration tool usage |
| Active users | 500+ within 6 months of GA | Auth session tracking |
| Inference GPU cost savings | 15-25% reduction via scaling optimization | Cost dashboard before/after |
| NPS | 50+ | In-app survey |
