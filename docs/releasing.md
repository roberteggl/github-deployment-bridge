<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# Releasing

Releases are fully automated on `v*` tag push via
[`.github/workflows/release.yml`](https://github.com/roberteggl/github-deployment-bridge/blob/main/.github/workflows/release.yml).

## Pipeline

1. **Validate** - tidy, fmt, vet, golangci-lint, govulncheck, race tests, build, Helm lint, chart/tag version match, REUSE
2. **Changelog** - [git-cliff](https://git-cliff.org/) from Conventional Commits (`cliff.toml`)
3. **Image** - native multi-arch build (no QEMU):
   - `linux/amd64` on `ubuntu-24.04`
   - `linux/arm64` on `ubuntu-24.04-arm`
   - Trivy scan of each digest (`HIGH`/`CRITICAL`, ignore unfixed) before merge
   - push-by-digest → merge manifest list on GHCR
   - BuildKit SBOM + provenance
   - Sigstore keyless Cosign signature (legacy `sha256-*.sig` tags; Cosign v3
     defaults break Artifact Hub signature detection)
   - GitHub Artifact Attestation (SLSA provenance, pushed to GHCR)
4. **Helm** - package chart, push to `oci://ghcr.io/<owner>/charts`, Cosign-sign
   the chart (Artifact Hub Signed badge), attest `.tgz` + OCI chart, and push
   [`artifacthub-repo.yml`](https://github.com/roberteggl/github-deployment-bridge/blob/main/artifacthub-repo.yml) as the
   `artifacthub.io` OCI tag (Artifact Hub ownership / verified publisher)
5. **GitHub Release** - cliff notes + verify commands; attaches chart `.tgz` plus the
   Sigstore attestation bundle as `.sigstore.json` and `.intoto.jsonl` (OpenSSF
   Scorecard Signed-Releases looks at those release-asset extensions)

Prerelease tags containing `-` (e.g. `v0.2.0-rc.1`) skip `latest` and major tags.

## Before tagging

1. Bump chart versions to the release version (no `v` prefix):

   ```yaml
   # charts/github-deployment-bridge/Chart.yaml
   version: 0.1.0
   appVersion: "0.1.0"
   ```

2. Prefer [Conventional Commits](https://www.conventionalcommits.org/) on `main`
   so git-cliff can group release notes (`feat:`, `fix:`, `docs:`, …).

3. Preview notes locally (optional):

   ```bash
   git cliff --unreleased
   # or for the next tag:
   git cliff --tag v0.1.0 --strip all
   ```

4. Commit with DCO sign-off (`git commit -s`), then tag and push:

   ```bash
   git tag v0.1.0
   git push origin v0.1.0
   ```

The release job uses the GitHub **production** environment. Create it under
repository Settings → Environments if it does not exist yet (optional
protection rules / reviewers).

## Install a release

```bash
# Container
docker pull ghcr.io/roberteggl/github-deployment-bridge:0.1.0

# Helm (OCI)
helm install github-deployment-bridge \
  oci://ghcr.io/roberteggl/charts/github-deployment-bridge \
  --version 0.1.0 \
  --namespace flux-system \
  --set config.clusterName=production \
  --set config.environment=production \
  --set github.existingSecret=github-deployment-bridge
```

## Verify signatures and attestations

Attestations appear under
[github.com/roberteggl/github-deployment-bridge/attestations](https://github.com/roberteggl/github-deployment-bridge/attestations).

Each GitHub Release also attaches the Helm chart Sigstore bundle next to the
`.tgz` (same attestation, two Scorecard-recognized filenames):

- `github-deployment-bridge-<version>.tgz.sigstore.json`
- `github-deployment-bridge-<version>.tgz.intoto.jsonl`

Prefer `gh attestation verify` against the chart (or Cosign against the image /
chart OCI refs). The release-side copies exist so [OpenSSF Scorecard
Signed-Releases](https://github.com/ossf/scorecard/blob/main/docs/checks.md#signed-releases)
can see signature/provenance assets; Scorecard does not yet consume the GitHub
Attestations API.

Cosign v3 defaults to the new Sigstore bundle format stored via the OCI
referrers API. Artifact Hub still indexes the legacy `sha256-<digest>.sig` tag
layout, so the release workflow signs with `--new-bundle-format=false
--use-signing-config=false`.

## Artifact Hub

Package page:
[artifacthub.io/packages/helm/github-deployment-bridge/github-deployment-bridge](https://artifacthub.io/packages/helm/github-deployment-bridge/github-deployment-bridge).

The OCI Helm chart can be listed on [Artifact Hub](https://artifacthub.io/).
Each release pushes repository metadata from
[`artifacthub-repo.yml`](https://github.com/roberteggl/github-deployment-bridge/blob/main/artifacthub-repo.yml) to:

```text
ghcr.io/roberteggl/charts/github-deployment-bridge:artifacthub.io
```

### One-time setup

1. Sign in to Artifact Hub and add a Helm repository with URL
   `oci://ghcr.io/roberteggl/charts/github-deployment-bridge`
   (repository ID `8a8ee5d5-8afa-41fc-9a95-5d51e6ccd952` is already set in
   `artifacthub-repo.yml`).
2. Push the metadata (each release does this automatically), or manually / via
   **Actions → Artifact Hub metadata → Run workflow**:

```bash
oras push \
  ghcr.io/roberteggl/charts/github-deployment-bridge:artifacthub.io \
  --config /dev/null:application/vnd.cncf.artifacthub.config.v1+yaml \
  artifacthub-repo.yml:application/vnd.cncf.artifacthub.repository-metadata.layer.v1.yaml
```

Ownership claim uses the `owners` entry (`Robert Eggl` /
`robert@eggl.dev`) and is processed immediately once the metadata is visible.

```bash
# Cosign (image signature)
cosign verify \
  --certificate-identity-regexp='https://github.com/roberteggl/github-deployment-bridge/.*' \
  --certificate-oidc-issuer=https://token.actions.githubusercontent.com \
  ghcr.io/roberteggl/github-deployment-bridge:0.1.0

# Cosign (Helm chart OCI signature - Artifact Hub Signed badge)
cosign verify \
  --certificate-identity-regexp='https://github.com/roberteggl/github-deployment-bridge/.*' \
  --certificate-oidc-issuer=https://token.actions.githubusercontent.com \
  ghcr.io/roberteggl/charts/github-deployment-bridge:0.1.0

# GitHub Artifact Attestations (image)
gh attestation verify \
  oci://ghcr.io/roberteggl/github-deployment-bridge:0.1.0 \
  --repo roberteggl/github-deployment-bridge

# GitHub Artifact Attestations (Helm chart asset from the release)
gh attestation verify github-deployment-bridge-0.1.0.tgz \
  --repo roberteggl/github-deployment-bridge
```
