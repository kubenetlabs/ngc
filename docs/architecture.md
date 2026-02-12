# Architecture

## System overview

```
                    +------------------+
                    |    Browser       |
                    |  React + Vite    |
                    +--------+---------+
                             |
                             v
                    +------------------+
                    |  Go API Server   |
                    |  (Chi Router)    |
                    +--+--+--+--+--+--+
                       |  |  |  |  |
          +------------+  |  |  |  +-----------+
          |               |  |  |              |
          v               v  |  v              v
  +-------+------+  +----++ |+-+------+  +----+--------+
  |  Kubernetes   |  |Prom| || Config  |  |  WebSocket  |
  |  (client-go)  |  |    | ||  DB     |  |  Hub        |
  +-------+------+  +----++ |+--------+  +-------------+
          |               |  |
          v               v  v
  +-------+------+  +----+--+-----+
  | K8s Clusters  |  | ClickHouse  |
  | (Gateway API) |  | + OTel      |
  +-+-------------+  +------------+
    |
    v
  +-+-----------+
  | Operator    |
  | (CRDs)      |
  +-------------+
```

## Components

### Frontend (`frontend/`)

React 18 single-page application built with Vite.

- **Routing**: React Router v7 with lazy-loaded pages (26 pages)
- **Data fetching**: TanStack Query with polling (10s for lists, 30s stale time)
- **State**: Zustand stores for cluster selection and settings
- **Charts**: Recharts for time-series, histograms, and area charts
- **Real-time**: Custom `useWebSocket` hook for live data streaming
- **Styling**: Tailwind CSS 4 with shadcn/ui-inspired components
- **Forms**: React Hook Form + Zod validation (gateway wizard, pool wizard, policy builder)

### API Server (`api/`)

Go HTTP server using Chi router. Acts as a BFF (Backend-for-Frontend) between the React UI and Kubernetes/data stores.

- **Handlers**: 30+ handler files organized by domain (gateways, routes, inference, policies, certificates, alerts, audit, migration, diagnostics, XC, coexistence, logs, metrics, topology)
- **Middleware**: CORS, request logging, body size limiting (1MB), cluster resolution, panic recovery
- **WebSocket**: Hub-based pub/sub with per-topic subscriptions and slow-client drop
- **Providers**: Interface-based data access (`MetricsProvider` for inference, Prometheus client for RED metrics)
- **Config DB**: SQLite (dev) / PostgreSQL (prod) for alert rules, audit log, and saved views
- **Alerting**: Background evaluation engine with webhook notifications

### Operator (`operator/`)

Kubernetes operator built with controller-runtime. Watches CRDs and reconciles child resources with drift detection.

- **InferenceStackReconciler**: Reconciles InferencePool, EPP ConfigMap, KEDA ScaledObject, HTTPRoute, DCGM DaemonSet
- **GatewayBundleReconciler**: Reconciles Gateway, NginxProxy, WAF, SnippetsFilter, TLS Secrets
- **Drift detection**: SHA-256 spec hashing with 60-second requeue interval
- **Self-healing**: Owns child resources via OwnerReference; recreates deleted children
- **Status aggregation**: Computes phase (Ready/Pending/Degraded/Error) from child statuses

### Controller (`controller/`)

Supplementary controllers for cross-cluster and routing concerns.

- **Route watcher**: Monitors Gateway API route resources for changes
- **XC publish controller**: Manages F5 Distributed Cloud publish lifecycle

### Cluster Manager (`api/internal/cluster/`)

Manages connections to one or more Kubernetes clusters.

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

- `Summary()` — Aggregated metrics: request rate, error rate, latency percentiles, active connections
- `ByRoute()` — Per-HTTPRoute metrics
- `ByGateway()` — Per-Gateway metrics

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

## API route structure

