# Architecture

## System overview

NGF Console uses a hub-spoke architecture for multi-cluster management. The hub cluster runs the API server, frontend, operator, and data stores. Workload clusters run lightweight agents that report health and forward telemetry.

```
                          +-----------------+
                          |    Browser      |
                          |  React + Vite   |
                          +-------+---------+
                                  |
                                  v
                 +-------------------------------+
                 |        Hub Cluster            |
                 |                               |
                 |  +-------------------------+  |
                 |  |   Go API Server (Chi)   |  |
                 |  |   + WebSocket Hub       |  |
                 |  +--+--+--+--+--+--+------+  |
                 |     |  |  |  |  |  |         |
                 |     v  v  |  v  v  v         |
                 |  K8s Prom |  DB  CH OTel     |
                 |           |                  |
                 |  +--------+----------+       |
                 |  | ClientPool        |       |
                 |  |  per-cluster K8s  |       |
                 |  |  circuit breakers |       |
                 |  |  health checker   |       |
                 |  +-------------------+       |
                 |                               |
                 |  ManagedCluster CRDs          |
                 |  Operator (InferenceStack,    |
                 |           GatewayBundle)      |
                 +------+-------------+---------+
                        |             |
              +---------+             +---------+
              v                                 v
   +---------------------+          +---------------------+
   |  Workload Cluster A |          |  Workload Cluster B |
   |                     |          |                     |
   |  Agent:             |          |  Agent:             |
   |   Operator          |          |   Operator          |
   |   Heartbeat (30s)   |          |   Heartbeat (30s)   |
   |   OTel Forwarder    |          |   OTel Forwarder    |
   |                     |          |                     |
   |  NGF + Workloads    |          |  NGF + Workloads    |
   +---------------------+          +---------------------+
```

### Single-cluster mode

When `--multicluster` is not set, the API server operates in single-cluster mode. It wraps one Kubernetes client as the "default" cluster. All API routes work identically -- the multi-cluster layer is transparent.

### File-based multi-cluster mode

When `--clusters-config` is set, the API server loads cluster definitions from a YAML file. Each cluster gets its own `client-go` client. Useful for local development with multiple kubeconfig files.

### CRD-based multi-cluster mode

When `--multicluster` is set, the API server reads `ManagedCluster` CRDs from the hub cluster's namespace. Each ManagedCluster references a kubeconfig Secret. The `ClientPool` builds and maintains Kubernetes clients for each registered cluster.

## Components

### Frontend (`frontend/`)

React 18 single-page application built with Vite.

- **Routing**: React Router v7 with lazy-loaded pages (26 pages)
- **Data fetching**: TanStack Query with polling (10s for lists, 30s stale time)
- **State**: Zustand stores for cluster selection (`clusterStore`) and settings
- **Cluster selector**: Supports individual cluster selection and "All Clusters" mode (`__all__`)
- **Global views**: When "All Clusters" is selected, list views call `/api/v1/global/*` endpoints and display a cluster badge column
- **Charts**: Recharts for time-series, histograms, and area charts
- **Real-time**: Custom `useWebSocket` hook with exponential backoff reconnection (1s base, 30s cap, jitter)
- **Styling**: Tailwind CSS 4 with shadcn/ui-inspired components
- **Forms**: React Hook Form + Zod validation (gateway wizard, pool wizard, policy builder)

### API Server (`api/`)

Go HTTP server using Chi router. Acts as a BFF (Backend-for-Frontend) between the React UI and Kubernetes/data stores.

- **Handlers**: 30+ handler files organized by domain (gateways, routes, inference, policies, certificates, alerts, audit, migration, diagnostics, XC, coexistence, logs, metrics, topology, clusters, global)
- **Middleware**: CORS (configurable via `CORS_ALLOWED_ORIGINS`), request logging, body size limiting (1MB), cluster resolution, panic recovery
- **WebSocket**: Hub-based pub/sub with per-topic subscriptions and slow-client drop
- **Providers**: Interface-based data access (`MetricsProvider` for inference, Prometheus client for RED metrics)
- **Config DB**: SQLite (dev) / PostgreSQL (prod) for alert rules, audit log, and saved views
- **Alerting**: Background evaluation engine with webhook notifications
- **Cluster management**: CRUD for ManagedCluster CRDs, heartbeat receiver, connectivity testing, agent install command generation
- **Global aggregation**: Cross-cluster fan-out for gateways, routes, and GPU capacity with per-cluster timeouts (10s)

### Operator (`operator/`)

Kubernetes operator built with controller-runtime. Watches CRDs and reconciles child resources with drift detection. Runs on both the hub and workload clusters.

