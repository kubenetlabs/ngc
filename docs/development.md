# Development Guide

## Prerequisites

- Go 1.25+
- Node.js 20+
- pnpm 9+ (`npm install -g pnpm`)
- A kubeconfig with access to a Kubernetes cluster (optional -- inference features work without one)
- Helm 3 (for packaging agent chart)

## Getting started

```bash
git clone <repo-url> && cd ngc
```

### Single-cluster mode (default)

**Terminal 1 -- API:**

```bash
cd api
go run ./cmd/server
```

**Terminal 2 -- Frontend:**

```bash
cd frontend
pnpm install
pnpm dev
```

Open http://localhost:5173. The API runs on port 8080 and the Vite dev server proxies API requests automatically.

**Terminal 3 -- Operator (optional, requires cluster):**

```bash
cd operator
go run ./cmd/
```

### Multi-cluster mode

Multi-cluster mode reads `ManagedCluster` CRDs from the hub cluster. You need at least one ManagedCluster resource and its kubeconfig Secret in the hub namespace.

**Terminal 1 -- API (multi-cluster):**

```bash
cd api
go run ./cmd/server \
  --multicluster \
  --multicluster-namespace=ngf-system \
  --multicluster-default=hub
```

**Terminal 2 -- Frontend:**

```bash
cd frontend
pnpm install
pnpm dev
```

**Terminal 3 -- Agent heartbeat (optional, simulates workload cluster):**

```bash
cd agent
go run ./cmd/heartbeat \
  --cluster-name=workload-west \
  --hub-api=http://localhost:8080
```

## Architecture overview

```
Browser (React)
    |
    v
Vite Dev Server (:5173) --proxy--> Go API Server (:8080)
                                        |
                          +-------------+-------------+-----------+
                          |             |             |           |
                    Kubernetes    MockProvider    Config DB   ClientPool
                    (Gateways)    (Inference)     (SQLite)   (Multi-cluster)
                          |                                      |
                    Operator (CRDs)                   +----------+---------+
                                                      |          |         |
                                                  Cluster A  Cluster B  Cluster N
                                                  (K8s API)  (K8s API)  (K8s API)
```

The API server has several data paths:

1. **Kubernetes** -- Gateway/HTTPRoute CRUD uses `client-go` talking to a real cluster
2. **MetricsProvider** -- Inference metrics use an interface with two implementations:
   - `MockProvider` (default) -- synthetic data, no external dependencies
   - `ClickHouse Provider` -- queries real data from ClickHouse tables
3. **Prometheus** -- RED metrics (Rate, Errors, Duration) from NGF's Prometheus exporter
4. **Config DB** -- SQLite for alert rules, audit log, and saved views
5. **Operator** -- CRD-based reconciliation for InferenceStack and GatewayBundle
6. **ClientPool** (multi-cluster) -- Thread-safe pool of K8s clients built from ManagedCluster CRDs, with per-cluster circuit breakers and health checks

## Project structure

