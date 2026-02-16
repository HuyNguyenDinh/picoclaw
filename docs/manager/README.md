# PicoClaw Multi-Tenant Manager

## What

PicoClaw Manager is a REST API that provisions isolated PicoClaw agent instances on Kubernetes. One API call creates a full tenant — its own namespace, agent deployment, gateway, config, RBAC, and persistent storage — without touching a single YAML file.

```
┌─────────────────────────────────────────────┐
│  picoclaw-system namespace                  │
│  ┌─────────────────────────────┐            │
│  │  picoclaw-manager           │            │
│  │  REST API (port 8080)       │──► PostgreSQL
│  │  K8s client-go              │            │
│  │  Go template renderer       │            │
│  └──────────┬──────────────────┘            │
└─────────────┼───────────────────────────────┘
              │ creates per tenant:
              ▼
┌─────────────────────────────────────────────┐
│  picoclaw-tenant-{id} namespace             │
│  ConfigMap · Agent · Gateway · Service      │
│  PVC · ServiceAccount · RBAC                │
└─────────────────────────────────────────────┘
```

Each tenant is fully isolated in its own Kubernetes namespace with dedicated resources, secrets, and storage.

## Why

PicoClaw agents should be accessible to anyone — including people who have never heard of Kubernetes.

Today, deploying a PicoClaw instance requires editing YAML manifests, running `kubectl apply`, and understanding namespaces, configmaps, and RBAC. That's fine for a single developer, but it doesn't scale to teams, classrooms, or managed services.

The Manager removes that barrier:

- **Teams**: Give each team member their own AI assistant with one API call.
- **Service providers**: Offer PicoClaw-as-a-Service — tenants sign up, you call the API, they get a working agent.
- **Educators**: Provision agents for an entire classroom in seconds.
- **Non-technical admins**: A future web dashboard or CLI can wrap this API, so nobody needs to learn K8s.

The goal is simple: **if you can send an HTTP request, you can give someone their own PicoClaw agent.**

## How

### Authentication

All API endpoints (except `/health`) require a Bearer token:

```
Authorization: Bearer <your-api-key>
```

Set the API key via the `API_KEY` environment variable when starting the manager.

### API Reference

| Method   | Endpoint                       | Description                    |
|----------|--------------------------------|--------------------------------|
| `GET`    | `/health`                      | Health check (no auth)         |
| `POST`   | `/api/v1/tenants`              | Create a new tenant            |
| `GET`    | `/api/v1/tenants`              | List all tenants               |
| `GET`    | `/api/v1/tenants/{id}`         | Get tenant details             |
| `PUT`    | `/api/v1/tenants/{id}`         | Update tenant configuration    |
| `DELETE` | `/api/v1/tenants/{id}`         | Delete tenant and namespace    |
| `POST`   | `/api/v1/tenants/{id}/restart` | Restart tenant pods            |

---

### Create a Tenant

```bash
curl -X POST http://localhost:8080/api/v1/tenants \
  -H "Authorization: Bearer test-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "acme",
    "display_name": "ACME Corp",
    "providers": {
      "anthropic": {
        "api_key": "sk-...",
        "api_base": "https://api.minimax.io/v1"
      }
    },
    "agents": {
      "defaults": {
        "provider": "anthropic",
        "model": "MiniMax-M2.5",
        "max_tokens": 200000,
        "temperature": 0.7
      }
    },
    "channels": {
      "telegram": {
        "enabled": true,
        "token": "your-telegram-bot-token",
        "allow_from": [123456]
      }
    },
    "resources": {
      "agent_cpu": "500m",
      "agent_memory": "1Gi",
      "gateway_cpu": "250m",
      "gateway_memory": "512Mi",
      "workspace_size": "500Mi"
    }
  }'
```

**Response** (201 Created):

```json
{
  "id": "acme",
  "display_name": "ACME Corp",
  "namespace": "picoclaw-tenant-acme",
  "status": "active",
  "resources": {
    "agent_cpu": "500m",
    "agent_memory": "1Gi",
    "gateway_cpu": "250m",
    "gateway_memory": "512Mi",
    "workspace_size": "500Mi"
  },
  "created_at": "2026-02-16T10:00:00Z",
  "updated_at": "2026-02-16T10:00:00Z"
}
```

**What happens behind the scenes:**

1. Validates `tenant_id` is DNS-safe (lowercase, alphanumeric, hyphens, max 53 chars)
2. Checks the tenant doesn't already exist
3. Builds `config.json` from providers, agents, and channels
4. Renders K8s manifest templates (namespace, configmap, deployments, service, RBAC, PVC)
5. Applies all manifests to the cluster via server-side apply
6. Saves tenant metadata to PostgreSQL

If any step fails, cleanup runs automatically (namespace deleted on DB failure).

**Key fields:**

| Field          | Required | Description                                              |
|----------------|----------|----------------------------------------------------------|
| `tenant_id`    | Yes      | Unique identifier, becomes part of the namespace name    |
| `display_name` | Yes      | Human-readable name                                      |
| `providers`    | Yes      | LLM provider config (API keys, base URLs)                |
| `agents`       | No       | Agent defaults (model, temperature, max tokens)          |
| `channels`     | Yes      | Channel config (Telegram, Discord, Slack, etc.)          |
| `resources`    | No       | CPU/memory/storage limits (sensible defaults if omitted) |

---

### List Tenants

```bash
curl http://localhost:8080/api/v1/tenants \
  -H "Authorization: Bearer test-api-key"
```

**Response** (200 OK):

```json
[
  {
    "id": "acme",
    "display_name": "ACME Corp",
    "namespace": "picoclaw-tenant-acme",
    "status": "active",
    "resources": { ... },
    "created_at": "2026-02-16T10:00:00Z",
    "updated_at": "2026-02-16T10:00:00Z"
  }
]
```

