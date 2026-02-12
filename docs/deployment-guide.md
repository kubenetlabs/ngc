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

### Hub prerequisites

Before installing any agents, expose the hub's OTel Collector so workload clusters can forward telemetry. This is a separate service from the API.

```bash
# Expose the hub OTel Collector (default is ClusterIP -- agents cannot reach it)
kubectl patch svc ngf-console-otel-collector -n ngf-console \
  -p '{"spec":{"type":"LoadBalancer"}}'

# Get the external address (wait for EXTERNAL-IP to populate)
kubectl get svc ngf-console-otel-collector -n ngf-console
```

### Install the agent

```bash
helm install ngf-console-agent charts/ngf-console-agent \
  --namespace ngf-system --create-namespace \
  --set cluster.name=workload-west \
  --set hub.apiEndpoint=https://hub.example.com \
  --set hub.otelEndpoint=<otel-collector-lb-address>:4317
```

> **Important:** `hub.apiEndpoint` and `hub.otelEndpoint` point to different services. The API endpoint is the hub's HTTP LB/Ingress. The OTel endpoint is the OTel Collector LB on port 4317 (gRPC). Using the API LB address for `hub.otelEndpoint` will fail.

The agent installs:
- **Operator** -- reconciles InferenceStack and GatewayBundle CRDs locally
- **Heartbeat reporter** -- sends health to hub API every 30s
- **OTel forwarder** -- forwards telemetry to hub OTel Collector with `cluster_name` tagging

### Verify

```bash
# Agent pods should all be 1/1 Running
kubectl get pods -n ngf-system

# Check OTel forwarder logs for connectivity (no connection errors = healthy)
kubectl logs -n ngf-system deploy/ngf-console-agent-otel-collector --tail=5

# Check heartbeat is reaching the hub (should show "heartbeat sent" with status 200)
kubectl logs -n ngf-system deploy/ngf-console-agent-heartbeat --tail=5
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
- [ ] Expose hub OTel Collector as LoadBalancer/NodePort before installing agents
- [ ] Verify `hub.otelEndpoint` points to the OTel Collector LB (not the API LB)
- [ ] Verify hub API endpoint is reachable from workload clusters (for heartbeats)
- [ ] Test port 4317 connectivity from a workload cluster to the hub OTel Collector LB
