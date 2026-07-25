<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# github-deployment-bridge

[![REUSE status](https://api.reuse.software/badge/github.com/roberteggl/github-deployment-bridge)](https://api.reuse.software/info/github.com/roberteggl/github-deployment-bridge)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/roberteggl/github-deployment-bridge/badge)](https://scorecard.dev/viewer/?uri=github.com/roberteggl/github-deployment-bridge)
[![Artifact Attestations](https://img.shields.io/badge/attestations-SLSA_provenance-2ea44f)](https://github.com/roberteggl/github-deployment-bridge/attestations)
[![Cosign](https://img.shields.io/badge/cosign-keyless_signed-2ea44f)](docs/releasing.md#verify-signatures-and-attestations)

A lightweight Kubernetes controller that bridges **FluxCD** reconciliations to the **GitHub Deployments API**.

When a Flux `Kustomization` becomes Ready, the bridge inspects deployed container images, reads standard OCI labels (with optional Kubernetes annotation overrides), and reports a successful GitHub Deployment for the discovered repository and commit - with zero per-application mapping database.

## How it works

```text
Flux Kustomization Ready=True
        ↓
Discover Deployment / StatefulSet / DaemonSet images + annotations
        ↓
Fetch OCI manifest + config labels (no layer pull)
        ↓
Resolve metadata (annotation > OCI > default)
        ↓
Authenticate as GitHub App
        ↓
Create Deployment + Status(success)
        ↓
Cache (owner, repo, environment, commit, deployment-name) to prevent duplicates
```

### OCI labels

| Label | Required | Example |
|---|---|---|
| `org.opencontainers.image.source` | yes\* | `https://github.com/example/backend` |
| `org.opencontainers.image.revision` | yes\* | `0123456789abcdef` |
| `org.opencontainers.image.version` | no | `v1.8.4` |
| `org.opencontainers.image.title` | no | `backend` |
| `org.opencontainers.image.created` | no | `2026-07-25T12:00:00Z` |

\*Unless overridden by a Kubernetes annotation. See [docs/architecture.md](docs/architecture.md#metadata-resolution).

### Kubernetes annotations (optional)

Prefix: `github-deployment-bridge.io/` — for example `environment`, `repository`, `auto-report`, `deployment-name`. Full list and priority rules: [docs/architecture.md](docs/architecture.md#kubernetes-annotations).

## Install

Full guide (GitHub App permissions, secrets, PVC, Helm values, verify):
**[docs/install.md](docs/install.md)**.

Quick start with an existing Secret:

```bash
kubectl -n flux-system create secret generic github-deployment-bridge \
  --from-literal=app-id=123456 \
  --from-literal=installation-id=987654 \
  --from-file=private-key=./github-app.pem

helm upgrade --install github-deployment-bridge \
  oci://ghcr.io/roberteggl/charts/github-deployment-bridge \
  --version 0.1.0 \
  --namespace flux-system \
  --set github.existingSecret=github-deployment-bridge \
  --set config.clusterName=production-eu \
  --set config.environment=production
```

### GitHub App (summary)

| Permission | Access |
|---|---|
| **Deployments** | Read & Write |
| **Contents** | Read |
| **Metadata** | Read |

PATs are not supported. Details: [docs/install.md#github-app-setup](docs/install.md#github-app-setup).

### Configuration

Env vars and Helm mapping: [docs/configuration.md](docs/configuration.md).

Why a PVC: SQLite at `/data/cache.db` deduplicates
`(owner, repo, environment, commitSHA, deploymentName)` across restarts - see
[docs/install.md#why-a-pvc](docs/install.md#why-a-pvc).

## Observability

| Endpoint | Purpose |
|---|---|
| `/metrics` | Prometheus metrics |
| `/healthz` | Liveness |
| `/readyz` | Readiness |

Metrics include `deployment_reports_total`, `deployment_failures_total`, `github_api_requests_total`, `github_api_latency_seconds`, `oci_requests_total`, `cache_hits_total`, and `cache_misses_total`.

Image: `ghcr.io/roberteggl/github-deployment-bridge:<version>` (multi-arch `amd64`/`arm64`, Cosign-signed, SLSA-attested).

Verify artifacts: [docs/releasing.md](docs/releasing.md#verify-signatures-and-attestations) · [Attestations](https://github.com/roberteggl/github-deployment-bridge/attestations)

## Development

```bash
make tidy test build
```

See [CONTRIBUTING.md](CONTRIBUTING.md), [docs/install.md](docs/install.md),
[docs/development.md](docs/development.md), [docs/architecture.md](docs/architecture.md),
and [docs/releasing.md](docs/releasing.md).

## Non-goals

This project does **not** reconcile GitOps state, trigger deployments, run CI, or manage image automation. It only observes successful Flux reconciliations and synchronizes deployment state into GitHub.

## Licensing

This project follows the [REUSE](https://reuse.software/) specification.
Copyright and license information is declared per file via SPDX headers. The full Apache License 2.0 text lives in
[`LICENSES/Apache-2.0.txt`](LICENSES/Apache-2.0.txt).
