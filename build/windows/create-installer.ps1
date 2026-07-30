#Requires -Version 5.1
<#
.SYNOPSIS
    Builds GitFlowPlusSetup_v<version>_x64.exe from an already-built
    git-flow-plus.exe using NSIS.

.DESCRIPTION
    1. Stages the executable, README, and license into a clean directory.
    2. Copies the canonical LICENSE from the repo root, so the bundled
       license text can never drift from the real one.
    3. Invokes makensis with the version derived from the given tag.
    4. Verifies the installer was produced and prints its path/size/hash.

    See ../../Packaging.md for how this fits into the release pipeline,
    and installer.nsi for the installer itself.

.PARAMETER Version
    The release version, with or without a leading "v" (e.g. "1.3.2" or
    "v1.3.2"). Used verbatim (with "v" prefixed) in the output filename,
    and normalized to a 4-field numeric form for the embedded Windows
    file-version resource.

.PARAMETER BinDir
    Directory containing the already-built git-flow-plus.exe (windows/
    amd64) to package.

.PARAMETER OutDir
    Directory to write GitFlowPlusSetup_v<version>_x64.exe into. Created
    if it doesn't exist. Defaults to dist/installer relative to the repo
    root.

.EXAMPLE
    ./create-installer.ps1 -Version 1.3.2 -BinDir ../../dist/windows-x64

.EXAMPLE
    ./create-installer.ps1 -Version v1.3.2 -BinDir dist/git-flow-plus_windows_amd64_v1 -OutDir dist/installer
#>
[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$Version,

    [Parameter(Mandatory = $true)]
    [string]$BinDir,

    [string]$OutDir = ""
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$ScriptDir = $PSScriptRoot
$RepoRoot = Resolve-Path (Join-Path $ScriptDir "..\..")

if ([string]::IsNullOrWhiteSpace($OutDir)) {
    $OutDir = Join-Path $RepoRoot "dist\installer"
}

function Resolve-VersionParts {
    <#
    Normalizes a version string to (bare, file) where:
      bare = the version with any leading "v" stripped, used in the
             output filename as GitFlowPlusSetup_v<bare>_x64.exe
      file = bare padded/truncated to exactly 4 numeric dot-separated
             fields, for VIProductVersion (Win32 file-version resources
             are strictly 4 integers; Git Flow Plus's own version can
             have 5 fields - Sprint.Feature.ReleaseFix.DevOps.QA - so a
             5th field is dropped, matching the same rule the Inno/MSI
             installers use; see Packaging.md#versioning).
    #>
    param([string]$RawVersion)

    $bare = $RawVersion.TrimStart("v", "V")
    if ([string]::IsNullOrWhiteSpace($bare)) {
        throw "Version '$RawVersion' is empty after stripping a leading 'v'."
    }

    $fields = New-Object System.Collections.Generic.List[string]
    foreach ($part in $bare.Split(".")) {
        $digits = ($part -replace '[^0-9].*$', '')
        if ($digits -eq "") { $digits = "0" }
        $fields.Add($digits)
    }
    while ($fields.Count -lt 4) { $fields.Add("0") }
    $file = ($fields[0..3] -join ".")

    return [PSCustomObject]@{ Bare = $bare; File = $file }
}

function Find-MakeNsis {
    $onPath = Get-Command makensis -ErrorAction SilentlyContinue
    $candidates = New-Object System.Collections.Generic.List[string]
    if ($onPath) { $candidates.Add($onPath.Source) }
    $candidates.Add("C:\Program Files (x86)\NSIS\makensis.exe")
    $candidates.Add("C:\Program Files\NSIS\makensis.exe")

    foreach ($candidate in $candidates) {
        if (Test-Path $candidate) { return $candidate }
    }
    throw "makensis.exe not found. Install NSIS first (see Building.md) - e.g. 'winget install NSIS.NSIS' or 'choco install nsis'."
}

Write-Host "== Git Flow Plus Windows installer build ==" -ForegroundColor Cyan

$versionParts = Resolve-VersionParts -RawVersion $Version
Write-Host "Version:      $($versionParts.Bare)"
Write-Host "File version: $($versionParts.File)"

$exeSource = Join-Path $BinDir "git-flow-plus.exe"
if (-not (Test-Path $exeSource)) {
    throw "git-flow-plus.exe not found at '$exeSource'. Build it first (see Building.md)."
}

# --- 1. Copy executable ---
# installer.nsi's `File "${BIN_DIR}\..."` directive reads directly from
# -BinDir at compile time, so no separate staging copy is needed here -
# this step just confirms the binary that will be bundled is the one
# actually meant to ship.
Write-Host "Copying executable..."
Write-Host "  $exeSource"

# --- 2. Copy README ---
# build/windows/README.txt is checked in and hand-maintained (a short
# plain-text blurb, not a raw README.md dump) - installer.nsi bundles it
# directly by relative path, so there's nothing to refresh here.
$readmePath = Join-Path $ScriptDir "README.txt"
if (-not (Test-Path $readmePath)) {
    throw "README.txt not found at '$readmePath'."
}
Write-Host "Copying README..."
Write-Host "  $readmePath"

# --- 3. Copy license ---
# Always sourced from the repo's canonical LICENSE - never let the
# bundled license text silently drift from the real one.
Write-Host "Copying license..."
Copy-Item (Join-Path $RepoRoot "LICENSE") (Join-Path $ScriptDir "license.txt") -Force

# --- 4. Build the installer with NSIS ---
New-Item -ItemType Directory -Path $OutDir -Force | Out-Null

$makensis = Find-MakeNsis
Write-Host "Using makensis: $makensis"

$nsiScript = Join-Path $ScriptDir "installer.nsi"
$outDirAbs = (Resolve-Path -LiteralPath $OutDir -ErrorAction SilentlyContinue)
if (-not $outDirAbs) { $outDirAbs = New-Item -ItemType Directory -Path $OutDir -Force }
$binDirAbs = Resolve-Path -LiteralPath $BinDir

$nsisArgs = @(
    "/DVERSION=$($versionParts.Bare)",
    "/DFILEVERSION=$($versionParts.File)",
    "/DBIN_DIR=$binDirAbs",
    "/DOUT_DIR=$outDirAbs",
    $nsiScript
)

Write-Host "Building installer..."
& $makensis @nsisArgs
if ($LASTEXITCODE -ne 0) {
    throw "makensis failed with exit code $LASTEXITCODE"
}

# --- 5. Output: verify and report ---
$outFile = Join-Path $outDirAbs "GitFlowPlusSetup_v$($versionParts.Bare)_x64.exe"
if (-not (Test-Path $outFile)) {
    throw "Expected installer not found at '$outFile' after a successful build."
}

$hash = (Get-FileHash -Path $outFile -Algorithm SHA256).Hash.ToLower()
$size = (Get-Item $outFile).Length

Write-Host ""
Write-Host "== Done ==" -ForegroundColor Green
Write-Host "Output: $outFile"
Write-Host "Size:   $size bytes"
Write-Host "SHA256: $hash"
