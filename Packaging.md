# Packaging

How every distributable artifact — binaries, archives, checksums,
`.deb`/`.rpm`, the Windows installer, the macOS `.pkg` — is actually
assembled. See [Building.md](Building.md) for the commands, and
[ReleaseProcess.md](ReleaseProcess.md) for how this all fires
automatically on a tag push.

## GoReleaser: the core pipeline

`.goreleaser.yaml` (repo root) is the single source of truth for
everything Go-native: cross-compiling all six `GOOS`/`GOARCH`
combinations, archiving (`.zip` for Windows, `.tar.gz` for Linux/macOS),
generating `checksums.txt` (SHA256, one manifest covering every
archive), packaging `.deb`/`.rpm` via its built-in `nfpm` integration,
generating a changelog, and creating the GitHub Release itself. This
replaced a hand-maintained `scripts/build.sh`/`build.ps1` +
`scripts/package.sh`/`package.ps1` pair — the same cross-compile/archive
logic, written twice, once per shell. One declarative config is the
single source of truth instead.

### Linux: `.deb`/`.rpm`

The `nfpms:` block in `.goreleaser.yaml` produces both formats from the
same binary GoReleaser already built — no hand-written `debian/control`
or `.spec` file. Metadata (vendor, homepage, maintainer, license,
description) lives directly in that block. Both packages install to
`/usr/bin/git-flow-plus` and include a `contents:` entry creating a
`/usr/bin/git-flow` **symlink** — that's the literal filename Git looks
for on `PATH` to resolve `git flow ...` as a subcommand (see
`internal/cli/doctor.go`'s `PATH` check and
[CommandReference.md](CommandReference.md)). A symlink, not a second
copy, since Linux filesystems make that essentially free.

## Windows: installer choice

**Decision:** [NSIS](https://nsis.sourceforge.io/) (Nullsoft Scriptable
Install System), producing a single `GitFlowPlusSetup_v<version>_x64.exe`
— the same installer technology (or the same category — a lightweight,
scriptable `.exe` wizard) GitHub CLI, Terraform, and Docker CLI ship on
Windows. This is a deliberate project choice, not a comparison-driven
one: Inno Setup and WiX are explicitly not used here. Source:
[build/windows/installer.nsi](build/windows/installer.nsi),
[build/windows/create-installer.ps1](build/windows/create-installer.ps1).

Full user-facing behavior is documented in
[WindowsInstallation.md](WindowsInstallation.md). Implementation notes:

### Executable icon and version resource

