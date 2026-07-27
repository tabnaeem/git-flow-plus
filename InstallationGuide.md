# Installation Guide

Git Flow Plus ships as a single, dependency-free binary named `git-flow-plus`
(or `git-flow`, either name works — see [Invoking it as `git flow
...`](#invoking-it-as-git-flow-)). No runtime, no interpreter, nothing else
to install.

## Option 1: Download a release archive

Every tagged release publishes six platform archives to GitHub Releases
(see [BuildGuide.md](BuildGuide.md) for how they're built):

| Platform | Architecture | Archive |
|---|---|---|
| Windows | x64 | `git-flow-plus-windows-amd64.zip` |
| Windows | ARM64 | `git-flow-plus-windows-arm64.zip` |
| Linux | x64 | `git-flow-plus-linux-amd64.tar.gz` |
| Linux | ARM64 | `git-flow-plus-linux-arm64.tar.gz` |
| macOS | Intel | `git-flow-plus-darwin-amd64.tar.gz` |
| macOS | Apple Silicon | `git-flow-plus-darwin-arm64.tar.gz` |

Each archive contains the binary and a copy of README.md — nothing else to
extract.

### Windows

```powershell
Expand-Archive git-flow-plus-windows-amd64.zip -DestinationPath C:\Tools\git-flow-plus
```

Add `C:\Tools\git-flow-plus` to your `PATH` (System Properties → Environment
Variables, or `setx PATH "$env:PATH;C:\Tools\git-flow-plus"` in an elevated
shell).

### Linux / macOS

```bash
tar -xzf git-flow-plus-linux-amd64.tar.gz -C /usr/local/bin --strip-components=0 git-flow-plus
chmod +x /usr/local/bin/git-flow-plus
```

(Substitute the archive matching your platform; macOS users on Apple Silicon
should grab `-darwin-arm64`, Intel Macs `-darwin-amd64`.)

macOS Gatekeeper may quarantine an unsigned downloaded binary. If running it
is refused, clear the quarantine attribute:

```bash
xattr -d com.apple.quarantine /usr/local/bin/git-flow-plus
```

## Option 2: Build from source

Requires Go (see `go.mod` for the minimum version) and `git`:

```bash
git clone <this repository>
cd git-flow-plus
go build -o bin/git-flow ./cmd/git-flow-plus
```

Or use the Makefile / build scripts to get version metadata embedded (see
[BuildGuide.md](BuildGuide.md)):

```bash
make build          # bin/git-flow, host platform only
make dist           # dist/, all 6 supported platforms
```

## Invoking it as `git flow ...`

Git resolves `git <subcommand>` to an executable named `git-<subcommand>` on
`PATH`. Name the installed binary `git-flow` (not `git-flow-plus`) and it
becomes usable as `git flow ...`:

```bash
git flow init
git flow release status
```

The binary itself doesn't care what it's named or how it's invoked — `git
flow-plus init` and `git-flow-plus init` (calling it directly) work
identically. Only the `git flow ...` spelling requires the `git-flow` name.

## Verify the install

```bash
git flow version
git flow doctor
```

`version` confirms the binary and build metadata (see
[CommandReference.md](CommandReference.md#git-flow-version)); `doctor` (run
inside a repository) confirms Git Flow Plus can see `git` and, once
initialized, that the branch/config state is healthy.

## Future packaging (not yet available)

Homebrew, Chocolatey, Scoop, `.deb`, `.rpm`, and an MSI installer are on the
[Roadmap](Roadmap.md) but not implemented yet — download-and-extract or
build-from-source are the only install paths today.
