# NGF Console — Multi-Cluster Architecture Specification

## Addendum to NGF-UI-SPEC.md

### Version: 0.2.0-alpha
### Author: Dan / F5 Product Incubation
### Date: February 2026

> **For Claude Code:** This spec EXTENDS the existing single-cluster NGF Console architecture (NGF-UI-SPEC.md). Assume all existing CRDs (InferenceStack, GatewayBundle, MigrationPlan, DistributedCloudPublish, CertificateBundle), controllers, API handlers, and frontend components are already implemented as specified. This document describes **only the additions and modifications** required to make NGF Console multi-cluster aware. When this spec says "modify X," the implementer should open the existing file and add to it — not rewrite from scratch.

---

## 1. Problem Statement

The current NGF Console runs inside a single Kubernetes cluster and manages the NGINX Gateway Fabric instance in that same cluster. This is a fundamental usability limitation:

- **Enterprise customers run 5-50+ clusters** — dev, staging, prod, regional, GPU-dedicated. Each cluster has its own NGF installation.
- **Without multi-cluster**, customers must install NGF Console in every cluster and switch between browser tabs to manage them. There is no unified view of gateway health, inference capacity, or XC publishing across clusters.
- **GPU inference clusters are often separate** from general-purpose clusters. Platform teams need to see all InferenceStacks across all GPU clusters in one view, compare GPU utilization across regions, and route traffic to the cluster with the most available capacity.
- **XC multi-region routing becomes concrete** when you can see all inference endpoints across all clusters and publish them as a single globally-distributed LLM API.

**The solution:** A hub-spoke architecture where NGF Console runs as a management plane in its own cluster (or any cluster) and deploys lightweight agents into managed workload clusters.

---

## 2. Architecture: Hub-Spoke Model

