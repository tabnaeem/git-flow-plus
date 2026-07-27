# Roadmap

What's implemented, what's deliberately deferred, and what's next. This is
a snapshot, not a promise — see `git log` for what actually shipped and
when.

## Implemented

- Standard Git Flow: `init`, `feature`, `hotfix`, `support`.
- Three permanent branches: `main` (production), `staging` (the release
  line), `develop` (feature development + unit testing only).
- Release management: `release start`/`build`/`finish`/`status`/`version`/
  `manifest`, `releasefix`, `devops` — merging never moves the version,
  only `release build` does.
- Mandatory annotated tagging for every QA build and every production
  release, with structured, self-describing tag messages.
- CI/CD-agnostic lifecycle hooks (`post-qa-tag`, `post-production-tag`).
- **Feature Management & Release Planning**: a permanent Feature Registry
  (`internal/feature`, `.gitflowplus/features.json`), `feature finish`
  auto-registration, and `release feature {list,add,remove,approve,defer,status}`
  for explicit, PM-driven release composition. See
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
- **Cross-platform release pipeline**: `scripts/build.sh`/`build.ps1` and
  `package.sh`/`package.ps1` producing Windows/Linux/macOS ×
  amd64/arm64 binaries and zip/tar.gz archives; a `Makefile` for the
  common developer workflows; `.github/workflows/ci.yml` (build/vet/test
  matrix, race detector, gofmt, lint) and `release.yml` (cross-compile,
  package, attach to GitHub Releases on a `v*` tag push). See
  [BuildGuide.md](BuildGuide.md).

## Deliberately out of scope (for now)

These aren't oversights — each was a conscious call to keep Git Flow Plus
a thin, predictable layer over Git rather than a service with its own
consistency model or a black box that moves code on your behalf.

- **No automated feature-code promotion onto staging.** Release Planning
  (`release feature add`/`defer`) is bookkeeping only. Git Flow Plus never
  cherry-picks or auto-merges a feature branch into `staging`; the code
  has to already be reachable there by whatever process your team uses.
  Automating this safely (conflict handling, partial-merge recovery)
  is a meaningfully different and riskier feature than anything else in
  the tool today.
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
2. **Package-manager installers**: Homebrew, Chocolatey, Scoop, `.deb`,
   `.rpm`, and an MSI, so installing is a one-line command rather than
   downloading and extracting an archive (see
   [InstallationGuide.md](InstallationGuide.md#future-packaging-not-yet-available)).
3. **Optional feature-promotion helper**: a strictly opt-in command (e.g.
   `release feature promote <id>`) that performs a supervised merge of a
   feature branch into `staging` — kept separate from the planning
   commands above so the default workflow stays merge-free.
4. **Code signing**: the release archives are unsigned, so macOS
   Gatekeeper and Windows SmartScreen may warn on first run (see
   [InstallationGuide.md](InstallationGuide.md)) — signing/notarization
   would remove that friction.

## How to propose a change

Read [Architecture.md](Architecture.md) and
[DeveloperGuide.md](DeveloperGuide.md) first — most extensions fit an
existing pattern (a new `Service` method, a fake-based + real-repo test
pair, a `CommandReference.md` entry) rather than needing new
infrastructure.
