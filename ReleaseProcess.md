# Release Process

How a Git Flow Plus maintainer cuts a release. This is about releasing
**Git Flow Plus itself** (the tool) — for the release lifecycle Git Flow
Plus manages *for a project that uses it*, see
[ReleaseManagement.md](ReleaseManagement.md).

## The short version

```bash
git tag v5.3.4.1.2
git push origin v5.3.4.1.2
```

Pushing a tag matching `v*` is the entire trigger. Everything else —
testing, linting, cross-compiling, packaging, checksumming, SBOM
generation, changelog generation, **building the Windows installer**,
publishing the GitHub Release, and attaching the macOS `.pkg` — happens
automatically in `.github/workflows/release.yml`. There is no separate
manual packaging step, and no manual copying of the installer into
place; if the tag builds and publishes, the release is done.

## What the tag should look like

Any `v*` tag triggers the pipeline, but tag content flows directly into
build artifacts: it becomes `cli.Version` (via `-ldflags`), the Windows
installer's `DisplayVersion` and embedded `VIProductVersion`, and the
GitHub Release's title — see
[Packaging.md#versioning](Packaging.md#versioning) for the exact
mapping, including the 5-field-to-4-field Windows truncation rule.
Use `vSprint.Feature.ReleaseFix.DevOps.QA` (Git Flow Plus's own version
format, e.g. `v5.3.4.1.2`) once the project is versioning itself that
way; a plain `vMAJOR.MINOR.PATCH` semver tag also works during any
pre-1.0 period — the pipeline doesn't require a specific field count,
only the leading `v`.

## Before the pipeline runs at all: the tag-exists guard

The `goreleaser` job's first real step checks
`gh release view "$GITHUB_REF_NAME"` and fails the run immediately
(`::error::...`) if a release for that tag already exists, rather than
letting GoReleaser attempt to publish over it. GitHub Releases are
treated as **immutable** here: neither this guard nor the Windows/macOS
upload steps ever use `gh release upload --clobber`, so re-running a job
against an already-published release fails loudly instead of silently
overwriting a published artifact. A genuinely bad release is fixed by
yanking it and pushing a new tag, not by mutating one that already
shipped.

## The pipeline, job by job

```mermaid
graph LR
    TAG["git push origin v5.3.4.1.2"] --> GR["goreleaser job (windows-latest)"]
    GR -->|"release now exists"| MAC["macos-pkg job"]
    GR --> R0["tag-exists guard"]
    R0 --> R1["test + lint"]
    R1 --> R2["install NSIS + syft"]
    R2 --> R3["cross-compile all 6 platforms + embed icon/version"]
    R3 --> R4["build hook: windows/amd64 -> NSIS installer"]
    R4 --> R5["archives + checksums.txt (incl. installer) + deb/rpm + SBOM"]
    R5 --> R6["changelog + GitHub Release created (incl. installer via extra_files)"]
    R6 --> R7["verify installer exists + attest build provenance"]
    MAC --> M1["build darwin/amd64+arm64"]
    M1 --> M2["lipo universal binary"]
    M2 --> M3["pkgbuild + productbuild -> .pkg"]
    M3 --> M4["checksums + gh release upload"]
```

1. **`goreleaser`** (**windows-latest**) — after the tag-exists guard
   above, runs `go test ./...` and `golangci-lint run`, installs NSIS
   and `syft`, then `goreleaser release --clean`. This **one command**
   cross-compiles all 6 `GOOS`/`GOARCH` targets, embeds the icon/version
   resource, builds the NSIS installer as soon as the `windows/amd64`
   binary exists (via the `builds[].hooks.post` described in
   [Packaging.md#wired-into-goreleaser-hookspost-and-extra_files](Packaging.md#wired-into-goreleaser-hookspost-and-extra_files)),
   archives everything, writes `checksums.txt` (including the
   installer), generates one SBOM per archive, builds `.deb`/`.rpm`,
   generates a changelog from commit history, and **creates the GitHub
   Release** with the installer attached via `extra_files` — all as one
   atomic operation. A dedicated step afterward double-checks the
   installer landed in `dist/installer/` (belt-and-suspenders: the build
   hook failing would already have aborted the run before this point),
   then a final step attests build provenance
   (`actions/attest-build-provenance@v2`) for every archive, package,
   SBOM, **and the installer** — see
   [Build provenance attestation](#build-provenance-attestation) below.
   The `macos-pkg` job depends on this release already existing, since
   it attaches to it rather than creating its own.
2. **`macos-pkg`** (macos-latest, `needs: goreleaser`) — rebuilds the
   darwin binaries with matching `-ldflags`, runs
   [scripts/package-macos-pkg.sh](scripts/package-macos-pkg.sh) to
   produce a universal `.pkg`, checksums it, and `gh release upload`s it
   to the release the first job created.

Only macOS runs as a **separate job**: `pkgbuild`/`productbuild`/`lipo`
only exist on macOS, and have nothing to do with GoReleaser's own
pipeline, so packaging the `.pkg` has to happen after the fact once the
release exists. The Windows installer doesn't need this treatment —
Go's cross-compilation is host-independent (a Windows host cross-
compiles the linux/darwin targets exactly as easily as a Linux host
cross-compiles windows targets), and `nfpm`'s `.deb`/`.rpm` packaging is
pure Go with no OS-native tool dependency either. That symmetry is what
lets the `goreleaser` job simply run on `windows-latest` instead of
`ubuntu-latest`, folding NSIS packaging into the same run instead of
needing its own follow-up job.

## Build provenance attestation

The `goreleaser` job's final step runs
`actions/attest-build-provenance@v2` over every archive, `.deb`/`.rpm`,
SBOM, **and the Windows installer** it produced (`permissions: id-token:
write, attestations: write` at the workflow level enable this). This
creates a verifiable, signed record — hosted by GitHub, not this
repository — of exactly which workflow run, commit, and source
repository produced a given artifact. Anyone can verify a downloaded
artifact wasn't tampered with or substituted after the fact:

```bash
gh attestation verify git-flow-plus-linux-amd64.tar.gz --repo tabnaeem/git-flow-plus
gh attestation verify GitFlowPlusSetup_v5.3.4.1.2_x64.exe --repo tabnaeem/git-flow-plus
```

The macOS `.pkg` is built in its own, separate job (see above) and is
not currently covered by this attestation step — a candidate follow-up,
not a gap in the core archives/packages/installer GoReleaser itself
produces.

## Verifying a release after it publishes

```bash
curl -sL https://github.com/tabnaeem/git-flow-plus/releases/download/v5.3.4.1.2/checksums.txt -o checksums.txt
sha256sum -c checksums.txt --ignore-missing
```

Then smoke-test at least the platform you're on:

```bash
git flow version   # confirms the tag's version made it into the binary
git flow doctor
```

## Before tagging

Everything the release pipeline runs, run locally first — see
[Building.md#local-verification-before-tagging](Building.md#local-verification-before-tagging):

```bash
make check    # fmt, vet, lint, test
make package  # goreleaser --snapshot: the exact same steps, without publishing
```

## After a release: updating ReleaseNotes.md

[ReleaseNotes.md](ReleaseNotes.md) is not auto-generated by the
pipeline (GoReleaser's own changelog goes into the GitHub Release body,
not this file) — rename its `## Unreleased` heading to the version just
tagged (e.g. `## v5.3.4.1.2 — 2026-08-01`) and start a fresh empty
`## Unreleased` section above it, per
[Keep a Changelog](https://keepachangelog.com/) convention.

## See also

- [Building.md](Building.md) — the underlying build commands
- [Packaging.md](Packaging.md) — how each artifact is assembled
- [UpgradeGuide.md](UpgradeGuide.md) — what a release means for existing installs
