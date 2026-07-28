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
| `LOG_URL_TEMPLATE_ESCAPE` | no | `false` | Percent-encode substituted values. Enable for templates containing path segments or query parameters; `false` preserves existing literal expansion. |
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

Only GitHub's `environment_url` and Deployment Status `log_url` are prominent
operational links. The bridge deliberately keeps the Deployment payload small;
use `log_url` to send operators to the system that already owns the rich data.

Set `LOG_URL_TEMPLATE_ESCAPE=true` (Helm: `config.logURLTemplateEscape: true`)
for the presets below. Each substituted value is percent-encoded without
changing the template's URL syntax. The final result must be an absolute HTTPS
URL. An invalid result is omitted with a warning; deployment reporting continues.

## Copy-paste presets

Replace only the hostname, datasource/dashboard identifiers, and any fixed
dashboard path.

### Grafana Loki Explore

```text
https://grafana.example.com/a/grafana-lokiexplore-app/explore/service/{name}/logs?from=now-1h&to=now&var-ds=loki&var-filters=service_name%7C%3D%7C{name}&var-filters=namespace%7C%3D%7C{namespace}
```

### Grafana dashboard

```text
https://grafana.example.com/d/workload/logs?var-cluster={cluster}&var-namespace={namespace}&var-service={service}&var-environment={environment}&var-sha={sha}
```

### Generic Flux/Kubernetes dashboard

```text
https://ops.example.com/clusters/{cluster}/namespaces/{namespace}/workloads/{name}?environment={environment}&service={service}&revision={sha}
```

## Workload overrides and monorepos

The annotation remains a template and takes precedence over the global value.
This Kustomization-managed Deployment uses the global preset and gives one
monorepo workload its own deployment identity:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: storefront-api
  namespace: storefront
  annotations:
    github-deployment-bridge.io/auto-report: "true"
    github-deployment-bridge.io/deployment-name: "storefront-api"
    github-deployment-bridge.io/service: "api"
```

A HelmRelease-managed StatefulSet can override the link (annotations belong on
the discovered workload, usually via the chart's pod/workload annotation values):

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: checkout-worker
  namespace: checkout
  annotations:
    github-deployment-bridge.io/auto-report: "true"
    github-deployment-bridge.io/deployment-name: "checkout-worker"
    github-deployment-bridge.io/log-url: "https://grafana.example.com/d/workload/logs?var-cluster={cluster}&var-namespace={namespace}&var-service={service}&var-sha={sha}"
```

`deployment-name` separates cache/supersession identity for repositories with
multiple workloads; it does not add a new visible GitHub payload field.

See also: [Helm values map](./helm-values.md)
