# Developer Guide

## Prerequisites

- Go (see `go.mod` for the minimum version; developed against 1.26).
- `git` on `PATH` — the test suite runs real git commands against
  temporary repositories, it doesn't mock them.
- Optional: [golangci-lint](https://golangci-lint.run/) (config in
  `.golangci.yml`) for linting, and GNU Make for the `Makefile` targets
  (Windows without `make` on `PATH` can use `scripts\build.ps1` /
  `scripts\package.ps1` directly instead — see
  [BuildGuide.md](BuildGuide.md)).

## Building

```bash
go build -o bin/git-flow ./cmd/git-flow-plus   # plain, no version metadata
make build                                     # same, with version metadata via -ldflags
make dist                                      # cross-compile all 6 supported platforms into dist/
```

See [BuildGuide.md](BuildGuide.md) for cross-compilation, packaging, and
the CI/CD release pipeline in full — this section is deliberately just the
"how do I get a binary to run locally" quick reference.

## Testing

```bash
go test ./... -cover
```

Every package is at 78%+ coverage (most 83–100%); `cmd/git-flow-plus` is
0% because it's a three-line `main` that just calls `cli.Execute()` —
everything meaningful is in `internal/cli`.

### Testing style

Every package with git-touching logic (`internal/git`, `internal/gitflow`,
`internal/release`, `internal/cli`) uses **two complementary styles**, not
one or the other:

