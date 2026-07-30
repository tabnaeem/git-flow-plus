# Release Notes

No version has been tagged yet — `git flow version` reports `0.1.0-dev`
until the first `v*` tag is pushed (see
[Building.md](Building.md#cicd)). This document tracks what's landed
so far, organized by milestone, so the first real release has an accurate
starting changelog rather than a single undifferentiated diff.

## Unreleased

### Installation and distribution system (this milestone)

A complete, professional install/release pipeline across all three
platforms, first-priority Windows:

- **Windows installer** (Inno Setup, `installer/windows/gitflowplus.iss`):
  silent install/uninstall, per-user or machine-wide, automatic PATH
  management (verified end-to-end against a real ~1,800-character PATH
  with 40+ existing entries — installed, `doctor` all green, uninstalled,
  PATH restored byte-for-byte), Git/Git Bash/PowerShell detection,
  automatic detection-and-silent-removal of a prior install on upgrade,
  and a seeded default configuration under `%APPDATA%`.
- **Optional Windows MSI** (WiX v6, `installer/windows/gitflowplus.wxs`)
  for Group Policy/SCCM/Intune deployment — per-machine, native
  `MajorUpgrade` handling, MSI-table-level-verified PATH management.
  Deliberately pinned to WiX v6, not v7, which gates its CLI behind a
  paid Open Source Maintenance Fee EULA.
- **Linux `.deb`/`.rpm`** via GoReleaser's `nfpm` integration, including
  a `git-flow` symlink so `git flow ...` resolves as a Git subcommand.
- **macOS `.pkg`** (`scripts/package-macos-pkg.sh`): a universal
  (Intel + Apple Silicon) binary via `lipo`, with a `postinstall` script
  seeding `~/Library/Application Support/GitFlowPlus`.
- **GoReleaser adoption** (`.goreleaser.yaml`): replaced the hand-
  maintained `scripts/build.sh`/`build.ps1` + `package.sh`/`package.ps1`
  pair with one declarative config — cross-compilation, archives,
  SHA256 checksums (previously missing entirely), and `.deb`/`.rpm` all
  from a single source of truth.
- **`git flow doctor` rewrite**: colorized output, plus new checks for
  Git version, repository write permissions, `PATH` resolution (whether
  `git flow ...` will actually work as a Git subcommand), Git Flow
  Plus's own build version, and whether a release is in progress.
- **`.github/workflows/release.yml` rewrite**: three coordinated jobs
  (GoReleaser on Linux; the Windows installer/MSI; the macOS `.pkg`),
  triggered by the same `v*` tag push as before, publishing everything
  to one GitHub Release.
- **LICENSE added** (MIT) — required for `.deb`/`.rpm` packaging
  metadata and now-correct for public distribution generally.
- **Module path corrected**: `go.mod` had declared
  `github.com/hulhub/git-flow-plus`, but the actual repository is
  `github.com/tabnaeem/git-flow-plus` — left uncorrected, this would
  have broken `go install` and GoReleaser's own GitHub Release step.
- **Docs**: new [Installation.md](Installation.md),
  [WindowsInstallation.md](WindowsInstallation.md),
  [Building.md](Building.md) (supersedes BuildGuide.md),
  [ReleaseProcess.md](ReleaseProcess.md), [Packaging.md](Packaging.md),
  [UpgradeGuide.md](UpgradeGuide.md), and
  [Troubleshooting.md](Troubleshooting.md). `InstallationGuide.md` and
  `BuildGuide.md` are now short pointers to their replacements.

### Feature Management architecture correction

`develop` is no longer part of the release lifecycle — it's a temporary
integration branch for unit testing only. This is a breaking change to the
feature workflow:

- **`feature start` now branches from `staging`**, not `develop`.
- **`feature finish` no longer exists.** A developer never merges their
  own feature branch — they commit, push, and open a pull request. Only a
  Release Manager decides if and when a feature ships, via `release
  feature add`.
- **`release feature add` now performs a real merge** of the feature
  branch into staging (previously bookkeeping-only), and is **safe to
  re-run**: a developer's QA follow-up commits, pushed to the
  still-alive branch, are pulled in by re-running the same command,
  without double-counting the feature in the version.
- **Feature branches (and release-fix/release-devops branches) are no
  longer deleted at merge time.** They stay alive through the whole QA
  cycle so follow-up fixes have somewhere to land, and are only deleted
  in bulk by `release finish`, once the release completes.
- **`release finish` no longer merges into `develop`** — only into `main`.
- **The Feature Registry gained an explicit state machine**
  (`Created`/`InDevelopment`/`AwaitingReview`/`Approved`/
  `IncludedInRelease`/`Released`/`Archived`), replacing the previous set
  of independent boolean flags.
- **`release feature remove` was removed** — once a merge is real, "un-
  merging" isn't a safe, generally-correct operation, so the command was
  dropped rather than given ill-defined semantics. `defer` still works,
  but only before the first `add`.
- **The version's 2nd field (`Release`) is now a live feature counter**,
  incrementing by one on every *new* feature merged in via `release
  feature add` (re-syncs don't increment it again) — independent of the
  release's own stable name (e.g. `"5.2"`), which is unaffected.
- `release.json` gained a `featureHistory` audit trail recording every
  `add` action, including resyncs.

See
[ReleaseManagement.md](ReleaseManagement.md#feature-management--release-planning)
for the full model.

### Production readiness

- **Logging**: seven-level structured logging (Trace, Debug, Info,
  Success, Warn, Error, Fatal) via a new colorized, human-readable console
  format (`[LEVEL] message  key: value`), alongside the existing
  newline-delimited JSON format for CI/CD. See
  `internal/logging`.
- **Errors**: a top-level CLI error boundary now renders every command
  failure as `[ERROR] <message>`, plus `Cause:`/`Resolution:` lines when
  the error opts in via `logging.AppError` (currently wired for malformed
  `config.json`). No stack traces are ever shown — Git Flow Plus's errors
  never carry one.
- **Configuration**: `config.json` gained an `environment`
  (development/testing/production) and a `logging` block (level, format,
  color, verbose, debug), with `GITFLOWPLUS_*` environment variable
  overrides and `--verbose`/`--debug`/`--json-log`/`--no-color` CLI flags
  taking final precedence.
- **`git flow version`**: prints Version, Build Number, Git Commit, Git
  Branch, Build Date, Go Version, OS, and Architecture — the first five
  embeddable at build time via `-ldflags` (see
  [BuildGuide.md](BuildGuide.md)).
- **Cross-platform release builds**: `scripts/build.sh` /
  `scripts\build.ps1` cross-compile Windows/Linux/macOS × amd64/arm64 (6
  targets) into `dist/`; `scripts/package.sh` / `scripts\package.ps1`
  package them into `dist/archives/` (zip for Windows, tar.gz for
  Linux/macOS). A `Makefile` wraps the common developer workflows
  (`build`, `test`, `lint`, `fmt`, `vet`, `check`, `dist`, `package`,
  `clean`).
- **CI/CD**: `.github/workflows/ci.yml` (build/vet/test matrix across
  Linux/macOS/Windows, a dedicated race-detector job, gofmt and
  golangci-lint checks) and `.github/workflows/release.yml` (on `v*` tag
  push: test, lint, cross-compile, package, attach to a GitHub Release).
- **Docs**: new [InstallationGuide.md](InstallationGuide.md),
  [BuildGuide.md](BuildGuide.md),
  [CrossPlatformGuide.md](CrossPlatformGuide.md), and this file.
- Removed unused empty scaffolding directories (`pkg/`, `docs/`,
  `templates/`, `tests/`, `configs/`) left over from initial project
  setup and never populated.
- Extracted a shared `checkoutStaging`/`requireActiveRelease` helper pair
  in `internal/release/service.go`, replacing seven near-identical
  checkout-and-guard preambles that had accumulated across
  `featureplanning.go`, `fix.go`, `finish.go`, and `start.go`.

### Feature Management & Release Planning

- A permanent Feature Registry (`internal/feature`,
  `.gitflowplus/features.json`) tracking every feature's lifecycle
  (merged into develop → unit tested → approved → shipped), independent
  of any single release cycle.
- `git flow feature finish` auto-registers the feature as merged.
- New `git flow release feature {list,add,remove,approve,defer,status}`
  commands implementing explicit Release Planning: a feature never joins
  a release just because it exists — a Release Manager has to decide.
- `release.json`'s schema restructured to nested `features`/
  `releaseFixes`/`devops` objects, each with `included`/`pending` (and
  `features` additionally `deferred`) — see
  [ReleaseManagement.md](ReleaseManagement.md#feature-management--release-planning).

### Release management core

- Three permanent branches: `main` (production), `staging` (the release
  line), `develop` (feature development and unit testing only).
- `release start`/`build`/`finish`/`status`/`version`/`manifest`,
  `releasefix`, `devops` — merging a release-fix or DevOps branch never
  moves the version; only `release build` does.
- Mandatory annotated Git tags for every QA build and every production
  release, with structured, self-describing tag messages (Release,
  Version, QA Build, Features, Release Fixes, DevOps, Branch, Commit,
  Release Manager, Release Date).
- CI/CD-agnostic lifecycle hooks (`post-qa-tag`, `post-production-tag`)
  via `internal/hooks`.

### Foundation

- Standard Git Flow (`init`, `feature`, `hotfix`, `support`) on top of a
  thin `internal/git` wrapper over the real `git` binary — every operation
  shells out, so behavior matches plain Git exactly.
- `doctor` health checks, `config` inspection.
- Full unit-test and real end-to-end test coverage (every package that
  touches Git is tested against the actual binary, not just fakes).

## How to read this file going forward

Once the first tag is pushed, this file's "Unreleased" section should be
renamed to that version (e.g. `## v1.0.0 — 2026-08-01`) and a fresh empty
`## Unreleased` section started above it — standard
[Keep a Changelog](https://keepachangelog.com/)-style practice, not
enforced by tooling here but the intended convention.
