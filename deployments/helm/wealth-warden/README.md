# Wealth Warden Helm Chart

This Helm chart deploys the Wealth Warden application on a Kubernetes cluster. The chart includes separate deployments for:
- **API**: The Go backend server (port 2000)
- **WebUI**: The Vue.js frontend served via Nginx (port 80)

The WebUI automatically proxies `/api/` requests to the API service.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- PostgreSQL database (can be deployed as a dependency or use external)

## Installation

### Quick Start

```bash
# Add any required Helm repositories (if using PostgreSQL subchart)
helm repo add bitnami https://charts.bitnami.com/bitnami
helm repo update

# Install the chart
helm install wealth-warden ./deployments/helm/wealth-warden

# Or install with custom values
helm install wealth-warden ./deployments/helm/wealth-warden -f my-values.yaml
```

### Using External PostgreSQL

If you're using an external PostgreSQL database, set `postgresql.enabled: false` and configure the connection details:

```yaml
postgresql:
  enabled: false

config:
  postgres:
    host: "your-postgres-host"
    user: "your-user"
    password: "your-password"
    port: 5432
    db: "wealth_warden"
```

### Running Migrations

The chart includes an optional migration job that can run database migrations before the API deployment starts. To enable it:

```yaml
migration:
  enabled: true
  type: "up"  # Options: up, down, status, fresh, fresh-seed-basic, fresh-seed-full
  hook: "pre-install,pre-upgrade"  # Run before install/upgrade
```

The migration job will:
- Run automatically before the API deployment (via Helm hooks)
- Wait for PostgreSQL to be ready (via init container)
- Clean up automatically after successful completion
- Retry up to 3 times on failure

**Important:** For production, use `type: "up"` to run migrations incrementally. Use `fresh-seed-basic` or `fresh-seed-full` only for initial setup or development.

### Using Secrets

For production deployments, it's recommended to use Kubernetes secrets instead of plain values. You can either:

1. **Let Helm create secrets** (set `secrets.create: true` and provide values):
```yaml
secrets:
  create: true
  postgresPassword: "your-secure-password"
  jwtWebClientAccess: "your-jwt-secret"
  jwtWebClientRefresh: "your-jwt-refresh-secret"
  jwtWebClientEncodeId: "your-jwt-encode-id"
  mailerPassword: "your-mailer-password"
  superAdminPassword: "your-admin-password"
```

2. **Create secrets manually** (set `secrets.create: false`):
```bash
kubectl create secret generic wealth-warden-secret \
  --from-literal=postgres-password='your-password' \
  --from-literal=jwt-web-client-access='your-jwt-secret' \
  --from-literal=jwt-web-client-refresh='your-refresh-secret' \
  --from-literal=jwt-web-client-encode-id='your-encode-id' \
  --from-literal=mailer-password='your-mailer-password' \
  --from-literal=super-admin-password='your-admin-password'
```

## Architecture

The chart creates two separate deployments:

1. **API Deployment** (`wealth-warden-api`): The Go backend server
   - Service: `wealth-warden-api` (ClusterIP, port 2000)
   - Handles all API requests

2. **WebUI Deployment** (`wealth-warden-webui`): The Vue.js frontend
   - Service: `wealth-warden-webui` (ClusterIP, port 80)
   - Serves static files via Nginx
   - Proxies `/api/` requests to the API service

## Configuration

### API Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `api.enabled` | Enable API deployment | `true` |
| `api.replicaCount` | Number of API replicas | `1` |
| `api.image.repository` | API image repository | `wealth-warden` |
| `api.image.tag` | API image tag | `latest` |
| `api.image.pullPolicy` | API image pull policy | `IfNotPresent` |
| `api.service.type` | API service type | `ClusterIP` |
| `api.service.port` | API service port | `2000` |
| `api.resources.limits.cpu` | API CPU limit | `500m` |
| `api.resources.limits.memory` | API memory limit | `512Mi` |
| `api.resources.requests.cpu` | API CPU request | `100m` |
| `api.resources.requests.memory` | API memory request | `128Mi` |
| `api.autoscaling.enabled` | Enable API HPA | `false` |
| `api.autoscaling.minReplicas` | Minimum API replicas | `2` |
| `api.autoscaling.maxReplicas` | Maximum API replicas | `10` |

