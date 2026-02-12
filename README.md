# NGF Console

Web-based management platform for [NGINX Gateway Fabric](https://github.com/nginx/nginx-gateway-fabric) with native Kubernetes Gateway API and [Gateway Inference Extensions](https://gateway-api.sigs.k8s.io/geps/gep-1742/) support.

## Features

- **Gateway Management** -- CRUD for Gateways, HTTPRoutes via GatewayBundle CRDs with operator-driven reconciliation
- **Inference Observability** -- Real-time GPU metrics, EPP decision visualization, per-pod KV-cache and queue depth monitoring, TTFT histograms, cost estimation
- **Operator-Driven Architecture** -- Declarative CRDs (InferenceStack, GatewayBundle) with drift detection, self-healing, and status aggregation
- **Multi-Cluster** -- Hub-spoke architecture with CRD-based cluster registration, per-cluster agents, global cross-cluster views, and circuit-breaker health checking
- **Migration Tooling** -- Import NGINX configs, Ingress, and VirtualServer resources; analyze, generate, and apply Gateway API equivalents
- **Observability Stack** -- ClickHouse analytics with per-cluster telemetry, Prometheus RED metrics, OpenTelemetry collection, log exploration
- **Alerting** -- Configurable alert rules with webhook notifications for cert expiry, error rates, and GPU saturation
- **WebSocket Streaming** -- Live EPP decision feed, GPU metrics updates, scaling events
- **F5 Distributed Cloud** -- Publish routes and inference endpoints to F5 XC (Enterprise)

## Architecture

```
                          +-----------------+
                          |    Browser      |
                          |  React + Vite   |
                          +-------+---------+
                                  |
                                  v
                 +-------------------------------+
                 |        Hub Cluster            |
                 |  +-------------------------+  |
                 |  |  Go API Server (Chi)    |  |
                 |  |  + WebSocket Hub        |  |
                 |  +--+--+--+--+--+--+------+  |
                 |     |  |  |  |  |  |         |
                 |     v  v  |  v  v  v         |
                 |  K8s Prom |  DB  CH OTel     |
                 |           |                  |
                 |  +--------+----------+       |
                 |  | ManagedCluster    |       |
                 |  | CRDs + Operator   |       |
                 |  +-------------------+       |
                 +------+-------------+---------+
                        |             |
              +---------+             +---------+
              v                                 v
   +---------------------+          +---------------------+
   |  Workload Cluster A |          |  Workload Cluster B |
   |  +---------------+  |          |  +---------------+  |
   |  | Agent:        |  |          |  | Agent:        |  |
   |  |  Operator     |  |          |  |  Operator     |  |
   |  |  Heartbeat    |  |          |  |  Heartbeat    |  |
   |  |  OTel Fwd     |  |          |  |  OTel Fwd     |  |
   |  +---------------+  |          |  +---------------+  |
   |  + NGF + Workloads  |          |  + NGF + Workloads  |
   +---------------------+          +---------------------+
```

**Components:**

| Component | Directory | Description |
|-----------|-----------|-------------|
| Frontend | `frontend/` | React 18 + TypeScript + Vite + Tailwind CSS |
| API Server | `api/` | Go + Chi router + WebSocket hub |
| Operator | `operator/` | controller-runtime, reconciles InferenceStack and GatewayBundle CRDs |
| Agent | `agent/` | Workload cluster heartbeat reporter |
| Migration CLI | `migration-cli/` | Cobra CLI for KIC-to-NGF migration |
| Agent Chart | `charts/ngf-console-agent/` | Helm chart for workload cluster agents (operator, heartbeat, OTel forwarder) |

## Container Images

Pre-built container images are available on Docker Hub:

| Component | Image | Description |
|-----------|-------|-------------|
| API | `danny2guns/ngf-console-api:0.1.0` | Go API server |
| Frontend | `danny2guns/ngf-console-frontend:0.1.0` | Nginx serving React build |
| Operator | `danny2guns/ngf-console-operator:0.1.0` | CRD operator (controller-runtime) |
| Agent Heartbeat | `danny2guns/ngf-console-agent-heartbeat:0.1.0` | Heartbeat reporter for workload clusters |

```bash
# Pull all images
docker pull danny2guns/ngf-console-api:0.1.0
docker pull danny2guns/ngf-console-frontend:0.1.0
docker pull danny2guns/ngf-console-operator:0.1.0
docker pull danny2guns/ngf-console-agent-heartbeat:0.1.0
```

These are the default images in the Helm charts. No `--set` overrides needed for a standard install.

## Prerequisites

| Tool | Version | Install |
|------|---------|---------|
| Go | 1.25+ | https://go.dev/dl/ |
| Node.js | 20+ | https://nodejs.org/ |
| pnpm | 9+ | `npm install -g pnpm` |
| kubectl | 1.28+ | https://kubernetes.io/docs/tasks/tools/ |
| Helm | 3.12+ | https://helm.sh/docs/intro/install/ |

Go and Node.js are only needed for local development. For Kubernetes deployment using the pre-built container images, only `kubectl` and `helm` are required.

A Kubernetes cluster with NGINX Gateway Fabric installed is required for gateway features. Inference features work fully with mock data (no cluster needed).

## Quick Start

### Demo mode (no cluster required)

**Terminal 1 -- API server:**

```bash
cd api && go run ./cmd/server
```

**Terminal 2 -- Frontend:**

```bash
cd frontend && pnpm install && pnpm dev
```

Open **http://localhost:5173**. Inference metrics use mock data. Gateway/route pages require a Kubernetes cluster.

### With a Kubernetes cluster

```bash
# Install Gateway API CRDs
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.2.0/standard-install.yaml

# Install NGF Console CRDs
kubectl apply -f deploy/k8s/ngf-console.f5.com_inferencestacks.yaml
kubectl apply -f deploy/k8s/ngf-console.f5.com_gatewaybundles.yaml
kubectl apply -f deploy/k8s/ngf-console.f5.com_distributedcloudpublishes.yaml

# Start API (auto-discovers ~/.kube/config)
cd api && go run ./cmd/server

# Or specify a kubeconfig
cd api && go run ./cmd/server --kubeconfig /path/to/kubeconfig
```

### With ClickHouse (real metrics)

```bash
# Start ClickHouse + OTel Collector
make dev-compose

# Start API with ClickHouse backend
cd api && go run ./cmd/server --db-type clickhouse --clickhouse-url localhost:9000
```

### Docker Compose (full stack)

```bash
docker compose -f deploy/docker-compose/docker-compose.yaml up
```

| Service | Port | Description |
|---------|------|-------------|
| Frontend | 3000 | Nginx serving React build |
| API | 8080 | Go API server |
| ClickHouse | 8123, 9000 | Analytics database |
| OTel Collector | 4317, 4318 | Telemetry ingestion |

## Kubernetes Deployment

### Helm

The Helm chart defaults to pulling pre-built images from Docker Hub (`danny2guns/ngf-console-*:0.1.0`).

```bash
# Install CRDs first (persisted across helm upgrades/uninstalls)
kubectl apply -f deploy/helm/ngf-console/crds/

# Install the chart
helm install ngf-console deploy/helm/ngf-console \
  --namespace ngf-console \
  --create-namespace

# Verify
kubectl get pods -n ngf-console
```

Common overrides:

```bash
# Minimal (no ClickHouse, no OTel, mock metrics)
helm install ngf-console deploy/helm/ngf-console \
  --namespace ngf-console --create-namespace \
  --set clickhouse.enabled=false \
  --set otelCollector.enabled=false

# Enterprise with Prometheus
helm install ngf-console deploy/helm/ngf-console \
  --namespace ngf-console --create-namespace \
  --set ngf.edition=enterprise \
  --set prometheus.url=http://prometheus.monitoring:9090

# External ClickHouse
helm install ngf-console deploy/helm/ngf-console \
  --namespace ngf-console --create-namespace \
  --set clickhouse.enabled=false \
  --set api.clickhouseUrl=clickhouse.monitoring:9000

# Custom image registry
helm install ngf-console deploy/helm/ngf-console \
  --namespace ngf-console --create-namespace \
  --set api.image.repository=myregistry/ngf-console-api \
  --set frontend.image.repository=myregistry/ngf-console-frontend \
  --set operator.image.repository=myregistry/ngf-console-operator
```

See [`docs/installation.md`](docs/installation.md) for full installation instructions and [`docs/configuration.md`](docs/configuration.md) for all configuration options.

## Multi-Cluster

NGF Console supports two multi-cluster modes:

### CRD-based (recommended for production)

Uses `ManagedCluster` custom resources on the hub cluster to register and manage workload clusters. Each workload cluster runs a lightweight agent (heartbeat reporter + OTel forwarder).

```bash
# 1. Enable CRD-based multi-cluster on the hub API server
cd api && go run ./cmd/server --multicluster --multicluster-namespace ngf-system

# 2. Register a workload cluster via the API
curl -X POST http://localhost:8080/api/v1/clusters \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "workload-west",
    "displayName": "Workload US-West-2",
    "region": "us-west-2",
    "environment": "production",
    "kubeconfig": "<base64-encoded-kubeconfig>"
  }'

# 3. Install the agent on the workload cluster
helm install ngf-console-agent charts/ngf-console-agent \
  --namespace ngf-system --create-namespace \
  --set cluster.name=workload-west \
  --set hub.apiEndpoint=https://hub.example.com \
  --set hub.otelEndpoint=hub.example.com:4317
```

The agent sends heartbeats every 30 seconds. The hub tracks cluster health with per-cluster circuit breakers and exposes global aggregation endpoints for cross-cluster views.

### File-based (lightweight, for development)

Uses a static YAML configuration file:

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
```

```bash
cd api && go run ./cmd/server --clusters-config clusters.yaml
```

The frontend cluster selector routes API requests to the selected cluster. Both cluster-scoped (`/api/v1/clusters/{name}/...`) and legacy (`/api/v1/...`) routes are supported. An "All Clusters" mode provides cross-cluster aggregation views.

See [`docs/installation.md`](docs/installation.md#multi-cluster-setup) for detailed multi-cluster setup instructions.

## Project Structure

```
ngc/
  api/                          # Go API server (Chi router)
    cmd/server/                 # Entry point
    internal/
      handlers/                 # HTTP handlers (30+ handler files)
      inference/                # MetricsProvider interface + mock
      clickhouse/               # ClickHouse provider implementation
      prometheus/               # Prometheus RED metrics client
      cluster/                  # File-based multi-cluster manager
      multicluster/             # CRD-based multi-cluster (ClientPool, health checker, circuit breaker)
      kubernetes/               # K8s client wrapper
      database/                 # SQLite/PostgreSQL config store
      alerting/                 # Alert evaluation engine + webhooks
      server/                   # Chi router, middleware, WebSocket hub
    pkg/
      types/                    # Shared API types
      version/                  # Build version info
  frontend/                     # React 18 + TypeScript + Vite
    src/
      api/                      # API client functions
      components/               # Shared UI components
      pages/                    # 26 page components
      types/                    # TypeScript interfaces
      store/                    # Zustand state stores (cluster, settings)
      hooks/                    # Custom React hooks (WebSocket, etc.)
  operator/                     # Kubernetes operator (controller-runtime)
    api/v1alpha1/               # CRD type definitions
    cmd/                        # Operator entry point
    internal/controller/        # Reconciliation controllers
    config/crd/bases/           # Generated CRD YAML
  agent/                        # Workload cluster agent
    cmd/heartbeat/              # Heartbeat reporter binary
  migration-cli/                # KIC migration CLI (cobra)
    cmd/                        # scan, plan, apply, validate commands
  charts/
    ngf-console-agent/          # Agent Helm chart for workload clusters
      templates/                # Operator, heartbeat, OTel deployments + RBAC
      values.yaml               # Agent configuration values
  deploy/
    docker-compose/             # Dev environment (ClickHouse, OTel)
    helm/ngf-console/           # Hub Helm chart
    k8s/                        # Standalone CRD manifests
    manifests/                  # Generated install manifests
  docs/                         # Documentation
```

## Custom Resource Definitions

The operator manages two primary CRDs, and the multi-cluster system uses a third:

### ManagedCluster

Represents a registered Kubernetes cluster in the hub. Used by the CRD-based multi-cluster system.

```yaml
apiVersion: ngf-console.f5.com/v1alpha1
kind: ManagedCluster
metadata:
  name: workload-west
  namespace: ngf-system
spec:
  displayName: "Workload US-West-2"
  region: us-west-2
  environment: production
  kubeconfigSecretRef:
    name: workload-west-kubeconfig
  ngfEdition: enterprise
```

### InferenceStack

Declares an inference serving stack. The operator reconciles child resources: InferencePool, EPP ConfigMap, KEDA ScaledObject, HTTPRoute, and DCGM DaemonSet.

```yaml
apiVersion: ngf-console.f5.com/v1alpha1
kind: InferenceStack
metadata:
  name: llama-70b
  namespace: inference
spec:
  modelName: meta-llama/Llama-3-70B-Instruct
  servingBackend: vllm
  pool:
    gpuType: H100
    gpuCount: 4
    replicas: 6
```

### GatewayBundle

Declares a gateway with optional enterprise features. The operator reconciles the Gateway child plus NginxProxy, WAF, SnippetsFilter, and TLS Secrets.

```yaml
apiVersion: ngf-console.f5.com/v1alpha1
kind: GatewayBundle
metadata:
  name: main-gateway
  namespace: default
spec:
  gatewayClassName: nginx
  listeners:
    - name: http
      port: 80
      protocol: HTTP
```

## API Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `8080` | HTTP server listen port |
| `--kubeconfig` | auto | Path to kubeconfig file |
| `--clusters-config` | (none) | Path to multi-cluster YAML config (file-based mode) |
| `--multicluster` | `false` | Enable CRD-based multi-cluster mode (reads ManagedCluster CRDs) |
| `--multicluster-namespace` | `ngf-system` | Namespace for ManagedCluster CRDs |
| `--multicluster-default` | (auto) | Default cluster name in multi-cluster mode |
| `--db-type` | `mock` | Metrics provider: `mock` or `clickhouse` |
| `--clickhouse-url` | `localhost:9000` | ClickHouse connection URL |
| `--prometheus-url` | (none) | Prometheus server URL for RED metrics |
| `--config-db` | `ngf-console.db` | Path to SQLite config database |
| `--alert-webhooks` | (none) | Comma-separated webhook URLs for alert notifications |
| `--version` | | Print version and exit |

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CORS_ALLOWED_ORIGINS` | `*` | Comma-separated list of allowed CORS origins. Set to specific origins in production |

## Build & Test

```bash
make build            # Build all components
make test             # Run all tests
make lint             # Run all linters
make docker-build     # Build all Docker images
make helm-template    # Render Helm templates
make generate-crds    # Regenerate CRD YAML from Go types
```

Individual components:

```bash
cd api && go build ./... && go test ./...
cd operator && go build ./... && go test ./...
cd frontend && pnpm build && pnpm lint
cd migration-cli && go build ./...
```

## Documentation

| Document | Description |
|----------|-------------|
| [Installation Guide](docs/installation.md) | Step-by-step installation for dev, Docker Compose, Kubernetes, and multi-cluster |
| [Configuration Reference](docs/configuration.md) | All API flags, Helm values, environment variables, agent chart values, and CRD specs |
| [Architecture](docs/architecture.md) | System design, hub-spoke multi-cluster, data flows, and component interactions |
| [Development Guide](docs/development.md) | Local setup, coding conventions, and contribution workflow |
| [API Reference](docs/api-reference.md) | Complete REST API endpoint documentation including cluster management and global aggregation |
| [Migration Guide](docs/migration-guide.md) | Migrating from NGINX Ingress Controller to Gateway API |
| [XC Integration](docs/xc-integration.md) | F5 Distributed Cloud publishing and metrics |
