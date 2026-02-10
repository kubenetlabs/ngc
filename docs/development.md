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

## Architecture overview

```
Browser (React)
    |
    v
Vite Dev Server (:5173) --proxy--> Go API Server (:8080)
                                        |
                          +-------------+-------------+
                          |             |             |
                    Kubernetes    MockProvider    ClickHouse
                    (Gateways)    (Inference)     (optional)
```

The API server has two data paths:

1. **Kubernetes** -- Gateway/HTTPRoute CRUD uses `controller-runtime` client talking to a real cluster
2. **MetricsProvider** -- Inference metrics use an interface with two implementations:
   - `MockProvider` (default) -- synthetic data, no external dependencies
   - `ClickHouse Provider` -- queries real data from ClickHouse tables

## Data flow: Inference metrics

```
MockProvider / ClickHouse
        |
        v
MetricsProvider interface
        |
        v
handlers/inference.go          -- ListPools, GetPool
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
ws_topics.go (mock generators, 1-15s intervals)
        |
        v
ws_hub.go (topic-based pub/sub)
        |
        v
/ws/inference/epp-decisions    -- EPP routing decisions (1s)
/ws/inference/gpu-metrics      -- Pod GPU metrics (2s)
/ws/inference/scaling-events   -- Autoscaling events (15s)
        |
        v
EPPDecisionVisualizer.tsx      -- Live pod cards with routing animation
```

## Adding a new inference metric

1. **Add the type** to `api/internal/inference/types.go`
2. **Add the method** to the `MetricsProvider` interface in `api/internal/inference/provider.go`
3. **Implement mock data** in `api/internal/inference/mock_provider.go`
4. **Add the response type** to `api/internal/handlers/inference_responses.go`
5. **Add the handler method** to `api/internal/handlers/inference_metrics.go`
6. **Register the route** in `api/internal/server/server.go` under the `/inference/metrics/` group
7. **Add the frontend fetch function** to `frontend/src/api/inference.ts`
8. **Add the TypeScript type** to `frontend/src/types/inference.ts`
9. **Use it in a component** (page or chart)

## Adding a new WebSocket topic

1. Add a generator in `api/internal/server/ws_topics.go` using `hub.AddGenerator()`
2. Register the WS route in `api/internal/server/server.go`:
   ```go
   r.Get("/ws/inference/my-topic", s.Hub.ServeWS("my-topic"))
   ```
3. Connect from the frontend using the pattern in `EPPDecisionVisualizer.tsx`

## Running tests

```bash
# All Go tests
cd api && go test ./...

# Specific package
cd api && go test ./internal/inference/...
cd api && go test ./internal/handlers/...
cd api && go test ./internal/server/...

# Frontend
cd frontend && pnpm test
cd frontend && pnpm lint

# Everything
make test
make lint
```

## Key conventions

- **Go handlers**: `func (h *Handler) Method(w http.ResponseWriter, r *http.Request)`
- **Go errors**: `fmt.Errorf("context: %w", err)`
- **Go imports**: stdlib, then external, then internal
- **React**: functional components, TanStack Query for data fetching, Zustand for state
- **Types**: Go response types in `handlers/inference_responses.go`, TypeScript types in `src/types/inference.ts`
- **Component files**: PascalCase (e.g., `GPUHeatmap.tsx`)
