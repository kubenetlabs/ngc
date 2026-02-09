# NGINX Gateway Fabric UI — Technical Specification

## Project: NGF Console

### Version: 0.1.0-alpha
### Author: Dan / F5 Product Incubation
### Date: February 2026

---

## 1. Executive Summary

NGF Console is a web-based management platform for NGINX Gateway Fabric — purpose-built for the AI inference era. It is the first gateway management UI with native support for Kubernetes Gateway Inference Extensions, providing GPU-aware observability, InferencePool lifecycle management, Endpoint Picker (EPP) decision visualization, and intelligent autoscaling configuration for LLM serving infrastructure.

Beyond inference, NGF Console provides comprehensive traffic management, observability, policy configuration, certificate lifecycle management, troubleshooting workflows, NGINX Ingress Controller migration tooling, and automated F5 Distributed Cloud integration. It supports both OSS and Enterprise NGF editions, with enterprise features gracefully degraded (greyed out) when running against OSS.

The project is containerized, installable via Helm, kubectl manifests, and docker-compose, and designed for production Kubernetes deployments.

---

## 2. Architecture Overview

### 2.1 High-Level Components

```
┌──────────────────────────────────────────────────────────────┐
│                       NGF Console UI                          │
│                (React + TypeScript + Tailwind)                 │
├──────────────────────────────────────────────────────────────┤
│                     API Gateway / BFF                          │
│                (Go REST + WebSocket + gRPC)                    │
├──────────┬──────────┬───────────┬──────────┬────────┬────────┤
│ K8s API  │ClickHouse│ Postgres/ │Prometheus │ F5 XC  │ Triton │
│ Client   │  Client  │ SQLite    │ + DCGM   │ API    │/metrics│
│(Gateway  │          │           │  Client   │ Client │ Client │
│ API +    │          │           │           │        │        │
│ Inference│          │           │           │        │        │
│ Ext CRDs)│          │           │           │        │        │
└──────────┴──────────┴───────────┴──────────┴────────┴────────┘
```

### 2.2 Component Inventory

| Component | Technology | Purpose |
|-----------|-----------|---------|
| **Frontend** | React 18, TypeScript, Tailwind CSS, Vite | SPA UI |
| **Backend API** | Go (preferred) | REST API + WebSocket for real-time |
| **Configuration DB** | PostgreSQL (production) / SQLite (single-node) | User prefs, audit log, alert rules, saved views |
| **Analytics DB** | ClickHouse | Access log + inference telemetry analytics |
| **Telemetry Pipeline** | OpenTelemetry Collector | NGINX → OTel → ClickHouse; Triton → OTel → ClickHouse |
| **K8s Controller** | Go controller-runtime | Watches Gateway API + Inference Extension resources, syncs XC |
| **Inference Metrics Collector** | Go sidecar/agent | Scrapes Triton /metrics, EPP state, DCGM; publishes to OTel + Custom Metrics API |
| **XC Sync Controller** | Go (or integrated into above) | Reconciles HTTPRoutes → F5 XC HTTP LBs |
| **Ingress Migration Tool** | Go CLI + UI integration | Converts NGINX Ingress Controller configs → Gateway API |

### 2.3 Container Images

```
ngf-console-frontend:0.1.0     # Nginx-served React SPA
ngf-console-api:0.1.0          # Backend API server
ngf-console-controller:0.1.0   # K8s controller + XC sync
ngf-console-migration:0.1.0    # Ingress migration tool (init container or Job)
```

### 2.4 Installation Methods

**Helm Chart** (primary):
```bash
helm install ngf-console oci://registry.f5.com/ngf-console \
  --namespace ngf-system \
  --set database.type=postgresql \
  --set clickhouse.enabled=true \
  --set xc.enabled=true \
  --set xc.tenantUrl=https://tenant.console.ves.volterra.io \
  --set xc.apiTokenSecretRef=xc-api-token \
  --set ngf.edition=enterprise  # or "oss"
```

**kubectl manifests**:
```bash
kubectl apply -f https://raw.githubusercontent.com/f5/ngf-console/main/deploy/manifests/install.yaml
```

**docker-compose** (dev/demo):
```bash
docker-compose -f deploy/docker-compose/docker-compose.yaml up -d
```

---

## 3. Feature Specifications

### 3.1 Edition Detection & Feature Gating

The UI must auto-detect whether the connected NGF instance is OSS or Enterprise.

**Detection Method:**
1. Query the GatewayClass for `parametersRef` pointing to NginxProxy with enterprise fields
2. Check for presence of enterprise CRDs: `APPolicies`, `SnippetsFilters`, etc.
3. Check NGF controller deployment labels/annotations for edition info

**Behavior:**
- Enterprise features show as greyed-out cards/buttons with a tooltip: "Requires NGINX Gateway Fabric Enterprise. Contact F5 for a trial."
- Link to F5 trial page from greyed-out features
- No enterprise API calls made when running against OSS

**Enterprise-only features:**
- App Protect WAF policy management
- SnippetsFilter builder
- Bot defense configuration
- Advanced NGINX tuning via NginxProxy
- F5 Distributed Cloud integration

---

### 3.2 Gateway Creation Workflow

A multi-step wizard for creating Gateway resources.

**Step 1 — GatewayClass Selection**
- List available GatewayClasses with feature comparison table
- Show parametersRef availability
- Highlight enterprise capabilities with edition badge
- API: `GET /api/v1/gatewayclasses`

**Step 2 — Gateway Configuration**
- Form fields: name, namespace, labels, annotations
- Listener builder (dynamic list):
  - Protocol selector: HTTP, HTTPS, TLS, TCP, UDP
  - Port number with conflict detection (real-time validation against cluster)
  - Hostname (with wildcard support and validation)
  - TLS configuration: mode (Terminate, Passthrough), certificateRef selector
  - Allowed routes: namespaces (Same, All, Selector), kinds
- Infrastructure settings via parametersRef (Enterprise):
  - Replica count, resource requests/limits
  - NGINX worker tuning
- API: `POST /api/v1/gateways`

**Step 3 — WAF Policy (Enterprise)**
- APPolicy selector or inline creation
- Threat signature browser with search/filter
- Violation rating threshold slider with plain-English labels
- Bot defense profile selector
- Log-only / Enforce toggle
- API: `POST /api/v1/policies/waf`

**Step 4 — Advanced Configuration (Enterprise)**
- NginxProxy resource builder
- SnippetsFilter builder with:
  - Context selector: http, server, location
  - Syntax highlighting editor (Monaco)
  - NGINX config validation (client-side + server-side)
  - "What this does" documentation panel
- API: `POST /api/v1/snippets`

**Step 5 — Review & Deploy**
- Complete YAML preview (read-only, copyable)
- Dry-run validation: `kubectl apply --dry-run=server`
- Diff view for modifications
- Deploy button with progress indicator
- Rollback capability (stores previous config in DB)
- API: `POST /api/v1/gateways/deploy`

**Post-Deploy — Topology View**
- Interactive DAG: Gateway → Listeners → HTTPRoutes → BackendRefs → Services → Pods
- Status badges on each node (Accepted, Programmed, etc.)
- Click-to-drill on any node
- Real-time status updates via WebSocket
- API: `GET /api/v1/topology?gateway=<name>`

---

### 3.3 Traffic Management

#### 3.3.1 Visual Route Configuration

**HTTPRoute Builder:**
- Form-based with live YAML preview panel
- Fields: parentRefs (gateway selector), hostnames, rules
- Rule builder:
  - Match conditions: path (Exact, PathPrefix, RegularExpression), headers, query params, method
  - Backend refs with weight sliders
  - Filters: RequestHeaderModifier, ResponseHeaderModifier, URLRewrite, RequestRedirect, RequestMirror, ExtensionRef
- Validation: real-time against cluster state
- API: `POST /api/v1/httproutes`

**GRPCRoute Builder:**
- Similar to HTTPRoute but with gRPC-specific matching (service, method)
- API: `POST /api/v1/grpcroutes`

**TLSRoute / TCPRoute / UDPRoute Builders:**
- Simplified forms for L4 routing
- API: `POST /api/v1/{tls,tcp,udp}routes`

#### 3.3.2 Traffic Splitting Visualization

- Animated traffic flow diagram showing request distribution across backends
- Weight adjustment via drag sliders with real-time preview
- Historical traffic distribution chart (from ClickHouse)
- Canary deployment helper: auto-configure progressive weight shifts with timers
- API: `GET /api/v1/traffic-split?route=<name>`

#### 3.3.3 Header/Query Parameter Matching Rules Builder

- Visual match condition builder (AND/OR logic tree)
- Test request simulator:
  - Input: method, path, headers, query params
  - Output: which route/rule would match, which backend would receive the request
  - Highlighting of the matching conditions
- API: `POST /api/v1/routes/simulate`

#### 3.3.4 Gateway & GatewayClass Lifecycle

- List view with status, listener count, route count, age
- Inline editing of existing Gateways
- Scale operations (update replica count)
- Delete with dependency check ("This gateway has 5 routes attached. Delete anyway?")
- GatewayClass management (view, not create — those are cluster-scoped and admin-managed)

---

### 3.4 Observability Dashboard

#### 3.4.1 Real-Time Metrics (Prometheus)

**Sources:** Prometheus endpoint exposed by NGF (existing)