1. **Fake-based unit tests** (`error_paths_test.go`, `fake_test.go` /
   `fake_client_test.go`) — a hand-written fake implementing the relevant
   interface (`git.Client`, `gitflow.Service`, ...) with per-test
   overridable hooks. Fast, and the only practical way to hit specific
   error-wrapping branches (e.g. "the third git call in this sequence
   fails") without engineering a real repo into that exact state.
2. **Real end-to-end tests** (`service_test.go`, `integration_test.go`,
   `cli_test.go`) — an actual temp directory, an actual `git init`, and the
   real code path through to the real `git` binary. These are what catch
   the bugs fakes can't: wrong merge order, a tag on the wrong commit, a
   file that should have been committed but wasn't. Several real
   architecture bugs were caught exactly this way during development (see
   `git log` for examples — a working-tree-cleanliness ordering bug in
   `ReleaseFixFinish`, a tag-name collision between QA and production
   tags).

Committer identity in real-repo tests is set via environment variables
(`GIT_AUTHOR_NAME`, `GIT_CONFIG_COUNT`/`GIT_CONFIG_KEY_0`/
`GIT_CONFIG_VALUE_0` for `commit.gpgsign=false`, etc.) scoped with
`t.Setenv`, never by mutating the host's global `~/.gitconfig`. Copy the
`setGitTestEnv`/`newInitializedRepo` pattern from
`internal/gitflow/service_test.go` or `internal/release/service_test.go`
when adding a new real-repo test.

When you touch a fake (`fakeClient`, `fakeGit`, `fakeGitFlow`, ...), the Go
compiler will tell you immediately if it no longer satisfies the real
interface — that's the point of building them against the interface type,
not a concrete struct.

### Linting

```bash
golangci-lint run ./...
```

Config in `.golangci.yml`. Should report 0 issues on `main` at all times —
fix findings before merging, don't accumulate `//nolint` comments.

## Code organization

See [Architecture.md](Architecture.md) for the package dependency graph
and why it's shaped the way it is. Short version: `internal/git` (raw git)
→ `internal/gitflow` (branching model) → `internal/release` (release
management, composes gitflow + version + feature + git + hooks) →
`internal/cli` (Cobra commands, thin — parse args, call a service, print
the result). `internal/feature` (the Feature Registry) sits alongside
`internal/version`, deliberately as small: a struct, a slice with
`Find`/`Upsert`/`Approved`, and a `Loader`. All Release Planning *logic*
(who can approve what, deriving Pending) lives in
`internal/release/featureplanning.go`, not in `internal/feature` itself —
see [Architecture.md](Architecture.md#why-this-split).

**Business logic never lives in `internal/cli`.** A command handler's
`RunE` should be: load config, construct a service, call one method,
format the result. If you're writing an `if`/`for` that isn't about
argument validation or output formatting, it belongs in `internal/gitflow`
or `internal/release` instead.

## Adding a new command

1. Decide which layer owns the logic:
   - Pure git branch mechanics (branch/merge/tag against a fixed base) →
     `internal/gitflow`, following the `FeatureStart`/`FeatureFinish`
     pattern in `internal/gitflow/feature.go`.
   - Anything involving the manifest, version, or tagging → `internal/release`.
2. Add the method to the relevant `Service` interface and implement it.
   Write both a fake-based error-path test and a real-repo test (see
   above).
3. Add a `*cobra.Command` in `internal/cli` that parses args, calls the
   service, and formats output via `app.Println`/`app.Printf` (these
   propagate write errors — never call `fmt.Fprintln`/`fmt.Fprintf`
   directly against `app.Out` in a command handler; `golangci-lint`'s
   `errcheck` will flag it, and it's a genuine correctness gap: `app.Out`
   can be a pipe that fails).
4. Wire it into the parent command in the relevant `new*Cmd` function.
5. Add a CLI-level test in `internal/cli/cli_test.go` exercising it through
   `run(t, app, ...)` against a real temp repo.
6. Document it in [CommandReference.md](CommandReference.md).

## Extending the version scheme or manifest schema

Both live entirely in `internal/version` (`Version` struct, `ApplyBuild`)
and `internal/release` (`Manifest` struct in `manifest.go`). If you add a
field to `Manifest`, update:

- `manifest.go`'s `New()` constructor (so a fresh release starts with a
  sane default, typically an empty slice — never `nil` vs `[]string{}`
  inconsistently, since that changes the JSON shape).
- `types.go`'s `StatusReport` if the field should be visible via `release
  status`.
- `internal/cli/release.go`'s `printReleaseStatus` if so.
- `tagmessage.go` if the field belongs in tag messages.

## Extending the Feature Registry / Release Planning

The registry itself (`internal/feature`) follows the same shape as
`internal/version`: add a field to `Feature` in `feature.go`, and it just
round-trips through `Loader.Load`/`Save` — no other changes needed there.

Release Planning logic lives in `internal/release/featureplanning.go`.
Two things to preserve if you touch it:

- **`Pending` is always derived, never stored.** Every mutation that
  changes `Features.Included`/`Features.Deferred` (or approves a new
  feature) recomputes `Features.Pending` via `pendingFeatureIDs` before
  saving — never append/remove from `Pending` directly, or it will drift
  out of sync with the registry.
- **Mutating feature-planning methods check out `staging` first** (like
  `StartRelease`/`Build`/`ReleaseFixFinish`), because the registry's
  canonical home is `staging` even though a feature's code is merged into
  `develop` first — see
  [Architecture.md](Architecture.md#data-files) for why. `ListApprovedFeatures`/
  `FeatureStatus` are the read-only exception, matching `Status`/
  `Manifest`/`Version`'s "read whatever's on the current branch" behavior.

Add both a fake-based error-path test (`fakeFeatureLoader` in
`fake_test.go`, override `deps.FeatureLoader` after `happyDeps()`) and a
real-repo test in `service_test.go` for any new feature-planning method —
see `TestFeaturePlanningLifecycleRealRepo` for the pattern.

## Logging & Configuration

`internal/logging` is Git Flow Plus's own diagnostic logging (distinct
from `git flow`'s user-facing command output, which stays on
`app.Println`/`app.Printf` — see "Business logic never lives in
`internal/cli`" above). It builds on `log/slog` rather than replacing it:
every service still takes a plain `*slog.Logger` and calls
`.Info(msg, "key", value, ...)` exactly as before. What's new lives on top
of that:

- **Extra levels** (`levels.go`): `LevelTrace` (finer than Debug),
  `LevelSuccess` (between Info and Warn — rendered `[SUCCESS]`), and
  `LevelFatal` (above Error), alongside slog's built-in four. Use
  `logging.Trace(logger, ...)`/`logging.Success(logger, ...)` — plain
  `*slog.Logger` has no methods for these, since they're not part of
  `log/slog` itself.
- **Console format** (`console_handler.go`): a custom `slog.Handler`
  rendering `[LEVEL] message  key: value` instead of slog's default
  `time=... level=INFO msg="..." key=value` text format, with optional
  ANSI color per level. This is what `logging.New(Options{Format:
  FormatText, ...})` returns; `FormatJSON` still uses `slog.NewJSONHandler`
  unchanged.
- **Error formatting** (`apperror.go`): `logging.AppError` optionally
  attaches a `Cause`/`Resolution` to an existing error without changing
  where that error was created — wrap at the CLI boundary (see
  `internal/cli/app.go`'s `LoadConfig` for the one real example so far),
  not deep in a service. `logging.LogError` is what `cli.Execute` calls on
  any command failure; it renders the message plus `Cause:`/`Resolution:`
  lines if present, nothing extra otherwise. This is additive — a plain
  `fmt.Errorf`-wrapped error anywhere else in the codebase still renders
  fine as a bare `[ERROR] <message>`.

Color/level/format resolution — precedence explicit CLI flag > config.json
`logging` block > built-in default — lives in `internal/cli/root.go`'s
`buildLogger`. Terminal detection and Windows's `ENABLE_VIRTUAL_TERMINAL_
PROCESSING` opt-in live in `internal/cli/color*.go` — see
[CrossPlatformGuide.md](CrossPlatformGuide.md#2-console-color-needs-an-opt-in-step-on-windows).

`config.Config` gained `Environment` (development/testing/production) and
`Logging` fields (`config.go`); `config.DefaultLogging(env)` seeds sane
defaults per environment, and `config.ApplyEnvOverrides` (called once,
inside `App.LoadConfig`) layers `GITFLOWPLUS_ENV`/`GITFLOWPLUS_LOG_LEVEL`/
`GITFLOWPLUS_LOG_FORMAT`/`GITFLOWPLUS_COLOR`/`GITFLOWPLUS_VERBOSE`/
`GITFLOWPLUS_DEBUG` on top. If you add a new configurable logging
knob, it needs a field on `config.Logging`, a case in `ApplyEnvOverrides`,
and a precedence line in `buildLogger` — follow the existing five as the
template.

## Adding a new lifecycle hook event

Hooks are just named events run via `service.runHook(ctx, "<event-name>",
env)` (see `internal/release/build.go` or `finish.go` for the existing
`post-qa-tag`/`post-production-tag` calls). To add a new one: pick a
`kebab-case` name, call `runHook` at the point in the service method where
the event has genuinely happened (i.e., after the state change it
describes is already committed — a hook should never fire for something
that might still roll back), and document the event and its environment
variables in
[ReleaseManagement.md](ReleaseManagement.md#lifecycle-hooks-cicd-integration).

## Cross-platform notes

See [CrossPlatformGuide.md](CrossPlatformGuide.md) for the full picture —
supported platforms, the path-handling discipline (`path.Join` for git
arguments vs `filepath.Join` for real filesystem access), Windows console
color, and lifecycle hook script dispatch (`.sh`/`.ps1`/`.bat`/`.cmd`).
The one-line version: `internal/git`, `internal/hooks`, and now
`internal/cli` (for Windows console-mode setup) are the only packages with
any platform-specific code, and every case is isolated behind an
interface or a `//go:build` tag rather than leaking into callers.
