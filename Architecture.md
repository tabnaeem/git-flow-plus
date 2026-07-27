# Architecture

## Layers

```
cmd/git-flow-plus          entrypoint (main.go: os.Exit(cli.Execute()))
        │
internal/cli                Cobra command tree — parses args, prints
        │                    output, delegates. No Git or release logic.
        │
        ├── internal/gitflow     Git Flow branching model: init, feature,
        │        │               hotfix, support, and the git-level parts
        │        │               of release (merge/tag mechanics only).
        │        │
        │        └── internal/git    Thin wrapper over the real `git`
        │                            binary. Every operation shells out,
        │                            so behavior matches plain Git exactly.
        │
        ├── internal/release      Git Flow Plus's release management:
        │        │                the manifest, version bootstrapping,
        │        │                QA builds, tagging, archiving, and
        │        │                Release Planning (feature selection).
        │        │                Composes gitflow + version + feature +
        │        │                git + hooks.
        │        │
        │        ├── internal/version   Sprint.Release.Fix.DevOps.QAIteration
        │        │                      parsing/formatting/storage.
        │        │
        │        ├── internal/feature   The Feature Registry: one permanent
        │        │                      record per feature ID, tracking its
        │        │                      lifecycle (merged into develop ->
        │        │                      unit tested -> approved -> shipped
        │        │                      in a release), independent of any
        │        │                      single release cycle.
        │        │
        │        └── internal/hooks     CI/CD-agnostic lifecycle hook
        │                                execution (post-qa-tag,
        │                                post-production-tag).
        │
        └── internal/config       .gitflowplus/config.json (branch names,
                                   prefixes, environment, and logging
                                   defaults) via Viper.

internal/logging             Structured, leveled logging (built on log/slog)
                              used throughout: a colorized human-readable
                              console format, a JSON format for CI/CD, and
                              optional Cause/Resolution error formatting.
```

Each arrow is a real Go import; there are no cycles. `internal/gitflow`
knows nothing about versions or manifests. `internal/git` knows nothing
about Git Flow — it's a general-purpose Git client that happens to be used
by one.

## Why this split

**`internal/git` is a real Git client, not a Git Flow client.** Every
method (`Checkout`, `MergeNoFF`, `Tag`, `TagCommit`, `CommitSHA`,
`ConfigValue`, ...) maps to one `git` invocation and knows nothing about
branch naming conventions. This is what lets `internal/gitflow` compose
primitives into `feature start` = "create branch from develop" without
`internal/git` needing to know what "develop" means. It also makes the
whole tool trivially compatible with any Git host or GUI — nothing here
does anything a `git` command line wouldn't.

