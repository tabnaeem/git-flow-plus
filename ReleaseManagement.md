# Release Management

This is the part of Git Flow Plus that isn't standard Git Flow. Read this
if you're going to run a release, not just contribute code.

## The branch model

```
main
 │
 ├── develop            feature development + unit testing, never released from directly
 │      └── feature/*
 │
 └── staging            THE release branch — permanent, reused every cycle
        ├── release-fix/*
        ├── release-devops/*
        ├── QA Build (tag)
        ├── QA Feedback
        └── Production Release (tag on main)
```

`staging` is not a temporary branch cut per release and thrown away. It's
created once by `git flow init` and lives forever, the same way `main` and
`develop` do. Each release cycle (`5.2`, then `5.3`, ...) reuses it in
turn: `release start` resets its metadata for a new cycle, `release
finish` clears that metadata back out when the cycle ends.

## The version scheme

`Sprint.Release.Fix.DevOps.QAIteration` — e.g. `5.2.4.1.3`:

| Field | Meaning |
|---|---|
| `5` | Sprint number |
| `2` | Feature Release number within the sprint — which batch of planned features this release is |
| `4` | Release fixes **included in the current QA build** |
| `1` | DevOps changes **included in the current QA build** |
| `3` | QA build number |

This is not Semantic Versioning. It exists to answer one question: *what,
exactly, is in this QA build.* The second field identifies *which planned
release* this is (see [Feature Management &
Release Planning](#feature-management--release-planning) below for how
features get assigned to it) — the version format itself doesn't change to
reflect that; a release is still just "Sprint.Release" (e.g. `5.3`), the
same name used by `release start`/`release finish` and the Feature
Registry's `release` field.

## The rule that matters: merging is not releasing

**Merging a `release-fix` or `release-devops` branch into `staging` never
changes the version.** You can merge ten release-fix branches in a row and
the version won't move. This is deliberate: it decouples "a fix landed" from
"a fix shipped in a QA build," so:

- Developers can merge fixes continuously without needing to coordinate a
  version bump for each one.
- The Release Manager decides *when* a batch of fixes becomes an official,
  tagged, deployable QA build — a conscious act, not a side effect of
  someone else's merge.
- The version number always corresponds to something a QA engineer can
  actually pull and test, not an arbitrary mid-merge state.

Concretely:

```
git flow release start 5.2       # staging: 5.2.0.0.1 (QA Build 1, tagged v5.2.0.0.1)

git flow releasefix finish BUG-101   # staging still 5.2.0.0.1
git flow releasefix finish BUG-102   # staging still 5.2.0.0.1
git flow releasefix finish BUG-103   # staging still 5.2.0.0.1
git flow releasefix finish BUG-104   # staging still 5.2.0.0.1

git flow release build           # NOW: 5.2.4.0.2 (4 new fixes, QA Build 2, tagged v5.2.4.0.2)

git flow releasefix finish BUG-201
git flow releasefix finish BUG-202   # staging still 5.2.4.0.2

git flow release build           # 5.2.6.0.3 (2 more fixes, QA Build 3, tagged v5.2.6.0.3)
```

`releasefix finish`/`devops finish` only ever do three things: merge the
branch into staging, delete it, and record the change as **pending** in
`release.json`. `release build` is the only command that reads the
pending lists, folds their counts into the version (`Fixes += len(pending
fixes)`, `DevOps += len(pending devops)`, `QA += 1`), moves them from
pending to included, and appends a build-history entry. Nothing else
touches `version.json`.

## Feature Management & Release Planning

Git Flow Plus is not just a Git branching tool — it's a release management
platform, and a release is a *curated collection of approved features*,
not everything that happens to be sitting on `develop`. The full SDLC it
models:

```
feature/* (Feature Development)
   │
   ▼
develop (Merge Feature into develop → Unit Testing → Feature Approved)
   │
   ▼
staging (Release Planning → Release Start → QA → Release Fixes → QA Builds)
   │
   ▼
main (Production)
```

### The Feature Registry

Every feature that's ever been merged into `develop` gets one permanent
entry in the **Feature Registry** (`.gitflowplus/features.json`),
independent of any single release cycle — unlike `release.json`, it's
never reset:

```json
{
  "features": [
    {
      "id": "LOGIN",
      "branch": "feature/LOGIN",
      "mergedIntoDevelop": true,
      "mergeCommit": "a1b2c3d...",
      "unitTested": true,
      "approved": true,
      "includedInRelease": true,
      "release": "5.3"
    }
  ]
}
```

`git flow feature finish <name>` registers (or updates) the entry
automatically — that's the only place `mergedIntoDevelop`/`branch`/
`mergeCommit` get set. Everything else about a feature's lifecycle is
driven by `release feature` subcommands:

| Command | What it does |
|---|---|
| `git flow release feature approve <id>` | Marks a feature approved after Unit Testing. Requires it to already be merged into develop. |
| `git flow release feature list` | Lists every approved feature in the registry. |
| `git flow release feature add <id>` | **Release Planning decision**: assigns an approved feature to the active release. |
| `git flow release feature remove <id>` | Undoes `add`, returning the feature to Pending. |
| `git flow release feature defer <id>` | Explicitly holds an approved feature back for a future release. |
| `git flow release feature status` | Shows Approved / Included / Deferred / Pending for the active release. |

**Git Flow Plus never assumes every approved feature belongs to the next
release.** A feature stays `Pending` — approved, but with no release
decision made yet — until a Release Manager explicitly runs `add` or
`defer`. Given LOGIN, PROFILE, PAYMENT, and REPORTS are all approved, a
Release Manager might decide only LOGIN, PROFILE, and PAYMENT ship in
`5.3`, deferring REPORTS to `5.4`:

```bash
git flow release feature approve LOGIN
git flow release feature approve PROFILE
git flow release feature approve PAYMENT
git flow release feature approve REPORTS

git flow release feature add LOGIN
git flow release feature add PROFILE
git flow release feature add PAYMENT
git flow release feature defer REPORTS
```

`Pending` is never hand-maintained — it's always recomputed as *approved
minus included minus deferred minus already-shipped-in-a-past-release*, so
it can't drift out of sync with the registry.

### What Release Planning does and doesn't do

`release feature add`/`defer`/`remove` only update bookkeeping in
`release.json` and, at `release finish` time, permanently mark the
registry entry as shipped (`includedInRelease: true`, `release: "5.3"`).
**They never move code.** Git Flow Plus performs no automated
cherry-picking or merging of feature branches into `staging` — a
feature's code has to already be reachable from `staging` by whatever
means your team uses to promote it there (typically: `staging` picks up
`develop`'s history the same way any two long-lived branches converge,
via the normal merges that already happen in this workflow). This is
deliberate: automated cross-branch merging of arbitrary feature code is a
much higher-risk operation than recording a planning decision, and it's
out of scope for what "Release Planning" means here.

### The release manifest

`release.json`'s `features` block mirrors this decision directly:

```json
{
  "release": "5.3",
  "branch": "staging",
  "currentVersion": "5.3.2.1.4",
  "currentQABuild": 4,
  "features": {
    "included": ["LOGIN", "PROFILE", "PAYMENT"],
    "deferred": ["REPORTS"],
    "pending": ["DASHBOARD"]
  },
  "releaseFixes": { "included": ["BUG-101"], "pending": [] },
  "devops": { "included": [], "pending": ["redis-cache"] },
  "history": [ ... ]
}
```

Only `release feature add`/`remove`/`defer` change `features.included`/
`.deferred` — `release build` does **not** fold "pending" features into
"included" the way it does for release fixes and DevOps changes, because
feature inclusion isn't something that accumulates by merging; it's a
one-time planning decision. `release build` does still snapshot whichever
features are currently `included` into that build's history entry and QA
tag, so every QA build's Feature List is always accurate.

## Tagging

**Every QA build gets an annotated tag — this is not optional.**
Deployments key off these tags, so a build without one isn't deployable.

- `git flow release start` tags the initial version (QA Build 1) —
  `v5.2.0.0.1`.
- `git flow release build` tags every subsequent build — `v5.2.4.0.2`,
  `v5.2.6.0.3`, etc.
- `git flow release finish` creates a **separate** production tag, named
  after the release itself — `v5.2` — not the full version string. That
  name is already taken by the last QA build's tag (pointing at a
  different commit on staging), and Git tag names must be globally
  unique, so the production tag can't reuse it. `v5.2` marks the one
  production release of 5.2; its message still records the exact version
  and commit that shipped.

