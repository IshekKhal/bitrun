# BitRun ⚡️

> **Zero-compile, self-contained standalone runner and model manager for Microsoft BitNet b1.58 and 1.58-bit ternary LLMs on macOS, Windows, and Linux CPUs.**

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/badge/release-v0.1.1--alpha-green.svg)](https://github.com/IshekKhal/bitrun/releases)
[![Architecture](https://img.shields.io/badge/arch-Apple%20Silicon%20%7C%20x86__64-orange.svg)](#)

---

## Why BitRun?

Microsoft's **BitNet b1.58** proved that ternary (1-bit / 1.58-bit) models can run at extreme speeds on commodity CPUs with 70–80% lower energy consumption by eliminating floating-point multiplications. 

Ternary models are engineered for **local CPU inference, battery-powered laptops, edge devices, and embedded tasks** where running heavy FP16 or INT4 models drains battery or requires dedicated GPUs.

However, testing `BitNet.cpp` today requires:
* Cloning recursive submodules and managing C++ compilers, CMake, and Python toolchains.
* Manually tuning CPU vector instruction flags (AVX2, AVX-512, ARM NEON).
* Dealing with 5–10 GB heavy Docker images that defeat the purpose of low-power computing.

**BitRun solves this by packaging the native inference engine into a self-contained runner with zero external dependencies and a familiar developer experience:** 

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
| **Falcon3 1B Instruct** (`falcon3-1b`) | **1.30 GB** | **137–142 t/s** | **98–127 t/s** | **~1.9 GB** | **0% (Pure CPU SIMD)** |
| **BitNet b1.58 2.4B** (`bitnet-2b`) | **1.13 GB** | **60.1 t/s** | **39.8 t/s** | **~1.8 GB** | **0% (Pure CPU SIMD)** |
| **BitNet 70M** (`bitnet-70m`) | **79 MB** | **8,758 t/s** | **1,355 t/s** | **< 150 MB** | **0% (Pure CPU SIMD)** |

### 2. Windows x86_64 (Intel Core i5-12600K CPU, 16 Threads, AVX2 SIMD)
| Model | Size | Prompt Speed | Generation Speed | Memory (RAM) | GPU Utilization |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Falcon3 1B Instruct** (`falcon3-1b`) | **1.30 GB** | **209.6 t/s** | **39–45 t/s** | **~1.9 GB** | **0% (Pure CPU AVX2)** |
| **BitNet 70M** (`bitnet-70m`) | **79 MB** | **2,419 t/s** | **492 t/s** | **< 150 MB** | **0% (Pure CPU AVX2)** |

### 3. Linux x86_64 (WSL2 / Ubuntu, 4 Cores, AVX2 SIMD)
| Model | Size | Prompt Speed | Generation Speed | Memory (RAM) | GPU Utilization |
| :--- | :--- | :--- | :--- | :--- | :--- |
| **Falcon3 1B Instruct** (`falcon3-1b`) | **1.30 GB** | **~130 t/s** | **40–45 t/s** | **~1.9 GB** | **0% (Pure CPU AVX2)** |
| **BitNet 70M** (`bitnet-70m`) | **79 MB** | **~2,100 t/s** | **~450 t/s** | **< 150 MB** | **0% (Pure CPU AVX2)** |

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

# Run with real-time power telemetry (Watts, Joules, Joules/token)
bitrun run bitnet-2b "Explain quantum tunneling" --power

# Benchmark generation speed, latency, and power consumption
bitrun bench bitnet-2b --tokens 64

# Create a custom model persona from a Modelfile
bitrun create gopher -f Modelfile

# Serve OpenAI-compatible REST API on port 11435 (avoids Ollama 11434 conflict)
bitrun serve falcon3-1b
```

---

## Energy & Power Telemetry (`--power` & `bitrun bench`)

Ternary 1-bit models shine in energy efficiency—running on 8–15W laptop and mini-PC CPUs rather than 250W–400W GPUs. BitRun includes built-in hardware power monitoring across macOS (Apple Silicon SmartBattery / SoC energy profiling), Linux (Intel/AMD RAPL microjoules), and Windows:

```bash
# Interactive run with energy monitoring
bitrun run bitnet-2b --power

# Run a synthetic benchmark
bitrun bench bitnet-2b --tokens 64 --threads 8
```

Example telemetry output:
```text
┌────────────────────────────────────────────────────────┐
│               BitRun Performance & Power               │
├───────────────────────┬────────────────────────────────┤
│ Prompt Evaluation     │ 83.3 tokens/sec (12 tokens)    │
│ Token Generation      │ 49.3 tokens/sec (64 tokens)    │
│ Time to First Token   │ 144 ms                         │
│ Average Power         │ 6.84 W                         │
│ Total Energy Consumed │ 8.89 Joules                    │
│ Energy Efficiency     │ 0.1389 Joules/token            │
│ Battery Drain Delta   │ 0.00%                          │
└───────────────────────┴────────────────────────────────┘
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

When running `bitrun serve`, you can point any OpenAI-compatible tool (Cursor, Continue.dev, Open WebUI, LangChain, Python SDK) to `http://localhost:11435/v1` (port 11435 avoids colliding with default Ollama installations):

```bash
curl -s http://127.0.0.1:11435/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "bitnet-70m",
    "messages": [
      {"role": "user", "content": "What is the capital of France?"}
    ],
    "temperature": 0.7
  }'
```

---

## Supported Models Catalog

| Model Tag | Parameters | Format | Disk Size | Context | Description |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `falcon3-1b` | 1B | `i2_s` Ternary | ~1.30 GB | 4,096 | TII Falcon3 1B Instruct conversational model |
| `bitnet-2b` | 2.4B | `i2_s` Ternary | ~1.13 GB | 2,048 | Official Microsoft BitNet b1.58 2B4T ternary model |
| `falcon3-3b` | 3B | `i2_s` Ternary | ~2.80 GB | 4,096 | TII Falcon3 3B Instruct conversational model |
| `bonsai-8b` | 8B | `i2_s` Ternary | ~2.08 GB | 4,096 | PrismML Bonsai 8B 1.58-bit model based on Qwen |
| `bonsai-1.7b` | 1.7B | `i2_s` Ternary | ~236 MB | 2,048 | PrismML Bonsai 1.7B compact 1.58-bit model |
| `bitnet-70m` | 70M | `Q8_0` | ~79 MB | 768 | Ultra-compact smoke test model |

---

## Upstream Kernel Patches & Engine Fixes

BitRun actively maintains and contributes low-level C++ fixes back to upstream `microsoft/BitNet`:

1. **ARM64 Block-Interleaved Weight Unpacking (51M Perplexity Resolution)**:
   - In GGUF `I2_S` format, weights are stored block-interleaved (32 bytes per 128 activations). Sequential unpacking caused severe matrix permutation and 51,732,149 perplexity on `wiki.test.raw`.
   - BitRun incorporates Scott Guyton's ARM NEON dot-product kernel (`vdotq_s32`) and 32-stride interleaved scalar fallback, bringing perplexity down to **18.84** (matching Hugging Face's 18.79 baseline) and achieving 49.3 t/s generation.
2. **CMake Submodule Auto-Patching**:
   - Automated CMake detection and patch application in `CMakeLists.txt` via `git apply --check` so developers building from source don't get unpatched submodule builds.
3. **MAD Kernel Batch Stride Fix**:
   - Fixed output pointer overwrites (`s[(row + rb) * bs]`) across `1x1`, `1x4`, and `1xN` kernels, preventing multi-row matrix evaluation memory corruption.
4. **Squared ReLU FFN Activation**:
   - Corrected model activation from SiLU to `LLM_FFN_RELU_SQR` matching Microsoft's 2.4B pre-training checkpoint.
5. **BLAS Context Safety & Tensor Alignment**:
   - Guarded `I2_S` tensors from BLAS crashes on context lengths $\ge 32$ tokens, and aligned `ggml_row_size()` to `ne / 4`.

All source patches and modified kernels are versioned and tracked directly in the [`patches/`](patches/) directory.

---

## Architecture Overview

BitRun uses a **decoupled supervisor-engine design**:
* **Supervisor Layer (Go)**: Cross-platform CLI, CPU topology detection, content-addressed asset store (`~/.bitrun`), power telemetry, and HTTP REST supervisor.
* **Engine Tier (Native C++)**: Statically compiled binary utilizing ARM NEON with dot-product (`ARM_FEATURE_DOTPROD`) and x86 AVX-512 kernels. Zero CGO dependencies, ensuring total crash isolation.

---

## Acknowledgements & Open Source Credits

BitRun stands on the shoulders of giants:
* **[llama.cpp](https://github.com/ggerganov/llama.cpp)** by Georgi Gerganov and the open-source community (MIT License) — The pioneering CPU inference architecture and GGUF specification.
* **[BitNet.cpp](https://github.com/microsoft/BitNet)** by Microsoft Research (MIT License) — Groundbreaking 1-bit and 1.58-bit ternary matrix kernels.

BitRun contributes back to upstream:
* **[microsoft/BitNet#642](https://github.com/microsoft/BitNet/pull/642)** — ARM64 2.4B batch stride write-backs, interleaved I2_S weight layout unpacking, squared ReLU activation, and CMake submodule auto-patching (bringing greedy perplexity to 18.84 on wiki.test.raw, matching reference).
* **[microsoft/BitNet#644](https://github.com/microsoft/BitNet/pull/644)** — Native MSVC compiler support and CLI build tools on Windows.

BitRun wraps these core engines with cross-platform zero-compile distribution, model management, Modelfile custom personas, and an OpenAI-compatible REST server.

---

## License

MIT License. See [LICENSE](LICENSE) for details.
