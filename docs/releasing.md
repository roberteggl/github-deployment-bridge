<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# Releasing

Releases are fully automated on `v*` tag push via
[`.github/workflows/release.yml`](../.github/workflows/release.yml).

## Pipeline

1. **Validate** — tidy, fmt, vet, race tests, build, Helm lint, chart/tag version match, REUSE
2. **Changelog** — [git-cliff](https://git-cliff.org/) from Conventional Commits (`cliff.toml`)
3. **Image** — native multi-arch build (no QEMU):
   - `linux/amd64` on `ubuntu-24.04`
   - `linux/arm64` on `ubuntu-24.04-arm`
   - push-by-digest → merge manifest list on GHCR
   - BuildKit SBOM + provenance
   - Sigstore keyless Cosign signature
   - GitHub Artifact Attestation (SLSA provenance, pushed to GHCR)
4. **Helm** — package chart, attest `.tgz` + OCI chart, push to `oci://ghcr.io/<owner>/charts`
5. **GitHub Release** — cliff notes + verify commands (chart `.tgz` attached)

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

```bash
# Cosign (image signature)
cosign verify \
  --certificate-identity-regexp='https://github.com/roberteggl/github-deployment-bridge/.*' \
  --certificate-oidc-issuer=https://token.actions.githubusercontent.com \
  ghcr.io/roberteggl/github-deployment-bridge:0.1.0

# GitHub Artifact Attestations (image)
gh attestation verify \
  oci://ghcr.io/roberteggl/github-deployment-bridge:0.1.0 \
  --repo roberteggl/github-deployment-bridge

# GitHub Artifact Attestations (Helm chart asset from the release)
gh attestation verify github-deployment-bridge-0.1.0.tgz \
  --repo roberteggl/github-deployment-bridge
```