- **InferenceStackReconciler**: Reconciles InferencePool, EPP ConfigMap, KEDA ScaledObject, HTTPRoute, DCGM DaemonSet
- **GatewayBundleReconciler**: Reconciles Gateway, NginxProxy, WAF, SnippetsFilter, TLS Secrets
- **Drift detection**: SHA-256 spec hashing with 60-second requeue interval
- **Self-healing**: Owns child resources via OwnerReference; recreates deleted children
- **Status aggregation**: Computes phase (Ready/Pending/Degraded/Error) from child statuses

### Multi-Cluster System (`api/internal/multicluster/`)

CRD-based multi-cluster management with the following sub-components:

#### ClientPool (`client_pool.go`)

Thread-safe pool of Kubernetes clients built from ManagedCluster CRDs.

- `NewClientPool(hubDynamic, namespace)`: Creates a pool that reads ManagedCluster CRDs from the specified namespace
- `Sync(ctx)`: Lists ManagedCluster CRDs, creates/updates/removes `ClusterClient` entries. Reads kubeconfig from referenced Secrets
- `Get(name)`: Returns a `ClusterClient` by name
- `List()`: Returns all `ClusterClient` instances
- `Names()`: Returns sorted list of cluster names

Each `ClusterClient` holds:
- Name, DisplayName, Region, Environment metadata
- `*kubernetes.Client` (K8s API client)
- Healthy flag (updated by health checker)
- K8sVersion, NGFVersion (updated by heartbeat)
- ResourceCounts, GPUCapacity (updated by heartbeat)
- `sync.RWMutex` for thread-safe field updates

#### Health Checker (`health_checker.go`)

Background goroutine that checks cluster health at a configurable interval (default 30s).

- Runs health checks in parallel with bounded concurrency (max 10 simultaneous)
- Each check calls `Discovery().ServerVersion()` with a 5-second timeout
- Updates `ClusterClient.Healthy` via thread-safe `SetHealthy()` method
- Skips clusters whose circuit breaker is open

#### Circuit Breaker (`circuit_breaker.go`)

Per-cluster circuit breaker to prevent slow cascading failures.

- **Closed** (normal): Requests pass through. Failures are counted.
- **Open** (tripped): After 3 consecutive failures, requests fail fast for 30 seconds.
- **Half-Open** (probing): After the reset timeout, one probe request is allowed. Success resets to Closed; failure reopens.

#### PoolAdapter (`adapter.go`)

Wraps `ClientPool` and implements the `cluster.Provider` interface so both file-based and CRD-based modes can be used interchangeably.

- Includes an edition detection cache (`sync.Map` with 5-minute TTL) to avoid repeated CRD list calls
- Provides `Get(name)`, `Default()`, `DefaultName()`, `List(ctx)`, `Names()` methods

### Cluster Manager (`api/internal/cluster/`)

File-based multi-cluster management (legacy mode).

- **Single-cluster mode**: Wraps one K8s client as "default"
- **Multi-cluster mode**: Loads from YAML config, each entry gets its own client
- **ClusterResolver middleware**: Extracts `{cluster}` from URL or falls back to default, injects client into request context

### MetricsProvider (`api/internal/inference/`)

Interface with two implementations:

| Implementation | Use case | Data source |
|---------------|----------|-------------|
| `MockProvider` | Development, demos | In-memory synthetic data with time-varying noise |
| `ClickHouse Provider` | Production | Queries against `ngf_inference_*` MergeTree tables |

The interface has 11 methods covering pools, metrics summaries, pod metrics, EPP decisions, histograms, time-series, and cost estimates.

### Prometheus Client (`api/internal/prometheus/`)

Wraps the Prometheus HTTP API for RED (Rate, Errors, Duration) metrics.

- `Summary()` -- Aggregated metrics: request rate, error rate, latency percentiles, active connections
- `ByRoute()` -- Per-HTTPRoute metrics
- `ByGateway()` -- Per-Gateway metrics

Queries use `nginx_gateway_fabric_*` metric names from NGF's Prometheus exporter.

### Config Database (`api/internal/database/`)

Stores application configuration that doesn't belong in Kubernetes.

- **Alert rules**: Configurable thresholds, operators, severity, duration
- **Audit log**: Records all CRD mutations with before/after diffs
- **Saved views**: User-defined dashboard configurations

Implementations: SQLite (dev, single-node) and PostgreSQL (production).

### Alert Evaluator (`api/internal/alerting/`)

Background goroutine that periodically evaluates alert rules.

- Reads rules from the config database
- Evaluates conditions against current metrics
- Sends webhook notifications when thresholds are breached
- Tracks firing state to avoid duplicate notifications

### WebSocket Hub (`api/internal/server/ws_hub.go`)

Topic-based pub/sub system for real-time streaming.

