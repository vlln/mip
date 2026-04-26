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
  - dockerproxy.cool
  - m.daocloud.io/docker.io
  - docker.1ms.run
```

## Rewrite Modes

`mip` infers the rewrite mode from each mirror host. A host path that ends with
the source registry uses `prefix`; other hosts use `host-replace`.

### `host-replace`

Replace the registry host and keep the repository path.

```text
docker.io/library/nginx:1.27
=> dockerproxy.cool/library/nginx:1.27
```

### `prefix`

Prefix the full normalized source reference with a mirror namespace.

```text
docker.io/library/nginx:1.27
=> m.daocloud.io/docker.io/library/nginx:1.27
```

This is often safer for multi-registry mirrors because the source registry remains explicit.

## Official Registry Scope

The official default config in `configs/mip.yaml` currently covers:

- `docker.io`
- `ghcr.io`
- `quay.io`
- `mcr.microsoft.com`
- `registry.k8s.io`
- `k8s.gcr.io`
- `gcr.io`
- `dhi.io`
- `docker.elastic.co`
- `nvcr.io`
- `registry.ollama.ai`

## Official Mirror Policy

Official default mirrors must have:

- A host that works with automatic rewrite-mode inference.
- No known intranet-only, hard rate-limit, or incompatible access restriction in
  the reference data used for curation.
- The same image-reference rewrite semantics as `mip`.

Do not copy third-party mirror lists verbatim. Official defaults should be curated
and documented, not scraped wholesale. Docker daemon `registry-mirrors` can be a
useful signal, but a daemon mirror URL is not enough by itself because `mip`
rewrites image references directly.

Mirrors that return `401` are kept as authentication-required candidates when
they otherwise match the registry rewrite model. `mip probe` reports them as
warnings because users can make them usable through the selected engine, for
example `docker login gcr.m.daocloud.io`. `mip` currently supports anonymous
bearer-token challenges during manifest probing, but it does not read Docker
credential stores or send username/password credentials to third-party mirrors.

Example mirror entry:

```yaml
- m.daocloud.io/docker.io
```

## User Configuration

Users can maintain a single config with the mirror hosts they want:

```yaml
registries:
  ghcr.io:
    mirrors:
      - registry.example.com/ghcr.io
```

Users can prefer or exclude mirrors by host:

```yaml
prefer:
  - registry.example.com/ghcr.io
exclude:
  - ghcr.m.daocloud.io
```

Preferred mirrors receive a large priority bonus before candidate sorting. Excluded mirrors are removed before rewrite and probe.

## Candidate Filtering

Candidates should be filtered before scoring:

- Excluded by user config.
- Unsupported rewrite mode.
- Registry mismatch.
- Authentication required and no engine credentials are available: warn, keep as
  a pull candidate, and let the engine decide.
- Known recent hard failure in local state.

HTTP status handling:

```text
200/307  usable
401      warn as auth-required; usable by pull if the engine is logged in
403      reject unless explicitly allowed
404      reject
429      penalize or reject depending on retry-after
5xx      penalize, retry later
```

## Digest Safety

When the source image is specified by digest, the selected mirror must resolve to the same digest.

When the source image is specified by tag, digest verification is best-effort unless the source registry can be probed successfully.
