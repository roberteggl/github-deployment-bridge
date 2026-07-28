<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# Metrics

Exposed on `METRICS_ADDR` (default `:8080`) at `/metrics` over unauthenticated HTTP.

| Metric | Meaning |
|---|---|
| `deployments_created_total` | GitHub Deployments created |
| `deployment_status_updates_total` | Deployment status POSTs that succeeded |
| `deployment_failures_total` | Status `failure` emitted (Flux/app failure) |
| `deployment_errors_total` | Status `error` emitted (bridge-only fault) |
| `deployment_duplicates_skipped_total` | Idempotent skips |
| `deployment_inactive_total` | Status `inactive` emitted |
| `github_api_requests_total` | GitHub API calls by `operation` and `result` |
| `github_api_failures_total` | Failed GitHub API calls |
| `github_api_latency_seconds` | GitHub API latency histogram |
| `github_installation_resolutions_total` | Installation resolution outcomes (`resolved`, `cache_hit`, `failure`, or fixed-ID `fallback`) |
| `oci_requests_total` | Registry inspect results |

## Scraping

The chart ships pod annotations for annotation-based scrapers
(`prometheus.io/scrape`, `port`, `path`). For Prometheus Operator, enable an
optional ServiceMonitor:

```yaml
serviceMonitor:
  enabled: true
  # Match your Prometheus serviceMonitorSelector, e.g.:
  labels:
    release: kube-prometheus-stack
  interval: 30s
```

## Access control

`/metrics` has no auth (bearer token / mTLS). Counters are not secrets, but
treat the endpoint as cluster-internal:

1. Keep the Service `ClusterIP` (default).
2. Enable `networkPolicy.enabled` and set `networkPolicy.ingress.metricsFrom`
   to the Prometheus / scrape namespace (or pod selectors).
3. Prefer the ServiceMonitor over open cluster-wide scrapes when using the
   Prometheus Operator.

Bearer-token or kube-rbac-proxy auth is intentionally out of scope; use
NetworkPolicy peers instead.

## Alerting

Optional `prometheusRule.enabled` ships four alerts (NotReady, bridge errors,
GitHub API failures, GitHub API p99 latency). Enable alongside ServiceMonitor
and match `prometheusRule.labels` to your Prometheus `ruleSelector`. Triage:
[Runbook](../operations/runbook.md).

## Grafana

Official dashboard JSON:
[`deploy/grafana/github-deployment-bridge.json`](https://github.com/roberteggl/github-deployment-bridge/blob/main/deploy/grafana/github-deployment-bridge.json).
Import steps and variables: [Grafana dashboard](../operations/grafana.md).