- Clients subscribe to a topic when they connect via `/api/v1/ws/inference/{topic}`
- Generators produce data on configurable intervals and broadcast to subscribed clients
- Client management with goroutine-per-connection (writePump/readPump)
- Thread-safe random number generation for mock data
- Slow clients get messages dropped (non-blocking send)

### Agent Components (workload clusters)

Installed via the `charts/ngf-console-agent` Helm chart on each workload cluster:

| Component | Description |
|-----------|-------------|
| **Operator** | Same operator binary as the hub. Reconciles InferenceStack and GatewayBundle CRDs on the workload cluster |
| **Heartbeat Reporter** | Sends cluster health (K8s version, NGF version, resource counts, GPU capacity) to hub every 30s via `POST /api/v1/clusters/{name}/heartbeat` |
| **OTel Forwarder** | OpenTelemetry Collector configured to add `cluster_name` resource attribute and forward telemetry to the hub's OTel endpoint |

RBAC is split into two service accounts:
- **Operator SA**: Full read-write access to CRDs and child resources
- **Heartbeat SA**: Read-only access (get/list on nodes, pods, CRDs)

All agent deployments run with hardened security contexts (non-root, read-only rootfs, dropped capabilities, RuntimeDefault seccomp).

## API route structure

```
/api/v1/
  health                              # Health check (liveness/readiness)

  # Cluster management (hub-level, no cluster middleware)
  clusters                            # List / Register clusters
  clusters/summary                    # Global cluster summary
  clusters/{cluster}/
    detail                            # Cluster detail (edition, health, agent info)
    test                              # Test connectivity
    install-agent                     # Generate agent Helm install command
    heartbeat                         # Receive agent heartbeat
    (delete)                          # Unregister cluster

  # Global cross-cluster aggregation
  global/
    gateways                          # All gateways across all clusters
    routes                            # All routes across all clusters
    gpu-capacity                      # Aggregated GPU capacity

  # Cluster-scoped routes (via cluster middleware)
  clusters/{cluster}/
    config                            # Server configuration
    gatewayclasses/                   # GatewayClass list/get
    gateways/                         # Gateway CRUD + deploy
    gatewaybundles/                   # GatewayBundle CRD CRUD + status
    httproutes/                       # HTTPRoute CRUD + simulate
    grpcroutes/                       # (501 Not Implemented)
    tlsroutes/                        # (501 Not Implemented)
    tcproutes/                        # (501 Not Implemented)
    udproutes/                        # (501 Not Implemented)
    policies/{type}/                  # Policy CRUD + conflicts
    certificates/                     # Certificate CRUD + expiring
    metrics/
      summary                         # Prometheus RED summary
      by-route                        # Prometheus per-route metrics
      by-gateway                      # Prometheus per-gateway metrics
    logs/
      query                           # ClickHouse log query
      topn                            # Top-N analytics
    topology/
      full                            # Full resource graph
      by-gateway/{name}               # Gateway-scoped graph
    diagnostics/
      route-check                     # Route diagnostic wizard
      trace                           # Request trace waterfall
    inference/
      pools/                          # InferencePool CRUD + deploy
      epp                             # EPP config get/update
      autoscaling                     # Autoscaling config get/update
      metrics/
        summary                       # Aggregated inference metrics
        by-pool                       # Per-pool metrics
        pods                          # Per-pod GPU metrics
        cost                          # Cost estimation
        epp-decisions                 # EPP routing decisions
        ttft-histogram/{pool}         # TTFT distribution
        tps-throughput/{pool}         # Tokens/sec timeseries
        queue-depth/{pool}            # Queue depth timeseries
        gpu-util/{pool}               # GPU utilization timeseries
        kv-cache/{pool}               # KV-cache utilization timeseries
      diagnostics/
        slow                          # Slow inference analysis
        replay                        # Request replay
        benchmark                     # Model benchmarking
      stacks/                         # InferenceStack CRD CRUD + status
    coexistence/
      overview                        # KIC + NGF side-by-side view
      migration-readiness             # Migration readiness score
    xc/
      status                          # XC connection status
      publish                         # Publish to XC
      publish/{id}                    # Get/delete publish
      metrics                         # XC metrics
    migration/
      import                          # Import NGINX/Ingress config
      analysis                        # Migration analysis
      generate                        # Generate Gateway API resources
      apply                           # Apply to cluster (dry-run or live)
      validate                        # Validate migrated resources
    audit/                            # Audit log list + diff
    alerts/                           # Alert rule CRUD + firing + toggle

  # Legacy routes (backward compat -- uses default cluster)
  config, gateways/, httproutes/, ... # Same routes without cluster prefix

  # WebSocket topics
  ws/inference/epp-decisions          # Live EPP decisions (~1s)
  ws/inference/gpu-metrics            # Live GPU metrics (~2s)
  ws/inference/scaling-events         # Scaling events (~15s)
```

