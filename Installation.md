# Installation

Git Flow Plus ships as a single, dependency-free binary — no runtime, no
interpreter, nothing else to install. Pick your platform below; each links
to the deeper guide for that install method.

| Platform | Recommended | Also available |
|---|---|---|
| Windows | [Installer (.exe)](WindowsInstallation.md) | Optional MSI, or a raw `.zip` |
| Linux | `.deb` / `.rpm` package | Raw `.tar.gz` |
| macOS | `.pkg` installer | Raw `.tar.gz` |

Every install method comes from the same [GitHub Releases](https://github.com/tabnaeem/git-flow-plus/releases)
page, built automatically by [ReleaseProcess.md](ReleaseProcess.md)'s
pipeline the moment a version tag is pushed — see
[Packaging.md](Packaging.md) for how each artifact is actually produced.

## Windows

The full writeup, including silent install/uninstall, per-user vs.
machine-wide, and PATH details, is in
[WindowsInstallation.md](WindowsInstallation.md). Quick version:

1. Download `git-flow-plus-<version>-windows-<x64|arm64>-setup.exe` from
   the latest release.
2. Run it. The installer detects Git, adds Git Flow Plus to `PATH`, and
   offers to run `git flow doctor` at the end to confirm everything
   works.

An optional `.msi` is also published, for teams deploying via Group
Policy/SCCM/Intune — see
[WindowsInstallation.md#msi-enterprise-deployment](WindowsInstallation.md#msi-enterprise-deployment).

## Linux

```bash
# Debian/Ubuntu
sudo dpkg -i git-flow-plus-<version>-amd64.deb

# Fedora/RHEL
sudo rpm -i git-flow-plus-<version>-x86_64.rpm
```

Both packages install to `/usr/bin/git-flow-plus` and create a
`/usr/bin/git-flow` symlink automatically — see
[Packaging.md#linux-debrpm](Packaging.md#linux-debrpm) for why that
symlink is required. Or grab the raw archive:

```bash
tar -xzf git-flow-plus-linux-amd64.tar.gz -C /usr/local/bin --strip-components=0 git-flow-plus
chmod +x /usr/local/bin/git-flow-plus
ln -s /usr/local/bin/git-flow-plus /usr/local/bin/git-flow
```

## macOS

```bash
# after downloading git-flow-plus-<version>-macos-universal.pkg
sudo installer -pkg git-flow-plus-<version>-macos-universal.pkg -target /
```

Or double-click the `.pkg` for the normal graphical installer. It's a
universal binary (Intel + Apple Silicon in one file), installed to
`/usr/local/bin`. Or grab the raw archive, same steps as Linux above,
substituting `git-flow-plus-darwin-<amd64|arm64>.tar.gz`.

macOS Gatekeeper may quarantine an unsigned downloaded binary — clear it
with:

```bash
xattr -d com.apple.quarantine /usr/local/bin/git-flow-plus
```

(Neither the `.pkg` nor any archive is code-signed/notarized yet — see
[Packaging.md#unsigned-artifacts](Packaging.md#unsigned-artifacts).)

## Build from source

Requires Go (see `go.mod` for the minimum version) and `git`:

```bash
git clone https://github.com/tabnaeem/git-flow-plus.git
cd git-flow-plus
go build -o bin/git-flow ./cmd/git-flow-plus
```

See [Building.md](Building.md) for version-stamped builds and the full
cross-compilation pipeline.

## Verifying a download

Every release includes `checksums.txt` (from GoReleaser, covering the
core archives) and platform-specific `windows-checksums.txt` /
`macos-checksums.txt` (covering the installer/`.pkg`, added by the jobs
described in [Packaging.md](Packaging.md)):

```bash
sha256sum -c checksums.txt --ignore-missing
```

## Verify the install

Regardless of platform or install method:

```bash
git flow version
git flow doctor
```

`version` confirms the binary and build metadata; `doctor` (run inside a
repository) walks through Git, PATH, configuration, and — once
initialized — branch health, printing a colorized pass/fail for each
check. See [CommandReference.md](CommandReference.md#git-flow-doctor).

## Upgrading or uninstalling

See [UpgradeGuide.md](UpgradeGuide.md).

## Something not working?

See [Troubleshooting.md](Troubleshooting.md) — PATH issues, antivirus/
SmartScreen warnings, and IDE/terminal-specific notes (VS Code,
JetBrains, Git Bash, PowerShell, SourceTree, GitKraken).
