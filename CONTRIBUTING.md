<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# Contributing

Thanks for helping improve **GitHub Deployment Bridge**.

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

Useful targets: `make lint`, `make govulncheck`, `make vet`, `make helm-lint`,
`make changelog`.

## Pull requests

1. Fork and create a topic branch from `main`.
2. Keep changes focused; prefer small PRs.
3. Add or update tests when changing behavior.
4. Ensure CI is green (`test`, `lint`, `govulncheck`, `helm`, `container`/Trivy,
   `reuse`, `CodeQL` as applicable).
5. Fill in the PR description (what/why, test plan).

### Commit messages

Use [Conventional Commits](https://www.conventionalcommits.org/) so
[git-cliff](https://git-cliff.org/) can group release notes by **type**. Format:
`<type>(optional-scope): <description>`.

**Always put the change kind in the type**, never as the scope. git-cliff matches
`^feat`, `^fix`, `^doc`, … — so `fix(docs): …` lands under **Bug Fixes**, not
**Documentation**.

```text
feat(github): report deployment failures when Ready=False
fix(cache): skip workloads with missing metadata
docs(install): clarify GitHub App permissions
ci: pin Scorecard action digest
chore(deps): update controller-runtime
```

| Kind of change | Type |
|---|---|
| New user-facing capability | `feat` |
| Bug fix in product code | `fix` |
| Docs / README / comments only | `docs` |
| CI / workflows | `ci` |
| Tests only | `test` |
| Internal restructure (no behavior change) | `refactor` |
| Perf | `perf` |
| Build, release prep, tooling, deps | `chore` |

Scope (if used) is an **area** (`github`, `cache`, `install`, `helm`, …), not
another type name. Prefer the most specific type that fits — use `docs:` for
doc-only changes, not `fix(docs):` or `chore(docs):`.

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

**In scope:** observing Flux `Kustomization` / `HelmRelease` reconciliations and
syncing the GitHub Deployments lifecycle from OCI labels and optional annotations;
reliability, metrics, docs, Helm chart, CI.

**Out of scope (unless agreed):** GitOps reconciliation, image automation,
triggering deployments, or turning the bridge into a general CD orchestrator.

Larger design changes are welcome as proposals - open an issue first.

## Releases

Maintainers cut releases by tagging (`vX.Y.Z`). See
[docs/releasing.md](docs/releasing.md). Contributors do not need to tag.