## Data flows

### Multi-cluster registration and health

```
Admin registers cluster via UI or API
        |
        v
API creates ManagedCluster CRD + kubeconfig Secret on hub
        |
        v
ClientPool.Sync() detects new CRD, builds K8s client from kubeconfig
        |
        v
Health checker starts probing cluster (30s interval, bounded concurrency)
        |
        v
Circuit breaker tracks consecutive failures (3 -> open, 30s reset)
        |
        v
Agent heartbeat (30s) updates in-memory state + ManagedCluster CRD status
        |
        v
Frontend polls /clusters for latest health info
```

### Gateway lifecycle (operator-driven)

```
User creates Gateway via UI
        |
        v
API creates GatewayBundle CRD (on target cluster)
        |
        v
Operator detects new CRD
        |
        v
Operator reconciles: creates Gateway, NginxProxy, WAF, etc.
        |
        v
Operator updates GatewayBundle.status (phase, children, conditions)
        |
        v
API reads Gateway + GatewayBundle status
        |
        v
UI displays Gateway with operator-reported status
```

### Inference metrics

```
NVIDIA GPUs -> DCGM Exporter -> OTel Collector -> ClickHouse
vLLM/Triton -> OTel Collector -> ClickHouse
                                    |
                                    v
                          ClickHouseProvider
                                    |
                                    v
                          handlers/inference_metrics.go
                                    |
                                    v
                          Chi Router (/api/v1/inference/metrics/...)
                                    |
                                    v
                          frontend/src/api/inference.ts
                                    |
                                    v
                          React pages (TanStack Query)
```

### Multi-cluster telemetry pipeline

```
Workload Cluster                        Hub Cluster
+-------------------+                  +-------------------+
| NGF + Workloads   |                  |                   |
|       |           |                  |                   |
|       v           |                  |                   |
| OTel Collector    |   OTLP/gRPC     | OTel Collector    |
| (adds cluster_    |  ------------>  | (receives from    |
|  name attribute)  |   port 4317     |  all clusters)    |
+-------------------+                  |       |           |
                                       |       v           |
                                       | ClickHouse        |
                                       | (cluster_name col)|
                                       +-------------------+
```

All ClickHouse tables include a `cluster_name LowCardinality(String)` column. Materialized views use `AggregatingMergeTree` with `-State` combinators for correct aggregation.

### WebSocket streaming

```
ws_topics.go (mock generators, 1-15s intervals)
        |
        v
ws_hub.go (topic-based pub/sub)
        |
        v
/ws/inference/epp-decisions    -> EPPDecisionVisualizer.tsx
/ws/inference/gpu-metrics      -> GPUHeatmap.tsx
/ws/inference/scaling-events   -> ScalingTimeline.tsx
```

### Alert evaluation

```
Config DB (alert rules)
        |
        v
alerting.Evaluator (background goroutine, periodic check)
        |
        v
MetricsProvider / PromClient (current metric values)
        |
        v
Threshold comparison (>, <, >=, <=, ==)
        |
        v
Webhook dispatch (Slack, PagerDuty, etc.)
```

### Global cross-cluster queries

```
Frontend selects "All Clusters"
        |
        v
API client routes to /api/v1/global/* endpoints
        |
        v
GlobalHandler fans out to all clusters in parallel (sync.WaitGroup)
        |
        v
Per-cluster goroutines query with 10s timeout
        |
        v
Results aggregated, wrapped with clusterName/clusterRegion
        |
        v
Frontend displays combined list with cluster badge column
```

## Database schema (ClickHouse)

Tables populated by OTel Collector. All tables include `cluster_name LowCardinality(String)`:

- `ngf_access_logs` -- HTTP access logs with request/response details
- `ngf_inference_logs` -- Inference request logs (model, tokens, latency)
- `ngf_inference_pools` -- Pool metadata (name, model, GPU type, replicas, status)
- `ngf_epp_decisions` -- EPP routing decisions with latency and strategy
- `ngf_pod_metrics` -- Per-pod GPU utilization, memory, temperature, queue depth
- `ngf_metrics_1m` -- 1-minute aggregated RED metrics (AggregatingMergeTree)
- `ngf_inference_metrics_1m` -- 1-minute aggregated inference metrics (AggregatingMergeTree)

See `deploy/docker-compose/clickhouse/init.sql` for the complete schema.

## Database schema (Config DB)

SQLite/PostgreSQL tables for application state:

- `alert_rules` -- Alert rule definitions (name, resource, metric, operator, threshold, severity, cluster_name)
- `audit_log` -- CRD mutation audit trail (action, resource, before/after JSON, timestamp, cluster_name)
- `saved_views` -- User-saved dashboard configurations (cluster_name)

Schema is auto-migrated on startup via `store.Migrate()`.
