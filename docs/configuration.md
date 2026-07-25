<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# Configuration

The bridge is configured entirely via environment variables. When installed with
Helm, set the corresponding `config.*` / `github.*` / `persistence.*` values
instead (see [install.md](install.md)).

## Environment variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `CLUSTER_NAME` | yes | | Logical cluster name used in logs |
| `ENVIRONMENT` | yes | | Default GitHub deployment environment (overridable per workload) |
| `WATCH_NAMESPACE` | no | _(all)_ | Limit Flux Kustomization watch to one namespace |
| `DATABASE` | no | `/data/cache.db` | SQLite path for duplicate prevention |
| `ENVIRONMENT_URL` | no | | Default HTTPS URL on deployment statuses (overridable) |
| `LOG_URL_TEMPLATE` | no | | Default log URL; `{sha}` is replaced with the commit (overridable) |
| `METRICS_ADDR` | no | `:8080` | Prometheus metrics + convenience probes |
| `PROBE_ADDR` | no | `:8081` | controller-runtime health probes |
| `LEADER_ELECTION` | no | `true` | Enable leader election |
| `LEADER_ELECTION_ID` | no | `github-deployment-bridge` | Lease name |
| `GITHUB_APP_ID` | yes | | GitHub App ID |
| `GITHUB_INSTALLATION_ID` | yes | | GitHub App installation ID |
| `GITHUB_PRIVATE_KEY_PATH` | yes | | Path to App private key PEM |
| `GITHUB_BASE_URL` | no | | GitHub Enterprise API base URL |
| `RETRY_MAX_ATTEMPTS` | no | `5` | Retry attempts for GitHub/OCI calls |
| `RETRY_INITIAL_BACKOFF` | no | `500ms` | Initial retry backoff |
| `RETRY_MAX_BACKOFF` | no | `30s` | Maximum retry backoff |

## Helm values map

| Helm value | Environment variable |
|---|---|
| `config.clusterName` | `CLUSTER_NAME` |
| `config.environment` | `ENVIRONMENT` |
| `config.watchNamespace` | `WATCH_NAMESPACE` |
| `config.environmentURL` | `ENVIRONMENT_URL` |
| `config.logURLTemplate` | `LOG_URL_TEMPLATE` |
| `config.leaderElection` | `LEADER_ELECTION` |
| `config.githubBaseURL` | `GITHUB_BASE_URL` |
| _(fixed by chart)_ | `DATABASE=/data/cache.db` |
| `github.existingSecret` / chart Secret | `GITHUB_APP_ID`, `GITHUB_INSTALLATION_ID`, key file |

Example `values.yaml` snippet:

```yaml
config:
  clusterName: production-eu
  environment: production
  watchNamespace: flux-system
  environmentURL: https://app.example.com
  logURLTemplate: https://grafana.example.com/explore?commit={sha}

github:
  existingSecret: github-deployment-bridge

persistence:
  enabled: true
  size: 1Gi
```

## Secrets

Credentials come from a Kubernetes Secret (never a PAT). Required keys:

| Key | Used as |
|---|---|
| `app-id` | `GITHUB_APP_ID` |
| `installation-id` | `GITHUB_INSTALLATION_ID` |
| `private-key` | PEM file mounted at `/github/private-key.pem` |

Create it yourself and set `github.existingSecret`, or pass `github.appId`,
`github.installationId`, and `github.privateKey` so the chart creates the
Secret. Prefer an externally managed Secret in production.

```bash
kubectl -n flux-system create secret generic github-deployment-bridge \
  --from-literal=app-id=123456 \
  --from-literal=installation-id=987654 \
  --from-file=private-key=./github-app.pem
```

## Persistence (PVC)

The SQLite cache at `DATABASE` (`/data/cache.db` in the chart) stores
`(owner, repo, environment, commitSHA, deploymentName)` so the same commit is not
reported twice when Flux re-reconciles (and so monorepo workloads with distinct
`deployment-name` annotations stay independent).

| Helm value | Default | Description |
|---|---|---|
| `persistence.enabled` | `true` | Use a PVC for `/data` |
| `persistence.size` | `1Gi` | Claim size |
| `persistence.storageClass` | `""` | Empty = cluster default |
| `persistence.accessMode` | `ReadWriteOnce` | Single writer |

Disable only for ephemeral/dev clusters. Without a PVC, an `emptyDir` is used
and the cache is wiped on every pod reschedule (duplicate Deployments may
appear in GitHub).

## GitHub App permissions

Repository permissions required:

| Permission | Access | Why |
|---|---|---|
| **Deployments** | Read & Write | Create Deployments and status updates |
| **Contents** | Read | Resolve the commit SHA / ref for a Deployment |
| **Metadata** | Read | Baseline App access to repository metadata |

Webhook: leave inactive. Personal access tokens are not supported.

Full setup steps: [install.md#github-app-setup](install.md#github-app-setup).

## Private registries

The bridge uses the default Docker/OCI keychain (`authn.DefaultKeychain`) when
reading OCI labels. That is **not** the same as `imagePullSecrets` (which only
pull this chart's own image).

For private registries such as GHCR, point the chart at an existing
`kubernetes.io/dockerconfigjson` Secret - often the same one Flux
`ImageRepository` / workload Deployments already use:

```yaml
registry:
  existingDockerConfigSecret: ghcr-pull-secret
  # dockerConfigKey: ".dockerconfigjson"   # default
```

The chart mounts the Secret key as `/var/run/secrets/docker/config.json` and
sets `DOCKER_CONFIG=/var/run/secrets/docker`.

For GHCR, the credentials need `read:packages` (PAT, fine-grained token, or
equivalent). The GitHub App used for Deployments is not used for registry auth.

Only the image manifest and config blob are fetched - layers are never pulled.
