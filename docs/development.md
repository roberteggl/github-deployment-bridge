<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# Development

## Prerequisites

- Go 1.25+
- Docker
- Helm 3
- `kubectl` / `kind` (for integration tests)

## Build & test

```bash
make tidy
make test
make build
```

## Run locally

```bash
export CLUSTER_NAME=dev
export ENVIRONMENT=staging
export GITHUB_APP_ID=123456
export GITHUB_INSTALLATION_ID=987654
export GITHUB_PRIVATE_KEY_PATH=./github-app.pem
export DATABASE=./cache.db
export KUBECONFIG=~/.kube/config

go run ./cmd/bridge
```

## Integration tests

Integration tests under `test/integration` expect:

1. A `kind` cluster with Flux installed
2. A local registry (or GHCR credentials)
3. A fake GitHub API (or recorded fixtures)

```bash
go test ./test/integration -tags=integration -count=1
```

## Container image

```bash
make docker-build IMG=ghcr.io/roberteggl/github-deployment-bridge VERSION=dev
```

## Dependency updates

[Renovate](https://docs.renovatebot.com/) is configured via [`renovate.json`](../renovate.json).
It opens `chore(deps):` PRs labeled `dependencies`, with automerge for safe
non-major updates (Go modules, Actions, Docker digests).

## Releasing

See [releasing.md](releasing.md). Preview changelog with:

```bash
git cliff --unreleased
```
