<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# Metrics

Exposed on `METRICS_ADDR` (default `:8080`).

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
| `oci_requests_total` | Registry inspect results |
