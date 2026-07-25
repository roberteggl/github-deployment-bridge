<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# Helm values map

| Helm value | Environment variable |
|---|---|
| `config.clusterName` | `CLUSTER_NAME` |
| `config.environment` | `ENVIRONMENT` |
| `config.watchNamespace` | `WATCH_NAMESPACE` |
| `config.environmentURL` | `ENVIRONMENT_URL` |
| `config.logURLTemplate` | `LOG_URL_TEMPLATE` |
| `config.logLevel` | `LOG_LEVEL` |
| `config.leaderElection` | `LEADER_ELECTION` |
| `config.githubBaseURL` | `GITHUB_BASE_URL` |
| `config.retry.maxAttempts` | `RETRY_MAX_ATTEMPTS` |
| `config.retry.initialBackoff` | `RETRY_INITIAL_BACKOFF` |
| `config.retry.maxBackoff` | `RETRY_MAX_BACKOFF` |
| _(fixed by chart)_ | `DATABASE=/data/cache.db` |
| `github.existingSecret` / chart Secret | `GITHUB_APP_ID`, `GITHUB_INSTALLATION_ID`, key file |
| `github.allowInsecureValues` | _(chart only)_ - allow inline `appId` / `installationId` / `privateKey` (Helm release Secret risk) |
| `commonLabels` / `podLabels` | _(chart only)_ - extra labels on all resources / pod template only |
| `rbac.create` | _(chart only)_ - emit RBAC |
| `networkPolicy.enabled` | _(chart only)_ - emit NetworkPolicy |
| `networkPolicy.ingress.metricsFrom` | _(chart only)_ - optional scrape peers |
| `networkPolicy.egress.allowDNS` / `allowHTTPS` / `allowKubeAPI` / `kubeAPIPorts` / `extraEgress` | _(chart only)_ |
| `containerPorts.metrics` / `containerPorts.probes` | `METRICS_ADDR` / `PROBE_ADDR` listen ports (also NetworkPolicy ingress) |
| `serviceMonitor.enabled` | _(chart only)_ - emit Prometheus Operator ServiceMonitor |
| `serviceMonitor.labels` / `interval` / `scrapeTimeout` / … | _(chart only)_ - scrape tuning |
| `prometheusRule.enabled` | _(chart only)_ - emit PrometheusRule alerts |
| `prometheusRule.labels` / thresholds / `runbookURL` | _(chart only)_ - alert tuning |

`values.schema.json` validates types and enums (e.g. `config.logLevel`) on
`helm install` / `upgrade` / `lint` / `template`. Config and chart-managed
Secret changes roll the Deployment via `checksum/config` and `checksum/secret`
pod annotations.

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

networkPolicy:
  enabled: true
  ingress:
    metricsFrom:
      - namespaceSelector:
          matchLabels:
            name: monitoring
  # allowKubeAPI: true (default) - TCP 6443 for the apiserver
  #   (:443 covered by allowHTTPS; add to kubeAPIPorts if you disable HTTPS)
  # egress.extraEgress: []  # optional CIDR-scoped kube-API rules
serviceMonitor:
  enabled: true
  labels:
    release: kube-prometheus-stack

prometheusRule:
  enabled: true
  labels:
    release: kube-prometheus-stack
```

When `config.watchNamespace` is set, the chart installs a namespaced `Role` in
that namespace (plus a lease `Role` in the release namespace) instead of a
`ClusterRole`. Flux objects and inventory workloads must share that namespace.

`/metrics` is unauthenticated HTTP. Pair `serviceMonitor.enabled` with
`networkPolicy` + `metricsFrom` when the CNI enforces NetworkPolicy. See
[Metrics](./metrics.md). Alert triage: [Runbook](../operations/runbook.md).

Full install examples: [Install with Helm](../install/helm.md)
