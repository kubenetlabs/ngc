# API Reference

The NGF Console API server exposes a RESTful API at `/api/v1/`. All resource routes support both cluster-scoped (`/api/v1/clusters/{cluster}/...`) and legacy (`/api/v1/...`) paths. Cluster management and global aggregation endpoints operate at the hub level.

## Health

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/health` | Health check for liveness/readiness probes |

Response: `{"status": "ok"}`

## Cluster Management

Hub-level endpoints for managing registered clusters. Available in CRD-based multi-cluster mode (`--multicluster`).

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/clusters` | List all registered clusters with health status |
| POST | `/api/v1/clusters` | Register a new cluster (creates ManagedCluster CRD + kubeconfig Secret) |
| GET | `/api/v1/clusters/summary` | Global summary across all clusters (total clusters, gateways, routes, GPUs) |
| GET | `/api/v1/clusters/{cluster}/detail` | Get detailed cluster info (edition, K8s version, NGF version, agent status, resource counts, GPU capacity) |
| DELETE | `/api/v1/clusters/{cluster}` | Unregister a cluster (deletes ManagedCluster CRD + kubeconfig Secret) |
| POST | `/api/v1/clusters/{cluster}/test` | Test connectivity to a cluster |
| POST | `/api/v1/clusters/{cluster}/install-agent` | Generate Helm install command for the agent chart |
| POST | `/api/v1/clusters/{cluster}/heartbeat` | Receive health report from a cluster agent |

### Register cluster

```bash
curl -X POST http://localhost:8080/api/v1/clusters \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "workload-west",
    "displayName": "Workload US-West-2",
    "region": "us-west-2",
    "environment": "production",
    "kubeconfig": "<kubeconfig-file-contents>",
    "ngfEdition": "enterprise"
  }'
```

Response (201):
```json
{"message": "cluster registered", "name": "workload-west"}
```

Cluster names must be valid DNS subdomains: lowercase alphanumeric and hyphens, 1-63 characters, starting and ending with alphanumeric. Request body is limited to 1MB.

### List clusters

```bash
curl http://localhost:8080/api/v1/clusters
```

In file-based mode, returns:
```json
[{"name": "production", "displayName": "Production US-East", "connected": true, "edition": "enterprise", "default": true}]
```

In CRD-based mode, returns extended detail:
```json
[{
  "name": "workload-west",
  "displayName": "Workload US-West-2",
  "region": "us-west-2",
  "environment": "production",
  "connected": true,
  "edition": "enterprise",
  "default": false,
  "kubernetesVersion": "1.30.2",
  "ngfVersion": "1.6.2",
  "agentInstalled": true,
  "lastHeartbeat": "2024-01-15T10:30:00Z",
  "resourceCounts": {"gateways": 3, "httpRoutes": 12},
  "gpuCapacity": {"totalGPUs": 8, "allocatedGPUs": 6},
  "isLocal": false
}]
```

### Cluster summary

```bash
curl http://localhost:8080/api/v1/clusters/summary
```

Response:
```json
{
  "totalClusters": 3,
  "healthyClusters": 2,
  "totalGateways": 8,
  "totalRoutes": 24,
  "totalGPUs": 16
}
```

### Heartbeat

Sent by the agent heartbeat reporter every 30 seconds. Request body is limited to 64KB.

```bash
curl -X POST http://localhost:8080/api/v1/clusters/workload-west/heartbeat \
  -H 'Content-Type: application/json' \
  -d '{
    "kubernetesVersion": "1.30.2",
    "ngfVersion": "1.6.2",
    "resourceCounts": {"gateways": 3, "httpRoutes": 12, "inferencePools": 2},
    "gpuCapacity": {"totalGPUs": 8, "allocatedGPUs": 6, "gpuTypes": {"H100": 4, "A100": 4}}
  }'
```

### Agent install command

```bash
curl -X POST http://localhost:8080/api/v1/clusters/workload-west/install-agent
```

Response:
```json
{
  "helmCommand": "helm install ngf-console-agent charts/ngf-console-agent --namespace ngf-system --create-namespace --set cluster.name=\"workload-west\" --set hub.apiEndpoint=\"hub.example.com\" --set hub.otelEndpoint=\"hub.example.com:4317\"",
  "clusterName": "workload-west"
}
```

## Global Cross-Cluster Aggregation

