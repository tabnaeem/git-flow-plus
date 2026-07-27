# Cross-compiles Git Flow Plus release binaries for every supported
# platform into dist/. Run from anywhere; paths are resolved relative to
# the repository root.
#
#   powershell -File scripts\build.ps1
#
# Version metadata is embedded into the binary via -ldflags (see
# `git flow version`). Override any of these environment variables before
# invoking the script — this is how CI supplies real values instead of
# the git-derived defaults used for local builds:
#   VERSION       e.g. "1.2.0"       (default: git describe, or 0.0.0-dev)
#   BUILD_NUMBER  e.g. a CI run number (default: "dev")
#   GIT_COMMIT    e.g. short SHA     (default: git rev-parse --short HEAD)
#   GIT_BRANCH    e.g. "main"        (default: git rev-parse --abbrev-ref HEAD)
#   BUILD_DATE    RFC 3339 UTC       (default: now)

$ErrorActionPreference = "Stop"

$RootDir = Split-Path -Parent $PSScriptRoot
Set-Location $RootDir

$DistDir = "dist"
$Pkg = "./cmd/git-flow-plus"
$LdflagsPkg = "github.com/hulhub/git-flow-plus/internal/cli"

function Get-GitOrDefault([string[]]$GitArgs, [string]$Default) {
    try {
        $value = & git @GitArgs 2>$null
        if ($LASTEXITCODE -eq 0 -and $value) { return $value.Trim() }
    } catch {
        # git not available or not a repository; fall through to default
    }
    return $Default
}

$Version = $env:VERSION
if (-not $Version) { $Version = Get-GitOrDefault @("describe", "--tags", "--always", "--dirty") "0.0.0-dev" }

$BuildNumber = $env:BUILD_NUMBER
if (-not $BuildNumber) { $BuildNumber = "dev" }

$GitCommit = $env:GIT_COMMIT
if (-not $GitCommit) { $GitCommit = Get-GitOrDefault @("rev-parse", "--short", "HEAD") "unknown" }

$GitBranch = $env:GIT_BRANCH
if (-not $GitBranch) { $GitBranch = Get-GitOrDefault @("rev-parse", "--abbrev-ref", "HEAD") "unknown" }

$BuildDate = $env:BUILD_DATE
if (-not $BuildDate) { $BuildDate = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ") }

$Ldflags = "-s -w -X $LdflagsPkg.Version=$Version -X $LdflagsPkg.BuildNumber=$BuildNumber " +
           "-X $LdflagsPkg.GitCommit=$GitCommit -X $LdflagsPkg.GitBranch=$GitBranch -X $LdflagsPkg.BuildDate=$BuildDate"

# GOOS / GOARCH / output-filename triples for every supported platform.
$Targets = @(
    @{ GOOS = "windows"; GOARCH = "amd64"; Out = "git-flow-plus-windows-amd64.exe" },
    @{ GOOS = "windows"; GOARCH = "arm64"; Out = "git-flow-plus-windows-arm64.exe" },
    @{ GOOS = "linux";   GOARCH = "amd64"; Out = "git-flow-plus-linux-amd64" },
    @{ GOOS = "linux";   GOARCH = "arm64"; Out = "git-flow-plus-linux-arm64" },
    @{ GOOS = "darwin";  GOARCH = "amd64"; Out = "git-flow-plus-darwin-amd64" },
    @{ GOOS = "darwin";  GOARCH = "arm64"; Out = "git-flow-plus-darwin-arm64" }
)

if (Test-Path $DistDir) { Remove-Item -Recurse -Force $DistDir }
New-Item -ItemType Directory -Path $DistDir | Out-Null

Write-Host "Building Git Flow Plus $Version (commit $GitCommit, branch $GitBranch)"

$OriginalGOOS = $env:GOOS
$OriginalGOARCH = $env:GOARCH
$OriginalCGO = $env:CGO_ENABLED

try {
    foreach ($target in $Targets) {
        Write-Host "  -> $($target.GOOS)/$($target.GOARCH)"
        $env:GOOS = $target.GOOS
        $env:GOARCH = $target.GOARCH
        $env:CGO_ENABLED = "0"
        $outPath = Join-Path $DistDir $target.Out
        go build -trimpath -ldflags $Ldflags -o $outPath $Pkg
        if ($LASTEXITCODE -ne 0) {
            throw "build failed for $($target.GOOS)/$($target.GOARCH)"
        }
    }
} finally {
    if ($null -eq $OriginalGOOS) { Remove-Item Env:\GOOS -ErrorAction SilentlyContinue } else { $env:GOOS = $OriginalGOOS }
    if ($null -eq $OriginalGOARCH) { Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue } else { $env:GOARCH = $OriginalGOARCH }
    if ($null -eq $OriginalCGO) { Remove-Item Env:\CGO_ENABLED -ErrorAction SilentlyContinue } else { $env:CGO_ENABLED = $OriginalCGO }
}

Write-Host "Done. Binaries in $DistDir/:"
Get-ChildItem $DistDir
