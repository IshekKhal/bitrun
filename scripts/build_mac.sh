#!/bin/bash
set -e

echo "==================================================="
echo "BitRun — macOS Native Build (Clang/Apple Silicon + Go)"
echo "Target: darwin-arm64 (Pure CPU ARM NEON + DOTPROD)"
echo "==================================================="

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

# Check prerequisites
command -v cmake >/dev/null 2>&1 || { echo "[ERROR] cmake not found on PATH. Run: brew install cmake"; exit 1; }
command -v go >/dev/null 2>&1 || { echo "[ERROR] go not found on PATH. Run: brew install go"; exit 1; }

mkdir -p bin

# 1. Build Native C++ Engine
echo "[1/3] Configuring CMake for Native BitRun C++ Engine..."
if [ ! -d "upstream/llama.cpp" ]; then
    echo "[INFO] Cloning BitRun engine (bitrun-engine branch) from GitHub..."
    git clone --depth 1 -b bitrun-engine https://github.com/IshekKhal/llama.cpp.git upstream/llama.cpp
fi
cd upstream/llama.cpp

NUM_CORES=$(sysctl -n hw.ncpu 2>/dev/null || echo 8)

cmake -B build -DBUILD_SHARED_LIBS=OFF -DCMAKE_BUILD_TYPE=Release -DLLAMA_BUILD_COMMON=ON -DLLAMA_BUILD_TOOLS=ON -DLLAMA_BUILD_SERVER=ON -DLLAMA_CURL=OFF
cmake --build build --config Release --target llama-cli llama-server llama-quantize -j "$NUM_CORES"

cd "$PROJECT_ROOT"

# Copy compiled binaries to bin/
cp upstream/llama.cpp/build/bin/llama-cli bin/
cp upstream/llama.cpp/build/bin/llama-server bin/
cp upstream/llama.cpp/build/bin/llama-quantize bin/

# 2. Compile Go Supervisor
echo "[2/3] Compiling Go Supervisor (bitrun)..."
go build -ldflags="-s -w" -o bin/bitrun main.go

echo "==================================================="
echo "[SUCCESS] BitRun macOS build complete!"
echo "Binaries ready in bin/"
echo "Test: ./bin/bitrun hardware"
echo "==================================================="
