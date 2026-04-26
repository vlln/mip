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
mip rewrite nginx:1.27 --best --plain
```

Machine-readable output:

```bash
mip pull nginx:1.27 --json
```

## Global Flags

```text
--config PATH        Load a specific config file
--engine NAME        docker, podman, or nerdctl
--timeout DURATION   Per network probe timeout
--pull-timeout DURATION
--retries N
--platform PLATFORM  Example: linux/amd64
--json               Emit JSON result
--quiet              Suppress progress output
--verbose            Include candidate and decision details
--no-color           Disable ANSI color
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
--verify-digest      Verify digest when the source digest is known
--no-retag           Keep the mirrored image name locally
--prefer HOST        Prefer a mirror host, repeatable
--exclude HOST       Exclude a mirror host, repeatable
--max-candidates N   Limit pull attempts after probing
```

Default behavior:

1. Normalize `IMAGE`.
2. Generate candidate mirror image references.
3. Probe candidates concurrently.
4. Pull the highest-ranked candidate.
5. Retag to the original normalized image name if needed.
6. Remove the temporary mirror tag unless `--no-retag` is set.

## `rewrite`

Print rewritten image references without pulling.

```bash
mip rewrite IMAGE
mip rewrite IMAGE --all
mip rewrite IMAGE --best --plain
```

## `probe`

Probe source and mirror candidates.

```bash
mip probe IMAGE
mip probe IMAGE --json
```

The probe command should check manifest availability and latency. It should not pull blobs.

## `mirrors list`

List built-in and configured mirrors.

```bash
mip mirrors list
mip mirrors list --registry ghcr.io
mip mirrors list --json
```

## `config show`

Print the merged config from defaults, config file, and environment.

```bash
mip config show
mip config show --config ./mip.yaml
```

The output includes `effective_profiles`, which is the registry profile list after applying built-ins, user mirrors, disabled mirrors, prefer, and exclude rules.

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
