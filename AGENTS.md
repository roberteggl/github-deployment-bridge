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
2. Reads OCI labels (`source`, `revision`, `version`) — no layer pulls
3. Authenticates as a GitHub App (never a PAT)
4. Creates a GitHub Deployment + `success` status
5. Deduplicates via SQLite on `(owner, repo, environment, commitSHA)`

Stack: Go 1.25+, chi, slog, controller-runtime, go-github, go-containerregistry,
modernc SQLite, Prometheus, distroless image, Helm chart.

Non-goals: GitOps reconciliation, image automation, CI triggering, or deployment
orchestration. Observe successful Flux reconciliations only.

## Documentation

Read these before making non-trivial changes:

| Doc | Contents |
|---|---|
| [README.md](README.md) | Overview, install, GitHub App setup |
| [docs/architecture.md](docs/architecture.md) | Event flow, workload discovery, OCI labels |
| [docs/configuration.md](docs/configuration.md) | Env vars, permissions, private registries |
| [docs/development.md](docs/development.md) | Build, test, local run, integration tests |

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
