<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# GitHub App setup

Personal access tokens are **not** supported. Create a GitHub App and install it
on every repository whose deployments you want reported.

## 1. Create the App

In GitHub: **Settings → Developer settings → GitHub Apps → New GitHub App**.

Suggested settings:

| Field | Value |
|---|---|
| GitHub App name | e.g. `flux-deployment-bridge` |
| Homepage URL | your org / docs URL |
| Webhook | **Inactive** (the bridge polls the cluster; no webhook needed) |
| Where can this GitHub App be installed? | Only on this account / org |

## 2. Repository permissions

Grant **only** these repository permissions:

| Permission | Access | Why |
|---|---|---|
| **Deployments** | Read & Write | Create Deployments and lifecycle statuses |
| **Contents** | Read | Resolve the commit SHA / ref when creating a Deployment |
| **Metadata** | Read | Required baseline for GitHub Apps (repo identity) |

No other permissions (Issues, Pull requests, Actions, Administration, …) are
needed.

## 3. Generate a private key

Under the App's **Private keys**, generate a key and download the `.pem` file.
Store it securely; the chart mounts it into the pod as a Secret.

## 4. Install the App

Install the App on every target organization or user account and grant it access
to the repositories the bridge may report. The bridge uses App authentication to
list the App's installations and selects the installation whose account login
matches the repository owner discovered from workload metadata. This App-level
installation listing does not require an additional repository or organization
permission.

Then note:

| Value | Where to find it |
|---|---|
| **App ID** | App settings → **About** → App ID |
| **Installation ID** (optional) | URL `…/installations/<id>`; set only to force one installation for backwards compatibility |
| **Private key** | The downloaded `.pem` |

For GitHub Enterprise Server, also set `config.githubBaseURL` (or
`GITHUB_BASE_URL`) to your instance base URL. Automatic resolution uses that
same server's App and installation APIs.

Next: [Secrets](./secrets.md)
