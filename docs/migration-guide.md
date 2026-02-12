# Migration Guide

Migrate from NGINX Ingress Controller (KIC) to NGINX Gateway Fabric using the NGF Console migration tooling.

## Overview

The migration workflow supports three input formats:

| Format | Source | Description |
|--------|--------|-------------|
| `nginx-conf` | Raw NGINX config | Parses `server` and `location` blocks |
| `ingress-yaml` | Kubernetes Ingress YAML | Parses `kind: Ingress` documents |
| `virtualserver-yaml` | NGINX VirtualServer YAML | Parses VirtualServer, VirtualServerRoute, TransportServer |

## Migration CLI

### Install

```bash
cd migration-cli && go build -o bin/ngf-migrate .
```

### Commands

**Scan** -- discover KIC resources in a cluster:

```bash
./bin/ngf-migrate scan --kubeconfig /path/to/kubeconfig
```

**Plan** -- generate a migration plan:

```bash
./bin/ngf-migrate plan --kubeconfig /path/to/kubeconfig
```

**Apply** -- apply the migration plan:

```bash
# Dry-run first
./bin/ngf-migrate apply --kubeconfig /path/to/kubeconfig --dry-run

# Apply for real
./bin/ngf-migrate apply --kubeconfig /path/to/kubeconfig
```

**Validate** -- verify migrated resources are healthy:

```bash
./bin/ngf-migrate validate --kubeconfig /path/to/kubeconfig
```

## Web UI migration wizard

The NGF Console UI provides a 4-step migration wizard at `/migration/new`:

### Step 1: Import

Upload or paste your source configuration:
- Raw NGINX config file
- Kubernetes Ingress YAML (single or multi-document)
- NGINX VirtualServer/VirtualServerRoute YAML

The import endpoint parses the content and returns discovered resources.

### Step 2: Analysis

Review the compatibility analysis:
- **Overall score** -- weighted percentage (high=100, medium=60, low=20)
- **Per-resource analysis** -- each resource gets a confidence rating:
  - **High** -- direct mapping to Gateway API
  - **Medium** -- requires review (e.g., annotation-based features need policy attachments)
  - **Low** -- manual intervention needed (e.g., TCPRoute varies by implementation)
- **Issues and notes** -- specific migration considerations per resource

### Step 3: Generate

Preview the generated Gateway API resources:
- Generated YAML for each Gateway and HTTPRoute
- Combined multi-document YAML for bulk apply
- Editable before applying

### Step 4: Apply

Apply to the cluster:
- **Dry-run** -- validates resources without creating them
- **Live apply** -- creates resources in the cluster (not yet implemented; returns 501)

## API endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/migration/import` | Import source configuration |
| POST | `/api/v1/migration/analysis` | Analyze for compatibility |
| POST | `/api/v1/migration/generate` | Generate Gateway API YAML |
| POST | `/api/v1/migration/apply` | Apply to cluster (dry-run supported) |
| POST | `/api/v1/migration/validate` | Validate migrated resources |

### Import request

```json
{
  "source": "file",
  "content": "server { listen 80; location / { proxy_pass http://backend; } }",
  "format": "nginx-conf"
}
```

### Import response

```json
{
  "id": "a1b2c3d4...",
  "resourceCount": 2,
  "resources": [
    {"kind": "Gateway", "name": "nginx-gateway-1", "namespace": "default", "apiVersion": "gateway.networking.k8s.io/v1"},
    {"kind": "HTTPRoute", "name": "nginx-route-1", "namespace": "default", "apiVersion": "gateway.networking.k8s.io/v1"}
  ]
}
```

## Coexistence dashboard

During migration, use the Coexistence Dashboard (`/coexistence`) to view KIC and NGF resources side-by-side:

- **Overview** -- split view showing Ingress resources on the left and Gateway API resources on the right
- **Migration readiness** -- percentage score indicating how ready the cluster is for full migration
- **Route mapping** -- which KIC routes have Gateway API equivalents

## Supported resource mappings

| Source (KIC) | Target (NGF) | Confidence |
|--------------|--------------|------------|
| Ingress with path rules | HTTPRoute | High |
| Ingress with TLS | Gateway listener + HTTPRoute | High |
| VirtualServer | HTTPRoute | Medium |
| VirtualServerRoute | HTTPRoute (merged) | Medium |
| TransportServer | TCPRoute | Low |
| Rate limiting annotations | RateLimitPolicy | Medium |
| WAF annotations | WAFPolicy (Enterprise) | Medium |

## Current limitations

- **Apply** and **Validate** endpoints return 501 Not Implemented until cluster-backed apply is built
- Annotation-based features (rate limiting, WAF) are flagged but not auto-converted
- TransportServer to TCPRoute conversion requires manual review
- Multi-document YAML with mixed resource types is supported for import but not for annotation mapping
