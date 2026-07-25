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
| _(fixed by chart)_ | `DATABASE=/data/cache.db` |
| `github.existingSecret` / chart Secret | `GITHUB_APP_ID`, `GITHUB_INSTALLATION_ID`, key file |
| `rbac.create` | _(chart only)_ — emit RBAC |
| `networkPolicy.enabled` | _(chart only)_ — emit NetworkPolicy |
| `networkPolicy.ingress.metricsFrom` | _(chart only)_ — optional scrape peers |
| `networkPolicy.egress.allowDNS` / `allowHTTPS` / `extraEgress` | _(chart only)_ |

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
```

When `config.watchNamespace` is set, the chart installs a namespaced `Role` in
that namespace (plus a lease `Role` in the release namespace) instead of a
`ClusterRole`. Flux objects and inventory workloads must share that namespace.

Full install examples: [Install with Helm](../install/helm.md)
