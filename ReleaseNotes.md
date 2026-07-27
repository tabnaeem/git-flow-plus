# Release Notes

No version has been tagged yet — `git flow version` reports `0.1.0-dev`
until the first `v*` tag is pushed (see
[BuildGuide.md](BuildGuide.md#cicd)). This document tracks what's landed
so far, organized by milestone, so the first real release has an accurate
starting changelog rather than a single undifferentiated diff.

## Unreleased

### Production readiness (this milestone)

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
