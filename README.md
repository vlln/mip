<h1 align="center">mip &amp; gip</h1>

<p align="center">
  <strong>mip: Pull container images through the fastest reachable mirror.<br/>
  gip: Clone, fetch, and download from GitHub through the fastest mirror.</strong>
</p>

<p align="center">
  <a href="https://github.com/vlln/mip/stargazers"><img src="https://badgen.net/github/stars/vlln/mip?label=%E2%98%85" alt="GitHub stars" /></a>
  <img src="https://badgen.net/badge/license/MIT/blue" alt="MIT" />
</p>

[简体中文](README.zh.md)

---

Two small CLIs for the moments when `docker pull` gets stuck, or `git clone` crawls
through an overloaded route. Keep using the names your project already has. They find
mirror candidates, check which ones are alive, and route through the best working path.

## mip — Container Image Mirror

```bash
mip pull nginx:1.27
```

## gip — GitHub Mirror

```bash
gip clone https://github.com/user/repo
gip get https://github.com/user/repo/releases/download/v1.0/binary.tar.gz
gip install    # transparent git insteadOf proxy
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

Pull all `FROM` images in a Dockerfile before building:

```bash
mip prefetch
mip prefetch -f path/to/Dockerfile --dry-run
```

If `Dockerfile` is not found, `mip prefetch` falls back to `Containerfile`.

Then build as usual — Docker will use the already-pulled images:

```bash
docker build -t myapp .
```

## gip — GitHub Mirror

`gip` does for GitHub what `mip` does for container registries. It probes mirror
candidates for GitHub URLs, finds the fastest reachable one, and routes git clone,
file downloads, and raw content through it.

### Clone through a mirror

```bash
gip clone https://github.com/user/repo
gip clone https://github.com/user/repo --dry-run
```

### Download release assets and raw files

```bash
gip get https://github.com/user/repo/releases/download/v1.0/binary.tar.gz
gip get https://raw.githubusercontent.com/user/repo/main/file.txt --output myfile.txt
```

### Transparent proxy (git insteadOf)

Configure git to automatically route all GitHub traffic through the fastest mirror:

```bash
gip install
# Now all git clone/fetch/pull to github.com go through the mirror
gip uninstall
```

### Diagnostics

```bash
gip probe https://github.com/user/repo --timeout 8s
gip rewrite https://github.com/user/repo --all
gip mirrors list --host github.com
gip config show
```

## Install CLI

### Homebrew

```bash
brew install vlln/tap/mip
mip version
gip version
```

### GitHub Release

Install the latest GitHub Release with the install script:

```bash
curl -fsSL https://raw.githubusercontent.com/vlln/mip/main/scripts/install.sh | sh
mip version
```

By default the script installs to `/usr/local/bin` when writable, otherwise to
`$HOME/.local/bin`. Set `MIP_BINDIR` to choose another directory.

## Why It Feels Different

`mip` and `gip` are not just text replacement tools. They check whether candidates can
actually serve the content you asked for, remember basic mirror health, and verify
results when possible.

`mip` handles platform-aware manifest lists, OCI/Docker manifest headers, and bearer
token auth. `gip` probes git smart-HTTP endpoints for clone mirrors and uses HTTP HEAD
for file downloads.

Both ship with default rules for common registries and hosts. Use them with no config
file, then add your own preferences when you need control.

```bash
mip mirrors list --registry docker.io
gip mirrors list --host github.com
mip config show
gip config show
```

## Configure

You do not need a config file to start. Default mirror rules are embedded in the
binary and kept in [configs/mip.yaml](configs/mip.yaml) and [configs/gip.yaml](configs/gip.yaml).

When you do want local policy, create one of:

- `$XDG_CONFIG_HOME/mip/config.yaml`
- `~/.config/mip/config.yaml`
- `$XDG_CONFIG_HOME/gip/config.yaml`
- `~/.config/gip/config.yaml`

Example (`mip/config.yaml`):

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

Example (`gip/config.yaml`):

```yaml
prefer:
  - my-company-mirror
exclude:
  - ghproxy.homeboyc.cn
mirrors:
  github.com:
    mirrors:
      - mirror.example.com:host-replace
```

Health state is kept in:

- `$XDG_STATE_HOME/mip/state.json`
- `$XDG_STATE_HOME/gip/state.json`

## Shell Completion

Shell completion lets your shell suggest `mip` commands, flags, and subcommands
when you press Tab.

```bash
mip completion bash > ~/.local/share/bash-completion/completions/mip
mip completion zsh > ~/.zfunc/_mip
mip completion fish > ~/.config/fish/completions/mip.fish
```

---

## Installation

### [skit](https://github.com/vlln/skit) (Recommended)

```bash
skit install https://github.com/vlln/mip/tree/main/skills/image-mirror-skill
```

### [skill.sh](https://github.com/vercel-labs/skills)

```bash
npx skills add vlln/mip
```

### Manually

| Agent | Command |
|-------|---------|
| **Claude Code** | `cp -r skills/image-mirror-skill .claude/skills/` |
| **Codex** | `cp -r skills/image-mirror-skill ~/.codex/skills/` |
| **OpenCode** | `git clone https://github.com/vlln/mip.git ~/.opencode/skills/mip` |
| **Kimi** | `cp -r skills/image-mirror-skill ~/.kimi/skills/` |

---

## Skills

| Skill | Description |
|-------|-------------|
| [image-mirror-skill](skills/image-mirror-skill/SKILL.md) | Accelerate and troubleshoot Docker/OCI image pulls using registry-aware mirror rewriting, probing, and pulling. |
| [github-mirror-skill](skills/github-mirror-skill/SKILL.md) | Accelerate and troubleshoot GitHub access (clone, downloads, raw files) using mirror-aware URL rewriting, probing, and transparent git proxying. |

## Requirements

- `mip`: Docker, Podman, or nerdctl for real image pulls.
- `gip`: git for clone operations.
- Network access to the selected registries, mirrors, and hosts.
- Go 1.22+ only for development builds.

## Develop

```bash
make test
make build
./bin/mip version
./bin/gip version
```

Create local release archives:

```bash
make release VERSION=0.1.0
ls dist/
```

## License

MIT for the `mip`, `gip` code and all skills.