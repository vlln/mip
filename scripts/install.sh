#!/usr/bin/env sh
set -eu

repo="${MIP_REPO:-vlln/mip}"
version="${MIP_VERSION:-latest}"
bindir="${MIP_BINDIR:-/usr/local/bin}"

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

if command -v curl >/dev/null 2>&1; then
  fetch='curl -fsSL'
elif command -v wget >/dev/null 2>&1; then
  fetch='wget -qO-'
else
  echo "missing required command: curl or wget" >&2
  exit 1
fi

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

if [ "$version" = "latest" ]; then
  base="https://github.com/${repo}/releases/latest/download"
  artifact_version="latest"
else
  base="https://github.com/${repo}/releases/download/v${version}"
  artifact_version="$version"
fi

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
"${bindir}/mip" version

