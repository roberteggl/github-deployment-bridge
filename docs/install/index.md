<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# Install in a cluster

Install **github-deployment-bridge** with Helm alongside FluxCD: GitHub App,
secrets, persistence, and verification.

```mermaid
flowchart LR
  A[Create GitHub App] --> B[Create Kubernetes Secret]
  B --> C[Helm install]
  C --> D[Verify Deployments]
```

## Prerequisites

- A Kubernetes cluster with [FluxCD](https://fluxcd.io/) installed
  (`Kustomization` CRDs; `HelmRelease` CRDs if you want Helm reporting)
- Helm 3
- A GitHub App (see [GitHub App setup](./github-app.md)) installed on the
  repositories you deploy
- Workload images that carry the required OCI labels, or equivalent
  `github-deployment-bridge.io/*` annotations (see
  [architecture](../architecture.md#metadata-resolution))

The bridge is an observer only. It does not reconcile GitOps state or mutate
workloads. Install it in a namespace that can watch Flux `Kustomization` and
`HelmRelease` resources (commonly `flux-system`).

HelmRelease inventory (required for image discovery) needs Flux ≥ 2.8 /
helm-controller ≥ 1.5; older clusters simply skip HelmReleases with empty inventory.

## Steps

1. [GitHub App setup](./github-app.md)
2. [Secrets](./secrets.md)
3. [Persistence (PVC)](./persistence.md)
4. [Install with Helm](./helm.md)
5. [Verify](./verify.md)

Optional: [Private image registries](./registries.md) · [Uninstall](./uninstall.md)
