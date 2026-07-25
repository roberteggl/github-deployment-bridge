<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# Configuration

Runtime settings are environment variables. With Helm, set the matching
`config.*` / `github.*` / `persistence.*` values instead.

| Page | Contents |
|---|---|
| [Environment variables](./environment.md) | Full env var reference |
| [Helm values map](./helm-values.md) | Chart value → env mapping |
| [Secrets](./secrets.md) | Credential Secret keys |
| [Persistence](./persistence.md) | SQLite cache PVC values |
| [GitHub App permissions](./github-app.md) | Required App scopes |
| [Private registries](./registries.md) | Docker config for OCI label reads |
| [Metrics](./metrics.md) | Prometheus metrics |
| [Operations runbook](../operations/runbook.md) | Alert triage (optional PrometheusRule) |

Install walkthrough: [Install](../install/).
