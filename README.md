# mip

`mip` is a registry-aware CLI for accelerating container image pulls through configurable mirrors.

## Current Status

Implemented:

- Image reference parsing and normalization.
- Official default config for common public registries including Docker Hub, GHCR, Quay, MCR, Kubernetes, GCR, Elastic, NVCR, DHI, and Ollama.
- Mirror candidate rewriting.
- Concurrent manifest probing with basic bearer-token auth handling.
- Engine adapter abstraction for Docker, Podman, and nerdctl.
- Digest verification after pull when the selected manifest digest is known.
- XDG config loading with mirror hosts, prefer, and exclude.
- XDG state file with historical mirror health scoring.
- Platform-aware manifest list selection for `--platform`.
- `mip rewrite`.
- `mip probe`.
- `mip pull` with Docker execution, retagging, and temporary mirror tag cleanup.
- `mip mirrors list`.
- `mip config show`.

## Install

### Homebrew

```bash
brew install vlln/tap/mip
mip version
```

### Release archive

Download the archive for your platform from GitHub Releases, then install the
binary:

```bash
tar -xzf mip_0.1.0_linux_amd64.tar.gz
sudo install -m 0755 mip_0.1.0_linux_amd64/mip /usr/local/bin/mip
mip version
```

### Install script

```bash
./scripts/install.sh
```

By default, the script installs the latest GitHub release to `/usr/local/bin`.
Use `MIP_VERSION` or `MIP_BINDIR` only when pinning a version or installing to a
custom directory.

## Quick Start

```bash
mip rewrite nginx:1.27 --all
mip probe nginx:1.27 --timeout 8s
mip probe hello-world:latest --platform linux/amd64 --json
mip pull hello-world:latest --timeout 8s
mip pull hello-world:latest --platform linux/amd64 --retries 2
mip pull hello-world:latest --engine podman --dry-run
mip mirrors list --registry registry.k8s.io
mip config show
```

## Developer

```bash
make test
make build
./bin/mip version
```

Create local release archives:

```bash
make release VERSION=0.1.0
ls dist/
```

## Shell Completion

```bash
mip completion bash > ~/.local/share/bash-completion/completions/mip
mip completion zsh > ~/.zfunc/_mip
mip completion fish > ~/.config/fish/completions/mip.fish
```

## Config

Default config paths:

- `$XDG_CONFIG_HOME/mip/config.yaml`
- `~/.config/mip/config.yaml`

The official default config is [configs/mip.yaml](configs/mip.yaml). It is
embedded into the binary for zero-config use and included in release archives so
users can copy it as a starting point. If a user config exists or `--config` is
provided, that single config replaces the official one.

Example:

```yaml
prefer:
  - company-cache
exclude:
  - dockerproxy.cool
registries:
  docker.io:
    mirrors:
      - registry.example.com/docker.io
```

State path:

- `$XDG_STATE_HOME/mip/state.json`
- `~/.local/state/mip/state.json`

State is only an optimization. If it cannot be read or written, `mip` warns and continues.

## Skills

This repository also includes Agent Skills under `skills/`. Each skill follows the
[Agent Skills specification](https://agentskills.io/specification) and can be used
by skills-compatible agents.

| Skill | Description |
|-------|-------------|
| [`image-mirror-skill`](skills/image-mirror-skill) | Accelerate and troubleshoot Docker/OCI image pulls with mip mirror workflows. |

### Skill Quick Start

Paste this into your AI agent:

```text
Install the Agent Skills from https://raw.githubusercontent.com/vlln/mip/main/README.md
```

### Skill Installation

Recommended: install skills with [`skit`](https://github.com/vlln/skit).

Install `skit` with Homebrew:

```sh
brew install --cask vlln/tap/skit
```

For other platforms, see the
[`skit` installation instructions](https://github.com/vlln/skit#installation).

Install this skill from the published repository:

```sh
skit install --global vlln/mip/skills/image-mirror-skill
```

Install all skills in this repository:

```sh
skit install --global vlln/mip --all
```

Manual install: copy [skills/image-mirror-skill/](skills/image-mirror-skill/) into your
agent's skills directory.

The root project remains the `mip` CLI; the skill is an additional agent-facing
guide for using and maintaining it.

## Requirements

- Docker, Podman, or nerdctl for real image pulls.
- Network access to the selected registries and mirrors.
- Go 1.22+ only for development builds.

## License

MIT for the `mip` code and `skills/image-mirror-skill`.
