<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# Runbook

Triage steps for the optional Helm `prometheusRule` alerts and related
operational failures. Metrics reference: [Metrics](../configuration/metrics.md).
Dashboard: [Grafana](./grafana.md).

Enable rules with Prometheus Operator:

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

`GitHubDeploymentBridgeNotReady` needs [kube-state-metrics](https://github.com/kubernetes/kube-state-metrics).
App-metric alerts need the ServiceMonitor (or equivalent scrape) so series carry
`namespace` and `pod` labels.

Do **not** page on `deployment_failures_total` - that counts Flux/app
`failure` statuses (expected when a reconciliation fails). Use
`deployment_errors_total` for bridge-only faults.

## Bridge not ready

**Alert:** `GitHubDeploymentBridgeNotReady`

**Meaning:** The Deployment has unavailable replicas for ≥ 5m.

**Check:**

1. `kubectl -n <ns> describe deploy <release>` - events for FailedAttachVolume,
   ImagePullBackOff, CrashLoopBackOff.
2. With `persistence.enabled` (default), upgrades use `Recreate`. A stuck old
   pod holding an RWO volume blocks the new pod - delete the old pod only if it
   is Terminating / stuck; otherwise wait for detach.
3. Confirm `replicaCount: 1` with a PVC (chart fails closed on multi-replica +
   PVC).
4. Logs: `kubectl -n <ns> logs deploy/<release> --previous` if crashing.

## Bridge errors

**Alert:** `GitHubDeploymentBridgeErrors`

**Meaning:** `deployment_errors_total` is rising - the bridge posted GitHub
Deployment status `error` (its own fault), not Flux `failure`.

**Check:**

1. Logs around the spike for GitHub/OCI/cache errors.
2. GitHub App Secret: `app-id`, `installation-id`, `private-key` present and
   valid; installation covers the target repos ([GitHub App](../install/github-app.md)).
3. PVC / SQLite: `/data` mounted and writable ([Persistence](../install/persistence.md)).
4. Workload opt-in: `github-deployment-bridge.io/auto-report=true` plus OCI
   labels or annotation overrides ([Architecture](../architecture.md#metadata-resolution)).

## GitHub API failures

**Alert:** `GitHubDeploymentBridgeGitHubAPIFailures`

**Meaning:** Sustained `github_api_failures_total` (auth, permissions, rate
limit, or GitHub outage).

**Check:**

1. App permissions: Deployments R/W, Contents R, Metadata R on the installation.
2. `GITHUB_BASE_URL` / `config.githubBaseURL` for GHES.
3. Rate limits: GitHub `Retry-After` / rate-reset headers are honored on
   retry (up to 5 minutes per attempt). Raise `config.retry.maxAttempts` /
   backoff for stubborn limits; check App secondary rate-limit docs if
   thrashing continues.
4. NetworkPolicy egress: HTTPS (and DNS) allowed to GitHub / GHES.

## GitHub API high latency

**Alert:** `GitHubDeploymentBridgeGitHubAPIHighLatency`

**Meaning:** p99 `github_api_latency_seconds` above the chart threshold
(default 5s) for 15m.

**Check:**

1. GitHub / GHES status and path latency from the cluster.
2. Whether a burst of reconciliations coincides (many Flux objects updating).
3. Optionally raise `prometheusRule.githubAPILatencyP99Seconds` if the baseline
   for your region/GHES is higher.

## Deployments missing in GitHub (no alert)

When alerts are quiet but GitHub shows no Deployment:

1. Flux object reconciled? Conditions and inventory populated (HelmRelease needs
   Flux ≥ 2.8).
2. Workload opted in (`auto-report=true`) and images reachable for OCI inspect
   ([Registries](../install/registries.md)).
3. Environment name: distinct `config.environment` per cluster if multiple
   bridges share a repo.
4. Dedup cache: same `(owner, repo, environment, commit, deploymentName)` skips
   duplicates - expected after retries.

## Deployment log link is missing (no alert)

1. Confirm `LOG_URL_TEMPLATE` / `config.logURLTemplate`, or the workload's
   `github-deployment-bridge.io/log-url` override, is non-empty.
2. Search bridge logs for `log URL template expanded to an invalid HTTPS URL`.
   Invalid links are intentionally omitted without failing reconciliation or
   GitHub deployment/status reporting.
3. The expanded URL must be absolute HTTPS. Enable
   `LOG_URL_TEMPLATE_ESCAPE=true` / `config.logURLTemplateEscape: true` when
   workload values appear in paths or query parameters.
4. Copy a known-good [preset](../configuration/environment.md#copy-paste-presets)
   and replace its host and dashboard/datasource identifiers.