### WebUI Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `webui.enabled` | Enable WebUI deployment | `true` |
| `webui.replicaCount` | Number of WebUI replicas | `1` |
| `webui.image.repository` | WebUI image repository | `wealth-warden-client` |
| `webui.image.tag` | WebUI image tag | `latest` |
| `webui.image.pullPolicy` | WebUI image pull policy | `IfNotPresent` |
| `webui.service.type` | WebUI service type | `ClusterIP` |
| `webui.service.port` | WebUI service port | `80` |
| `webui.resources.limits.cpu` | WebUI CPU limit | `200m` |
| `webui.resources.limits.memory` | WebUI memory limit | `256Mi` |
| `webui.resources.requests.cpu` | WebUI CPU request | `50m` |
| `webui.resources.requests.memory` | WebUI memory request | `64Mi` |
| `webui.autoscaling.enabled` | Enable WebUI HPA | `false` |

### Migration Job Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `migration.enabled` | Enable migration job | `false` |
| `migration.type` | Migration type: `up`, `down`, `status`, `fresh`, `fresh-seed-basic`, `fresh-seed-full` | `up` |
| `migration.hook` | Helm hook to run migration (e.g., `pre-install,pre-upgrade`) | `pre-install,pre-upgrade` |
| `migration.hookWeight` | Hook weight (lower runs earlier) | `-5` |
| `migration.hookDeletePolicy` | When to delete the job | `before-hook-creation,hook-succeeded` |
| `migration.image.repository` | Migration image repository | `wealth-warden` |
| `migration.image.tag` | Migration image tag | `latest` |
| `migration.backoffLimit` | Number of retries on failure | `3` |
| `migration.activeDeadlineSeconds` | Maximum job duration (seconds) | `600` |
| `migration.ttlSecondsAfterFinished` | Clean up job after completion (seconds) | `3600` |
| `migration.waitForPostgres` | Wait for PostgreSQL to be ready | `true` |
| `migration.resources.limits.cpu` | Migration CPU limit | `500m` |
| `migration.resources.limits.memory` | Migration memory limit | `512Mi` |

**Migration Types:**
- `up`: Runs all pending migrations
- `down`: Rolls back the last migration
- `status`: Shows migration status (read-only)
- `fresh`: Drops database and runs all migrations
- `fresh-seed-basic`: Fresh migrations + basic seeders
- `fresh-seed-full`: Fresh migrations + all seeders

### Common Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `ingress.enabled` | Enable ingress | `false` |
| `ingress.className` | Ingress class name | `""` |
| `ingress.hosts` | Ingress hosts | `[{host: wealth-warden.local, paths: [{path: /, pathType: Prefix}]}]` |
| `postgresql.enabled` | Enable PostgreSQL subchart | `true` |
| `config.postgres.host` | PostgreSQL host | `postgres` |
| `config.postgres.user` | PostgreSQL user | `postgres` |
| `config.postgres.password` | PostgreSQL password | `""` |
| `config.postgres.port` | PostgreSQL port | `5432` |
| `config.postgres.db` | PostgreSQL database | `wealth_warden` |
| `secrets.create` | Create secrets from values | `false` |

## Examples

### Production Deployment

