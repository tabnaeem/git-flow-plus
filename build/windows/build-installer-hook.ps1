#Requires -Version 5.1
<#
.SYNOPSIS
    GoReleaser build-hook wrapper around create-installer.ps1.

.DESCRIPTION
    Registered as this project's `builds[].hooks.post` in
    .goreleaser.yaml, so GoReleaser invokes it once per (GOOS, GOARCH)
    target it builds - six times in a single `goreleaser release`
    (windows/amd64, windows/arm64, linux/amd64, linux/arm64,
    darwin/amd64, darwin/arm64). installer.nsi only produces a single
    x64 installer, so every target except windows/amd64 is a silent
    no-op here.

    This is what makes "generate the installer" part of GoReleaser's
    own build phase instead of a separate, easy-to-forget step: it runs
    strictly after the windows/amd64 binary is written to disk (that's
    what a *post*-build hook means) and strictly before GoReleaser's
    archive/checksum/publish phases, which all happen later in the same
    `goreleaser release` invocation.

    A non-zero exit here fails GoReleaser's build step outright, which
    aborts the entire run before it ever reaches the publish phase -
    confirmed via a throwaway GoReleaser project during development
    (a hook that does `exit 1` produces "build failed", not a partial
    release). So "the installer didn't build" and "the release doesn't
    get published" are enforced by GoReleaser itself, not two things a
    human has to keep in sync by hand. create-installer.ps1 already
    verifies its own output file exists and throws if not, which is
    sufficient to trigger this - no separate existence check is needed
    here.

.PARAMETER Os
    GoReleaser's `{{ .Os }}` for the target just built.

.PARAMETER Arch
    GoReleaser's `{{ .Arch }}` for the target just built.

.PARAMETER BinaryPath
    GoReleaser's `{{ .Path }}` - the just-built binary's full path.

.PARAMETER Version
    GoReleaser's `{{ .Version }}` (no leading "v").
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)][string]$Os,
    [Parameter(Mandatory = $true)][string]$Arch,
    [Parameter(Mandatory = $true)][string]$BinaryPath,
    [Parameter(Mandatory = $true)][string]$Version
)

$ErrorActionPreference = "Stop"

if ($Os -ne "windows" -or $Arch -ne "amd64") {
    Write-Host "[installer-hook] skipping $Os/$Arch (installer.nsi is windows/amd64-only)"
    exit 0
}

$ScriptDir = $PSScriptRoot
$RepoRoot = Resolve-Path (Join-Path $ScriptDir "..\..")
$BinDir = Split-Path -Parent (Resolve-Path -LiteralPath $BinaryPath)
$OutDir = Join-Path $RepoRoot "dist\installer"

Write-Host "[installer-hook] windows/amd64 binary ready at $BinaryPath - building installer"
& (Join-Path $ScriptDir "create-installer.ps1") -Version $Version -BinDir $BinDir -OutDir $OutDir
