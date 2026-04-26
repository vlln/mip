# Roadmap

## Phase 0: Specification

- Create product, CLI, architecture, and mirror model docs.
- Decide implementation language.
- Decide binary name.
- Define license and repository structure.

## Phase 1: MVP

Commands:

- `mip IMAGE`
- `mip pull IMAGE`
- `mip rewrite IMAGE`
- `mip probe IMAGE`
- `mip mirrors list`
- `mip config show`

Supported registries:

- Docker Hub
- GHCR
- Quay
- MCR
- registry.k8s.io
- GCR
- Elastic
- NVCR

Supported engines:

- Docker

Required behavior:

- Normalize image references.
- Generate mirror candidates.
- Probe manifests concurrently.
- Pull from the best candidate.
- Retag to the original image name.
- Emit text and JSON output.
- Return documented exit codes.

## Phase 2: Engine and Config Expansion

- Add Podman adapter. Done.
- Add nerdctl adapter. Done.
- Implement XDG config loading. Done.
- Add `prefer`, `exclude`, and per-registry config. Done.
- Implement local probe cache and historical mirror health scoring. Done.
- Add platform-aware probing. Done.

## Phase 3: Reliability

- Digest verification.
- Auth handling for registries and mirrors.
- Historical mirror health scoring.
- Retry backoff with `Retry-After` support.
- Structured error taxonomy.

## Phase 4: Distribution

- Static binaries for Linux, macOS, and Windows.
- Shell completions.
- Man page.
- Homebrew formula.
- Installation script with checksum verification.

## Open Decisions

- Implementation language: Go is currently the strongest fit for a fast static binary and registry API support.
- Binary name: `mip` is short, but may collide with existing tools. Alternatives: `imp`, `imgpull`, `mirpull`, `pullx`.
- Built-in mirror list: should be curated manually with source metadata and review dates.
- Whether to support configuring Docker daemon mirrors as a separate opt-in command.
