# mip

Pull container images when the default registry path lets you down.

[简体中文](README.zh.md)

`mip` is a small CLI for the moments when `docker pull` gets stuck, times out,
or crawls through an overloaded registry route. Keep using the image names your
project already has. `mip` finds mirror candidates, checks which ones are alive,
pulls through a working path, and leaves the image tagged the way your tools
expect.

```bash
mip pull nginx:1.27
```

Use it once from a terminal, drop it into CI, or ask it why an image will not
pull cleanly from where you are.

## The Problem

Container images are part of every build now, but pulling them is still oddly
fragile:

- Docker Hub is slow or unreachable on your network.
- A public mirror works for one image but not the next.
- The right mirror path is different for Docker Hub, GHCR, Quay, MCR, and
  Kubernetes images.
- CI fails before your tests even start because a base image did not arrive.
- A quick manual rewrite gets the image pulled, but your scripts now depend on
  a URL nobody wants to maintain.

`mip` sits in front of that mess. It understands image references, knows about
common public registries, probes mirror candidates, and hands the final pull to
Docker, Podman, or nerdctl.

## Fast Path

Find working routes for an image:

```bash
mip probe nginx:1.27 --timeout 8s
```

See exactly how the image can be rewritten:

```bash
mip rewrite nginx:1.27 --all
```

Pull through the best reachable mirror and keep the original image tag:

```bash
mip pull hello-world:latest --timeout 8s
```

Need a specific platform or runtime?

```bash
mip pull hello-world:latest --platform linux/amd64 --retries 2
mip pull hello-world:latest --engine podman --dry-run
```

## Why It Feels Different

`mip` is not just a text replacement tool. It checks whether candidates can
actually serve the manifest you asked for, handles platform-aware manifest
lists, remembers basic mirror health, and verifies the pulled digest when the
selected manifest digest is known.

It ships with default rules for common public registries, including Docker Hub,
GHCR, Quay, MCR, Kubernetes, GCR, Elastic, NVCR, DHI, and Ollama. You can use it
with no config file, then add your own preferences when you need control.

```bash
mip mirrors list --registry registry.k8s.io
mip config show
```

## Install

### Homebrew

```bash
brew install vlln/tap/mip
mip version
```

### GitHub Release

Install the latest GitHub Release with the install script:

```bash
curl -fsSL https://raw.githubusercontent.com/vlln/mip/main/scripts/install.sh | sh
mip version
```

Set `MIP_BINDIR` to install somewhere other than `/usr/local/bin`.

## Configure

You do not need a config file to start. The default mirror rules are embedded in
the binary and kept in [configs/mip.yaml](configs/mip.yaml).

When you do want local policy, create one of:

- `$XDG_CONFIG_HOME/mip/config.yaml`
- `~/.config/mip/config.yaml`

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

`mip` also keeps lightweight mirror health state in:

- `$XDG_STATE_HOME/mip/state.json`
- `~/.local/state/mip/state.json`

If state cannot be read or written, `mip` warns and keeps going.

## Shell Completion

Shell completion lets your shell suggest `mip` commands, flags, and subcommands
when you press Tab.

```bash
mip completion bash > ~/.local/share/bash-completion/completions/mip
mip completion zsh > ~/.zfunc/_mip
mip completion fish > ~/.config/fish/completions/mip.fish
```

## Agent Skill

This repository includes an Agent Skill for AI agents that help diagnose and
repair container image pull failures.

```sh
skit install --global vlln/mip/skills/image-mirror-skill
```

Install all skills in this repository:

```sh
skit install --global vlln/mip --all
```

Manual install: copy [skills/image-mirror-skill/](skills/image-mirror-skill/) into
your agent's skills directory.

## Develop

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

## Requirements

- Docker, Podman, or nerdctl for real image pulls.
- Network access to the selected registries and mirrors.
- Go 1.22+ only for development builds.

## License

MIT for the `mip` code and `skills/image-mirror-skill`.
