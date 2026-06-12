#!/usr/bin/env bash
# Build the clipd binary and (optionally) a .deb package.
#
# Usage:
#   ./scripts/build.sh          # builds binary into ./build/bin/clipd
#   ./scripts/build.sh deb      # additionally builds dist/clipd_<v>_amd64.deb
#
# Prereqs:
#   - Go 1.22+
#   - Wails v2 CLI (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)
#   - Yarn
#   - System libs (see README — libgtk-3-dev, libwebkit2gtk-4.1-dev,
#     libx11-dev, libayatana-appindicator3-dev)
#   - nfpm (only for `deb` step): https://nfpm.goreleaser.com

set -euo pipefail
cd "$(dirname "$0")/.."

export PATH="/usr/local/go/bin:${HOME}/go/bin:${PATH}"
export GOCACHE="${GOCACHE:-/tmp/clipd-go-cache}"
export GOMODCACHE="${GOMODCACHE:-/tmp/clipd-go-mod}"

echo "==> Building clipd binary (webkit2_41 build tag for modern Ubuntu/Mint)"
wails build -tags webkit2_41 -clean -ldflags="-s -w"

echo "==> Binary size:"
ls -lh ./build/bin/clipd | awk '{print "    " $5 "  " $9}'

if [[ "${1:-}" == "deb" ]]; then
  if ! command -v nfpm >/dev/null 2>&1; then
    echo "error: nfpm is not installed. Install via:"
    echo "       go install github.com/goreleaser/nfpm/v2/cmd/nfpm@latest"
    exit 1
  fi
  mkdir -p dist
  echo "==> Packaging .deb"
  nfpm pkg --packager deb --config nfpm.yaml --target dist/
  echo "==> Package:"
  ls -lh dist/*.deb | awk '{print "    " $5 "  " $9}'
fi

echo "==> Done."
