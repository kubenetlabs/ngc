# Deployment Guide

## Local development

Two terminals, no infrastructure required:

```bash
# Terminal 1
cd api && go run ./cmd/server

# Terminal 2
cd frontend && pnpm install && pnpm dev
```

Open http://localhost:5173. Inference metrics use mock data by default.

## Local with ClickHouse

For real metrics storage:

```bash
# Start ClickHouse + OTel Collector
make dev-compose

# Seed demo data
docker exec -i $(docker ps -q -f ancestor=clickhouse/clickhouse-server:24.1) \
  clickhouse-client < deploy/docker-compose/clickhouse/seed.sql

# Start API with ClickHouse
cd api && go run ./cmd/server --db-type clickhouse --clickhouse-url localhost:9000

# Start frontend
cd frontend && pnpm dev
```

## Docker Compose (full stack)

Runs all services in containers:

```bash
docker compose -f deploy/docker-compose/docker-compose.yaml up
```

| Service | Port | Description |
|---------|------|-------------|
| Frontend | 3000 | Nginx serving React build |
| API | 8080 | Go API server |
| ClickHouse | 8123, 9000 | Analytics database |
| OTel Collector | 4317, 4318 | Telemetry ingestion |

## Kubernetes with Helm

### Prerequisites

- Kubernetes 1.28+
- Helm 3.12+
- Gateway API CRDs installed
- NGINX Gateway Fabric deployed

### Install

```bash
helm install ngf-console deploy/helm/ngf-console \
  --namespace ngf-system \
  --create-namespace
```

### Configuration

Key values in `deploy/helm/ngf-console/values.yaml`:

```yaml
ngf:
  edition: enterprise          # or "oss"
  controllerNamespace: nginx-gateway

clickhouse:
  enabled: true                # Set false to use mock data
  persistence:
    size: 50Gi

inference:
  enabled: true
  costEstimation:
    enabled: true
    gpuPricing:
      A100: 3.67
      H100: 8.10
      L40S: 1.84

ingress:
  enabled: true
  className: nginx-gateway-fabric
  hostname: ngf-console.example.com
  tls:
    enabled: true
    secretName: ngf-console-tls
```

### Common overrides

```bash
# OSS edition, no ClickHouse
helm install ngf-console deploy/helm/ngf-console \
  --set ngf.edition=oss \
  --set clickhouse.enabled=false

# Custom hostname
helm install ngf-console deploy/helm/ngf-console \
  --set ingress.hostname=console.mycompany.com

# External ClickHouse
helm install ngf-console deploy/helm/ngf-console \
  --set clickhouse.enabled=false \
  --set api.env.CLICKHOUSE_URL=clickhouse.monitoring:9000
```

### Upgrade

```bash
helm upgrade ngf-console deploy/helm/ngf-console --namespace ngf-system
```

### Uninstall

```bash
helm uninstall ngf-console --namespace ngf-system
```

## Multi-cluster deployment

When the NGF Console API is deployed to one cluster but manages multiple clusters:

1. Create kubeconfig secrets for each remote cluster:

   ```bash
   kubectl create secret generic cluster-prod-kubeconfig \
     --from-file=kubeconfig=/path/to/prod-kubeconfig.yaml \
     -n ngf-system

   kubectl create secret generic cluster-staging-kubeconfig \
     --from-file=kubeconfig=/path/to/staging-kubeconfig.yaml \
     -n ngf-system
   ```

2. Create the clusters config:

   ```yaml
   # clusters.yaml
   clusters:
     - name: production
       displayName: "Production US-East"
       kubeconfig: /etc/ngf/kubeconfigs/prod.yaml
       default: true
     - name: staging
       displayName: "Staging EU-West"
       kubeconfig: /etc/ngf/kubeconfigs/staging.yaml
   ```

3. Mount secrets and config into the API pod via Helm values or a ConfigMap, and pass `--clusters-config /etc/ngf/clusters.yaml` to the API server.
