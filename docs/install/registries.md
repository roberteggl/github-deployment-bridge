<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# Private image registries

The bridge only fetches the image **manifest and config blob** (never layers).
Private registries (including private GHCR packages) need a Docker config
Secret mounted into the pod - typically the same `kubernetes.io/dockerconfigjson`
Secret Flux already uses for image pulls:

```yaml
registry:
  existingDockerConfigSecret: ghcr-pull-secret
```

```bash
helm upgrade --install github-deployment-bridge \
  oci://ghcr.io/roberteggl/charts/github-deployment-bridge \
  --namespace flux-system \
  --set github.existingSecret=github-deployment-bridge \
  --set config.clusterName=production-eu \
  --set config.environment=production \
  --set registry.existingDockerConfigSecret=ghcr-pull-secret
```

`imagePullSecrets` on this chart only pull the bridge image itself; they do not
authenticate OCI label lookups. Details:
[Configuration → Private registries](../configuration/registries.md).
