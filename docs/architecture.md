# Architecture

## Overview

`mip` is a local command-line orchestrator. It does not implement a registry and does not modify the container engine daemon configuration by default.

High-level flow:

```text
CLI args
  -> image reference parser
  -> registry profile lookup
  -> candidate generator
  -> manifest prober
  -> candidate scorer
  -> engine adapter pull
  -> retag and cleanup
  -> result output
```

## Suggested Package Layout

```text
cmd/mip/                 CLI entrypoint
internal/ref/            image reference parser and normalizer
internal/registry/       registry profiles and mirror rewrite rules
internal/probe/          manifest probe, auth challenge, latency measurement
internal/score/          candidate ranking
internal/engine/         docker, podman, nerdctl adapters
internal/config/         XDG config loading and merging
internal/state/          probe cache and mirror health history
internal/output/         text, JSON, and plain output formats
```

The package names are intentionally generic. The binary name can change without reshaping internals.

## Reference Normalization

The parser must convert user input into a canonical form.

Examples:

```text
nginx:1.27
=> docker.io/library/nginx:1.27

redis
=> docker.io/library/redis:latest

ghcr.io/org/app:v1
=> ghcr.io/org/app:v1

registry.k8s.io/pause@sha256:...
=> registry.k8s.io/pause@sha256:...
```

Avoid ad hoc string splitting in command code. All downstream modules should receive a parsed reference object.

## Candidate Generation

Candidate generation is registry-aware. A registry profile defines aliases, default namespace behavior, and rewrite modes.

The candidate generator receives:

- A normalized image reference.
- A registry profile.
- User preference and exclusion rules.

It returns ordered candidates but does not perform network I/O.

## Probing

The prober checks manifest availability and latency. It should support:

- Docker Registry HTTP API v2.
- `WWW-Authenticate` bearer token challenges.
- Manifest list and OCI index media types.
- Platform-aware manifest selection when `--platform` is set.
- Per-host timeout.
- Bounded concurrency.

The prober should not pull image blobs.

When `--platform` is set, the prober reads manifest list or OCI index responses and selects the matching child manifest digest. The result keeps both:

- `digest`: selected platform digest when available
- `index_digest`: original index digest

Docker-compatible engines may expose only the index digest in `image inspect`, so pull verification accepts either digest while reporting both.

## Scoring

Initial scoring can be simple and explainable:

```text
score = base
      - latency_penalty
      - rate_limit_penalty
      - historical_failure_penalty
      + user_preference_bonus
      + digest_match_bonus
```

The score explanation should be available in verbose or JSON output.

## Engine Adapters

Supported engines:

- Docker
- Podman
- nerdctl

Each adapter should expose:

```text
Available() error
Pull(image, platform, timeout) error
Tag(source, target) error
Remove(image) error
InspectDigest(image) result
```

Engine command output should stream to stderr in interactive mode and be captured in JSON/quiet mode.

## Retagging

If a mirrored image reference differs from the normalized original, the tool should retag after a successful pull.

Example:

```text
pulled: docker.m.daocloud.io/library/nginx:1.27
target: docker.io/library/nginx:1.27
```

For a user input of `nginx:1.27`, the final local tag should also support the familiar short name if the selected engine uses that convention.

## Persistence

Use XDG paths:

```text
$XDG_CONFIG_HOME/mip/config.yaml
$XDG_CACHE_HOME/mip/probe-cache.json
$XDG_STATE_HOME/mip/state.json
```

Fallback:

```text
~/.config/mip/config.yaml
~/.cache/mip/probe-cache.json
~/.local/state/mip/state.json
```

State should be an optimization only. The tool must work if cache/state files are missing.

The state file stores per-candidate mirror health:

- success and failure counts
- last HTTP status
- last latency
- last digest
- last error
- update timestamp

Historical health influences candidate priority before live probing. It never replaces live probing.
