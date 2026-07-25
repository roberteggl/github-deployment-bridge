<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# GitHub Deployment Bridge

Helm chart for [GitHub Deployment Bridge](https://github.com/roberteggl/github-deployment-bridge) - a Kubernetes controller that reports FluxCD `Kustomization` / `HelmRelease` reconciliations to the GitHub Deployments API.

**Docs:** [Install](https://roberteggl.github.io/github-deployment-bridge/install/) · [Configuration](https://roberteggl.github.io/github-deployment-bridge/configuration/) · [Architecture](https://roberteggl.github.io/github-deployment-bridge/architecture)

## Prerequisites

- Kubernetes cluster with [FluxCD](https://fluxcd.io/) (`Kustomization` / `HelmRelease` CRDs)
- A [GitHub App](https://roberteggl.github.io/github-deployment-bridge/install/github-app) with **Deployments** (read & write), **Contents** (read), and **Metadata** (read)
- Helm 3.8+ (OCI support)

## Install

Create the GitHub App Secret first (recommended):

```bash
kubectl -n flux-system create secret generic github-deployment-bridge \
  --from-literal=app-id=123456 \
  --from-literal=installation-id=987654 \
  --from-file=private-key=./github-app.pem
```

Install from the published OCI chart:

```bash
helm upgrade --install github-deployment-bridge \
  oci://ghcr.io/roberteggl/charts/github-deployment-bridge \
  --version 1.3.2 \
  --namespace flux-system \
  --create-namespace \
  --set github.existingSecret=github-deployment-bridge \
  --set config.clusterName=production-eu \
  --set config.environment=production
```

## Configuration

| Value | Default | Description |
|---|---|---|
| `config.clusterName` | `production` | Logical cluster name in logs |
| `config.environment` | `production` | GitHub deployment environment name |
| `config.watchNamespace` | `""` | Limit watches to one namespace; empty = cluster-wide |
| `config.environmentURL` | `""` | Optional URL attached to deployment statuses |
| `config.logURLTemplate` | `""` | Optional log link; `{sha}` → commit SHA |
| `github.existingSecret` | `""` | Secret with `app-id`, `installation-id`, `private-key` |
| `persistence.enabled` | `true` | PVC for SQLite deduplication cache (keep on in production) |
| `networkPolicy.enabled` | `false` | Opt-in NetworkPolicy |
| `serviceMonitor.enabled` | `false` | Opt-in Prometheus Operator ServiceMonitor |
| `prometheusRule.enabled` | `false` | Opt-in PrometheusRule alerts |

Full values reference: [Helm values](https://roberteggl.github.io/github-deployment-bridge/configuration/helm-values).

## Workload metadata

Container images should include OCI labels (`org.opencontainers.image.source`, `org.opencontainers.image.revision`). Workloads opt in with `github-deployment-bridge.io/auto-report=true`. See [Architecture](https://roberteggl.github.io/github-deployment-bridge/architecture).

## License

[Apache License 2.0](https://github.com/roberteggl/github-deployment-bridge/blob/main/LICENSE)
