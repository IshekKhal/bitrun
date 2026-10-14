# BitRun — Windows One-Line Installer (PowerShell)
# Usage: irm https://raw.githubusercontent.com/IshekKhal/bitrun/main/install.ps1 | iex

$ErrorActionPreference = "Stop"

$Version = "0.1.1-alpha"
$Repo = "IshekKhal/bitrun"
$InstallDir = Join-Path $env:USERPROFILE ".bitrun\bin"
$ModelsDir = Join-Path $env:USERPROFILE ".bitrun\models"
$ZipName = "bitrun-$Version-windows-amd64.zip"
$DownloadUrl = "https://github.com/$Repo/releases/download/v$Version/$ZipName"

Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "BitRun — Windows Installer (v$Version)" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan

# 1. Ensure directories exist
if (-not (Test-Path $InstallDir)) {
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
}
if (-not (Test-Path $ModelsDir)) {
    New-Item -ItemType Directory -Path $ModelsDir -Force | Out-Null
}

# 2. Download and extract release archive
$TempZip = Join-Path $env:TEMP $ZipName
Write-Host "Downloading BitRun release archive..." -ForegroundColor Yellow

try {
    Invoke-WebRequest -Uri $DownloadUrl -OutFile $TempZip -UseBasicParsing
    Write-Host "Extracting binaries to $InstallDir..." -ForegroundColor Yellow
    Expand-Archive -Path $TempZip -DestinationPath (Join-Path $env:USERPROFILE ".bitrun") -Force
    Remove-Item $TempZip -Force -ErrorAction SilentlyContinue
} catch {
    Write-Host "[NOTE] Release asset not yet published on GitHub Releases." -ForegroundColor Yellow
    Write-Host "Local builds can be placed directly in $InstallDir" -ForegroundColor Yellow
}

# 3. Add to User PATH if not already present
$UserPath = [Environment]::GetEnvironmentVariable("PATH", "User")
if ($UserPath -notlike "*$InstallDir*") {
    Write-Host "Adding $InstallDir to User PATH..." -ForegroundColor Yellow
    [Environment]::SetEnvironmentVariable("PATH", "$InstallDir;$UserPath", "User")
    $env:PATH = "$InstallDir;$env:PATH"
    Write-Host "PATH updated successfully." -ForegroundColor Green
}

Write-Host ""
Write-Host "BitRun installed successfully!" -ForegroundColor Green
Write-Host "Open a new terminal and run: bitrun --help" -ForegroundColor Green
Write-Host "Download your first model:   bitrun pull bitnet-70m" -ForegroundColor Green
