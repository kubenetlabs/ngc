# Configuration Reference

## API server flags

The API server accepts the following command-line flags:

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | `8080` | HTTP server listen port |
| `--kubeconfig` | auto | Path to kubeconfig file. Auto-discovers in order: in-cluster, `KUBECONFIG` env, `~/.kube/config` |
| `--clusters-config` | (none) | Path to multi-cluster YAML config file. Enables file-based multi-cluster mode |
| `--multicluster` | `false` | Enable CRD-based multi-cluster mode. Reads ManagedCluster CRDs from the hub cluster |
| `--multicluster-namespace` | `ngf-system` | Namespace where ManagedCluster CRDs and kubeconfig Secrets are stored |
| `--multicluster-default` | (auto) | Default cluster name for legacy routes. If not set, uses the first registered cluster |
| `--db-type` | `mock` | Inference metrics backend. `mock` uses synthetic data, `clickhouse` queries real ClickHouse tables |
| `--clickhouse-url` | `localhost:9000` | ClickHouse native protocol URL. Only used when `--db-type=clickhouse` |
| `--prometheus-url` | (none) | Prometheus server URL (e.g., `http://prometheus:9090`). Enables RED metrics endpoints. Without this, `/metrics/*` returns 503 |
| `--config-db` | `ngf-console.db` | Path to SQLite config database for alert rules, audit logs, and saved views |
| `--alert-webhooks` | (none) | Comma-separated webhook URLs for alert notifications |
| `--version` | | Print version and exit |

### Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CORS_ALLOWED_ORIGINS` | `*` | Comma-separated list of allowed CORS origins. In production, set to specific origins (e.g., `https://console.example.com`). When set to `*`, all origins are allowed (development only) |
| `KUBECONFIG` | (none) | Path to kubeconfig file. Used if `--kubeconfig` flag is not set |

### Examples

```bash
# Minimal (mock data, no cluster)
go run ./cmd/server

# With Kubernetes and ClickHouse
go run ./cmd/server \
  --kubeconfig ~/.kube/config \
  --db-type clickhouse \
  --clickhouse-url localhost:9000

# CRD-based multi-cluster
go run ./cmd/server \
  --multicluster \
  --multicluster-namespace ngf-system \
  --multicluster-default hub \
  --db-type clickhouse \
  --clickhouse-url clickhouse:9000 \
  --prometheus-url http://prometheus:9090

# File-based multi-cluster
go run ./cmd/server \
  --clusters-config /etc/ngf/clusters.yaml \
  --db-type clickhouse \
  --clickhouse-url clickhouse:9000

# Full production configuration
CORS_ALLOWED_ORIGINS=https://console.example.com \
go run ./cmd/server \
  --port 8080 \
  --multicluster \
  --multicluster-namespace ngf-system \
  --db-type clickhouse \
  --clickhouse-url clickhouse:9000 \
  --prometheus-url http://prometheus:9090 \
  --config-db /data/ngf-console.db \
  --alert-webhooks https://hooks.slack.com/xxx,https://webhook.site/yyy
```

## Hub Helm values

The hub Helm chart is configured via `deploy/helm/ngf-console/values.yaml`. Below is the complete reference.

### NGF settings

```yaml
ngf:
  edition: enterprise          # "enterprise" or "oss". Enterprise enables WAF, NginxProxy, SnippetsFilter features
  controllerNamespace: nginx-gateway   # Namespace where NGF controller is deployed
```

### Frontend

```yaml
frontend:
  replicas: 2
  image:
    repository: danny2guns/ngf-console-frontend
    tag: "0.1.0"
    pullPolicy: IfNotPresent
  resources:
    requests:
      cpu: 50m
      memory: 64Mi
    limits:
      cpu: 200m
      memory: 128Mi
```

### API server

```yaml
api:
  replicas: 2
  image:
    repository: danny2guns/ngf-console-api
    tag: "0.1.0"
    pullPolicy: IfNotPresent
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 512Mi
```

### Operator

