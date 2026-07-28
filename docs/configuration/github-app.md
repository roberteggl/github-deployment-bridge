<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# GitHub App permissions

| Permission | Access | Why |
|---|---|---|
| **Deployments** | Read & Write | Create Deployments and lifecycle status updates |
| **Contents** | Read | Resolve the commit SHA / ref for a Deployment |
| **Metadata** | Read | Baseline App access to repository metadata |

Webhook: inactive. PATs are not supported.

The App must be installed on each target repository owner. By default the bridge
lists installations using App JWT authentication, matches the repository owner,
and caches the installation ID in SQLite for one hour. This lookup requires no
additional configured App permission. `GITHUB_INSTALLATION_ID` remains available
as a fixed backwards-compatible override.

Create and install the App: [Install → GitHub App setup](../install/github-app.md).