**Dashboard Panels:**
- Request rate (RPS) — global, per gateway, per route, per backend
- Error rate (4xx, 5xx) — with breakdown by status code
- Latency (p50, p95, p99) — histograms per route
- Active connections — per gateway/listener
- Upstream health — per backend service
- SSL handshake rate and errors
- Connection pool utilization

**Implementation:**
- Prometheus client in backend API queries Prometheus directly
- WebSocket push for real-time updates (1s interval configurable)
- Time range selector: 5m, 15m, 1h, 6h, 24h, custom
- Auto-refresh toggle

**Grafana Integration:**
- "Open in Grafana" button per panel (configurable Grafana URL)
- Prometheus endpoint remains the source of truth for Grafana users
- No duplication of data — both UI and Grafana read from same Prometheus

#### 3.4.2 Log Analytics (ClickHouse)

**Data Model:**

```sql
CREATE TABLE ngf_access_logs (
    timestamp DateTime64(3),
    gateway String,
    listener String,
    route String,
    namespace String,
    method LowCardinality(String),
    path String,
    status UInt16,
    latency_ms Float64,
    upstream_latency_ms Float64,
    request_size UInt64,
    response_size UInt64,
    upstream_name String,
    upstream_addr String,
    client_ip String,
    user_agent String,
    request_id String,
    trace_id String,
    tls_version LowCardinality(String),
    tls_cipher LowCardinality(String),
    waf_action LowCardinality(String),    -- Enterprise: pass/block/monitor
    waf_violation_rating Float32,          -- Enterprise
    waf_signatures Array(String),          -- Enterprise
    bot_classification LowCardinality(String), -- Enterprise
    xc_edge_latency_ms Float64            -- When XC-published
) ENGINE = MergeTree()
PARTITION BY toYYYYMMDD(timestamp)
ORDER BY (gateway, route, timestamp)
TTL timestamp + INTERVAL 7 DAY;
```

**Materialized Views for Rollups:**

```sql
-- 1-minute rollups
CREATE MATERIALIZED VIEW ngf_metrics_1m
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMMDD(window_start)
ORDER BY (gateway, route, status_class, window_start)
TTL window_start + INTERVAL 90 DAY
AS SELECT
    toStartOfMinute(timestamp) AS window_start,
    gateway, route,
    multiIf(status < 200, '1xx', status < 300, '2xx', status < 400, '3xx',
            status < 500, '4xx', '5xx') AS status_class,
    count() AS request_count,
    avg(latency_ms) AS avg_latency,
    quantile(0.95)(latency_ms) AS p95_latency,
    quantile(0.99)(latency_ms) AS p99_latency,
    sum(request_size) AS total_request_bytes,
    sum(response_size) AS total_response_bytes
FROM ngf_access_logs
GROUP BY window_start, gateway, route, status_class;

-- 1-hour rollups (same structure, longer TTL)
-- 1-day rollups (same structure, 1 year TTL)
```

**UI Panels Powered by ClickHouse:**
- Access log explorer with full-text search and filters
- Top-N: busiest routes, highest error routes, slowest backends, top user agents
- Request size distribution
- Geographic traffic distribution (via client IP → GeoIP)
- WAF event timeline (Enterprise)
- Custom SQL query editor for power users

**Telemetry Pipeline:**

```
NGINX → OTel Sidecar/DaemonSet → OTel Collector → ClickHouse
                                       ↓
                               Prometheus (metrics)
```

OTel Collector config (deployed as part of Helm chart):
- Receivers: otlp (from NGINX OTel module)
- Processors: batch, resource attributes
- Exporters: clickhouse (logs), prometheus (metrics)

---

### 3.5 Policy Management

#### 3.5.1 Policy Types

| Policy | Scope | Edition |
|--------|-------|---------|
| RateLimitPolicy | Route, Gateway | OSS + Enterprise |
| BackendTLSPolicy | BackendRef | OSS + Enterprise |
| ClientSettingsPolicy | Route, Gateway | OSS + Enterprise |
| ObservabilityPolicy | Route, Gateway | OSS + Enterprise |
| APPolicy (WAF) | Route, Gateway | Enterprise |
| SnippetsFilter | Route | Enterprise |

#### 3.5.2 Policy Builder UI

- Template gallery: pre-built policies for common scenarios
  - "Rate limit API to 100 req/min"
  - "Enable mTLS to backend"
  - "Block SQL injection"
  - "Add CORS headers"
- Visual policy builder: form-based with YAML preview
- Policy attachment: drag-and-drop onto routes/gateways in topology view
- Conflict detection: show which policies are winning at each attachment point
  - Gateway API hierarchy: GatewayClass → Gateway → Route
  - Visual indicators: green (active), yellow (overridden), red (conflicting)
- API: `POST /api/v1/policies/{type}`

#### 3.5.3 WAF Management (Enterprise)

- Threat signature browser with categories, CVE references
- Custom rule builder with ModSecurity-compatible syntax
- WAF event dashboard: blocked requests timeline, top violations, attack source map
- Policy testing: replay captured requests against policy to preview enforcement
- API: `GET/POST /api/v1/policies/waf`

---

### 3.6 Certificate & TLS Management

**Inventory View:**
- All TLS Secrets in monitored namespaces
- cert-manager Certificate resources (if installed)
- Fields: common name, SANs, issuer, expiry date, key type, attached-to
- Expiry status: green (>30d), yellow (7-30d), red (<7d), expired

**Certificate Lifecycle:**
- Upload new cert/key pair
- Generate CSR (for enterprise CA integration)
- Renew: trigger cert-manager renewal or manual replacement
- cert-to-listener mapping visualization

**Alerting:**
- Configurable expiry threshold alerts (email, Slack webhook, PagerDuty)
- Alert rules stored in configuration DB

**mTLS Configuration:**
- Backend TLS policy builder
- CA bundle management
- Client certificate rotation workflow

**API:** `GET/POST /api/v1/certificates`

---

### 3.7 Troubleshooting & Debugging

#### 3.7.1 Diagnostic Wizard — "Why Isn't My Route Working?"

Interactive wizard that checks (in order):
1. **Route status**: Is the HTTPRoute accepted? Are conditions met?
2. **Gateway attachment**: Is the route attached to a programmed gateway?
3. **Listener match**: Does the route's parentRef match a listener? (hostname, port, protocol, allowed namespaces)
4. **Route precedence**: Is another route taking priority? (longest path match, creation timestamp)
5. **Backend health**: Are backend services healthy? Endpoints present?
6. **Policy blocks**: Is a WAF, rate limit, or other policy blocking requests?
7. **TLS issues**: Certificate valid? TLS version mismatch?
8. **Network connectivity**: Can the gateway pod reach the backend? (optional kubectl exec probe)

Output: step-by-step diagnostic report with pass/fail/warning per check and remediation suggestions.

API: `POST /api/v1/diagnostics/route-check`

#### 3.7.2 Request Tracing

- Input: request method, path, headers (or paste a curl command)
- Trace the request through: gateway → listener → route matching → policy evaluation → upstream selection → backend response
- Correlate with access logs in ClickHouse via request_id or trace_id
- Display as waterfall timeline

API: `POST /api/v1/diagnostics/trace`

#### 3.7.3 Configuration Diff Viewer

- Compare current vs. previous configuration for any resource
- Side-by-side diff with syntax highlighting
- Historical versions stored in configuration DB (audit log)
- "Revert to this version" button

API: `GET /api/v1/audit/diff?resource=<kind>/<namespace>/<name>&from=<version>&to=<version>`

#### 3.7.4 Event Stream

- Real-time Kubernetes events for NGF-related resources
- Filterable by: gateway, route, namespace, event type, severity
- Searchable full-text
- WebSocket-based streaming

API: `WS /api/v1/events/stream`

---

### 3.8 F5 Distributed Cloud Integration

#### 3.8.1 XC Auto-Registration Controller

**CRD Definition:**

```yaml
apiVersion: ngf-console.f5.com/v1alpha1
kind: DistributedCloudPublish
metadata:
  name: my-app-xc
  namespace: default
spec:
  httpRouteRef:
    name: my-app-route
    namespace: default
  distributedCloud:
    tenant: my-tenant
    namespace: production
    wafPolicy: default-waf        # XC App Firewall name
    botDefense:
      enabled: true
      profile: standard
    ddosProtection:
      enabled: true
      profile: standard
    publicHostname: app.example.com
    tls:
      mode: managed               # managed (XC Let's Encrypt) | bringYourOwn
      certificateRef:              # only if mode=bringYourOwn
        name: my-tls-cert
        namespace: default
    originPool:
      autoDetect: true             # auto-detect NGF external IP/LB
      overrideAddress: ""          # manual override
      port: 443
      useTLS: true
      tlsConfig:
        sni: internal.example.com
        skipVerify: false
status:
  state: Published                 # Pending | Published | Error | Degraded
  xcHttpLoadBalancerName: "ves-io-http-lb-my-app-xc"
  xcOriginPoolName: "ves-io-op-my-app-xc"
  publicEndpoint: "https://app.example.com"
  lastSyncTime: "2026-02-09T12:00:00Z"
  conditions:
    - type: Synced
      status: "True"
      message: "XC HTTP Load Balancer is active"
```

**Controller Reconciliation Loop:**