```
ngc/
├── api/                           # Go API server
│   ├── cmd/server/                # main.go entry point
│   ├── internal/
│   │   ├── handlers/              # HTTP handlers (gateways, routes, clusters, global, ...)
│   │   ├── server/                # Chi router, middleware, WebSocket hub
│   │   ├── cluster/               # Single-cluster and file-based cluster providers
│   │   ├── multicluster/          # CRD-based multi-cluster: ClientPool, HealthChecker, CircuitBreaker, PoolAdapter
│   │   ├── inference/             # MetricsProvider interface and MockProvider
│   │   ├── clickhouse/            # ClickHouse MetricsProvider implementation
│   │   ├── prometheus/            # Prometheus client for RED metrics
│   │   ├── database/              # SQLite config database (alerts, audit, saved views)
│   │   ├── kubernetes/            # K8s client wrapper
│   │   └── alerting/              # Alert rule evaluator and webhook dispatcher
│   └── pkg/version/               # Build-time version info
├── agent/                         # Workload cluster agent
│   ├── cmd/heartbeat/             # Heartbeat reporter binary
│   ├── Dockerfile.heartbeat
│   ├── go.mod
│   └── go.sum
├── operator/                      # Controller-runtime operator
│   ├── cmd/                       # Operator entry point
│   ├── api/v1alpha1/              # CRD types (InferenceStack, GatewayBundle, DistributedCloudPublish)
│   ├── internal/controller/       # Reconcilers
│   └── config/crd/bases/          # Generated CRD YAML
├── migration-cli/                 # NGINX config migration tool
├── frontend/                      # React + TypeScript + Vite
│   ├── src/
│   │   ├── api/                   # API client functions (clusters.ts, global.ts, inference.ts, ...)
│   │   ├── components/
│   │   │   ├── clusters/          # ClusterHealthCard, ClusterBadge
│   │   │   ├── layout/            # MainLayout, Sidebar, ClusterSelector
│   │   │   └── ...                # Domain-specific components
│   │   ├── hooks/                 # useWebSocket, useActiveCluster, ...
│   │   ├── pages/                 # ClusterManagement, ClusterDetail, ClusterRegister, ...
│   │   ├── store/                 # Zustand stores (clusterStore.ts)
│   │   ├── types/                 # TypeScript types (cluster.ts, inference.ts, ...)
│   │   └── routes.tsx             # React Router route definitions
│   └── ...
├── charts/
│   └── ngf-console-agent/         # Agent Helm chart for workload clusters
│       ├── Chart.yaml
│       ├── values.yaml
│       ├── crds/                  # InferenceStack, GatewayBundle, DistributedCloudPublish CRDs
│       └── templates/             # Operator, heartbeat, OTel Collector deployments + RBAC
├── deploy/
│   ├── docker-compose/            # Local development stack
│   │   ├── docker-compose.yaml
│   │   └── clickhouse/            # ClickHouse schema and migrations
│   └── helm/ngf-console/          # Hub Helm chart
│       ├── crds/                  # ManagedCluster CRD + operator CRDs
│       ├── templates/
│       └── values.yaml
└── docs/                          # Documentation
```

## Multi-cluster package (`api/internal/multicluster/`)

This package implements the CRD-based hub-spoke architecture. All files are in `api/internal/multicluster/`.

### types.go

Defines the `ManagedCluster` CRD Go types: `ManagedClusterSpec`, `ManagedClusterStatus`, `SecretReference`, `ResourceCounts`, `GPUCapacitySummary`. Registered with `GroupVersion: ngf-console.f5.com/v1alpha1`.

### scheme.go

Registers the ManagedCluster types with `runtime.Scheme` so client-go can serialize/deserialize them. Exports `SchemeBuilder` and `AddToScheme`.

### client_pool.go

`ClientPool` manages a thread-safe map of `ClusterClient` structs (one per registered cluster). Key methods:

- `NewClientPool(dynamicClient, namespace)` -- constructor, takes the hub's dynamic client and the namespace where ManagedCluster CRDs live
- `Sync(ctx)` -- lists ManagedCluster CRDs, reads kubeconfig Secrets, builds K8s clients. Call on startup and periodically
- `Get(name)` -- returns a `ClusterClient` by name (errors if unhealthy or not found)
- `List()` -- returns all ClusterClients
- `Names()` -- returns registered cluster names

Each `ClusterClient` holds: `DynamicClient`, `RestConfig`, `Healthy` flag, cluster metadata (region, environment, edition), and a `sync.RWMutex` for thread-safe health updates.

### health_checker.go

`HealthChecker` runs a background goroutine that pings each cluster every 30s via `Discovery().ServerVersion()` with a 5s timeout. Bounded concurrency (max 10 simultaneous checks). Updates both the `ClusterClient.Healthy` flag and the `ManagedCluster.Status` on the hub.

### circuit_breaker.go