```yaml
operator:
  replicas: 1
  leaderElection: true         # Enable leader election for HA
  reconcileInterval: 60s       # Drift detection interval
  image:
    repository: danny2guns/ngf-console-operator
    tag: "0.1.0"
    pullPolicy: IfNotPresent
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 256Mi
```

### Controller

```yaml
controller:
  replicas: 1
  image:
    repository: danny2guns/ngf-console-controller
    tag: "0.1.0"
    pullPolicy: IfNotPresent
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 256Mi
```

### Config database

```yaml
database:
  type: postgresql             # "postgresql" for production, "sqlite" for dev/single-node
  postgresql:
    host: ""                   # PostgreSQL host
    port: 5432
    database: ngf_console
    existingSecret: ""         # K8s Secret containing "username" and "password" keys
  sqlite:
    path: /data/ngf-console.db
    persistence:
      enabled: true
      size: 1Gi
```

### ClickHouse

```yaml
clickhouse:
  enabled: true                # Set false to use mock inference data or external ClickHouse
  replicas: 1
  image:
    repository: clickhouse/clickhouse-server
    tag: "24.1"
  persistence:
    size: 50Gi
  retention:
    rawLogs: 7d                # Raw log retention
    rollups1m: 90d             # 1-minute aggregation retention
    rollups1h: 365d            # 1-hour aggregation retention
```

### OpenTelemetry Collector

```yaml
otelCollector:
  enabled: true
  mode: deployment             # "deployment" or "daemonset"
  image:
    repository: otel/opentelemetry-collector-contrib
    tag: "0.96.0"
```

### Prometheus

```yaml
prometheus:
  url: http://prometheus:9090  # Prometheus server URL for RED metrics
```

### Grafana

```yaml
grafana:
  enabled: false
  url: http://grafana:3000
```

### Inference

```yaml
inference:
  enabled: true
  dcgmExporter:
    enabled: true
    image: nvcr.io/nvidia/k8s/dcgm-exporter:3.3.5-3.4.0-ubuntu22.04
  tritonMetrics:
    scrapeInterval: 5s
    metricsPath: /metrics
  scaling:
    backend: keda              # Autoscaling backend
    kedaNamespace: keda
  costEstimation:
    enabled: true
    gpuPricing:                # USD per GPU-hour
      A100: 3.67
      H100: 8.10
      L40S: 1.84
      T4: 0.53
```

### F5 Distributed Cloud (XC)

```yaml
xc:
  enabled: false
  tenantUrl: ""                # XC tenant URL (e.g., https://tenant.console.ves.volterra.io)
  apiTokenSecretRef: ""        # K8s Secret containing the XC API token
  defaultNamespace: default
  defaultWafPolicy: ""         # Default WAF policy name for published routes
```

### Authentication

```yaml
auth:
  type: kubernetes             # "kubernetes" (ServiceAccount) or "oidc"
  oidc:
    issuerUrl: ""              # OIDC issuer URL (e.g., https://accounts.google.com)
    clientId: ""
    clientSecretRef: ""        # K8s Secret containing the OIDC client secret
```

### Ingress

```yaml
ingress:
  enabled: true
  className: nginx-gateway-fabric
  hostname: ngf-console.example.com
  tls:
    enabled: true
    secretName: ngf-console-tls
```

### Service account

```yaml
serviceAccount:
  create: true
  name: ngf-console
  annotations: {}              # Add annotations for IAM roles, etc.
```

## Agent Helm values

The agent Helm chart (`charts/ngf-console-agent/values.yaml`) is installed on each workload cluster. It configures the operator, heartbeat reporter, and OTel forwarder.

### Cluster identity

```yaml
cluster:
  name: ""                     # Required: name of this cluster as registered on the hub
```

### Hub connection

```yaml
hub:
  apiEndpoint: ""              # Required: hub API URL (e.g., https://hub.example.com)
  otelEndpoint: ""             # Required: hub OTel Collector gRPC endpoint (e.g., hub.example.com:4317)
  authToken: ""                # Optional: authentication token for hub API
```

### Operator

The agent runs the same operator binary as the hub, reconciling CRDs locally on the workload cluster.

