---
pageType: home
title: GitHub Deployment Bridge
titleSuffix: FluxCD to GitHub Deployments
hero:
  name: GitHub Deployment Bridge
  text: Flux reconciliations, reported as GitHub Deployments
  tagline: A lightweight Kubernetes controller that observes Flux Kustomizations and HelmReleases and mirrors lifecycle state to the GitHub Deployments API.
  actions:
    - theme: brand
      text: Install
      link: /install/
    - theme: alt
      text: Architecture
      link: /architecture
features:
  - title: Observe, don't orchestrate
    details: Watches Flux conditions and reports queued, in_progress, success, and failure — without reconciling GitOps state or triggering deploys.
    icon: "◎"
    span: 4
  - title: OCI-native identity
    details: Discovers workload images from inventory, reads OCI labels (no layer pulls), and merges optional Kubernetes annotation overrides.
    icon: "▣"
    span: 4
  - title: GitHub App auth
    details: Authenticates as a GitHub App (never a PAT), creates one Deployment per key, and updates statuses through the full lifecycle.
    icon: "◇"
    span: 4
  - title: Idempotent by design
    details: SQLite deduplicates on owner, repo, environment, commit, and deployment name; crash recovery reuses existing GitHub Deployments when the cache has a provisional row — and marks prior successes inactive when a newer commit lands.
    icon: "⬡"
    span: 4
  - title: Flux Kustomization & HelmRelease
    details: Reacts to condition and revision changes on Flux sources, including HelmRelease inventory on Flux ≥ 2.8.
    icon: "△"
    span: 4
  - title: Built for operators
    details: Helm chart, Prometheus metrics, GitHub Enterprise base URL, and dockerconfig auth for private registries.
    icon: "▢"
    span: 4
---
