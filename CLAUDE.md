# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build / Test / Lint

```bash
make build      # builds bin/mip and bin/gip (CGO_ENABLED=0, stripped)
make test       # go test ./...
make version    # build + print both versions (dev/CI smoke test)
make release VERSION=0.1.0   # cross-compile release archives into dist/
```

CI (`.github/workflows/ci.yml`) runs `go test ./...`, `make build`, and a smoke test for both binaries. Tagged commits (`v*`) trigger a cross-platform release job and Homebrew tap update.

## Architecture

This is a monorepo containing two single-binary Go CLIs: `mip` (container image mirror) and `gip` (GitHub mirror). Both share the same core pattern: **rewrite → probe → select → execute → retag**.

**Entry points**: `cmd/mip/main.go` and `cmd/gip/main.go` — manual flag parsing (no third-party CLI library).

### Shared packages (`internal/`)

| Package | Role |
|---|---|
| `output` | Small JSON/line output helpers. |
| `version` | Build-time vars (`-X`) for version/commit/date + runtime info. |

### mip packages (`internal/`)

| Package | Role |
|---|---|
| `ref` | Parses OCI image references (e.g. `nginx:1.27` → `docker.io/library/nginx:1.27`). |
| `registry` | Data types: `Profile` (registry name + aliases + mirrors), `Mirror` (host, mode, priority). Two rewrite modes: `host-replace` and `prefix`. |
| `rewrite` | Generates candidate image URLs from a source image + registry profile. Sorts by priority. |
| `probe` | HTTP-based probing. Fetches OCI/Docker manifests, handles bearer token auth, resolves platform-specific digests from manifest lists. |
| `engine` | Abstraction over Docker/Podman/nerdctl. Interface: `Pull`, `Tag`, `Remove`, `RepoDigests`. |
| `config` | Loads/merges embedded official config (`configs/mip.yaml`) with user config (`$XDG_CONFIG_HOME/mip/config.yaml`). |
| `state` | Persistent mirror health tracking. Score-based ranking. Written to `$XDG_STATE_HOME/mip/state.json`. |
| `completion` | Hardcoded bash/zsh/fish completion scripts for mip. |

### gip packages (`internal/`)

| Package | Role |
|---|---|
| `giturl` | Parses GitHub URLs. Distinguishes `KindClone`, `KindRelease`, `KindRaw`, `KindArchive`, `KindBlob`, `KindGist`. Provides `CanonicalHost()` to map subdomains (raw.githubusercontent.com, api.github.com, etc.) to `github.com`. |
| `gitmirror` | Data types: `Mirror`, `RewriteMode`, `Profile`. Three rewrite modes: `host-replace`, `prefix`, `path-transform`. |
| `gitrewrite` | Generates candidate mirror URLs from a source URL + profile. Sorts by priority. |
| `gitprobe` | Concurrent HTTP probing. For clone URLs, probes git `info/refs` endpoint and validates the response is a real git advertisement. For other URLs, uses HTTP HEAD. |
| `gitstate` | Persistent mirror health tracking. Same scoring algorithm as mip's `state`. Written to `$XDG_STATE_HOME/gip/state.json`. |
| `gitops` | Git operations: `Clone` (mirror URL → remote set-url back to original), `Install`/`Uninstall` (git config `insteadOf` for transparent proxy). |
| `gitconfig` | Loads/merges embedded official config (`configs/gip.yaml`) with user config (`$XDG_CONFIG_HOME/gip/config.yaml`). Supports `host:mode` syntax for mirror entries. |
| `gipcompletion` | Hardcoded bash/zsh/fish completion scripts for gip. |

### mip pull flow (`runPull` in `cmd/mip/main.go`)
1. Parse image reference → look up registry profile → generate rewrite candidates.
2. Apply health ranking from saved state, sort, append a low-priority "source" fallback.
3. Probe all candidates concurrently (HTTP manifest HEAD/GET with timeout).
4. Sort results: OK > auth-required > failure, mirrors before source, fastest first.
5. Iterate candidates with retry: `engine.Pull` mirror image, verify digest, `engine.Tag` to original name, `engine.Remove` mirror image.
6. Save updated mirror health state.

### gip clone flow (`runClone` in `cmd/gip/main.go`)
1. Parse GitHub URL → look up host profile (using canonical host) → generate rewrite candidates.
2. Apply health ranking, append source fallback.
3. Probe all candidates concurrently (git `info/refs` for clone, HTTP HEAD for downloads).
4. Sort results, select best.
5. `gitops.Clone` from mirror URL, then `git remote set-url origin` back to original.
6. Save updated mirror health state.

### gip install flow
1. Probe mirrors for the target host using a synthetic test URL.
2. Select best mirror.
3. Write `git config --global url.<mirror-base>.insteadOf https://<host>/`.
4. All subsequent git operations to that host are transparently proxied.

## Key design details

- **No config file needed at runtime.** `configs/mip.yaml` and `configs/gip.yaml` are embedded in the respective binaries. User config augments/overrides them.
- **Move-flag-first arg handling.** The custom `moveFlagsFirst` function reorders args so flag-style arguments precede operands, working around Go's `flag` package requirement.
- **Mirror mode inference.** mip infers `host-replace` vs `prefix` from whether the mirror host contains the registry name. gip supports `host:mode` syntax (e.g. `ghproxy.com:prefix`) and defaults to `host-replace`.
- **Subdomain canonicalization.** gip's `giturl.CanonicalHost()` maps `raw.githubusercontent.com`, `api.github.com`, `gist.githubusercontent.com`, etc. to `github.com` for unified mirror profile lookup.
- **Git advertisement validation.** gip's probe validates that clone mirror responses contain actual git protocol advertisements, filtering out mirrors that return HTML pages.
- **Platform-aware manifest selection.** When `--platform` is passed, mip's `probe` looks inside OCI index / manifest list JSON and resolves to the child manifest digest matching the requested platform.