```yaml
operator:
  enabled: true
  replicas: 1
  leaderElection: true
  reconcileInterval: 60s
  image:
    repository: danny2guns/ngf-console-operator
    tag: "0.1.0"
    pullPolicy: IfNotPresent
  resources:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 500m
      memory: 256Mi
```

### Heartbeat reporter

Sends cluster health to the hub at a configurable interval.

```yaml
heartbeat:
  enabled: true
  image:
    repository: danny2guns/ngf-console-agent-heartbeat
    tag: "0.1.0"
    pullPolicy: IfNotPresent
  intervalSeconds: 30          # How often to send heartbeats to the hub
  resources:
    requests:
      cpu: 10m
      memory: 32Mi
    limits:
      cpu: 50m
      memory: 64Mi
```

### OTel Collector (forwarder)

Forwards telemetry from the workload cluster to the hub's OTel Collector, adding `cluster_name` as a resource attribute.

```yaml
otelCollector:
  enabled: true
  image:
    repository: otel/opentelemetry-collector-contrib
    tag: "0.96.0"
    pullPolicy: IfNotPresent
  resources:
    requests:
      cpu: 50m
      memory: 64Mi
    limits:
      cpu: 200m
      memory: 256Mi
```

### Service account

```yaml
serviceAccount:
  create: true
  name: ngf-console-agent
  annotations: {}
```

The chart creates two service accounts with separate RBAC:
- `ngf-console-agent` -- Operator service account with full read-write access to CRDs and child resources
- `ngf-console-agent-heartbeat` -- Heartbeat service account with read-only access (get/list on nodes, pods, CRDs)

### Security

All agent deployments run with hardened security contexts:
- `runAsNonRoot: true` (UID 65534)
- `readOnlyRootFilesystem: true`
- `allowPrivilegeEscalation: false`
- `capabilities: drop: [ALL]`
- `seccompProfile: type: RuntimeDefault`

Liveness and readiness probes are configured on all agent deployments.

## Multi-cluster configuration

### File-based (clusters.yaml)

The clusters config file defines which Kubernetes clusters the API server manages in file-based mode (`--clusters-config`).

```yaml
# clusters.yaml
clusters:
  - name: production           # Required: lowercase alphanumeric + hyphens, 1-63 chars
    displayName: "Prod US-East" # Optional: shown in UI cluster switcher
    kubeconfig: /path/to/kubeconfig.yaml  # Required: absolute path
    context: prod-admin        # Optional: kubeconfig context; defaults to current-context
    default: true              # Optional: at most one cluster can be default

  - name: staging
    displayName: "Staging EU-West"
    kubeconfig: /path/to/staging.yaml
```

When file-based multi-cluster mode is active:
- API routes are available at `/api/v1/clusters/{name}/...` (cluster-scoped)
- Legacy routes at `/api/v1/...` use the default cluster
- The frontend cluster switcher sets the `X-Cluster` header

### CRD-based (ManagedCluster)

