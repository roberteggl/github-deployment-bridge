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
| `DESCRIPTION` | no | | Default GitHub Deployment description (overridable). Empty → `Deployed by FluxCD` |
| `LOG_URL_TEMPLATE` | no | | Default log URL template (overridable). Placeholders: `{sha}`, `{namespace}`, `{name}`, `{service}`, `{environment}`, `{cluster}` |
| `LOG_LEVEL` | no | `info` | slog level: `debug`, `info`, `warn`, or `error` |
| `METRICS_ADDR` | no | `:8080` | Prometheus metrics + convenience probes |
| `PROBE_ADDR` | no | `:8081` | controller-runtime health probes |
| `LEADER_ELECTION` | no | `true` | Enable leader election |
| `LEADER_ELECTION_ID` | no | `github-deployment-bridge` | Lease name |
| `GITHUB_APP_ID` | yes | | GitHub App ID |
| `GITHUB_INSTALLATION_ID` | no | _(automatic)_ | Explicit installation override; when unset, resolve by repository owner and cache in SQLite |
| `GITHUB_INSTALLATION_CACHE_TTL` | no | `1h` | TTL for automatically resolved owner-to-installation mappings |
| `GITHUB_PRIVATE_KEY_PATH` | yes | | Path to App private key PEM |
| `GITHUB_BASE_URL` | no | | GitHub Enterprise API base URL |
| `RETRY_MAX_ATTEMPTS` | no | `5` | Retry attempts for GitHub/OCI calls |
| `RETRY_INITIAL_BACKOFF` | no | `500ms` | Initial retry backoff |
| `RETRY_MAX_BACKOFF` | no | `30s` | Maximum exponential backoff (GitHub `Retry-After` may wait longer, up to 5m) |

## Log URL placeholders

`LOG_URL_TEMPLATE` and the per-workload `github-deployment-bridge.io/log-url`
annotation both expand these tokens (annotation wins when set):

| Placeholder | Value |
|---|---|
| `{sha}` | Commit SHA |
| `{namespace}` | Workload namespace |
| `{name}` | Workload name (`Deployment` / `StatefulSet` / `DaemonSet`) |
| `{service}` | `github-deployment-bridge.io/service`, else `{name}` |
| `{environment}` | Resolved GitHub Deployment environment |
| `{cluster}` | Resolved cluster name |

Example (Grafana Loki Explore per service):

```text
https://grafana.example.com/a/grafana-lokiexplore-app/explore/service/{name}/logs?from=now-1h&to=now&var-ds=loki&var-filters=service_name%7C%3D%7C{name}&var-filters=namespace%7C%3D%7C{namespace}
```

See also: [Helm values map](./helm-values.md)
