<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# Grafana dashboard

Official Grafana dashboard for GitHub Deployment Bridge. It visualizes
Prometheus metrics (lifecycle counters, GitHub API / OCI rates, controller
health) and lists firing alerts.

Dashboard JSON:
[`deploy/grafana/github-deployment-bridge.json`](https://github.com/roberteggl/github-deployment-bridge/blob/main/deploy/grafana/github-deployment-bridge.json)
(`uid`: `github-deployment-bridge`).

## Prerequisites

1. Scrape `/metrics` (Helm `serviceMonitor.enabled=true`, or equivalent).
2. A Prometheus-compatible Grafana datasource (Prometheus, VictoriaMetrics, …).
3. Optional: enable `prometheusRule.enabled` so the Alerts row has series to
   show. Triage: [Runbook](./runbook.md).

```yaml
serviceMonitor:
  enabled: true
  labels:
    release: kube-prometheus-stack

prometheusRule:
  enabled: true
  labels:
    release: kube-prometheus-stack
```

## Import

### Grafana UI

1. **Dashboards → New → Import**.
2. Upload
   [`github-deployment-bridge.json`](https://raw.githubusercontent.com/roberteggl/github-deployment-bridge/main/deploy/grafana/github-deployment-bridge.json)
   (or paste the file contents).
3. Select your Prometheus datasource when prompted (`DS_PROMETHEUS`).
4. Set **Release** to your Helm release name / `fullnameOverride` (default:
   `github-deployment-bridge`).

### grafana.com provisioning / ConfigMap sidecar

Mount the same JSON from the repo (ConfigMap, grafana-operator `GrafanaDashboard`,
or sidecar). Keep the file `uid` stable so upgrades replace the existing
dashboard instead of creating duplicates.

## Dashboard variables

| Variable | Purpose |
|---|---|
| **Datasource** | Prometheus-compatible datasource |
| **Release** | Helm release / `fullnameOverride`; filters `job` via regex |
| **Namespace** / **Job** / **Pod** | Narrow to one scrape target (`All` by default) |

If panels stay empty, check that `up{job=~"<release>"}` returns series for your
ServiceMonitor job label. With kube-prometheus-stack the job name is often the
ServiceMonitor name (chart fullname). Override **Release** if you renamed the
release.

## What to watch

| Area | Signals |
|---|---|
| Lifecycle | `deployments_created_total`, status updates, inactive marks |
| Flux/app vs bridge | `deployment_failures_total` (expected on bad reconciles) vs `deployment_errors_total` (actionable) |
| GitHub / OCI | request rates, API latency histogram, OCI inspect success |
| Health | process CPU / memory / goroutines, target `up` |

Do **not** page on `deployment_failures_total` - use `deployment_errors_total`
and the GitHub API alerts. Metric catalog: [Metrics](../configuration/metrics.md).