In CRD-based mode (`--multicluster`), clusters are registered as `ManagedCluster` custom resources. See [Custom Resource Definitions](#managedcluster) below.

When CRD-based multi-cluster mode is active:
- The API reads ManagedCluster CRDs from `--multicluster-namespace`
- Each ManagedCluster references a kubeconfig Secret (or uses `isLocal: true` for the hub)
- The ClientPool builds and maintains K8s clients per cluster
- A background health checker runs every 30s with per-cluster circuit breakers
- Cluster management endpoints are available at `/api/v1/clusters`
- Global aggregation endpoints are available at `/api/v1/global/*`
- Agents on workload clusters send heartbeats to `/api/v1/clusters/{name}/heartbeat`

## Custom Resource Definitions

### ManagedCluster

Represents a registered Kubernetes cluster in the hub (`ngf-console.f5.com/v1alpha1`).

```yaml
apiVersion: ngf-console.f5.com/v1alpha1
kind: ManagedCluster
metadata:
  name: workload-west
  namespace: ngf-system
spec:
  displayName: "Workload US-West-2"     # Human-readable name
  region: us-west-2                      # Cloud region
  environment: production                # Environment label (production, staging, dev, etc.)
  isLocal: false                         # true for the hub cluster (uses in-cluster config)
  ngfEdition: enterprise                 # "enterprise" or "oss"
  kubeconfigSecretRef:
    name: workload-west-kubeconfig       # Secret containing kubeconfig data
  agentConfig:
    heartbeatInterval: 30s               # Agent heartbeat interval
    otelEndpoint: hub.example.com:4317   # Override OTel endpoint

status:
  phase: Ready                           # Ready, Pending, Degraded, Unreachable
  kubernetesVersion: "1.30.2"
  ngfVersion: "1.6.2"
  agentInstalled: true
  lastHeartbeat: "2024-01-15T10:30:00Z"
  resourceCounts:
    gateways: 3
    httpRoutes: 12
    inferencePools: 2
  gpuCapacity:
    totalGPUs: 8
    allocatedGPUs: 6
    gpuTypes:
      H100: 4
      A100: 4
```

### InferenceStack

Full spec for the InferenceStack CRD (`ngf-console.f5.com/v1alpha1`):

```yaml
apiVersion: ngf-console.f5.com/v1alpha1
kind: InferenceStack
metadata:
  name: llama-70b
  namespace: inference
spec:
  # Required fields
  modelName: meta-llama/Llama-3-70B-Instruct    # HuggingFace model ID
  servingBackend: vllm                            # "vllm", "triton", "tgi"

  # Pool configuration
  pool:
    gpuType: H100                                 # GPU type (A100, H100, L40S, T4)
    gpuCount: 4                                   # GPUs per pod
    replicas: 6                                   # Number of pods

  # Optional: EPP configuration
  epp:
    strategy: least-load                          # "round-robin", "least-load", "prefix-hash"
    weights:
      queueDepth: 0.4
      kvCacheUtilization: 0.3
      gpuUtilization: 0.3

  # Optional: Autoscaling
  autoscaling:
    enabled: true
    minReplicas: 2
    maxReplicas: 10
    metrics:
      - type: gpu-utilization
        target: 80
      - type: queue-depth
        target: 5

  # Optional: Gateway attachment
  gatewayRef:
    name: main-gateway
    namespace: default

  # Optional: DCGM GPU monitoring
  dcgm:
    enabled: true

  # Optional: F5 XC publishing
  distributedCloud:
    enabled: false
    tenantUrl: ""
```

The operator reconciles these child resources from an InferenceStack:

| Child | Kind | Name pattern | Description |
|-------|------|-------------|-------------|
| InferencePool | `inference.networking.x-k8s.io/v1alpha2` | `{name}-pool` | Gateway Inference Extension pool |
| EPP Config | `ConfigMap` | `{name}-epp-config` | Endpoint Picker configuration |
| Autoscaler | `ScaledObject` (KEDA) | `{name}-scaler` | KEDA autoscaling rules |
| HTTPRoute | `gateway.networking.k8s.io/v1` | `{name}-route` | Gateway API route attachment |
| DCGM Exporter | `DaemonSet` | `{name}-dcgm` | NVIDIA GPU metrics exporter |

### GatewayBundle

Full spec for the GatewayBundle CRD (`ngf-console.f5.com/v1alpha1`):

```yaml
apiVersion: ngf-console.f5.com/v1alpha1
kind: GatewayBundle
metadata:
  name: main-gateway
  namespace: default
spec:
  # Required
  gatewayClassName: nginx

  # Listeners
  listeners:
    - name: http
      port: 80
      protocol: HTTP
    - name: https
      port: 443
      protocol: HTTPS
      tls:
        mode: Terminate
        certificateRefs:
          - name: tls-secret

  # Optional: NginxProxy (Enterprise)
  nginxProxy:
    enabled: false

  # Optional: WAF (Enterprise)
  waf:
    enabled: false

  # Optional: SnippetsFilter (Enterprise)
  snippetsFilter:
    enabled: false

  # Optional: TLS
  tls:
    secretRefs:
      - name: tls-secret
        namespace: default
```

The operator reconciles these child resources from a GatewayBundle:

| Child | Kind | Description |
|-------|------|-------------|
| Gateway | `gateway.networking.k8s.io/v1` | The Gateway API gateway |
| NginxProxy | Enterprise CRD | NGINX proxy configuration (Enterprise only) |
| WAFPolicy | Enterprise CRD | Web Application Firewall policy (Enterprise only) |
| SnippetsFilter | Enterprise CRD | NGINX config snippets (Enterprise only) |
| TLS Secrets | `Secret` | TLS certificate secrets |

### Status fields

Both InferenceStack and GatewayBundle CRDs report status with:

```yaml
status:
  phase: Ready                     # Ready, Pending, Degraded, Error
  observedSpecHash: "a1b2c3..."    # SHA-256 of current spec for drift detection
  lastReconciledAt: "2024-01-15T10:30:00Z"
  children:                        # Per-child status
    - kind: InferencePool
      name: llama-70b-pool
      ready: true
    - kind: ConfigMap
      name: llama-70b-epp-config
      ready: true
      message: "configured"
  conditions:                      # Standard K8s conditions
    - type: Ready
      status: "True"
      reason: AllChildrenReady
      message: "All child resources are ready"
    - type: Reconciled
      status: "True"
      reason: ReconcileSucceeded
```

## ClickHouse schema

Tables populated by the OTel Collector. All tables include `cluster_name LowCardinality(String)` for multi-cluster filtering:

| Table | Engine | Description |
|-------|--------|-------------|
| `ngf_access_logs` | MergeTree | HTTP access logs with request/response details |
| `ngf_inference_logs` | MergeTree | Inference request logs (model, tokens, latency) |
| `ngf_inference_pools` | MergeTree | Pool metadata (name, model, GPU type, replicas, status) |
| `ngf_epp_decisions` | MergeTree | EPP routing decisions with latency and strategy |
| `ngf_pod_metrics` | MergeTree | Per-pod GPU utilization, memory, temperature, queue depth |
| `ngf_metrics_1m` | AggregatingMergeTree | 1-minute aggregated RED metrics |
| `ngf_inference_metrics_1m` | AggregatingMergeTree | 1-minute aggregated inference metrics |

Materialized views use `-State` combinators (`countState()`, `avgState()`, `quantileState()`, `sumState()`, `maxState()`) for correct aggregation across merges.

See `deploy/docker-compose/clickhouse/init.sql` for the complete schema and demo data.

## Alert configuration

Alert rules are stored in the SQLite/PostgreSQL config database and managed via the API.

### Creating an alert rule

```bash
curl -X POST http://localhost:8080/api/v1/alerts \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "High GPU Utilization",
    "resource": "inference-pool",
    "metric": "gpu-utilization",
    "operator": ">",
    "threshold": 90,
    "severity": "warning",
    "duration": "5m"
  }'
```

### Webhook notifications

Alert notifications are sent to webhook URLs configured via `--alert-webhooks`:

```bash
go run ./cmd/server --alert-webhooks https://hooks.slack.com/services/xxx
```

The evaluation engine checks alert rules periodically and fires webhooks when thresholds are breached.

## WebSocket topics

The API server provides three WebSocket topics for real-time streaming:

| Topic | Endpoint | Interval | Description |
|-------|----------|----------|-------------|
| EPP Decisions | `/api/v1/ws/inference/epp-decisions` | ~1s | Live EPP routing decisions |
| GPU Metrics | `/api/v1/ws/inference/gpu-metrics` | ~2s | Per-pod GPU utilization |
| Scaling Events | `/api/v1/ws/inference/scaling-events` | ~15s | Autoscaling events |

Connect with any WebSocket client:

```bash
wscat -c ws://localhost:8080/api/v1/ws/inference/epp-decisions
```

In development, these stream mock data. In production with ClickHouse, they stream real telemetry data.

The WebSocket client reconnects with exponential backoff (1s base, 30s max, with random jitter) on disconnection.
