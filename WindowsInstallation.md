# Windows Installation

Windows is Git Flow Plus's primary-priority platform for installation
tooling. This document covers the `GitFlowPlusSetup_v<version>_x64.exe`
installer in full — a modern-wizard NSIS installer, the same category of
tool GitHub CLI, Terraform, and Docker CLI ship on Windows. Source: see
[build/windows/installer.nsi](build/windows/installer.nsi) and
[build/windows/create-installer.ps1](build/windows/create-installer.ps1);
build details in [Packaging.md](Packaging.md).

## Installing

Download `GitFlowPlusSetup_v<version>_x64.exe` from
[the latest release](https://github.com/tabnaeem/git-flow-plus/releases)
and run it. Windows will prompt for administrator elevation (UAC) — the
installer always installs machine-wide, under `Program Files`. The
wizard has six pages:

1. **Welcome**
2. **License** — the project's MIT license
3. **Choose Installation Folder** — defaults to
   `C:\Program Files\Git Flow Plus`
4. **Select Components**:
   - ☑ **Add to PATH** (checked by default)
   - ☑ **Start Menu Shortcut** (checked by default)
   - ☐ **Desktop Shortcut** (unchecked by default)
5. **Install**
6. **Completed** — offers to run `git flow doctor` before closing, so
   you can confirm the install actually worked without opening a
   terminal yourself.

If a previous version is already installed, the wizard detects it (via
its own Add/Remove Programs registry entry) and, after a confirmation
prompt, silently uninstalls it first — so an upgrade never leaves stale
files behind. See [Upgrading](#upgrading) below.

## Silent install

```powershell
GitFlowPlusSetup_v1.4.0_x64.exe /S
```

That's the entire invocation — standard NSIS silent-install syntax. No
UI, no prompts, including the upgrade-confirmation dialog (an existing
install is removed automatically rather than asking). Component
defaults still apply in silent mode exactly as they would if you
clicked through the wizard: PATH and the Start Menu shortcut are
installed, the Desktop shortcut is not.

- `/D=C:\Custom\Path` — override the install location. Must be the
  **last** argument, unquoted even if the path contains spaces (a stock
  NSIS requirement, not a Git Flow Plus one).
- Because the installer requests admin elevation
  (`RequestExecutionLevel admin`), the *invoking* process needs to
  already be elevated for a truly unattended silent install (e.g. a
  provisioning script running as SYSTEM/Administrator) — a UAC prompt
  can't be answered non-interactively any more than it can for any
  other Windows installer.

## Silent uninstall

```powershell
"C:\Program Files\Git Flow Plus\Uninstall.exe" /S
```

This removes the installed binaries, shortcuts, the Add/Remove Programs
entry, and — precisely — the `PATH` entry it added, spliced out without
touching any of your other `PATH` directories (see
[PATH and `git flow`](#path-and-git-flow) below for how that splice is
implemented safely).

## PATH and `git flow`

Git resolves `git <word>` to an executable literally named `git-<word>`
on `PATH`. That means `git flow ...` only works if a binary named
exactly `git-flow.exe` — not `git-flow-plus.exe` — is discoverable. The
installer places **both** files in the install directory (a byte-
identical copy), so:

- `git-flow-plus.exe version` works directly.
- `git flow version` works as a Git subcommand.

`git flow doctor`'s `PATH` check verifies this exact condition — if it
ever reports `FAIL`, see [Troubleshooting.md](Troubleshooting.md).

The installer's PATH management is hand-written NSIS (no third-party
plugin — see [Packaging.md#path-management](Packaging.md#path-management)
for why) operating on the machine `PATH`
(`HKLM\SYSTEM\CurrentControlSet\Control\Session Manager\Environment`):
adding is a no-op if the directory is already present (no duplicate
entries on repeated installs/repairs), and removal on uninstall splices
out exactly that one entry, correctly re-joining its two neighbors
whichever position it was in — first, last, middle, or the only entry.
A `WM_WININICHANGE` broadcast tells already-open programs (Explorer,
other terminals) to pick up the change without a reboot; a **freshly
opened** terminal always sees it immediately either way.

## Default locations

| What | Location |
|---|---|
| Program files | `C:\Program Files\Git Flow Plus` |
| Start Menu shortcuts | `Start Menu\Programs\Git Flow Plus` |
| Desktop shortcut (optional) | `%USERPROFILE%\Desktop\Git Flow Plus.lnk` |
| Uninstaller | `C:\Program Files\Git Flow Plus\Uninstall.exe` |
| Add/Remove Programs entry | `HKLM\...\Uninstall\GitFlowPlus` |

There is no seeded configuration file — Git Flow Plus only reads
configuration from a repository's own `.gitflowplus/config.json` (see
[ReleaseManagement.md](ReleaseManagement.md)), so there's nothing global
for the installer to create ahead of time.

## Upgrading

Just run the new version's installer. `.onInit` looks up the previous
install's own uninstall registry key and, once you confirm (or silently,
under `/S`), runs that uninstaller first — before laying down the new
files. Your `PATH` entry, Start Menu shortcut, and any Desktop shortcut
are recreated by the new install, so nothing is lost across the swap.

## Administrator rights

The installer always requires elevation (`RequestExecutionLevel admin`)
— there's no per-user install mode. Windows will show the standard UAC
consent prompt when you launch it, unless the invoking process is
already elevated. There's no way to silently elevate without an
already-privileged caller — this is a Windows security boundary, not a
Git Flow Plus limitation.

## Windows integration

Git Flow Plus is a normal console binary invoked via `git flow ...` (or
directly as `git-flow-plus ...`) — anything that can run a shell command
already supports it:

| Tool | Notes |
|---|---|
| **PowerShell** | Works natively; colorized output requires a real terminal (Windows Terminal or PowerShell's own console host both qualify). |
| **Git Bash** | Works natively — this is the environment Git Flow Plus's own development and testing happens in. |
| **Windows Terminal** | Full color support in any hosted shell (PowerShell, Git Bash, CMD). |
| **CMD** | Works; ANSI color requires Windows 10+ (`ENABLE_VIRTUAL_TERMINAL_PROCESSING`, which Git Flow Plus enables itself — see `internal/cli/color_windows.go`). |
| **VS Code** | Use the integrated terminal (any of the shells above) — no extension needed. |
| **JetBrains IDEs** (GoLand, IntelliJ, etc.) | Same — the built-in terminal panel runs any of the shells above. |
| **SourceTree** | Use its "Open in Terminal" / "Open in Git Bash" action, or add `git flow` commands as [custom actions](https://confluence.atlassian.com/sourcetreekb) if you want them in the GUI menu. |
| **GitKraken** | Use its integrated terminal, or run Git Flow Plus alongside it in any external terminal pointed at the same repository. |

## See also

- [Installation.md](Installation.md) — cross-platform overview
- [Packaging.md](Packaging.md) — how the installer itself is built
- [UpgradeGuide.md](UpgradeGuide.md) — upgrading an existing install
- [Troubleshooting.md](Troubleshooting.md) — PATH issues, SmartScreen, and more