```
Watch DistributedCloudPublish CRDs
  │
  ├─ On Create/Update:
  │   ├─ Resolve HTTPRoute → get hostnames, paths, backends
  │   ├─ Resolve NGF Gateway external IP/LB address
  │   ├─ Call XC API: Create/Update Origin Pool
  │   │   └─ Origin server = NGF external address
  │   ├─ Call XC API: Create/Update HTTP Load Balancer
  │   │   ├─ Attach WAF policy
  │   │   ├─ Attach bot defense
  │   │   ├─ Configure TLS
  │   │   └─ Set routes to match HTTPRoute paths
  │   ├─ Update CRD status with XC object references
  │   └─ Store sync metadata in annotations
  │
  ├─ On Delete:
  │   ├─ Call XC API: Delete HTTP Load Balancer
  │   ├─ Call XC API: Delete Origin Pool
  │   └─ Clean up annotations
  │
  └─ Periodic Reconciliation (every 60s):
      ├─ Check XC object health
      ├─ Update CRD status
      └─ Detect drift (XC config changed outside controller)
```

**XC API Integration:**
- Authentication: API token stored in Kubernetes Secret
- Base URL: `https://<tenant>.console.ves.volterra.io/api`
- Key endpoints:
  - `POST /config/namespaces/<ns>/http_loadbalancers` — create HTTP LB
  - `POST /config/namespaces/<ns>/origin_pools` — create origin pool
  - `POST /config/namespaces/<ns>/app_firewalls` — reference WAF policy
  - `GET /config/namespaces/<ns>/http_loadbalancers/<name>` — check status

#### 3.8.2 UI for XC Publishing

In the HTTPRoute creation/edit form:

```
┌─────────────────────────────────────────────────────┐
│ ☑ Publish via F5 Distributed Cloud                  │
│                                                      │
│   WAF Profile:       [Default ▼]                    │
│   Bot Defense:       [● Enabled  ○ Disabled]        │
│   DDoS Protection:   [● Standard ○ Advanced]        │
│   Public Hostname:   [app.example.com        ]      │
│   TLS:               [● Managed by XC  ○ BYOC]     │
│   XC Namespace:      [production ▼]                 │
│                                                      │
│   Status: ● Published — https://app.example.com     │
└─────────────────────────────────────────────────────┘
```

**Dashboard Integration:**
- Unified metrics: NGF (internal) + XC (edge) in single view
- WAF event correlation: XC block events shown on route dashboard
- End-to-end latency breakdown: Client → XC → NGF → Backend
- Security posture score per route: WAF ✓, Bot ✓, DDoS ✓, TLS ✓

---

### 3.9 NGINX Ingress Controller Migration Tool

#### 3.9.1 Purpose

Provide an on-ramp for existing F5/NGINX customers running NGINX Ingress Controller (the commercial KIC product) to migrate to NGINX Gateway Fabric.

#### 3.9.2 Supported Input Formats

- Kubernetes Ingress resources (with NGINX-specific annotations)
- NGINX Ingress Controller ConfigMap (global settings)
- VirtualServer / VirtualServerRoute CRDs (NGINX KIC custom resources)
- TransportServer CRDs
- Policy CRDs (rate limiting, JWT, OIDC, WAF, etc.)
- Exported NGINX config files (nginx.conf)

#### 3.9.3 Migration Workflow

**Step 1 — Import**
- Upload YAML files or paste content
- Or: connect to source cluster and auto-discover Ingress/VirtualServer resources
- Or: import from a Git repository URL
- API: `POST /api/v1/migration/import`

**Step 2 — Analysis**
- Parse all resources and build a dependency graph
- Identify:
  - Direct mappings: Ingress path → HTTPRoute
  - Annotation translations: `nginx.org/proxy-read-timeout` → BackendSettingsPolicy
  - Unsupported features: features with no Gateway API equivalent
  - Enterprise requirements: features needing enterprise NGF
- Generate a migration report with:
  - Confidence score per resource (high/medium/low)
  - Warnings and manual review items
  - Feature gap analysis
- API: `GET /api/v1/migration/{id}/analysis`

**Step 3 — Review & Edit**
- Side-by-side view: source Ingress/VS ↔ generated Gateway API resources
- Inline editing of generated resources
- Toggle individual migrations on/off
- Resolve warnings interactively

**Step 4 — Apply**
- Generate complete YAML manifests
- Option to apply directly to cluster or download for review
- Phased migration support: migrate one route at a time, validate, proceed
- Rollback plan generation

**Annotation Translation Table (partial):**

| NGINX KIC Annotation | Gateway API Equivalent |
|---|---|
| `nginx.org/proxy-connect-timeout` | ClientSettingsPolicy or BackendTLSPolicy |
| `nginx.org/proxy-read-timeout` | ClientSettingsPolicy |
| `nginx.org/server-snippets` | SnippetsFilter (Enterprise) |
| `nginx.org/location-snippets` | SnippetsFilter (Enterprise) |
| `nginx.org/lb-method` | HTTPRoute backendRef weight + custom policy |
| `nginx.org/rate-limiting` | RateLimitPolicy |
| `nginx.org/jwt-*` | External auth integration |
| `nginx.org/waf-*` | APPolicy (Enterprise) |

**VirtualServer CRD Translation:**

| VS Field | Gateway API Equivalent |
|---|---|
| `upstreams` | Service backendRefs |
| `routes[].path` | HTTPRoute rules[].matches[].path |
| `routes[].splits` | HTTPRoute backendRefs with weights |
| `routes[].matches.headers` | HTTPRoute rules[].matches[].headers |
| `routes[].action.redirect` | HTTPRouteFilter (RequestRedirect) |
| `routes[].action.proxy.rewritePath` | HTTPRouteFilter (URLRewrite) |
| `routes[].errorPages` | No direct equivalent (SnippetsFilter) |
| `policies[].rateLimit` | RateLimitPolicy |
| `tls.secret` | Gateway listener TLS certificateRef |

#### 3.9.4 CLI Mode

The migration tool also runs as a standalone CLI for CI/CD integration:

```bash
# Scan source cluster
ngf-migrate scan --kubeconfig=source.yaml --namespace=production

# Generate migration plan
ngf-migrate plan --input=scan-output.yaml --output=migration-plan/

# Apply migration
ngf-migrate apply --plan=migration-plan/ --kubeconfig=target.yaml --dry-run

# Validate migration
ngf-migrate validate --kubeconfig=target.yaml --namespace=production
```

---

### 3.10 Inference Management (Gateway Inference Extensions)

This is the centerpiece capability of NGF Console — the first gateway management UI built for AI/ML inference workloads.

#### 3.10.1 Overview

NGF Console provides native support for the Kubernetes Gateway Inference Extensions, which introduce the InferencePool resource as a GPU/LLM-aware replacement for standard Kubernetes Services. Instead of round-robin load balancing, an Endpoint Picker (EPP) routes inference requests based on real-time backend telemetry from model serving infrastructure (NVIDIA Triton, vLLM, TGI).

**Key Concepts:**
- **InferencePool**: A CRD defining a pool of model-serving pods (e.g., Triton) with GPU-aware routing
- **Endpoint Picker (EPP)**: A component that scrapes backend metrics and makes LLM-aware routing decisions
- **GPU-Aware Metrics**: Queue depth, KV-cache utilization, prefix cache state, GPU memory/utilization (DCGM)

#### 3.10.2 InferencePool Creation Wizard

A guided workflow oriented toward ML platform engineers who understand GPUs and models but may not be Kubernetes networking experts.

**Step 1 — Model Backend Selection**
- Select serving backend: NVIDIA Triton (primary), vLLM, TGI (future)
- Model name/identifier (e.g., "llama-3-70b", "mixtral-8x7b")
- Backend version and container image
- GPU type and count per pod (A100, H100, L40S, etc.)
- API: `GET /api/v1/inference/backends`

**Step 2 — Pool Configuration**
- InferencePool name, namespace, labels
- Pod selector (match existing Triton deployment or create new)
- GPU affinity / node selector / tolerations
- Min/max replicas
- Resource requests: GPU count, GPU memory, CPU, RAM per pod
- NVIDIA MPS (Multi-Process Service) configuration if sharing GPUs
- API: `POST /api/v1/inference/pools`

**Step 3 — EPP Configuration**
- Metric scraping interval (default: 5s)
- Triton /metrics endpoint path
- Routing strategy selector with visual explanation:
  - **Least Queue Depth**: Route to pod with shortest request queue (default)
  - **Best KV-Cache Available**: Route to pod with most available KV-cache memory
  - **Prefix Cache Affinity**: Route similar prompts to same pod for cache hits
  - **Composite**: Weighted combination of all signals (advanced)
- Strategy weights (for composite): queue_depth=0.4, kv_cache=0.3, prefix_cache=0.3
- DCGM integration toggle (enables GPU utilization safety-net metrics)
- API: `POST /api/v1/inference/pools/{ns}/{name}/epp`

**Step 4 — Autoscaling Policy**
- Scaling backend selector: HPA (native), KEDA (recommended for inference)
- Scaling triggers with visual threshold configuration:
  - Queue depth threshold (e.g., "Scale up when avg queue > 5 for 60s")
  - KV-cache utilization threshold (e.g., "Scale up when avg KV-cache > 80%")
  - GPU utilization threshold via DCGM (e.g., "Safety net: scale up when GPU util > 90% for 120s")
  - Request rate / tokens-per-second
