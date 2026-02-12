# Development Guide

## Prerequisites

- Go 1.25+
- Node.js 20+
- pnpm 9+ (`npm install -g pnpm`)
- A kubeconfig with access to a Kubernetes cluster (optional -- inference features work without one)

## Getting started

```bash
git clone <repo-url> && cd ngc
```

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

## Architecture overview

```
Browser (React)
    |
    v
Vite Dev Server (:5173) --proxy--> Go API Server (:8080)
                                        |
                          +-------------+-------------+
                          |             |             |
                    Kubernetes    MockProvider    Config DB
                    (Gateways)    (Inference)     (SQLite)
                          |
                    Operator (CRDs)
```

The API server has several data paths:

1. **Kubernetes** -- Gateway/HTTPRoute CRUD uses `client-go` talking to a real cluster
2. **MetricsProvider** -- Inference metrics use an interface with two implementations:
   - `MockProvider` (default) -- synthetic data, no external dependencies
   - `ClickHouse Provider` -- queries real data from ClickHouse tables
3. **Prometheus** -- RED metrics (Rate, Errors, Duration) from NGF's Prometheus exporter
4. **Config DB** -- SQLite for alert rules, audit log, and saved views
5. **Operator** -- CRD-based reconciliation for InferenceStack and GatewayBundle

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
useWebSocket.ts hook           -- Auto-reconnect, ref-stable callbacks
        |
        v
EPPDecisionVisualizer.tsx      -- Live pod cards with routing animation
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

## Adding a new API handler

1. **Create the handler** in `api/internal/handlers/{domain}.go`
2. **Define types** -- request/response structs with JSON tags
3. **Register routes** in `api/internal/server/server.go` in `mountResourceRoutes()`
4. **Add frontend API client** in `frontend/src/api/{domain}.ts`
5. **Add TypeScript types** in `frontend/src/types/{domain}.ts`
6. **Create page component** in `frontend/src/pages/{Page}.tsx`
7. **Register route** in `frontend/src/routes.tsx`

## Adding a new inference metric

1. **Add the type** to `api/internal/inference/types.go`
2. **Add the method** to the `MetricsProvider` interface in `api/internal/inference/provider.go`
3. **Implement mock data** in `api/internal/inference/mock_provider.go`
4. **Implement ClickHouse query** in `api/internal/clickhouse/provider.go`
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

## Running tests

```bash
# All Go tests
cd api && go test ./...
cd operator && go test ./...

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
make build            # Build all components
make test             # Run all tests
make lint             # Run all linters
make docker-build     # Build all Docker images
make helm-template    # Render Helm templates
make generate-crds    # Regenerate CRD YAML from Go types
make generate-deepcopy # Regenerate deepcopy methods
make clean            # Remove build artifacts and stop containers
```

Individual builds:

```bash
cd api && go build -o bin/api-server ./cmd/server
cd operator && go build -o bin/operator ./cmd/
cd migration-cli && go build -o bin/ngf-migrate .
cd frontend && pnpm build
```

## Key conventions

- **Go handlers**: `func (h *Handler) Method(w http.ResponseWriter, r *http.Request)`
- **Go errors**: `fmt.Errorf("context: %w", err)`
- **Go imports**: stdlib, then external, then internal
- **Go logging**: `slog` structured logging (no `log` or `fmt.Println`)
- **React**: functional components, TanStack Query for data fetching, Zustand for state
- **Types**: Go response types in handler files, TypeScript types in `src/types/`
- **Component files**: PascalCase (e.g., `GPUHeatmap.tsx`)
- **API client files**: camelCase (e.g., `inference.ts`)
- **CRD naming**: `{name}-{child}` (e.g., `llama-70b-pool`, `llama-70b-epp-config`)
- **Enterprise features**: gated by edition detection, greyed out in OSS mode
- **Stubs**: unimplemented endpoints return 501 Not Implemented with JSON `{"error": "not implemented"}`

## Frontend pages

| Page | Path | Description |
|------|------|-------------|
| Dashboard | `/` | Cluster overview with quick actions |
| GatewayList | `/gateways` | Gateway inventory |
| GatewayCreate | `/gateways/new` | 5-step gateway wizard |
| GatewayDetail | `/gateways/:ns/:name` | Gateway detail with listeners and routes |
| GatewayEdit | `/gateways/:ns/:name/edit` | Edit gateway configuration |
| RouteList | `/routes` | HTTPRoute inventory |
| RouteCreate | `/routes/new` | Route creation form |
| RouteDetail | `/routes/:ns/:name` | Route detail with rules and backends |
| RouteEdit | `/routes/:ns/:name/edit` | Edit route configuration |
| InferenceOverview | `/inference` | Summary cards and pool list |
| InferencePoolList | `/inference/pools` | Pool inventory with GPU utilization |
| InferencePoolCreate | `/inference/pools/new` | 6-step pool wizard |
| InferencePoolDetail | `/inference/pools/:name` | Tabbed: Overview, EPP, Metrics, Cost |
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