Every tag's message is structured and complete, so you never need to dig
through commit history to know what a tag represents:

```
Release: 5.2
Version: 5.2.4.0.2
QA Build: 2
Release Date: 2026-07-24T10:42:34Z
Release Manager: Jane Doe <jane@example.com>

Features:
LOGIN
PROFILE

Release Fixes:
BUG-101
BUG-102
BUG-103
BUG-104

DevOps:
None

Branch: staging
Commit: a1b2c3d4e5f6...
```

(The production tag's message is the same shape, with `Branch: main` and
no `QA Build` line — a production release isn't itself a QA iteration.)

## Lifecycle hooks (CI/CD integration)

Git Flow Plus stays CI/CD-agnostic: it never talks to GitHub Actions,
GitLab CI, Azure DevOps, Jenkins, or Bitbucket Pipelines directly. Two
ways to trigger a pipeline off a release event:

1. **Trigger off the tag push itself.** Since every QA build and every
   production release is a real annotated tag, any CI/CD system configured
   to watch tag patterns (e.g. `v*`) will fire automatically once you
   `git push --tags`. This needs no configuration in Git Flow Plus at all.
2. **Use a lifecycle hook script**, for anything that needs to happen
   locally, before a push, or independent of tag-watching (e.g. calling a
   webhook directly, writing to a local queue).

To use a hook, drop an executable script at:

```
.gitflowplus/hooks/post-qa-tag[.sh|.ps1|.bat|.cmd]
.gitflowplus/hooks/post-production-tag[.sh|.ps1|.bat|.cmd]
```

Pick the extension for your platform: `.sh` (run via `sh`), `.ps1` (run
via `powershell -File`), `.bat`/`.cmd` (run via `cmd /C`), or no extension
at all (run directly — relies on a shebang line and the executable bit on
Unix). Git Flow Plus tries them in that order and runs the first one it
finds; if none exist, nothing happens (hooks are entirely opt-in and never
cause a command to fail just because a script is missing).

**Commit hook scripts to `staging`** — that's the branch `release
start`/`release build` run from, and where `post-qa-tag` fires. The
`post-production-tag` hook fires after `release finish`'s merge sequence,
which ends on `develop`, so a hook needed there should be committed on
`develop` too (or, simplest: commit hook scripts once on `main` before
`staging`/`develop` diverge — `git flow init` seeds `config.json` onto all
three branches this same way, and hook scripts can ride along).

Each hook receives these environment variables:

| Variable | `post-qa-tag` | `post-production-tag` |
|---|---|---|
| `GITFLOWPLUS_EVENT` | `qa-build` | `production-release` |
| `GITFLOWPLUS_RELEASE` | ✓ (e.g. `5.2`) | ✓ |
| `GITFLOWPLUS_VERSION` | ✓ (e.g. `5.2.4.0.2`) | ✓ (full version that shipped) |
| `GITFLOWPLUS_TAG` | ✓ (e.g. `v5.2.4.0.2`) | ✓ (e.g. `v5.2`) |
| `GITFLOWPLUS_QA_BUILD` | ✓ | — |
| `GITFLOWPLUS_COMMIT` | — | ✓ |
| `GITFLOWPLUS_BRANCH` | ✓ (`staging`) | ✓ (`main`) |

A hook script that exits non-zero is logged as a warning but does **not**
fail the release command or undo the tag — the tag and version bookkeeping
are already committed by the time hooks run, and a broken deployment
trigger shouldn't be able to corrupt release state.

## The full command sequence

```bash
git flow init                        # main, staging, develop

git flow release start 5.2           # staging: version.json = 5.2.0.0.1
                                      # tags v5.2.0.0.1 (QA Build 1)

git flow releasefix start BUG-101    # release-fix/BUG-101, from staging
# ...fix, commit...
git flow releasefix finish BUG-101   # merge into staging; BUG-101 now PENDING

git flow devops start redis-cache    # release-devops/redis-cache, from staging
# ...change, commit...
git flow devops finish redis-cache   # merge into staging; redis-cache now PENDING

git flow release status              # see what's pending vs. included

git flow release build               # fold pending -> included, version -> 5.2.1.1.2
                                      # tags v5.2.1.1.2 (QA Build 2)
                                      # fires post-qa-tag hook

# ...QA validates the build...

git flow release finish 5.2          # requires: nothing pending (run build first)
                                      # archives release.json -> .gitflowplus/archive/5.2.json
                                      # removes live release.json/version.json (reset for next cycle)
                                      # merges staging -> main -> develop
                                      # tags v5.2 (production) at the exact commit that merged into main
                                      # fires post-production-tag hook

git flow release start 5.3           # staging reused, fresh cycle: version.json = 5.3.0.0.1
```

`release finish` refuses to run while anything is pending
(`ErrPendingChangesNotBuilt`) — nothing should ship without ever having
moved the version and been through a tagged QA build.

## `release status`

Shows everything about the release active on `staging` right now:

```
Release:                   5.2
Branch:                    staging
Version:                   5.2.4.0.2
QA Build:                  2
Included Release Fixes:    BUG-101, BUG-102, BUG-103, BUG-104
Pending Release Fixes:     (none)
Included DevOps:           (none)
Pending DevOps:            (none)
Included Features:         LOGIN, PROFILE, PAYMENT
Deferred Features:         REPORTS
Pending Features:          DASHBOARD
Open Release-Fix Branches: (none)
Open DevOps Branches:      (none)
```

"Active" means "does `staging`'s current working tree have a
`release.json`" — there's no separate global pointer to track. Run it from
any other branch and you'll see "No active release on staging." (`release
status` always reports on `staging` specifically, regardless of which
branch you're currently on, matching where release state actually lives.)

## Known gaps

- **One release at a time.** `release start` refuses to run if a release
  is already active on staging (`ErrReleaseAlreadyActive`). There's no
  support for two releases in flight simultaneously.
- **No automated feature-code promotion.** As covered above, Release
  Planning is bookkeeping-only — Git Flow Plus never cherry-picks or
  merges a feature branch into `staging` on your behalf.

See [Roadmap.md](Roadmap.md) for what's planned to close these.
