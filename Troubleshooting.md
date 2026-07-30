# Troubleshooting

Start here for anything install/PATH/environment-related. For everyday
usage questions, see [DeveloperGuide.md](DeveloperGuide.md) or
[CommandReference.md](CommandReference.md) instead.

## First step, always: `git flow doctor`

```bash
git flow doctor
```

Colorized pass/fail for: Git present and its version, whether you're in
a Git repository, `main`/`staging`/`develop` branch presence, working
tree state, repository write permissions, `.gitflowplus/config.json`
presence, Git Flow Plus's own build version, whether `PATH` is set up
correctly, and whether a release is currently in progress. Every failure
line includes a specific, actionable detail message — read that first;
the sections below cover the ones worth expanding on.

## "`git flow` is not a git command"

This means `git-flow` (not `git-flow-plus`) isn't resolving on `PATH` —
Git only forwards `git <word>` to a subcommand if a binary literally
named `git-<word>` exists somewhere on `PATH`. Confirm with:

```bash
git flow doctor
```

and look at the `PATH` line specifically. If it reports `FAIL`:

- **Installed via the Windows/macOS installer or a Linux package**: this
  shouldn't happen — every installer places (or symlinks) a `git-flow`
  binary automatically. Re-run the installer, or file an issue with your
  `doctor` output.
- **Installed from a raw archive**: you likely only extracted
  `git-flow-plus` and never created the `git-flow` name — see
  [Installation.md](Installation.md) for the exact `ln -s`/copy step.
- **Just installed, in an already-open terminal**: some shells cache
  `PATH` lookups per-session. Open a new terminal window (a full restart
  isn't usually necessary — see
  [WindowsInstallation.md#path-and-git-flow](WindowsInstallation.md#path-and-git-flow)
  for what the Windows installer does to minimize this).

You can always invoke it directly regardless of PATH setup:
`git-flow-plus <command>` works identically to `git flow <command>`.

## Windows: "Windows protected your PC" (SmartScreen)

Expected — none of the installers or binaries are code-signed yet (see
[Packaging.md#unsigned-artifacts](Packaging.md#unsigned-artifacts)).
Click **More info → Run anyway**. This is a one-time warning per
downloaded file, not a sign anything is actually wrong.

## Windows: antivirus flags the installer or binary

Also a symptom of being unsigned, not of anything malicious. If your
organization's antivirus/EDR blocks it outright rather than just
warning, you'll need an allowlist exception from IT — there's no way
around this without a code-signing certificate (see
[Packaging.md#unsigned-artifacts](Packaging.md#unsigned-artifacts) and
[Roadmap.md](Roadmap.md)).

## macOS: "cannot be opened because the developer cannot be verified"

Gatekeeper quarantining an unsigned download. Clear it:

```bash
xattr -d com.apple.quarantine /usr/local/bin/git-flow-plus
```

Or, for the `.pkg` itself if it won't open: right-click → Open, then
confirm in the dialog (bypasses Gatekeeper for that one file, once).

## "permission denied" running `git flow doctor`'s permissions check

The `permissions` check fails if the repository directory itself isn't
writable — check you actually own the directory and aren't, for
example, working inside a read-only mounted path or a directory owned
by another user (common in shared/CI environments run as the wrong
UID). This is unrelated to the *installation* being read-only; it's
about the Git repository you're currently standing in.

## Colors look wrong / no colors at all

Color is disabled automatically whenever output isn't a real terminal
(e.g. piped to a file, or `NO_COLOR` is set in your environment) — see
`internal/cli/color.go`. If you expect color and aren't getting it in
an interactive terminal, check:

- `NO_COLOR` isn't set (`echo $NO_COLOR` / `echo $env:NO_COLOR`)
- You didn't pass `--no-color`
- Your `.gitflowplus/config.json`'s `logging.color` isn't `false`
- On Windows CMD specifically: needs Windows 10+ for ANSI support —
  Git Flow Plus enables `ENABLE_VIRTUAL_TERMINAL_PROCESSING` itself
  (`internal/cli/color_windows.go`), but very old `cmd.exe`/Windows
  builds don't support it regardless.

## Want more diagnostic detail?

```bash
git flow <command> --verbose   # debug-level logging
git flow <command> --debug     # trace-level, the most detailed
git flow <command> --json-log  # newline-delimited JSON, for feeding into log aggregation
```

There's no file-based logging yet — `%LOCALAPPDATA%\GitFlowPlus\logs`
(Windows) / `~/Library/Logs/GitFlowPlus` (macOS) are created by the
installers but reserved for a future capability; every diagnostic today
goes to the console. Redirect it yourself if you need a persistent log:
`git flow release build --debug 2>&1 | tee build.log`.

## IDE/tool-specific issues

See [WindowsInstallation.md#windows-integration](WindowsInstallation.md#windows-integration)
for VS Code, JetBrains, Git Bash, PowerShell, Windows Terminal,
SourceTree, and GitKraken specifics — Git Flow Plus is a normal console
binary, so anything that can run a shell command already supports it;
issues there are almost always the `PATH` problem above, surfacing
inside that tool's terminal instead of a standalone one.

## Still stuck?

Open an issue at
[github.com/tabnaeem/git-flow-plus/issues](https://github.com/tabnaeem/git-flow-plus/issues)
with your `git flow doctor` output, your platform/install method, and
`git flow version`'s output attached.
