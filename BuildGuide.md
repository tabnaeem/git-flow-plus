# Build Guide

How Git Flow Plus itself is built, versioned, and released. For day-to-day
development (running tests, adding a command), see
[DeveloperGuide.md](DeveloperGuide.md) — this document is specifically about
producing distributable binaries.

## Quick reference

| Command | What it does |
|---|---|
| `go build -o bin/git-flow ./cmd/git-flow-plus` | Plain build, no version metadata, host platform only. |
| `make build` | Same, but embeds real version metadata via `-ldflags`. |
| `make dist` / `./scripts/build.sh` / `scripts\build.ps1` | Cross-compiles all 6 supported platforms into `dist/`. |
| `make package` / `./scripts/package.sh` / `scripts\package.ps1` | Runs `dist`, then packages `dist/` into `dist/archives/` (zip for Windows, tar.gz for Linux/macOS). |
| `make check` | `fmt` + `vet` + `lint` + `test` — what CI runs. |

Windows without a `make` on `PATH`: use `scripts\build.ps1` and
`scripts\package.ps1` directly — they're equivalent to the `.sh` versions,
just written for PowerShell instead of bash.

## Version metadata

`git flow version` prints Version, Build Number, Git Commit, Git Branch,
Build Date, Go Version, OS, and Architecture (see
[CommandReference.md](CommandReference.md#git-flow-version)). The first five
are build-time constants in `internal/cli/version.go` (`Version` lives in
`internal/cli/root.go` alongside it), overridden via `-ldflags "-X
<package>.<Var>=<value>"`:

```
-X github.com/hulhub/git-flow-plus/internal/cli.Version=1.2.0
-X github.com/hulhub/git-flow-plus/internal/cli.BuildNumber=456
-X github.com/hulhub/git-flow-plus/internal/cli.GitCommit=a1b2c3d
-X github.com/hulhub/git-flow-plus/internal/cli.GitBranch=main
-X github.com/hulhub/git-flow-plus/internal/cli.BuildDate=2026-07-27T12:00:00Z
```

A plain `go build` (no `-ldflags`) leaves these at their zero-value
defaults — `"dev"`/`"unknown"` — which is how you can tell an unstamped
development build apart from a real one. `scripts/build.sh` and
`scripts\build.ps1` set all five automatically from `git describe`/`git
rev-parse` (or from `VERSION`/`BUILD_NUMBER`/`GIT_COMMIT`/`GIT_BRANCH`/
`BUILD_DATE` environment variables, if set — this is how CI supplies the
real release version instead of a git-derived guess). Go Version, OS, and
Architecture are never build-time constants — they're read at runtime via
`runtime.Version()`/`runtime.GOOS`/`runtime.GOARCH`, so they're always
accurate regardless of how the binary was built.

## Cross-compilation

Go's cross-compilation is native — no Docker, no toolchain installation
per target, just `GOOS`/`GOARCH` environment variables:

```bash
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o git-flow-plus-linux-arm64 ./cmd/git-flow-plus
```

`scripts/build.sh` / `scripts\build.ps1` do exactly this for all six
supported targets (see [CrossPlatformGuide.md](CrossPlatformGuide.md) for
the full list and platform-specific notes). `CGO_ENABLED=0` is set
explicitly — Git Flow Plus has no C dependencies, and disabling cgo is what
makes the binaries fully static and reliably cross-compilable without a
target-platform C toolchain.

`-trimpath` strips local filesystem paths from the compiled binary (so
`/home/you/git-flow-plus/...` doesn't leak into stack traces or debug
info), and `-s -w` strip the symbol table and DWARF debug info to shrink
the binary — standard flags for a release build, not used for local
development builds where you'd want debug info intact.

## Packaging

`scripts/package.sh` / `scripts\package.ps1` take the binaries in `dist/`
and produce one archive per platform in `dist/archives/`: `.zip` for
Windows (the platform convention — `Compress-Archive`/`zip` both produce
standard zip files any Windows user can double-click), `.tar.gz` for
Linux/macOS (via `tar`, present on every supported platform including
modern Windows). Each archive also bundles `README.md` so a download is
self-explanatory without needing the source repository.

## CI/CD

- **`.github/workflows/ci.yml`** runs on every push to `main`/`develop`/
  `staging` and every pull request: build + vet + test on a
  Linux/macOS/Windows matrix, a dedicated `-race` job (Linux only — the
  race detector needs cgo, which the release builds deliberately disable),
  a `gofmt -l` check, and `golangci-lint run`.
- **`.github/workflows/release.yml`** runs when a tag matching `v*` is
  pushed: re-runs tests and lint (so a bad release can't slip through even
  if CI was somehow bypassed), cross-compiles all six platforms, packages
  them, and attaches the archives to a GitHub Release for that tag via
  `softprops/action-gh-release`.

To cut a release: tag and push.

```bash
git tag v1.2.0
git push origin v1.2.0
```

`release.yml` picks up `github.ref_name` (the tag, e.g. `v1.2.0`) as
`VERSION` and `github.run_number` as `BUILD_NUMBER` automatically.

## Local verification before tagging

```bash
make check    # fmt, vet, lint, test
make package  # full dist + archive pipeline, same as CI's release job
```

Then smoke-test at least one produced binary directly (`dist/` isn't
committed — see `.gitignore` — so this is a manual, pre-release step, not
something CI redundantly re-verifies beyond what `release.yml` already
does):

```bash
./dist/git-flow-plus-<your-platform> version
./dist/git-flow-plus-<your-platform> doctor
```