Endpoints that query all registered clusters in parallel and return combined results. Each result is wrapped with `clusterName` and `clusterRegion` metadata. Per-cluster queries have a 10-second timeout.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/global/gateways` | List gateways from all clusters |
| GET | `/api/v1/global/routes` | List HTTP routes from all clusters |
| GET | `/api/v1/global/gpu-capacity` | Aggregated GPU capacity across all clusters |

### Global gateways

```bash
curl http://localhost:8080/api/v1/global/gateways
```

Response:
```json
[
  {
    "clusterName": "hub",
    "clusterRegion": "us-east-1",
    "gateway": {"name": "main-gw", "namespace": "default", "listeners": [...]}
  },
  {
    "clusterName": "workload-west",
    "clusterRegion": "us-west-2",
    "gateway": {"name": "inference-gw", "namespace": "inference", "listeners": [...]}
  }
]
```

Optional query parameter: `?namespace=default` to filter by namespace.

### Global GPU capacity

```bash
curl http://localhost:8080/api/v1/global/gpu-capacity
```

Response:
```json
{
  "totalGPUs": 16,
  "allocatedGPUs": 12,
  "clusters": [
    {"clusterName": "hub", "clusterRegion": "us-east-1", "totalGPUs": 8, "allocatedGPUs": 6, "gpuTypes": {"H100": 4, "A100": 4}},
    {"clusterName": "workload-west", "clusterRegion": "us-west-2", "totalGPUs": 8, "allocatedGPUs": 6, "gpuTypes": {"H100": 8}}
  ]
}
```

## Gateway Classes

| Method | Path | Description |
|--------|------|-------------|
| GET | `/gatewayclasses` | List all GatewayClasses |
| GET | `/gatewayclasses/{name}` | Get a GatewayClass by name |

## Gateways

| Method | Path | Description |
|--------|------|-------------|
| GET | `/gateways` | List all Gateways |
| POST | `/gateways` | Create a Gateway |
| GET | `/gateways/{namespace}/{name}` | Get a Gateway |
| PUT | `/gateways/{namespace}/{name}` | Update a Gateway |
| DELETE | `/gateways/{namespace}/{name}` | Delete a Gateway |
| POST | `/gateways/{namespace}/{name}/deploy` | Deploy a Gateway |

## GatewayBundles

CRD-backed gateway management via the operator.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/gatewaybundles` | List all GatewayBundles |
| POST | `/gatewaybundles` | Create a GatewayBundle |
| GET | `/gatewaybundles/{namespace}/{name}` | Get a GatewayBundle |
| PUT | `/gatewaybundles/{namespace}/{name}` | Update a GatewayBundle |
| DELETE | `/gatewaybundles/{namespace}/{name}` | Delete a GatewayBundle |
| GET | `/gatewaybundles/{namespace}/{name}/status` | Get operator reconciliation status |

## HTTP Routes

| Method | Path | Description |
|--------|------|-------------|
| GET | `/httproutes` | List all HTTPRoutes |
| POST | `/httproutes` | Create an HTTPRoute |
| GET | `/httproutes/{namespace}/{name}` | Get an HTTPRoute |
| PUT | `/httproutes/{namespace}/{name}` | Update an HTTPRoute |
| DELETE | `/httproutes/{namespace}/{name}` | Delete an HTTPRoute |
| POST | `/httproutes/{namespace}/{name}/simulate` | Simulate route matching |

## Other Route Types

gRPC, TLS, TCP, and UDP routes are defined but return 501 Not Implemented.

## Policies

| Method | Path | Description |
|--------|------|-------------|
| GET | `/policies/{type}` | List policies of a given type |
| POST | `/policies/{type}` | Create a policy |
| GET | `/policies/{type}/{name}` | Get a policy |
| PUT | `/policies/{type}/{name}` | Update a policy |
| DELETE | `/policies/{type}/{name}` | Delete a policy |
| GET | `/policies/{type}/conflicts` | Check for policy conflicts |

Supported types: `ratelimit`, `backendtls`, `clientsettings`

## Certificates

| Method | Path | Description |
|--------|------|-------------|
| GET | `/certificates` | List all TLS certificates |
| POST | `/certificates` | Create a certificate |
| GET | `/certificates/expiring` | List certificates expiring soon |
| GET | `/certificates/{name}` | Get a certificate |
| DELETE | `/certificates/{name}` | Delete a certificate |

## Prometheus Metrics

