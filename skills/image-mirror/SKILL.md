---
name: image-mirror
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

# Image Mirror

Use this skill to help users accelerate container image pulls through registry-aware mirrors with the `mip` CLI.

## Decision Flow

1. If the user wants to **pull an image faster**, use `mip pull IMAGE` or dry-run first.
2. If the user wants to **inspect mirror rewrites**, use `mip rewrite IMAGE --all`.
3. If the user wants to **test mirror availability or speed**, use `mip probe IMAGE`.
4. If the user wants to **customize mirrors**, edit or generate an XDG config file.

For detailed command syntax, read `references/mip-cli.md`.

## Core Commands

Start with non-destructive commands:

```bash
mip rewrite nginx:1.27 --all
mip probe hello-world:latest --platform linux/amd64 --json
mip pull hello-world:latest --dry-run
```

Pull after the selected mirror looks reasonable:

```bash
mip pull nginx:1.27 --engine docker --platform linux/amd64
```

## Safety Defaults

- Prefer `--dry-run` before real pulls when changing mirrors or config.
- Do not edit Docker daemon config unless the user explicitly asks; `mip` is designed to avoid daemon mutation.
- Keep public mirrors as convenience infrastructure, not a production supply-chain trust boundary.
- For production, recommend syncing required images into a trusted private registry and pulling by digest.
- If `mip` is unavailable, tell the user to install it before using this skill.
- Docker, Podman, or nerdctl is required only for real image pulls; rewrite/probe/config inspection can still be useful before pulling.