```
/api/v1/
  health                              # Health check (liveness/readiness)
  clusters                            # List all clusters
  clusters/{cluster}/                  # Cluster-scoped routes
    config                             # Server configuration
    gatewayclasses/                    # GatewayClass list/get
    gateways/                          # Gateway CRUD + deploy
    gatewaybundles/                    # GatewayBundle CRD CRUD + status
    httproutes/                        # HTTPRoute CRUD + simulate
    grpcroutes/                        # (501 Not Implemented)
    tlsroutes/                         # (501 Not Implemented)
    tcproutes/                         # (501 Not Implemented)
    udproutes/                         # (501 Not Implemented)
    policies/{type}/                   # Policy CRUD + conflicts
    certificates/                      # Certificate CRUD + expiring
    metrics/
      summary                          # Prometheus RED summary
      by-route                         # Prometheus per-route metrics
      by-gateway                       # Prometheus per-gateway metrics
    logs/
      query                            # ClickHouse log query
      topn                             # Top-N analytics
    topology/
      full                             # Full resource graph
      by-gateway/{name}                # Gateway-scoped graph
    diagnostics/
      route-check                      # Route diagnostic wizard
      trace                            # Request trace waterfall
    inference/
      pools/                           # InferencePool CRUD + deploy
      epp                              # EPP config get/update
      autoscaling                      # Autoscaling config get/update
      metrics/
        summary                        # Aggregated inference metrics
        by-pool                        # Per-pool metrics
        pods                           # Per-pod GPU metrics
        cost                           # Cost estimation
        epp-decisions                  # EPP routing decisions
        ttft-histogram/{pool}          # TTFT distribution
        tps-throughput/{pool}          # Tokens/sec timeseries
        queue-depth/{pool}             # Queue depth timeseries
        gpu-util/{pool}                # GPU utilization timeseries
        kv-cache/{pool}                # KV-cache utilization timeseries
      diagnostics/
        slow                           # Slow inference analysis
        replay                         # Request replay
        benchmark                      # Model benchmarking
      stacks/                          # InferenceStack CRD CRUD + status
    coexistence/
      overview                         # KIC + NGF side-by-side view
      migration-readiness              # Migration readiness score
    xc/
      status                           # XC connection status
      publish                          # Publish to XC
      publish/{id}                     # Get/delete publish
      metrics                          # XC metrics
    migration/
      import                           # Import NGINX/Ingress config
      analysis                         # Migration analysis
      generate                         # Generate Gateway API resources
      apply                            # Apply to cluster (dry-run or live)
      validate                         # Validate migrated resources
    audit/                             # Audit log list + diff
    alerts/                            # Alert rule CRUD + firing + toggle

  # WebSocket topics
  ws/inference/epp-decisions           # Live EPP decisions (~1s)
  ws/inference/gpu-metrics             # Live GPU metrics (~2s)
  ws/inference/scaling-events          # Scaling events (~15s)
```

## Data flows

### Gateway lifecycle (operator-driven)

```
User creates Gateway via UI
        |
        v
API creates GatewayBundle CRD
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
NVIDIA GPUs → DCGM Exporter → OTel Collector → ClickHouse
vLLM/Triton → OTel Collector → ClickHouse
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

### WebSocket streaming

```
ws_topics.go (mock generators, 1-15s intervals)
        |
        v
ws_hub.go (topic-based pub/sub)
        |
        v
/ws/inference/epp-decisions    → EPPDecisionVisualizer.tsx
/ws/inference/gpu-metrics      → GPUHeatmap.tsx
/ws/inference/scaling-events   → ScalingTimeline.tsx
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

## Database schema (ClickHouse)

Four MergeTree tables populated by OTel Collector:

- `ngf_inference_pools` — Pool metadata (name, model, GPU type, replicas, status)
- `ngf_epp_decisions` — EPP routing decisions with latency and strategy
- `ngf_pod_metrics` — Per-pod GPU utilization, memory, temperature, queue depth
- `ngf_inference_metrics_1m` — 1-minute aggregated metrics (TTFT, TPS, GPU util, KV-cache)

See `deploy/docker-compose/clickhouse/seed.sql` for the complete schema and demo data.

## Database schema (Config DB)

SQLite/PostgreSQL tables for application state:

- `alert_rules` — Alert rule definitions (name, resource, metric, operator, threshold, severity)
- `audit_log` — CRD mutation audit trail (action, resource, before/after JSON, timestamp)
- `saved_views` — User-saved dashboard configurations

Schema is auto-migrated on startup via `store.Migrate()`.
