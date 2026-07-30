# Building

How Git Flow Plus itself is built and version-stamped. For day-to-day
development (running tests, adding a command), see
[DeveloperGuide.md](DeveloperGuide.md) — this document is specifically
about producing distributable binaries and installers. For how a
release actually gets cut and published, see
[ReleaseProcess.md](ReleaseProcess.md); for how each platform's
installer/package is assembled, see [Packaging.md](Packaging.md).

## Quick reference

| Command | What it does |
|---|---|
| `go build -o bin/git-flow ./cmd/git-flow-plus` | Plain build, no version metadata, host platform only. |
| `make build` | Same, but embeds real version metadata via `-ldflags`. |
| `make dist` | `goreleaser build --snapshot --clean` — cross-compiles all 6 platforms into `dist/`, no tag or token required. |
| `make package` | `goreleaser release --snapshot --clean` — `dist`, plus archives, checksums, and `.deb`/`.rpm`, still without publishing anything. |
| `make check` | `fmt` + `vet` + `lint` + `test` — what CI runs. |

`dist`/`package` require [GoReleaser](https://goreleaser.com) on `PATH`
(`go install github.com/goreleaser/goreleaser/v2@latest`, or see their
install docs for a prebuilt binary). This one tool works identically on
Windows, Linux, and macOS — there's no separate PowerShell/bash script
pair to maintain, unlike the hand-rolled `scripts/build.sh`/`build.ps1`
this replaced.

## Version metadata

`git flow version` prints Version, Build Number, Git Commit, Git Branch,
Build Date, Go Version, OS, and Architecture (see
[CommandReference.md](CommandReference.md#git-flow-version)). The first
five are build-time constants in `internal/cli/version.go` (`Version`
lives in `internal/cli/root.go` alongside it), overridden via `-ldflags
"-X <package>.<Var>=<value>"`:

```
-X github.com/tabnaeem/git-flow-plus/internal/cli.Version=5.3.4.1.2
-X github.com/tabnaeem/git-flow-plus/internal/cli.BuildNumber=456
-X github.com/tabnaeem/git-flow-plus/internal/cli.GitCommit=a1b2c3d
-X github.com/tabnaeem/git-flow-plus/internal/cli.GitBranch=main
-X github.com/tabnaeem/git-flow-plus/internal/cli.BuildDate=2026-07-27T12:00:00Z
```

A plain `go build` (no `-ldflags`) leaves these at their zero-value
defaults — `"dev"`/`"unknown"` — which is how you can tell an unstamped
development build apart from a real one. `.goreleaser.yaml`'s `builds`
section sets all five automatically from the pushed tag (`{{ .Version }}`,
with the leading `v` stripped), `BUILD_NUMBER` (from CI's
`github.run_number`, defaulting to `"dev"` for local snapshot builds),
and Git's own commit/branch/date. Go Version, OS, and Architecture are
never build-time constants — they're read at runtime via
`runtime.Version()`/`runtime.GOOS`/`runtime.GOARCH`, so they're always
accurate regardless of how the binary was built.

## Cross-compilation

Go's cross-compilation is native — no Docker, no toolchain installation
per target, just `GOOS`/`GOARCH` environment variables:

```bash
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o git-flow-plus-linux-arm64 ./cmd/git-flow-plus
```

`.goreleaser.yaml` does exactly this for all six supported targets (see
[CrossPlatformGuide.md](CrossPlatformGuide.md) for the full list and
platform-specific notes). `CGO_ENABLED=0` is set explicitly — Git Flow
Plus has no C dependencies, and disabling cgo is what makes the binaries
fully static and reliably cross-compilable without a target-platform C
toolchain. `-trimpath` strips local filesystem paths from the compiled
binary, and `-s -w` strip the symbol table and DWARF debug info to
shrink the binary — standard release-build flags, not used for local
development builds where you'd want debug info intact.

## Installer toolchains

Building the platform installers described in
[Packaging.md](Packaging.md) needs their native tools, which only run on
their own OS:

| Tool | Platform | Install |
|---|---|---|
| NSIS 3 (`makensis`) | Windows | `winget install NSIS.NSIS` or `choco install nsis` — installed explicitly in `windows-installer`'s CI job, not preinstalled on GitHub's `windows-latest` runner |
| `go-winres` | any | `go install github.com/tc-hib/go-winres@latest` — pure Go, embeds the executable icon/version resource before the Windows build (see [Packaging.md#executable-icon-and-version-resource](Packaging.md#executable-icon-and-version-resource)) |
| `lipo`/`pkgbuild`/`productbuild` | macOS | Part of Xcode Command Line Tools, preinstalled on GitHub's `macos-latest` runner |
| `nfpm` (`.deb`/`.rpm`) | any | Bundled into GoReleaser itself — no separate install |
| `syft` (SBOM generation) | any | Required on `PATH` for `goreleaser release`'s `sboms:` step; CI installs it via `anchore/sbom-action/download-syft@v0` — see [Packaging.md#release-integrity-sbom-and-reproducible-builds](Packaging.md#release-integrity-sbom-and-reproducible-builds) |

## CI/CD

- **`.github/workflows/ci.yml`** runs on every push to `main`/`develop`/
  `staging` and every pull request: build + vet + test on a
  Linux/macOS/Windows matrix, a dedicated `-race` job (Linux only — the
  race detector needs cgo, which release builds deliberately disable), a
  `gofmt -l` check, and `golangci-lint run`.
- **`.github/workflows/release.yml`** runs when a tag matching `v*` is
  pushed — see [ReleaseProcess.md](ReleaseProcess.md) for the full
  three-job pipeline (GoReleaser, Windows installer, macOS `.pkg`).

## Local verification before tagging

```bash
make check    # fmt, vet, lint, test
make package  # full goreleaser snapshot: binaries, archives, checksums, deb/rpm
```

Then smoke-test at least one produced binary directly (`dist/` isn't
committed — see `.gitignore` — so this is a manual, pre-release step):

```bash
./dist/git-flow-plus_<os>_<arch>*/git-flow-plus version
./dist/git-flow-plus_<os>_<arch>*/git-flow-plus doctor
```

To also test-build the Windows installer or macOS `.pkg` locally, see
their respective commands in
[Packaging.md](Packaging.md#building-locally).