---

### Get Tenant

```bash
curl http://localhost:8080/api/v1/tenants/acme \
  -H "Authorization: Bearer test-api-key"
```

Returns the same shape as the create response. Returns `404` if the tenant doesn't exist.

---

### Update Tenant

```bash
curl -X PUT http://localhost:8080/api/v1/tenants/acme \
  -H "Authorization: Bearer test-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "display_name": "ACME Corporation",
    "resources": {
      "agent_memory": "2Gi"
    }
  }'
```

Only include the fields you want to change. Omitted fields remain unchanged. The manager re-renders and re-applies all K8s manifests with the updated config.

---

### Delete Tenant

```bash
curl -X DELETE http://localhost:8080/api/v1/tenants/acme \
  -H "Authorization: Bearer test-api-key"
```

Returns `204 No Content`. Deletes the entire `picoclaw-tenant-acme` namespace and all resources within it, then removes the record from PostgreSQL.

---

### Restart Tenant Pods

```bash
curl -X POST http://localhost:8080/api/v1/tenants/acme/restart \
  -H "Authorization: Bearer test-api-key"
```

Triggers a rolling restart of the agent and gateway deployments. Useful after config changes or to recover from issues.

---

### Health Check

```bash
curl http://localhost:8080/health
```

Returns `{"status": "ok"}`. No authentication required.

---

### Error Responses

All errors return JSON with an `error` field:

```json
{
  "error": "tenant \"acme\" already exists"
}
```

Common HTTP status codes:

| Code  | Meaning                                      |
|-------|----------------------------------------------|
| `400` | Invalid request body or validation failure   |
| `401` | Missing authorization header                 |
| `403` | Invalid API key                              |
| `404` | Tenant not found                             |
| `500` | Internal error (K8s failure, DB error, etc.) |

## Where

### Local Development

**Prerequisites**: Go 1.25+, Docker, kubectl configured to a cluster (minikube, kind, etc.)

1. Start PostgreSQL:

   ```bash
   docker run -d --name picoclaw-db \
     -p 5432:5432 \
     -e POSTGRES_PASSWORD=test \
     -e POSTGRES_DB=picoclaw \
     postgres:16
   ```

2. Run the manager:

   ```bash
   go run ./cmd/picoclaw-manager
   ```

   Default config: listens on `:8080`, API key `test-api-key`, templates from `k8s/base/`.

3. Test it:

   ```bash
   curl http://localhost:8080/health
   ```

### Docker

Build the manager image:

```bash
docker build -f Dockerfile.manager -t picoclaw-manager:latest .
```

Run with environment variables:

```bash
docker run -d --name picoclaw-manager \
  -p 8080:8080 \
  -e DATABASE_URL="postgres://postgres:test@host.docker.internal:5432/picoclaw?sslmode=disable" \
  -e API_KEY="your-secret-key" \
  -e PICOCLAW_IMAGE="huy2408/picoclaw:latest" \
  -v $HOME/.kube:/root/.kube:ro \
  picoclaw-manager:latest
```

### Kubernetes

Apply the manager manifests:

```bash
# Create namespace and RBAC
kubectl apply -f k8s/manager/namespace.yaml
kubectl apply -f k8s/manager/rbac.yaml

# Configure secrets (edit these first!)
kubectl apply -f k8s/manager/secret.yaml

# Deploy
kubectl apply -f k8s/manager/configmap.yaml
kubectl apply -f k8s/manager/deployment.yaml
kubectl apply -f k8s/manager/service.yaml
```

**Important**: Edit `k8s/manager/secret.yaml` before applying — change `DATABASE_URL` and `API_KEY` to real values.

### Environment Variables

| Variable        | Default                                       | Description                          |
|-----------------|-----------------------------------------------|--------------------------------------|
| `DATABASE_URL`  | `postgres://postgres:test@localhost:5432/picoclaw?sslmode=disable` | PostgreSQL connection string |
| `API_KEY`       | `test-api-key`                                | Bearer token for API authentication  |
| `LISTEN_ADDR`   | `:8080`                                       | HTTP listen address                  |
| `PICOCLAW_IMAGE`| `huy2408/picoclaw:latest`                     | Container image for tenant workloads |
| `TEMPLATE_DIR`  | `k8s/base`                                    | Path to K8s manifest templates       |
| `KUBECONFIG`    | *(empty — uses in-cluster config)*            | Path to kubeconfig file              |

## When

### Tenant Lifecycle

```
  Create ──► Active ──► Update (repeat) ──► Delete
               │                               ▲
               └──► Restart ───────────────────┘
```

| Operation   | When to use                                                     |
|-------------|-----------------------------------------------------------------|
| **Create**  | Onboarding a new user, team, or customer                        |
| **Update**  | Changing LLM provider, switching models, enabling new channels  |
| **Restart** | After config changes don't take effect, or to recover from issues |
| **Delete**  | Offboarding a tenant, cleaning up test environments             |

### Monitoring

- **Health endpoint**: `GET /health` — use for liveness/readiness probes
- **Check tenant pods**: `kubectl get pods -n picoclaw-tenant-{id}`
- **Check tenant logs**: `kubectl logs -n picoclaw-tenant-{id} deployment/picoclaw-agent`
- **List all tenant namespaces**: `kubectl get ns -l picoclaw.io/tenant`

### Default Resource Limits

If `resources` is omitted during tenant creation, these defaults apply:

| Resource        | Default  |
|-----------------|----------|
| Agent CPU       | 500m     |
| Agent Memory    | 1Gi      |
| Gateway CPU     | 250m     |
| Gateway Memory  | 512Mi    |
| Workspace Size  | 500Mi    |
