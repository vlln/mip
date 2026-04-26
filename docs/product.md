# Product Brief

## Problem

Pulling public container images from mainland China is often slow or unreliable. A developer may need images from Docker Hub, GHCR, Quay, MCR, registry.k8s.io, GCR, Elastic, or NVCR, each with different mirror rules and availability patterns.

Existing approaches usually fall into one of two categories:

- Configure Docker daemon mirrors, which mainly helps Docker Hub and does not generalize cleanly to other registries.
- Try a flat list of mirror hosts, which ignores registry-specific naming, authentication, cache behavior, and failure modes.

## Goal

Build a modern Unix-style CLI that accelerates image pulls without taking over the user's container engine configuration.

The tool should:

- Accept normal image references.
- Normalize the reference into a canonical registry/repository/tag or digest form.
- Generate registry-specific mirror candidates.
- Probe candidates before attempting a full pull.
- Select the best reachable candidate before pulling.
- Retry pull attempts and fall back to the next reachable candidate, ending with the source registry as a final candidate.
- Pull with Docker, Podman, or nerdctl.
- Retag the result back to the original image name when needed.
- Provide scriptable output and predictable exit codes.

## Non-Goals

- Do not edit `/etc/docker/daemon.json` by default.
- Do not restart Docker, containerd, or system services.
- Do not become a full registry server.
- Do not manage Kubernetes admission webhooks in the initial version.
- Do not claim a public mirror is safe for production supply chains.
- Do not copy third-party reference tools in this repository.

## Design Principles

- Unix first: predictable commands, clean output, no decorative output by default.
- Fast first: probe manifests before pulling large blobs.
- Registry-aware: model each registry separately instead of applying global string replacement.
- Safe by default: preserve original tags locally and verify digest when possible.
- Configurable but useful with no config.
- Composable: `rewrite`, `probe`, and `pull` should each be useful alone.
- Transparent: show which mirror was selected and why when asked.

## Primary Users

- Individual developers pulling public images on laptops or workstations.
- CI jobs that need stable and scriptable image pull behavior.
- Platform engineers preparing images for local clusters.
- Teams that want a stepping stone before adopting a private registry sync workflow.

## Production Guidance

Public mirrors should be treated as convenience infrastructure. For production workloads, prefer syncing images into a trusted private registry and pulling by digest.
