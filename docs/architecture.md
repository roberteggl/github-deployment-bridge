<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# Architecture

```text
GitHub Actions
    ↓
Build container
    ↓
Push GHCR image (with OCI labels)
    ↓
Flux Image Automation
    ↓
Flux reconciles Kustomization
    ↓
Deployment succeeds (Ready=True)
    ↓
Flux GitHub Deployment Bridge
        ↓
Read deployed image(s)
        ↓
Fetch OCI manifest + config (no layers)
        ↓
Authenticate as GitHub App
        ↓
Create GitHub Deployment
        ↓
Create Deployment Status (success)
```

## Design principles

- **Stateless except duplicate cache** - SQLite stores `(owner, repo, environment, commitSHA)`.
- **Zero per-app mapping** - repository and commit come from OCI labels.
- **Observe only** - the bridge never triggers deployments or mutates cluster workloads.
- **Safe reconcile loop** - malformed labels skip a workload; transient GitHub/OCI errors retry with backoff.

## Readiness filter

The controller reacts only when a Flux `Kustomization` satisfies:

- `status.conditions[type=Ready].status == True`
- `metadata.generation == status.observedGeneration`

## Workload discovery

1. Parse `.status.inventory` for `Deployment`, `StatefulSet`, and `DaemonSet`.
2. Resolve `ReplicaSet` entries via owner references to their controlling `Deployment`.
3. Ignore `Job` / `CronJob`.

## OCI labels

Required on every image:

| Label | Example |
|---|---|
| `org.opencontainers.image.source` | `https://github.com/example/backend` |
| `org.opencontainers.image.revision` | `0123456789abcdef` |
| `org.opencontainers.image.version` | `v1.8.4` |
