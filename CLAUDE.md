# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build / Test / Lint

```bash
make build      # builds bin/mip (CGO_ENABLED=0, stripped)
make test       # go test ./...
make version    # build + print version (dev/CI smoke test)
make release VERSION=0.1.0   # cross-compile release archives into dist/
```

CI (`.github/workflows/ci.yml`) runs `go test ./...`, `make build`, and a smoke test. Tagged commits (`v*`) trigger a cross-platform release job and Homebrew tap update.

## Architecture

`mip` is a single-binary Go CLI that probes container registry mirrors, pulls through the best working path, and retags the result under the original image name.

**Entry point**: `cmd/mip/main.go` — manual flag parsing (no third-party CLI library). Subcommands: `version`, `rewrite`, `probe`, `pull`, `prefetch`, `mirrors list`, `config show`, `completion`. Bare `mip IMAGE` is shorthand for `pull IMAGE`.

**Core packages** (`internal/`):

| Package | Role |
|---|---|
| `ref` | Parses OCI image references (e.g. `nginx:1.27` → `docker.io/library/nginx:1.27`). Understands registry detection, default namespace insertion, digest vs. tag. |
| `registry` | Data types: `Profile` (registry name + aliases + mirrors), `Mirror` (host, mode, priority). Two rewrite modes: `host-replace` (swap registry host) and `prefix` (prepend mirror path). |
| `rewrite` | Generates candidate image URLs from a source image + registry profile. Sorts by priority (config order + health-based adjustments). |
| `probe` | HTTP-based probing. Fetches container manifests with OCI/Docker accept headers, handles bearer token auth, resolves platform-specific digests from manifest lists. |
| `engine` | Abstraction over Docker/Podman/nerdctl. Interface: `Pull`, `Tag`, `Remove`, `RepoDigests`. Implemented via `exec.Command` calls to the respective CLI binary. |
| `config` | Loads/merges embedded official config (`configs/mip.yaml`, embedded via `//go:embed`) with user config file (`$XDG_CONFIG_HOME/mip/config.yaml`). Produces registry profiles with prefer/exclude applied. |
| `state` | Persistent mirror health tracking: counts successes/failures, derives a score that boosts or demotes mirror priority. Written to `$XDG_STATE_HOME/mip/state.json`. |
| `version` | Build-time vars (`-X`) for version/commit/date + runtime info. |
| `output` | Small JSON/line output helpers. |
| `completion` | Hardcoded bash/zsh/fish completion scripts. |

**Pull flow** (`runPull` in `main.go`):
1. Parse image reference → look up registry profile → generate rewrite candidates.
2. Apply health ranking from saved state, sort, append a low-priority "source" (original registry) fallback.
3. Probe all candidates concurrently (HTTP manifest HEAD/GET with timeout).
4. Sort results: OK > auth-required > failure, mirrors before source, fastest first.
5. Iterate candidates with retry: `engine.Pull` mirror image, verify digest (comparing pulled repo digests against both child and index digests from probe), `engine.Tag` to original name, `engine.Remove` mirror image.
6. Save updated mirror health state.

**Prefetch flow** (`runPrefetch` in `main.go`): extracts unique `FROM` images from a Dockerfile (skipping `scratch` and comments), then runs the pull pipeline for each sequentially. Useful before `docker build` to pre-seed images through mirrors.

**Exit codes**: 0=ok, 1=general, 2=invalid ref, 3=no usable mirror, 4=engine unavailable, 5=pull failed, 6=digest mismatch, 9=config error.

## Key design details

- **No config file needed at runtime.** `configs/mip.yaml` is embedded in the binary and serves as the default. User config augments/overrides it.
- **Move-flag-first arg handling.** The custom `moveFlagsFirst` function reorders args so flag-style arguments precede operands, working around Go's `flag` package requirement. This lets `mip pull nginx:1.27 --dry-run` work like `mip pull --dry-run nginx:1.27`.
- **`-f` shorthand.** For `prefetch`, both `--file` and `-f` are accepted (Go's flag package doesn't support shorthand natively, so both are registered as separate flags).
- **Engine abstraction** has a single `CLI` struct implementing `Engine` via `exec.Command`. The `fakeEngine` in `main_test.go` is the canonical test double.
- **Auth-required mirrors are treated as warnings, not errors.** They remain in the pull fallback chain (handles registries that require `docker login`).
- **Platform-aware manifest selection.** When `--platform` is passed, `probe` looks inside OCI index / manifest list JSON and resolves to the child manifest digest matching the requested platform.

## Skills directory

`skills/image-mirror-skill/` is an Agent Skill (packaged for `skit`) that guides AI agents through diagnosing and fixing image pull failures using `mip`.