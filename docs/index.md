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
  - title: OCI-native identity
    details: Discovers workload images from inventory, reads OCI labels (no layer pulls), and merges optional Kubernetes annotation overrides.
    icon: "▣"
  - title: GitHub App auth
    details: Authenticates as a GitHub App (never a PAT), creates one Deployment per key, and updates statuses through the full lifecycle.
    icon: "◇"
  - title: Idempotent by design
    details: SQLite deduplicates on owner, repo, environment, commit, and deployment name — and marks prior successes inactive when a newer commit lands.
    icon: "⬡"
---
