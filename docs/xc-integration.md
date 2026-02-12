# F5 Distributed Cloud Integration

Publish NGINX Gateway Fabric routes and inference endpoints to F5 Distributed Cloud (XC) for global load balancing, WAF, bot protection, and DDoS mitigation.

## Overview

The XC integration creates resources in F5 Distributed Cloud that front your Kubernetes-hosted services:

```
Client → F5 XC (WAF, DDoS, Bot) → Origin Pool → NGF Gateway → Backend Service
```

## Configuration

### Helm values

```yaml
xc:
  enabled: true
  tenantUrl: "https://tenant.console.ves.volterra.io"
  apiTokenSecretRef: "xc-api-token"     # K8s Secret with the XC API token
  defaultNamespace: default
  defaultWafPolicy: "default-waf"       # Optional: default WAF policy for published routes
```

### API token

Create a Kubernetes Secret with your XC API token:

```bash
kubectl create secret generic xc-api-token \
  --from-literal=token=<your-xc-api-token> \
  -n ngf-system
```

## API endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/xc/status` | Connection status and tenant info |
| POST | `/api/v1/xc/publish` | Publish a route or pool to XC |
| GET | `/api/v1/xc/publish/{id}` | Get publish status |
| DELETE | `/api/v1/xc/publish/{id}` | Remove a publish |
| GET | `/api/v1/xc/metrics` | XC traffic metrics |

## Publishing workflow

### From the UI

1. Navigate to XC Overview (`/xc`)
2. Click "Publish" on a route or inference pool
3. Configure:
   - Origin pool settings (name, port, health check)
   - HTTP load balancer settings (hostname, WAF policy, bot protection)
   - TLS settings (auto-cert or bring-your-own)
4. Review and publish

### From the API

```bash
curl -X POST http://localhost:8080/api/v1/xc/publish \
  -H 'Content-Type: application/json' \
  -d '{
    "resourceType": "httproute",
    "resourceName": "my-route",
    "resourceNamespace": "default",
    "hostname": "app.example.com",
    "wafEnabled": true,
    "botProtection": true
  }'
```

## DistributedCloudPublish CRD

The operator manages XC publishing via a CRD:

```yaml
apiVersion: ngf-console.f5.com/v1alpha1
kind: DistributedCloudPublish
metadata:
  name: my-route-xc
  namespace: default
spec:
  sourceRef:
    kind: HTTPRoute
    name: my-route
    namespace: default
  tenantUrl: "https://tenant.console.ves.volterra.io"
  originPool:
    name: my-route-origin
    port: 80
  httpLoadBalancer:
    hostname: app.example.com
    wafPolicy: default-waf
    botProtection: true
    ddosProtection: true
  tls:
    mode: auto-cert
```

The controller reconciles this CRD by:
1. Creating an Origin Pool in XC pointing at the NGF Gateway address
2. Creating an HTTP Load Balancer in XC with the configured hostname
3. Attaching WAF, bot protection, and DDoS policies
4. Updating status with the XC-assigned public endpoint

## XC Overview page

The XC Overview page (`/xc`) displays:

- **Connection status** -- whether the XC tenant is reachable
- **Published resources** -- list of routes and pools published to XC
- **XC metrics** -- traffic volume, latency, WAF blocks, bot detections
- **Quick publish** -- publish new routes directly from the overview

## Enterprise only

XC integration requires the Enterprise edition (`ngf.edition: enterprise` in Helm values). In OSS mode, the XC menu item is visible but greyed out.