Requires `--prometheus-url` to be configured. Returns 503 otherwise.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/metrics/summary` | Aggregated RED metrics (rate, errors, duration) |
| GET | `/metrics/by-route` | Per-HTTPRoute RED metrics |
| GET | `/metrics/by-gateway` | Per-Gateway RED metrics |

## Logs

Requires ClickHouse to be configured.

| Method | Path | Description |
|--------|------|-------------|
| POST | `/logs/query` | Query logs from ClickHouse |
| GET | `/logs/topn` | Top-N log analytics |

## Topology

| Method | Path | Description |
|--------|------|-------------|
| GET | `/topology/full` | Full resource topology graph |
| GET | `/topology/by-gateway/{name}` | Gateway-scoped topology |

## Diagnostics

| Method | Path | Description |
|--------|------|-------------|
| POST | `/diagnostics/route-check` | Route diagnostic checklist |
| POST | `/diagnostics/trace` | Request trace waterfall |

## Inference Pools

| Method | Path | Description |
|--------|------|-------------|
| GET | `/inference/pools` | List all InferencePools |
| POST | `/inference/pools` | Create an InferencePool |
| GET | `/inference/pools/{name}` | Get an InferencePool |
| PUT | `/inference/pools/{name}` | Update an InferencePool |
| DELETE | `/inference/pools/{name}` | Delete an InferencePool |
| POST | `/inference/pools/{name}/deploy` | Deploy an InferencePool |

## Inference EPP & Autoscaling

| Method | Path | Description |
|--------|------|-------------|
| GET | `/inference/epp` | Get EPP configuration |
| PUT | `/inference/epp` | Update EPP configuration |
| GET | `/inference/autoscaling` | Get autoscaling configuration |
| PUT | `/inference/autoscaling` | Update autoscaling configuration |

## Inference Metrics

| Method | Path | Description |
|--------|------|-------------|
| GET | `/inference/metrics/summary` | Aggregated inference metrics |
| GET | `/inference/metrics/by-pool` | Per-pool metrics |
| GET | `/inference/metrics/pods?pool=X` | Per-pod GPU metrics |
| GET | `/inference/metrics/cost?pool=X` | Cost estimation |
| GET | `/inference/metrics/epp-decisions?pool=X` | EPP routing decisions |
| GET | `/inference/metrics/ttft-histogram/{pool}` | TTFT distribution |
| GET | `/inference/metrics/tps-throughput/{pool}` | Tokens/sec timeseries |
| GET | `/inference/metrics/queue-depth/{pool}` | Queue depth timeseries |
| GET | `/inference/metrics/gpu-util/{pool}` | GPU utilization timeseries |
| GET | `/inference/metrics/kv-cache/{pool}` | KV-cache utilization timeseries |

## Inference Diagnostics

| Method | Path | Description |
|--------|------|-------------|
| GET | `/inference/diagnostics/slow` | Slow inference analysis |
| POST | `/inference/diagnostics/replay` | Request replay |
| POST | `/inference/diagnostics/benchmark` | Model benchmarking |

## InferenceStacks

CRD-backed inference stack management via the operator.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/inference/stacks` | List all InferenceStacks |
| POST | `/inference/stacks` | Create an InferenceStack |
| GET | `/inference/stacks/{namespace}/{name}` | Get an InferenceStack |
| PUT | `/inference/stacks/{namespace}/{name}` | Update an InferenceStack |
| DELETE | `/inference/stacks/{namespace}/{name}` | Delete an InferenceStack |
| GET | `/inference/stacks/{namespace}/{name}/status` | Get operator reconciliation status |

## Coexistence

| Method | Path | Description |
|--------|------|-------------|
| GET | `/coexistence/overview` | KIC + NGF side-by-side resource view |
| GET | `/coexistence/migration-readiness` | Migration readiness percentage |

## F5 Distributed Cloud (XC)

| Method | Path | Description |
|--------|------|-------------|
| GET | `/xc/status` | XC connection status |
| POST | `/xc/publish` | Publish route/pool to XC |
| GET | `/xc/publish/{id}` | Get publish status |
| DELETE | `/xc/publish/{id}` | Delete a publish |
| GET | `/xc/metrics` | XC traffic metrics |

## Migration

| Method | Path | Description |
|--------|------|-------------|
| POST | `/migration/import` | Import NGINX config, Ingress YAML, or VirtualServer YAML |
| POST | `/migration/analysis` | Analyze imported config for Gateway API compatibility |
| POST | `/migration/generate` | Generate Gateway API resources from analysis |
| POST | `/migration/apply` | Apply generated resources to cluster (501 until cluster-backed) |
| POST | `/migration/validate` | Validate migrated resources (501 until cluster-backed) |

## Audit

| Method | Path | Description |
|--------|------|-------------|
| GET | `/audit` | List audit log entries (paginated) |
| GET | `/audit/diff/{id}` | Get before/after diff for an audit entry |

## Alerts

| Method | Path | Description |
|--------|------|-------------|
| GET | `/alerts` | List all alert rules |
| POST | `/alerts` | Create an alert rule |
| GET | `/alerts/firing` | List currently firing alerts |
| GET | `/alerts/{id}` | Get an alert rule |
| PUT | `/alerts/{id}` | Update an alert rule |
| DELETE | `/alerts/{id}` | Delete an alert rule |
| POST | `/alerts/{id}/toggle` | Enable/disable an alert rule |

## WebSocket Topics

| Endpoint | Interval | Description |
|----------|----------|-------------|
| `/api/v1/ws/inference/epp-decisions` | ~1s | Live EPP routing decisions |
| `/api/v1/ws/inference/gpu-metrics` | ~2s | Per-pod GPU utilization |
| `/api/v1/ws/inference/scaling-events` | ~15s | Autoscaling events |

Connect with any WebSocket client. In dev mode, topics stream mock data. The client reconnects with exponential backoff (1s base, 30s max) on disconnection.

## Request/Response Conventions

- All responses are JSON with `Content-Type: application/json`
- Error responses use `{"error": "message"}` format
- Internal error details are logged server-side with `slog`; clients receive generic error messages
- Request bodies are limited to 1MB (64KB for heartbeats)
- Cluster names in URLs are validated against RFC 1123 DNS subdomain rules
- The `X-Cluster` header can be used to specify the target cluster for legacy routes
