#!/usr/bin/env bash
# Packages the binaries in dist/ (produced by scripts/build.sh) into
# per-platform release archives under dist/archives/: a .zip for Windows
# targets (the platform convention), a .tar.gz for Linux/macOS targets.
# Each archive also bundles README.md so a download is self-explanatory
# without needing the repository.
#
#   ./scripts/build.sh && ./scripts/package.sh
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

DIST_DIR="${ROOT_DIR}/dist"
ARCHIVE_DIR="${DIST_DIR}/archives"
NAME="git-flow-plus"

if [ ! -d "$DIST_DIR" ] || [ -z "$(find "$DIST_DIR" -maxdepth 1 -name "${NAME}-*" 2>/dev/null)" ]; then
  echo "no binaries found in ${DIST_DIR}/ — run scripts/build.sh first" >&2
  exit 1
fi

rm -rf "$ARCHIVE_DIR"
mkdir -p "$ARCHIVE_DIR"

# platform/arch/binary-filename triples, matching scripts/build.sh's output.
TARGETS=(
  "windows amd64 git-flow-plus-windows-amd64.exe"
  "windows arm64 git-flow-plus-windows-arm64.exe"
  "linux   amd64 git-flow-plus-linux-amd64"
  "linux   arm64 git-flow-plus-linux-arm64"
  "darwin  amd64 git-flow-plus-darwin-amd64"
  "darwin  arm64 git-flow-plus-darwin-arm64"
)

for target in "${TARGETS[@]}"; do
  read -r goos goarch bin <<<"$target"
  src="${DIST_DIR}/${bin}"
  if [ ! -f "$src" ]; then
    echo "  skipping ${goos}/${goarch}: ${src} not found"
    continue
  fi

  stage="$(mktemp -d)"

  if [ "$goos" = "windows" ]; then
    staged_bin="${stage}/${NAME}.exe"
  else
    staged_bin="${stage}/${NAME}"
  fi
  cp "$src" "$staged_bin"
  chmod +x "$staged_bin"
  cp "${ROOT_DIR}/README.md" "$stage/"

  archive_base="${NAME}-${goos}-${goarch}"
  if [ "$goos" = "windows" ]; then
    out="${ARCHIVE_DIR}/${archive_base}.zip"
    echo "  -> ${out}"
    (cd "$stage" && zip -q -r "$out" .)
  else
    out="${ARCHIVE_DIR}/${archive_base}.tar.gz"
    echo "  -> ${out}"
    tar -C "$stage" -czf "$out" .
  fi

  rm -rf "$stage"
done

echo "Done. Archives in ${ARCHIVE_DIR}/:"
ls -la "$ARCHIVE_DIR"
