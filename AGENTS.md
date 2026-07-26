<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# Agent notes

## Project

**GitHub Deployment Bridge** (`github-deployment-bridge`) is a lightweight Go
Kubernetes controller that
bridges FluxCD reconciliations to the GitHub Deployments API.

When a Flux `Kustomization` or `HelmRelease` reconciles, the bridge:

1. Derives a deployment phase (`queued` / `in_progress` / `success` / `failure`)
2. Discovers workload images from `.status.inventory` (`Deployment`, `StatefulSet`, `DaemonSet`)
3. Reads OCI labels (`source`, `revision`, optional `version`/`title`/`created`) - no layer pulls
4. Merges `github-deployment-bridge.io/*` annotations (opt-in via `auto-report=true`; annotation > OCI > default)
5. Authenticates as a GitHub App (never a PAT)
6. Creates a GitHub Deployment once and updates Deployment Status through the lifecycle
7. Deduplicates via SQLite on `(owner, repo, environment, commitSHA, deploymentName)` + latest status
8. Marks prior successful deployments `inactive` when a newer commit succeeds

Stack: Go 1.25+, chi, slog, controller-runtime, go-github, go-containerregistry,
modernc SQLite, Prometheus, distroless image, Helm chart.

Non-goals: GitOps reconciliation, image automation, CI triggering, or deployment
orchestration. Observe Flux reconciliations and report GitHub Deployment state only.

## Documentation

Read these before making non-trivial changes:

| Doc | Contents |
|---|---|
| [README.md](README.md) | Overview, quick start |
| [docs/install/](docs/install/) | Cluster install, GitHub App, secrets, PVC, Helm |
| [docs/configuration/](docs/configuration/) | Env vars, Helm values, permissions, registries, metrics |
| [docs/operations/](docs/operations/) | Grafana dashboard, alert runbook (optional PrometheusRule) |
| [docs/architecture.md](docs/architecture.md) | Event flow, lifecycle, workload discovery, metadata (OCI + annotations) |
| [CONTRIBUTING.md](CONTRIBUTING.md) | DCO, Conventional Commits, PR process |
| [docs/development.md](docs/development.md) | Build, test, local run, integration tests, Rspress docs |
| [docs/releasing.md](docs/releasing.md) | Tag-driven release pipeline (git-cliff, GHCR, Helm) |

Published docs site: Rspress (`rspress.config.ts`) → GitHub Pages via
[`.github/workflows/docs.yml`](.github/workflows/docs.yml). Local: `npm ci && npm run dev`.

## Commits (DCO / `-s` required)

All commits **must** be signed off with `git commit -s` (Developer Certificate of
Origin). This adds a `Signed-off-by:` trailer matching the committer identity.

```bash
git commit -s -m "$(cat <<'EOF'
Commit subject.

Optional body.
EOF
)"
```

Do not use `--no-gpg-sign` / `--no-verify` unless the user explicitly asks.
Do not commit unless the user asks. Push only when requested.

### Conventional Commits (type is required; scope is optional)

Use [Conventional Commits](https://www.conventionalcommits.org/) so
[git-cliff](https://git-cliff.org/) (`cliff.toml`) can group release notes by
**type**. Format: `<type>(optional-scope): <description>`.

**Always put the change kind in the type**, never as the scope. git-cliff
matches `^feat`, `^fix`, `^doc`, … — so `fix(docs): …` lands under **Bug Fixes**
(with a `docs` scope label), not under **Documentation**.

| Kind of change | Type | Example |
|---|---|---|
| New user-facing capability | `feat` | `feat(github): mark prior deployments inactive` |
| Bug fix in product code | `fix` | `fix(cache): reuse provisional deployment row` |
| Docs / README / comments only | `docs` | `docs(install): clarify GitHub App permissions` |
| CI / workflows | `ci` | `ci: pin Scorecard action digest` |
| Tests only | `test` | `test(reconciler): cover Ready=False failure` |
| Internal restructure (no behavior change) | `refactor` | `refactor(oci): simplify label merge` |
| Perf | `perf` | `perf(inventory): skip duplicate image lookups` |
| Build, release prep, tooling, deps | `chore` | `chore(deps): update controller-runtime` |

Rules:

- Prefer the most specific type that fits (`docs` for doc-only; `fix` only when
  fixing a product bug).
- Scope (if used) is an **area** (`github`, `cache`, `install`, `helm`, …), not
  another type name.
- **Wrong:** `fix(docs): …`, `feat(ci): …`, `chore(docs): …` when the change is
  really docs/CI — those pollute the wrong changelog section.
- **Right:** `docs: …`, `docs(install): …`, `ci: …`, `fix(github): …`.
- Do not invent types outside the table above unless the user asks.

## OpenSSF Scorecard

Weekly + `main` push analysis via [`.github/workflows/scorecard.yml`](.github/workflows/scorecard.yml).
Results publish to the Scorecard API (README badge) and GitHub code scanning
(`category: scorecard`). Keep that workflow free of top-level `env`/`defaults`
and workflow-level write permissions so `publish_results: true` stays valid.

## Dependency updates (Renovate)

Configured in [`renovate.json`](renovate.json) (`config:best-practices`):

- Semantic commits: `chore(deps): …`
- `gomodTidy` after Go bumps
- Automerge: Go non-major, GitHub Actions, Docker patch/minor/digest
- Grouped majors for Kubernetes + Flux libraries
- Custom manager for Helm CLI pins (`# helm` in workflows)

Enable the [Renovate GitHub App](https://github.com/apps/renovate) on this
repository if it is not already installed for the org/user.

## Releasing

Automated on `v*` tag push (see [docs/releasing.md](docs/releasing.md)):

1. Validate (tests, Helm, chart version == tag, REUSE)
2. Changelog via git-cliff (`cliff.toml`)
3. Native multi-arch image (`ubuntu-24.04` + `ubuntu-24.04-arm`, no QEMU) → GHCR
4. Cosign keyless sign + BuildKit SBOM/provenance
5. GitHub Artifact Attestations (`actions/attest-build-provenance`) for image + Helm chart
6. Helm chart → `oci://ghcr.io/<owner>/charts`
7. GitHub Release (`softprops/action-gh-release`)

```bash
# Bump charts/github-deployment-bridge/Chart.yaml version + appVersion first
git commit -s -m "chore(release): prepare v0.1.0"
git tag v0.1.0
git push origin v0.1.0
```

## REUSE licensing

This project follows the [REUSE](https://reuse.software/) specification with
Apache License 2.0. Keep files compliant when adding or editing code.

### Lint

Check compliance from the repository root:

```bash
reuse lint
```

### Annotate

Add copyright and license headers to files:

```bash
reuse annotate \
  --copyright="Robert Eggl <robert@eggl.dev>" \
  --license=Apache-2.0 \
  --year=2026 \
  path/to/file
```

For a directory tree:

```bash
reuse annotate \
  --copyright="Robert Eggl <robert@eggl.dev>" \
  --license=Apache-2.0 \
  --year=2026 \
  --recursive \
  --skip-existing \
  --fallback-dot-license \
  path/to/dir
```

- `--skip-existing` leaves files that already have REUSE info alone.
- `--fallback-dot-license` writes a `.license` sidecar for unrecognized comment styles.
  Prefer SPDX headers in source files; avoid committing `.license` sidecars under
  `charts/*/templates/` (Helm rejects unknown extensions).