**`internal/gitflow` is the branching model, not release management.** It
implements `init`/`feature`/`hotfix`/`support` fully (branch, merge, tag,
delete), and for release it implements only the shared git mechanics:
`ReleaseFixStart/Finish` and `DevOpsStart/Finish` (branch from staging,
merge into staging), and `ReleaseFinish` (merge staging → main → develop,
returning the merge commit's SHA). It does **not** know about versions,
manifests, or QA builds — ask it "what release is active" and it has no
answer, because that's not its job.

**`internal/release` is Git Flow Plus's actual value-add.** It owns the
manifest schema, the version-bootstrap-and-tag sequence, and the rule that
*only* `Build` moves the version — never a merge. It composes
`gitflow.Service` for git mechanics, `git.Client` directly for
staging/committing/tagging the metadata files (and now the annotated
release tags themselves — see below), `version.Loader`/`Loader` for
persistence, and `hooks.Runner` for CI/CD notifications.

**`internal/feature` is a registry, not a workflow engine.** It's
deliberately as small as `internal/version`: a `Feature` struct, a
`Registry` (a slice with `Find`/`Upsert`/`Approved`), and a `Loader` for
`features.json` — the same shape as `internal/version`'s `Version`/
`Loader`. It has no dependency on `internal/gitflow` or `internal/release`;
those depend on *it*. All the actual Release Planning logic (who can
approve what, what's pending, how a release's Included/Deferred sets are
derived) lives in `internal/release/featureplanning.go`, not here — mirrors
how `internal/version` just stores numbers while `internal/release`
decides when they change.

**Why does `internal/release` create tags itself instead of asking
`internal/gitflow` to?** The production tag must reference the exact
commit that merged into `main`, and its message needs manifest data
(`internal/gitflow` doesn't have a manifest). So `gitflow.ReleaseFinish`
merges and hands back a commit SHA; `internal/release` builds the tag
message from the manifest and tags that SHA directly via
`git.Client.TagCommit`. This keeps `internal/gitflow` a pure git-mechanics
layer and keeps "what goes in a Git Flow Plus tag message" a single
concern in one file (`internal/release/tagmessage.go`).

**`internal/logging` builds on `log/slog`, it doesn't replace it.** Every
service still takes a plain `*slog.Logger` and calls `.Info(msg, "key",
value)` — nothing about that changed. What `internal/logging` adds sits
entirely inside `logging.New`: three extra levels (Trace, Success, Fatal)
interleaved with slog's own, a custom `slog.Handler` for the human-readable
`[LEVEL] message  key: value` console format (JSON still uses
`slog.NewJSONHandler` directly, unchanged), and an optional `AppError` type
for attaching a `Cause`/`Resolution` to an error without touching where
that error was originally created. Terminal detection and color
enablement are deliberately *not* in this package — they're an
`internal/cli` concern (see `color.go`/`color_windows.go`/
`color_other.go`), so `internal/logging` stays pure and trivially testable
against a `bytes.Buffer`, the same way every other package here separates
"what to do" from "how to talk to the OS."

**Why a separate `internal/hooks` package instead of hardcoding
CI/CD integrations?** Because the tool has to stay CI/CD-agnostic
(GitHub Actions, GitLab CI, Azure DevOps, Jenkins, Bitbucket Pipelines, or
anything else). The simplest way to do that without picking favorites is
the same approach Git itself uses: a `Runner` interface that looks for an
executable script at a well-known path and runs it, passing context via
environment variables. Any CI/CD system can also just trigger off the
pushed tag directly — the hook exists for cases that need something
*before* or *independent of* a tag push (e.g. calling a webhook, or
notifying a system that doesn't watch this repo's tags at all).

## Dependency injection

Nothing in `internal/gitflow` or `internal/release` touches `os.Exec`,
`os.Stdout`, or the filesystem directly except through an injected
interface (`git.Runner`, `git.Client`, `config.Loader`, `version.Loader`,
`release.Loader`, `feature.Loader`, `hooks.Runner`). `internal/cli.App` wires the real
implementations; tests substitute fakes (for fast, isolated unit tests of
error paths) or point the real implementations at a `t.TempDir()` repo
(for true end-to-end tests that actually invoke the `git` binary). Both
styles exist side by side in every package's test suite — see
[DeveloperGuide.md](DeveloperGuide.md#testing-style).

## Data files

Everything lives under `.gitflowplus/` in the repository, tracked and
committed like any other file:

| File | Owner | Branch(es) | Purpose |
|---|---|---|---|
| `config.json` | `internal/config` | main, staging, develop | Branch names, prefixes, environment (development/testing/production), and logging defaults. |
| `version.json` | `internal/version` | staging (while a release is active) | Current `{sprint, release, fixes, devops, qa}`. |
| `release.json` | `internal/release` | staging (while a release is active) | The manifest: included/pending/deferred features, included/pending fixes and DevOps changes, build history. |
| `features.json` | `internal/feature` | staging (permanent, never reset) | The Feature Registry: every feature ever merged into develop, and its lifecycle state. |
| `archive/<release>.json` | `internal/release` | main, develop (post-finish) | Permanent snapshot of a finished release's manifest. |
| `hooks/<event>[.sh\|.ps1\|.bat\|.cmd]` | user-authored | wherever needed (typically staging) | Lifecycle hook scripts. |

`version.json`/`release.json` only exist on staging *while a release is in
progress* — `release finish` removes them (after archiving) so staging
doesn't carry a stale "current release" between cycles. `features.json` is
different: unlike the manifest, it's never removed or reset — a feature's
lifecycle (registered, approved, shipped) has to survive across many
release cycles, so it's committed to staging by every feature-planning
operation (`git flow feature finish`, and every `release feature ...`
subcommand) and flows forward into `main`/`develop` the same way
`release.json` does, just without ever being deleted. `config.json` is
seeded onto `main`/`staging`/`develop` at `init` time so it (and anything
else placed in `.gitflowplus/`, like hook scripts) is available regardless
of which branch a command runs from.

## What's read-only vs. what mutates

`internal/git.Client` and `internal/gitflow.Service` are stateless with
respect to Go — all state lives in the actual Git repository on disk. Two
processes running commands against the same repo concurrently would race
exactly as if two people ran `git` commands by hand; there is no
additional locking. This is intentional: Git Flow Plus is a thin,
predictable layer over Git, not a service with its own consistency model.
