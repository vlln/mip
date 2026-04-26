---
name: image-mirror-skill
description: Use the mip CLI to accelerate and troubleshoot Docker/OCI image pulls with registry-aware mirror rewrite, probe, pull, and configuration workflows.
license: MIT
metadata:
  skit:
    version: 0.1.0
    requires:
      bins:
        - mip
    keywords:
      - container-images
      - docker
      - mirrors
      - registry
---

# Image Mirror Skill

Use this skill to help users accelerate and troubleshoot container image pulls
through registry-aware mirrors with the `mip` CLI.

`mip` is published at <https://github.com/vlln/mip>. It helps when `docker pull`
is slow, unstable, or blocked. It rewrites image references to known mirrors,
probes reachability, pulls through Docker, Podman, or nerdctl, and retags the
local image back to the original name by default.

## Install mip

Before using this skill, check whether `mip` is available:

```bash
mip version
```

If missing, install the latest GitHub Release:

```bash
curl -fsSL https://raw.githubusercontent.com/vlln/mip/main/scripts/install.sh | sh
```

To install outside `/usr/local/bin`:

```bash
curl -fsSL https://raw.githubusercontent.com/vlln/mip/main/scripts/install.sh | MIP_BINDIR="$HOME/.local/bin" sh
```

After installation, verify:

```bash
mip version
mip config show
```

Homebrew:

```bash
brew install vlln/tap/mip
```

## Decision Flow

1. Slow or failed pull: start with `mip probe IMAGE --timeout 8s`.
2. Explain mirror rewrites: use `mip rewrite IMAGE --all`.
3. Pull safely: use `mip pull IMAGE --dry-run`, then `mip pull IMAGE`.
4. Keep original local tag: use default `mip pull IMAGE`.
5. Keep mirror tag for debugging: use `mip pull IMAGE --no-retag`.
6. Customize mirror order: edit XDG config, then run `mip config show`.

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

- `mip probe IMAGE --timeout 8s`: find reachable mirrors and compare latency.
- `mip rewrite IMAGE --all`: check whether the registry has configured mirrors.
- `mip pull IMAGE --dry-run`: confirm the selected mirror before pulling.
- `mip pull IMAGE --retries 2`: retry transient pull errors per candidate.
- `mip pull IMAGE --engine podman`: use Podman instead of Docker.
- `mip pull IMAGE --no-verify-digest`: use only when digest inspection is broken and the user accepts the tradeoff.
- `mip pull IMAGE --no-retag`: keep the mirror image name locally for debugging.

## Safety Defaults

- Prefer `--dry-run` before real pulls when changing mirrors or config.
- Prefer `mip probe` when diagnosing network or mirror issues.
- Do not edit Docker daemon config unless the user explicitly asks; `mip` is designed to avoid daemon mutation.
- Keep public mirrors as convenience infrastructure, not a production supply-chain trust boundary.
- For production, recommend syncing required images into a trusted private registry and pulling by digest.
- If `mip` is unavailable, install from the official repository or ask first when approval is required.
- Docker, Podman, or nerdctl is required only for real image pulls; rewrite/probe/config inspection can still be useful before pulling.
