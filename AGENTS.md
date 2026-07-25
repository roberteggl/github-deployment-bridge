<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# Agent notes

## Project

**github-deployment-bridge** is a lightweight Go Kubernetes controller that
bridges FluxCD reconciliations to the GitHub Deployments API.

When a Flux `Kustomization` becomes Ready, the bridge:

1. Discovers workload images (`Deployment`, `StatefulSet`, `DaemonSet`)
2. Reads OCI labels (`source`, `revision`, optional `version`/`title`/`created`) - no layer pulls
3. Merges optional `github-deployment-bridge.io/*` annotations (annotation > OCI > default)
4. Authenticates as a GitHub App (never a PAT)
5. Creates a GitHub Deployment + `success` status
6. Deduplicates via SQLite on `(owner, repo, environment, commitSHA, deploymentName)`

Stack: Go 1.25+, chi, slog, controller-runtime, go-github, go-containerregistry,
modernc SQLite, Prometheus, distroless image, Helm chart.

Non-goals: GitOps reconciliation, image automation, CI triggering, or deployment
orchestration. Observe successful Flux reconciliations only.

## Documentation

Read these before making non-trivial changes:

| Doc | Contents |
|---|---|
| [README.md](README.md) | Overview, quick start |
| [docs/install.md](docs/install.md) | Cluster install, GitHub App, secrets, PVC |
| [docs/configuration.md](docs/configuration.md) | Env vars, Helm values, permissions, registries |
| [docs/architecture.md](docs/architecture.md) | Event flow, workload discovery, metadata (OCI + annotations) |
| [CONTRIBUTING.md](CONTRIBUTING.md) | DCO, Conventional Commits, PR process |
| [docs/development.md](docs/development.md) | Build, test, local run, integration tests |
| [docs/releasing.md](docs/releasing.md) | Tag-driven release pipeline (git-cliff, GHCR, Helm) |

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

Use [Conventional Commits](https://www.conventionalcommits.org/) (`feat:`,
`fix:`, `docs:`, `ci:`, …) so [git-cliff](https://git-cliff.org/) can generate
release notes.

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
