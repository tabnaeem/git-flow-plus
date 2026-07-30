# Packaging

How every distributable artifact — binaries, archives, checksums,
`.deb`/`.rpm`, the Windows `.exe`/`.msi`, the macOS `.pkg` — is actually
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

Evaluated three options for the primary `.exe` installer:

| | Inno Setup | WiX | NSIS |
|---|---|---|---|
| Output | `.exe` | real `.msi` | `.exe` |
| Silent install/uninstall | native | native | native |
| Per-user vs machine-wide | native dual-mode picker (v6.1+) | `Scope` attribute, one mode per build | via plugin |
| PATH management | hand-written (well-documented recipe) | native `<Environment>` element | community plugin |
| Upgrade/prior-version detection | native (`AppId` + uninstall-string lookup) | native (`MajorUpgrade`) | manual |
| Pre-install checks (Git, admin) | easy (Pascal Script) | needs custom actions | possible, clunkier |
| CI availability | **preinstalled on GitHub's `windows-latest`** | `dotnet tool install` (fast) | manual install |
| Enterprise deployment (GPO/SCCM/Intune) | not ideal (not a real MSI) | **this is what MSI is for** | not ideal |

**Decision:** Inno Setup as the primary `.exe`
([installer/windows/gitflowplus.iss](installer/windows/gitflowplus.iss)),
WiX as the optional `.msi`
([installer/windows/gitflowplus.wxs](installer/windows/gitflowplus.wxs))
for GPO/SCCM/Intune deployment. NSIS didn't offer anything Inno doesn't
already provide for this use case, and Inno being preinstalled on
GitHub's Windows runner is a concrete, ongoing maintenance win.

### The `.exe` installer

Full user-facing behavior is documented in
[WindowsInstallation.md](WindowsInstallation.md). Implementation notes:

- **PATH management** is hand-written Pascal (`EnvAddPath`/
  `EnvRemovePath` in the `.iss`'s `[Code]` section) rather than Inno's
  `[Registry]` section with `uninsdeletevalue` — that flag deletes the
  **entire** `Path` registry value on uninstall, which would wipe out
  every other program's PATH entries too. The hand-written version
  splices out exactly the installed directory, verified via a real
  install → `doctor` (all green, including the `PATH` check) → uninstall
  → PATH restored **byte-for-byte** to its pre-install value, tested
  against a real ~1,800-character PATH with 40+ existing entries.
- **Upgrade detection** looks up the previous install's own uninstall
  registry key (via its fixed `AppId` GUID) and silently runs that
  uninstaller first — the standard Inno "self-upgrade" recipe.
- **The seeded `default-config.json`** is a real file
  ([installer/windows/default-config.json](installer/windows/default-config.json)),
  bundled via `[Files] ... Flags: dontcopy` and copied at install time —
  not a string constructed in Pascal — specifically so the WiX `.msi`
  can reference the exact same file rather than duplicating its content.
- **GUID escaping gotcha:** `[Setup]` section values (like `AppId`) pass
  through Inno's own `{constant}` parser *after* the preprocessor
  substitutes `{#MyAppId}` — a single-braced GUID there gets misread as
  an unknown constant reference. The fix is a second preprocessor
  constant with doubled braces used only in `[Setup]`; Pascal `[Code]`
  string literals use the plain single-braced form. See the comments at
  the top of the `.iss` for the exact mechanism.

### `.msi` (enterprise deployment)

See [WindowsInstallation.md#msi-enterprise-deployment](WindowsInstallation.md#msi-enterprise-deployment)
for user-facing behavior. It's deliberately narrower in scope than the
`.exe`:

- **Per-machine only** — MSI's `Scope` is fixed per build; Group
  Policy/SCCM/Intune push to machines, not interactively-logged-in
  users, so there's no dual-mode picker to build.
- **No Git/Git Bash/PowerShell detection** — meaningless for an
  unattended push to already-vetted machines, and WiX doesn't have
  Inno's easy scripting for this without a custom-action DLL.
- **Config seeded under `C:\ProgramData`**, not a user's `%APPDATA%` — a
  per-machine install has no single "current user" to scope a roaming
  profile to.
- **PATH management uses WiX's native `<Environment>` element** — since
  WiX v4, `Environment` moved into the *core* `wxs` schema (no `util:`
  extension prefix needed, unlike WiX v3). Its MSI-table-level encoding
  was verified directly: `Action="set" Permanent="no" Part="last"
  System="yes"` compiles to the Environment table row
  `=-*PATH | [~];[INSTALLFOLDER]` — `=` (set), `-` (remove on
  uninstall), `*` (per-machine), and `[~]` (existing value) `;`
  (append) `[INSTALLFOLDER]`, exactly matching Windows Installer's
  documented format.
- **No `NeverOverwrite`** — modern WiX dropped that File attribute.
  The seeded config's component is marked `Permanent="yes"` so it
  survives uninstall, matching the `.exe`'s behavior, but a version
  *upgrade* follows ordinary MSI file-versioning rules rather than an
  unconditional "leave it alone" — a documented, minor difference from
  the `.exe` installer.

#### WiX v7 licensing

WiX Toolset v7 gates its own CLI behind accepting a paid **Open Source
Maintenance Fee (OSMF)** EULA (`wix extension add` refuses to run
without it — see https://wixtoolset.org/osmf/). That's a licensing
decision for whoever maintains this project to make, not something to
accept unattended in a build script. **Pin to WiX v6** instead
(`dotnet tool install --global wix --version 6.0.1`), the last major
version under the traditional open license — everything in
`gitflowplus.wxs` was authored and verified against v6.

### Building locally

```bash
# stage a flat binary directory — see Building.md for the full build command
mkdir -p dist/windows-x64 && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o dist/windows-x64/git-flow-plus.exe ./cmd/git-flow-plus

# Inno Setup .exe
iscc /DMyAppVersion=5.3.4.1.2 /DMyAppArch=x64 installer/windows/gitflowplus.iss

# WiX .msi (run from installer/windows/)
wix build gitflowplus.wxs -arch x64 -d MsiVersion=5.3.4.1 -d BinDir=../../dist/windows-x64 -o ../../dist/installer/git-flow-plus-5.3.4.1.2-windows-x64.msi
```

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

One real technical wrinkle: Windows' native `VERSIONINFO` resource (Inno
`VersionInfoVersion`) and MSI's `ProductVersion` are conventionally 3–4
numeric fields, but Git Flow Plus's own version has 5
(`Sprint.Feature.ReleaseFix.DevOps.QA`, e.g. `5.3.4.1.2`). Both Windows
installers drop the trailing QA-build digit for their embedded version
resource only (`5.3.4.1`) — the full 5-part string still appears as the
human-facing display version (Inno's `AppVersion`, the `.msi`'s
`ProductName`/release notes). QA-iteration count isn't meaningful
metadata for the *shipped file's own* version; Sprint+Feature+Fix+DevOps
is. This mapping is intentionally the same on both installers (and in
`release.yml`'s version-computation step) so they never drift apart.

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
