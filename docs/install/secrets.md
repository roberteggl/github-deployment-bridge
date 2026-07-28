<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# Secrets

The controller needs two credentials from the GitHub App, plus an optional
installation override. They are read from a
Kubernetes Secret with these keys:

| Secret key | Env var | Description |
|---|---|---|
| `app-id` | `GITHUB_APP_ID` | Numeric GitHub App ID |
| `installation-id` (optional) | `GITHUB_INSTALLATION_ID` | Fixed installation override; omit to resolve by repository owner |
| `private-key` | (mounted as file) | PEM private key; path set via `GITHUB_PRIVATE_KEY_PATH` |

The Helm chart mounts `private-key` at `/github/private-key.pem` and sets
`GITHUB_PRIVATE_KEY_PATH` accordingly.

## Recommended: manage the Secret yourself

```bash
kubectl -n flux-system create secret generic github-deployment-bridge \
  --from-literal=app-id=123456 \
  --from-file=private-key=./github-app.pem
```

Then install with `github.existingSecret=github-deployment-bridge`.

Rotating an `existingSecret` does not restart pods by itself (the chart cannot
checksum an external Secret). After updating keys:

```bash
kubectl -n flux-system rollout restart deployment/github-deployment-bridge
```

Or use [Reloader](https://github.com/stakater/Reloader) / a similar controller
watching the Secret.

## Alternative: let Helm create the Secret

Pass values at install time **and** opt in with `github.allowInsecureValues=true`.
Inline credentials are stored in the Helm release Secret (`sh.helm.release.v1.*`)
as well as the chart-managed Secret - fine for local/dev; prefer an external
Secret manager or sealed/SOPS secret in production:

```bash
helm upgrade --install github-deployment-bridge \
  oci://ghcr.io/roberteggl/charts/github-deployment-bridge \
  --version 1.4.0 \
  --namespace flux-system \
  --set github.allowInsecureValues=true \
  --set github.appId=123456 \
  --set-file github.privateKey=./github-app.pem \
  --set config.clusterName=production-eu \
  --set config.environment=production
```

Add `--from-literal=installation-id=987654` or
`--set github.installationId=987654` only when all repositories must use one
specific installation. Existing secrets containing that key remain supported.

Next: [PVC](./persistence.md) · [Install with Helm](./helm.md)
