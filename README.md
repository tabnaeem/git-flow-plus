# Git Flow Plus

Git Flow Plus extends the standard [Git Flow](https://nvie.com/posts/a-successful-git-branching-model/)
branching model with enterprise release management, built on three
permanent branches:

- **`develop`** — ongoing feature development and unit testing. Never the
  release line.
- **`staging`** — the release line. Release fixes and DevOps changes merge
  here; QA builds are cut here; this is where a release actually lives
  while it's being stabilized.
- **`main`** — production. A release lands here once, at Production Release
  time.

It adds no new Git concepts — every workflow is a normal Git branch and
every release a normal annotated tag, so the repository stays fully
compatible with GitHub, GitLab, Azure DevOps, Bitbucket, SourceTree,
GitKraken, and any Git-aware IDE or CI/CD system.

## Installing

Download a prebuilt binary for your platform from GitHub Releases, or build
from source — see [InstallationGuide.md](InstallationGuide.md). Quick
version:

```bash
go build -o bin/git-flow ./cmd/git-flow-plus
```

## Testing

```bash
go test ./... -cover
```

## Usage

```bash
git-flow --help
git-flow config list
git-flow version
```

To invoke it as `git flow ...`, put the built binary on your `PATH` as
`git-flow` (Git resolves `git <subcommand>` to an executable named
`git-<subcommand>`).

Every command logs structured diagnostics to stderr — colorized,
human-readable text by default (`[INFO] message  key: value`), or
newline-delimited JSON with `--json-log` for CI/CD. `--verbose`/`--debug`
raise the log level; `--no-color` disables ANSI colors. Defaults for all of
this live in `config.json`'s `logging` block and can be overridden per
environment (development/testing/production) or via `GITFLOWPLUS_*`
environment variables — see
[DeveloperGuide.md](DeveloperGuide.md#logging--configuration).

## Documentation

- **[Architecture.md](Architecture.md)** — package structure, dependency
  graph, and the design decisions behind them.
- **[ReleaseManagement.md](ReleaseManagement.md)** — the full release
  workflow: why merging a fix doesn't change the version, how QA builds and
  tagging work, Feature Management/Release Planning, and the lifecycle
  hooks that drive CI/CD.
- **[CommandReference.md](CommandReference.md)** — every command, its
  syntax, and what it does.
- **[DeveloperGuide.md](DeveloperGuide.md)** — building, testing, and
  extending the codebase.
- **[InstallationGuide.md](InstallationGuide.md)** — installing a release
  binary or building from source.
- **[BuildGuide.md](BuildGuide.md)** — cross-compiling, packaging, version
  metadata, and the CI/CD release pipeline.
- **[CrossPlatformGuide.md](CrossPlatformGuide.md)** — supported platforms
  and the handful of places platform differences actually matter.
- **[Roadmap.md](Roadmap.md)** — what's implemented, what's deliberately
  deferred, and what's next.
- **[ReleaseNotes.md](ReleaseNotes.md)** — what's changed, by milestone.

## The short version

```bash
git flow init                          # main, staging, develop

git flow feature start LOGIN           # feature/LOGIN, from develop
# ...develop, commit...
git flow feature finish LOGIN          # merges into develop; registers LOGIN in the Feature Registry

git flow release start 5.2             # staging: version.json = 5.2.0.0.1, tags v5.2.0.0.1

git flow release feature approve LOGIN # LOGIN is unit-tested and eligible for Release Planning
git flow release feature add LOGIN     # Release Planning decision: LOGIN ships in 5.2

git flow releasefix start BUG-101      # release-fix/BUG-101, from staging
# ...fix, commit...
git flow releasefix finish BUG-101     # merges into staging; version UNCHANGED, fix recorded as pending

git flow release build                 # only this changes the version: 5.2.1.0.2, tags v5.2.1.0.2

git flow release finish 5.2            # staging -> main -> develop, tags v5.2 (production)
                                        # LOGIN is now permanently recorded as shipped in 5.2
```

The version only moves when a Release Manager runs `release build` —
merging release-fix/DevOps branches never does. Similarly, a feature never
joins a release just because it was merged — a Release Manager has to
explicitly add it via `release feature add`. See
[ReleaseManagement.md](ReleaseManagement.md) for the full model and why.

## Project status

Every command in the original spec is implemented and tested (unit tests
plus real end-to-end tests against the actual `git` binary), including
Feature Management and Release Planning (`internal/feature`, `git flow
release feature ...`), production-grade structured logging, environment-
aware configuration, and a full cross-platform build/release pipeline
(`make dist`/`make package`, `.github/workflows/release.yml`) producing
binaries for Windows, Linux, and macOS on both amd64 and arm64. Not yet
built: multi-release support (starting a second release while one is
active is rejected outright, not disambiguated), automated promotion of
feature code from `develop` onto `staging` (Release Planning is
bookkeeping-only by design — see
[ReleaseManagement.md](ReleaseManagement.md#feature-management--release-planning)),
and package-manager installers (Homebrew, Chocolatey, Scoop, `.deb`,
`.rpm`, MSI). See [Roadmap.md](Roadmap.md) for the full picture.
