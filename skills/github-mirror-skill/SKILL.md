---
name: github-mirror-skill
description: Use this skill when you need to accelerate or troubleshoot GitHub access that is slow, unstable, or blocked, including git clone, release downloads, raw file access, and gist retrieval, using mirror-aware URL rewriting, probing, and transparent proxy setup.
license: MIT
metadata:
  author: vlln
  version: "0.1.0"
requires:
  bins:
    - gip
    - git
---

# GitHub Mirror Skill

Use this skill when GitHub access is slow, unstable, or blocked. `gip` rewrites
GitHub URLs to known mirrors, probes reachability, routes git clone through the
container engine, downloads files, and can set up transparent git `insteadOf`
proxying.

## Trigger Keywords

- git clone slow, git clone timeout, git clone blocked
- github download slow, github release download failed
- github mirror, github proxy, github 加速, github 镜像
- raw.githubusercontent.com blocked
- gip

## Capabilities

- **Rewrite** GitHub URLs to mirror candidates for a given host.
- **Probe** mirror reachability and latency without cloning or downloading.
- **Clone** repositories through the fastest reachable mirror, resetting the
  origin remote back to the original URL.
- **Download** release assets, raw files, gists, and archives through mirrors.
- **Install** transparent git `insteadOf` proxy so all git operations to a host
  automatically route through the best mirror.
- **Inspect** configured mirrors and runtime config.

For command syntax, flags, config format, and state file paths, read
`$_S/references/gip-cli.md`.

## Workflow

When a user reports slow or failed GitHub access:

1. **Probe first.** Run `gip probe <URL> --timeout 8s` to find reachable
   mirrors and compare latency. Use `--json` for structured output.
2. **Show mirrors.** Run `gip rewrite <URL> --all` to list all rewrite
   candidates. If the host has no configured mirrors, report it.
3. **Dry-run.** Run `gip clone <URL> --dry-run` or `gip get <URL> --dry-run`
   to confirm the selected mirror before executing.
4. **Execute.** Run `gip clone <URL>` for repositories or `gip get <URL>` for
   files. The tool will route through the fastest working mirror.
5. **Verify.** After clone, confirm the repo is available locally with the
   correct remote. After download, check the file exists and has expected size.

For creating a persistent transparent proxy, use `gip install` to set up git
`insteadOf` config:

```bash
gip install
# After this, all git clone/fetch/pull to github.com go through the fastest mirror
gip uninstall   # remove the proxy
```

## Gotchas

- **Mirrors may not support git clone.** Some mirrors only proxy file downloads
  (HTTP), not git smart-HTTP protocol. `gip probe` correctly distinguishes these
  by validating git advertisement responses. A mirror that works for `gip get`
  may not work for `gip clone`.
- **Subdomain URLs are aliased to github.com.** `raw.githubusercontent.com`,
  `api.github.com`, `gist.githubusercontent.com`, etc. are all treated as
  aliases of `github.com` for mirror profile lookup. Rewrite candidates are
  generated accordingly.
- **Prefix vs host-replace mirrors.** Prefix mirrors require the full original
  URL in the path (e.g. `https://ghproxy.com/https://github.com/user/repo`).
  Host-replace mirrors swap the domain (e.g. `https://kkgithub.com/user/repo`).
- **Mirror availability changes frequently.** The default mirror list in
  `configs/gip.yaml` may become stale. Always probe before relying on a mirror.
- **GitHub.com may be reachable when mirrors are not.** In some network
  environments, direct GitHub access works while all mirrors are blocked. `gip`
  always includes the source as a fallback candidate.
- **Private repositories require authentication.** For clone of private repos
  through mirrors, use `git clone https://user:token@mirror/https://github.com/...`.
  `gip clone` does not handle authentication forwarding.
- **Probe, rewrite, and config inspection work without git.** Git is only
  needed for actual clone operations.
- **Mirror is not a security boundary.** Public mirrors can serve different
  content than GitHub. For production, verify downloaded content against
  expected checksums.