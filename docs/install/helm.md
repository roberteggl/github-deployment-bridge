<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# Install with Helm

Chart on [Artifact Hub](https://artifacthub.io/packages/helm/github-deployment-bridge/github-deployment-bridge)
(Cosign-signed, verified publisher).

## From the published OCI chart

```bash
# Secret must already exist (see Secrets)
helm upgrade --install github-deployment-bridge \
  oci://ghcr.io/roberteggl/charts/github-deployment-bridge \
  --version 1.3.3 \
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
  --version 1.3.3 \
  --namespace flux-system \
  -f values-production.yaml
```

## What the chart installs

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

- **Ingress:** TCP on `containerPorts.metrics` / `containerPorts.probes`
  (optional `ingress.metricsFrom` peers). These are pod listen ports - not
  `service.metricsPort` / `service.probePort`.
- **Egress:** DNS + HTTPS + kube-API (`allowKubeAPI`, port `6443` by default;
  in-cluster `:443` is covered by `allowHTTPS`). Tighten destinations with
  `egress.extraEgress` (CIDRs) if your policy requires it; set
  `allowKubeAPI: false` only when you supply equivalent rules yourself.

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

## Configuration

All runtime settings are environment variables. Helm `config.*` / `github.*`
values map to those vars - see [Configuration](../configuration/) for the full
reference (env vars, Helm map, metrics, registries).

| Helm value | Env | Meaning |
|---|---|---|
| `config.clusterName` | `CLUSTER_NAME` | Logical name in logs |
| `config.environment` | `ENVIRONMENT` | GitHub deployment environment |
| `config.watchNamespace` | `WATCH_NAMESPACE` | Limit to one namespace; empty = cluster-wide |
| `config.environmentURL` | `ENVIRONMENT_URL` | Optional URL on deployment statuses |
| `config.logURLTemplate` | `LOG_URL_TEMPLATE` | Optional log link; `{sha}` → commit |
| `config.logLevel` | `LOG_LEVEL` | `debug` / `info` / `warn` / `error` |
| `config.githubBaseURL` | `GITHUB_BASE_URL` | GitHub Enterprise base URL |

Retry knobs (`config.retry.*`), leader election, and chart-only options
(NetworkPolicy, ServiceMonitor, PrometheusRule) are listed in
[Helm values map](../configuration/helm-values.md).

Next: [Verify](./verify.md)
