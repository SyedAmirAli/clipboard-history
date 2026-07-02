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

# Wails links GTK/WebKit via cgo. If cgo is off (CGO_ENABLED=0 in `go env`) or
# no C compiler is installed, Go silently drops every `import "C"` file and the
# build dies with confusing "undefined" errors. Force cgo on and fail fast with
# a clear message if there's no compiler to honour it.
export CGO_ENABLED=1
if ! command -v "${CC:-cc}" >/dev/null 2>&1 && ! command -v gcc >/dev/null 2>&1; then
  echo "error: no C compiler found (need gcc/cc for cgo — GTK/WebKit)." >&2
  echo "       install it with: sudo apt install build-essential" >&2
  exit 1
fi

# Single source of truth for the version: wails.json's productVersion. Inject
# it into the binary so `clipd --version` matches the packaged release.
VERSION="$(sed -n 's/.*"productVersion"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' wails.json)"
VERSION="${VERSION:-dev}"

echo "==> Building clipd binary v${VERSION} (webkit2_41 build tag for modern Ubuntu/Mint)"
wails build -tags webkit2_41 -clean -ldflags="-s -w -X main.version=${VERSION}"

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
  # nfpm.yaml references ${VERSION}; nfpm expands env vars in its config, so
  # the package version always matches wails.json's productVersion.
  VERSION="${VERSION}" nfpm pkg --packager deb --config nfpm.yaml --target dist/
  echo "==> Package:"
  ls -lh dist/*.deb | awk '{print "    " $5 "  " $9}'
fi

echo "==> Done."
