# BitRun - Windows Packaging Script (PowerShell)
# Produces dist/bitrun-0.2.0-beta-windows-amd64.zip

$ErrorActionPreference = "Stop"
$ProjectRoot = Resolve-Path "$PSScriptRoot\.."

$Version = "0.2.0-beta"
$PackageName = "bitrun-$Version-windows-amd64"
$DistDir = Join-Path $ProjectRoot "dist"
$StagingDir = Join-Path $DistDir $PackageName
$StagingBinDir = Join-Path $StagingDir "bin"

Write-Host "Packaging BitRun $Version for windows-amd64..." -ForegroundColor Cyan

if (Test-Path $StagingDir) {
    Remove-Item $StagingDir -Recurse -Force
}
New-Item -ItemType Directory -Path $StagingBinDir -Force | Out-Null

# Copy binaries
Copy-Item (Join-Path $ProjectRoot "bin\bitrun.exe") $StagingBinDir -Force
Copy-Item (Join-Path $ProjectRoot "bin\llama-cli.exe") $StagingBinDir -Force
Copy-Item (Join-Path $ProjectRoot "bin\llama-server.exe") $StagingBinDir -Force
Copy-Item (Join-Path $ProjectRoot "bin\llama-quantize.exe") $StagingBinDir -Force

# Copy readme and license
Copy-Item (Join-Path $ProjectRoot "README.md") $StagingDir -Force
Copy-Item (Join-Path $ProjectRoot "LICENSE") $StagingDir -Force

# Create zip archive
$ZipPath = Join-Path $DistDir "$PackageName.zip"
if (Test-Path $ZipPath) {
    Remove-Item $ZipPath -Force
}
Compress-Archive -Path "$StagingDir\*" -DestinationPath $ZipPath -Force

Write-Host "Release archive created: $ZipPath" -ForegroundColor Green
Get-Item $ZipPath | Select-Object Name, Length, LastWriteTime
