<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# Private registries

The bridge reads OCI labels via `authn.DefaultKeychain` - **not** via
`imagePullSecrets` (those only pull this chart's own image).

For private registries (including private GHCR packages), mount a
`kubernetes.io/dockerconfigjson` Secret - typically the same one Flux already
uses:

```yaml
registry:
  existingDockerConfigSecret: ghcr-pull-secret
  # dockerConfigKey: ".dockerconfigjson"   # default
```

The chart mounts the key as `/var/run/secrets/docker/config.json` and sets
`DOCKER_CONFIG=/var/run/secrets/docker`. Only the manifest and config blob are
fetched; layers are never pulled.

GHCR credentials need `read:packages`. The Deployments GitHub App is not used
for registry auth.

Helm install example: [Install → Private image registries](../install/registries.md).