Before NSIS ever runs, [go-winres](https://github.com/tc-hib/go-winres)
embeds `build/windows/icon.ico` and version metadata directly into
`git-flow-plus.exe` itself (not just the installer wrapper) — a
`before.hooks` entry in `.goreleaser.yaml` runs `go-winres simply
--icon build/windows/icon.ico --product-version git-tag ...`, which
writes a `.syso` Windows resource object next to
`cmd/git-flow-plus/main.go`. Go's build toolchain links a `.syso` in
automatically for any `windows` target with no special `go build` flags
— it just has to be present. `--product-version git-tag`/`--file-version
git-tag` pull the version directly from the tag being released, and
gracefully fall back to `0.0.0` for local snapshot builds with no tags
(matching GoReleaser's own `v0.0.0` snapshot convention). `.syso` files
are build output, not source — `.gitignore`d, regenerated on every
build.

The installer itself (`installer.nsi`) embeds the **same** icon
(`MUI_ICON`/`MUI_UNICON`) and its own `VIProductVersion`/
`VIAddVersionKey` block, so both the installer `.exe` and the installed
`git-flow-plus.exe` show correct icon/version/publisher metadata in
Windows Explorer's Properties dialog.

### PATH management

NSIS's stock distribution does **not** bundle the community `EnVar`
plugin that most PATH-manipulation tutorials assume — confirmed by
inspecting `C:\Program Files (x86)\NSIS\Plugins\x86-unicode\` directly,
which contains only stock plugins. Rather than take on an extra plugin
dependency, PATH add/remove is hand-written NSIS operating on
`HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Environment`
(`WriteRegExpandStr`, broadcasting `WM_WININICHANGE` so already-open
programs pick up the change without a reboot):

- **Add** is idempotent — a `PathContains` check runs first, so
  reinstalling or repairing never produces a duplicate entry.
- **Remove** explicitly handles all four positions an entry can be in
  (only entry / first / last / middle), correctly re-joining the two
  neighbors with a single separator when removing from the middle. An
  earlier, simpler version that padded both sides of the haystack and
  spliced out the padded match looked correct but silently ate the
  separator between the two remaining neighbors on a middle removal —
  caught before it ever shipped by a throwaway, unelevated
  (`RequestExecutionLevel user`) NSIS test harness that exercised all
  four cases against a scratch `HKCU` registry value instead of the
  real machine PATH, so it could run without admin rights or risk to a
  real environment. See the comments directly above
  `un.RemoveFromPath` in `installer.nsi` for the corrected algorithm.

### Add/Remove Programs

Standard `WriteRegStr`/`WriteRegDWORD` calls under
`Software\Microsoft\Windows\CurrentVersion\Uninstall\GitFlowPlus`
(`DisplayName`, `DisplayVersion`, `Publisher`, `InstallLocation`,
`UninstallString`, `QuietUninstallString`, `NoModify`/`NoRepair` = 1,
and `EstimatedSize` via `FileFunc.nsh`'s `${GetSize}` macro) — this is
what makes the install show up correctly in Settings → Apps and support
a one-click uninstall from there.

### Upgrade detection

`.onInit` reads `UninstallString` from the same registry key. If a
previous install is found, it prompts for confirmation (skipped
entirely under `/S`) and then `ExecWait`s that prior uninstaller with
`/S _?=$INSTDIR` before continuing — the standard "self-upgrade" NSIS
recipe, always machine-wide since the installer only supports
`RequestExecutionLevel admin` (no per-user mode, unlike the project's
earlier Inno Setup prototype).

### Wired into GoReleaser: `hooks.post` and `extra_files`

The installer is not a separate, after-the-fact step — it's built
*during* GoReleaser's own build phase and published *by* GoReleaser's
own release step, via two ordinary GoReleaser mechanisms:

- **`builds[].hooks.post`** in `.goreleaser.yaml` runs
  [build/windows/build-installer-hook.ps1](build/windows/build-installer-hook.ps1)
  once for every `(GOOS, GOARCH)` target the build produces — six times
  in one `goreleaser release` run. The hook is a no-op for every target
  except `windows/amd64` (the only one `installer.nsi` packages); for
  that one target it calls `create-installer.ps1` with the binary
  GoReleaser just wrote (`{{ .Path }}`) and the version being released
  (`{{ .Version }}`), producing
  `dist/installer/GitFlowPlusSetup_v<version>_x64.exe` well before
  GoReleaser reaches its archive/checksum/publish phases — a *post*-
  build hook runs after that target's binary exists, not after the
  whole pipeline finishes.
- **`release.extra_files`** and **`checksum.extra_files`** both glob
  `dist/installer/GitFlowPlusSetup_*.exe`, so the installer gets
  uploaded to the GitHub Release and included in `checksums.txt`
  exactly like any GoReleaser-native artifact, with no manual copy step
  anywhere in between.

The practical effect: **a hook that fails aborts the entire
`goreleaser release` command**, confirmed empirically against
GoReleaser v2.17.0 during development (a deliberately-failing hook
produced `build failed`, not a partial release). So there's no separate
"did the installer actually get built" check to maintain — if
`create-installer.ps1` doesn't produce the expected file (it verifies
this itself and throws if not), the release never reaches the publish
step at all. This is also why `.goreleaser.yaml` must run on a Windows
host with `makensis` on `PATH` — see
[ReleaseProcess.md](ReleaseProcess.md) for why that means the whole
`goreleaser` CI job runs on `windows-latest`, not `ubuntu-latest`.

### Building locally

The normal path is simply running GoReleaser itself from a Windows
machine with NSIS installed — the `hooks.post` above builds the
installer as part of the same command, no separate invocation needed:

```powershell
goreleaser release --snapshot --clean   # or `make package`
# → dist\installer\GitFlowPlusSetup_v<version>_x64.exe, already in dist\checksums.txt
```

To iterate on just the installer itself without a full 6-platform
cross-compile, call the same script the hook calls directly:

```powershell
# 1. Build the binary (see Building.md for the full ldflags-stamped command)
mkdir dist\windows-x64 -Force
$env:GOOS = "windows"; $env:GOARCH = "amd64"; $env:CGO_ENABLED = "0"
go build -o dist\windows-x64\git-flow-plus.exe .\cmd\git-flow-plus

# 2. Build the installer
.\build\windows\create-installer.ps1 -Version 1.4.0 -BinDir dist\windows-x64 -OutDir dist\installer
```

`create-installer.ps1` stages nothing itself (NSIS's `File` directives
read the binary directly from `-BinDir`, and `README.txt`/`license.txt`
are checked-in/synced-from-`LICENSE` respectively) — it normalizes the
version string into NSIS's `/DVERSION`/`/DFILEVERSION` preprocessor
defines, locates `makensis.exe` (`winget install NSIS.NSIS` or
`choco install nsis`, checking the two conventional install paths if
it's not already on `PATH`), and reports the output file's path, size,
and SHA256 on success. **Invoke `makensis` from PowerShell, not Git
Bash** — Git Bash mangles a leading `/D` on an argument (POSIX path
translation kicks in), producing NSIS's usage/help text instead of a
compile; `create-installer.ps1` itself is PowerShell for exactly this
reason.

## Release integrity: SBOM and reproducible builds

- **Software Bill of Materials**: `.goreleaser.yaml`'s `sboms:` block
  generates one CycloneDX/SPDX SBOM per archive (`{{ .ArtifactName
  }}.sbom.json`) via [syft](https://github.com/anchore/syft), listing
  every Go module compiled into that binary with license and checksum
  data. This is GoReleaser's own documented mechanism, not a custom
  script. `syft` must be on `PATH` when building — CI installs it via
  `anchore/sbom-action/download-syft@v0`
  (see [ReleaseProcess.md](ReleaseProcess.md)); for a local
  `goreleaser release --snapshot` run, download a `syft` release
  binary and add it to `PATH` **natively** (e.g. via PowerShell's
  `$env:PATH`) rather than through Git Bash's `export` — GoReleaser is
  a native Windows binary, and Git Bash's POSIX-style PATH entries
  don't reliably translate into a native child process's own `PATH`
  environment variable, which surfaces as `syft: executable file not
  found in %PATH%` even though `syft` is genuinely reachable from the
  Bash shell itself.
- **Reproducible builds**: `ldflags`'s `BuildDate` uses GoReleaser's
  `{{ .CommitDate }}` template variable, not `{{ .Date }}` — `.Date` is
  the actual wall-clock build time, which would make two builds of the
  exact same commit/tag produce byte-different binaries purely because
  they ran at different times. `.CommitDate` is derived from the commit
  itself, so it's identical no matter when or where the build runs.
  `mod_timestamp: "{{ .CommitTimestamp }}"` applies the same principle
  to file timestamps inside the built archives.

## macOS: `.pkg`

[scripts/package-macos-pkg.sh](scripts/package-macos-pkg.sh) wraps the
darwin/amd64 and darwin/arm64 binaries GoReleaser already built into a
single **universal binary** via `lipo -create`, installed to
`/usr/local/bin` (on the default `PATH` on both Intel and Apple Silicon
Macs, unlike Homebrew's arch-specific `/usr/local` vs `/opt/homebrew`
split — one universal binary at one path works unmodified on either
architecture). A `postinstall` script seeds
`~/Library/Application Support/GitFlowPlus/config.json` and creates
`~/Library/Logs/GitFlowPlus` (mirroring the Windows installers'
`%APPDATA%`/`%LOCALAPPDATA%` treatment), resolving the real invoking
user via `dscl` since postinstall scripts always run as root. `pkgbuild`
produces the component package; `productbuild` wraps it with a welcome
screen and the project's `LICENSE` for the standard graphical installer
experience.

This script — and macOS packaging generally — can only run on macOS
(`lipo`/`pkgbuild`/`productbuild` don't exist elsewhere); it was written
and syntax-checked (`bash -n`) but its first real execution happens in
the `macos-pkg` CI job, since no macOS machine was available during
development.

## Versioning

Every artifact's version comes from exactly one place: the pushed Git
tag (stripped of its leading `v`), the same value that already flows
into `-ldflags` for `cli.Version`. No artifact re-derives or
independently guesses a version.

One real technical wrinkle: Windows' native `VERSIONINFO` resource
(NSIS's `VIProductVersion`, and the `.syso` go-winres embeds into
`git-flow-plus.exe` itself) is strictly 4 numeric fields, but Git Flow
Plus's own version has 5 (`Sprint.Feature.ReleaseFix.DevOps.QA`, e.g.
`5.3.4.1.2`). `create-installer.ps1`'s `Resolve-VersionParts` drops the
trailing QA-build digit for the embedded version resource only
(`5.3.4.1`) — the full 5-part string still appears as the human-facing
display version (the installer's `DisplayVersion` in Add/Remove
Programs, and the output filename `GitFlowPlusSetup_v5.3.4.1.2_x64.exe`
itself). QA-iteration count isn't meaningful metadata for the *shipped
file's own* version; Sprint+Feature+Fix+DevOps is.

## Unsigned artifacts

None of the installers or archives are code-signed or notarized. Expect:

- **Windows**: SmartScreen may warn "Windows protected your PC" on
  first run (unsigned publisher). Users can click "More info" → "Run
  anyway".
- **macOS**: Gatekeeper quarantines downloaded, unsigned binaries — see
  [Installation.md](Installation.md#macos) for the `xattr` workaround.

Code signing needs a real certificate (Windows: an EV/OV code-signing
cert; macOS: an Apple Developer ID + notarization) — a purchase and
identity-verification process outside what can be automated here. It's
on [Roadmap.md](Roadmap.md) as a candidate next step.

## See also

- [Building.md](Building.md) — commands and toolchain setup
- [ReleaseProcess.md](ReleaseProcess.md) — the automated pipeline this all runs inside
- [WindowsInstallation.md](WindowsInstallation.md) — user-facing Windows install docs
