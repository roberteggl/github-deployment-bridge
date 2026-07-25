<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# Install in a cluster

This guide walks through installing **github-deployment-bridge** with Helm
alongside FluxCD, including GitHub App setup, secrets, and persistence.

## Prerequisites

- A Kubernetes cluster with [FluxCD](https://fluxcd.io/) installed
  (`Kustomization` CRDs; `HelmRelease` CRDs if you want Helm reporting)
- Helm 3
- A GitHub App (see [GitHub App setup](#github-app-setup)) installed on the
  repositories you deploy
- Workload images that carry the required OCI labels, or equivalent
  `github-deployment-bridge.io/*` annotations (see
  [architecture](architecture.md#metadata-resolution))

The bridge is an observer only. It does not reconcile GitOps state or mutate
workloads. Install it in a namespace that can watch Flux `Kustomization` and
`HelmRelease` resources (commonly `flux-system`).

HelmRelease inventory (required for image discovery) needs Flux ≥ 2.8 /
helm-controller ≥ 1.5; older clusters simply skip HelmReleases with empty inventory.

## GitHub App setup

Personal access tokens are **not** supported. Create a GitHub App and install it
on every repository whose deployments you want reported.

### 1. Create the App

In GitHub: **Settings → Developer settings → GitHub Apps → New GitHub App**.

Suggested settings:

| Field | Value |
|---|---|
| GitHub App name | e.g. `flux-deployment-bridge` |
| Homepage URL | your org / docs URL |
| Webhook | **Inactive** (the bridge polls the cluster; no webhook needed) |
| Where can this GitHub App be installed? | Only on this account / org |

### 2. Repository permissions

Grant **only** these repository permissions:

| Permission | Access | Why |
|---|---|---|
| **Deployments** | Read & Write | Create Deployments and lifecycle statuses |
| **Contents** | Read | Resolve the commit SHA / ref when creating a Deployment |
| **Metadata** | Read | Required baseline for GitHub Apps (repo identity) |

No other permissions (Issues, Pull requests, Actions, Administration, …) are
needed.

### 3. Generate a private key

Under the App's **Private keys**, generate a key and download the `.pem` file.
Store it securely; the chart mounts it into the pod as a Secret.

### 4. Install the App

Install the App on the target org or repositories, then note:

| Value | Where to find it |
|---|---|
| **App ID** | App settings → **About** → App ID |
| **Installation ID** | After install, the URL looks like `…/installations/<id>` |
| **Private key** | The downloaded `.pem` |

For GitHub Enterprise Server, also set `config.githubBaseURL` (or
`GITHUB_BASE_URL`) to your instance base URL.

## Secrets

The controller needs three credentials from the GitHub App. They are read from a
Kubernetes Secret with these keys:

| Secret key | Env var | Description |
|---|---|---|
| `app-id` | `GITHUB_APP_ID` | Numeric GitHub App ID |
| `installation-id` | `GITHUB_INSTALLATION_ID` | Installation ID for the org/repos |
| `private-key` | (mounted as file) | PEM private key; path set via `GITHUB_PRIVATE_KEY_PATH` |

The Helm chart mounts `private-key` at `/github/private-key.pem` and sets
`GITHUB_PRIVATE_KEY_PATH` accordingly.

### Recommended: manage the Secret yourself

```bash
kubectl -n flux-system create secret generic github-deployment-bridge \
  --from-literal=app-id=123456 \
  --from-literal=installation-id=987654 \
  --from-file=private-key=./github-app.pem
```

Then install with `github.existingSecret=github-deployment-bridge`.

### Alternative: let Helm create the Secret

Pass values at install time (fine for local/dev; prefer an external Secret
manager or sealed/SOPS secret in production):

```bash
helm upgrade --install github-deployment-bridge \
  oci://ghcr.io/roberteggl/charts/github-deployment-bridge \
  --version 1.2.0 \
  --namespace flux-system \
  --set github.appId=123456 \
  --set github.installationId=987654 \
  --set-file github.privateKey=./github-app.pem \
  --set config.clusterName=production-eu \
  --set config.environment=production
```

## Why a PVC?

The bridge keeps a small SQLite database at `/data/cache.db` keyed by
`(owner, repo, environment, commitSHA, deploymentName)`. That cache prevents duplicate GitHub
Deployments when Flux re-reconciles the same commit.

With `persistence.enabled=true` (the chart default):

- A 1Gi `PersistentVolumeClaim` backs `/data`
- Cache entries survive pod restarts and upgrades
- After a restart, the bridge still knows which commits were already reported

Without a PVC (`persistence.enabled=false`), the chart uses an `emptyDir`. The
cache is lost on every reschedule, so previously reported commits may be
reported again to GitHub.

| Setting | Default | Notes |
|---|---|---|
| `persistence.enabled` | `true` | Keep enabled in production |
| `persistence.size` | `1Gi` | More than enough for the SQLite file |
| `persistence.storageClass` | `""` | Cluster default StorageClass |
| `persistence.accessMode` | `ReadWriteOnce` | Single-replica controller |

Leave `replicaCount` at `1` when using `ReadWriteOnce`. Leader election is on by
default if you scale later with a shared/ReadWriteMany volume.

## Install with Helm

### From the published OCI chart

```bash
# Secret must already exist (see above)
helm upgrade --install github-deployment-bridge \
  oci://ghcr.io/roberteggl/charts/github-deployment-bridge \
  --version 1.2.0 \
  --namespace flux-system \
  --create-namespace \
  --set github.existingSecret=github-deployment-bridge \
  --set config.clusterName=production-eu \
  --set config.environment=production \
  --set config.environmentURL=https://app.example.com \
  --set config.logURLTemplate='https://grafana.example.com/explore?commit={sha}'
```

### From a local checkout

```bash
helm upgrade --install github-deployment-bridge \
  charts/github-deployment-bridge \
  --namespace flux-system \
  --set github.existingSecret=github-deployment-bridge \
  --set config.clusterName=production-eu \
  --set config.environment=production
```

### Minimal values file

```yaml
# values-production.yaml
config:
  clusterName: production-eu
  environment: production
  watchNamespace: ""          # empty = all namespaces
  environmentURL: https://app.example.com
  logURLTemplate: https://grafana.example.com/explore?commit={sha}

github:
  existingSecret: github-deployment-bridge

persistence:
  enabled: true
  size: 1Gi
```

```bash
helm upgrade --install github-deployment-bridge \
  oci://ghcr.io/roberteggl/charts/github-deployment-bridge \
  --version 1.2.0 \
  --namespace flux-system \
  -f values-production.yaml
```

## What the chart installs

| Resource | Purpose |
|---|---|
| `Deployment` | Controller pod |
| `ServiceAccount` + `ClusterRole` / `ClusterRoleBinding` | Watch Flux Kustomizations, HelmReleases, and workloads |
| `Secret` | GitHub App credentials (unless `existingSecret`) |
| `PersistentVolumeClaim` | SQLite duplicate-prevention cache |
| `Service` | Metrics (`:8080`) and probes (`:8081`) |

Cluster RBAC (read-only for workloads):

- `kustomize.toolkit.fluxcd.io/kustomizations`: get, list, watch
- `helm.toolkit.fluxcd.io/helmreleases`: get, list, watch
- `apps` Deployments / StatefulSets / DaemonSets / ReplicaSets: get, list, watch
- `coordination.k8s.io/leases`: leader election
- `events`: create, patch

## Configuration

All runtime settings are environment variables. Helm `config.*` / `github.*`
values map to those vars. Full reference: [configuration.md](configuration.md).

Key knobs:

| Helm value | Env | Meaning |
|---|---|---|
| `config.clusterName` | `CLUSTER_NAME` | Logical name in logs |
| `config.environment` | `ENVIRONMENT` | GitHub deployment environment name |
| `config.watchNamespace` | `WATCH_NAMESPACE` | Limit to one namespace; empty = cluster-wide |
| `config.environmentURL` | `ENVIRONMENT_URL` | Optional URL on deployment statuses |
| `config.logURLTemplate` | `LOG_URL_TEMPLATE` | Optional log link; `{sha}` → commit |
| `config.githubBaseURL` | `GITHUB_BASE_URL` | GitHub Enterprise base URL |

## Verify

```bash
# Pod running and ready
kubectl -n flux-system get deploy,pods,pvc -l app.kubernetes.io/name=github-deployment-bridge

# Logs
kubectl -n flux-system logs -l app.kubernetes.io/name=github-deployment-bridge -f

# Metrics
kubectl -n flux-system port-forward svc/github-deployment-bridge 8080:8080
curl -s localhost:8080/metrics | grep deployments_created
```

After Flux reconciles a `Kustomization` or `HelmRelease` whose inventory images
have resolvable metadata (OCI labels and/or annotations), you should see a
GitHub Deployment with lifecycle statuses for that commit under **Environments**
in the repository.

## Private image registries

The bridge only fetches the image **manifest and config blob** (never layers).
Private registries (including private GHCR packages) need a Docker config
Secret mounted into the pod - typically the same `kubernetes.io/dockerconfigjson`
Secret Flux already uses for image pulls:

```yaml
registry:
  existingDockerConfigSecret: ghcr-pull-secret
```

```bash
helm upgrade --install github-deployment-bridge \
  oci://ghcr.io/roberteggl/charts/github-deployment-bridge \
  --namespace flux-system \
  --set github.existingSecret=github-deployment-bridge \
  --set config.clusterName=production-eu \
  --set config.environment=production \
  --set registry.existingDockerConfigSecret=ghcr-pull-secret
```

`imagePullSecrets` on this chart only pull the bridge image itself; they do not
authenticate OCI label lookups. Details:
[configuration.md#private-registries](configuration.md#private-registries).

## Uninstall

```bash
helm -n flux-system uninstall github-deployment-bridge
# PVC may remain depending on reclaim policy; delete if you no longer need the cache:
kubectl -n flux-system delete pvc -l app.kubernetes.io/name=github-deployment-bridge
```
