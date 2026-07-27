#!/usr/bin/env bash
# Cross-compiles Git Flow Plus release binaries for every supported
# platform into dist/. Run from anywhere; paths are resolved relative to
# the repository root.
#
#   ./scripts/build.sh
#
# Version metadata is embedded into the binary via -ldflags (see
# `git flow version`). Override any of these environment variables before
# invoking the script — this is how CI supplies real values instead of
# the git-derived defaults used for local builds:
#   VERSION       e.g. "1.2.0"       (default: `git describe`, or 0.0.0-dev)
#   BUILD_NUMBER  e.g. a CI run number (default: "dev")
#   GIT_COMMIT    e.g. short SHA     (default: `git rev-parse --short HEAD`)
#   GIT_BRANCH    e.g. "main"        (default: `git rev-parse --abbrev-ref HEAD`)
#   BUILD_DATE    RFC 3339 UTC       (default: now)
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

DIST_DIR="dist"
PKG="./cmd/git-flow-plus"
LDFLAGS_PKG="github.com/hulhub/git-flow-plus/internal/cli"

VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "0.0.0-dev")}"
BUILD_NUMBER="${BUILD_NUMBER:-dev}"
GIT_COMMIT="${GIT_COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo unknown)}"
GIT_BRANCH="${GIT_BRANCH:-$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)}"
BUILD_DATE="${BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"

LDFLAGS="-s -w \
  -X ${LDFLAGS_PKG}.Version=${VERSION} \
  -X ${LDFLAGS_PKG}.BuildNumber=${BUILD_NUMBER} \
  -X ${LDFLAGS_PKG}.GitCommit=${GIT_COMMIT} \
  -X ${LDFLAGS_PKG}.GitBranch=${GIT_BRANCH} \
  -X ${LDFLAGS_PKG}.BuildDate=${BUILD_DATE}"

# GOOS / GOARCH / output-filename triples for every supported platform.
TARGETS=(
  "windows amd64 git-flow-plus-windows-amd64.exe"
  "windows arm64 git-flow-plus-windows-arm64.exe"
  "linux   amd64 git-flow-plus-linux-amd64"
  "linux   arm64 git-flow-plus-linux-arm64"
  "darwin  amd64 git-flow-plus-darwin-amd64"
  "darwin  arm64 git-flow-plus-darwin-arm64"
)

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

echo "Building Git Flow Plus ${VERSION} (commit ${GIT_COMMIT}, branch ${GIT_BRANCH})"

for target in "${TARGETS[@]}"; do
  read -r goos goarch out <<<"$target"
  echo "  -> ${goos}/${goarch}"
  GOOS="$goos" GOARCH="$goarch" CGO_ENABLED=0 \
    go build -trimpath -ldflags "$LDFLAGS" -o "${DIST_DIR}/${out}" "$PKG"
done

echo "Done. Binaries in ${DIST_DIR}/:"
ls -la "$DIST_DIR"
