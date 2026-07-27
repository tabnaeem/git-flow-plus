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

## `git flow feature start <name>` / `finish <name>`

Standard Git Flow. Branches `feature/<name>` from `develop`; finish merges
back into `develop`, deletes the branch, then registers (or updates) the
feature in the **Feature Registry** (`.gitflowplus/features.json`) as
merged into develop — checking out `staging` to do so, since the registry
is permanent and lives there (see
[ReleaseManagement.md](ReleaseManagement.md#feature-management--release-planning)).

```bash
git flow feature start LOGIN
git flow feature finish LOGIN
# Merged "feature/LOGIN" into "develop" and deleted it
# Registered feature "LOGIN" as merged into develop
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
Run `release build` to fold it in.

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
only records the change as pending.

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

Marks every feature currently `included` in the release as permanently
shipped in the Feature Registry (`includedInRelease: true`, `release:
"<name>"`). Archives `release.json` to
`.gitflowplus/archive/<name>.json`, removes the live `release.json`/
`version.json` (resetting staging's counters for the next cycle), merges
`staging` → `main` → `develop`, and tags the exact commit that merged into
`main` as `v<name>` (not the full version string — see
[ReleaseManagement.md](ReleaseManagement.md#tagging) for why). Triggers
the `post-production-tag` lifecycle hook. `staging` is never deleted.

```bash
git flow release finish 5.2
# Finished release "5.2" at version 5.2.4.0.2, tagged "v5.2"
```

---

## `git flow release status`

Reports the release active on `staging` — release name, branch, version,
QA build number, included/pending release fixes and DevOps changes,
included/deferred/pending features, and open `release-fix`/
`release-devops` branches. If no release is active: "No active release on
staging."

```bash
git flow release status
```

---

## `git flow release feature list`

Lists every **approved** feature in the Feature Registry, regardless of
release assignment, with its shipped-release status.

```bash
git flow release feature list
# LOGIN            release=(none)   included=false
# PROFILE          release=5.2      included=true
```

---

## `git flow release feature approve <id>`

Marks a feature approved after Unit Testing — the gate that makes it
eligible for Release Planning. Requires the feature to already be
registered as merged into develop (`ErrFeatureNotMergedIntoDevelop`
otherwise — run `feature finish` first). Fails with `ErrFeatureNotFound`
if `id` isn't in the registry, or `ErrFeatureAlreadyApproved` if it's
already approved.

```bash
git flow release feature approve LOGIN
```

---

## `git flow release feature add <id>`

The **Release Planning decision**: assigns an approved feature to the
active release. Requires an active release on staging
(`ErrNoActiveRelease`) and an approved, unshipped feature
(`ErrFeatureNotApproved`, `ErrFeatureAlreadyAssigned`). Moves `id` out of
`deferred` if it was there. Does not move any code — see
[ReleaseManagement.md](ReleaseManagement.md#what-release-planning-does-and-doesnt-do).

```bash
git flow release feature add LOGIN
```

---

## `git flow release feature remove <id>`

Undoes `add`, returning the feature to `pending`. Fails with
`ErrFeatureNotAssignedToCurrentRelease` if `id` isn't currently included
in the active release.

```bash
git flow release feature remove LOGIN
```

---

## `git flow release feature defer <id>`

Explicitly holds an approved feature back for a future release, moving it
out of `included` if it was there. Same precondition errors as `add`.

```bash
git flow release feature defer REPORTS
```

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

Health checks: git binary present, valid repository, `main`/`staging`/
`develop` branches present, working tree clean, `.gitflowplus/config.json`
present. Exits non-zero if any check fails.

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
via `-ldflags` (see [BuildGuide.md](BuildGuide.md)); an unstamped `go
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
