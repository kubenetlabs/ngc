# Installation Guide

## Prerequisites

| Tool | Version | Required for |
|------|---------|--------------|
| Go | 1.25+ | API, operator, controller, migration CLI (only for local dev) |
| Node.js | 20+ | Frontend |
| pnpm | 9+ | Frontend |
| kubectl | 1.28+ | Kubernetes interaction |
| Helm | 3.12+ | Helm-based deployment |
| Docker | 24+ | Container builds, Docker Compose |

### Kubernetes requirements

- Kubernetes 1.28+
- [Gateway API CRDs](https://gateway-api.sigs.k8s.io/guides/#installing-gateway-api) v1.2.0+
- [NGINX Gateway Fabric](https://github.com/nginx/nginx-gateway-fabric) v1.6+
- Cluster admin RBAC for CRD installation

## Option 1: Local development (demo mode)

The fastest way to run NGF Console. Uses mock data for inference metrics -- no Kubernetes cluster, Docker, or ClickHouse required.

### 1. Clone and build

```bash
git clone <repo-url> && cd ngc
```

### 2. Start the API server

```bash
cd api && go run ./cmd/server
```

The API starts on port 8080 with mock inference data and an in-process SQLite database at `./ngf-console.db`.

### 3. Start the frontend

In a second terminal:

```bash
cd frontend && pnpm install && pnpm dev
```

Open **http://localhost:5173**. The Vite dev server proxies API requests to `localhost:8080`.

### What works in demo mode

| Feature | Status |
|---------|--------|
| Inference Overview | Mock data |
| Inference Pool List + Detail | Mock data with live WebSocket |
| GPU Heatmap, EPP Decisions, Metrics charts | Mock data |
| Cost Estimation | Mock data |
| Dashboard | Mock data |
| Gateway/Route CRUD | Requires Kubernetes |
| Policies, Certificates | Requires Kubernetes |
| Prometheus RED metrics | Requires Prometheus |
| Log Explorer | Requires ClickHouse |

## Option 2: Local with Kubernetes

### 1. Install Gateway API CRDs

```bash
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.2.0/standard-install.yaml
```

### 2. Install NGF Console CRDs

```bash
kubectl apply -f deploy/k8s/ngf-console.f5.com_inferencestacks.yaml
kubectl apply -f deploy/k8s/ngf-console.f5.com_gatewaybundles.yaml
kubectl apply -f deploy/k8s/ngf-console.f5.com_distributedcloudpublishes.yaml
```

### 3. Install NGINX Gateway Fabric

Follow the [NGF installation guide](https://docs.nginx.com/nginx-gateway-fabric/installation/) or:

```bash
kubectl apply -f https://github.com/nginx/nginx-gateway-fabric/releases/download/v1.6.2/deploy.yaml
```

### 4. Start the operator

```bash
cd operator && go run ./cmd/
```

The operator watches for InferenceStack and GatewayBundle CRDs and reconciles child resources.

### 5. Start the API server

```bash
# Auto-discovers kubeconfig
cd api && go run ./cmd/server

# Or specify explicitly
cd api && go run ./cmd/server --kubeconfig /path/to/kubeconfig
```

Kubeconfig discovery order:
1. `--kubeconfig` flag
2. `KUBECONFIG` environment variable
3. In-cluster config (when running inside a pod)
4. `~/.kube/config`

### 6. Start the frontend

```bash
cd frontend && pnpm install && pnpm dev
```

## Option 3: Local with ClickHouse

Adds real metrics storage and log exploration.

### 1. Start infrastructure

```bash
make dev-compose
```

This starts ClickHouse (ports 8123, 9000) and the OpenTelemetry Collector (ports 4317, 4318).

### 2. Start the API with ClickHouse

```bash
cd api && go run ./cmd/server \
  --db-type clickhouse \
  --clickhouse-url localhost:9000
```

When `--db-type=clickhouse` is specified and the connection fails, the API will exit rather than silently falling back to mock data.

### 3. Start the frontend

```bash
cd frontend && pnpm install && pnpm dev
```

### Optional: Add Prometheus

If you have a Prometheus instance scraping NGINX Gateway Fabric metrics:

```bash
cd api && go run ./cmd/server \
  --db-type clickhouse \
  --clickhouse-url localhost:9000 \
  --prometheus-url http://localhost:9090
```

This enables the RED metrics endpoints (`/api/v1/metrics/summary`, `/by-route`, `/by-gateway`). Without `--prometheus-url`, these endpoints return 503.

## Option 4: Docker Compose

Runs the full stack in containers.

### 1. Build and start

```bash
docker compose -f deploy/docker-compose/docker-compose.yaml up --build
```

### 2. Access

| Service | URL | Description |
|---------|-----|-------------|
| Frontend | http://localhost:3000 | Nginx serving React build |
| API | http://localhost:8080 | Go API server |
| ClickHouse HTTP | http://localhost:8123 | ClickHouse HTTP interface |
| ClickHouse Native | localhost:9000 | ClickHouse native protocol |
| OTel Collector gRPC | localhost:4317 | OpenTelemetry gRPC receiver |
| OTel Collector HTTP | localhost:4318 | OpenTelemetry HTTP receiver |

### 3. Tear down

```bash
docker compose -f deploy/docker-compose/docker-compose.yaml down -v
```

## Option 5: Kubernetes with Helm

Production deployment to a Kubernetes cluster using pre-built container images from Docker Hub.

### Container images

The Helm chart uses the following Docker Hub images by default:

| Component | Image |
|-----------|-------|
| API | `danny2guns/ngf-console-api:0.1.0` |
| Frontend | `danny2guns/ngf-console-frontend:0.1.0` |
| Operator | `danny2guns/ngf-console-operator:0.1.0` |

No `--set` overrides are needed for image repositories -- the defaults in `values.yaml` point to Docker Hub.

### 1. Install CRDs

CRDs are installed separately from the Helm chart so they persist across upgrades and uninstalls:

```bash
kubectl apply -f deploy/helm/ngf-console/crds/
```

### 2. Install Gateway API CRDs

```bash
kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.2.0/standard-install.yaml
```

### 3. Install NGINX Gateway Fabric

```bash
kubectl apply -f https://github.com/nginx/nginx-gateway-fabric/releases/download/v1.6.2/deploy.yaml
```

### 4. Install NGF Console

```bash
helm install ngf-console deploy/helm/ngf-console \
  --namespace ngf-console \
  --create-namespace
```

### 5. Verify

```bash
# Check pods are running
kubectl get pods -n ngf-console

# Expected output:
# ngf-console-api-xxxx          1/1     Running
# ngf-console-frontend-xxxx     1/1     Running
# ngf-console-operator-xxxx     1/1     Running
# ngf-console-clickhouse-0      1/1     Running   (if enabled)
# ngf-console-otel-collector-0  1/1     Running   (if enabled)

# Check the API health endpoint
kubectl port-forward -n ngf-console svc/ngf-console-api 8080:8080
curl http://localhost:8080/api/v1/health
# {"status":"ok"}
```

### 6. Access the UI

```bash
# Port forward the frontend
kubectl port-forward -n ngf-console svc/ngf-console-frontend 3000:80

# Open http://localhost:3000
```

Or configure an Ingress/Gateway route (see [Configuration Reference](configuration.md#ingress)).

### Common Helm overrides

```bash
# Minimal install (no ClickHouse, no OTel, mock metrics)
helm install ngf-console deploy/helm/ngf-console \
  --namespace ngf-console --create-namespace \
  --set clickhouse.enabled=false \
  --set otelCollector.enabled=false

# Enterprise with external Prometheus
helm install ngf-console deploy/helm/ngf-console \
  --namespace ngf-console --create-namespace \
  --set ngf.edition=enterprise \
  --set prometheus.url=http://prometheus.monitoring:9090

# External ClickHouse (don't deploy built-in)
helm install ngf-console deploy/helm/ngf-console \
  --namespace ngf-console --create-namespace \
  --set clickhouse.enabled=false \
  --set api.clickhouseUrl=clickhouse.monitoring:9000

# Custom ingress hostname
helm install ngf-console deploy/helm/ngf-console \
  --namespace ngf-console --create-namespace \
  --set ingress.hostname=console.mycompany.com \
  --set ingress.tls.secretName=console-tls

# Use custom image registry
helm install ngf-console deploy/helm/ngf-console \
  --namespace ngf-console --create-namespace \
  --set api.image.repository=myregistry.example.com/ngf-console-api \
  --set frontend.image.repository=myregistry.example.com/ngf-console-frontend \
  --set operator.image.repository=myregistry.example.com/ngf-console-operator
```

### Building custom images

To build and push your own images:

```bash
# Build for linux/amd64 (required for most cloud Kubernetes clusters)
docker build --platform linux/amd64 -t myregistry/ngf-console-api:0.1.0 -f api/Dockerfile api/
docker build --platform linux/amd64 -t myregistry/ngf-console-frontend:0.1.0 -f frontend/Dockerfile frontend/
docker build --platform linux/amd64 -t myregistry/ngf-console-operator:0.1.0 -f operator/Dockerfile operator/

# Push
docker push myregistry/ngf-console-api:0.1.0
docker push myregistry/ngf-console-frontend:0.1.0
docker push myregistry/ngf-console-operator:0.1.0
```

Then install with your custom repository:

```bash
helm install ngf-console deploy/helm/ngf-console \
  --namespace ngf-console --create-namespace \
  --set api.image.repository=myregistry/ngf-console-api \
  --set frontend.image.repository=myregistry/ngf-console-frontend \
  --set operator.image.repository=myregistry/ngf-console-operator
```

### Upgrade

```bash
helm upgrade ngf-console deploy/helm/ngf-console \
  --namespace ngf-console
```

### Uninstall

```bash
# Remove the Helm release (CRDs and their instances are preserved)
helm uninstall ngf-console --namespace ngf-console

# Optionally remove CRDs (WARNING: deletes all InferenceStack and GatewayBundle instances)
kubectl delete -f deploy/helm/ngf-console/crds/
```

## Multi-cluster setup

### Local development

Create a `clusters.yaml` file:

```yaml
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

Start with:

```bash
cd api && go run ./cmd/server --clusters-config clusters.yaml
```

### Kubernetes deployment

1. Create kubeconfig secrets for each remote cluster:

```bash
kubectl create secret generic cluster-prod-kubeconfig \
  --from-file=kubeconfig=/path/to/prod-kubeconfig.yaml \
  -n ngf-system

kubectl create secret generic cluster-staging-kubeconfig \
  --from-file=kubeconfig=/path/to/staging-kubeconfig.yaml \
  -n ngf-system
```

2. Create a ConfigMap with the clusters config:

```bash
kubectl create configmap clusters-config \
  --from-file=clusters.yaml=/path/to/clusters.yaml \
  -n ngf-system
```

3. Mount the secrets and config into the API pod. Pass `--clusters-config /etc/ngf/clusters.yaml` to the API server.

### Cluster config rules

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Lowercase alphanumeric with hyphens, 1-63 chars |
| `displayName` | No | Human-readable name shown in the UI |
| `kubeconfig` | Yes | Absolute path to the kubeconfig file |
| `context` | No | Specific context within the kubeconfig; defaults to current-context |
| `default` | No | At most one cluster; if none, the first entry is used |

## Migration CLI

The migration CLI helps migrate from NGINX Ingress Controller to NGINX Gateway Fabric.

### Install

```bash
cd migration-cli && go build -o bin/ngf-migrate .
```

### Usage

```bash
# Scan cluster for KIC resources
./bin/ngf-migrate scan --kubeconfig /path/to/kubeconfig

# Generate a migration plan
./bin/ngf-migrate plan --kubeconfig /path/to/kubeconfig

# Apply the migration (dry-run first)
./bin/ngf-migrate apply --kubeconfig /path/to/kubeconfig --dry-run

# Validate migrated resources
./bin/ngf-migrate validate --kubeconfig /path/to/kubeconfig
```

See the [Migration Guide](migration-guide.md) for detailed migration workflows.

## Verification

After installation, verify all components are working:

```bash
# API health
curl http://localhost:8080/api/v1/health
# {"status":"ok"}

# List clusters
curl http://localhost:8080/api/v1/clusters
# [{"name":"default","displayName":"default","default":true}]

# Inference summary (mock data)
curl http://localhost:8080/api/v1/inference/metrics/summary
# {...metrics data...}

# WebSocket test (requires wscat: npm install -g wscat)
wscat -c ws://localhost:8080/api/v1/ws/inference/epp-decisions
# Streams live EPP decision data
```

## Troubleshooting

### API won't start

- **"failed to create kubernetes client"**: No kubeconfig found. Pass `--kubeconfig` or ensure `~/.kube/config` exists. The API still starts if you only need inference mock data -- check your kubeconfig path.
- **"failed to create clickhouse client"**: ClickHouse is unreachable at the given URL. When `--db-type=clickhouse` is explicitly set, a connection failure is fatal. Ensure ClickHouse is running or omit `--db-type` to use mock data.
- **"failed to open config database"**: SQLite database path is not writable. Check permissions on the `--config-db` path.

### Operator pods crash-looping

- Check CRDs are installed: `kubectl get crd inferencestacks.ngf-console.f5.com`
- Check RBAC: the operator needs `get/list/watch/create/update/patch/delete` on its CRDs and child resources
- Check logs: `kubectl logs -n ngf-system deploy/ngf-console-operator`

### Frontend shows "Network Error"

- Ensure the API is running on port 8080
- In development, the Vite proxy config in `frontend/vite.config.ts` forwards `/api` to `localhost:8080`
- In Docker Compose/Kubernetes, check the frontend's `VITE_API_URL` configuration

### WebSocket disconnects

- WebSocket connections go through `/api/v1/ws/inference/{topic}`
- Check that no reverse proxy is terminating WebSocket connections prematurely
- The hub drops messages to slow clients rather than blocking -- this is expected behavior
