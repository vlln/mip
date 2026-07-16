# gip CLI Reference

## Common Workflows

Rewrite candidates:

```bash
gip rewrite https://github.com/user/repo --all
gip rewrite https://github.com/user/repo.git --plain --all
gip rewrite https://raw.githubusercontent.com/user/repo/main/file.txt --all
```

Probe mirror health:

```bash
gip probe https://github.com/user/repo --timeout 8s
gip probe https://github.com/user/repo/releases/download/v1.0/file.tar.gz --json
```

Clone through the fastest usable mirror:

```bash
gip clone https://github.com/user/repo
gip clone https://github.com/user/repo --dry-run
gip clone https://github.com/user/repo --dir my-project
```

Download files through mirrors:

```bash
gip get https://github.com/user/repo/releases/download/v1.0/binary.tar.gz
gip get https://raw.githubusercontent.com/user/repo/main/config.yaml --output config.yaml
gip get https://github.com/user/repo/archive/refs/tags/v1.0.zip --dry-run
```

Transparent git proxy:

```bash
gip install                           # probe and set up git insteadOf for github.com
gip install --host gitlab.com         # proxy for another host
gip uninstall                         # remove all insteadOf entries for github.com
gip uninstall --host gitlab.com
```

Inspect config and mirrors:

```bash
gip config show
gip mirrors list
gip mirrors list --host github.com
gip mirrors list --json
```

## URL Types Supported

gip recognizes and correctly rewrites these GitHub URL patterns:

| URL Pattern | Kind | Example |
|-------------|------|---------|
| Clone (HTTPS) | `clone` | `https://github.com/user/repo.git` |
| Clone (SSH) | `clone` | `git@github.com:user/repo.git` |
| Release download | `release` | `https://github.com/user/repo/releases/download/v1.0/file.tar.gz` |
| Archive (branch) | `archive` | `https://github.com/user/repo/archive/refs/heads/main.zip` |
| Archive (tag) | `archive` | `https://github.com/user/repo/archive/refs/tags/v1.0.zip` |
| Raw file | `raw` | `https://raw.githubusercontent.com/user/repo/main/file.txt` |
| Blob page | `blob` | `https://github.com/user/repo/blob/main/file.txt` |
| Gist | `gist` | `https://gist.githubusercontent.com/user/id/raw/file` |
| API | *(unknown)* | `https://api.github.com/repos/user/repo` |
| Avatars | *(unknown)* | `https://avatars.githubusercontent.com/u/10000` |
| Desktop | *(unknown)* | `https://desktop.githubusercontent.com/releases/...` |

All subdomain hosts (`raw.githubusercontent.com`, `api.github.com`, etc.) are
aliased to `github.com` for mirror profile lookup.

## Rewrite Modes

| Mode | Description | Example |
|------|-------------|---------|
| `host-replace` | Swap the host domain | `github.com/user/repo` → `kkgithub.com/user/repo` |
| `prefix` | Prepend mirror URL to the full original URL | `github.com/user/repo` → `ghproxy.com/https://github.com/user/repo` |
| `path-transform` | Embed the source host as a path segment | `github.com/user/repo` → `gitclone.com/github.com/user/repo` |

## Config

Default paths:

- `$XDG_CONFIG_HOME/gip/config.yaml`
- `~/.config/gip/config.yaml`

The official default config is distributed as `configs/gip.yaml` and is embedded
into `gip` for zero-config mirror use. User config augments the official one.

Example:

```yaml
prefer:
  - my-company-mirror
exclude:
  - ghproxy.homeboyc.cn
mirrors:
  github.com:
    mirrors:
      - mirror.example.com:host-replace
      - ghproxy.example.com:prefix
```

Mirror entries support `host:mode` syntax. When mode is omitted, it defaults to
`host-replace` for simple hostnames, `prefix` for hosts with path components.

## State

State path:

- `$XDG_STATE_HOME/gip/state.json`
- `~/.local/state/gip/state.json`

State records mirror success/failure, latency, status code, and errors. It only
influences candidate priority; it never replaces live probe.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | General error |
| 2 | Invalid URL |
| 3 | No usable mirror |
| 4 | Git error |
| 5 | Download failed |
| 9 | Config error |