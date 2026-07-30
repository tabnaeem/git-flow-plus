#!/usr/bin/env bash
# Packages the darwin/amd64 + darwin/arm64 binaries (produced by
# `goreleaser build`/`goreleaser release` — see .goreleaser.yaml) into a
# single universal .pkg installer via lipo/pkgbuild/productbuild.
#
# macOS-only: those three tools don't exist anywhere else, so this
# cannot run (or be tested) on Windows/Linux — see Packaging.md. CI runs
# it in the macos-latest job; this is its first real execution.
#
#   ./scripts/package-macos-pkg.sh <version> <amd64-binary> <arm64-binary> <output.pkg>
#
# Example (from repo root, after goreleaser has produced dist/):
#   ./scripts/package-macos-pkg.sh 5.3.4.1.2 \
#     dist/git-flow-plus_darwin_amd64_v1/git-flow-plus \
#     dist/git-flow-plus_darwin_arm64_v8.0/git-flow-plus \
#     dist/installer/git-flow-plus-5.3.4.1.2-macos-universal.pkg
set -euo pipefail

if [ "$#" -ne 4 ]; then
  echo "usage: $0 <version> <amd64-binary> <arm64-binary> <output.pkg>" >&2
  exit 1
fi

VERSION="$1"
AMD64_BIN="$2"
ARM64_BIN="$3"
OUT_PKG="$4"

if [ "$(uname -s)" != "Darwin" ]; then
  echo "error: this script must run on macOS (uses lipo/pkgbuild/productbuild)" >&2
  exit 1
fi

for f in "$AMD64_BIN" "$ARM64_BIN"; do
  if [ ! -f "$f" ]; then
    echo "error: input binary not found: $f" >&2
    exit 1
  fi
done

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STAGE="$(mktemp -d)"
trap 'rm -rf "$STAGE"' EXIT

# /usr/local/bin is on the default PATH on both Intel and Apple Silicon
# Macs (unlike Homebrew's arch-specific /usr/local vs /opt/homebrew
# split) — a single universal binary there works unmodified on either.
INSTALL_LOCATION="/usr/local/bin"
PAYLOAD_DIR="$STAGE/payload$INSTALL_LOCATION"
mkdir -p "$PAYLOAD_DIR"

echo "Building universal binary from:"
echo "  amd64: $AMD64_BIN"
echo "  arm64: $ARM64_BIN"
lipo -create -output "$PAYLOAD_DIR/git-flow-plus" "$AMD64_BIN" "$ARM64_BIN"
chmod +x "$PAYLOAD_DIR/git-flow-plus"
lipo -info "$PAYLOAD_DIR/git-flow-plus"

# `git flow ...` only resolves as a Git subcommand if a binary literally
# named "git-flow" is on PATH — see internal/cli/doctor.go's PATH check
# and CommandReference.md. A symlink (rather than a second copy) keeps
# both names pointing at the one universal binary.
ln -s "git-flow-plus" "$PAYLOAD_DIR/git-flow"

SCRIPTS_DIR="$STAGE/scripts"
mkdir -p "$SCRIPTS_DIR"
cp "$ROOT_DIR/installer/windows/default-config.json" "$SCRIPTS_DIR/default-config.json"

# postinstall seeds a default config under ~/Library/Application Support
# and a logs directory under ~/Library/Logs — macOS's conventional
# per-user locations, mirroring the Windows installers' %APPDATA%/
# %LOCALAPPDATA% treatment (see installer/windows/gitflowplus.iss and
# .wxs). Mirrors internal/config.Default()'s exact JSON shape, not a
# placeholder; Git Flow Plus itself only reads the repo-local
# .gitflowplus/config.json today — see Packaging.md. Runs as root (pkg
# postinstall scripts always do), so it resolves the actual invoking
# user via $USER/sudo's SUDO_USER to seed *their* home directory rather
# than root's.
cat > "$SCRIPTS_DIR/postinstall" <<'POSTINSTALL'
#!/bin/bash
set -euo pipefail

REAL_USER="${SUDO_USER:-$USER}"
REAL_HOME=$(dscl . -read "/Users/$REAL_USER" NFSHomeDirectory | awk '{print $2}')
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

CONFIG_DIR="$REAL_HOME/Library/Application Support/GitFlowPlus"
LOGS_DIR="$REAL_HOME/Library/Logs/GitFlowPlus"

mkdir -p "$CONFIG_DIR" "$LOGS_DIR"
chown "$REAL_USER" "$CONFIG_DIR" "$LOGS_DIR"

if [ ! -f "$CONFIG_DIR/config.json" ]; then
  cp "$SCRIPT_DIR/default-config.json" "$CONFIG_DIR/config.json"
  chown "$REAL_USER" "$CONFIG_DIR/config.json"
fi

exit 0
POSTINSTALL
chmod +x "$SCRIPTS_DIR/postinstall"

IDENTIFIER="com.tabnaeem.gitflowplus"

pkgbuild \
  --root "$STAGE/payload" \
  --scripts "$SCRIPTS_DIR" \
  --identifier "$IDENTIFIER" \
  --version "$VERSION" \
  --install-location "/" \
  "$STAGE/component.pkg"

cat > "$STAGE/welcome.txt" <<WELCOME
Git Flow Plus $VERSION

Git Flow with enterprise release management: Sprint.Feature.ReleaseFix.
DevOps.QA versioning, mandatory release tagging, and a Feature Registry,
on top of the standard Git Flow branching model.

This installs a universal (Intel + Apple Silicon) binary to
$INSTALL_LOCATION and seeds a default configuration under
~/Library/Application Support/GitFlowPlus.

After installing, open a terminal and run:
  git flow doctor
  git flow version
WELCOME
cp "$ROOT_DIR/LICENSE" "$STAGE/license.txt"

cat > "$STAGE/distribution.xml" <<XML
<?xml version="1.0" encoding="utf-8"?>
<installer-gui-script minSpecVersion="1">
    <title>Git Flow Plus</title>
    <options customize="never" require-scripts="true" rootVolumeOnly="true" />
    <welcome file="welcome.txt" />
    <license file="license.txt" />
    <choices-outline>
        <line choice="default">
            <line choice="ChoiceGitFlowPlus" />
        </line>
    </choices-outline>
    <choice id="default" />
    <choice id="ChoiceGitFlowPlus" visible="false" title="Git Flow Plus">
        <pkg-ref id="$IDENTIFIER" />
    </choice>
    <pkg-ref id="$IDENTIFIER" version="$VERSION" onConclusion="none">component.pkg</pkg-ref>
</installer-gui-script>
XML

mkdir -p "$(dirname "$OUT_PKG")"
productbuild \
  --distribution "$STAGE/distribution.xml" \
  --package-path "$STAGE" \
  --resources "$STAGE" \
  "$OUT_PKG"

echo "Done: $OUT_PKG"
