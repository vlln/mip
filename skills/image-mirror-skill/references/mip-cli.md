# mip CLI Reference

## Common Workflows

Rewrite candidates:

```bash
mip rewrite nginx:1.27 --all
mip rewrite ghcr.io/actions/actions-runner:latest --plain --all
```

Probe mirror health:

```bash
mip probe nginx:1.27 --timeout 8s
mip probe hello-world:latest --platform linux/amd64 --json
```

Pull through the fastest usable mirror:

```bash
mip pull nginx:1.27
mip pull hello-world:latest --platform linux/amd64
mip pull ghcr.io/org/app:v1 --engine podman --dry-run
```

Inspect config and mirrors:

```bash
mip config show
mip mirrors list
mip mirrors list --registry registry.k8s.io
```

## Config

Default paths:

- `$XDG_CONFIG_HOME/mip/config.yaml`
- `~/.config/mip/config.yaml`

The official default config is distributed as `configs/mip.yaml` and is embedded
into `mip` for zero-config mirror use. If a user config exists or `--config` is
provided, that single config is used instead of merging with the official one.

Example:

```yaml
prefer:
  - company-cache
exclude:
  - docker.m.daocloud.io
registries:
  docker.io:
    mirrors:
      - name: company-cache
        host: registry.example.com/docker.io
```

Mirror entries can usually be written as host strings:

```yaml
registries:
  docker.io:
    mirrors:
      - docker.m.daocloud.io
      - m.daocloud.io/docker.io
```

## State

State path:

- `$XDG_STATE_HOME/mip/state.json`
- `~/.local/state/mip/state.json`

State records mirror success/failure, latency, status, digest, and errors. It only influences candidate priority; it never replaces live probe.
