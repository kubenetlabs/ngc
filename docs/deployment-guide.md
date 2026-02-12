# Deployment Guide

This document has been consolidated into the [Installation Guide](installation.md) and [Configuration Reference](configuration.md).

- For step-by-step installation instructions, see [Installation Guide](installation.md)
- For all configuration options (API flags, Helm values, CRD specs), see [Configuration Reference](configuration.md)

## Quick reference

### Local development

```bash
# Terminal 1: API
cd api && go run ./cmd/server

# Terminal 2: Frontend
cd frontend && pnpm install && pnpm dev

# Open http://localhost:5173
```

### Docker Compose

```bash
docker compose -f deploy/docker-compose/docker-compose.yaml up
```

| Service | Port |
|---------|------|
| Frontend | 3000 |
| API | 8080 |
| ClickHouse | 8123, 9000 |
| OTel Collector | 4317, 4318 |

### Kubernetes (Helm)

```bash
# Install CRDs
kubectl apply -f deploy/helm/ngf-console/crds/

# Install chart
helm install ngf-console deploy/helm/ngf-console \
  --namespace ngf-system --create-namespace
```

### Upgrade

```bash
helm upgrade ngf-console deploy/helm/ngf-console --namespace ngf-system
```

### Uninstall

```bash
helm uninstall ngf-console --namespace ngf-system
```
