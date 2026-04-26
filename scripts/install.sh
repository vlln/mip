#!/usr/bin/env sh
set -eu

repo="${MIP_REPO:-vlln/mip}"
default_bindir="${MIP_DEFAULT_BINDIR:-/usr/local/bin}"

need() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing required command: $1" >&2
    exit 1
  }
}

need uname
need tar
need sha256sum
need mktemp
need sed
need install
need dirname

if command -v curl >/dev/null 2>&1; then
  fetch='curl -fsSL'
elif command -v wget >/dev/null 2>&1; then
  fetch='wget -qO-'
else
  echo "missing required command: curl or wget" >&2
  exit 1
fi

resolve_latest_version() {
  tag="$($fetch "https://api.github.com/repos/${repo}/releases/latest" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
  tag="${tag#v}"
  if [ -z "$tag" ]; then
    echo "could not resolve latest release for ${repo}" >&2
    exit 1
  fi
  printf '%s\n' "$tag"
}

choose_bindir() {
  if [ -n "${MIP_BINDIR:-}" ]; then
    printf '%s\n' "$MIP_BINDIR"
    return
  fi

  if [ -d "$default_bindir" ] && [ -w "$default_bindir" ]; then
    printf '%s\n' "$default_bindir"
    return
  fi

  if [ ! -d "$default_bindir" ]; then
    parent="$(dirname "$default_bindir")"
    if [ -d "$parent" ] && [ -w "$parent" ]; then
      printf '%s\n' "$default_bindir"
      return
    fi
  fi

  if [ -z "${HOME:-}" ]; then
    echo "default install directory ${default_bindir} is not writable and HOME is not set" >&2
    echo "set MIP_BINDIR to a writable directory and retry" >&2
    exit 1
  fi

  printf '%s\n' "${HOME}/.local/bin"
}

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"

case "$os" in
  linux) os="linux" ;;
  darwin) os="darwin" ;;
  *) echo "unsupported OS: $os" >&2; exit 1 ;;
esac

case "$arch" in
  x86_64|amd64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *) echo "unsupported architecture: $arch" >&2; exit 1 ;;
esac

version="$(resolve_latest_version)"
bindir="$(choose_bindir)"

base="https://github.com/${repo}/releases/download/v${version}"
artifact_version="$version"
name="mip_${artifact_version}_${os}_${arch}"
archive="${name}.tar.gz"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT INT TERM

echo "downloading ${archive}"
$fetch "${base}/${archive}" > "${tmp}/${archive}"
$fetch "${base}/checksums.txt" > "${tmp}/checksums.txt"

(cd "$tmp" && sha256sum -c --ignore-missing checksums.txt)
tar -xzf "${tmp}/${archive}" -C "$tmp"

mkdir -p "$bindir"
install -m 0755 "${tmp}/${name}/mip" "${bindir}/mip"
echo "installed ${bindir}/mip"
case ":${PATH:-}:" in
  *":${bindir}:"*) ;;
  *) echo "note: ${bindir} is not in PATH; add it before running mip directly" >&2 ;;
esac
"${bindir}/mip" version
