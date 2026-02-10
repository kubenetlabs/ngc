# NGF Console

Web-based management platform for [NGINX Gateway Fabric](https://github.com/nginx/nginx-gateway-fabric) with native Kubernetes Gateway API and [Gateway Inference Extensions](https://gateway-api.sigs.k8s.io/geps/gep-1742/) support.

## What it does

- **Gateway Management** -- CRUD for Gateways, HTTPRoutes, GRPCRoutes, TLSRoutes, TCPRoutes, UDPRoutes
- **Inference Observability** -- Real-time GPU metrics, EPP (Endpoint Picker) decision visualization, per-pod KV-cache and queue depth monitoring, TTFT histograms, cost estimation
- **Multi-Cluster** -- Manage multiple Kubernetes clusters from a single console
- **WebSocket Streaming** -- Live EPP decision feed, GPU metrics updates, scaling events

## Prerequisites

| Tool | Version | Install |
|------|---------|---------|
| Go | 1.25+ | https://go.dev/dl/ |
| Node.js | 20+ | https://nodejs.org/ |
| pnpm | 9+ | `npm install -g pnpm` |
| kubectl | any | https://kubernetes.io/docs/tasks/tools/ |

A Kubernetes cluster with NGINX Gateway Fabric installed is required for gateway features. Inference features work fully with mock data (no cluster needed).

## Quick Start (Demo Mode)

The fastest way to see the product end-to-end. Uses mock data for inference metrics -- no Kubernetes cluster, Docker, or ClickHouse required.

**Terminal 1 -- API server:**

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

Open **http://localhost:5173** in your browser.

### Demo walkthrough

| Page | URL | What you'll see |
|------|-----|-----------------|
| Dashboard | http://localhost:5173/ | Overview with cluster status |
| Inference Overview | http://localhost:5173/inference | Summary cards (Total Pools, GPUs, Avg GPU Util, Avg TTFT), pool list, recent EPP decisions |
| Inference Pools | http://localhost:5173/inference/pools | Table of all pools with GPU utilization bars, status badges |
| Pool Detail | Click any pool name | Tabbed view: Overview, EPP Decisions (live WebSocket feed), Metrics (time-series charts), Cost |
| Gateways | http://localhost:5173/gateways | Gateway list (requires Kubernetes connection) |
| Routes | http://localhost:5173/routes | HTTPRoute list (requires Kubernetes connection) |

The Inference Pool Detail page is the hero feature. Click a pool, then explore the tabs:

- **Overview** -- Summary metrics + GPU heatmap showing per-pod utilization
- **EPP Decisions** -- Live visualization of request routing decisions streaming via WebSocket. Pod cards flash when they receive new requests.
- **Metrics** -- TTFT histogram, tokens-per-second throughput, queue depth, GPU utilization, and KV-cache time-series charts
- **Cost** -- Hourly/daily/monthly cost estimates based on GPU type and replica count

## Connecting to a Kubernetes Cluster

### Single cluster (default)

The API server auto-discovers your kubeconfig in this order:

1. In-cluster config (when running inside Kubernetes)
2. `--kubeconfig` flag
3. `KUBECONFIG` environment variable
4. `~/.kube/config`

```bash
# Uses ~/.kube/config automatically
cd api && go run ./cmd/server

# Or specify explicitly
cd api && go run ./cmd/server --kubeconfig /path/to/kubeconfig
```

The cluster must have the [Gateway API CRDs](https://gateway-api.sigs.k8s.io/guides/#installing-gateway-api) installed:

```bash
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.2.0/standard-install.yaml
```

### Multi-cluster

To manage multiple clusters, create a YAML config file:

```yaml
# clusters.yaml
clusters:
  - name: production
    displayName: "Production US-East"
    kubeconfig: /path/to/prod-kubeconfig.yaml
    default: true

  - name: staging
    displayName: "Staging EU-West"
    kubeconfig: /path/to/staging-kubeconfig.yaml
    context: staging-admin    # optional: use a specific context

  - name: dev
    displayName: "Dev Local"
    kubeconfig: /path/to/dev-kubeconfig.yaml
```

Start the server with:

```bash
cd api && go run ./cmd/server --clusters-config clusters.yaml
```

**Config rules:**
- `name` -- lowercase alphanumeric with hyphens, 1-63 chars (e.g., `prod-us-east`)
- `kubeconfig` -- absolute path to the kubeconfig file for this cluster
- `context` -- (optional) specific context within the kubeconfig; defaults to current-context
- `default` -- (optional) at most one cluster can be marked default; if none, the first entry is used

The frontend automatically routes API requests to the selected cluster. Use the cluster switcher in the UI to change between clusters.

**API routing with multi-cluster:**

```
# Cluster-scoped (explicit)
GET /api/v1/clusters/production/gateways
GET /api/v1/clusters/staging/inference/pools

# Legacy (uses default cluster)
GET /api/v1/gateways
GET /api/v1/inference/pools
```

### Adding a new NGF instance

Each NGF instance is a Kubernetes cluster running NGINX Gateway Fabric. To add one:

1. **Install NGINX Gateway Fabric** on the target cluster:

   ```bash
   kubectl apply -f https://github.com/nginx/nginx-gateway-fabric/releases/download/v1.6.2/deploy.yaml
   ```

2. **Ensure you have a kubeconfig** that can reach the cluster:

   ```bash
   # Verify connectivity
   kubectl --kubeconfig /path/to/new-cluster.yaml get gateways -A
   ```

3. **Add to clusters.yaml** (if using multi-cluster mode):

   ```yaml
   clusters:
     # ... existing clusters ...
     - name: new-cluster
       displayName: "New NGF Instance"
       kubeconfig: /path/to/new-cluster.yaml
   ```

4. **Restart the API server.** The new cluster appears in the UI cluster switcher immediately.

For single-cluster mode, just point `--kubeconfig` at the new cluster's kubeconfig and restart.

## Running with ClickHouse (Production Metrics)

For real observability data instead of mock data:

```bash
# Start ClickHouse and OpenTelemetry Collector
make dev-compose

# Seed demo data
docker exec -i $(docker ps -q -f ancestor=clickhouse/clickhouse-server:24.1) \
  clickhouse-client < deploy/docker-compose/clickhouse/seed.sql

# Start API with ClickHouse backend
cd api && go run ./cmd/server --db-type clickhouse --clickhouse-url localhost:9000
```

## Project Structure

```
ngc/
  api/                        # Go API server
    cmd/server/               # Entry point
    internal/
      handlers/               # HTTP handlers (gateways, routes, inference, metrics)
      inference/              # MetricsProvider interface, mock implementation, types
      clickhouse/             # ClickHouse provider implementation
      cluster/                # Multi-cluster manager
      kubernetes/             # K8s client wrapper
      server/                 # Chi router, middleware, WebSocket hub
  frontend/                   # React + TypeScript + Vite
    src/
      api/                    # Axios API functions
      components/inference/   # GPU heatmap, EPP visualizer, charts, pod cards
      pages/                  # InferenceOverview, InferencePoolList, InferencePoolDetail
      types/                  # TypeScript interfaces
      store/                  # Zustand state (cluster, settings)
  controller/                 # Kubernetes controller (controller-runtime)
  migration-cli/              # NGINX config migration tool
  deploy/
    docker-compose/           # Dev environment (ClickHouse, OTel Collector)
    helm/ngf-console/         # Production Helm chart
```

## Build & Test

```bash
# Build everything
make build

# Run all tests
make test

# Lint
make lint

# Individual components
cd api && go test ./...
cd frontend && pnpm build
cd frontend && pnpm lint
```

## API Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `8080` | HTTP server listen port |
| `--kubeconfig` | auto | Path to kubeconfig file |
| `--clusters-config` | (none) | Path to multi-cluster YAML config |
| `--db-type` | `mock` | Metrics provider: `mock` or `clickhouse` |
| `--clickhouse-url` | `localhost:9000` | ClickHouse connection URL |
| `--version` | | Print version and exit |

## Deployment

### Helm (production)

```bash
helm install ngf-console deploy/helm/ngf-console \
  --namespace ngf-system \
  --create-namespace \
  --set clickhouse.enabled=true \
  --set inference.enabled=true
```

See `deploy/helm/ngf-console/values.yaml` for all configuration options.

### Docker Compose (full local stack)

```bash
docker compose -f deploy/docker-compose/docker-compose.yaml up
```

This starts the frontend (port 3000), API (port 8080), ClickHouse (port 9000), and OTel Collector (port 4317).