Per-cluster circuit breaker with three states: **Closed** (normal), **Open** (failing, fast-reject), **Half-Open** (probe one request). Configurable thresholds: 3 consecutive failures open the breaker, 30s reset timeout. Called by `ClientPool.Get()` before returning a client.

### adapter.go

`PoolAdapter` wraps `ClientPool` and implements the `cluster.Provider` interface that all existing handlers use. This allows the API server to swap between file-based and CRD-based cluster management without changing handler code. Includes an edition cache (`sync.Map` with 5-minute TTL) to avoid repeated CRD reads.

### Tests

- `client_pool_test.go` -- tests Sync with mock dynamic client, Get for missing/unhealthy clusters
- `circuit_breaker_test.go` -- tests state transitions (closed -> open -> half-open -> closed)

## Data flow: Multi-cluster request routing

```
Frontend request (e.g., GET /api/v1/gateways?cluster=workload-west)
        |
        v
cluster_middleware.go: resolve cluster from URL / query / X-Cluster header
        |
        v
PoolAdapter.Get("workload-west")
        |
        v
ClientPool.Get("workload-west") → checks circuit breaker → returns ClusterClient
        |
        v
handlers/gateways.go: uses ClusterClient.DynamicClient to list Gateways on workload-west
        |
        v
JSON response
```

When no cluster is specified, the middleware uses the default cluster (set via `--multicluster-default`).

## Data flow: Global aggregation

```
Frontend request (e.g., GET /api/v1/global/gateways)
        |
        v
handlers/global.go: globalQueryTimeout = 10s
        |
        v
For each healthy cluster in ClientPool (parallel with errgroup):
    ClusterClient.DynamicClient → list Gateways
        |
        v
Wrap each result with { clusterName, clusterRegion, gateway }
        |
        v
Merge results, return JSON array
```

Per-cluster errors are included inline with `error` field rather than failing the entire request.

## Data flow: Inference metrics

```
MockProvider / ClickHouse
        |
        v
MetricsProvider interface
        |
        v
handlers/inference_metrics.go  -- Summary, PodMetrics, EPPDecisions, timeseries, histograms
        |
        v
Chi Router (/api/v1/inference/...)
        |
        v
frontend/src/api/inference.ts  -- Axios fetch functions
        |
        v
React pages (TanStack Query)   -- InferencePoolList, InferencePoolDetail, InferenceOverview
```

In multi-cluster mode, ClickHouse queries include `AND cluster_name = ?` to scope results to the active cluster.

## Data flow: WebSocket streaming

```
ws_topics.go (mock generators, 1-15s intervals, thread-safe RNG)
        |
        v
ws_hub.go (topic-based pub/sub, slow-client drop)
        |
        v
/ws/inference/epp-decisions    -- EPP routing decisions (1s)
/ws/inference/gpu-metrics      -- Pod GPU metrics (2s)
/ws/inference/scaling-events   -- Autoscaling events (15s)
        |
        v
useWebSocket.ts hook           -- Auto-reconnect with exponential backoff (1s base, 30s cap)
        |
        v
EPPDecisionVisualizer.tsx      -- Live pod cards with routing animation
```

WebSocket connections accept a `?cluster=` query parameter to scope data to a specific cluster.

## Data flow: Agent heartbeat

```
agent/cmd/heartbeat (workload cluster, every 30s)
        |
        v
POST /api/v1/clusters/{name}/heartbeat
    Body: { kubernetesVersion, ngfVersion, resourceCounts, gpuCapacity }
        |
        v
handlers/clusters.go: validate cluster name, parse body (64KB limit)
        |
        v
Update ManagedCluster CRD status on hub via dynamic client
    status.phase = Ready
    status.lastHeartbeat = now
    status.kubernetesVersion, ngfVersion, resourceCounts, gpuCapacity
```

## Data flow: Operator reconciliation

