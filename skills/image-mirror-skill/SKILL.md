---
name: image-mirror-skill
description: Use this skill when you need to accelerate or troubleshoot Docker/OCI image pulls that are slow, unstable, or blocked, using registry-aware mirror rewriting, probing, pulling, and retagging. Activate when the user mentions docker pull, podman pull, nerdctl pull, image pull timeout, container registry mirror, Docker Hub blocked, accelerating image pulls, prefetching FROM images, or any container image not pulling cleanly from the current network.
license: MIT
metadata:
  author: vlln
  version: "0.1.0"
requires:
  bins:
    - mip
---

# Image Mirror Skill

Use this skill when `docker pull` (or `podman`/`nerdctl`) is slow, unstable, or
blocked. `mip` rewrites image references to known mirrors, probes reachability,
pulls through the container engine, and retags the local image back to the
original name.

## Trigger Keywords

- docker pull, podman pull, nerdctl pull
- image pull slow, image pull timeout, image pull blocked
- container registry mirror, Docker mirror, registry mirror
- pull image from mirror, accelerate image pull, speed up image pull
- Docker Hub unreachable, GHCR blocked, Quay timeout
- prefetch, Dockerfile FROM images, pre-pull base images
- mip

## Capabilities

- **Rewrite** image references to mirror candidates for a given registry.
- **Probe** mirror reachability and latency without pulling.
- **Pull** images through the fastest reachable mirror, retagging to the
  original reference by default.
- **Prefetch** all `FROM` base images from a Dockerfile before building.
- **Inspect** configured mirrors and runtime config.

For command syntax, flags, config format, and state file paths, read
`$_S/references/mip-cli.md`.

## Workflow

When a user reports slow or failed image pulls:

1. **Probe first.** Run `mip probe <image> --timeout 8s` to find reachable
   mirrors and compare latency. Use `--json` for structured output.
2. **Show mirrors.** Run `mip rewrite <image> --all` to list all rewrite
   candidates. If the registry has no configured mirrors, report it.
3. **Dry-run.** Run `mip pull <image> --dry-run` to confirm the selected mirror
   before pulling.
4. **Pull.** Run `mip pull <image>` with the user's platform and engine
   preferences. Default to `docker` unless the user specifies otherwise.
5. **Verify.** After pull, confirm the image is available locally under the
   original name (`docker images`).

For Dockerfile builds, use `mip prefetch --dry-run` first, then `mip prefetch`
before `docker build`.

## Gotchas

- **Do not edit the Docker daemon config.** `mip` rewrites at the pull command
  level — it does not need changes to `/etc/docker/daemon.json` or
  `registries.conf`. Never suggest daemon config changes.
- **Digest verification may fail.** If a registry serves different manifests
  for the same tag across mirrors, digest mismatch errors occur. Use
  `--no-verify-digest` only after confirming the user understands the risk.
- **Auth-required mirrors remain in the fallback chain.** They appear as
  warnings but are not skipped — `docker login` to the mirror may still work.
- **Platform-specific manifests may not exist on all mirrors.** When using
  `--platform`, probe first to check availability.
- **Rewrite, probe, and config inspection work without a container engine.**
  Docker, Podman, or nerdctl are only needed for actual pulls.
- **Mirror pull is not a security boundary.** Public mirrors can serve
  different content than the original registry. For production, sync required
  images into a trusted private registry and pull by digest.
- **Mirror availability changes frequently.** The default mirror list in
  `configs/mip.yaml` may become stale. Always probe before relying on a mirror.
- **Source registry may be reachable when mirrors are not.** `mip` always
  includes the source registry as a low-priority fallback candidate. If all
  mirrors fail, it will try the original registry.
- **Two rewrite modes exist.** `host-replace` swaps the registry host (e.g.
  `docker.io` → `dockerproxy.net`). `prefix` prepends a mirror path (e.g.
  `docker.io` → `m.daocloud.io/docker.io`). The mode is inferred from the
  mirror host format.