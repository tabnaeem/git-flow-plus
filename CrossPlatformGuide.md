# Cross-Platform Guide

Git Flow Plus targets Windows, Linux, and macOS equally — this isn't a
Windows tool with Unix support bolted on, or vice versa. This document
covers what "supported" means and the handful of places platform
differences actually mattered during development.

## Supported platforms

| OS | Architectures |
|---|---|
| Windows | x64 (amd64), ARM64 |
| Linux | x64 (amd64), ARM64 |
| macOS | Intel (amd64), Apple Silicon (arm64) |

All six are built and tested for on every CI run (see
[Building.md](Building.md)) and cross-compile cleanly from any host —
Go's toolchain needs no per-target C compiler because Git Flow Plus builds
with `CGO_ENABLED=0`.

## What actually differs across platforms

Everything else in this section exists because one of these three things is
true somewhere in the codebase.

### 1. Git command-line arguments always use forward slashes

`git` itself expects `/` in paths passed as command arguments, even on
Windows, regardless of the OS's native separator. Filesystem access
(`os.ReadFile`, `os.Stat`, ...) needs the OS-native separator instead.
`internal/release/loader.go` and `internal/feature/loader.go` keep these
distinct on purpose:

- `path.Join` (always `/`) for anything passed to a `git.Client` method —
  see `releaseRelPath`/`versionRelPath`/`archiveRelPath`/`feature.RelPath`.
- `filepath.Join` (OS-native) for anything touching the filesystem
  directly — see `Path`/`ArchivePath`.

If you add a new file under `.gitflowplus/` that both gets committed via
`git.Client.Add` *and* read/written directly, it needs both a `path.Join`
relative form and a `filepath.Join` absolute form, exactly like the
existing ones. Using `filepath.Join`'s output as a git argument is a real
bug on Windows (it would pass backslashes to git); the reverse (using
`path.Join`'s output for `os.ReadFile`) happens to work on Unix and
silently break on Windows, so a test that only runs on Linux CI wouldn't
catch it — this is exactly why `internal/release`'s and `internal/feature`'s
loader tests round-trip through the actual OS filesystem, not just fakes.

### 2. Console color needs an opt-in step on Windows

Unix terminals interpret ANSI escape codes natively. Windows consoles
don't, by default, outside Windows Terminal — the classic `conhost.exe`
needs `ENABLE_VIRTUAL_TERMINAL_PROCESSING` explicitly set via
`SetConsoleMode`. `internal/cli/color_windows.go` (behind a `//go:build
windows` tag) does this; `internal/cli/color_other.go` (behind `//go:build
!windows`) is a no-op, since every other supported platform's terminal
already interprets ANSI natively with no such flag to opt into. Both are
called through the same `enableVirtualTerminal(*os.File)` signature from
`internal/cli/color.go`, so the platform-specific logic never leaks into
callers — see [Architecture.md](Architecture.md) for the general dependency
pattern this follows.

Terminal *detection* itself (is this even a real terminal, as opposed to a
redirected file or a pipe) is genuinely cross-platform via
`golang.org/x/term.IsTerminal`, which wraps the right syscall per OS
(`ioctl` on Unix, `GetConsoleMode` on Windows) — no platform-specific code
needed there.

### 3. Line endings

Git Flow Plus never writes `\r\n` explicitly anywhere — all generated
files (`config.json`, `release.json`, `features.json`, `version.json`, tag
messages) use `\n`. Git's own `core.autocrlf` handles any checkout-time
conversion, which is standard practice and not something this tool second-
guesses or overrides.

### 4. Lifecycle hook scripts

`internal/hooks` (see
[ReleaseManagement.md](ReleaseManagement.md#lifecycle-hooks-cicd-integration))
dispatches a hook script by trying `.sh` (run via `sh`), `.ps1` (run via
`powershell -File`), `.bat`/`.cmd` (run via `cmd /C`), or no extension at
all (run directly, relying on a shebang line and the executable bit — Unix
only) in that order. This is the one piece of genuinely platform-specific
*user-facing* behavior: a hook script authored for one platform won't run
on another unless you provide the matching variant, by design — Git Flow
Plus doesn't attempt to transpile or emulate shell scripts across
platforms.

## Testing across platforms

The full test suite (`go test ./...`) runs real `git` commands against
real temporary repositories rather than mocking them (see
[DeveloperGuide.md](DeveloperGuide.md#testing-style)), so it exercises the
path-handling and process-invocation code on whichever OS it runs on. CI
runs this matrix on Linux, macOS, and Windows on every push (see
[Building.md](Building.md#cicd)) — a change that only works on one
platform fails there before it fails for a user.

The race detector (`go test ./... -race`) requires cgo, which isn't always
available (e.g. this project's own primary development environment has no
C compiler on `PATH`) — CI runs it on a dedicated Linux job where cgo is
reliably present, rather than requiring every contributor's machine to have
one.
