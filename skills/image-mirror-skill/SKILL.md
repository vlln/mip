---
name: image-mirror-skill
description: Use this skill when you need to accelerate or troubleshoot Docker/OCI image pulls that are slow, unstable, or blocked, using registry-aware mirror rewriting, probing, and pulling.
license: MIT
metadata:
  author: vlln
  version: "0.1.0"
requires:
  bins:
    - mip
---

# Image Mirror Skill

Use this skill to help users accelerate and troubleshoot container image pulls
through registry-aware mirrors with the `mip` CLI.

`mip` helps when `docker pull` is slow, unstable, or blocked. It rewrites image
references to known mirrors, probes reachability, pulls through Docker, Podman,
or nerdctl, and retags the local image back to the original name by default.

## Trigger Keywords

- docker pull, podman pull, nerdctl pull
- image pull slow, image pull timeout, image pull blocked
- container registry mirror, Docker mirror, registry mirror
- pull image from mirror, accelerate image pull, speed up image pull
- mip

## Decision Flow

1. Slow or failed pull: start by probing mirror reachability with a timeout.
2. Explain mirror rewrites: show all rewrite candidates for the image.
3. Pull safely: dry-run first, then pull for real.
4. Keep original local tag: use default pull (retags automatically).
5. Keep mirror tag for debugging: use `--no-retag`.
6. Prefetch Dockerfile base images: dry-run first, then prefetch before
   `docker build`.
7. Customize mirror order: edit XDG config, then show the config.

For detailed command syntax, read `references/mip-cli.md`.

## Core Workflows

Non-destructive inspection:

```bash
mip rewrite nginx:1.27 --all
mip probe nginx:1.27 --timeout 8s
mip pull hello-world:latest --dry-run
```

Pull after the selected mirror looks reasonable:

```bash
mip pull nginx:1.27 --engine docker --platform linux/amd64
```

Prefetch all base images from a Dockerfile before building:

```bash
mip prefetch --dry-run
mip prefetch
docker build -t myapp .
```

Use JSON for structured agent/tool output:

```bash
mip probe nginx:1.27 --platform linux/amd64 --json
mip pull nginx:1.27 --json
```

Inspect mirrors and config:

```bash
mip mirrors list
mip mirrors list --registry registry.k8s.io
mip config show
```

## Troubleshooting Patterns

- Probe with `--timeout 8s`: find reachable mirrors and compare latency.
- Rewrite with `--all`: check whether the registry has configured mirrors.
- Pull with `--dry-run`: confirm the selected mirror before pulling.
- Pull with `--retries 2`: retry transient pull errors per candidate.
- Pull with `--engine podman`: use Podman instead of Docker.
- Pull with `--no-verify-digest`: use only when digest inspection is broken
  and the user accepts the tradeoff.
- Pull with `--no-retag`: keep the mirror image name locally for debugging.

## Gotchas

- **Mirror pull is not a security boundary.** Public mirrors can serve
  different content than the original registry. For production, sync required
  images into a trusted private registry and pull by digest.
- **Digest verification may fail on some registries.** If a registry serves
  different manifests for the same tag across mirrors, digest mismatch errors
  occur. Use `--no-verify-digest` only after confirming the user understands
  the risk.
- **Auth-required mirrors are not skipped.** They remain in the fallback
  chain as warnings — this is intentional to handle registries that require
  `docker login` but may still work after authentication.
- **Platform-specific manifests may not exist on all mirrors.** When using
  `--platform`, probe first to check availability before pulling.
- **Do not edit the Docker daemon config.** `mip` rewrites at the pull
  command level — it does not need changes to `/etc/docker/daemon.json` or
  `registries.conf`. Avoid suggesting daemon config changes unless the user
  explicitly asks for them.
- **A container engine is only required for actual pulls.** Rewrite, probe,
  and config inspection all work without Docker, Podman, or nerdctl installed.

## Safety Defaults

- Prefer `--dry-run` before real pulls when changing mirrors or config.
- Prefer probing when diagnosing network or mirror issues.
- Do not edit Docker daemon config unless the user explicitly asks; `mip` is
  designed to avoid daemon mutation.
- Keep public mirrors as convenience infrastructure, not a production
  supply-chain trust boundary.
- For production, recommend syncing required images into a trusted private
  registry and pulling by digest.
- If `mip` is unavailable, install it first or ask for approval when required.