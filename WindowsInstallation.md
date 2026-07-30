# Windows Installation

Windows is Git Flow Plus's primary-priority platform for installation
tooling — this document covers the `.exe` installer (Inno Setup) in
full, plus the optional `.msi` for enterprise deployment. Source: see
[installer/windows/gitflowplus.iss](installer/windows/gitflowplus.iss)
and [installer/windows/gitflowplus.wxs](installer/windows/gitflowplus.wxs);
build details in [Packaging.md](Packaging.md).

## Installing

Download `git-flow-plus-<version>-windows-<x64|arm64>-setup.exe` from
[the latest release](https://github.com/tabnaeem/git-flow-plus/releases)
and run it. It will:

1. Detect a prior Git Flow Plus install (by its own installer ID) and
   silently remove it first, so an upgrade never leaves stale files
   behind.
2. Detect whether `git` is on `PATH`, and warn (not block) if it isn't —
   Git Flow Plus needs Git to function, but you can install Git
   afterwards and re-run `git flow doctor` to confirm.
3. Detect Git Bash and PowerShell (informational only).
4. Let you choose **per-user** (default, no admin prompt) or
   **machine-wide** (all users, requires elevation) — Inno Setup's
   install-mode page, or pass `/CURRENTUSER`/`/ALLUSERS` on the command
   line to skip the prompt (see [Silent install](#silent-install)
   below).
5. Install to `%LOCALAPPDATA%\Programs\GitFlowPlus` (per-user) or
   `C:\Program Files\GitFlowPlus` (machine-wide).
6. Add the install directory to `PATH` (the user's `PATH` for a per-user
   install, the system `PATH` for machine-wide) — see
   [PATH and `git flow`](#path-and-git-flow) below for exactly why this
   matters.
7. Seed a default configuration file — see [Seeded files](#seeded-files).
8. Offer to run `git flow doctor` and `git flow version` at the end, so
   you can see the install actually worked before closing the wizard.

## Silent install

```powershell
git-flow-plus-5.3.4.1.2-windows-x64-setup.exe /VERYSILENT /SUPPRESSMSGBOXES /NORESTART /CURRENTUSER
```

- `/VERYSILENT /SUPPRESSMSGBOXES /NORESTART` — no UI at all, including
  the Git-detection warning dialog.
- `/CURRENTUSER` — per-user install, no admin elevation prompt. Use
  `/ALLUSERS` instead for a machine-wide silent install (this does
  require the invoking process to already be elevated — a UAC prompt
  can't be answered non-interactively).
- `/DIR="C:\Custom\Path"` — override the install location.
- `/LOG="install.log"` — write a log file, useful for CI/scripted
  deployment debugging.

This exact invocation (per-user, `/CURRENTUSER`) was run and verified
end-to-end during development: install → `PATH` updated → `git flow
doctor` all green → silent uninstall → `PATH` restored to its exact
original value, config preserved. See [Silent
uninstall](#silent-uninstall) below for the reverse.

## Silent uninstall

```powershell
"%LOCALAPPDATA%\Programs\GitFlowPlus\unins000.exe" /VERYSILENT /SUPPRESSMSGBOXES /NORESTART
```

(Substitute the machine-wide path, `C:\Program Files\GitFlowPlus\unins000.exe`,
for a machine-wide install.) This removes the installed binaries and the
`PATH` entry — precisely that entry, spliced out without touching any of
your other `PATH` directories. It does **not** delete your seeded
configuration or logs directory (see [Seeded files](#seeded-files)) —
consistent with every other installer on Windows not deleting your data
just because you uninstalled the program that created it.

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

## Default locations

| What | Per-user | Machine-wide |
|---|---|---|
| Program files | `%LOCALAPPDATA%\Programs\GitFlowPlus` | `C:\Program Files\GitFlowPlus` |
| Configuration | `%APPDATA%\GitFlowPlus\config.json` | `C:\ProgramData\GitFlowPlus\config.json` |
| Logs (reserved) | `%LOCALAPPDATA%\GitFlowPlus\logs` | `C:\ProgramData\GitFlowPlus\logs` |

## Seeded files

The installer creates the configuration directory above and seeds a
`config.json` matching `internal/config.Default()`'s exact shape — a
real, valid config, not a placeholder. **Git Flow Plus itself currently
only reads configuration from a repository's own
`.gitflowplus/config.json`** (see
[ReleaseManagement.md](ReleaseManagement.md)) — this seeded copy exists
as a ready-to-copy template and to reserve the conventional location for
a possible future global-config feature. It is never overwritten if it
already exists (an upgrade won't clobber edits you've made to it).

The logs directory is created but not yet written to — reserved for a
future file-based logging capability; see
[Troubleshooting.md](Troubleshooting.md) for how to get diagnostic
output today (`--verbose`/`--debug`/`--json-log`, all printed to the
console).

## Administrator rights

Per-user installs never require elevation. A machine-wide install does,
and Windows will prompt for it (standard UAC consent) unless the
invoking process is already elevated. There's no way to silently elevate
without an already-privileged caller — this is a Windows security
boundary, not a Git Flow Plus limitation.

## MSI (enterprise deployment)

An optional `.msi` is published alongside the `.exe` for teams deploying
via Group Policy, SCCM, or Intune — those tools expect a genuine MSI,
not an Inno Setup executable. Deliberate differences from the `.exe`
installer (see [Packaging.md](Packaging.md#windows-msi) for the full
reasoning):

- **Machine-wide only** — no per-user mode. Enterprise deployment always
  targets a machine, not an interactively-logged-in user.
- **No interactive Git/Git Bash/PowerShell detection** — meaningless for
  an unattended push to already-vetted machines.
- **Config seeded under `C:\ProgramData`**, not a specific user's
  `%APPDATA%` — a per-machine install has no single "current user" to
  scope a roaming profile to.

```powershell
msiexec /i git-flow-plus-5.3.4.1.2-windows-x64.msi /quiet /norestart
msiexec /x git-flow-plus-5.3.4.1.2-windows-x64.msi /quiet /norestart
```

Upgrades are handled by MSI's native `MajorUpgrade` mechanism — installing
a newer `.msi` automatically removes the older one first.

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
