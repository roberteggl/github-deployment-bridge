<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# Configuration

The bridge is configured entirely via environment variables.

| Variable | Required | Default | Description |
|---|---|---|---|
| `CLUSTER_NAME` | yes | | Logical cluster name used in logs |
| `ENVIRONMENT` | yes | | GitHub deployment environment (e.g. `production`) |
| `WATCH_NAMESPACE` | no | _(all)_ | Limit Flux Kustomization watch to one namespace |
| `DATABASE` | no | `/data/cache.db` | SQLite path for duplicate prevention |
| `ENVIRONMENT_URL` | no | | Optional URL attached to deployment statuses |
| `LOG_URL_TEMPLATE` | no | | Optional log URL; `{sha}` is replaced with the commit |
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

## Example

```yaml
clusterName: production-eu
environment: production
watchNamespace: flux-system
database: /data/cache.db
environmentURL: https://app.example.com
logURLTemplate: https://grafana.example.com/explore?commit={sha}
```

## GitHub App permissions

The GitHub App needs only:

- **Deployments**: Read & Write
- **Metadata**: Read
- **Contents**: Read

Never use a personal access token.

## Private registries

The bridge uses the default Docker/OCI keychain (`authn.DefaultKeychain`).
For private registries, mount a Docker config into the pod and set
`DOCKER_CONFIG` (or mount at `/home/nonroot/.docker/config.json`).

Only the image manifest and config blob are fetched — layers are never pulled.
