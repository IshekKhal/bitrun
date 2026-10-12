# BitRun ⚡️

> **The "Ollama of 1-bit LLMs"** — Zero-compile, single-binary runner and model manager for Microsoft BitNet b1.58 models on macOS, Linux, and Windows CPUs.

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/badge/release-v0.1.0--alpha-green.svg)](https://github.com/IshekKhal/bitrun/releases)
[![Architecture](https://img.shields.io/badge/arch-Apple%20Silicon%20%7C%20x86__64-orange.svg)](#)

---

## Why BitRun?

Microsoft's **BitNet b1.58** proved that ternary (1-bit / 1.58-bit) models can run at extreme speeds on commodity CPUs with 70–80% lower energy consumption by eliminating floating-point multiplications. 

However, running `BitNet.cpp` today requires:
* Cloning submodules and having modern C++ compilers, CMake, and Python toolchains.
* Manually tuning CPU vector instruction flags (AVX-512, AMX, ARM NEON).
* Dealing with 5–10 GB heavy Docker images that drain laptop batteries.

**BitRun solves this by packaging the native inference engine into a single binary.** 

```bash
# 1. Pull a 1-bit model directly from Hugging Face
bitrun pull bitnet-2b

# 2. Run an interactive chat session on your CPU
bitrun run bitnet-2b

# 3. Or spin up a local OpenAI-compatible REST server
bitrun serve bitnet-2b
```

---

## Performance Benchmark

Tested live on Apple Silicon (M4 Pro CPU, pure CPU inference with `-ngl 0`):

| Model | Size | Prompt Speed | Generation Speed | Memory (RAM) | GPU Utilization |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **BitNet b1.58 2.4B** (`bitnet-2b`) | **1.13 GB** | **60.1 t/s** | **39.8 t/s** | **~1.8 GB** | **0% (Pure CPU SIMD)** |
| **BitNet 70M** (`bitnet-70m`) | **79 MB** | **8,758 t/s** | **1,355 t/s** | **< 150 MB** | **0% (Pure CPU SIMD)** |

---

## Quickstart

### 1. One-Line Installation

```bash
curl -fsSL https://raw.githubusercontent.com/IshekKhal/bitrun/main/install.sh | sh
```

### 2. Available Commands

```bash
# Display CPU topology and vector instruction support
bitrun hardware

# List all cached models in ~/.bitrun/models
bitrun list

# Download a model with dynamic progress and resume support
bitrun pull bitnet-70m

# Run interactive terminal chat
bitrun run bitnet-70m

# Serve OpenAI-compatible REST API on port 11435
bitrun serve bitnet-70m
```

---

## OpenAI-Compatible API

When running `bitrun serve`, you can point any OpenAI-compatible tool (Cursor, Continue.dev, Open WebUI, LangChain, Python SDK) to `http://localhost:11435/v1`:

```bash
curl -s http://127.0.0.1:11435/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "Bitnet-M7-70m",
    "messages": [
      {"role": "user", "content": "What is the capital of France?"}
    ],
    "temperature": 0.7
  }'
```

---

## Supported Models Catalog

| Model Tag | Parameters | Format | Size | Description |
| :--- | :--- | :--- | :--- | :--- |
| `bitnet-70m` | 70M | `Q8_0` / Ternary | ~79 MB | Ultra-compact smoke test model |
| `bitnet-2b` | 2.4B | `i2_s` Ternary | ~1.2 GB | Official Microsoft BitNet b1.58 2B |
| `falcon3-1b` | 1B | `i2_s` Ternary | ~700 MB | TII Falcon3 1B Instruct 1.58-bit |
| `falcon3-3b` | 3B | `i2_s` Ternary | ~1.8 GB | TII Falcon3 3B Instruct 1.58-bit |
| `llama3-8b` | 8B | `i2_s` Ternary | ~2.4 GB | 1-Bit LLM Llama3 8B 1.58-bit |

---

## Architecture Overview

BitRun uses a **decoupled supervisor-engine design**:
* **Supervisor Layer (Go)**: Cross-platform CLI, CPU topology detection, content-addressed asset store (`~/.bitrun`), and HTTP REST supervisor.
* **Engine Tier (Native C++)**: Statically compiled binary utilizing ARM NEON with dot-product (`ARM_FEATURE_DOTPROD`) and x86 AVX-512 kernels. Zero CGO dependencies, ensuring total crash isolation.

---

## License

MIT License. Built on top of Microsoft Research's BitNet.cpp and Georgi Gerganov's llama.cpp ecosystem.