```
User creates InferenceStack via API
        |
        v
API writes InferenceStack CRD to K8s
        |
        v
InferenceStackReconciler detects new/changed CRD
        |
        v
Reconcile children:
  1. InferencePool (unstructured)
  2. EPP ConfigMap
  3. KEDA ScaledObject (stub)
  4. HTTPRoute (stub)
  5. DCGM DaemonSet (stub)
        |
        v
Compute phase from child statuses
        |
        v
Update InferenceStack.status (phase, children, conditions, observedSpecHash)
        |
        v
60s requeue for drift detection
```

The same operator binary runs on both hub and workload clusters. On workload clusters, it is deployed via the agent Helm chart.

## Adding a new API handler

1. **Create the handler** in `api/internal/handlers/{domain}.go`
2. **Define types** -- request/response structs with JSON tags
3. **Register routes** in `api/internal/server/server.go` in `mountResourceRoutes()`
4. **Add frontend API client** in `frontend/src/api/{domain}.ts`
5. **Add TypeScript types** in `frontend/src/types/{domain}.ts`
6. **Create page component** in `frontend/src/pages/{Page}.tsx`
7. **Register route** in `frontend/src/routes.tsx`

If the handler should work in multi-cluster mode, it will automatically receive the resolved cluster client from `cluster_middleware.go`. No additional code is needed -- the middleware handles cluster resolution.

## Adding a global aggregation endpoint

1. **Add the handler** to `api/internal/handlers/global.go`
2. Use `pool.List()` to get all healthy clusters, fan out with `errgroup`
3. Set a per-cluster timeout (convention: 10s via `globalQueryTimeout`)
4. Wrap each result with cluster metadata (`clusterName`, `clusterRegion`)
5. **Register the route** in `api/internal/server/server.go` under the `/global/` group
6. **Add frontend fetch function** in `frontend/src/api/clusters.ts` or a domain-specific client
7. **Use in a component** -- check `activeCluster === '__all__'` in the cluster store

## Adding a new inference metric

1. **Add the type** to `api/internal/inference/types.go`
2. **Add the method** to the `MetricsProvider` interface in `api/internal/inference/provider.go`
3. **Implement mock data** in `api/internal/inference/mock_provider.go`
4. **Implement ClickHouse query** in `api/internal/clickhouse/provider.go` (include `cluster_name` in the WHERE clause)
5. **Add the handler method** to `api/internal/handlers/inference_metrics.go`
6. **Register the route** in `api/internal/server/server.go` under `/inference/metrics/`
7. **Add the frontend fetch function** to `frontend/src/api/inference.ts`
8. **Add the TypeScript type** to `frontend/src/types/inference.ts`
9. **Use it in a component** (page or chart)

## Adding a new WebSocket topic

1. Add a generator in `api/internal/server/ws_topics.go` using `hub.AddGenerator()`
2. Register the WS route in `api/internal/server/server.go`:
   ```go
   r.Get("/ws/inference/my-topic", s.Hub.ServeWS("my-topic"))
   ```
3. Connect from the frontend using the `useWebSocket` hook in `frontend/src/hooks/useWebSocket.ts`

WebSocket topics support the `?cluster=` query parameter automatically.

## Adding a new CRD

1. **Define types** in `operator/api/v1alpha1/{name}_types.go` with kubebuilder markers
2. **Generate deepcopy**: `make generate-deepcopy`
3. **Generate CRD YAML**: `make generate-crds`
4. **Write controller** in `operator/internal/controller/{name}_controller.go`
5. **Write children builders** in `operator/internal/controller/{name}_children.go`
6. **Register controller** in `operator/cmd/main.go`
7. **Write tests** in `operator/internal/controller/{name}_controller_test.go`
8. **Add API handler** in `api/internal/handlers/{name}s.go`
9. **Install CRD**: `kubectl apply -f operator/config/crd/bases/`
10. **Copy CRD to agent chart** if the CRD needs to exist on workload clusters: `cp operator/config/crd/bases/{name}.yaml charts/ngf-console-agent/crds/`

