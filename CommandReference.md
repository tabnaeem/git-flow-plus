# Command Reference

Global flags (available on every command):

| Flag | Default | Meaning |
|---|---|---|
| `-v`, `--verbose` | off | Debug-level logging. |
| `--debug` | off | Trace-level logging (the most detailed level). |
| `--json-log` | off | Structured logs as JSON instead of text (for CI/CD). |
| `--no-color` | off | Disable ANSI color in console (text-format) log output. |
| `--path <dir>` | current directory | Repository to operate on. |

All commands operate on the repository at `--path` (or the current
directory) and print human-readable output to stdout; structured,
optionally colorized logs (`[LEVEL] message  key: value`, or JSON with
`--json-log`) go to stderr. Defaults for level/format/color come from
`config.json`'s `logging` block (itself seeded by `environment`) and can
be overridden with `GITFLOWPLUS_*` environment variables before falling
back to these flags — see
[DeveloperGuide.md](DeveloperGuide.md#logging--configuration) for the full
precedence order and environment variable names. A command's own failure
is always reported as `[ERROR] <message>`, with `Cause:`/`Resolution:`
lines when available — never a raw stack trace.

---

## `git flow init`

Bootstraps `main`, `staging`, and `develop` (creating an initial commit
only if the repository is otherwise empty) and commits
`.gitflowplus/config.json` to all three. Safe to re-run — it's a no-op if
everything already exists. Ends on `develop`.

```bash
git flow init
```

---

## `git flow feature start <name>`

Branches `feature/<name>` from **`staging`** (not `develop` — the release
lifecycle begins on staging; see
[ReleaseManagement.md](ReleaseManagement.md#the-branch-model)), then
registers the feature in the **Feature Registry**
(`.gitflowplus/features.json`) as `Created` — checking out `staging`
briefly to do so (the registry lives there permanently), then back to the
new feature branch.

**There is no `feature finish`.** A developer never merges their own
feature branch: commit, push, and open a pull request on whatever Git host
you use. Only a Release Manager decides if and when a feature becomes part
of a release — see `git flow release feature add` below.

```bash
git flow feature start LOGIN
# Switched to a new branch "feature/LOGIN", based on "staging"
# Registered feature "LOGIN"
```

---

## `git flow hotfix start <name>` / `finish <name>`

Branches `hotfix/<name>` from `main`. Finish tags `main` (`v<name>`), then
merges into `main`, `staging`, and `develop` (in that order — so `staging`
never drifts behind a production hotfix), then deletes the branch.

```bash
git flow hotfix start critical-auth-bug
git flow hotfix finish critical-auth-bug
```

---

## `git flow support start <name>` / `finish <name>`

Branches `support/<name>` from `main` for long-term maintenance of an old
release line. **`finish` does not merge or delete the branch** — support
lines are intentionally permanent; merging old support work forward would
reintroduce outdated code. `finish` just validates the branch exists and
confirms it's staying in place.

```bash
git flow support start v4-maintenance
git flow support finish v4-maintenance   # branch remains, untouched
```

---

## `git flow release start <name>`

`<name>` is `Sprint.Release`, e.g. `5.2`. Checks out `staging` (no new
branch — staging is permanent), bootstraps `version.json` at
`Sprint.Release.0.0.1` and `release.json`, commits both, and tags the
initial version (QA Build 1).

Fails with `ErrReleaseAlreadyActive` if a release is already active on
staging — finish it first.

```bash
git flow release start 5.2
# Started release "5.2" on "staging" at version 5.2.0.0.1
# Tagged "v5.2.0.0.1"
```

---

## `git flow releasefix start <name>` / `finish <name>`

Branches `release-fix/<name>` from `staging`. **`finish` merges into
staging and records the fix as pending — it does not touch the version.**
Run `release build` to fold it in. The branch is **not** deleted by
`finish` — like feature branches, it stays alive until `release finish`
deletes it, so a follow-up fix can land on the same branch if needed.

```bash
git flow releasefix start BUG-101
# ...fix, commit...
git flow releasefix finish BUG-101
# Merged release fix "BUG-101" into staging (release "5.2"); pending inclusion in the next QA build
```

---

## `git flow devops start <name>` / `finish <name>`

Same shape as `releasefix`, for infrastructure/pipeline changes:
`release-devops/<name>`, branched from and merged into `staging`. Also
only records the change as pending, and also leaves the branch alive
until `release finish`.

```bash
git flow devops start redis-cache
# ...change, commit...
git flow devops finish redis-cache
```

---

## `git flow release build`

The only command that changes the version. **Must be run from `staging`**
(`ErrNotOnStaging` otherwise — deliberately not auto-checked-out; cutting
a build is a conscious action). Fails with `ErrNothingToBuild` if nothing
is pending.

Folds every pending release fix and DevOps change into the version
(`Fixes += pending fixes`, `DevOps += pending devops`, `QA += 1`), moves
them from pending to included, appends a build-history entry, and
**always** creates an annotated tag (`v<version>`) — tagging every QA
build is mandatory, not optional. Triggers the `post-qa-tag` lifecycle
hook.

```bash
git flow release build
# QA Build #2: version 5.2.4.0.2 (+4 release fix(es), +0 devops change(s))
# Tagged "v5.2.4.0.2"
```

---

## `git flow release finish <name>`

The Production Release step. Fails with `ErrPendingChangesNotBuilt` if
anything is still pending — run `release build` first.

Marks every feature currently `included` in the release `Released` in the
Feature Registry (`release: "<name>"`). Archives `release.json` to
`.gitflowplus/archive/<name>.json`, removes the live `release.json`/
`version.json` (resetting staging's counters for the next cycle), merges
`staging` → `main` (`develop` is untouched — it's not part of the release
lifecycle), and tags the exact commit that merged into `main` as `v<name>`
(not the full version string — see
[ReleaseManagement.md](ReleaseManagement.md#tagging) for why). **Only now**
deletes every feature branch that was included in the release, every
`release-fix/*` branch, and every `release-devops/*` branch — best-effort;
a branch that somehow can't be deleted cleanly is logged as a warning, not
a command failure (see
[ReleaseManagement.md](ReleaseManagement.md#release-completion)). Triggers
the `post-production-tag` lifecycle hook. `staging` is never deleted.

```bash
git flow release finish 5.2
# Finished release "5.2" at version 5.2.4.0.2, tagged "v5.2"
```

---

## `git flow release status`

Prints a full human-readable summary of the release active on `staging`:
the version's Sprint/Features/Release Fixes/DevOps/QA Build counts, every
feature and release fix (Included/Deferred/Pending), the QA build
history, and a Release Readiness checklist. If no release is active:
"No active release on staging."

```bash
git flow release status
```

```
Git Flow Plus Release Status
──────────────────────────────────────

Release:       v5.2.1.1.2
Sprint:        5
Features:      2
Release Fixes: 1
DevOps:        1
QA Builds:     2

Features
──────────────────────────────────────

✓ LOGIN        Included
○ REPORT       Pending

Release Fixes
──────────────────────────────────────

✓ FIX-101

QA
──────────────────────────────────────

✓ Build #1
→ Build #2 Current

Release Readiness
──────────────────────────────────────

Features              ○ Pending (1)
Release Fixes         ✓
DevOps Changes        ✓
QA                    → Build #2 in progress
Production Release    ○ Pending (run 'git flow release finish')

Status:
NOT READY FOR PRODUCTION
```

Readiness is computed purely from what release.json already tracks:
Features/Release Fixes/DevOps Changes turn `✓` the moment nothing is left
`Pending`. QA and Production Release are always shown "in progress"/
"Pending" — Git Flow Plus records no "QA complete" or "production
approved" flag anywhere, and an active release is by definition one
`release finish` hasn't run for yet (that command archives and removes
the manifest `status` reads).

### `--json`

```bash
git flow release status --json
```

```json
{
  "release": "v5.2.1.1.2",
  "sprint": 5,
  "features": 2,
  "release_fixes": 1,
  "devops": 1,
  "qa_build": 2,
  "status": "not_ready"
}
```

`status` is `"ready"` once Features/Release Fixes/DevOps Changes have
nothing pending, `"not_ready"` otherwise, or `"no_release"` (with just
`{"active": false, "status": "no_release"}`, since there's no release to
report the other fields for) if `staging` has no active release.

---

## `git flow release validate`

A read-only pre-flight for `git flow release finish`: checks whether the
release active on `staging` is actually ready to finish, reusing exactly
the same guards `finish` itself enforces (pending release fixes/DevOps
changes, the production tag not already existing) plus a handful of
structural-integrity checks against the manifest, version, and Feature
Registry. Never mutates anything. Exits non-zero on failure, so it also
works as a CI gate before a scripted `release finish`.

```bash
git flow release validate
```

```
Release validation passed.

✓ Release exists
✓ Release branch exists
✓ Release state valid
✓ Features finalized
✓ Features approved
✓ Release fixes finalized
✓ DevOps changes finalized
✓ QA status valid
✓ Version valid
✓ Working tree clean
✓ Release tag available
✓ No conflicting release state

Release is ready.
```

On failure, only the specific problems found are printed — one line per
problem, not a generic "some check failed":

```
Release validation failed.

✗ Feature REPORT is pending a Release Planning decision (approve and add, or defer, it)
✗ Release fix FIX-101 has been merged but is not yet included in a QA build; run 'git flow release build'

Release cannot be finalized.
```

A few of these check names describe concepts Git Flow Plus tracks no
dedicated flag for, so they're grounded in the closest real, derivable
fact instead of a second approval system:

- **Features finalized** — nothing left in `Manifest.Features.Pending`
  (approved, but not yet added to or deferred from the release).
- **Features approved** — a distinct integrity check: every feature the
  manifest already lists as `Included` must still resolve, in the
  Feature Registry, to at least `Approved` — catches registry/manifest
  drift, since `release feature add` itself can never produce this.
- **QA status valid** — not an "approved build" flag (none exists); the
  manifest's build history must be internally consistent with the live
  version (the latest recorded build matches the current QA iteration
  and version string).
- **No conflicting release state** — the manifest's recorded branch
  matches the currently configured staging branch. Git Flow Plus only
  ever tracks one active release at a time by construction (`release
  start` refuses a second one), so this is the one place drift between
  the manifest and live config could otherwise go unnoticed.

---

## `git flow release feature list`

Lists every feature in the Feature Registry that has reached at least
`Approved`, regardless of release assignment, with its current state and
shipped-release status.

```bash
git flow release feature list
# LOGIN            state=IncludedInRelease  release=(none)
# PROFILE          state=Released           release=5.2
```

---

## `git flow release feature approve <id>`

Marks a feature `Approved` — the gate that makes it eligible for Release
Planning. Fails with `ErrFeatureNotFound` if `id` isn't in the registry
(run `feature start` first), or `ErrFeatureAlreadyApproved` if it has
already reached at least `Approved`.

```bash
git flow release feature approve LOGIN
```

---

## `git flow release feature add <id>`

The **Release Planning decision, and a real merge**: validates
`feature/<id>` exists, merges it into staging, advances the version's
feature counter, and marks the feature `IncludedInRelease`. The branch is
**not** deleted — it stays alive until `release finish`. Requires an
active release on staging (`ErrNoActiveRelease`) and an approved feature
(`ErrFeatureNotApproved`); fails with `ErrFeatureAlreadyAssigned` only for
a feature that already shipped in a *finished* release.

**Safe to re-run** for a feature already `IncludedInRelease` in the active
cycle — this is how a developer's QA follow-up commits, pushed to the
still-alive branch, actually reach staging. A repeat call merges again
(a no-op if there's nothing new) without incrementing the feature counter
a second time. See
[ReleaseManagement.md](ReleaseManagement.md#what-release-planning-does-now).

```bash
git flow release feature add LOGIN
# Merged feature "LOGIN" into staging and added it to the active release
```

---

## `git flow release feature defer <id>`

Explicitly holds an approved feature back for a future release. Bookkeeping
only — no merge, no branch touched. Only works *before* `add` has merged
the feature (same precondition errors as `add`, minus the merge step).

```bash
git flow release feature defer REPORTS
```

There is no `release feature remove`: once `add` has performed a real
merge, reverting it isn't something Git Flow Plus attempts — see
[ReleaseManagement.md](ReleaseManagement.md#release-feature-commands).

---

## `git flow release feature status`

Shows Approved / Included / Deferred / Pending feature IDs for the active
release (Pending is always derived, never stored).

```bash
git flow release feature status
# Approved:  LOGIN, PROFILE, PAYMENT, REPORTS, DASHBOARD
# Included:  LOGIN, PROFILE, PAYMENT
# Deferred:  REPORTS
# Pending:   DASHBOARD
```

---

## `git flow release version`

Prints the current `Sprint.Release.Fix.DevOps.QAIteration` version, e.g.
`5.2.4.0.2`. Errors if no release is active.

```bash
git flow release version
```

---

## `git flow release manifest`

Prints the current `release.json` as formatted JSON. Errors if no release
is active.

```bash
git flow release manifest
```

---

## `git flow doctor`

Colorized health checks, in order: git binary present, git version,
valid repository, `main`/`staging`/`develop` branches present, working
tree state, repository directory writable, `.gitflowplus/config.json`
present, Git Flow Plus's own build version, whether `git-flow` resolves
on `PATH` (required for `git flow ...` to work as a Git subcommand), and
whether a release is currently in progress. Exits non-zero if any
non-informational check fails — see
[Troubleshooting.md](Troubleshooting.md) for what to do about each one.

```bash
git flow doctor
```

---

## `git flow config` / `config list` / `config path`

`config` (bare) and `config list` print the effective configuration
(environment, branch names, prefixes, and logging settings), defaulting to
human-readable text; pass `--json` for JSON. `config path` prints the
resolved path to `config.json` for the current repository.

```bash
git flow config              # same as config list
git flow config list --json
git flow config path
```

---

## `git flow version`

Prints Version, Build Number, Git Commit, Git Branch, Build Date, Go
Version, OS, and Architecture. The first five are embedded at build time
via `-ldflags` (see [Building.md](Building.md)); an unstamped `go
build` shows `"dev"`/`"unknown"` defaults instead.

```bash
git flow version
# Git Flow Plus
#
# Version      : 1.2.0
# Build Number : 456
# Git Commit   : a1b2c3d
# Git Branch   : main
# Build Date   : 2026-08-01T09:00:00Z
# Go Version   : go1.26.5
# OS           : linux
# Architecture : amd64
```

This is distinct from `git flow release version`, which prints the
*active release's* `Sprint.Release.Fix.DevOps.QAIteration` version — a
property of the repository, not of the `git-flow-plus` binary itself.
