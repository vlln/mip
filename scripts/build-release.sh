#!/usr/bin/env bash
set -euo pipefail

version="${VERSION:-dev}"
commit="${COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || printf none)}"
date="${DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
dist="${DIST:-dist}"

ldflags="-s -w"
ldflags+=" -X github.com/vlln/mip/internal/version.Version=${version}"
ldflags+=" -X github.com/vlln/mip/internal/version.Commit=${commit}"
ldflags+=" -X github.com/vlln/mip/internal/version.Date=${date}"

targets=(
  "linux/amd64"
  "linux/arm64"
  "darwin/amd64"
  "darwin/arm64"
  "windows/amd64"
  "windows/arm64"
)

rm -rf "${dist}"
mkdir -p "${dist}"

for target in "${targets[@]}"; do
  goos="${target%/*}"
  goarch="${target#*/}"
  name="mip_${version}_${goos}_${goarch}"
  bin="mip"
  if [[ "${goos}" == "windows" ]]; then
    bin="mip.exe"
  fi

  work="${dist}/${name}"
  mkdir -p "${work}"
  printf 'building %s/%s\n' "${goos}" "${goarch}"
  GOOS="${goos}" GOARCH="${goarch}" CGO_ENABLED=0 go build -trimpath -ldflags "${ldflags}" -o "${work}/${bin}" ./cmd/mip
  cp README.md "${work}/"
  cp -R docs "${work}/docs"
  mkdir -p "${work}/configs"
  cp configs/mip.yaml "${work}/configs/"

  if [[ "${goos}" == "windows" ]]; then
    (cd "${dist}" && zip -qr "${name}.zip" "${name}")
    rm -rf "${work}"
  else
    tar -C "${dist}" -czf "${dist}/${name}.tar.gz" "${name}"
    rm -rf "${work}"
  fi
done

(cd "${dist}" && sha256sum ./* > checksums.txt)
printf 'release artifacts written to %s\n' "${dist}"
