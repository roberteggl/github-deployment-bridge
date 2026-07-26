<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# Persistence

SQLite at `DATABASE` (chart: `/data/cache.db`) stores
`(owner, repo, environment, commitSHA, deploymentName)`, the GitHub
`deployment_id`, and the latest status so reconciles stay idempotent across
restarts (including monorepo workloads with distinct `deployment-name`
annotations). For UI-friendly monorepo setups, prefer distinct `environment`
values — see [Architecture → Monorepos](../architecture.md#monorepos).

| Helm value | Default | Description |
|---|---|---|
| `persistence.enabled` | `true` | PVC for `/data` (otherwise `emptyDir`) |
| `persistence.size` | `1Gi` | Claim size |
| `persistence.storageClass` | `""` | Empty = cluster default |
| `persistence.accessMode` | `ReadWriteOnce` | Single writer |
| `replicaCount` | `1` | Required with a SQLite-backed PVC |

Multi-replica with persistence is unsupported - the chart fails closed. Keep
`replicaCount: 1` and `persistence.enabled: true` in production.

Why this matters, and what breaks without a PVC:
[Install → PVC](../install/persistence.md).
