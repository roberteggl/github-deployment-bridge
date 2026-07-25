<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# Install with Helm

## From the published OCI chart

```bash
# Secret must already exist (see Secrets)
helm upgrade --install github-deployment-bridge \
  oci://ghcr.io/roberteggl/charts/github-deployment-bridge \
  --version 1.2.2 \
  --namespace flux-system \
  --create-namespace \
  --set github.existingSecret=github-deployment-bridge \
  --set config.clusterName=production-eu \
  --set config.environment=production \
  --set config.environmentURL=https://app.example.com \
  --set config.logURLTemplate='https://grafana.example.com/explore?commit={sha}'
```

## From a local checkout

```bash
helm upgrade --install github-deployment-bridge \
  charts/github-deployment-bridge \
  --namespace flux-system \
  --set github.existingSecret=github-deployment-bridge \
  --set config.clusterName=production-eu \
  --set config.environment=production
```

## Minimal values file

```yaml
# values-production.yaml
config:
  clusterName: production-eu
  environment: production
  watchNamespace: ""          # empty = all namespaces
  environmentURL: https://app.example.com
  logURLTemplate: https://grafana.example.com/explore?commit={sha}
  logLevel: info

github:
  existingSecret: github-deployment-bridge

persistence:
  enabled: true
  size: 1Gi
```

```bash
helm upgrade --install github-deployment-bridge \
  oci://ghcr.io/roberteggl/charts/github-deployment-bridge \
  --version 1.2.2 \
  --namespace flux-system \
  -f values-production.yaml
```

## What the chart installs

```mermaid
flowchart TB
  subgraph chart[Helm release]
    D[Deployment]
    SA[ServiceAccount + RBAC]
    S[Secret]
    PVC[PersistentVolumeClaim]
    SVC[Service :8080 / :8081]
    NP[NetworkPolicy optional]
    SM[ServiceMonitor optional]
    PR[PrometheusRule optional]
  end
  D --> Flux[Watch Kustomization / HelmRelease]
  D --> GH[GitHub Deployments API]
  S --> D
  PVC --> D
  SM --> SVC
  PR --> SM
```

| Resource | Purpose |
|---|---|
| `Deployment` | Controller pod (`replicaCount: 1`, Recreate when PVC enabled) |
| `ServiceAccount` + RBAC | Watch Flux + workloads; lease Role in the release namespace |
| `Secret` | GitHub App credentials (unless `existingSecret`) |
| `PersistentVolumeClaim` | SQLite duplicate-prevention cache |
| `Service` | Metrics (`:8080`) and probes (`:8081`) |
| `NetworkPolicy` | Optional; metrics/probes ingress + DNS/HTTPS egress |
| `ServiceMonitor` | Optional; Prometheus Operator scrape of `/metrics` |
| `PrometheusRule` | Optional; high-signal alerts (see [Runbook](../operations/runbook.md)) |

### RBAC modes

| `config.watchNamespace` | Watch / inventory | Leader election |
|---|---|---|
| `""` (default) | `ClusterRole` + `ClusterRoleBinding` | `Role` / `RoleBinding` in the **release** namespace |
| e.g. `flux-system` | `Role` / `RoleBinding` in that namespace | Same lease Role in the **release** namespace |

When `watchNamespace` is set, Flux CRs **and** inventory workloads must live in
that namespace (the controller cache is namespaced the same way).

Watch permissions (read-only for workloads):

- `kustomize.toolkit.fluxcd.io/kustomizations`: get, list, watch
- `helm.toolkit.fluxcd.io/helmreleases`: get, list, watch
- `apps` Deployments / StatefulSets / DaemonSets / ReplicaSets: get, list, watch
- `events`: create, patch

Lease permissions (release namespace only):

- `coordination.k8s.io/leases`: get, list, watch, create, update, patch, delete

### NetworkPolicy (optional)

Set `networkPolicy.enabled: true` to restrict the pod:

- **Ingress:** TCP metrics + probe ports (optional `ingress.metricsFrom` peers)
- **Egress:** DNS + HTTPS by default (GitHub API, registries); add
  `egress.extraEgress` for kube-apiserver CIDRs / `:6443` if needed

### ServiceMonitor (optional)

Requires the Prometheus Operator CRDs (`monitoring.coreos.com/v1`). Set
`serviceMonitor.enabled: true` (and usually `serviceMonitor.labels` to match
your Prometheus `serviceMonitorSelector`).

`/metrics` is unauthenticated HTTP. Restrict scrape peers with
`networkPolicy.ingress.metricsFrom` when NetworkPolicy is enforced. Details:
[Metrics](../configuration/metrics.md).

### PrometheusRule (optional)

Requires the same CRDs (and kube-state-metrics for the NotReady alert). Set
`prometheusRule.enabled: true` with `prometheusRule.labels` matching your
Prometheus `ruleSelector`. Pair with `serviceMonitor.enabled` so app metrics
exist. Alert meanings and triage: [Runbook](../operations/runbook.md).

## Configuration knobs

All runtime settings are environment variables. Helm `config.*` / `github.*`
values map to those vars. Full reference: [Configuration](../configuration/).

| Helm value | Env | Meaning |
|---|---|---|
| `config.clusterName` | `CLUSTER_NAME` | Logical name in logs |
| `config.environment` | `ENVIRONMENT` | GitHub deployment environment name |
| `config.watchNamespace` | `WATCH_NAMESPACE` | Limit to one namespace; empty = cluster-wide |
| `config.environmentURL` | `ENVIRONMENT_URL` | Optional URL on deployment statuses |
| `config.logURLTemplate` | `LOG_URL_TEMPLATE` | Optional log link; `{sha}` → commit |
| `config.logLevel` | `LOG_LEVEL` | slog level (`debug` / `info` / `warn` / `error`) |
| `config.githubBaseURL` | `GITHUB_BASE_URL` | GitHub Enterprise base URL |
| `config.retry.maxAttempts` | `RETRY_MAX_ATTEMPTS` | GitHub/OCI retry attempts (default `5`) |
| `config.retry.initialBackoff` | `RETRY_INITIAL_BACKOFF` | Initial backoff (default `500ms`) |
| `config.retry.maxBackoff` | `RETRY_MAX_BACKOFF` | Max backoff (default `30s`) |

Next: [Verify](./verify.md)
