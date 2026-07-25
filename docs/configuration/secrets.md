<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# Secrets

Credentials come from a Kubernetes Secret (never a PAT). Required keys:

| Key | Used as |
|---|---|
| `app-id` | `GITHUB_APP_ID` |
| `installation-id` | `GITHUB_INSTALLATION_ID` |
| `private-key` | PEM file mounted at `/github/private-key.pem` |

Create it yourself and set `github.existingSecret`, or pass `github.appId`,
`github.installationId`, and `github.privateKey` so the chart creates the
Secret. Prefer an externally managed Secret in production.

```bash
kubectl -n flux-system create secret generic github-deployment-bridge \
  --from-literal=app-id=123456 \
  --from-literal=installation-id=987654 \
  --from-file=private-key=./github-app.pem
```

Step-by-step setup: [Install → Secrets](../install/secrets.md)
