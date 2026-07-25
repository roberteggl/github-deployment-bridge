---
pageType: home
title: GitHub Deployment Bridge
titleSuffix: FluxCD to GitHub Deployments
hero:
  name: GitHub Deployment Bridge
  text: Flux state, mirrored to GitHub Deployments
  tagline: A Kubernetes controller that watches Flux Kustomizations and HelmReleases and reports lifecycle to the GitHub Deployments API — observe only, never orchestrate.
  actions:
    - theme: brand
      text: Install
      link: /install/
    - theme: alt
      text: Architecture
      link: /architecture
features:
  - title: Observe, don't orchestrate
    details: Derives queued, in_progress, success, and failure from Flux conditions. No GitOps reconciliation, no deploy triggers.
    icon: "01"
    span: 6
  - title: OCI identity, annotation overrides
    details: Resolves workload images from inventory, reads OCI labels without pulling layers, and merges optional github-deployment-bridge.io annotations.
    icon: "02"
    span: 6
  - title: GitHub App, full lifecycle
    details: Authenticates as a GitHub App (never a PAT). One Deployment per key; statuses update through the reconcile, and prior successes go inactive on a newer commit.
    icon: "03"
    span: 6
  - title: Idempotent under reconcile churn
    details: SQLite deduplicates on owner, repo, environment, commit, and deployment name. Crash recovery reuses existing Deployments when the cache holds a provisional row.
    icon: "04"
    span: 6
---
