# Packages the binaries in dist/ (produced by scripts\build.ps1) into
# per-platform release archives under dist\archives\: a .zip for Windows
# targets (the platform convention), a .tar.gz for Linux/macOS targets
# (via the `tar` bundled with Windows 10 1803+). Each archive also
# bundles README.md so a download is self-explanatory without needing the
# repository.
#
#   powershell -File scripts\build.ps1
#   powershell -File scripts\package.ps1

$ErrorActionPreference = "Stop"

$RootDir = Split-Path -Parent $PSScriptRoot
Set-Location $RootDir

$DistDir = Join-Path $RootDir "dist"
$ArchiveDir = Join-Path $DistDir "archives"
$Name = "git-flow-plus"

if (-not (Test-Path $DistDir) -or -not (Get-ChildItem -Path $DistDir -Filter "$Name-*" -File -ErrorAction SilentlyContinue)) {
    Write-Error "no binaries found in $DistDir\ - run scripts\build.ps1 first"
    exit 1
}

if (Test-Path $ArchiveDir) { Remove-Item -Recurse -Force $ArchiveDir }
New-Item -ItemType Directory -Path $ArchiveDir | Out-Null

$Targets = @(
    @{ GOOS = "windows"; GOARCH = "amd64"; Bin = "git-flow-plus-windows-amd64.exe" },
    @{ GOOS = "windows"; GOARCH = "arm64"; Bin = "git-flow-plus-windows-arm64.exe" },
    @{ GOOS = "linux";   GOARCH = "amd64"; Bin = "git-flow-plus-linux-amd64" },
    @{ GOOS = "linux";   GOARCH = "arm64"; Bin = "git-flow-plus-linux-arm64" },
    @{ GOOS = "darwin";  GOARCH = "amd64"; Bin = "git-flow-plus-darwin-amd64" },
    @{ GOOS = "darwin";  GOARCH = "arm64"; Bin = "git-flow-plus-darwin-arm64" }
)

foreach ($target in $Targets) {
    $src = Join-Path $DistDir $target.Bin
    if (-not (Test-Path $src)) {
        Write-Host "  skipping $($target.GOOS)/$($target.GOARCH): $src not found"
        continue
    }

    $stage = Join-Path ([System.IO.Path]::GetTempPath()) ([System.IO.Path]::GetRandomFileName())
    New-Item -ItemType Directory -Path $stage | Out-Null

    if ($target.GOOS -eq "windows") {
        $stagedBin = Join-Path $stage "$Name.exe"
    } else {
        $stagedBin = Join-Path $stage $Name
    }
    Copy-Item $src $stagedBin
    Copy-Item (Join-Path $RootDir "README.md") $stage

    $archiveBase = "$Name-$($target.GOOS)-$($target.GOARCH)"
    if ($target.GOOS -eq "windows") {
        $out = Join-Path $ArchiveDir "$archiveBase.zip"
        Write-Host "  -> $out"
        Compress-Archive -Path (Join-Path $stage "*") -DestinationPath $out -Force
    } else {
        $out = Join-Path $ArchiveDir "$archiveBase.tar.gz"
        Write-Host "  -> $out"
        tar -czf $out -C $stage .
        if ($LASTEXITCODE -ne 0) {
            throw "tar failed for $archiveBase"
        }
    }

    Remove-Item -Recurse -Force $stage
}

Write-Host "Done. Archives in $ArchiveDir\:"
Get-ChildItem $ArchiveDir
