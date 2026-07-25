<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# Environment variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `CLUSTER_NAME` | yes | | Logical cluster name used in logs |
| `ENVIRONMENT` | yes | | Default GitHub deployment environment (overridable per workload) |
| `WATCH_NAMESPACE` | no | _(all)_ | Limit Flux Kustomization / HelmRelease watch to one namespace |
| `DATABASE` | no | `/data/cache.db` | SQLite path for duplicate prevention |
| `ENVIRONMENT_URL` | no | | Default HTTPS URL on deployment statuses (overridable) |
| `LOG_URL_TEMPLATE` | no | | Default log URL; `{sha}` is replaced with the commit (overridable) |
| `LOG_LEVEL` | no | `info` | slog level: `debug`, `info`, `warn`, or `error` |
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
| `RETRY_MAX_BACKOFF` | no | `30s` | Maximum exponential backoff (GitHub `Retry-After` may wait longer, up to 5m) |

See also: [Helm values map](./helm-values.md)
