# Deployment Guide

Quick reference for common deployment scenarios. For full details see [Installation Guide](installation.md) and [Configuration Reference](configuration.md).

## Local development

```bash
# Terminal 1: API
cd api && go run ./cmd/server

# Terminal 2: Frontend
cd frontend && pnpm install && pnpm dev

# Open http://localhost:5173
```

## Docker Compose

```bash
docker compose -f deploy/docker-compose/docker-compose.yaml up
```

| Service | Port |
|---------|------|
| Frontend | 3000 |
| API | 8080 |
| ClickHouse | 8123, 9000 |
| OTel Collector | 4317, 4318 |

## Kubernetes (Helm) -- Single cluster

```bash
# Install CRDs
kubectl apply -f deploy/helm/ngf-console/crds/

# Install chart
helm install ngf-console deploy/helm/ngf-console \
  --namespace ngf-console --create-namespace
```

## Kubernetes (Helm) -- Multi-cluster hub

```bash
# Install CRDs (including ManagedCluster)
kubectl apply -f deploy/helm/ngf-console/crds/

# Install chart with multi-cluster enabled
helm install ngf-console deploy/helm/ngf-console \
  --namespace ngf-console --create-namespace \
  --set api.multicluster.enabled=true \
  --set api.multicluster.namespace=ngf-system
```

## Kubernetes -- Workload cluster agent

Install the agent chart on each workload cluster:

```bash
helm install ngf-console-agent charts/ngf-console-agent \
  --namespace ngf-system --create-namespace \
  --set cluster.name=workload-west \
  --set hub.apiEndpoint=https://hub.example.com \
  --set hub.otelEndpoint=hub.example.com:4317
```

The agent installs:
- **Operator** -- reconciles InferenceStack and GatewayBundle CRDs locally
- **Heartbeat reporter** -- sends health to hub every 30s
- **OTel forwarder** -- forwards telemetry to hub with `cluster_name` tagging

Verify agent pods:
```bash
kubectl get pods -n ngf-system
```

## Upgrade

```bash
# Hub
helm upgrade ngf-console deploy/helm/ngf-console --namespace ngf-console

# Agent (on each workload cluster)
helm upgrade ngf-console-agent charts/ngf-console-agent --namespace ngf-system
```

## Uninstall

```bash
# Hub
helm uninstall ngf-console --namespace ngf-console

# Agent (on each workload cluster)
helm uninstall ngf-console-agent --namespace ngf-system

# Optionally remove CRDs (WARNING: deletes all CRD instances)
kubectl delete -f deploy/helm/ngf-console/crds/
```

## Production checklist

- [ ] Set `CORS_ALLOWED_ORIGINS` to specific origins (not `*`)
- [ ] Use PostgreSQL for config database (not SQLite)
- [ ] Configure TLS for hub ingress
- [ ] Set resource limits on all deployments
- [ ] Configure alert webhooks (`--alert-webhooks`)
- [ ] Use a private container registry for images
- [ ] Verify agent RBAC: operator has write access, heartbeat has read-only
- [ ] Ensure hub OTel Collector port 4317 is reachable from workload clusters
- [ ] Verify hub API endpoint is reachable from workload clusters (for heartbeats)
