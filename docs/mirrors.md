# Mirror Model

## Registry Profile

A registry profile describes one source registry and the safe ways to rewrite image references for it.

```yaml
name: docker.io
aliases:
  - index.docker.io
  - registry-1.docker.io
default_namespace: library
mirrors:
  - name: daocloud-docker
    host: docker.m.daocloud.io
    mode: host-replace
    priority: 90
  - name: daocloud-prefix
    host: m.daocloud.io/docker.io
    mode: prefix
    priority: 80
```

## Rewrite Modes

### `host-replace`

Replace the registry host and keep the repository path.

```text
docker.io/library/nginx:1.27
=> docker.m.daocloud.io/library/nginx:1.27
```

### `prefix`

Prefix the full normalized source reference with a mirror namespace.

```text
docker.io/library/nginx:1.27
=> m.daocloud.io/docker.io/library/nginx:1.27
```

This is often safer for multi-registry mirrors because the source registry remains explicit.

### `template`

Reserved for registries that need special path mapping.

```text
template: "{{ .MirrorHost }}/{{ .Registry }}/{{ .Repository }}:{{ .Tag }}"
```

Do not add template rules until a concrete mirror requires them.

## Built-In Registry Scope

Initial built-in profiles:

- `docker.io`
- `ghcr.io`
- `quay.io`
- `mcr.microsoft.com`
- `registry.k8s.io`
- `gcr.io`
- `docker.elastic.co`
- `nvcr.io`

## Built-In Mirror Policy

Built-in mirrors must have:

- A public source URL documenting the service.
- A known rewrite mode.
- A last-reviewed date.
- A conservative default priority.
- A way for users to disable them.

Do not copy third-party mirror lists verbatim. Built-ins should be curated and documented, not scraped wholesale.

Example metadata:

```yaml
name: daocloud-prefix
host: m.daocloud.io/docker.io
mode: prefix
source: https://github.com/DaoCloud/public-image-mirror
reviewed_at: 2026-04-26
priority: 80
enabled_by_default: true
```

## User Configuration

Users can add or override mirrors:

```yaml
engine: docker
timeout: 8s
pull_timeout: 10m
parallel_probe: 6

registries:
  ghcr.io:
    mirrors:
      - name: company-cache-ghcr
        host: registry.example.com/ghcr.io
        mode: prefix
        priority: 100
```

Users can disable built-ins:

```yaml
disable_builtin_mirrors: true
```

Or disable one mirror:

```yaml
disabled_mirrors:
  - daocloud-prefix
```

Users can prefer or exclude mirrors by name or host:

```yaml
prefer:
  - company-cache-ghcr
exclude:
  - ghcr.m.daocloud.io
```

Preferred mirrors receive a large priority bonus before candidate sorting. Excluded mirrors are removed before rewrite and probe.

## Candidate Filtering

Candidates should be filtered before scoring:

- Excluded by user config.
- Unsupported rewrite mode.
- Registry mismatch.
- Requires authentication and no credentials are available.
- Known recent hard failure in local state.

HTTP status handling:

```text
200/307  usable
401      usable only if token flow succeeds or credentials exist
403      reject unless explicitly allowed
404      reject
429      penalize or reject depending on retry-after
5xx      penalize, retry later
```

## Digest Safety

When the source image is specified by digest, the selected mirror must resolve to the same digest.

When the source image is specified by tag, digest verification is best-effort unless the source registry can be probed successfully.
