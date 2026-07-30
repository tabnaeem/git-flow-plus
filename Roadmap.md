# Roadmap

What's implemented, what's deliberately deferred, and what's next. This is
a snapshot, not a promise — see `git log` for what actually shipped and
when.

## Implemented

- Standard Git Flow: `init`, `hotfix`, `support`, and `feature start`
  (there is no `feature finish` — see below).
- Three permanent branches: `main` (production), `staging` (the release
  line — the release lifecycle, and feature branches, begin here),
  `develop` (a temporary integration branch for unit testing, outside the
  release lifecycle entirely).
- Release management: `release start`/`build`/`finish`/`status`/`version`/
  `manifest`, `releasefix`, `devops` — merging a release-fix/DevOps
  branch never moves the version, only `release build` does; the branch
  stays alive until `release finish`.
- Mandatory annotated tagging for every QA build and every production
  release, with structured, self-describing tag messages.
- CI/CD-agnostic lifecycle hooks (`post-qa-tag`, `post-production-tag`).
- **Feature Management & Release Planning**: a permanent Feature Registry
  (`internal/feature`, `.gitflowplus/features.json`) with an explicit
  lifecycle (`Created`/`Approved`/`IncludedInRelease`/`Released`/...),
  `feature start` auto-registration, and `release feature
  {list,add,approve,defer,status}` — `add` performs a **real merge** of
  the feature branch into staging (not bookkeeping), is safely re-runnable
  to pull in QA follow-up commits, and only `release finish` deletes the
  branch. See
  [ReleaseManagement.md](ReleaseManagement.md#feature-management--release-planning).
- `doctor` health checks, `config` inspection.
- Full unit + real end-to-end test coverage (every package that touches
  Git is tested against the actual `git` binary, not just fakes) and a
  clean `golangci-lint run ./...`.
- **Production-grade logging**: seven log levels (Trace through Fatal), a
  colorized human-readable console format alongside JSON for CI/CD, and
  `Cause`/`Resolution` error formatting at the CLI boundary. See
  [DeveloperGuide.md](DeveloperGuide.md#logging--configuration).
- **Environment-aware configuration**: `development`/`testing`/
  `production` environments seeding logging defaults, with
  `GITFLOWPLUS_*` environment variable overrides and CLI flags
  (`--verbose`/`--debug`/`--json-log`/`--no-color`) taking final
  precedence.
- **`git flow version`** with build metadata embeddable via `-ldflags`.
- **Cross-platform release pipeline**: GoReleaser (`.goreleaser.yaml`)
  cross-compiles Windows/Linux/macOS × amd64/arm64, archives (zip for
  Windows, tar.gz elsewhere), checksums, and packages `.deb`/`.rpm` for
  Linux, from one declarative config instead of parallel bash/PowerShell
  scripts; a `Makefile` for the common developer workflows;
  `.github/workflows/ci.yml` (build/vet/test matrix, race detector,
  gofmt, lint) and `release.yml` (test, lint, `goreleaser release`, plus
  dedicated Windows-installer and macOS-`.pkg` jobs, all attached to a
  GitHub Release on a `v*` tag push). See [Building.md](Building.md).
- **Windows installer** (Inno Setup, optional WiX MSI), Linux
  `.deb`/`.rpm` (via GoReleaser's `nfpm` integration), and a macOS
  `.pkg` — silent install/uninstall, PATH registration (including the
  `git-flow` alias `git flow ...` needs to resolve as a Git subcommand),
  upgrade/reinstall detection, and Git/Git Bash/PowerShell detection on
  Windows. See [WindowsInstallation.md](WindowsInstallation.md) and
  [Packaging.md](Packaging.md).

## Deliberately out of scope (for now)

These aren't oversights — each was a conscious call to keep Git Flow Plus
a thin, predictable layer over Git rather than a service with its own
consistency model or a black box that moves code on your behalf.

- **No way to un-merge a feature.** There is deliberately no `release
  feature remove` — once `add` has performed its real merge, reverting it
  safely and generally isn't something Git Flow Plus attempts. `defer`
  covers the "hold this back" case, but only before the first `add`.
- **No automatic detection of follow-up commits.** A developer can push a
  QA-fix commit to an already-merged feature branch at any time; Git Flow
  Plus doesn't watch for it or pull it in automatically. The Release
  Manager has to notice (via their normal QA process) and re-run `release
  feature add` — see
  [ReleaseManagement.md](ReleaseManagement.md#what-release-planning-does-now).
- **One release at a time.** `release start` rejects a second concurrent
  release outright (`ErrReleaseAlreadyActive`) rather than trying to
  disambiguate which release a command targets.
- **No cross-process locking.** Two operators running commands against the
  same repo concurrently race exactly as if they'd run `git` by hand — this
  matches Git's own concurrency model and is intentional, not a gap to close.

## Candidate next steps

Roughly in the order they'd likely be tackled, none currently scheduled:

1. **Multi-release support**: allow more than one release cycle active at
   once (e.g. a `5.3` bugfix line and a `5.4` feature line in parallel),
   which would require `release.json` to stop being a single file keyed
   implicitly by "whatever's on staging" and instead support multiple
   named manifests.
2. **Package-manager installers**: Homebrew, Chocolatey, and Scoop, so
   installing is a one-line command rather than downloading an archive
   or running an installer — `.deb`, `.rpm`, and an optional Windows MSI
   are already implemented (see [Packaging.md](Packaging.md)).
3. **Follow-up-commit notification**: some way to surface "this
   already-merged feature branch has new commits" (e.g. as part of
   `release feature status` or `doctor`) so a Release Manager doesn't have
   to rely purely on their own QA process to know a re-run of `release
   feature add` is needed.
4. **Code signing**: the release archives and installers are unsigned,
   so macOS Gatekeeper and Windows SmartScreen may warn on first run
   (see [Packaging.md](Packaging.md#unsigned-artifacts)) — signing/
   notarization would remove that friction.

## How to propose a change

Read [Architecture.md](Architecture.md) and
[DeveloperGuide.md](DeveloperGuide.md) first — most extensions fit an
existing pattern (a new `Service` method, a fake-based + real-repo test
pair, a `CommandReference.md` entry) rather than needing new
infrastructure.
