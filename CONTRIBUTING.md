<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# Contributing

Thanks for helping improve **github-deployment-bridge**.

## Before you start

- Read [README.md](README.md) for project scope and non-goals.
- See [docs/architecture.md](docs/architecture.md) and
  [docs/development.md](docs/development.md) for how the controller works and
  how to build/test locally.
- Security issues: report privately via [SECURITY.md](SECURITY.md) - do not open
  a public issue for exploitable vulnerabilities.

## Development setup

Prerequisites: Go 1.25+, Make, Docker, Helm.

```bash
make tidy
make test
make build
```

Useful targets: `make vet`, `make helm-lint`, `make changelog`.

## Pull requests

1. Fork and create a topic branch from `main`.
2. Keep changes focused; prefer small PRs.
3. Add or update tests when changing behavior.
4. Ensure CI is green (`test`, `helm`, `container`, `reuse`, `CodeQL` as applicable).
5. Fill in the PR description (what/why, test plan).

### Commit messages

Use [Conventional Commits](https://www.conventionalcommits.org/) so
[git-cliff](https://git-cliff.org/) can generate release notes:

```text
feat: report deployment failures when Ready=False
fix: skip workloads with missing metadata
docs: clarify GitHub App permissions
ci: pin Scorecard action digest
chore(deps): update controller-runtime
```

Common types: `feat`, `fix`, `docs`, `test`, `refactor`, `perf`, `ci`, `chore`.

### DCO sign-off (required)

Every commit must include a `Signed-off-by` trailer (Developer Certificate of
Origin). Use `-s`:

```bash
git commit -s -m "$(cat <<'EOF'
feat: short summary.

Optional body explaining why.
EOF
)"
```

Your `user.name` and `user.email` must match the sign-off identity.

## Coding guidelines

- Idiomatic Go; keep packages small and focused.
- Prefer interfaces where they improve testability.
- Pass `context.Context` through I/O and reconcile paths.
- Handle errors explicitly; never crash the reconcile loop on bad labels.
- Prefer table-driven unit tests.
- Avoid unrelated refactors in the same PR.

### Licensing (REUSE)

This project follows [REUSE](https://reuse.software/) with Apache-2.0.
New files should carry SPDX headers (or be covered by `REUSE.toml`). Check with:

```bash
reuse lint
```

## What is in / out of scope

**In scope:** observing successful Flux reconciliations and syncing GitHub
Deployments from OCI labels and optional annotations; reliability, metrics, docs, Helm chart, CI.

**Out of scope (unless agreed):** GitOps reconciliation, image automation,
triggering deployments, or turning the bridge into a general CD orchestrator.

Future ideas (e.g. `failure` / `in_progress` statuses, HelmRelease support)
are welcome as proposals - open an issue first for larger work.

## Releases

Maintainers cut releases by tagging (`vX.Y.Z`). See
[docs/releasing.md](docs/releasing.md). Contributors do not need to tag.