## Developing the agent

The agent runs on workload clusters and consists of three components deployed by the agent Helm chart (`charts/ngf-console-agent/`).

### Heartbeat reporter

The heartbeat binary lives in `agent/cmd/heartbeat/`. It discovers cluster state and POSTs it to the hub every 30s.

```bash
# Build
cd agent && go build -o bin/heartbeat ./cmd/heartbeat

# Run locally (points at a local hub API)
./bin/heartbeat \
  --cluster-name=dev-cluster \
  --hub-api=http://localhost:8080 \
  --interval=10s

# Docker
docker build -t agent-heartbeat:dev -f agent/Dockerfile.heartbeat agent/
```

Key flags / env vars:

| Flag | Env | Default | Description |
|------|-----|---------|-------------|
| `--cluster-name` | `CLUSTER_NAME` | (required) | Name of this cluster (must match ManagedCluster CRD) |
| `--hub-api` | `HUB_API_ENDPOINT` | (required) | Hub API endpoint URL |
| `--auth-token` | `HUB_AUTH_TOKEN` | `""` | Auth token for hub API |
| `--interval` | -- | `30s` | Heartbeat interval |
| `--kubeconfig` | -- | in-cluster | Path to kubeconfig |

### Operator

The operator is the same binary used on the hub. On workload clusters, it reconciles InferenceStack and GatewayBundle CRDs locally. The agent chart deploys it with the same image and RBAC as the hub chart.

### OTel Collector

The agent OTel Collector receives telemetry from local workloads and forwards it to the hub's OTel endpoint. It adds a `cluster_name` resource attribute so the hub can distinguish data by cluster. Configuration is in the agent chart's `configmap-otel.yaml` template.

### Agent Helm chart development

```bash
# Render templates locally
helm template ngf-console-agent charts/ngf-console-agent \
  --set cluster.name=dev \
  --set hub.apiEndpoint=http://localhost:8080 \
  --set hub.otelEndpoint=localhost:4317

# Package chart
make helm-package-agent

# Install on a cluster
helm install ngf-console-agent charts/ngf-console-agent \
  --namespace ngf-system --create-namespace \
  --set cluster.name=workload-west \
  --set hub.apiEndpoint=https://hub.example.com \
  --set hub.otelEndpoint=hub.example.com:4317
```

## Running tests

```bash
# All Go tests
cd api && go test ./...
cd operator && go test ./...
cd agent && go test ./...

# Multi-cluster package specifically
cd api && go test ./internal/multicluster/...

# Specific package
cd api && go test ./internal/handlers/...
cd api && go test ./internal/server/...
cd operator && go test ./internal/controller/...

# Frontend
cd frontend && pnpm test
cd frontend && pnpm lint

# Everything
make test
make lint
```

## Build commands

```bash
make build            # Build all components (includes agent heartbeat)
make test             # Run all tests
make lint             # Run all linters
make docker-build     # Build all Docker images (includes agent)
make helm-template    # Render Helm templates
make helm-package-agent # Package agent Helm chart
make generate-crds    # Regenerate CRD YAML from Go types
make generate-deepcopy # Regenerate deepcopy methods
make clean            # Remove build artifacts and stop containers
```

Individual builds:

```bash
cd api && go build -o bin/api-server ./cmd/server
cd operator && go build -o bin/operator ./cmd/
cd agent && go build -o bin/heartbeat ./cmd/heartbeat
cd migration-cli && go build -o bin/ngf-migrate .
cd frontend && pnpm build
```

Docker images:

```bash
make docker-build-api                # API server image
make docker-build-operator           # Operator image
make docker-build-agent-heartbeat    # Agent heartbeat image
make docker-build-agent              # All agent images
make docker-build-frontend           # Frontend image
```

## Key conventions

