# CLI Specification

## Command Shape

The binary name is currently `mip`.

```bash
mip IMAGE
mip pull IMAGE
mip rewrite IMAGE
mip probe IMAGE
mip mirrors list
mip config show
```

`pull` is the default command when the first argument looks like an image reference.

## Examples

```bash
mip nginx:1.27
mip ghcr.io/actions/actions-runner:latest
mip quay.io/prometheus/prometheus:v2.53.0
mip registry.k8s.io/pause:3.10
mip mcr.microsoft.com/dotnet/runtime:8.0
```

Use a specific engine:

```bash
mip pull nginx:1.27 --engine docker
mip pull nginx:1.27 --engine podman
mip pull nginx:1.27 --engine nerdctl
```

Dry run:

```bash
mip pull ghcr.io/org/app:v1 --dry-run
```

Scriptable rewrite:

```bash
mip rewrite nginx:1.27 --plain
mip rewrite nginx:1.27 --all --plain
```

Machine-readable output:

```bash
mip pull nginx:1.27 --json
```

## Common Flags

```text
--config PATH        Load a specific config file
--timeout DURATION   Per network probe timeout
--platform PLATFORM  Example: linux/amd64
--json               Emit JSON result
```

`--config` may be placed before or after the image argument.

## `pull`

Pull an image through the best available candidate.

```bash
mip pull IMAGE [flags]
```

Important flags:

```text
--dry-run            Show candidate plan without pulling
--engine NAME        docker, podman, or nerdctl
--platform PLATFORM  Platform passed to probe and pull
--timeout DURATION   Per candidate probe timeout
--pull-timeout DURATION
--concurrency N      Maximum concurrent probes
--retries N          Pull attempts per candidate
--json               Emit JSON result
--no-retag           Keep the mirrored image name locally
--no-verify-digest   Skip digest verification after pull
```

Default behavior:

1. Normalize `IMAGE`.
2. Generate candidate mirror image references.
3. Add the source registry as the final fallback candidate.
4. Probe candidates concurrently.
5. Pull the fastest reachable candidate.
6. Retry failed pulls according to `--retries`, then fall back to the next reachable candidate.
7. Retag to the original normalized image name if needed.
8. Remove the temporary mirror tag unless `--no-retag` is set.

## `rewrite`

Print rewritten image references without pulling.

```bash
mip rewrite IMAGE
mip rewrite IMAGE --all
mip rewrite IMAGE --plain
mip rewrite IMAGE --all --plain
```

## `probe`

Probe source and mirror candidates.

```bash
mip probe IMAGE
mip probe IMAGE --json
```

The probe command should check manifest availability and latency. It should not pull blobs.

## `mirrors list`

List configured mirrors.

```bash
mip mirrors list
mip mirrors list --registry ghcr.io
mip mirrors list --json
```

## `config show`

Print the effective config from defaults and the selected config file.

```bash
mip config show
mip config show --config ./mip.yaml
```

The output includes `effective_profiles`, which is the registry profile list after applying prefer and exclude rules.

## Output

Default human output should be compact:

```text
image: docker.io/library/nginx:1.27
selected: docker.m.daocloud.io/library/nginx:1.27
engine: docker
status: pulled
elapsed: 8.2s
```

Progress and diagnostics go to stderr. Stable results go to stdout.

JSON output:

```json
{
  "image": "docker.io/library/nginx:1.27",
  "selected": "docker.m.daocloud.io/library/nginx:1.27",
  "registry": "docker.io",
  "mirror": "docker.m.daocloud.io",
  "engine": "docker",
  "status": "pulled",
  "elapsed_ms": 8200
}
```

## Exit Codes

```text
0  success
1  general error
2  invalid image reference
3  no usable mirror
4  engine unavailable
5  pull failed
6  digest verification failed
7  authentication required
8  network timeout
9  configuration error
```
