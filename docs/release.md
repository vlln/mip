# Release

## Local Build

```bash
make test
make build
./bin/mip version
```

`make build` writes the local binary to `bin/mip`.

## Release Archives

```bash
make release VERSION=0.1.0
```

This creates archives under `dist/` for:

- `linux/amd64`
- `linux/arm64`
- `darwin/amd64`
- `darwin/arm64`
- `windows/amd64`
- `windows/arm64`

Each archive includes:

- `mip` or `mip.exe`
- `README.md`
- `docs/`

`dist/checksums.txt` contains SHA-256 checksums.

## Install Script

The install script downloads a GitHub release archive, checks SHA-256 sums, and installs `mip`. When `MIP_VERSION=latest`, it resolves the latest GitHub release tag before selecting the archive name.

```bash
MIP_VERSION=0.1.0 MIP_BINDIR="$HOME/.local/bin" ./scripts/install.sh
```

Environment variables:

- `MIP_VERSION`: release version without the leading `v`; defaults to `latest`
- `MIP_REPO`: GitHub repository, defaults to `vlln/mip`
- `MIP_BINDIR`: install directory, defaults to `/usr/local/bin`

## Shell Completion

```bash
mip completion bash
mip completion zsh
mip completion fish
```

Example installation:

```bash
mip completion bash > ~/.local/share/bash-completion/completions/mip
mip completion zsh > ~/.zfunc/_mip
mip completion fish > ~/.config/fish/completions/mip.fish
```

## Version Metadata

Version metadata is injected with Go linker flags:

```text
github.com/vlln/mip/internal/version.Version
github.com/vlln/mip/internal/version.Commit
github.com/vlln/mip/internal/version.Date
```

Check it with:

```bash
mip version
mip version --json
```

## CI

`.github/workflows/ci.yml` runs:

- `go test ./...`
- `make build VERSION=ci`
- smoke tests for `version` and `rewrite`
- release archive dry-run for `v*` tags
