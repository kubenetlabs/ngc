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
                    +--+-----+-----+--+
                       |     |     |
            +----------+     |     +----------+
            |                |                |
            v                v                v
    +-------+------+  +-----+------+  +------+-------+
    |  Kubernetes   |  | Metrics    |  |  WebSocket   |
    |  (client-go)  |  | Provider   |  |  Hub         |
    +-------+------+  +-----+------+  +------+-------+
            |                |                |
            v                v                v
    +-------+------+  +-----+------+  +------+-------+
    | K8s Clusters  |  | Mock /     |  | Topic        |
    | (Gateway API) |  | ClickHouse |  | Generators   |
    +---------------+  +------------+  +--------------+
```

## Components

### Frontend (`frontend/`)

React 18 single-page application built with Vite.

- **Routing**: React Router v7 with lazy-loaded pages
- **Data fetching**: TanStack Query with polling (10s for lists, 30s stale time)
- **State**: Zustand stores for cluster selection and settings
- **Charts**: Recharts 3.7 for time-series, histograms, and area charts
- **Real-time**: Native WebSocket for EPP decision streaming
- **Styling**: Tailwind CSS 4 with shadcn/ui-inspired components

### API Server (`api/`)

Go HTTP server using Chi router.

- **Handlers**: One handler struct per domain (GatewayHandler, InferenceHandler, InferenceMetricsHandler)
- **Middleware**: CORS, request logging, cluster resolution (injects K8s client into context)
- **WebSocket**: Hub-based pub/sub with per-topic subscriptions
- **Providers**: Interface-based data access (`MetricsProvider` for inference, `controller-runtime` client for K8s)

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

### WebSocket Hub (`api/internal/server/ws_hub.go`)

Topic-based pub/sub system for real-time streaming.

- Clients subscribe to a topic when they connect via `/ws/inference/{topic}`
- Generators produce data on configurable intervals and broadcast to subscribed clients
- Client management with goroutine-per-connection (writePump/readPump)
- Slow clients get messages dropped (non-blocking send)

## API route structure

```
/api/v1/
  clusters                         # List all clusters
  clusters/{cluster}/              # Cluster-scoped routes
    gateways/                      # Gateway CRUD
    httproutes/                    # HTTPRoute CRUD
    inference/
      pools/                       # InferencePool CRUD
      metrics/
        summary                    # Aggregated metrics
        pods?pool=X                # Per-pod GPU metrics
        cost?pool=X                # Cost estimation
        epp-decisions?pool=X       # EPP routing decisions
        ttft-histogram/{pool}      # TTFT distribution
        tps-throughput/{pool}      # Tokens/sec timeseries
        queue-depth/{pool}         # Queue depth timeseries
        gpu-util/{pool}            # GPU utilization timeseries
        kv-cache/{pool}            # KV-cache utilization timeseries
  ws/inference/epp-decisions       # WebSocket: live EPP decisions
  ws/inference/gpu-metrics         # WebSocket: live GPU metrics
  ws/inference/scaling-events      # WebSocket: scaling events
```

## Database schema (ClickHouse)

Four MergeTree tables populated by OTel Collector:

- `ngf_inference_pools` -- Pool metadata (name, model, GPU type, replicas, status)
- `ngf_epp_decisions` -- EPP routing decisions with latency and strategy
- `ngf_pod_metrics` -- Per-pod GPU utilization, memory, temperature, queue depth
- `ngf_inference_metrics_1m` -- 1-minute aggregated metrics (TTFT, TPS, GPU util, KV-cache)

See `deploy/docker-compose/clickhouse/seed.sql` for the complete schema and demo data.
