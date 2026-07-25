<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# Uninstall

```bash
helm -n flux-system uninstall github-deployment-bridge
# PVC may remain depending on reclaim policy; delete if you no longer need the cache:
kubectl -n flux-system delete pvc -l app.kubernetes.io/name=github-deployment-bridge
```
