# BitRun — Windows Native Build Script (PowerShell with Dynamic Discovery)
# Target: windows-amd64 (Pure CPU 1-bit inference)

$ErrorActionPreference = "Stop"
$ProjectRoot = Resolve-Path "$PSScriptRoot\.."

Write-Host "===================================================" -ForegroundColor Cyan
Write-Host "BitRun — Windows Native Build (Dynamic Discovery)" -ForegroundColor Cyan
Write-Host "===================================================" -ForegroundColor Cyan

# 1. Dynamic Toolchain & PATH Discovery
if (-not (Get-Command cmake -ErrorAction SilentlyContinue)) {
    Write-Host "[INFO] cmake not found on current PATH. Searching common locations..." -ForegroundColor Yellow
    $vswhere = Join-Path ${env:ProgramFiles(x86)} "Microsoft Visual Studio\Installer\vswhere.exe"
    $vsPath = $null
    if (Test-Path $vswhere) {
        $vsPath = & $vswhere -latest -products * -property installationPath
        if ($vsPath -is [array]) { $vsPath = $vsPath[0] }
    }
    $CommonCmake = @(
        "C:\Program Files\CMake\bin",
        "C:\Program Files (x86)\CMake\bin",
        if ($vsPath) { Join-Path $vsPath "Common7\IDE\CommonExtensions\Microsoft\CMake\CMake\bin" },
        "C:\Program Files (x86)\Microsoft Visual Studio\2022\BuildTools\Common7\IDE\CommonExtensions\Microsoft\CMake\CMake\bin",
        "C:\Program Files (x86)\Microsoft Visual Studio\18\BuildTools\Common7\IDE\CommonExtensions\Microsoft\CMake\CMake\bin",
        "C:\Program Files\Microsoft Visual Studio\2022\Community\Common7\IDE\CommonExtensions\Microsoft\CMake\CMake\bin",
        "C:\Program Files\Microsoft Visual Studio\2022\Professional\Common7\IDE\CommonExtensions\Microsoft\CMake\CMake\bin",
        "C:\Program Files\Microsoft Visual Studio\2022\Enterprise\Common7\IDE\CommonExtensions\Microsoft\CMake\CMake\bin",
        "C:\Program Files (x86)\Microsoft Visual Studio\2019\Community\Common7\IDE\CommonExtensions\Microsoft\CMake\CMake\bin"
    )
    foreach ($p in $CommonCmake) {
        if ($p -and (Test-Path (Join-Path $p "cmake.exe"))) {
            $env:PATH = "$p;$env:PATH"
            Write-Host "[INFO] Added cmake to PATH from: $p" -ForegroundColor Green
            break
        }
    }
}

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "[INFO] go not found on current PATH. Searching common locations..." -ForegroundColor Yellow
    $CommonGo = @(
        "C:\Program Files\Go\bin",
        "C:\Go\bin",
        "$env:LOCALAPPDATA\Programs\go\bin",
        "$env:LOCALAPPDATA\Programs\Go\bin"
    )
    foreach ($p in $CommonGo) {
        if (Test-Path (Join-Path $p "go.exe")) {
            $env:PATH = "$p;$env:PATH"
            Write-Host "[INFO] Added go to PATH from: $p" -ForegroundColor Green
            break
        }
    }
}

# Verify tools
if (-not (Get-Command cmake -ErrorAction SilentlyContinue)) {
    Write-Error "cmake could not be located. Please install CMake or configure PATH."
}
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Error "go could not be located. Please install Go for Windows."
}

Write-Host "[INFO] Found cmake at: $(Get-Command cmake | Select-Object -ExpandProperty Source)" -ForegroundColor Green
Write-Host "[INFO] Found go at:    $(Get-Command go | Select-Object -ExpandProperty Source)" -ForegroundColor Green

# 2. Ensure bin folder exists
$BinDir = Join-Path $ProjectRoot "bin"
if (-not (Test-Path $BinDir)) {
    New-Item -ItemType Directory -Path $BinDir | Out-Null
}

# 3. Build native C++ engine
Write-Host "[1/3] Configuring CMake for Native BitNet C++ Engine..." -ForegroundColor Yellow
$BitnetDir = Join-Path $ProjectRoot "upstream\bitnet"
if (-not (Test-Path $BitnetDir)) {
    Write-Host "[INFO] Cloning BitRun engine (bitrun-engine branch) from GitHub..." -ForegroundColor Yellow
    git clone --depth 1 -b bitrun-engine --recursive https://github.com/IshekKhal/BitNet.git $BitnetDir
}
Push-Location $BitnetDir

cmake -B build -DGGML_AVX2=ON -DBUILD_SHARED_LIBS=OFF -DLLAMA_BUILD_COMMON=ON -DLLAMA_BUILD_TOOLS=ON -DLLAMA_BUILD_SERVER=ON -DLLAMA_CURL=OFF
if ($LASTEXITCODE -ne 0) {
    Pop-Location
    Write-Error "CMake configuration failed."
}

Write-Host "[2/3] Compiling C++ Inference Engine (Release)..." -ForegroundColor Yellow
cmake --build build --config Release --target llama-cli llama-server llama-quantize
if ($LASTEXITCODE -ne 0) {
    Pop-Location
    Write-Error "C++ engine build failed."
}

Pop-Location

# Copy native C++ binaries to bin\ (checking both flat and config directories)
$FoundBinaries = Get-ChildItem -Path (Join-Path $BitnetDir "build\bin") -Recurse -Include "llama-cli.exe","llama-server.exe","llama-quantize.exe" -ErrorAction SilentlyContinue
foreach ($bin in $FoundBinaries) {
    Copy-Item $bin.FullName $BinDir -Force
    Write-Host "[INFO] Copied $($bin.Name) to bin\" -ForegroundColor Green
}

# 4. Compile Go Supervisor
Write-Host "[3/3] Compiling Go Supervisor (bitrun.exe)..." -ForegroundColor Yellow
Push-Location $ProjectRoot
go build -ldflags="-s -w" -o (Join-Path $BinDir "bitrun.exe") main.go
if ($LASTEXITCODE -ne 0) {
    Pop-Location
    Write-Error "Go supervisor build failed."
}
Pop-Location

Write-Host "===================================================" -ForegroundColor Green
Write-Host "[SUCCESS] BitRun Windows build complete!" -ForegroundColor Green
Write-Host "Binaries ready in: $BinDir" -ForegroundColor Green
Write-Host "Test by running: .\bin\bitrun.exe hardware" -ForegroundColor Green
Write-Host "===================================================" -ForegroundColor Green