- **Go handlers**: `func (h *Handler) Method(w http.ResponseWriter, r *http.Request)`
- **Go errors**: `fmt.Errorf("context: %w", err)`
- **Go imports**: stdlib, then external, then internal
- **Go logging**: `slog` structured logging (no `log` or `fmt.Println`)
- **Go concurrency**: use `sync.RWMutex` for shared state (see `ClusterClient`), `errgroup` for parallel fan-out
- **React**: functional components, TanStack Query for data fetching, Zustand for state
- **Types**: Go response types in handler files, TypeScript types in `src/types/`
- **Component files**: PascalCase (e.g., `GPUHeatmap.tsx`)
- **API client files**: camelCase (e.g., `inference.ts`)
- **CRD naming**: `{name}-{child}` (e.g., `llama-70b-pool`, `llama-70b-epp-config`)
- **Enterprise features**: gated by edition detection, greyed out in OSS mode
- **Stubs**: unimplemented endpoints return 501 Not Implemented with JSON `{"error": "not implemented"}`
- **Cluster scoping**: all resource handlers receive the active cluster from middleware; global handlers fan out to all clusters
- **CORS**: set via `CORS_ALLOWED_ORIGINS` env var (comma-separated origins, defaults to `*` in dev)

## Frontend pages

| Page | Path | Description |
|------|------|-------------|
| Dashboard | `/` | Cluster overview with quick actions |
| ClusterManagement | `/clusters` | All clusters with status cards and register button |
| ClusterRegister | `/clusters/register` | Step-by-step cluster registration wizard |
| ClusterDetail | `/clusters/:name` | Single cluster health, agent status, resource counts, GPU capacity |
| GatewayList | `/gateways` | Gateway inventory |
| GatewayCreate | `/gateways/create` | 5-step gateway wizard |
| GatewayDetail | `/gateways/:ns/:name` | Gateway detail with listeners and routes |
| GatewayEdit | `/gateways/:ns/:name/edit` | Edit gateway configuration |
| RouteList | `/routes` | HTTPRoute inventory |
| RouteCreate | `/routes/new` | Route creation form |
| RouteDetail | `/routes/:ns/:name` | Route detail with rules and backends |
| RouteEdit | `/routes/:ns/:name/edit` | Edit route configuration |
| InferenceOverview | `/inference` | Summary cards and pool list |
| InferencePoolList | `/inference/pools` | Pool inventory with GPU utilization |
| InferencePoolCreate | `/inference/pools/create` | 6-step pool wizard |
| InferencePoolDetail | `/inference/pools/:ns/:name` | Tabbed: Overview, EPP, Metrics, Cost |
| PolicyList | `/policies` | Policy inventory by type |
| PolicyCreate | `/policies/new` | Policy creation form |
| CertificateList | `/certificates` | Certificate inventory with expiry |
| ObservabilityDashboard | `/observability` | Prometheus RED metrics |
| LogExplorer | `/logs` | ClickHouse log query UI |
| AuditLog | `/audit` | Audit trail with diff viewer |
| MigrationList | `/migration` | Migration plan inventory |
| MigrationNew | `/migration/new` | 4-step migration wizard |
| DiagnosticsHome | `/diagnostics` | Diagnostic tool hub |
| RouteCheck | `/diagnostics/route-check` | Route diagnostic checklist |
| CoexistenceDashboard | `/coexistence` | KIC + NGF side-by-side view |
| XCOverview | `/xc` | F5 XC publishing dashboard |
| SettingsPage | `/settings` | Application settings |

## Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CORS_ALLOWED_ORIGINS` | `*` | Comma-separated allowed CORS origins. Set to specific origins in production |
| `KUBECONFIG` | `~/.kube/config` | Path to kubeconfig file |
| `CLUSTER_NAME` | -- | (Agent) cluster name for heartbeat reporter |
| `HUB_API_ENDPOINT` | -- | (Agent) hub API URL for heartbeat reporter |
| `HUB_AUTH_TOKEN` | -- | (Agent) auth token for hub API |