### 2.1 High-Level Multi-Cluster Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        MANAGEMENT CLUSTER (Hub)                              │
│                                                                              │
│  ┌──────────────────────────────────────────────────────────────────────┐   │
│  │                         NGF Console UI                                │   │
│  │              (React + TypeScript + Tailwind)                          │   │
│  │              ┌─────────────┐                                          │   │
│  │              │ Cluster     │  ← Global cluster switcher/selector      │   │
│  │              │ Selector    │                                          │   │
│  │              └─────────────┘                                          │   │
│  ├──────────────────────────────────────────────────────────────────────┤   │
│  │                    API Server (Thin BFF Layer)                         │   │
│  │                                                                        │   │
│  │  ┌──────────────────────────────────────────────────────────────────┐ │   │
│  │  │              Multi-Cluster Client Pool                            │ │   │
│  │  │                                                                    │ │   │
│  │  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐                │ │   │
│  │  │  │ prod-east   │ │ prod-west   │ │ gpu-cluster  │  ...           │ │   │
│  │  │  │ K8s Client  │ │ K8s Client  │ │ K8s Client   │               │ │   │
│  │  │  └─────────────┘ └─────────────┘ └─────────────┘                │ │   │
│  │  └──────────────────────────────────────────────────────────────────┘ │   │
│  │                                                                        │   │
│  │  Reads: CRD .status (per-cluster), Aggregated ClickHouse, PostgreSQL  │   │
│  │  Writes: Parent CRDs into target cluster's API server                  │   │
│  ├──────────────────────────────────────────────────────────────────────┤   │
│  │                                                                        │   │
│  │  ┌───────────┐  ┌──────────────┐  ┌──────────────┐                  │   │
│  │  │ PostgreSQL│  │  ClickHouse  │  │ OTel Collector│                  │   │
│  │  │ (config + │  │  (aggregated │  │ (receives     │                  │   │
│  │  │  clusters)│  │   all clusters│  │  from agents) │                  │   │
│  │  └───────────┘  └──────────────┘  └──────────────┘                  │   │
│  └──────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  Hub has NO operator. It does NOT reconcile resources locally.               │
│  It writes CRDs into REMOTE clusters via their kubeconfigs.                  │
└──────────────┬───────────────────────┬───────────────────┬──────────────────┘
               │ kubeconfig            │ kubeconfig         │ kubeconfig
               ▼                       ▼                    ▼
┌──────────────────────┐ ┌──────────────────────┐ ┌──────────────────────────┐
│  WORKLOAD CLUSTER 1  │ │  WORKLOAD CLUSTER 2  │ │  WORKLOAD CLUSTER 3      │
│  (prod-east)         │ │  (prod-west)         │ │  (gpu-inference)         │
│                      │ │                      │ │                          │
│  ┌────────────────┐  │ │  ┌────────────────┐  │ │  ┌────────────────────┐  │
│  │ NGF Console    │  │ │  │ NGF Console    │  │ │  │ NGF Console        │  │
│  │ Agent          │  │ │  │ Agent          │  │ │  │ Agent              │  │
│  │                │  │ │  │                │  │ │  │                    │  │
│  │ ┌────────────┐ │  │ │  │ ┌────────────┐ │  │ │  │ ┌────────────────┐│  │
│  │ │ Operator   │ │  │ │  │ │ Operator   │ │  │ │  │ │ Operator       ││  │
│  │ │(controller-│ │  │ │  │ │(controller-│ │  │ │  │ │(controller-    ││  │
│  │ │ runtime)   │ │  │ │  │ │ runtime)   │ │  │ │  │ │ runtime)       ││  │
│  │ └────────────┘ │  │ │  │ └────────────┘ │  │ │  │ └────────────────┘│  │
│  │ ┌────────────┐ │  │ │  │ ┌────────────┐ │  │ │  │ ┌────────────────┐│  │
│  │ │ Telemetry  │ │  │ │  │ │ Telemetry  │ │  │ │  │ │ Telemetry      ││  │
│  │ │ Forwarder  │─┼──┼─┼──┼─┤ Forwarder  │─┼──┼─┼──┼─┤ Forwarder      ││  │
│  │ │ (OTel)     │ │  │ │  │ │ (OTel)     │ │  │ │  │ │ (OTel)         ││  │
│  │ └────────────┘ │  │ │  │ └────────────┘ │  │ │  │ └────────────────┘│  │
│  │ ┌────────────┐ │  │ │  │ ┌────────────┐ │  │ │  │ ┌────────────────┐│  │
│  │ │ Heartbeat  │ │  │ │  │ │ Heartbeat  │ │  │ │  │ │ Heartbeat      ││  │
│  │ │ Reporter   │ │  │ │  │ │ Reporter   │ │  │ │  │ │ Reporter       ││  │
│  │ └────────────┘ │  │ │  │ └────────────┘ │  │ │  │ └────────────────┘│  │
│  └────────────────┘  │ │  └────────────────┘  │ │  └────────────────────┘  │
│                      │ │                      │ │                          │
│  ┌────────────────┐  │ │  ┌────────────────┐  │ │  ┌────────────────────┐  │
│  │ NGINX Gateway  │  │ │  │ NGINX Gateway  │  │ │  │ NGINX Gateway      │  │
│  │ Fabric         │  │ │  │ Fabric         │  │ │  │ Fabric + GPU Pods  │  │
│  └────────────────┘  │ │  └────────────────┘  │ │  └────────────────────┘  │
└──────────────────────┘ └──────────────────────┘ └──────────────────────────┘
```

### 2.2 Key Architectural Decisions

**Decision 1: The hub cluster has NO operator.** The operator runs only in workload clusters. The hub's API server writes parent CRDs into remote cluster API servers using per-cluster kubeconfigs. The remote operator reconciles locally. This means:
- No "double reconciliation" (hub operator fighting remote operator)
- Each workload cluster is self-healing even if the hub goes down
- Hub is stateless from a K8s-reconciliation perspective — it's a UI + API + database

**Decision 2: The operator does not change.** The existing operator code (`operator/`) runs identically whether deployed as part of a single-cluster install or as a remote agent. No multi-cluster logic lives in the operator. It watches local CRDs and reconciles local children. Period.

**Decision 3: Multi-cluster awareness is entirely in the API server.** The API server maintains a pool of Kubernetes clients (one per managed cluster) and routes CRD operations to the correct cluster based on a `cluster` parameter in every API request.

**Decision 4: Telemetry flows from agents to hub.** Each workload cluster's OTel Collector forwards to the hub's OTel Collector, which writes to the central ClickHouse. All clusters' inference logs, access logs, and GPU telemetry are queryable from a single ClickHouse instance. Prometheus metrics stay local per-cluster — the API server queries each cluster's Prometheus directly via its kubeconfig (using the Prometheus API, not K8s API).

**Decision 5: The hub can also manage its own cluster.** If the hub cluster has NGF installed, it can register itself as a managed cluster with `isLocal: true`. This preserves single-cluster functionality — the hub just has an agent running locally.

---

## 3. New CRD: ManagedCluster

This CRD lives in the **hub cluster only**. It is NOT deployed to workload clusters. It describes a registered workload cluster.

> **Claude Code: This CRD does NOT use the operator pattern.** There is no controller watching ManagedCluster CRDs. The API server reads them directly. The reason: ManagedCluster is a hub-only management concept, not a K8s reconciliation target. Health status is updated by the API server's cluster health checker goroutine.

```go
// api/internal/multicluster/types.go (NOT in operator/api/ — this is hub-only)

package multicluster

import (
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Display Name",type=string,JSONPath=`.spec.displayName`
// +kubebuilder:printcolumn:name="Region",type=string,JSONPath=`.spec.region`
// +kubebuilder:printcolumn:name="Status",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="NGF",type=string,JSONPath=`.status.ngfVersion`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
type ManagedCluster struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`
    Spec              ManagedClusterSpec   `json:"spec,omitempty"`
    Status            ManagedClusterStatus `json:"status,omitempty"`
}

type ManagedClusterSpec struct {
    // Human-readable display name shown in UI cluster selector
    DisplayName string `json:"displayName"`

    // Logical region/location for grouping (e.g., "us-east-1", "eu-west-1")
    Region string `json:"region,omitempty"`

    // Environment label for filtering (e.g., "production", "staging", "dev")
    Environment string `json:"environment,omitempty"`

    // Arbitrary labels for grouping/filtering in the UI
    Labels map[string]string `json:"labels,omitempty"`

    // Reference to the Secret containing the kubeconfig for this cluster
    // The Secret MUST be in the same namespace as the ManagedCluster resource
    // Key in Secret: "kubeconfig" (the full kubeconfig YAML)
    KubeconfigSecretRef SecretReference `json:"kubeconfigSecretRef"`

    // Reference to the Secret containing the Prometheus URL and optional auth
    // Key in Secret: "url" (e.g., "http://prometheus.monitoring:9090")
    // Optional keys: "username", "password" (for basic auth)
    // If not set, API server will try to discover Prometheus via K8s service discovery
    PrometheusSecretRef *SecretReference `json:"prometheusSecretRef,omitempty"`

    // Is this the local cluster (hub managing itself)?
    // If true, the API server uses its in-cluster config instead of the kubeconfig
    IsLocal bool `json:"isLocal,omitempty"`

    // Expected NGF edition in this cluster (for pre-flight validation)
    NGFEdition string `json:"ngfEdition,omitempty"` // "enterprise" | "oss"

    // Agent configuration overrides for this cluster
    AgentConfig *AgentConfigOverrides `json:"agentConfig,omitempty"`
}

type AgentConfigOverrides struct {
    // Operator reconcile interval override (default: 60s)
    ReconcileInterval string `json:"reconcileInterval,omitempty"`

    // OTel Collector endpoint in the hub cluster for telemetry forwarding
    // This is set automatically during agent installation, but can be overridden
    HubOTelEndpoint string `json:"hubOTelEndpoint,omitempty"`

    // Heartbeat interval (default: 30s)
    HeartbeatInterval string `json:"heartbeatInterval,omitempty"`
}

type SecretReference struct {
    Name      string `json:"name"`
    Namespace string `json:"namespace,omitempty"` // defaults to ManagedCluster namespace
}

type ManagedClusterStatus struct {
    // Overall phase
    // +kubebuilder:validation:Enum=Pending;Connecting;Ready;Degraded;Unreachable;Error
    Phase string `json:"phase,omitempty"`

    // Kubernetes version of the remote cluster
    KubernetesVersion string `json:"kubernetesVersion,omitempty"`

    // NGF version detected in the remote cluster
    NGFVersion string `json:"ngfVersion,omitempty"`

    // NGF edition detected (enterprise or oss)
    NGFEdition string `json:"ngfEdition,omitempty"`

    // Whether the NGF Console Agent is installed and reporting
    AgentInstalled bool `json:"agentInstalled,omitempty"`

    // Agent version
    AgentVersion string `json:"agentVersion,omitempty"`

    // Last heartbeat received from the agent
    LastHeartbeat *metav1.Time `json:"lastHeartbeat,omitempty"`

    // Prometheus reachability
    PrometheusReachable bool `json:"prometheusReachable,omitempty"`

    // Summary counts from the remote cluster (refreshed periodically)
    ResourceCounts ResourceCounts `json:"resourceCounts,omitempty"`

    // GPU capacity summary (only for clusters with inference workloads)
    GPUCapacity *GPUCapacitySummary `json:"gpuCapacity,omitempty"`

    // Standard K8s conditions
    Conditions []metav1.Condition `json:"conditions,omitempty"`

    // Last time the API server successfully connected to this cluster
    LastConnectedAt *metav1.Time `json:"lastConnectedAt,omitempty"`
}

type ResourceCounts struct {
    Gateways        int `json:"gateways"`
    HTTPRoutes      int `json:"httproutes"`
    InferenceStacks int `json:"inferenceStacks"`
    InferencePools  int `json:"inferencePools"`
    Certificates    int `json:"certificates"`
    XCPublishes     int `json:"xcPublishes"`
}

type GPUCapacitySummary struct {
    // Total GPU count across all nodes
    TotalGPUs int `json:"totalGPUs"`
    // GPUs currently allocated to pods
    AllocatedGPUs int `json:"allocatedGPUs"`
    // Available GPU capacity
    AvailableGPUs int `json:"availableGPUs"`
    // GPU types present (e.g., {"H100": 8, "A100": 4})
    GPUTypes map[string]int `json:"gpuTypes,omitempty"`
    // Average GPU utilization percentage across all GPUs (from DCGM)
    AverageUtilization float64 `json:"averageUtilization,omitempty"`
}

// +kubebuilder:object:root=true
type ManagedClusterList struct {
    metav1.TypeMeta `json:",inline"`
    metav1.ListMeta `json:"metadata,omitempty"`
    Items           []ManagedCluster `json:"items"`
}
```

**Example ManagedCluster YAML:**

```yaml
apiVersion: ngf-console.f5.com/v1alpha1
kind: ManagedCluster
metadata:
  name: gpu-inference-east
  namespace: ngf-system
spec:
  displayName: "GPU Inference (US-East)"
  region: us-east-1
  environment: production
  labels:
    tier: gpu
    team: ml-platform
  kubeconfigSecretRef:
    name: gpu-inference-east-kubeconfig
    namespace: ngf-system
  prometheusSecretRef:
    name: gpu-inference-east-prometheus
    namespace: ngf-system
  ngfEdition: enterprise
---
apiVersion: v1
kind: Secret
metadata:
  name: gpu-inference-east-kubeconfig
  namespace: ngf-system
type: Opaque
data:
  kubeconfig: <base64-encoded kubeconfig>
```

---

## 4. Multi-Cluster Client Pool

The core of the multi-cluster API server: a managed pool of Kubernetes clients, one per registered ManagedCluster.

```go
// api/internal/multicluster/client_pool.go

package multicluster

import (
    "context"
    "fmt"
    "sync"
    "time"

    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/rest"
    "k8s.io/client-go/tools/clientcmd"
    "sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterClient wraps a Kubernetes client with cluster metadata
type ClusterClient struct {
    // The cluster name (ManagedCluster.metadata.name)
    Name string

    // Display name for UI
    DisplayName string

    // Region for grouping
    Region string

    // Environment
    Environment string

    // controller-runtime client (for CRD operations)
    Client client.Client

    // kubernetes clientset (for discovery, pod exec, etc.)
    Clientset *kubernetes.Clientset

    // REST config (for building dynamic clients)
    RestConfig *rest.Config

    // Prometheus URL for this cluster
    PrometheusURL string

    // Whether this is the local/hub cluster
    IsLocal bool

    // Health status
    Healthy bool
    LastHealthCheck time.Time
}

// ClientPool manages K8s clients for all registered clusters
type ClientPool struct {
    mu      sync.RWMutex
    clients map[string]*ClusterClient // key = ManagedCluster name

    // Hub's own K8s client (for reading ManagedCluster CRDs)
    hubClient client.Client

    // Namespace where ManagedCluster resources live
    namespace string
}

func NewClientPool(hubClient client.Client, namespace string) *ClientPool {
    return &ClientPool{
        clients:   make(map[string]*ClusterClient),
        hubClient: hubClient,
        namespace: namespace,
    }
}

// Sync reads all ManagedCluster CRDs and creates/updates/removes clients
// Called on startup and whenever a ManagedCluster CRD changes (via watch)
func (p *ClientPool) Sync(ctx context.Context) error {
    var clusters ManagedClusterList
    if err := p.hubClient.List(ctx, &clusters,
        client.InNamespace(p.namespace)); err != nil {
        return fmt.Errorf("listing ManagedClusters: %w", err)
    }

    p.mu.Lock()
    defer p.mu.Unlock()

    active := make(map[string]bool)
    for _, mc := range clusters.Items {
        active[mc.Name] = true

        // Skip if client already exists and kubeconfig hasn't changed
        if existing, ok := p.clients[mc.Name]; ok {
            existing.DisplayName = mc.Spec.DisplayName
            existing.Region = mc.Spec.Region
            existing.Environment = mc.Spec.Environment
            continue // TODO: detect kubeconfig secret changes via resourceVersion
        }

        // Build new client
        cc, err := p.buildClient(ctx, &mc)
        if err != nil {
            // Update ManagedCluster status to Error
            mc.Status.Phase = "Error"
            mc.Status.Conditions = append(mc.Status.Conditions, metav1.Condition{
                Type:    "Connected",
                Status:  "False",
                Reason:  "ClientBuildFailed",
                Message: err.Error(),
            })
            p.hubClient.Status().Update(ctx, &mc)
            continue
        }
        p.clients[mc.Name] = cc
    }

    // Remove clients for deleted ManagedClusters
    for name := range p.clients {
        if !active[name] {
            delete(p.clients, name)
        }
    }

    return nil
}

// Get returns the ClusterClient for a given cluster name
func (p *ClientPool) Get(name string) (*ClusterClient, error) {
    p.mu.RLock()
    defer p.mu.RUnlock()
    cc, ok := p.clients[name]
    if !ok {
        return nil, fmt.Errorf("cluster %q not registered", name)
    }
    if !cc.Healthy {
        return cc, fmt.Errorf("cluster %q is unhealthy", name)
    }
    return cc, nil
}

// List returns all registered cluster clients
func (p *ClientPool) List() []*ClusterClient {
    p.mu.RLock()
    defer p.mu.RUnlock()
    result := make([]*ClusterClient, 0, len(p.clients))
    for _, cc := range p.clients {
        result = append(result, cc)
    }
    return result
}

func (p *ClientPool) buildClient(ctx context.Context, mc *ManagedCluster) (*ClusterClient, error) {
    var restCfg *rest.Config

    if mc.Spec.IsLocal {
        // Use in-cluster config
        cfg, err := rest.InClusterConfig()
        if err != nil {
            return nil, fmt.Errorf("in-cluster config: %w", err)
        }
        restCfg = cfg
    } else {
        // Read kubeconfig from Secret
        var secret corev1.Secret
        if err := p.hubClient.Get(ctx, types.NamespacedName{
            Name:      mc.Spec.KubeconfigSecretRef.Name,
            Namespace: mc.Spec.KubeconfigSecretRef.Namespace,
        }, &secret); err != nil {
            return nil, fmt.Errorf("reading kubeconfig secret: %w", err)
        }

        kubeconfigBytes, ok := secret.Data["kubeconfig"]
        if !ok {
            return nil, fmt.Errorf("kubeconfig secret missing 'kubeconfig' key")
        }

        cfg, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
        if err != nil {
            return nil, fmt.Errorf("parsing kubeconfig: %w", err)
        }
        restCfg = cfg
    }

    // Create controller-runtime client with our CRD scheme
    c, err := client.New(restCfg, client.Options{Scheme: scheme})
    if err != nil {
        return nil, fmt.Errorf("creating client: %w", err)
    }

    cs, err := kubernetes.NewForConfig(restCfg)
    if err != nil {
        return nil, fmt.Errorf("creating clientset: %w", err)
    }

    // Resolve Prometheus URL
    promURL := ""
    if mc.Spec.PrometheusSecretRef != nil {
        var promSecret corev1.Secret
        if err := p.hubClient.Get(ctx, types.NamespacedName{
            Name:      mc.Spec.PrometheusSecretRef.Name,
            Namespace: mc.Spec.PrometheusSecretRef.Namespace,
        }, &promSecret); err == nil {
            promURL = string(promSecret.Data["url"])
        }
    }

    return &ClusterClient{
        Name:          mc.Name,
        DisplayName:   mc.Spec.DisplayName,
        Region:        mc.Spec.Region,
        Environment:   mc.Spec.Environment,
        Client:        c,
        Clientset:     cs,
        RestConfig:    restCfg,
        PrometheusURL: promURL,
        IsLocal:       mc.Spec.IsLocal,
        Healthy:       true,
        LastHealthCheck: time.Now(),
    }, nil
}
```

### 4.1 Health Checker

A background goroutine that periodically checks each cluster's connectivity and updates ManagedCluster status.

```go
// api/internal/multicluster/health_checker.go

// RunHealthChecker starts a goroutine that checks all clusters every 30s
func (p *ClientPool) RunHealthChecker(ctx context.Context, interval time.Duration) {
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            p.checkAllClusters(ctx)
        }
    }
}

func (p *ClientPool) checkAllClusters(ctx context.Context) {
    p.mu.Lock()
    defer p.mu.Unlock()

    for name, cc := range p.clients {
        checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)

        // 1. K8s API reachability
        _, err := cc.Clientset.Discovery().ServerVersion()
        if err != nil {
            cc.Healthy = false
            p.updateClusterStatus(checkCtx, name, "Unreachable", err.Error())
            cancel()
            continue
        }

        // 2. Check if agent is installed (look for agent Deployment)
        // 3. Check NGF version (look for GatewayClass)
        // 4. Check Prometheus reachability
        // 5. Collect resource counts
        // 6. Collect GPU capacity (if inference enabled)
        // 7. Check last heartbeat age

        cc.Healthy = true
        cc.LastHealthCheck = time.Now()

        // Update ManagedCluster CRD status in hub
        p.updateClusterStatus(checkCtx, name, "Ready", "")

        cancel()
    }
}

func (p *ClientPool) updateClusterStatus(ctx context.Context, name, phase, errMsg string) {
    var mc ManagedCluster
    if err := p.hubClient.Get(ctx, types.NamespacedName{
        Name: name, Namespace: p.namespace,
    }, &mc); err != nil {
        return
    }

    mc.Status.Phase = phase
    mc.Status.LastConnectedAt = &metav1.Time{Time: time.Now()}
    // ... set conditions, resource counts, etc.

    p.hubClient.Status().Update(ctx, &mc)
}
```

---

## 5. API Server Changes — Multi-Cluster Request Routing

### 5.1 Cluster Context in Every Request

Every API request that operates on a cluster's resources MUST include a `cluster` parameter. This tells the API server which cluster's K8s client to use.

**Convention:** The `cluster` parameter is passed via:
- Query parameter: `?cluster=gpu-inference-east`
- Header: `X-Cluster: gpu-inference-east`
- URL path (for some aggregation endpoints): `/api/v1/clusters/{cluster}/...`

If no cluster is specified and there is only one registered cluster, use it as default. If no cluster is specified and there are multiple clusters, return 400 with a list of available clusters.

```go
// api/internal/middleware/cluster_context.go

package middleware

import (
    "context"
    "net/http"
)

type clusterContextKey struct{}

// ClusterContext middleware extracts cluster from request and injects into context
func ClusterContext(pool *multicluster.ClientPool) func(next http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Priority: URL path > query param > header
            cluster := chi.URLParam(r, "cluster")
            if cluster == "" {
                cluster = r.URL.Query().Get("cluster")
            }
            if cluster == "" {
                cluster = r.Header.Get("X-Cluster")
            }

            if cluster == "" {
                // Check if single-cluster mode
                clients := pool.List()
                if len(clients) == 1 {
                    cluster = clients[0].Name
                } else if len(clients) == 0 {
                    http.Error(w, "no clusters registered", http.StatusServiceUnavailable)
                    return
                } else {
                    // Return available clusters
                    http.Error(w, "cluster parameter required", http.StatusBadRequest)
                    return
                }
            }

            cc, err := pool.Get(cluster)
            if err != nil {
                http.Error(w, err.Error(), http.StatusNotFound)
                return
            }

            ctx := context.WithValue(r.Context(), clusterContextKey{}, cc)
            next.ServeHTTP(w, r.WithContext(ctx))
        })
    }
}

// GetClusterClient retrieves the ClusterClient from request context
func GetClusterClient(ctx context.Context) *multicluster.ClusterClient {
    cc, _ := ctx.Value(clusterContextKey{}).(*multicluster.ClusterClient)
    return cc
}
```

### 5.2 Modifying Existing Handlers

Every existing API handler that uses `h.k8sClient` must change to use the cluster-scoped client from context. This is the most mechanical change — find every handler file and replace the client source.

**Before (single-cluster):**
```go
func (h *InferenceHandler) CreateInferenceStack(w http.ResponseWriter, r *http.Request) {
    // ...
    err := h.k8sClient.Create(r.Context(), stack)  // ← uses single client
}
```

**After (multi-cluster):**
```go
func (h *InferenceHandler) CreateInferenceStack(w http.ResponseWriter, r *http.Request) {
    cc := middleware.GetClusterClient(r.Context())
    if cc == nil {
        http.Error(w, "cluster context required", 400)
        return
    }
    // ...
    err := cc.Client.Create(r.Context(), stack)  // ← uses cluster-specific client

    // Audit log now includes cluster
    h.auditLog.Record(r.Context(), AuditEntry{
        ClusterName:  cc.Name,               // NEW FIELD
        Action:       "create",
        ResourceKind: "InferenceStack",
        ResourceName: stack.Name,
        NewConfig:    stack.Spec,
    })
}
```

> **Claude Code: This is a systematic refactor.** Every handler in `api/internal/handlers/` needs this change. The pattern is identical for all of them:
> 1. Extract `cc := middleware.GetClusterClient(r.Context())`
> 2. Replace `h.k8sClient` with `cc.Client` for all K8s operations
> 3. Add `ClusterName` to audit log entries
> 4. For Prometheus queries, use `cc.PrometheusURL` instead of `h.prometheusURL`
>
> Files to modify:
> - `api/internal/handlers/gateways.go`
> - `api/internal/handlers/routes.go`
> - `api/internal/handlers/policies.go`
> - `api/internal/handlers/certificates.go`
> - `api/internal/handlers/metrics.go` (use cluster-specific Prometheus URL)
> - `api/internal/handlers/logs.go` (ClickHouse queries add cluster filter)
> - `api/internal/handlers/topology.go`
> - `api/internal/handlers/diagnostics.go`
> - `api/internal/handlers/inference.go`
> - `api/internal/handlers/inference_metrics.go`
> - `api/internal/handlers/inference_diag.go`
> - `api/internal/handlers/coexistence.go`
> - `api/internal/handlers/xc.go`
> - `api/internal/handlers/migration.go`
> - `api/internal/handlers/audit.go` (filter by cluster)
>
> Also modify `api/internal/kubernetes/client.go` to accept a client.Client parameter instead of constructing one, or delete it entirely in favor of the client pool.

### 5.3 New Cluster Management Endpoints

```go
// api/internal/handlers/clusters.go — NEW FILE

package handlers

// GET /api/v1/clusters
// Returns all registered clusters with their status
func (h *ClusterHandler) ListClusters(w http.ResponseWriter, r *http.Request) {
    var clusters ManagedClusterList
    err := h.hubClient.List(r.Context(), &clusters, client.InNamespace(h.namespace))
    // Returns: [{name, displayName, region, environment, phase, ngfVersion,
    //            agentInstalled, resourceCounts, gpuCapacity, ...}]
}

// POST /api/v1/clusters
// Register a new cluster (creates ManagedCluster CRD + kubeconfig Secret)
func (h *ClusterHandler) RegisterCluster(w http.ResponseWriter, r *http.Request) {
    var req RegisterClusterRequest
    // req contains: displayName, region, environment, kubeconfig (base64), labels
    // 1. Create kubeconfig Secret
    // 2. Create ManagedCluster CRD
    // 3. Trigger ClientPool.Sync()
    // 4. Test connectivity
    // 5. Return cluster with initial status
}

// DELETE /api/v1/clusters/{name}
// Unregister a cluster (does NOT delete resources in the remote cluster)
func (h *ClusterHandler) UnregisterCluster(w http.ResponseWriter, r *http.Request) {
    // 1. Delete ManagedCluster CRD
    // 2. Delete kubeconfig Secret
    // 3. Trigger ClientPool.Sync()
    // Note: Agent remains in remote cluster until manually uninstalled
}

// POST /api/v1/clusters/{name}/test
// Test connectivity to a cluster
func (h *ClusterHandler) TestConnection(w http.ResponseWriter, r *http.Request) {
    // 1. Try K8s API connection
    // 2. Check for NGF GatewayClass
    // 3. Check for agent Deployment
    // 4. Check Prometheus reachability
    // 5. Return detailed connectivity report
}

// POST /api/v1/clusters/{name}/install-agent
// Generate agent installation command for a cluster
func (h *ClusterHandler) GenerateAgentInstall(w http.ResponseWriter, r *http.Request) {
    // Generates a Helm install command with pre-populated values:
    // - Hub OTel endpoint
    // - Hub API endpoint for heartbeat
    // - Cluster-specific API token
    // Returns: {helmCommand: "helm install ngf-console-agent ...", valuesYaml: "..."}
}

// GET /api/v1/clusters/summary
// Global summary across all clusters (for the global dashboard)
func (h *ClusterHandler) GlobalSummary(w http.ResponseWriter, r *http.Request) {
    // Aggregates across all clusters:
    // - Total gateways, routes, inference stacks
    // - Total GPU capacity and utilization
    // - Clusters by health status
    // - Recent drift detections across clusters
    // - Active XC publishes
}
```

### 5.4 Cross-Cluster Aggregation Endpoints

New endpoints that query across ALL clusters in parallel:

```go
// api/internal/handlers/global.go — NEW FILE

// GET /api/v1/global/inferencestacks
// Returns all InferenceStacks across all clusters
func (h *GlobalHandler) ListAllInferenceStacks(w http.ResponseWriter, r *http.Request) {
    // Query all clusters in parallel using errgroup
    var mu sync.Mutex
    var allStacks []ClusterInferenceStack // wrapper that adds cluster metadata

    g, ctx := errgroup.WithContext(r.Context())
    for _, cc := range h.pool.List() {
        cc := cc // capture
        g.Go(func() error {
            var stacks v1alpha1.InferenceStackList
            if err := cc.Client.List(ctx, &stacks); err != nil {
                return nil // skip unavailable clusters, don't fail the whole request
            }
            mu.Lock()
            for _, s := range stacks.Items {
                allStacks = append(allStacks, ClusterInferenceStack{
                    ClusterName: cc.Name,
                    ClusterDisplayName: cc.DisplayName,
                    ClusterRegion: cc.Region,
                    InferenceStack: s,
                })
            }
            mu.Unlock()
            return nil
        })
    }
    g.Wait()

    json.NewEncoder(w).Encode(allStacks)
}

// GET /api/v1/global/gateways
// Same pattern for gateways across all clusters

// GET /api/v1/global/gpu-capacity
// Aggregated GPU capacity across all clusters (for global inference routing decisions)
func (h *GlobalHandler) GlobalGPUCapacity(w http.ResponseWriter, r *http.Request) {
    // Returns: [{cluster, region, gpuType, total, allocated, available, avgUtil}]
}

// GET /api/v1/global/xc-publishes
// All DistributedCloudPublish resources across all clusters

// GET /api/v1/global/alerts
// Active alerts across all clusters

// GET /api/v1/global/drift-events
// Recent drift detection events across all clusters (from audit log)
```

---

## 6. Cross-Cluster Observability

### 6.1 Telemetry Architecture

```
Workload Cluster 1                    Hub Cluster
┌─────────────────────┐              ┌─────────────────────────────┐
│ NGF → OTel Collector │──OTLP/gRPC──▶│ Hub OTel Collector           │
│ Triton → OTel       │              │  │                           │
│ DCGM → OTel         │              │  ▼                           │
└─────────────────────┘              │ ┌─────────────────────────┐ │
                                      │ │      ClickHouse         │ │
Workload Cluster 2                    │ │  (all clusters' data)   │ │
┌─────────────────────┐              │ │                         │ │
│ NGF → OTel Collector │──OTLP/gRPC──▶│ │  cluster_name column   │ │
│ Triton → OTel       │              │ │  in every table         │ │
│ DCGM → OTel         │              │ └─────────────────────────┘ │
└─────────────────────┘              └─────────────────────────────┘
```

### 6.2 ClickHouse Schema Changes

Every existing ClickHouse table must add a `cluster_name` column. This is the only schema change needed.

```sql
-- Modify existing tables (add cluster_name as first column after timestamp)

ALTER TABLE access_logs ADD COLUMN cluster_name String DEFAULT '' AFTER timestamp;
ALTER TABLE inference_logs ADD COLUMN cluster_name String DEFAULT '' AFTER timestamp;
ALTER TABLE epp_decisions ADD COLUMN cluster_name String DEFAULT '' AFTER timestamp;
ALTER TABLE gpu_metrics ADD COLUMN cluster_name String DEFAULT '' AFTER timestamp;
ALTER TABLE scaling_events ADD COLUMN cluster_name String DEFAULT '' AFTER timestamp;

-- Add cluster_name to all partition keys and indices
-- This is critical for query performance when filtering by cluster

-- For new installations, create tables with cluster_name from the start:
-- (These replace the existing CREATE TABLE statements)

CREATE TABLE access_logs (
    timestamp DateTime64(3),
    cluster_name LowCardinality(String),      -- NEW
    namespace LowCardinality(String),
    gateway_name LowCardinality(String),
    route_name LowCardinality(String),
    method LowCardinality(String),
    path String,
    status_code UInt16,
    latency_ms Float64,
    request_size UInt64,
    response_size UInt64,
    upstream_name String,
    upstream_latency_ms Float64,
    trace_id String
) ENGINE = MergeTree()
PARTITION BY (cluster_name, toYYYYMMDD(timestamp))    -- partition by cluster
ORDER BY (cluster_name, namespace, gateway_name, timestamp)
TTL timestamp + INTERVAL 7 DAY;

CREATE TABLE inference_logs (
    timestamp DateTime64(3),
    cluster_name LowCardinality(String),      -- NEW
    namespace LowCardinality(String),
    pool_name LowCardinality(String),
    model_name LowCardinality(String),
    request_id String,
    ttft_ms Float64,
    total_latency_ms Float64,
    tokens_generated UInt32,
    tokens_per_second Float64,
    epp_decision_reason LowCardinality(String),
    selected_pod String,
    queue_depth_at_routing UInt16,
    kv_cache_at_routing Float32,
    gpu_util_at_routing Float32,
    input_tokens UInt32,
    prompt_hash String
) ENGINE = MergeTree()
PARTITION BY (cluster_name, toYYYYMMDD(timestamp))
ORDER BY (cluster_name, namespace, pool_name, timestamp)
TTL timestamp + INTERVAL 7 DAY;

-- Similar changes for gpu_metrics, epp_decisions, scaling_events tables
```

### 6.3 Agent OTel Collector Configuration

Each agent's OTel Collector adds `cluster_name` as a resource attribute and forwards to the hub:

```yaml
# agent/config/otel-collector-config.yaml

receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
      http:
        endpoint: 0.0.0.0:4318

processors:
  resource:
    attributes:
      - key: cluster_name
        value: "${CLUSTER_NAME}"        # Set via env var from ManagedCluster name
        action: upsert

exporters:
  otlp/hub:
    endpoint: "${HUB_OTEL_ENDPOINT}"    # e.g., otel-collector.ngf-system.hub-cluster:4317
    tls:
      insecure: false
      ca_file: /etc/certs/hub-ca.pem
    headers:
      x-cluster-name: "${CLUSTER_NAME}"

  # Also keep local Prometheus export for the operator's local metrics
  prometheus:
    endpoint: 0.0.0.0:8889

service:
  pipelines:
    traces:
      receivers: [otlp]
      processors: [resource]
      exporters: [otlp/hub]
    metrics:
      receivers: [otlp]
      processors: [resource]
      exporters: [otlp/hub, prometheus]
    logs:
      receivers: [otlp]
      processors: [resource]
      exporters: [otlp/hub]
```

### 6.4 ClickHouse Query Changes

All existing ClickHouse queries in `api/internal/clickhouse/queries.go` must add cluster filtering:

**Before:**
```sql
SELECT ... FROM inference_logs WHERE namespace = ? AND pool_name = ?
```

**After:**
```sql
SELECT ... FROM inference_logs WHERE cluster_name = ? AND namespace = ? AND pool_name = ?
```

For cross-cluster queries (global endpoints):
```sql
-- All inference logs across all clusters
SELECT cluster_name, ... FROM inference_logs WHERE timestamp > now() - INTERVAL 1 HOUR

-- Per-cluster aggregation
SELECT cluster_name, avg(ttft_ms), sum(tokens_generated)
FROM inference_logs
WHERE timestamp > now() - INTERVAL 1 HOUR
GROUP BY cluster_name
```

### 6.5 Prometheus Query Changes

Prometheus stays local per-cluster. The API server queries each cluster's Prometheus directly using the URL from ManagedCluster.

```go
// api/internal/prometheus/client.go — MODIFY

// Before: single Prometheus URL
type PrometheusClient struct {
    url string
}

// After: cluster-aware
type PrometheusClient struct {
    defaultURL string // fallback
}

// Query accepts a cluster-specific URL
func (c *PrometheusClient) Query(ctx context.Context, prometheusURL, query string, ts time.Time) (model.Value, error) {
    // Use prometheusURL from ClusterClient
}

// In handlers, the pattern becomes:
func (h *MetricsHandler) GetGPUMetrics(w http.ResponseWriter, r *http.Request) {
    cc := middleware.GetClusterClient(r.Context())
    result, err := h.promClient.Query(r.Context(), cc.PrometheusURL, "dcgm_gpu_utilization", time.Now())
}
```

---

## 7. NGF Console Agent (Workload Cluster Component)

### 7.1 Agent Components

The agent is a Helm chart installed in each workload cluster. It contains:

1. **NGF Console Operator** — the EXACT SAME operator binary as the single-cluster install. No code changes. It watches parent CRDs and reconciles children locally.

2. **OTel Collector** — configured to forward telemetry to the hub's OTel Collector. Adds `cluster_name` resource attribute.

3. **Heartbeat Reporter** — a lightweight sidecar or CronJob that periodically reports cluster health to the hub API server.

4. **CRD Registrations** — the 5 parent CRDs (InferenceStack, GatewayBundle, etc.) must be installed in the workload cluster so the operator can watch them.

### 7.2 Agent Helm Chart

```
charts/ngf-console-agent/
├── Chart.yaml
├── values.yaml
├── templates/
│   ├── _helpers.tpl
│   ├── deployment-operator.yaml         # Same operator image as hub chart
│   ├── deployment-otel-collector.yaml   # Forwards telemetry to hub
│   ├── deployment-heartbeat.yaml        # Reports health to hub
│   ├── serviceaccount-operator.yaml
│   ├── clusterrole-operator.yaml        # Same RBAC as single-cluster operator
│   ├── clusterrolebinding-operator.yaml
│   ├── configmap-otel.yaml              # OTel Collector config
│   ├── secret-hub-connection.yaml       # Hub API endpoint + auth token
│   ├── crds/                            # Same CRDs as hub chart
│   │   ├── ngf-console.f5.com_inferencestacks.yaml
│   │   ├── ngf-console.f5.com_gatewaybundles.yaml
│   │   ├── ngf-console.f5.com_migrationplans.yaml
│   │   ├── ngf-console.f5.com_distributedcloudpublishes.yaml
│   │   └── ngf-console.f5.com_certificatebundles.yaml
│   └── tests/
│       └── test-connection.yaml
└── charts/
```

### 7.3 Agent values.yaml

```yaml
# charts/ngf-console-agent/values.yaml

# Cluster identity (set during registration)
cluster:
  name: ""                              # REQUIRED: must match ManagedCluster name in hub
  displayName: ""

# Hub connection
hub:
  apiEndpoint: ""                       # REQUIRED: e.g., https://ngf-console.hub-cluster.example.com
  otelEndpoint: ""                      # REQUIRED: e.g., otel-collector.ngf-system.hub-cluster:4317
  authTokenSecretRef: ""                # Secret containing hub API token

# Operator (same config as hub chart's operator section)
operator:
  replicas: 2
  image:
    repository: registry.f5.com/ngf-console/operator
    tag: "0.2.0"
  leaderElection:
    enabled: true
    id: ngf-console-operator
  reconcileInterval: 60s
  resources:
    requests:
      cpu: 100m
      memory: 256Mi
    limits:
      cpu: 500m
      memory: 512Mi

# OTel Collector (forwards to hub)
otelCollector:
  image:
    repository: otel/opentelemetry-collector-contrib
    tag: "latest"
  resources:
    requests:
      cpu: 50m
      memory: 128Mi

# Heartbeat Reporter
heartbeat:
  interval: 30s
  image:
    repository: registry.f5.com/ngf-console/agent-heartbeat
    tag: "0.2.0"
```

### 7.4 Agent Installation Flow

```
1. User clicks "Add Cluster" in NGF Console UI
2. UI prompts for: cluster name, kubeconfig, region, environment
3. API server creates ManagedCluster CRD + kubeconfig Secret in hub
4. API server tests connectivity using the kubeconfig
5. API server generates Helm install command:

   helm install ngf-console-agent oci://registry.f5.com/ngf-console-agent \
     --namespace ngf-system --create-namespace \
     --set cluster.name=gpu-inference-east \
     --set hub.apiEndpoint=https://ngf-console.hub.example.com \
     --set hub.otelEndpoint=otel-collector.ngf-system.svc:4317 \
     --set hub.authTokenSecretRef=ngf-console-agent-token

6. User runs the Helm command in the target cluster
7. Agent starts, operator begins watching for CRDs
8. Heartbeat reporter contacts hub — hub updates ManagedCluster status
9. OTel Collector begins forwarding telemetry to hub
10. Cluster appears as "Ready" in the UI
```

### 7.5 Heartbeat Reporter

A simple Go binary that periodically reports cluster health to the hub API:

```go
// agent/cmd/heartbeat/main.go

package main

// Heartbeat payload sent to hub every interval
type HeartbeatPayload struct {
    ClusterName       string            `json:"clusterName"`
    AgentVersion      string            `json:"agentVersion"`
    KubernetesVersion string            `json:"kubernetesVersion"`
    NGFVersion        string            `json:"ngfVersion"`
    NGFEdition        string            `json:"ngfEdition"`
    ResourceCounts    ResourceCounts    `json:"resourceCounts"`
    GPUCapacity       *GPUCapacity      `json:"gpuCapacity,omitempty"`
    OperatorHealthy   bool              `json:"operatorHealthy"`
    Timestamp         time.Time         `json:"timestamp"`
}

// POST /api/v1/clusters/{name}/heartbeat — hub endpoint
func main() {
    // 1. Read cluster.name from env/config
    // 2. Every 30s:
    //    a. Discover K8s version
    //    b. Find NGF GatewayClass → extract version
    //    c. Count InferenceStacks, GatewayBundles, etc.
    //    d. Query node GPU capacity (nvidia.com/gpu resources)
    //    e. Check operator Deployment health
    //    f. POST payload to hub API
}
```

Hub endpoint that receives heartbeats:

```go
// api/internal/handlers/clusters.go — ADD

// POST /api/v1/clusters/{name}/heartbeat
func (h *ClusterHandler) ReceiveHeartbeat(w http.ResponseWriter, r *http.Request) {
    clusterName := chi.URLParam(r, "name")
    var payload HeartbeatPayload
    json.NewDecoder(r.Body).Decode(&payload)

    // Update ManagedCluster status
    var mc ManagedCluster
    h.hubClient.Get(r.Context(), types.NamespacedName{
        Name: clusterName, Namespace: h.namespace,
    }, &mc)

    mc.Status.Phase = "Ready"
    mc.Status.AgentInstalled = true
    mc.Status.AgentVersion = payload.AgentVersion
    mc.Status.KubernetesVersion = payload.KubernetesVersion
    mc.Status.NGFVersion = payload.NGFVersion
    mc.Status.NGFEdition = payload.NGFEdition
    mc.Status.LastHeartbeat = &metav1.Time{Time: payload.Timestamp}
    mc.Status.ResourceCounts = payload.ResourceCounts
    if payload.GPUCapacity != nil {
        mc.Status.GPUCapacity = payload.GPUCapacity
    }

    h.hubClient.Status().Update(r.Context(), &mc)
}
```

---

## 8. Database Schema Changes

### 8.1 Add cluster_name to Existing Tables

```sql
-- Migration: 003_multicluster.sql

-- Add cluster_name to audit_log
ALTER TABLE audit_log ADD COLUMN cluster_name VARCHAR(255) DEFAULT '';
CREATE INDEX idx_audit_cluster ON audit_log(cluster_name);

-- Add cluster_name to alert_rules
ALTER TABLE alert_rules ADD COLUMN cluster_name VARCHAR(255) DEFAULT '';
-- Empty cluster_name = global alert rule (applies to all clusters)

-- Add cluster_name to saved_views
ALTER TABLE saved_views ADD COLUMN cluster_name VARCHAR(255) DEFAULT '';
-- Empty cluster_name = cross-cluster view

-- Add cluster_name to migration_projects
ALTER TABLE migration_projects ADD COLUMN cluster_name VARCHAR(255) NOT NULL DEFAULT '';

-- Add cluster_name to xc_publish_state
ALTER TABLE xc_publish_state ADD COLUMN cluster_name VARCHAR(255) NOT NULL DEFAULT '';
```

### 8.2 New Tables

```sql
-- Cluster registration state (denormalized from ManagedCluster CRD for fast queries)
CREATE TABLE managed_clusters (
    name VARCHAR(255) PRIMARY KEY,
    display_name VARCHAR(255),
    region VARCHAR(255),
    environment VARCHAR(255),
    labels JSONB,
    phase VARCHAR(50),                  -- Ready, Degraded, Unreachable, Error
    kubernetes_version VARCHAR(50),
    ngf_version VARCHAR(50),
    ngf_edition VARCHAR(50),
    agent_installed BOOLEAN DEFAULT false,
    agent_version VARCHAR(50),
    last_heartbeat TIMESTAMPTZ,
    resource_counts JSONB,
    gpu_capacity JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Cluster groups for logical grouping
CREATE TABLE cluster_groups (
    id UUID PRIMARY KEY,
    name VARCHAR(255) UNIQUE,
    description TEXT,
    label_selector JSONB,               -- {"tier": "gpu"} matches clusters with that label
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

---

## 9. Frontend Changes

### 9.1 Global Cluster Selector

A persistent UI element in the header/sidebar that lets users switch between clusters or view global state.

```
┌──────────────────────────────────────────────────────────┐
│  🌐 All Clusters ▾                                       │
│  ┌────────────────────────────────────────────────────┐  │
│  │ 🌐 All Clusters (Global View)                      │  │
│  │ ─────────────────────────────────────────────────── │  │
│  │ PRODUCTION                                         │  │
│  │ ● prod-east         us-east-1    Ready    3 NGF    │  │
│  │ ● prod-west         us-west-2    Ready    2 NGF    │  │
│  │ ─────────────────────────────────────────────────── │  │
│  │ GPU CLUSTERS                                       │  │
│  │ ● gpu-inference     us-east-1    Ready    8x H100  │  │
│  │ ● gpu-staging       eu-west-1    Degraded 4x A100  │  │
│  │ ─────────────────────────────────────────────────── │  │
│  │ STAGING                                            │  │
│  │ ○ staging-1         us-east-1    Unreachable       │  │
│  │ ─────────────────────────────────────────────────── │  │
│  │ + Add Cluster                                      │  │
│  └────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────┘
```

**Behavior:**
- **"All Clusters" selected:** Dashboard shows global aggregation. Lists show resources from all clusters with a cluster badge on each row. GPU heatmap shows all clusters' GPUs.
- **Specific cluster selected:** All views filter to that cluster's resources. Behaves exactly like the current single-cluster UI. Every API call includes `?cluster=<name>`.
- **Cluster selector state persists** in localStorage and URL query parameter (`?cluster=gpu-inference`). Bookmarkable.

### 9.2 New Frontend Components

```
frontend/src/
├── components/
│   ├── clusters/                          # NEW DIRECTORY
│   │   ├── ClusterSelector.tsx            # Global header dropdown
│   │   ├── ClusterSelectorItem.tsx        # Individual cluster row
│   │   ├── ClusterManagementPage.tsx      # /clusters — list + register
│   │   ├── ClusterRegistrationWizard.tsx  # Add cluster flow
│   │   ├── ClusterDetailPage.tsx          # /clusters/:name — single cluster detail
│   │   ├── ClusterHealthCard.tsx          # Health status card
│   │   ├── ClusterAgentStatus.tsx         # Agent installation status
│   │   ├── ClusterGPUCapacity.tsx         # GPU capacity visualization
│   │   └── ClusterGroupManager.tsx        # Manage logical cluster groups
│   ├── global/                            # NEW DIRECTORY
│   │   ├── GlobalDashboard.tsx            # Cross-cluster overview
│   │   ├── GlobalInferenceView.tsx        # All InferenceStacks across clusters
│   │   ├── GlobalGPUHeatmap.tsx           # GPU utilization across all clusters
│   │   ├── GlobalGatewayList.tsx          # All gateways across clusters
│   │   ├── GlobalXCPublishes.tsx          # All XC publishes across clusters
│   │   ├── GlobalAlerts.tsx               # Alerts from all clusters
│   │   └── CrossClusterTopology.tsx       # Multi-cluster topology graph
│   └── common/
│       ├── ClusterBadge.tsx               # NEW: shows cluster name/icon on resource rows
│       └── ...existing...
├── hooks/
│   ├── useCluster.ts                      # NEW: current cluster context hook
│   ├── useClusterList.ts                  # NEW: list of all clusters
│   └── ...existing...
├── store/
│   ├── clusterStore.ts                    # NEW: selected cluster, cluster list
│   └── ...existing...
├── api/
│   ├── client.ts                          # MODIFY: add cluster param to all requests
│   ├── clusters.ts                        # NEW: cluster management API client
│   ├── global.ts                          # NEW: cross-cluster aggregation API client
│   └── ...existing...
└── types/
    ├── cluster.ts                         # NEW: ManagedCluster types
    └── ...existing...
```

### 9.3 Modifying the API Client

The API client layer must add the cluster parameter to every request:

```typescript
// frontend/src/api/client.ts — MODIFY

import { useClusterStore } from '../store/clusterStore';

const apiClient = axios.create({ baseURL: '/api/v1' });

// Interceptor: add cluster to every request
apiClient.interceptors.request.use((config) => {
  const cluster = useClusterStore.getState().selectedCluster;
  if (cluster && cluster !== '__all__') {
    config.params = { ...config.params, cluster };
  }
  return config;
});
```

### 9.4 Modifying Existing List Views

Every list view (GatewayList, RouteList, InferencePoolList, etc.) needs:

1. **When "All Clusters" is selected:** Add a "Cluster" column showing a `<ClusterBadge>` component. API calls go to global aggregation endpoints.

2. **When a specific cluster is selected:** No cluster column. API calls include `?cluster=<name>`. Behaves identically to current single-cluster UI.

> **Claude Code: This is a pattern applied to every list component.**
> Files to modify:
> - `frontend/src/components/gateway/GatewayList.tsx`
> - `frontend/src/components/routes/RouteList.tsx`
> - `frontend/src/components/policies/PolicyList.tsx`
> - `frontend/src/components/certificates/CertificateInventory.tsx`
> - `frontend/src/components/inference/InferencePoolList.tsx`
> - `frontend/src/components/xc/XCOverview.tsx`
> - `frontend/src/components/migration/MigrationWizard.tsx`
> - `frontend/src/components/observability/Dashboard.tsx`
> - `frontend/src/components/audit/AuditLog.tsx`

### 9.5 New Pages and Routes

```typescript
// frontend/src/routes.tsx — ADD

// Cluster management
{ path: '/clusters', component: ClusterManagementPage },
{ path: '/clusters/:name', component: ClusterDetailPage },
{ path: '/clusters/register', component: ClusterRegistrationWizard },

// Global views (when "All Clusters" selected)
{ path: '/global/dashboard', component: GlobalDashboard },
{ path: '/global/inference', component: GlobalInferenceView },
{ path: '/global/gpu', component: GlobalGPUHeatmap },
```

---

## 10. XC Integration Enhancement

Multi-cluster makes XC's "route to nearest cluster with GPU capacity" a real feature instead of a theoretical one.

### 10.1 Multi-Origin Pool for XC

When publishing an InferencePool to XC from the global view, the DistributedCloudPublish can reference inference endpoints across multiple clusters:

```yaml
apiVersion: ngf-console.f5.com/v1alpha1
kind: DistributedCloudPublish
metadata:
  name: global-llm-api
  namespace: ngf-system
spec:
  # NEW: multi-cluster origin
  multiClusterOrigin:
    clusterRefs:
      - clusterName: gpu-inference-east
        inferencePoolRef:
          name: llama-3-pool
          namespace: inference
        weight: 50                       # traffic weight
        priority: 1                      # failover priority
      - clusterName: gpu-inference-west
        inferencePoolRef:
          name: llama-3-pool
          namespace: inference
        weight: 50
        priority: 1
    healthCheck:
      path: /health
      interval: 10s
  distributedCloud:
    wafPolicy: llm-waf
    publicHostname: api.example.com
    tls:
      mode: managed
    loadBalancing:
      strategy: geo-proximity           # route to nearest healthy cluster
      # OR: round-robin, least-connections, gpu-capacity-aware
```

> **Claude Code:** This requires extending the DistributedCloudPublish CRD spec (in `operator/api/v1alpha1/distributedcloudpublish_types.go`) with the `MultiClusterOrigin` field. However, the XC publish controller does NOT run in workload clusters for multi-cluster origins — it runs as a separate controller in the hub that the API server invokes. This is an exception to the "hub has no operator" rule: the hub needs a lightweight XC reconciler for multi-cluster publishes. Implement this as a separate controller registered only in the hub's API server, not in the agent operator.

---

## 11. Security

### 11.1 Kubeconfig Management

- Kubeconfigs are stored as Kubernetes Secrets in the hub cluster's `ngf-system` namespace
- Secrets are encrypted at rest (standard K8s encryption config)
- Each kubeconfig should use a dedicated ServiceAccount in the remote cluster with scoped RBAC (the operator ClusterRole from the existing spec)
- The kubeconfig Secret should have an annotation recording when it was last rotated
- The UI should warn when a kubeconfig is older than 90 days

### 11.2 Agent Authentication to Hub

- The agent's heartbeat reporter authenticates to the hub API using a per-cluster API token
- Token is a long-lived JWT or opaque token stored in a Secret in both the hub and agent cluster
- Token is generated during cluster registration and included in the agent Helm install command
- The hub validates the token and matches it to the cluster name in the heartbeat payload

```go
// api/internal/middleware/agent_auth.go

// AgentAuth middleware authenticates heartbeat requests from agents
func AgentAuth(tokenStore TokenStore) func(next http.Handler) http.Handler {
    // Validates X-Agent-Token header against stored tokens per cluster
}
```

### 11.3 OTel Collector Security

- Agent OTel Collector → Hub OTel Collector uses mTLS
- CA certificate is generated during hub installation and distributed to agents via the Helm install command
- Each agent gets a unique client certificate signed by the hub CA

### 11.4 RBAC Model

```
Hub Cluster RBAC:
├── ngf-console-api ServiceAccount
│   ├── Full CRUD on ManagedCluster CRDs (hub-only)
│   ├── Read kubeconfig Secrets (to build remote clients)
│   └── Standard API server role (existing)
│
└── Per-cluster kubeconfig ServiceAccount (in remote cluster)
    ├── Full CRUD on parent CRDs (InferenceStack, GatewayBundle, etc.)
    └── Read-only on K8s resources (for topology, metrics, status)
    (This is the existing ngf-console-api ClusterRole from the single-cluster spec)

Workload Cluster RBAC:
├── ngf-console-operator ServiceAccount
│   └── Full operator ClusterRole (existing — unchanged)
│
└── ngf-console-heartbeat ServiceAccount
    ├── Read-only: GatewayClasses, nodes (for version + GPU detection)
    ├── Read-only: parent CRDs (for resource counts)
    └── Read-only: operator Deployment (for health check)
```

---

## 12. Backward Compatibility: Single-Cluster Mode

NGF Console MUST continue to work as a single-cluster install. Multi-cluster is an opt-in capability.

**Detection:** If no ManagedCluster CRDs exist in the hub cluster, the API server operates in single-cluster mode:
- Uses its in-cluster K8s config directly (no client pool)
- Cluster selector is hidden in the UI
- All API calls work without the `cluster` parameter
- The operator runs locally as before

**Upgrade path:** To go from single-cluster to multi-cluster:
1. Register the current cluster as a ManagedCluster with `isLocal: true`
2. Install the agent chart (or just point to the existing operator)
3. Begin registering additional clusters

```go
// api/internal/server/server.go — MODIFY startup

func (s *Server) Start() {
    // Check if any ManagedCluster CRDs exist
    var clusters ManagedClusterList
    err := s.hubClient.List(ctx, &clusters)

    if err != nil || len(clusters.Items) == 0 {
        // Single-cluster mode — use in-cluster config
        s.singleClusterMode = true
        s.defaultClient = buildInClusterClient()
    } else {
        // Multi-cluster mode — start client pool
        s.clientPool = NewClientPool(s.hubClient, "ngf-system")
        s.clientPool.Sync(ctx)
        go s.clientPool.RunHealthChecker(ctx, 30*time.Second)
    }
}
```

---

## 13. Container Images

```
# Existing images (unchanged)
ngf-console-frontend:0.2.0
ngf-console-api:0.2.0
ngf-console-operator:0.2.0
ngf-console-migration:0.2.0

# New image
ngf-console-agent-heartbeat:0.2.0     # Lightweight heartbeat reporter
```

The operator image is the SAME for hub and agent installs. No separate agent-operator image.

---

## 14. Project Structure Updates

```
ngf-console/
├── ...existing...
│
├── api/
│   ├── internal/
│   │   ├── multicluster/                     # NEW DIRECTORY
│   │   │   ├── types.go                      # ManagedCluster CRD types (hub-only)
│   │   │   ├── client_pool.go                # Multi-cluster client pool
│   │   │   ├── health_checker.go             # Periodic health checks
│   │   │   └── prometheus_discovery.go       # Auto-discover Prometheus in remote clusters
│   │   ├── middleware/
│   │   │   ├── cluster_context.go            # NEW: extract cluster from request
│   │   │   ├── agent_auth.go                 # NEW: authenticate agent heartbeats
│   │   │   └── ...existing...
│   │   ├── handlers/
│   │   │   ├── clusters.go                   # NEW: cluster management CRUD
│   │   │   ├── global.go                     # NEW: cross-cluster aggregation
│   │   │   └── ...existing (modified to use cluster context)...
│   │   ├── database/
│   │   │   └── migrations/
│   │   │       ├── ...existing...
│   │   │       └── 003_multicluster.sql      # NEW: add cluster_name columns
│   │   └── ...existing...
│   └── ...existing...
│
├── agent/                                     # NEW DIRECTORY
│   ├── cmd/
│   │   └── heartbeat/
│   │       └── main.go                       # Heartbeat reporter binary
│   ├── Dockerfile.heartbeat
│   └── config/
│       └── otel-collector-config.yaml        # Agent OTel config template
│
├── charts/
│   ├── ngf-console/                          # EXISTING: Hub chart (modified)
│   │   ├── values.yaml                       # Add multiCluster section
│   │   ├── templates/
│   │   │   ├── ...existing...
│   │   │   ├── crds/
│   │   │   │   ├── ...existing CRDs...
│   │   │   │   └── ngf-console.f5.com_managedclusters.yaml   # NEW
│   │   │   └── ...existing...
│   │   └── ...existing...
│   │
│   └── ngf-console-agent/                    # NEW: Agent chart
│       ├── Chart.yaml
│       ├── values.yaml
│       └── templates/
│           ├── _helpers.tpl
│           ├── deployment-operator.yaml
│           ├── deployment-otel-collector.yaml
│           ├── deployment-heartbeat.yaml
│           ├── serviceaccount-operator.yaml
│           ├── serviceaccount-heartbeat.yaml
│           ├── clusterrole-operator.yaml
│           ├── clusterrole-heartbeat.yaml
│           ├── clusterrolebinding-operator.yaml
│           ├── clusterrolebinding-heartbeat.yaml
│           ├── configmap-otel.yaml
│           ├── secret-hub-connection.yaml
│           └── crds/                         # Same CRDs as hub
│               ├── ngf-console.f5.com_inferencestacks.yaml
│               ├── ngf-console.f5.com_gatewaybundles.yaml
│               ├── ngf-console.f5.com_migrationplans.yaml
│               ├── ngf-console.f5.com_distributedcloudpublishes.yaml
│               └── ngf-console.f5.com_certificatebundles.yaml
│
├── frontend/src/
│   ├── components/
│   │   ├── clusters/                         # NEW
│   │   ├── global/                           # NEW
│   │   └── ...existing (modified)...
│   ├── api/
│   │   ├── clusters.ts                       # NEW
│   │   ├── global.ts                         # NEW
│   │   └── client.ts                         # MODIFIED: add cluster interceptor
│   ├── hooks/
│   │   ├── useCluster.ts                     # NEW
│   │   ├── useClusterList.ts                 # NEW
│   │   └── ...existing...
│   ├── store/
│   │   ├── clusterStore.ts                   # NEW
│   │   └── ...existing...
│   └── types/
│       ├── cluster.ts                        # NEW
│       └── ...existing...
│
└── ...existing...
```

---

## 15. Hub values.yaml Additions

```yaml
# charts/ngf-console/values.yaml — ADD to existing

# Multi-Cluster Configuration
multiCluster:
  enabled: false                        # Set to true to enable multi-cluster mode
  namespace: ngf-system                 # Namespace for ManagedCluster CRDs

  # Hub OTel Collector configuration (receives from agents)
  hubOTelCollector:
    enabled: true                       # Deploy hub OTel collector for receiving agent telemetry
    service:
      type: ClusterIP                   # Or LoadBalancer if agents are in different networks
      port: 4317
    tls:
      enabled: true
      certSecretRef: ngf-console-otel-tls   # TLS cert for agent connections

  # Health checker configuration
  healthChecker:
    interval: 30s                       # How often to check cluster health
    timeout: 5s                         # Per-cluster check timeout

  # Agent authentication
  agentAuth:
    tokenTTL: 8760h                     # 1 year default token TTL
```

---

## 16. API Routes — Multi-Cluster Additions

Add to the existing API routes table in Section 5.1:

```
# ═══════════════════════════════════════════════════════════════════
# CLUSTER MANAGEMENT (Hub-only)
# ═══════════════════════════════════════════════════════════════════

GET    /api/v1/clusters                                  # list all managed clusters
POST   /api/v1/clusters                                  # register a new cluster
GET    /api/v1/clusters/:name                            # get cluster detail + status
PUT    /api/v1/clusters/:name                            # update cluster config
DELETE /api/v1/clusters/:name                            # unregister cluster
POST   /api/v1/clusters/:name/test                       # test connectivity
POST   /api/v1/clusters/:name/install-agent              # generate agent install command
POST   /api/v1/clusters/:name/heartbeat                  # receive agent heartbeat (agent auth)
POST   /api/v1/clusters/:name/rotate-token               # rotate agent auth token
GET    /api/v1/clusters/summary                          # global summary across all clusters

# ═══════════════════════════════════════════════════════════════════
# CROSS-CLUSTER AGGREGATION (Global View)
# ═══════════════════════════════════════════════════════════════════

GET    /api/v1/global/inferencestacks                    # all InferenceStacks across clusters
GET    /api/v1/global/gateways                           # all Gateways across clusters
GET    /api/v1/global/httproutes                         # all HTTPRoutes across clusters
GET    /api/v1/global/gpu-capacity                       # GPU capacity across clusters
GET    /api/v1/global/xc-publishes                       # all XC publishes across clusters
GET    /api/v1/global/alerts                             # alerts from all clusters
GET    /api/v1/global/drift-events                       # drift events from all clusters

# ═══════════════════════════════════════════════════════════════════
# EXISTING ENDPOINTS — All now accept ?cluster=<name> parameter
# ═══════════════════════════════════════════════════════════════════
# (No changes to URL structure — just add cluster routing middleware)

# WebSocket Endpoints — NEW additions
WS     /api/v1/ws/cluster-status                         # real-time cluster health updates
```

---

## 17. Build & Development Commands — Additions

```bash
# Agent
make build-agent-heartbeat  # Build heartbeat reporter
make docker-build-agent     # Build agent container images

# Helm
make helm-package-agent     # Package agent Helm chart
make helm-install-agent     # Install agent to current cluster (for testing)

# Multi-cluster development
make dev-multicluster       # Start hub + 2 kind clusters + agents
make dev-register-cluster   # Register a kind cluster with the hub
make test-multicluster      # Integration tests: hub ↔ agent communication
```

---

## 18. Development Priorities — Multi-Cluster Phase

> **This is Phase 5, after the existing 4 phases are complete.**

### Phase 5a — Multi-Cluster Foundation (Weeks 31-36)

**Goal:** Hub-spoke connectivity, cluster registration, basic global views.

- [ ] ManagedCluster CRD types + controller-gen generation
- [ ] Multi-cluster client pool (`api/internal/multicluster/`)
- [ ] Cluster context middleware (add `?cluster=` routing to all existing handlers)
- [ ] Cluster management endpoints (register, unregister, test, list)
- [ ] Agent Helm chart (operator + OTel + heartbeat)
- [ ] Heartbeat reporter binary + hub endpoint
- [ ] Hub OTel Collector configured to receive from agents
- [ ] ClickHouse schema migration: add `cluster_name` columns
- [ ] Agent OTel Collector config: add `cluster_name` attribute + forward to hub
- [ ] Frontend: ClusterSelector component in header
- [ ] Frontend: ClusterManagementPage (list + register wizard)
- [ ] Frontend: modify API client to include cluster parameter
- [ ] Single-cluster backward compatibility (no ManagedClusters = single-cluster mode)
- [ ] dev-multicluster: Kind-based multi-cluster dev environment
- [ ] Integration tests: hub ↔ agent connectivity, heartbeat flow, CRD CRUD through hub

### Phase 5b — Global Views + Cross-Cluster Features (Weeks 37-42)

**Goal:** Cross-cluster dashboards, global inference management, multi-cluster XC.

- [ ] Global aggregation endpoints (`/api/v1/global/*`)
- [ ] GlobalDashboard: cross-cluster summary (cluster health, total resources, GPU capacity)
- [ ] GlobalInferenceView: all InferenceStacks across clusters with cluster badges
- [ ] GlobalGPUHeatmap: GPU utilization across all clusters in one view
- [ ] Modify all existing list views to support "All Clusters" mode (add cluster column)
- [ ] Cross-cluster ClickHouse queries (filter by cluster_name, aggregate across clusters)
- [ ] Cross-cluster Prometheus queries (query each cluster's Prometheus in parallel)
- [ ] ClusterDetailPage: per-cluster health, resources, agent status, GPU capacity
- [ ] Multi-cluster XC publish: multi-origin pools from multiple clusters
- [ ] Hub-side XC controller for multi-cluster publishes
- [ ] Cross-cluster topology graph (show clusters as top-level nodes)
- [ ] Cluster-scoped audit log (filter audit events by cluster)
- [ ] Alert rules with cluster scope (per-cluster or global)
- [ ] Agent token rotation endpoint + UI

---

## 19. Key Technical Decisions — Multi-Cluster

| Decision | Choice | Rationale |
|----------|--------|-----------|
| **Hub has no operator** | API server writes CRDs into remote clusters | Avoids double reconciliation, keeps hub stateless from K8s perspective |
| **Operator is unchanged** | Same binary for single-cluster and agent | Zero code divergence between single-cluster and multi-cluster operator |
| **Telemetry centralized in hub** | Agent OTel → Hub OTel → Hub ClickHouse | Single pane of glass for logs, one query endpoint, simpler alerting |
| **Prometheus stays local** | API server queries remote Prometheus via kubeconfig | Avoids Prometheus federation complexity, PromQL stays standard |
| **ManagedCluster is NOT operator-managed** | API server reads/writes directly, no controller | Cluster registration is a management-plane concern, not a reconciliation target |
| **Backward-compatible single-cluster** | No ManagedClusters = single-cluster mode | Zero disruption for existing installs, multi-cluster is opt-in |
| **Agent chart is separate from hub chart** | `ngf-console-agent` vs `ngf-console` | Different RBAC, different components, cleaner separation of concerns |
| **Kubeconfig-based connectivity** | ServiceAccount kubeconfig per remote cluster | Standard K8s pattern, works across cloud providers, no custom tunneling |

---

## 20. Success Metrics — Multi-Cluster

| Metric | Target |
|--------|--------|
| Cluster registration to agent-ready | < 5 minutes |
| Hub health check latency (per cluster) | < 2 seconds |
| Cross-cluster query response time | < 3 seconds for 10 clusters |
| Telemetry forwarding lag | < 30 seconds |
| Hub downtime impact on workload clusters | Zero — agents are self-healing |
| Single-cluster → multi-cluster upgrade | Zero downtime, no data loss |

---

## 21. WebSocket Multi-Cluster Routing

The existing spec defines 3 WebSocket endpoints that stream real-time data. In multi-cluster mode, these must be cluster-scoped.

### 21.1 Existing WebSocket Endpoints — Changes

```
WS /api/v1/ws/events           — streams K8s events
WS /api/v1/ws/metrics          — streams real-time metrics
WS /api/v1/ws/topology         — streams topology status updates
WS /api/v1/ws/inference/epp    — streams EPP routing decisions
WS /api/v1/ws/crd-status       — streams CRD reconciliation updates
```

All existing WebSocket endpoints accept a `cluster` query parameter on the connection URL:

```
ws://ngf-console.example.com/api/v1/ws/events?cluster=gpu-inference-east
```

### 21.2 WebSocket Handler Changes

```go
// api/internal/server/websocket.go — MODIFY

func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil { return }
    defer conn.Close()

    // Extract cluster from query params (WS doesn't use middleware)
    clusterName := r.URL.Query().Get("cluster")
    if clusterName == "" && !s.singleClusterMode {
        conn.WriteJSON(WSError{Error: "cluster parameter required"})
        return
    }

    cc, err := s.clientPool.Get(clusterName)
    if err != nil {
        conn.WriteJSON(WSError{Error: err.Error()})
        return
    }

    // For events: start K8s informer on the REMOTE cluster's API server
    // using cc.Client (NOT the hub's client)
    switch r.URL.Path {
    case "/api/v1/ws/events":
        s.streamClusterEvents(conn, cc)
    case "/api/v1/ws/metrics":
        s.streamClusterMetrics(conn, cc)
    case "/api/v1/ws/topology":
        s.streamClusterTopology(conn, cc)
    case "/api/v1/ws/inference/epp":
        s.streamClusterEPP(conn, cc)
    case "/api/v1/ws/crd-status":
        s.streamClusterCRDStatus(conn, cc)
    }
}

// Stream events from a specific cluster using its K8s client
func (s *Server) streamClusterEvents(conn *websocket.Conn, cc *ClusterClient) {
    // Create informer on cc.Client (remote cluster)
    informer := cache.NewInformer(
        cache.NewListWatchFromClient(cc.Clientset.CoreV1().RESTClient(), "events", "", fields.Everything()),
        &corev1.Event{},
        0,
        cache.ResourceEventHandlerFuncs{
            AddFunc: func(obj interface{}) {
                event := obj.(*corev1.Event)
                conn.WriteJSON(ClusterEvent{
                    ClusterName: cc.Name,
                    Event:       event,
                })
            },
        },
    )
    // Run until connection closes
    stop := make(chan struct{})
    go informer.Run(stop)
    // Wait for connection close
    for { if _, _, err := conn.ReadMessage(); err != nil { close(stop); return } }
}
```

### 21.3 Global WebSocket (All Clusters)

When the user selects "All Clusters," the frontend opens a special global WebSocket that multiplexes events from all clusters:

```
WS /api/v1/ws/global/events    — events from ALL clusters
WS /api/v1/ws/global/alerts    — alerts from ALL clusters
```

The hub server opens one informer per cluster internally and multiplexes onto the single WebSocket connection. Each message includes `clusterName`.

```go
// api/internal/server/websocket_global.go — NEW

func (s *Server) HandleGlobalWebSocket(conn *websocket.Conn) {
    stop := make(chan struct{})
    for _, cc := range s.clientPool.List() {
        cc := cc
        go func() {
            // Start informer on this cluster
            // Send events with cc.Name attached
            // On error, send degraded status for this cluster
        }()
    }
    // Wait for connection close
    for { if _, _, err := conn.ReadMessage(); err != nil { close(stop); return } }
}
```

### 21.4 Frontend WebSocket Hook Changes

```typescript
// frontend/src/hooks/useWebSocket.ts — MODIFY

import { useClusterStore } from '../store/clusterStore';

export function useWebSocket(path: string) {
  const { selectedCluster } = useClusterStore();

  const url = useMemo(() => {
    const base = `${wsBaseUrl}${path}`;
    if (selectedCluster === '__all__') {
      // Use global WebSocket endpoint
      return base.replace('/ws/', '/ws/global/');
    }
    return `${base}?cluster=${selectedCluster}`;
  }, [path, selectedCluster]);

  // Reconnect when cluster changes
  useEffect(() => {
    const ws = new WebSocket(url);
    ws.onmessage = (event) => {
      const data = JSON.parse(event.data);
      // data.clusterName is always present in multi-cluster mode
      onMessage(data);
    };
    return () => ws.close();
  }, [url]);
}
```

---

## 22. Feature Spec Multi-Cluster Modifications

> **Claude Code: This section details how EACH existing feature from Section 3 of the original spec changes in multi-cluster mode.** Do not skip any feature — every one needs at least the cluster routing change.

### 22.1 Edition Detection (3.1)

**Change:** Edition detection must happen per-cluster because different clusters may run different NGF editions.

```go
// api/internal/kubernetes/edition.go — MODIFY

// Before: checks local cluster's GatewayClass
func DetectEdition(client client.Client) (string, error)

// After: checks a specific cluster's GatewayClass
func DetectEdition(ctx context.Context, cc *ClusterClient) (string, error) {
    var gwClasses gatewayv1.GatewayClassList
    if err := cc.Client.List(ctx, &gwClasses); err != nil {
        return "unknown", err
    }
    // ... same detection logic but using cc.Client
}
```

The frontend stores edition per-cluster in clusterStore:

```typescript
// frontend/src/store/clusterStore.ts
interface ClusterInfo {
  name: string;
  displayName: string;
  region: string;
  edition: 'enterprise' | 'oss' | 'unknown';  // per-cluster
}
```

The `useEdition()` hook reads the CURRENT cluster's edition, not a global one.

### 22.2 Gateway Creation Wizard (3.2)

**Change:** The wizard must show which cluster the GatewayBundle will be created in. When "All Clusters" is selected, the wizard's first step becomes cluster selection.

```
Wizard Flow (multi-cluster):
  Step 0 — Target Cluster Selection (NEW — only shown when "All Clusters" active)
    - Cluster dropdown showing only healthy clusters
    - Shows each cluster's NGF edition, region, available resources
    - Selected cluster determines which GatewayClasses are listed in Step 1

  Step 1-5 — Unchanged (operate on selected cluster)

  Step 5 (Review) — Shows:
    - Target cluster badge: "Creating in: gpu-inference-east (US-East)"
    - GatewayBundle YAML preview
    - API call includes ?cluster=gpu-inference-east
```

> **Claude Code:** Modify `frontend/src/components/gateway/GatewayCreateWizard.tsx` to:
> 1. Check if `selectedCluster === '__all__'` — if so, prepend a ClusterSelectionStep
> 2. Pass the selected cluster to all API calls
> 3. Show a ClusterBadge in the review step

### 22.3 Traffic Management / Route Builders (3.3)

**Change:** Route creation targets the same cluster as the parent Gateway. When listing routes in "All Clusters" mode, each route shows its cluster badge. The route builder's Gateway selector lists Gateways from the currently selected cluster only.

### 22.4 Observability Dashboard (3.4)

**Change:** In "All Clusters" mode, the dashboard shows aggregated metrics. Users can drill down into per-cluster views.

```
Global Dashboard View:
┌─────────────────────────────────────────────────────────────┐
│ 📊 Observability — All Clusters                              │
│                                                              │
│ ┌──────────────┐ ┌──────────────┐ ┌──────────────────────┐  │
│ │ prod-east    │ │ prod-west    │ │ gpu-inference         │  │
│ │ 2.1k rps     │ │ 1.8k rps     │ │ 450 rps (inference)  │  │
│ │ p99: 12ms    │ │ p99: 15ms    │ │ p99: 2.1s (TTFT)     │  │
│ │ 0.2% errors  │ │ 0.1% errors  │ │ 0.05% errors         │  │
│ └──────────────┘ └──────────────┘ └──────────────────────┘  │
│                                                              │
│ ┌─ Combined Request Rate ──────────────────────────────────┐ │
│ │ [stacked area chart: each cluster is a color]            │ │
│ └──────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

> **Claude Code:** Modify `frontend/src/components/observability/Dashboard.tsx`:
> - When `selectedCluster === '__all__'`: show per-cluster summary cards + stacked metrics
> - Metrics queries use `/api/v1/global/metrics/summary` instead of per-cluster endpoint
> - ClickHouse queries group by `cluster_name`

### 22.5 Policy Management (3.5)

**Change:** Policy CRUD targets the selected cluster. In "All Clusters" mode, the policy list shows policies across all clusters with cluster badges. Conflict detection runs per-cluster (policies in different clusters cannot conflict).

### 22.6 Certificate Management (3.6)

**Change:** Certificate inventory shows certs across all clusters. The expiry timeline is global — a cert expiring in any cluster triggers global alerts.

```
Global Cert View:
┌───────────────────────────────────────────────────────────┐
│ 🔐 Certificates — All Clusters                            │
│                                                            │
│ ⚠️ 2 certificates expiring within 30 days                  │
│                                                            │
│ Cluster          | Name           | Expires    | Status    │
│ prod-east        | wildcard-prod  | Mar 12     | ⚠️ 28d    │
│ gpu-inference    | api-tls        | Mar 15     | ⚠️ 31d    │
│ prod-west        | wildcard-prod  | Jun 01     | ✅ 109d   │
└───────────────────────────────────────────────────────────┘
```

### 22.7 Troubleshooting (3.7)

**Change:** The "Why isn't my route working?" wizard needs a cluster selector. Diagnostics run against a specific cluster. Event stream can show events from all clusters or filtered.

### 22.8 XC Integration (3.8)

**Change:** Covered in Section 10 of this spec (multi-origin pools). Additionally, the XC publish form shows which cluster an InferencePool or HTTPRoute lives in when publishing.

### 22.9 Migration Tool (3.9)

**Change:** Migration targets a specific cluster. The migration wizard's Import step must know which cluster to import from (the source KIC cluster) and which cluster to migrate TO (which may be a different cluster).

```
Migration Wizard (multi-cluster):
  Step 0 — Source & Target Cluster Selection (NEW)
    - Source: "Which cluster has the NGINX Ingress Controller?"
    - Target: "Which cluster should receive the NGF resources?"
    - Can be same cluster (in-place migration) or different (cross-cluster migration)

  Step 1 — Import from SOURCE cluster
  Step 2 — Analyze compatibility against TARGET cluster's NGF
  Step 3 — Review generated resources
  Step 4 — Apply to TARGET cluster (MigrationPlan CRD in target)
```

### 22.10 Inference Management (3.10)

**Change:** This is the feature that benefits MOST from multi-cluster. The inference dashboard in "All Clusters" mode becomes a global GPU operations center.

```
Global Inference View:
┌─────────────────────────────────────────────────────────────┐
│ 🧠 Inference — All Clusters                                  │
│                                                              │
│ GPU Fleet: 48 GPUs across 3 clusters                        │
│ ┌─── gpu-east (24x H100) ──┐ ┌── gpu-west (16x H100) ──┐  │
│ │ ██████████████░░░░░░░░░░ │ │ ████████████░░░░░░░░░░░░ │  │
│ │ 62% utilized              │ │ 48% utilized              │  │
│ └───────────────────────────┘ └───────────────────────────┘  │
│ ┌─── gpu-staging (8x A100) ─┐                               │
│ │ ████░░░░░░░░░░░░░░░░░░░░ │                               │
│ │ 22% utilized              │                               │
│ └───────────────────────────┘                               │
│                                                              │
│ InferenceStacks:                                             │
│ Cluster       | Model      | Replicas | TTFT  | GPU Util   │
│ gpu-east      | llama-3    | 8        | 1.2s  | 71%        │
│ gpu-east      | mixtral    | 4        | 0.8s  | 58%        │
│ gpu-west      | llama-3    | 6        | 1.4s  | 52%        │
│ gpu-staging   | llama-3    | 2        | 2.1s  | 22%        │
│                                                              │
│ [Create InferenceStack ▾]  ← cluster selector in dropdown   │
└─────────────────────────────────────────────────────────────┘
```

> **Claude Code:** The InferencePool creation wizard (`InferencePoolCreateWizard.tsx`) needs the same Step 0 cluster selector as the Gateway wizard. The GPU heatmap (`GPUMetricsHeatmap.tsx`) in "All Clusters" mode queries `/api/v1/global/gpu-capacity` and renders all clusters' GPUs in a single view.

### 22.11 Audit Log

**Change:** The audit log stores `cluster_name` with every entry. In "All Clusters" mode, the audit log shows changes across all clusters with a cluster column. Filter by cluster.

### 22.12 Coexistence Dashboard

**Change:** Coexistence detection (KIC + NGF) is per-cluster. In "All Clusters" mode, show which clusters have coexistence situations and their migration readiness scores.

---

## 23. Network Connectivity Requirements

### 23.1 Required Network Paths

```
Hub API Server → Workload Cluster K8s API (port 443/6443)
  - REQUIRED for CRD CRUD, topology reads, version discovery
  - Uses kubeconfig — standard K8s client-go TLS
  - Must be routable from hub cluster's pods

Hub API Server → Workload Cluster Prometheus (port 9090)
  - REQUIRED for real-time metrics queries
  - Uses Prometheus HTTP API
  - Can go through K8s API server proxy if Prometheus is not externally exposed:
    /api/v1/namespaces/monitoring/services/prometheus:9090/proxy/

Hub OTel Collector ← Agent OTel Collector (port 4317 gRPC)
  - REQUIRED for telemetry forwarding
  - Agent PUSHES to hub — hub OTel endpoint must be reachable from agent
  - Options: LoadBalancer service, Ingress, or VPN

Hub API Server ← Agent Heartbeat Reporter (port 443)
  - REQUIRED for health reporting
  - Agent PUSHES to hub API — hub API endpoint must be reachable from agent
  - Same network path as user browser → hub API
```

### 23.2 When Clusters Are in Different Networks

If workload clusters are behind firewalls or in different VPCs:

**Option A: Hub has public endpoint (recommended for cloud)**
- Hub API + OTel Collector exposed via LoadBalancer or Ingress
- Agents push outbound (no inbound firewall rules needed on workload clusters)
- Hub API connects to workload K8s API via the kubeconfig's server URL (must be reachable)

**Option B: VPN/Peering**
- VPC peering or VPN between hub and workload networks
- All connections use private IPs

**Option C: K8s API Server Proxy for Prometheus**
- If Prometheus is not externally exposed, the hub can use the K8s API server's built-in service proxy:
```go
// api/internal/multicluster/prometheus_discovery.go

func (p *ClientPool) discoverPrometheus(ctx context.Context, cc *ClusterClient) (string, error) {
    // Try well-known service names
    candidates := []string{
        "monitoring/prometheus-server:9090",
        "monitoring/prometheus:9090",
        "prometheus/prometheus:9090",
    }
    for _, candidate := range candidates {
        parts := strings.SplitN(candidate, "/", 2)
        ns, svcPort := parts[0], parts[1]
        // Try K8s API proxy path
        proxyURL := fmt.Sprintf("%s/api/v1/namespaces/%s/services/%s/proxy/",
            cc.RestConfig.Host, ns, svcPort)
        // Test with a /api/v1/status call
        if testPrometheusURL(proxyURL) {
            return proxyURL, nil
        }
    }
    return "", fmt.Errorf("Prometheus not discovered; set prometheusSecretRef manually")
}
```

### 23.3 Handling Unreachable Clusters

When the hub cannot reach a workload cluster:

```go
// api/internal/multicluster/circuit_breaker.go — NEW

// Per-cluster circuit breaker to avoid blocking API responses
type ClusterCircuitBreaker struct {
    mu              sync.Mutex
    failureCount    int
    lastFailure     time.Time
    state           string // "closed" (healthy), "open" (failing), "half-open" (testing)
    failureThreshold int   // consecutive failures before opening (default: 3)
    resetTimeout     time.Duration // time before trying again (default: 30s)
}

// Wrap executes an operation with circuit breaker protection
func (cb *ClusterCircuitBreaker) Wrap(fn func() error) error {
    if cb.state == "open" && time.Since(cb.lastFailure) < cb.resetTimeout {
        return ErrCircuitOpen // skip immediately, don't block
    }
    // ... standard circuit breaker pattern
}
```

**UI behavior when a cluster is unreachable:**
- Cluster shows yellow/red status indicator in selector
- In "All Clusters" mode, resources from unreachable clusters show a "stale" badge with last-known timestamp
- CRD creation targeting an unreachable cluster returns an error with guidance: "Cluster gpu-east is unreachable. Last connected: 5 minutes ago."

---

## 24. Agent Version Compatibility & Upgrades

### 24.1 Version Compatibility Matrix

The hub API server and agent operator MUST be compatible. We use semantic versioning with the following compatibility guarantee:

- **Hub version N** supports agents at version **N, N-1, and N-2**
- Agents MUST NOT be newer than the hub (upgrade hub first, then agents)

### 24.2 Version Negotiation

The heartbeat payload includes `agentVersion`. The hub API checks compatibility:

```go
// api/internal/handlers/clusters.go — ADD to ReceiveHeartbeat

func checkVersionCompatibility(hubVersion, agentVersion string) (string, error) {
    hub := semver.MustParse(hubVersion)
    agent := semver.MustParse(agentVersion)

    if agent.Major() != hub.Major() {
        return "incompatible", fmt.Errorf("major version mismatch: hub=%s agent=%s", hubVersion, agentVersion)
    }
    if agent.Minor() > hub.Minor() {
        return "incompatible", fmt.Errorf("agent newer than hub: hub=%s agent=%s — upgrade hub first", hubVersion, agentVersion)
    }
    if hub.Minor() - agent.Minor() > 2 {
        return "deprecated", nil // warn but don't block
    }
    return "compatible", nil
}
```

### 24.3 Agent Upgrade Flow

```
1. Hub admin upgrades hub Helm chart to new version
2. Hub shows banner: "3 agents running older versions — upgrade recommended"
3. For each cluster, the UI shows an "Upgrade Agent" button
4. Button generates updated Helm upgrade command with new image tag:
   helm upgrade ngf-console-agent oci://registry.f5.com/ngf-console-agent \
     --set operator.image.tag=0.3.0 \
     --set heartbeat.image.tag=0.3.0
5. User runs command in workload cluster
6. Agent restarts, reports new version in heartbeat
7. Hub confirms compatibility
```

### 24.4 CRD Schema Evolution

If a new hub version introduces new CRD fields:
- New OPTIONAL fields: backward compatible — old agents ignore them
- New REQUIRED fields: must upgrade agent first to register the new CRD version
- CRD versions use `v1alpha1` during incubation; once stable, `v1beta1` with conversion webhooks

> **Claude Code:** The agent Helm chart includes CRD YAML in `charts/ngf-console-agent/templates/crds/`. When the agent is upgraded, the CRDs are also upgraded. The operator's scheme registration handles version differences gracefully — unknown fields in CRD specs are ignored by older operator versions.

---

## 25. Multi-Cluster Development Environment

### 25.1 Kind-Based Dev Environment

A Makefile target that creates a hub + 2 workload clusters locally using Kind:

```bash
# Makefile additions

.PHONY: dev-multicluster
dev-multicluster: dev-multicluster-create dev-multicluster-install dev-multicluster-register

.PHONY: dev-multicluster-create
dev-multicluster-create:
	@echo "Creating Kind clusters..."
	kind create cluster --name ngf-hub --config deploy/dev/kind-hub.yaml
	kind create cluster --name ngf-worker-1 --config deploy/dev/kind-worker.yaml
	kind create cluster --name ngf-worker-2 --config deploy/dev/kind-worker-gpu.yaml

.PHONY: dev-multicluster-install
dev-multicluster-install:
	@echo "Installing NGF Console hub..."
	kubectl config use-context kind-ngf-hub
	helm install ngf-console ./charts/ngf-console \
		--namespace ngf-system --create-namespace \
		--set multiCluster.enabled=true \
		--set database.type=sqlite \
		-f deploy/dev/hub-values.yaml

	@echo "Installing NGF Console agent in worker-1..."
	kubectl config use-context kind-ngf-worker-1
	helm install ngf-console-agent ./charts/ngf-console-agent \
		--namespace ngf-system --create-namespace \
		-f deploy/dev/agent-worker1-values.yaml

	@echo "Installing NGF Console agent in worker-2..."
	kubectl config use-context kind-ngf-worker-2
	helm install ngf-console-agent ./charts/ngf-console-agent \
		--namespace ngf-system --create-namespace \
		-f deploy/dev/agent-worker2-values.yaml

.PHONY: dev-multicluster-register
dev-multicluster-register:
	@echo "Registering worker clusters with hub..."
	kubectl config use-context kind-ngf-hub
	# Extract kubeconfigs and create ManagedCluster resources
	./scripts/register-kind-cluster.sh ngf-worker-1 "Worker 1" us-east-1 production
	./scripts/register-kind-cluster.sh ngf-worker-2 "GPU Worker" us-west-2 production

.PHONY: dev-multicluster-destroy
dev-multicluster-destroy:
	kind delete cluster --name ngf-hub
	kind delete cluster --name ngf-worker-1
	kind delete cluster --name ngf-worker-2
```

### 25.2 Kind Cluster Configs

```yaml
# deploy/dev/kind-hub.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
    extraPortMappings:
      - containerPort: 30080    # NGF Console UI
        hostPort: 3000
      - containerPort: 30317    # OTel Collector gRPC
        hostPort: 4317

# deploy/dev/kind-worker.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
  - role: worker
```

### 25.3 Registration Script

```bash
#!/bin/bash
# scripts/register-kind-cluster.sh
# Usage: ./register-kind-cluster.sh <cluster-name> <display-name> <region> <env>

CLUSTER_NAME=$1
DISPLAY_NAME=$2
REGION=$3
ENV=$4

# Extract kubeconfig for the Kind cluster
KIND_KUBECONFIG=$(kind get kubeconfig --name "$CLUSTER_NAME")

# Replace the server URL with the Docker-internal address
# (Kind clusters communicate via Docker network)
DOCKER_IP=$(docker inspect -f '{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}' \
    "${CLUSTER_NAME}-control-plane")
KIND_KUBECONFIG=$(echo "$KIND_KUBECONFIG" | sed "s|server: .*|server: https://${DOCKER_IP}:6443|")

# Create Secret with kubeconfig
kubectl create secret generic "${CLUSTER_NAME}-kubeconfig" \
    --namespace ngf-system \
    --from-literal=kubeconfig="$KIND_KUBECONFIG"

# Create ManagedCluster CRD
cat <<EOF | kubectl apply -f -
apiVersion: ngf-console.f5.com/v1alpha1
kind: ManagedCluster
metadata:
  name: ${CLUSTER_NAME}
  namespace: ngf-system
spec:
  displayName: "${DISPLAY_NAME}"
  region: ${REGION}
  environment: ${ENV}
  kubeconfigSecretRef:
    name: ${CLUSTER_NAME}-kubeconfig
    namespace: ngf-system
EOF

echo "Registered cluster: ${CLUSTER_NAME}"
```

---

## 26. Error Handling & Partial Failure Patterns

### 26.1 Cross-Cluster Query Failures

When querying all clusters in parallel (e.g., `/api/v1/global/inferencestacks`), some clusters may be unreachable. The API server MUST NOT fail the entire request — it returns a partial response with degraded cluster info.

```go
// api/internal/handlers/global.go — ADD response wrapper

type GlobalResponse struct {
    // Resources from healthy clusters
    Items []interface{} `json:"items"`

    // Clusters that were successfully queried
    QueriedClusters []string `json:"queriedClusters"`

    // Clusters that failed (with error details)
    FailedClusters []ClusterError `json:"failedClusters,omitempty"`

    // Whether this is a complete result
    Complete bool `json:"complete"`
}

type ClusterError struct {
    ClusterName string `json:"clusterName"`
    Error       string `json:"error"`
    LastKnownGood *time.Time `json:"lastKnownGood,omitempty"`
}
```

The frontend displays a warning banner when `failedClusters` is non-empty:

```
⚠️ Results incomplete: gpu-staging is unreachable (last seen: 5 min ago)
```

### 26.2 CRD Write Failures

When creating a CRD in a remote cluster fails:

```go
// API handler pattern for write operations

func (h *Handler) CreateInferenceStack(w http.ResponseWriter, r *http.Request) {
    cc := middleware.GetClusterClient(r.Context())

    // 1. Pre-flight: check cluster health
    if !cc.Healthy {
        // Return 503 with cluster status details
        respondError(w, 503, "target cluster %s is unhealthy: %s", cc.Name, cc.LastError)
        return
    }

    // 2. Attempt CRD creation with timeout
    ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
    defer cancel()

    err := cc.Client.Create(ctx, stack)
    if err != nil {
        // 3. Distinguish between network errors and validation errors
        if isNetworkError(err) {
            // Mark cluster as potentially unhealthy, trigger health check
            cc.Healthy = false
            respondError(w, 503, "cluster %s unreachable: %v", cc.Name, err)
        } else {
            // Validation or conflict error — pass through
            respondError(w, 400, "creation failed: %v", err)
        }
        return
    }

    // 4. Record in audit log with cluster context
    h.auditLog.Record(ctx, AuditEntry{
        ClusterName: cc.Name,
        // ...
    })
}
```

### 26.3 Telemetry Gap Handling

When an agent's OTel Collector cannot reach the hub:
- The agent OTel Collector has a **sending queue** configured with file-backed persistence (survives restarts)
- Queue holds up to 10,000 items or 50MB (configurable)
- When connectivity resumes, queued telemetry is forwarded automatically
- The hub ClickHouse may have time gaps — the UI should detect and show "data gap" indicators

```yaml
# agent/config/otel-collector-config.yaml — ADD to exporter config

exporters:
  otlp/hub:
    endpoint: "${HUB_OTEL_ENDPOINT}"
    sending_queue:
      enabled: true
      num_consumers: 10
      queue_size: 10000
      storage: file_storage/otlp_queue
    retry_on_failure:
      enabled: true
      initial_interval: 5s
      max_interval: 60s
      max_elapsed_time: 300s

extensions:
  file_storage/otlp_queue:
    directory: /var/lib/otel/queue
    timeout: 10s
```

---

## 27. DistributedCloudPublish CRD Go Type Changes

The existing DistributedCloudPublish CRD needs a new field for multi-cluster origins.

```go
// operator/api/v1alpha1/distributedcloudpublish_types.go — MODIFY

type DistributedCloudPublishSpec struct {
    // EXISTING: single-cluster origin (httpRouteRef)
    HTTPRouteRef *ObjectReference `json:"httpRouteRef,omitempty"`

    // NEW: multi-cluster origin (for cross-cluster XC publishing)
    // Mutually exclusive with httpRouteRef — use one or the other
    MultiClusterOrigin *MultiClusterOriginSpec `json:"multiClusterOrigin,omitempty"`

    // EXISTING: all distributedCloud fields remain unchanged
    DistributedCloud DistributedCloudConfig `json:"distributedCloud"`
}

// NEW types
type MultiClusterOriginSpec struct {
    // References to inference endpoints in multiple clusters
    ClusterRefs []ClusterEndpointRef `json:"clusterRefs"`

    // Health check configuration for the XC origin pool
    HealthCheck *HealthCheckSpec `json:"healthCheck,omitempty"`
}

type ClusterEndpointRef struct {
    // Name of the ManagedCluster (must exist in hub)
    ClusterName string `json:"clusterName"`

    // Reference to InferencePool or HTTPRoute in the remote cluster
    InferencePoolRef *ObjectReference `json:"inferencePoolRef,omitempty"`
    HTTPRouteRef     *ObjectReference `json:"httpRouteRef,omitempty"`

    // Traffic distribution
    Weight   int `json:"weight,omitempty"`   // 0-100 (default: equal distribution)
    Priority int `json:"priority,omitempty"` // failover priority (1 = primary)
}

type HealthCheckSpec struct {
    Path     string `json:"path,omitempty"`     // e.g., "/health"
    Interval string `json:"interval,omitempty"` // e.g., "10s"
}

// EXISTING: DistributedCloudConfig gets one new optional field
type DistributedCloudConfig struct {
    // ... all existing fields ...

    // NEW: load balancing strategy for multi-cluster origins
    LoadBalancing *LoadBalancingSpec `json:"loadBalancing,omitempty"`
}

type LoadBalancingSpec struct {
    // Strategy for distributing traffic across clusters
    // +kubebuilder:validation:Enum=round-robin;geo-proximity;least-connections;weighted
    Strategy string `json:"strategy,omitempty"`
}
```

> **Claude Code:** Multi-cluster DistributedCloudPublish resources are created by the HUB's API server, not by the workload cluster operator. The hub needs a lightweight XC reconciliation loop for multi-cluster publishes. Implement this in `api/internal/xc/multicluster_reconciler.go` — NOT in the operator binary. It runs as a goroutine in the API server that watches DistributedCloudPublish CRDs with `multiClusterOrigin` set, resolves endpoints from each referenced cluster, and calls the XC API. Single-cluster DistributedCloudPublish CRDs (with `httpRouteRef`) continue to be reconciled by the workload cluster's operator as before.

---

## 28. Testing Strategy — Multi-Cluster

### 28.1 Unit Tests

```go
// api/internal/multicluster/client_pool_test.go
// - Test client creation from kubeconfig
// - Test pool sync (add/remove clusters)
// - Test health checker marking clusters healthy/unhealthy
// - Test circuit breaker behavior

// api/internal/middleware/cluster_context_test.go
// - Test cluster extraction from query param, header, URL path
// - Test single-cluster fallback
// - Test missing cluster returns 400

// api/internal/handlers/global_test.go
// - Test cross-cluster aggregation with mock clients
// - Test partial failure (one cluster unreachable)
// - Test GlobalResponse.FailedClusters population
```

### 28.2 Integration Tests (Kind)

```go
// test/integration/multicluster_test.go

func TestMultiClusterCRDLifecycle(t *testing.T) {
    // 1. Register two Kind clusters with the hub
    // 2. Create InferenceStack in cluster-1 via hub API
    // 3. Verify operator in cluster-1 reconciles child resources
    // 4. Create InferenceStack in cluster-2 via hub API
    // 5. Query global endpoint — both stacks returned
    // 6. Delete stack in cluster-1
    // 7. Global endpoint returns only cluster-2 stack
}

func TestClusterUnreachable(t *testing.T) {
    // 1. Register two clusters
    // 2. Stop cluster-2's Kind control plane
    // 3. Verify health checker marks cluster-2 unreachable
    // 4. Global endpoint returns partial result with failedClusters
    // 5. Restart cluster-2
    // 6. Verify health checker marks cluster-2 ready
}

func TestTelemetryForwarding(t *testing.T) {
    // 1. Generate inference log entries in agent cluster
    // 2. Wait for OTel forwarding
    // 3. Query hub ClickHouse with cluster_name filter
    // 4. Verify data arrived with correct cluster_name
}

func TestAgentHeartbeat(t *testing.T) {
    // 1. Install agent in worker cluster
    // 2. Wait for heartbeat
    // 3. Verify ManagedCluster status updated (phase=Ready, agentInstalled=true)
    // 4. Stop agent
    // 5. Wait for heartbeat timeout
    // 6. Verify ManagedCluster status shows stale heartbeat
}
```

### 28.3 E2E Tests (Playwright)

```typescript
// test/e2e/multicluster.spec.ts

test('cluster selector shows all registered clusters', async ({ page }) => {
    // Open cluster selector dropdown
    // Verify all registered clusters are listed with correct status
    // Select a specific cluster
    // Verify URL updates with ?cluster= param
    // Verify list views show only that cluster's resources
});

test('global view shows resources from all clusters', async ({ page }) => {
    // Select "All Clusters"
    // Verify InferenceStack list shows resources from both clusters
    // Verify each row has a cluster badge
    // Verify GPU heatmap shows GPUs from all clusters
});

test('cluster registration wizard', async ({ page }) => {
    // Navigate to /clusters
    // Click "Add Cluster"
    // Fill in cluster details
    // Upload kubeconfig
    // Verify connectivity test
    // Verify cluster appears in list
});
```
