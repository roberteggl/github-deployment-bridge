<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# Secrets

Credentials come from a Kubernetes Secret (never a PAT):

| Key | Used as |
|---|---|
| `app-id` | `GITHUB_APP_ID` |
| `installation-id` | `GITHUB_INSTALLATION_ID` |
| `private-key` | PEM mounted at `/github/private-key.pem` |

Prefer `github.existingSecret`. Inline `github.appId` / `installationId` /
`privateKey` require `github.allowInsecureValues=true` and are also stored in
Helm release history - fine for local/dev only.

Chart-managed Secrets roll the Deployment via `checksum/secret`. With
`existingSecret`, restart the Deployment (or Reloader) after rotating keys.

Create and wire the Secret: [Install → Secrets](../install/secrets.md).
