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

## Performance Benchmarks

### 1. Apple Silicon (M4 Pro CPU, 12 Cores, Pure CPU SIMD `-ngl 0`)
| Model | Size | Prompt Speed | Generation Speed | Memory (RAM) | GPU Utilization |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Falcon3 1B Instruct** (`falcon3-1b`) | **1.30 GB** | **137.5 t/s** | **98.6 t/s** | **~1.9 GB** | **0% (Pure CPU SIMD)** |
| **BitNet b1.58 2.4B** (`bitnet-2b`) | **1.13 GB** | **60.1 t/s** | **39.8 t/s** | **~1.8 GB** | **0% (Pure CPU SIMD)** |
| **BitNet 70M** (`bitnet-70m`) | **79 MB** | **8,758 t/s** | **1,355 t/s** | **< 150 MB** | **0% (Pure CPU SIMD)** |

### 2. Windows x86_64 (Intel Core i5-12600K CPU, 16 Threads, AVX2 SIMD)
| Model | Size | Prompt Speed | Generation Speed | Memory (RAM) | GPU Utilization |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Falcon3 1B Instruct** (`falcon3-1b`) | **1.30 GB** | **93–210 t/s** | **40–45 t/s** | **~1.9 GB** | **0% (Pure CPU AVX2)** |
| **BitNet 70M** (`bitnet-70m`) | **79 MB** | **2,419 t/s** | **492 t/s** | **< 150 MB** | **0% (Pure CPU AVX2)** |

### 3. Linux x86_64 (WSL2 / Ubuntu, 4 Cores, AVX2 SIMD)
| Model | Size | Prompt Speed | Generation Speed | Memory (RAM) | GPU Utilization |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Falcon3 1B Instruct** (`falcon3-1b`) | **1.30 GB** | **~130 ms TTFT** | **40–45 t/s** | **~1.9 GB** | **0% (Pure CPU AVX2)** |
| **BitNet 70M** (`bitnet-70m`) | **79 MB** | **< 25 ms TTFT** | **Fast CPU SIMD** | **< 150 MB** | **0% (Pure CPU AVX2)** |

---

## Quickstart

### 1. One-Line Installation

**macOS (Apple Silicon) & Linux (x86_64)**:
```bash
curl -fsSL https://raw.githubusercontent.com/IshekKhal/bitrun/main/install.sh | sh
```

**Windows (PowerShell)**:
```powershell
irm https://raw.githubusercontent.com/IshekKhal/bitrun/main/install.ps1 | iex
```

### 2. Available Commands

```bash
# Display CPU topology and vector instruction support
bitrun hardware

# List all downloaded models and custom personas
bitrun list

# Download a model with dynamic progress and resume support
bitrun pull falcon3-1b

# Run interactive terminal chat
bitrun run falcon3-1b

# Run a one-shot prompt directly
bitrun run falcon3-1b "Explain RAII in C++"

# Create a custom model persona from a Modelfile
bitrun create gopher -f Modelfile

# Serve OpenAI-compatible REST API on port 11435
bitrun serve falcon3-1b
```

---

## Modelfiles & Custom Personas

Create custom system personas and parameter presets using standard Modelfiles:

```dockerfile
# Modelfile
FROM falcon3-1b
SYSTEM You are an expert Go systems engineer. Write dense, production-grade code.
PARAMETER temperature 0.2
PARAMETER context_size 4096
```

Build and run your custom persona:

```bash
# Create custom persona
bitrun create gopher -f Modelfile

# Chat with your persona
bitrun run gopher

# Or run one-shot completions
bitrun run gopher "Write a ring buffer implementation"
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

## Acknowledgements & Open Source Credits

BitRun stands on the shoulders of giants:
* **[llama.cpp](https://github.com/ggerganov/llama.cpp)** by Georgi Gerganov and the open-source community (MIT License) — The pioneering CPU inference architecture and GGUF specification.
* **[BitNet.cpp](https://github.com/microsoft/BitNet)** by Microsoft Research (MIT License) — Groundbreaking 1-bit and 1.58-bit ternary matrix kernels.

BitRun contributes back to upstream:
* **[microsoft/BitNet#642](https://github.com/microsoft/BitNet/pull/642)** — ARM64 2B batch stride, FFN activation, and LLaMA-3 tokenizer inference fix.
* **[microsoft/BitNet#644](https://github.com/microsoft/BitNet/pull/644)** — Native MSVC compiler support and CLI build tools on Windows.

BitRun wraps these core engines with cross-platform zero-compile distribution, model management, Modelfile custom personas, and an OpenAI-compatible REST server.

---

## License

MIT License. See [LICENSE](LICENSE) for details.
