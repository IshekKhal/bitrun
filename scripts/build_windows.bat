@echo off
setlocal enabledelayedexpansion

echo ===================================================
echo BitRun — Windows Native Build (Dynamic Discovery)
echo ===================================================

cd /d "%~dp0\.."

REM 1. Dynamic Toolchain & PATH Discovery
where cmake >nul 2>nul
if %errorlevel% neq 0 (
    echo [INFO] cmake not found on PATH. Searching for Visual Studio toolchain...
    set "VSWHERE=%ProgramFiles(x86)%\Microsoft Visual Studio\Installer\vswhere.exe"
    if exist "!VSWHERE!" (
        for /f "usebackq tokens=*" %%i in (`"!VSWHERE!" -latest -products * -property installationPath`) do (
            set "VSINSTALLDIR=%%i"
        )
        if exist "!VSINSTALLDIR!\Common7\Tools\VsDevCmd.bat" (
            echo [INFO] Activating Visual Studio Developer Command Prompt from:
            echo        !VSINSTALLDIR!
            call "!VSINSTALLDIR!\Common7\Tools\VsDevCmd.bat" -arch=x64 >nul 2>nul
        )
    )
    REM Also check common CMake standalone paths
    if not defined CMAKE_FOUND (
        if exist "!VSINSTALLDIR!\Common7\IDE\CommonExtensions\Microsoft\CMake\CMake\bin\cmake.exe" set "PATH=!VSINSTALLDIR!\Common7\IDE\CommonExtensions\Microsoft\CMake\CMake\bin;!PATH!"
        if exist "C:\Program Files\CMake\bin\cmake.exe" set "PATH=C:\Program Files\CMake\bin;!PATH!"
        if exist "C:\Program Files (x86)\CMake\bin\cmake.exe" set "PATH=C:\Program Files (x86)\CMake\bin;!PATH!"
    )
)

where go >nul 2>nul
if %errorlevel% neq 0 (
    echo [INFO] go not found on PATH. Checking default installation directories...
    if exist "C:\Program Files\Go\bin\go.exe" set "PATH=C:\Program Files\Go\bin;!PATH!"
    if exist "C:\Go\bin\go.exe" set "PATH=C:\Go\bin;!PATH!"
    if exist "%LOCALAPPDATA%\Programs\go\bin\go.exe" set "PATH=%LOCALAPPDATA%\Programs\go\bin;!PATH!"
    if exist "%LOCALAPPDATA%\Programs\Go\bin\go.exe" set "PATH=%LOCALAPPDATA%\Programs\Go\bin;!PATH!"
)

REM Verify tools after dynamic discovery
where cmake >nul 2>nul
if %errorlevel% neq 0 (
    echo [ERROR] cmake not found on PATH or standard directories.
    echo         Please install CMake or open a Developer Command Prompt for VS.
    exit /b 1
)

where go >nul 2>nul
if %errorlevel% neq 0 (
    echo [ERROR] go not found on PATH or standard directories.
    echo         Please install Go for Windows.
    exit /b 1
)

echo [INFO] Found cmake:
where cmake
echo [INFO] Found go:
where go

REM 2. Create bin directory
if not exist "bin" mkdir bin

echo [1/3] Configuring CMake for Native BitNet C++ Engine...
cd upstream\bitnet

cmake -B build -DGGML_AVX2=ON -DBUILD_SHARED_LIBS=OFF -DLLAMA_BUILD_COMMON=ON -DLLAMA_BUILD_TOOLS=ON -DLLAMA_BUILD_SERVER=ON -DLLAMA_CURL=OFF
if %errorlevel% neq 0 (
    echo [ERROR] CMake configuration failed.
    cd /d "%~dp0\.."
    exit /b 1
)

echo [2/3] Compiling C++ Inference Engine (Release)...
cmake --build build --config Release --target llama-cli llama-server llama-quantize
if %errorlevel% neq 0 (
    echo [ERROR] C++ Engine build failed.
    cd /d "%~dp0\.."
    exit /b 1
)

cd /d "%~dp0\.."

REM Copy native C++ binaries to bin\ (checking both flat and config directories)
for /r "upstream\bitnet\build\bin" %%f in (llama-cli.exe llama-server.exe llama-quantize.exe) do (
    if exist "%%f" (
        copy /y "%%f" bin\ >nul
        echo [INFO] Copied %%~nxf to bin\
    )
)

echo [3/3] Compiling Go Supervisor (bitrun.exe)...
go build -ldflags="-s -w" -o bin\bitrun.exe main.go
if %errorlevel% neq 0 (
    echo [ERROR] Go supervisor compilation failed.
    exit /b 1
)

echo ===================================================
echo [SUCCESS] BitRun Windows build complete!
echo Binaries ready in bin\
echo Test: .\bin\bitrun.exe hardware
echo ===================================================
