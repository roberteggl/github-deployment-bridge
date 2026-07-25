<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# Development

## Prerequisites

- Go 1.25+
- Docker
- Helm 3

## Build & test

```bash
make tidy
make test
make build
```

## Documentation site

Docs are built with [Rspress](https://rspress.rs/) from this `docs/` tree and
published to GitHub Pages on pushes to `main`.

```bash
npm ci
npm run dev      # local preview
npm run build    # output in doc_build/
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

Integration tests under `test/integration` exercise the reporter lifecycle
against fakes (SQLite cache, registry, GitHub API). They run with the rest of
the suite:

```bash
go test ./test/integration -count=1
```

## Container image

```bash
make docker-build IMG=ghcr.io/roberteggl/github-deployment-bridge VERSION=dev
```

## Dependency updates

[Renovate](https://docs.renovatebot.com/) is configured via
[`renovate.json`](https://github.com/roberteggl/github-deployment-bridge/blob/main/renovate.json).
It opens `chore(deps):` PRs labeled `dependencies`, with automerge for safe
non-major updates (Go modules, Actions, Docker digests).

## Releasing

See [releasing.md](releasing.md). Preview changelog with:

```bash
git cliff --unreleased
```
