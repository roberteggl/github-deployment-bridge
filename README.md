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

When a Flux `Kustomization` becomes Ready, the bridge inspects deployed container images, reads standard OCI labels, and reports a successful GitHub Deployment for the discovered repository and commit - with zero per-application mapping.

## How it works

```text
Flux Kustomization Ready=True
        ↓
Discover Deployment / StatefulSet / DaemonSet images
        ↓
Fetch OCI manifest + config labels (no layer pull)
        ↓
Authenticate as GitHub App
        ↓
Create Deployment + Status(success)
        ↓
Cache (owner, repo, environment, commit) to prevent duplicates
```

Required image labels:

| Label | Example |
|---|---|
| `org.opencontainers.image.source` | `https://github.com/example/backend` |
| `org.opencontainers.image.revision` | `0123456789abcdef` |
| `org.opencontainers.image.version` | `v1.8.4` |

## Install with Helm

```bash
helm upgrade --install github-deployment-bridge \
  charts/github-deployment-bridge \
  --namespace flux-system \
  --set config.clusterName=production-eu \
  --set config.environment=production \
  --set config.environmentURL=https://app.example.com \
  --set config.logURLTemplate='https://grafana.example.com/explore?commit={sha}' \
  --set github.appId=123456 \
  --set github.installationId=987654 \
  --set-file github.privateKey=./github-app.pem
```

Or reference an existing secret:

```bash
kubectl -n flux-system create secret generic github-deployment-bridge \
  --from-literal=app-id=123456 \
  --from-literal=installation-id=987654 \
  --from-file=private-key=./github-app.pem

helm upgrade --install github-deployment-bridge charts/github-deployment-bridge \
  --namespace flux-system \
  --set github.existingSecret=github-deployment-bridge \
  --set config.clusterName=production-eu \
  --set config.environment=production
```

## GitHub App setup

Create a GitHub App with:

- **Deployments**: Read & Write
- **Metadata**: Read
- **Contents**: Read

Install it on the repositories you deploy, then configure:

- `GITHUB_APP_ID`
- `GITHUB_INSTALLATION_ID`
- `GITHUB_PRIVATE_KEY_PATH`

Personal access tokens are not supported.

## Configuration

See [docs/configuration.md](docs/configuration.md).

Key settings:

```yaml
clusterName: production-eu
environment: production
watchNamespace: flux-system
database: /data/cache.db
environmentURL: https://app.example.com
logURLTemplate: https://grafana.example.com/explore?commit={sha}
```

## Observability

| Endpoint | Purpose |
|---|---|
| `/metrics` | Prometheus metrics |
| `/healthz` | Liveness |
| `/readyz` | Readiness |

Metrics include `deployment_reports_total`, `deployment_failures_total`, `github_api_requests_total`, `github_api_latency_seconds`, `oci_requests_total`, `cache_hits_total`, and `cache_misses_total`.

## Install from a release

```bash
helm install github-deployment-bridge \
  oci://ghcr.io/roberteggl/charts/github-deployment-bridge \
  --version 0.1.0 \
  --namespace flux-system \
  --set config.clusterName=production-eu \
  --set config.environment=production \
  --set github.existingSecret=github-deployment-bridge
```

Image: `ghcr.io/roberteggl/github-deployment-bridge:<version>` (multi-arch `amd64`/`arm64`, Cosign-signed, SLSA-attested).

Verify: [docs/releasing.md](docs/releasing.md#verify-signatures-and-attestations) · [Attestations](https://github.com/roberteggl/github-deployment-bridge/attestations)

## Development

```bash
make tidy test build
```

See [CONTRIBUTING.md](CONTRIBUTING.md), [docs/development.md](docs/development.md),
[docs/architecture.md](docs/architecture.md), and [docs/releasing.md](docs/releasing.md).

## Non-goals

This project does **not** reconcile GitOps state, trigger deployments, run CI, or manage image automation. It only observes successful Flux reconciliations and synchronizes deployment state into GitHub.

## Licensing

This project follows the [REUSE](https://reuse.software/) specification.
Copyright and license information is declared per file via SPDX headers. The full Apache License 2.0 text lives in
[`LICENSES/Apache-2.0.txt`](LICENSES/Apache-2.0.txt).