- Cool-down period configuration
- Min/max replica bounds
- Preview: "What-if" simulator showing how current traffic would trigger scaling under these rules
- Generates: HPA resource or KEDA ScaledObject + Prometheus Adapter config or KEDA trigger
- API: `POST /api/v1/inference/pools/{ns}/{name}/autoscaling`

**Step 5 — Gateway Attachment**
- Select or create Gateway listener for inference traffic
- HTTPRoute generation with path/header matching for model endpoints
- Request timeout configuration (inference requests are long-running)
- Streaming response support (SSE/chunked for token-by-token output)
- API: `POST /api/v1/inference/pools/{ns}/{name}/attach`

**Step 6 — Review & Deploy**
- Complete YAML preview of all generated resources:
  - InferencePool CRD
  - EPP configuration
  - HTTPRoute
  - HPA or KEDA ScaledObject
  - DCGM Exporter DaemonSet (if not already present)
  - Prometheus Adapter ConfigMap (if using HPA)
- Dry-run validation
- Deploy with progress tracking
- API: `POST /api/v1/inference/pools/deploy`

#### 3.10.3 Inference Observability Dashboard

**Real-Time EPP Decision Visualization**

A live view showing how EPP distributes requests across the pool:
- Each pod rendered as a card showing:
  - Pod name, GPU type, node
  - Queue depth (bar graph, color-coded)
  - KV-cache utilization (percentage ring)
  - Prefix cache state indicator
  - GPU utilization + memory (from DCGM)
  - Current requests in-flight
- Animated request flow showing incoming requests being routed to selected pods
- EPP decision overlay: click any routed request to see WHY that pod was chosen
  - "Selected pod-3: queue_depth=2 (lowest), kv_cache=45% (available), prefix_hit=true"
- WebSocket-driven, 1-second refresh

**Inference Metrics Dashboard Panels:**

| Panel | Source | Description |
|-------|--------|-------------|
| Time-to-First-Token (TTFT) | ClickHouse | Distribution histogram per model/pool — the #1 metric for LLM teams |
| Tokens-per-Second (TPS) | ClickHouse + Prometheus | Throughput per pod and aggregate |
| KV-Cache Heatmap | Triton /metrics via EPP | Pool-wide KV-cache utilization — see hot spots instantly |
| Queue Depth Over Time | Triton /metrics via EPP | Correlate with request rate to validate scaling thresholds |
| Prefix Cache Hit Rate | Triton /metrics via EPP | Effectiveness of prefix-affinity routing |
| GPU Utilization | DCGM Exporter | Per-GPU utilization and memory across all pool pods |
| GPU Memory Pressure | DCGM Exporter | Track memory headroom, predict OOM events |
| Scaling Events Timeline | K8s Events + HPA status | When HPA/KEDA scaled up/down, which metric triggered it |
| Request Latency Breakdown | ClickHouse | Queue wait → inference time → token generation → response |
| Model Throughput Comparison | ClickHouse | Compare TTFT and TPS across different models/pools |
| Cost Estimation | Computed | Estimated hourly/daily GPU cost based on pool size + cloud pricing |

**ClickHouse Schema for Inference Telemetry:**

```sql
CREATE TABLE ngf_inference_logs (
    timestamp DateTime64(3),
    inference_pool String,
    model_name String,
    model_version String,
    pod_name String,
    node_name String,
    gpu_id UInt8,
    gpu_type LowCardinality(String),      -- A100, H100, L40S
    request_id String,
    trace_id String,

    -- Inference performance
    time_to_first_token_ms Float64,
    total_inference_time_ms Float64,
    tokens_generated UInt32,
    input_tokens UInt32,
    output_tokens UInt32,
    tokens_per_second Float32,

    -- EPP decision context
    epp_selected_reason LowCardinality(String),  -- least_queue, prefix_hit, kv_available, composite
    epp_decision_latency_us Float32,              -- time EPP took to decide
    queue_depth_at_selection UInt16,
    kv_cache_pct_at_selection Float32,
    prefix_cache_hit Boolean,
    candidate_pods_considered UInt8,

    -- GPU state at routing decision
    gpu_utilization_pct Float32,
    gpu_memory_used_mb UInt32,
    gpu_memory_total_mb UInt32,
    gpu_temperature_c UInt16,

    -- Scaling context
    pool_replica_count UInt16,
    pool_target_replica_count UInt16,

    -- Standard HTTP fields
    status UInt16,
    client_ip String,
    path String,
    method LowCardinality(String),
    request_size UInt64,
    response_size UInt64,

    -- XC fields (when published via Distributed Cloud)
    xc_edge_latency_ms Float64,
    xc_waf_action LowCardinality(String),
    xc_bot_classification LowCardinality(String)

) ENGINE = MergeTree()
PARTITION BY toYYYYMMDD(timestamp)
ORDER BY (inference_pool, model_name, timestamp)
TTL timestamp + INTERVAL 14 DAY;

-- Inference-specific rollups
CREATE MATERIALIZED VIEW ngf_inference_metrics_1m
ENGINE = SummingMergeTree()
PARTITION BY toYYYYMMDD(window_start)
ORDER BY (inference_pool, model_name, window_start)
TTL window_start + INTERVAL 90 DAY
AS SELECT
    toStartOfMinute(timestamp) AS window_start,
    inference_pool,
    model_name,
    count() AS request_count,
    avg(time_to_first_token_ms) AS avg_ttft,
    quantile(0.50)(time_to_first_token_ms) AS p50_ttft,
    quantile(0.95)(time_to_first_token_ms) AS p95_ttft,
    quantile(0.99)(time_to_first_token_ms) AS p99_ttft,
    avg(tokens_per_second) AS avg_tps,
    sum(tokens_generated) AS total_tokens,
    avg(queue_depth_at_selection) AS avg_queue_depth,
    avg(kv_cache_pct_at_selection) AS avg_kv_cache_pct,
    countIf(prefix_cache_hit = true) / count() AS prefix_cache_hit_rate,
    avg(gpu_utilization_pct) AS avg_gpu_util,
    avg(gpu_memory_used_mb) AS avg_gpu_mem_used,
    max(gpu_memory_used_mb) AS max_gpu_mem_used,
    avg(epp_decision_latency_us) AS avg_epp_latency
FROM ngf_inference_logs
GROUP BY window_start, inference_pool, model_name;
```

#### 3.10.4 Autoscaling Management & Visualization

**Scaling Policy Builder UI:**
- Visual threshold editor with sliders and real-time preview
- Multi-signal scaling rules (combine queue depth + KV-cache + GPU utilization)
- Preview mode: overlay current metrics against proposed thresholds to show when scaling would trigger
- Cost impact estimation: "Adding 1 replica at current GPU pricing = +$X/hour"
- Generates appropriate K8s resources (HPA or KEDA ScaledObject)

**Scaling Timeline View:**
- Combined visualization showing:
  - Request rate (RPS) overlay
  - Queue depth across pool
  - GPU utilization aggregate
  - Replica count changes (scale-up / scale-down events)
  - Cost per hour (computed from replica count × GPU pricing)
- Annotations on scaling events showing which metric triggered the change
- "Missed scaling" detection: highlight periods where metrics exceeded thresholds but scaling didn't occur (misconfigured HPA, cool-down too long, etc.)

**Cost Optimization Insights:**
- Over-provisioning detection: "Your pool averaged 2 active pods but maintained 5 replicas overnight"
- Recommendation engine: "Based on 7-day patterns, scheduled scaling to 2 replicas at 10pm and 6 replicas at 8am would save ~$X/month"
- GPU type comparison: "Switching from A100 to H100 would increase TPS by ~40% — break-even in 3 months at current usage"

#### 3.10.5 Inference-Aware Troubleshooting

**"Why Is My Inference Slow?" Diagnostic Wizard**

Interactive wizard checking (in order):
1. **EPP Health**: Is the EPP running and scraping Triton /metrics?
2. **Queue Depth**: Are any pods showing high queue depth? (>10 = warning, >25 = critical)
3. **KV-Cache Exhaustion**: Is KV-cache utilization >90% on any pod? (causes context swapping delays)
4. **Prefix Cache Effectiveness**: Is prefix routing working? Low hit rate may indicate scattered request distribution
5. **GPU Saturation**: GPU utilization >95%? Memory >90%? (DCGM metrics)
6. **Thermal Throttling**: GPU temperature >85°C? (DCGM)
7. **Scaling Responsiveness**: Is HPA/KEDA responding to demand? Time since last scale event?
8. **Gateway Overhead**: Is NGF itself adding latency? (compare total latency vs inference time)
9. **Network Issues**: MTU mismatches, DNS resolution delays for pod-to-pod communication?

Output: Prioritized findings with specific remediation steps.

API: `POST /api/v1/inference/diagnostics/slow-inference`

**Request Replay Tool:**
- Capture an inference request (prompt, parameters, headers)
- Show what EPP would do with it NOW vs what it did at the original time
- Compare: "At 2:05pm, pod-3 was chosen (queue=2, kv=45%). Right now, pod-1 would be chosen (queue=0, kv=12%)."
- Useful for debugging intermittent latency issues

API: `POST /api/v1/inference/diagnostics/replay`