```yaml
# Enable migrations to run before API deployment
migration:
  enabled: true
  type: "up"  # Use "up" for incremental migrations in production
  waitForPostgres: true

api:
  enabled: true
  replicaCount: 3
  image:
    repository: your-registry/wealth-warden
    tag: "v1.0.0"
    pullPolicy: Always
  resources:
    limits:
      cpu: 1000m
      memory: 1Gi
    requests:
      cpu: 500m
      memory: 512Mi
  autoscaling:
    enabled: true
    minReplicas: 3
    maxReplicas: 10
    targetCPUUtilizationPercentage: 70

webui:
  enabled: true
  replicaCount: 2
  image:
    repository: your-registry/wealth-warden-client
    tag: "v1.0.0"
    pullPolicy: Always
  resources:
    limits:
      cpu: 200m
      memory: 256Mi
    requests:
      cpu: 100m
      memory: 128Mi

ingress:
  enabled: true
  className: "nginx"
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
  hosts:
    - host: wealth-warden.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: wealth-warden-tls
      hosts:
        - wealth-warden.example.com

postgresql:
  enabled: false

config:
  postgres:
    host: "production-postgres.example.com"
    user: "wealth_warden"
    password: ""  # Set via secret

secrets:
  create: true
  postgresPassword: "secure-password-from-secret-manager"
  jwtWebClientAccess: "secure-jwt-secret"
  jwtWebClientRefresh: "secure-refresh-secret"
  jwtWebClientEncodeId: "secure-encode-id"
  mailerPassword: "secure-mailer-password"
  superAdminPassword: "secure-admin-password"
```

### Development Deployment

```yaml
# Enable migrations with seeding for development
migration:
  enabled: true
  type: "fresh-seed-basic"  # Fresh database with basic seeders
  waitForPostgres: true

api:
  enabled: true
  replicaCount: 1
  image:
    tag: "dev"
  resources:
    limits:
      cpu: 200m
      memory: 256Mi
    requests:
      cpu: 100m
      memory: 128Mi

webui:
  enabled: true
  replicaCount: 1
  image:
    tag: "dev"

ingress:
  enabled: true
  hosts:
    - host: wealth-warden.local
      paths:
        - path: /
          pathType: Prefix

postgresql:
  enabled: true
  auth:
    postgresPassword: "dev-password"
    database: "wealth_warden"

config:
  cors:
    allowedOrigins:
      - "http://localhost:5000"
      - "http://wealth-warden.local"
```

### Deploying Only API or WebUI

You can disable either component if needed:

```yaml
# Deploy only API
api:
  enabled: true
webui:
  enabled: false

# Deploy only WebUI (requires external API)
api:
  enabled: false
webui:
  enabled: true
```

## Upgrading

```bash
helm upgrade wealth-warden ./deployments/helm/wealth-warden -f my-values.yaml
```

## Uninstallation

```bash
helm uninstall wealth-warden
```

## Troubleshooting

### Check Pod Status

```bash
# Check all pods
kubectl get pods -l app.kubernetes.io/name=wealth-warden

# Check API pods only
kubectl get pods -l app.kubernetes.io/component=api

# Check WebUI pods only
kubectl get pods -l app.kubernetes.io/component=webui

# Check migration jobs
kubectl get jobs -l app.kubernetes.io/name=wealth-warden
```

### View Logs

```bash
# View API logs
kubectl logs -l app.kubernetes.io/component=api

# View WebUI logs
kubectl logs -l app.kubernetes.io/component=webui

# View migration job logs
kubectl logs -l app.kubernetes.io/component=migration --tail=100
```

### Check Services

```bash
# Check all services
kubectl get svc -l app.kubernetes.io/name=wealth-warden

# Check API service
kubectl get svc wealth-warden-api

# Check WebUI service
kubectl get svc wealth-warden-webui
```

### Port Forward for Local Testing

```bash
# Port forward WebUI (which proxies to API)
kubectl port-forward svc/wealth-warden-webui 8080:80

# Or port forward API directly
kubectl port-forward svc/wealth-warden-api 2000:2000
```

Then access the application at `http://localhost:8080` (WebUI) or `http://localhost:2000` (API directly)

## Notes

- The application requires a PostgreSQL database. You can either use the included PostgreSQL subchart or configure an external database.
- Health checks are configured by default at `/healthz` for the API
- The WebUI automatically proxies `/api/` requests to the API service via Nginx
- Make sure to set secure passwords and secrets for production deployments
- The API uses environment variables for configuration, which are set from the Helm values
- Both API and WebUI can be scaled independently
- The ingress routes traffic to the WebUI service, which handles both static files and API proxying
- **Migrations**: The migration job runs automatically before API deployment when enabled. It includes an init container that waits for PostgreSQL to be ready. Use `type: "up"` for production migrations and `fresh-seed-basic`/`fresh-seed-full` only for initial setup.