**Model Performance Benchmarking:**
- Run standardized benchmark prompts against an InferencePool
- Measure TTFT, TPS, total latency across pool sizes
- Generate performance profile: "At 1 replica: p95 TTFT=280ms, at 3 replicas: p95 TTFT=95ms"
- Compare models: "Llama-3-70b vs Mixtral-8x7b on your current GPU fleet"

API: `POST /api/v1/inference/diagnostics/benchmark`

#### 3.10.6 Inference + F5 Distributed Cloud

The XC auto-publish workflow extends to inference endpoints with additional capabilities:

**Global Inference Routing:**
- Publish InferencePool endpoints to XC for global distribution
- XC routes inference requests to the nearest cluster with available GPU capacity
- Multi-region failover: if GPU pool in region A is saturated, route to region B
- Geographic compliance: keep certain model inference in specific regions

**LLM Security at the Edge:**
- WAF rules tuned for LLM endpoints: prompt injection detection, output filtering
- Per-tenant / per-API-key rate limiting at XC edge (critical for multi-tenant LLM platforms)
- DDoS protection specifically for inference endpoints (GPU resources are expensive — don't waste them on attack traffic)
- Token-based billing integration: track token consumption per tenant

**Inference-Specific XC Dashboard:**
- Global inference traffic map: which regions are sending requests, latency by region
- Edge-to-inference latency breakdown: Client → XC PoP → Origin cluster → NGF → EPP → Triton pod
- Cost per region: inference cost varies by GPU availability and pricing per region

**CRD Extension for Inference Publishing:**

```yaml
apiVersion: ngf-console.f5.com/v1alpha1
kind: DistributedCloudPublish
metadata:
  name: llm-inference-xc
spec:
  # Can reference an HTTPRoute OR an InferencePool directly
  inferencePoolRef:
    name: llama3-pool
    namespace: ml-serving
  distributedCloud:
    tenant: my-tenant
    namespace: production
    wafPolicy: llm-waf-policy       # LLM-tuned WAF rules
    botDefense:
      enabled: true
    rateLimiting:
      perTenant: true                # Rate limit per API key
      defaultRate: 100               # requests per minute
      defaultTokenBudget: 50000      # tokens per minute
    publicHostname: inference.example.com
    tls:
      mode: managed
    multiRegion:
      enabled: true
      preferredRegions: ["us-west-2", "eu-west-1"]
      failoverPolicy: nearest-available
```

#### 3.10.7 Coexistence Dashboard (KIC + NGF)

Visualize the side-by-side deployment of NGINX Ingress Controller and NGINX Gateway Fabric:

- **Split View**: Left panel shows KIC-managed Ingress/VirtualServer resources, right panel shows NGF-managed Gateway API resources
- **Traffic Split**: Percentage of cluster traffic handled by each controller
- **Route Mapping**: Which routes have equivalents on both controllers (for migration tracking)
- **Migration Readiness Score**: "74% of your KIC routes have Gateway API equivalents ready — 12 routes remaining"
- **Recommendation Engine**: "These 5 KIC VirtualServer routes use only basic path matching — they can be migrated to NGF immediately"
- **Workload Segmentation**: Clearly show which workloads are best suited for each controller
  - Standard L7 / API traffic → KIC (today) → NGF (migration target)
  - LLM inference traffic → NGF + Gateway Inference Extensions (today)

API: `GET /api/v1/coexistence/overview`

#### 3.10.8 Inference API Endpoints

```
# InferencePool Management
GET    /api/v1/inference/pools
POST   /api/v1/inference/pools
GET    /api/v1/inference/pools/:namespace/:name
PUT    /api/v1/inference/pools/:namespace/:name
DELETE /api/v1/inference/pools/:namespace/:name
POST   /api/v1/inference/pools/deploy

# EPP Configuration
GET    /api/v1/inference/pools/:namespace/:name/epp
PUT    /api/v1/inference/pools/:namespace/:name/epp
GET    /api/v1/inference/pools/:namespace/:name/epp/decisions  # live EPP decision stream

# Autoscaling
GET    /api/v1/inference/pools/:namespace/:name/autoscaling
PUT    /api/v1/inference/pools/:namespace/:name/autoscaling
GET    /api/v1/inference/pools/:namespace/:name/autoscaling/simulate  # what-if

# Inference Metrics
GET    /api/v1/inference/metrics/summary                       # aggregate inference metrics
GET    /api/v1/inference/metrics/pool/:namespace/:name         # per-pool metrics
GET    /api/v1/inference/metrics/pool/:namespace/:name/pods    # per-pod GPU state
GET    /api/v1/inference/metrics/cost                          # cost estimation

# Inference Diagnostics
POST   /api/v1/inference/diagnostics/slow-inference            # "why is it slow" wizard
POST   /api/v1/inference/diagnostics/replay                    # request replay
POST   /api/v1/inference/diagnostics/benchmark                 # performance benchmark

# Coexistence
GET    /api/v1/coexistence/overview                            # KIC + NGF split view
GET    /api/v1/coexistence/migration-readiness                 # migration score

# WebSocket
WS     /api/v1/ws/inference/epp-decisions                      # live EPP decision stream
WS     /api/v1/ws/inference/gpu-metrics                        # live GPU metrics stream
WS     /api/v1/ws/inference/scaling-events                     # scaling event stream
```

---

## 4. Configuration Database

### 4.1 Schema Overview

**Tables:**

```sql
-- User preferences and UI state
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email VARCHAR(255),
    display_name VARCHAR(255),
    preferences JSONB,              -- theme, default namespace, dashboard layout
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Audit log of all configuration changes
CREATE TABLE audit_log (
    id BIGSERIAL PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    action VARCHAR(50),             -- create, update, delete, deploy, rollback
    resource_kind VARCHAR(100),     -- Gateway, HTTPRoute, Policy, etc.
    resource_namespace VARCHAR(255),
    resource_name VARCHAR(255),
    previous_config JSONB,          -- full resource before change
    new_config JSONB,               -- full resource after change
    diff TEXT,                      -- unified diff
    metadata JSONB,                 -- extra context (dry-run result, etc.)
    created_at TIMESTAMPTZ DEFAULT NOW()
);
CREATE INDEX idx_audit_resource ON audit_log(resource_kind, resource_namespace, resource_name);
CREATE INDEX idx_audit_time ON audit_log(created_at DESC);

-- Alert rules
CREATE TABLE alert_rules (
    id UUID PRIMARY KEY,
    name VARCHAR(255),
    description TEXT,
    rule_type VARCHAR(50),          -- cert_expiry, error_rate, latency_threshold, etc.
    config JSONB,                   -- thresholds, conditions
    notification_channels JSONB,    -- [{type: "slack", webhook: "..."}, ...]
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Saved dashboard views and custom queries
CREATE TABLE saved_views (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),
    name VARCHAR(255),
    view_type VARCHAR(50),          -- dashboard, query, filter
    config JSONB,                   -- panel layout, filters, time range
    shared BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Migration projects
CREATE TABLE migration_projects (
    id UUID PRIMARY KEY,
    name VARCHAR(255),
    source_type VARCHAR(50),        -- cluster, file, git
    source_config JSONB,
    analysis_result JSONB,
    generated_resources JSONB,
    status VARCHAR(50),             -- importing, analyzed, reviewed, applied, completed
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- XC publish state tracking
CREATE TABLE xc_publish_state (
    id UUID PRIMARY KEY,
    httproute_namespace VARCHAR(255),
    httproute_name VARCHAR(255),
    xc_tenant VARCHAR(255),
    xc_namespace VARCHAR(255),
    xc_http_lb_name VARCHAR(255),
    xc_origin_pool_name VARCHAR(255),
    public_endpoint VARCHAR(255),
    last_sync_status VARCHAR(50),
    last_sync_time TIMESTAMPTZ,
    config_hash VARCHAR(64),        -- detect drift
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

### 4.2 SQLite Compatibility

For single-node / docker-compose deployments:
- Replace JSONB with TEXT (JSON stored as string)
- Replace UUID with TEXT
- Replace TIMESTAMPTZ with TEXT (ISO 8601)
- Replace BIGSERIAL with INTEGER PRIMARY KEY AUTOINCREMENT
- Use application-level JSON parsing

The backend API uses an ORM/query builder with adapter pattern to support both.

---

## 5. API Specification

### 5.1 REST API Routes

```
# Gateway Management
GET    /api/v1/gatewayclasses
GET    /api/v1/gateways
POST   /api/v1/gateways
GET    /api/v1/gateways/:namespace/:name
PUT    /api/v1/gateways/:namespace/:name
DELETE /api/v1/gateways/:namespace/:name
POST   /api/v1/gateways/deploy                # dry-run + apply

# Route Management
GET    /api/v1/httproutes
POST   /api/v1/httproutes
GET    /api/v1/httproutes/:namespace/:name
PUT    /api/v1/httproutes/:namespace/:name
DELETE /api/v1/httproutes/:namespace/:name
POST   /api/v1/routes/simulate                 # test request simulation

# Same pattern for grpcroutes, tlsroutes, tcproutes, udproutes

# Policy Management
GET    /api/v1/policies/:type                  # type = ratelimit|backendtls|waf|...
POST   /api/v1/policies/:type
GET    /api/v1/policies/:type/:namespace/:name
PUT    /api/v1/policies/:type/:namespace/:name
DELETE /api/v1/policies/:type/:namespace/:name
GET    /api/v1/policies/conflicts               # policy conflict analysis

# Certificates
GET    /api/v1/certificates
POST   /api/v1/certificates
GET    /api/v1/certificates/:namespace/:name
DELETE /api/v1/certificates/:namespace/:name
GET    /api/v1/certificates/expiring            # certs within threshold

# Observability
GET    /api/v1/metrics/summary                  # RED metrics overview
GET    /api/v1/metrics/route/:namespace/:name   # per-route metrics
GET    /api/v1/metrics/gateway/:namespace/:name # per-gateway metrics
GET    /api/v1/logs/query                       # ClickHouse log query
GET    /api/v1/logs/topn                        # top-N analytics

# Topology
GET    /api/v1/topology                         # full topology graph
GET    /api/v1/topology/gateway/:namespace/:name

# Diagnostics
POST   /api/v1/diagnostics/route-check          # "Why isn't my route working?"
POST   /api/v1/diagnostics/trace                 # request tracing
GET    /api/v1/events                            # filtered event stream

# Audit
GET    /api/v1/audit                             # audit log entries
GET    /api/v1/audit/diff                        # config diff

# F5 Distributed Cloud
GET    /api/v1/xc/status                         # XC connection status
POST   /api/v1/xc/publish                        # create DistributedCloudPublish
GET    /api/v1/xc/publish/:namespace/:name       # publish status
DELETE /api/v1/xc/publish/:namespace/:name       # unpublish
GET    /api/v1/xc/metrics/:namespace/:name       # XC-side metrics

# Migration
POST   /api/v1/migration/import                  # import source configs
GET    /api/v1/migration/:id/analysis            # analysis results
POST   /api/v1/migration/:id/generate            # generate Gateway API resources
POST   /api/v1/migration/:id/apply               # apply to cluster
GET    /api/v1/migration/:id/validate            # validate migration

# WebSocket Endpoints
WS     /api/v1/ws/events                         # real-time event stream
WS     /api/v1/ws/metrics                        # real-time metrics push
WS     /api/v1/ws/topology                       # topology status updates
```

### 5.2 Authentication & Authorization

- Kubernetes ServiceAccount token-based auth (when running in-cluster)
- OIDC integration for user-facing auth (Keycloak, Okta, etc.)
- RBAC mapped to Kubernetes namespaces
- Role model: Admin, Operator, Viewer
- API: Bearer token in Authorization header

---

## 6. Frontend Architecture

### 6.1 Technology Stack

| Layer | Choice | Rationale |
|-------|--------|-----------|
| Framework | React 18 + TypeScript | Industry standard, strong ecosystem |
| Build | Vite | Fast dev server, optimized builds |
| Styling | Tailwind CSS | Utility-first, consistent design |
| Component Library | shadcn/ui | Accessible, customizable |
| State Management | Zustand | Lightweight, TypeScript-native |
| Data Fetching | TanStack Query (React Query) | Caching, real-time refetching |
| Routing | React Router v6 | Standard SPA routing |
| Charts | Recharts + D3 | Metrics dashboards |
| Code Editor | Monaco Editor | YAML editing, NGINX config |
| Topology Graph | React Flow | Interactive node graph |
| Forms | React Hook Form + Zod | Validation, type safety |

### 6.2 Page Structure

```
/                           → Dashboard (overview metrics + topology + inference summary)
/gateways                   → Gateway list
/gateways/create            → Gateway creation wizard
/gateways/:ns/:name         → Gateway detail + topology
/inference                  → Inference Pool overview dashboard
/inference/pools            → InferencePool list
/inference/pools/create     → InferencePool creation wizard
/inference/pools/:ns/:name  → Pool detail + EPP decisions + GPU metrics
/inference/pools/:ns/:name/epp → Live EPP decision visualizer
/inference/pools/:ns/:name/scaling → Autoscaling config + timeline
/inference/pools/:ns/:name/benchmark → Performance benchmarking
/inference/diagnostics      → "Why is my inference slow?" wizard
/inference/cost             → Cost estimation + optimization
/routes                     → Route list (all types)
/routes/create/:type        → Route creation form
/routes/:type/:ns/:name     → Route detail + metrics
/policies                   → Policy list
/policies/create/:type      → Policy builder
/policies/:type/:ns/:name   → Policy detail
/certificates               → Certificate inventory
/observability              → Main observability dashboard
/observability/logs         → ClickHouse log explorer
/observability/metrics      → Prometheus metrics browser
/diagnostics                → Troubleshooting home
/diagnostics/route-check    → Route diagnostic wizard
/diagnostics/trace          → Request tracing
/xc                         → F5 Distributed Cloud overview
/xc/publish                 → XC publish management
/coexistence                → KIC + NGF coexistence dashboard
/migration                  → Migration project list
/migration/new              → New migration wizard
/migration/:id              → Migration project detail
/settings                   → User preferences, alert rules, DB config
/audit                      → Audit log viewer
```

### 6.3 Design System

- Dark mode primary (infrastructure tools convention)
- Light mode toggle
- F5 brand colors as accents (configurable)
- Consistent status colors: green (healthy), yellow (warning), red (error), grey (unknown)
- Enterprise feature badge: purple/gold accent on enterprise-only elements

---

## 7. Deployment Architecture

### 7.1 Helm Chart Structure

```
charts/ngf-console/
├── Chart.yaml
├── values.yaml
├── templates/
│   ├── _helpers.tpl
│   ├── deployment-frontend.yaml
│   ├── deployment-api.yaml
│   ├── deployment-controller.yaml
│   ├── service-frontend.yaml
│   ├── service-api.yaml
│   ├── ingress.yaml                    # or HTTPRoute for self-hosting on NGF
│   ├── configmap.yaml
│   ├── secret.yaml
│   ├── serviceaccount.yaml
│   ├── clusterrole.yaml
│   ├── clusterrolebinding.yaml
│   ├── crds/
│   │   └── distributedcloudpublish-crd.yaml
│   ├── clickhouse/
│   │   ├── statefulset.yaml
│   │   ├── service.yaml
│   │   ├── configmap-schema.yaml
│   │   └── pvc.yaml
│   ├── postgresql/                      # optional, when database.type=postgresql
│   │   ├── statefulset.yaml
│   │   ├── service.yaml
│   │   └── pvc.yaml
│   └── otel-collector/
│       ├── deployment.yaml
│       ├── service.yaml
│       └── configmap.yaml
└── charts/                              # subcharts for clickhouse, postgresql
```

### 7.2 values.yaml Key Configuration

```yaml
# NGF Edition
ngf:
  edition: enterprise          # enterprise | oss
  controllerNamespace: nginx-gateway

# Frontend
frontend:
  replicas: 2
  image:
    repository: registry.f5.com/ngf-console/frontend
    tag: "0.1.0"

# API Server
api:
  replicas: 2
  image:
    repository: registry.f5.com/ngf-console/api
    tag: "0.1.0"

# Database
database:
  type: postgresql             # postgresql | sqlite
  postgresql:
    host: ""                   # external PG, or leave empty for bundled
    port: 5432
    database: ngf_console
    existingSecret: ""
  sqlite:
    path: /data/ngf-console.db
    persistence:
      enabled: true
      size: 1Gi

# ClickHouse
clickhouse:
  enabled: true
  replicas: 1                  # single-node for small deployments
  persistence:
    size: 50Gi
  retention:
    rawLogs: 7d
    rollups1m: 90d
    rollups1h: 365d

# OpenTelemetry Collector
otelCollector:
  enabled: true
  mode: deployment             # deployment | daemonset

# Prometheus
prometheus:
  url: http://prometheus:9090  # existing Prometheus URL

# Grafana
grafana:
  enabled: false
  url: http://grafana:3000     # link to existing Grafana

# Inference / GPU Metrics
inference:
  enabled: true
  dcgmExporter:
    enabled: true              # deploy DCGM Exporter DaemonSet for GPU metrics
    image: nvcr.io/nvidia/k8s/dcgm-exporter:3.3.5-3.4.0-ubuntu22.04
  tritonMetrics:
    scrapeInterval: 5s         # EPP scraping interval for Triton /metrics
    metricsPath: /metrics
  scaling:
    backend: keda              # keda | hpa
    kedaNamespace: keda        # namespace where KEDA is installed
  costEstimation:
    enabled: true
    gpuPricing:                # per-hour pricing by GPU type (configurable)
      A100: 3.67
      H100: 8.10
      L40S: 1.84
      T4: 0.53

# F5 Distributed Cloud
xc:
  enabled: false
  tenantUrl: ""
  apiTokenSecretRef: ""
  defaultNamespace: default
  defaultWafPolicy: ""

# Authentication
auth:
  type: kubernetes             # kubernetes | oidc
  oidc:
    issuerUrl: ""
    clientId: ""
    clientSecretRef: ""

# Ingress / HTTPRoute for the console itself
ingress:
  enabled: true
  className: nginx-gateway-fabric
  hostname: ngf-console.example.com
  tls:
    enabled: true
    secretName: ngf-console-tls
```

### 7.3 Docker-Compose (Development / Demo)

```yaml
# deploy/docker-compose/docker-compose.yaml
version: '3.8'
services:
  frontend:
    build: ../../frontend
    ports: ["3000:80"]
    depends_on: [api]

  api:
    build: ../../api
    ports: ["8080:8080"]
    environment:
      - DATABASE_TYPE=sqlite
      - DATABASE_PATH=/data/ngf-console.db
      - CLICKHOUSE_URL=http://clickhouse:8123
      - KUBECONFIG=/root/.kube/config
    volumes:
      - api-data:/data
      - ~/.kube:/root/.kube:ro
    depends_on: [clickhouse]

  clickhouse:
    image: clickhouse/clickhouse-server:24.1
    ports: ["8123:8123", "9000:9000"]
    volumes:
      - clickhouse-data:/var/lib/clickhouse
      - ./clickhouse/init.sql:/docker-entrypoint-initdb.d/init.sql

  otel-collector:
    image: otel/opentelemetry-collector-contrib:latest
    ports: ["4317:4317", "4318:4318"]
    volumes:
      - ./otel/config.yaml:/etc/otelcol-contrib/config.yaml

volumes:
  api-data:
  clickhouse-data:
```

---

## 8. Project Structure for Claude Code

```
ngf-console/
├── README.md
├── Makefile                           # build, test, lint, docker targets
├── .github/
│   └── workflows/
│       ├── ci.yaml                    # lint, test, build
│       └── release.yaml               # build + push images
│
├── frontend/
│   ├── Dockerfile
│   ├── package.json
│   ├── tsconfig.json
│   ├── vite.config.ts
│   ├── tailwind.config.js
│   ├── index.html
│   ├── public/
│   │   └── favicon.ico
│   └── src/
│       ├── main.tsx
│       ├── App.tsx
│       ├── routes.tsx
│       ├── api/                       # API client layer
│       │   ├── client.ts              # axios/fetch wrapper
│       │   ├── gateways.ts
│       │   ├── routes.ts
│       │   ├── policies.ts
│       │   ├── certificates.ts
│       │   ├── metrics.ts
│       │   ├── logs.ts
│       │   ├── xc.ts
│       │   ├── migration.ts
│       │   └── diagnostics.ts
│       ├── components/
│       │   ├── layout/
│       │   │   ├── Sidebar.tsx
│       │   │   ├── Header.tsx
│       │   │   ├── MainLayout.tsx
│       │   │   └── EnterpriseBadge.tsx
│       │   ├── gateway/
│       │   │   ├── GatewayList.tsx
│       │   │   ├── GatewayCreateWizard.tsx
│       │   │   ├── GatewayDetail.tsx
│       │   │   ├── ListenerBuilder.tsx
│       │   │   └── GatewayClassSelector.tsx
│       │   ├── routes/
│       │   │   ├── RouteList.tsx
│       │   │   ├── HTTPRouteBuilder.tsx
│       │   │   ├── MatchConditionBuilder.tsx
│       │   │   ├── TrafficSplitVisualizer.tsx
│       │   │   └── RequestSimulator.tsx
│       │   ├── policies/
│       │   │   ├── PolicyList.tsx
│       │   │   ├── PolicyBuilder.tsx
│       │   │   ├── PolicyTemplateGallery.tsx
│       │   │   ├── WafManager.tsx
│       │   │   ├── ConflictDetector.tsx
│       │   │   └── SnippetsEditor.tsx
│       │   ├── certificates/
│       │   │   ├── CertificateInventory.tsx
│       │   │   ├── CertificateDetail.tsx
│       │   │   └── ExpiryTimeline.tsx
│       │   ├── observability/
│       │   │   ├── Dashboard.tsx
│       │   │   ├── MetricsPanel.tsx
│       │   │   ├── LogExplorer.tsx
│       │   │   ├── TopNView.tsx
│       │   │   └── GrafanaLink.tsx
│       │   ├── topology/
│       │   │   ├── TopologyGraph.tsx
│       │   │   ├── TopologyNode.tsx
│       │   │   └── TopologyEdge.tsx
│       │   ├── diagnostics/
│       │   │   ├── RouteCheckWizard.tsx
│       │   │   ├── RequestTracer.tsx
│       │   │   ├── ConfigDiffViewer.tsx
│       │   │   └── EventStream.tsx
│       │   ├── xc/
│       │   │   ├── XCOverview.tsx
│       │   │   ├── XCPublishForm.tsx
│       │   │   ├── XCStatusDashboard.tsx
│       │   │   └── SecurityPostureScore.tsx
│       │   ├── inference/
│       │   │   ├── InferencePoolList.tsx
│       │   │   ├── InferencePoolCreateWizard.tsx
│       │   │   ├── InferencePoolDetail.tsx
│       │   │   ├── EPPDecisionVisualizer.tsx
│       │   │   ├── EPPConfigEditor.tsx
│       │   │   ├── GPUMetricsHeatmap.tsx
│       │   │   ├── KVCacheUtilization.tsx
│       │   │   ├── PrefixCacheMonitor.tsx
│       │   │   ├── InferenceDashboard.tsx
│       │   │   ├── TTFTDistribution.tsx
│       │   │   ├── TokenThroughputChart.tsx
│       │   │   ├── ScalingPolicyBuilder.tsx
│       │   │   ├── ScalingTimeline.tsx
│       │   │   ├── CostEstimator.tsx
│       │   │   ├── InferenceSlowDiagnostic.tsx
│       │   │   ├── RequestReplayTool.tsx
│       │   │   ├── ModelBenchmark.tsx
│       │   │   └── QueueDepthMonitor.tsx
│       │   ├── coexistence/
│       │   │   ├── CoexistenceDashboard.tsx
│       │   │   ├── ControllerSplitView.tsx
│       │   │   └── MigrationReadinessScore.tsx
│       │   ├── migration/
│       │   │   ├── MigrationWizard.tsx
│       │   │   ├── ImportStep.tsx
│       │   │   ├── AnalysisReport.tsx
│       │   │   ├── SideBySideDiff.tsx
│       │   │   └── MigrationProgress.tsx
│       │   ├── common/
│       │   │   ├── YAMLPreview.tsx
│       │   │   ├── StatusBadge.tsx
│       │   │   ├── NamespaceSelector.tsx
│       │   │   ├── TimeRangeSelector.tsx
│       │   │   ├── ConfirmDialog.tsx
│       │   │   └── EnterpriseGate.tsx  # wraps enterprise features
│       │   └── audit/
│       │       ├── AuditLog.tsx
│       │       └── AuditDiffView.tsx
│       ├── hooks/
│       │   ├── useWebSocket.ts
│       │   ├── useEdition.ts          # OSS vs Enterprise detection
│       │   ├── useMetrics.ts
│       │   └── useAuth.ts
│       ├── store/
│       │   ├── index.ts
│       │   ├── gatewayStore.ts
│       │   ├── metricsStore.ts
│       │   └── settingsStore.ts
│       ├── types/
│       │   ├── gateway.ts
│       │   ├── route.ts
│       │   ├── policy.ts
│       │   ├── metrics.ts
│       │   ├── xc.ts
│       │   └── migration.ts
│       └── utils/
│           ├── yaml.ts
│           ├── validation.ts
│           └── formatting.ts
│
├── api/
│   ├── Dockerfile
│   ├── go.mod
│   ├── go.sum
│   ├── main.go
│   ├── cmd/
│   │   └── server/
│   │       └── main.go
│   ├── internal/
│   │   ├── server/
│   │   │   ├── server.go              # HTTP server setup
│   │   │   ├── middleware.go           # auth, logging, CORS
│   │   │   └── websocket.go           # WS handler
│   │   ├── handlers/
│   │   │   ├── gateways.go
│   │   │   ├── routes.go
│   │   │   ├── policies.go
│   │   │   ├── certificates.go
│   │   │   ├── metrics.go
│   │   │   ├── logs.go
│   │   │   ├── topology.go
│   │   │   ├── diagnostics.go
│   │   │   ├── inference.go           # InferencePool CRUD + EPP + scaling
│   │   │   ├── inference_metrics.go   # inference-specific metrics/dashboards
│   │   │   ├── inference_diag.go      # inference diagnostics + benchmark
│   │   │   ├── coexistence.go         # KIC + NGF coexistence view
│   │   │   ├── xc.go
│   │   │   ├── migration.go
│   │   │   └── audit.go
│   │   ├── kubernetes/
│   │   │   ├── client.go              # K8s client wrapper
│   │   │   ├── gateway.go             # Gateway API operations
│   │   │   ├── inference.go           # InferencePool + EPP operations
│   │   │   ├── watcher.go             # informer-based resource watching
│   │   │   └── edition.go             # OSS vs Enterprise detection
│   │   ├── clickhouse/
│   │   │   ├── client.go
│   │   │   ├── queries.go
│   │   │   └── schema.go
│   │   ├── database/
│   │   │   ├── interface.go           # DB adapter interface
│   │   │   ├── postgresql.go
│   │   │   ├── sqlite.go
│   │   │   └── migrations/
│   │   │       ├── 001_initial.sql
│   │   │       └── 002_xc_state.sql
│   │   ├── prometheus/
│   │   │   ├── client.go
│   │   │   └── queries.go
│   │   ├── inference/
│   │   │   ├── pool_manager.go        # InferencePool lifecycle
│   │   │   ├── epp_client.go          # EPP metrics scraping + decision stream
│   │   │   ├── triton_client.go       # Triton /metrics scraper
│   │   │   ├── dcgm_client.go         # DCGM GPU metrics
│   │   │   ├── scaling.go             # HPA/KEDA ScaledObject generation
│   │   │   ├── cost.go                # GPU cost estimation
│   │   │   ├── diagnostics.go         # Inference diagnostics engine
│   │   │   └── benchmark.go           # Performance benchmark runner
│   │   ├── coexistence/
│   │   │   ├── detector.go            # Detect KIC + NGF installations
│   │   │   ├── mapper.go              # Map KIC routes to NGF equivalents
│   │   │   └── readiness.go           # Migration readiness scoring
│   │   ├── xc/
│   │   │   ├── client.go              # F5 XC API client
│   │   │   ├── types.go
│   │   │   └── sync.go
│   │   └── migration/
│   │       ├── parser.go              # Ingress/VS/TS parsers
│   │       ├── analyzer.go            # compatibility analysis
│   │       ├── translator.go          # → Gateway API translation
│   │       └── annotations.go         # annotation mapping table
│   └── pkg/
│       ├── types/
│       │   └── api.go                 # shared API types
│       └── version/
│           └── version.go
│
├── controller/
│   ├── Dockerfile
│   ├── go.mod
│   ├── main.go
│   ├── internal/
│   │   ├── controller/
│   │   │   ├── xc_publish_controller.go    # DistributedCloudPublish reconciler
│   │   │   └── route_watcher.go            # watches HTTPRoute for annotation triggers
│   │   └── xc/
│   │       ├── reconciler.go               # XC object lifecycle
│   │       └── client.go                   # shared with api/internal/xc
│   └── config/
│       ├── crd/
│       │   └── bases/
│       │       └── ngf-console.f5.com_distributedcloudpublishes.yaml
│       ├── rbac/
│       │   ├── role.yaml
│       │   └── role_binding.yaml
│       └── manager/
│           └── manager.yaml
│
├── migration-cli/
│   ├── Dockerfile
│   ├── go.mod
│   ├── main.go
│   └── cmd/
│       ├── scan.go
│       ├── plan.go
│       ├── apply.go
│       └── validate.go
│
├── deploy/
│   ├── helm/
│   │   └── ngf-console/
│   │       ├── Chart.yaml
│   │       ├── values.yaml
│   │       └── templates/
│   │           └── ... (as described in section 7.1)
│   ├── manifests/
│   │   └── install.yaml              # generated from Helm template
│   └── docker-compose/
│       ├── docker-compose.yaml
│       ├── clickhouse/
│       │   └── init.sql
│       └── otel/
│           └── config.yaml
│
└── docs/
    ├── architecture.md
    ├── development.md
    ├── api-reference.md
    ├── deployment-guide.md
    ├── migration-guide.md
    └── xc-integration.md
```

---

## 9. Development Priorities (Phased Approach)

### Phase 1 — Foundation + Inference Observability (Weeks 1-6)

**Goal:** Ship the inference dashboard first — it's the differentiator and the demo magnet

- [ ] Project scaffolding (monorepo, CI/CD, Docker builds)
- [ ] Backend API server with K8s client (Gateway API + Inference Extension CRDs)
- [ ] Frontend shell: layout, routing, sidebar, edition detection
- [ ] InferencePool list view with GPU status
- [ ] EPP decision visualizer (live WebSocket view)
- [ ] GPU metrics heatmap (DCGM integration)
- [ ] KV-cache utilization + queue depth monitoring
- [ ] Inference dashboard: TTFT, TPS, queue depth, GPU util panels
- [ ] ClickHouse deployment + OTel pipeline (inference telemetry)
- [ ] Gateway + Route list views with status
- [ ] Basic topology graph (read-only, React Flow)
- [ ] Helm chart (basic deployment with ClickHouse + OTel)
- [ ] Docker-compose for local development

### Phase 2 — Inference Management + Configuration (Weeks 7-14)

**Goal:** Enable active InferencePool lifecycle and gateway management

- [ ] InferencePool creation wizard (full 6-step flow)
- [ ] EPP configuration editor (routing strategy, weights)
- [ ] Autoscaling policy builder (HPA/KEDA with visual threshold config)
- [ ] Scaling timeline visualization
- [ ] Cost estimation dashboard
- [ ] Gateway creation wizard (full 5-step flow)
- [ ] HTTPRoute builder with YAML preview
- [ ] Policy builder (rate limit, backend TLS, client settings)
- [ ] WAF management (Enterprise)
- [ ] Configuration DB (PostgreSQL + SQLite support)
- [ ] Prometheus metrics dashboard (RED metrics for standard traffic)
- [ ] Certificate inventory with expiry tracking
- [ ] Audit log with config diffing

### Phase 3 — XC Integration + Migration + Diagnostics (Weeks 15-22)

**Goal:** Revenue-driving XC features, customer on-ramp, and operational tooling

- [ ] DistributedCloudPublish CRD + controller (supporting both HTTPRoute and InferencePool)
- [ ] XC publish UI (route builder + inference pool)
- [ ] XC-specific inference features: multi-region routing, LLM WAF, per-tenant rate limiting
- [ ] Unified NGF + XC metrics view (including inference edge-to-origin latency)
- [ ] Security posture score
- [ ] NGINX Ingress Controller migration tool (UI + CLI)
- [ ] VirtualServer/VirtualServerRoute translation
- [ ] Coexistence dashboard (KIC + NGF side-by-side)
- [ ] Migration readiness scoring
- [ ] "Why isn't my route working?" diagnostic wizard
- [ ] "Why is my inference slow?" diagnostic wizard
- [ ] Request tracing + inference request replay

### Phase 4 — Advanced Operations (Weeks 23-30)

**Goal:** Enterprise-grade operational tooling and advanced inference capabilities

- [ ] RBAC and multi-user support
- [ ] OIDC authentication
- [ ] Model performance benchmarking tool
- [ ] Cost optimization recommendations engine
- [ ] Traffic splitting with canary automation
- [ ] Request simulation tool
- [ ] Custom ClickHouse query editor
- [ ] SnippetsFilter editor (Enterprise)
- [ ] Multi-cluster inference pool management (future)
- [ ] Capacity planning dashboards
- [ ] Log explorer with full-text search
- [ ] Alert rule configuration (cert expiry, error rate, GPU saturation)

---

## 10. Build & Development Commands

```bash
# Development
make dev                    # Start all services in dev mode
make dev-frontend           # Frontend dev server (Vite, hot reload)
make dev-api               # API server with air (Go hot reload)
make dev-compose           # Start docker-compose dependencies

# Build
make build                 # Build all images
make build-frontend        # Build frontend (npm run build)
make build-api            # Build API (go build)
make build-controller     # Build controller (go build)
make build-migration-cli  # Build migration CLI

# Docker
make docker-build         # Build all container images
make docker-push          # Push to registry

# Helm
make helm-package         # Package Helm chart
make helm-install         # Install to current cluster
make helm-template        # Render templates (dry run)

# Test
make test                 # Run all tests
make test-frontend        # Jest + React Testing Library
make test-api            # Go test ./...
make test-e2e            # Playwright E2E tests

# Lint
make lint                 # Run all linters
make lint-frontend        # ESLint + Prettier
make lint-api            # golangci-lint

# Generate
make generate-api         # Generate OpenAPI spec from handlers
make generate-crds        # Generate CRD YAML from Go types
make generate-manifests   # Generate kubectl manifests from Helm
```

---

## 11. Key Technical Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Backend language | Go | Same as NGF itself, strong K8s ecosystem, controller-runtime |
| Frontend framework | React + TypeScript | Largest ecosystem, strong typing |
| Config DB | PostgreSQL / SQLite | PG for production scale, SQLite for simplicity |
| Analytics DB | ClickHouse | Columnar, time-series optimized for access logs + inference telemetry |
| Telemetry pipeline | OpenTelemetry | Vendor-neutral, standard, pluggable exporters |
| GPU metrics | NVIDIA DCGM Exporter | Industry standard for GPU telemetry, Prometheus-compatible |
| Inference metrics | Triton /metrics + EPP | Direct scraping for real-time EPP decisions |
| Autoscaling | KEDA (preferred) / HPA | KEDA supports custom metric triggers natively, better for inference |
| Real-time updates | WebSocket | Lower latency than polling for EPP decisions/GPU metrics |
| Container runtime | Multi-stage Docker builds | Small images, reproducible builds |
| CRD for XC | Custom CRD (supports HTTPRoute + InferencePool refs) | Declarative, GitOps-friendly |
| Migration tool | Go CLI + UI | Works in CI/CD and interactive workflows |

---

## 12. Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| **Inference pool MTTR** | <5 min (from 45+ min) | Inference diagnostic wizard usage |
| **Inference pool creation time** | <3 min via wizard | Wizard completion time |
| **EPP routing visibility** | 100% of decisions traceable | EPP decision log completeness |
| Gateway creation time | <2 min (from 10+ min YAML) | Wizard completion time |
| Route issue MTTR | <5 min (from 30+ min) | Diagnostic wizard usage |
| XC publish adoption | 40% of routes + inference pools XC-published within 6 months | DistributedCloudPublish CRD count |
| Migration conversion | 80% of KIC customers migrated within 12 months | Migration tool usage |
| Active users | 500+ within 6 months of GA | Auth session tracking |
| Inference GPU cost savings | 15-25% reduction via scaling optimization | Cost dashboard before/after |
| NPS | 50+ | In-app survey |
