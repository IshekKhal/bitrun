# BitRun ⚡️

> **Zero-compile runner, model manager, and live terminal dashboard for 1-bit and 1.58-bit ternary LLMs on Apple Silicon and x86_64 CPUs.**

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/badge/release-v0.2.0--beta-green.svg)](https://github.com/IshekKhal/bitrun/releases)
[![Architecture](https://img.shields.io/badge/arch-Apple%20Silicon%20%7C%20x86__64-orange.svg)](#)

---

## Why BitRun?

Microsoft's **BitNet b1.58** demonstrated that ternary (1-bit / 1.58-bit) language models can run at high speed on commodity CPUs with 70–80% lower energy consumption by replacing floating-point multiplications with integer additions. 

Ternary weights (`{-1, 0, +1}`) are engineered for **local CPU inference, battery-powered laptops, and edge devices** where running standard FP16 or INT4 models drains battery or demands an expensive GPU.

However, testing research implementations directly often requires:
* Cloning recursive C++ submodules and managing compiler toolchains, CMake, or virtual environments.
* Manually configuring CPU vector instruction flags (AVX2, AVX-512, ARM NEON).
* Dealing with multi-gigabyte Docker images that defeat the point of low-power computing.
* Debugging memory stride crashes and unhandled tensor formats on commodity hardware.

**BitRun packages the native inference engine into a standalone runner with zero external dependencies and a clean developer workflow:**

```bash
# 1. Pull a 1-bit model directly from Hugging Face
bitrun pull falcon3-1b

# 2. Launch the BitRun Cockpit full-screen dashboard
bitrun cockpit falcon3-1b

# 3. Or spin up a local OpenAI-compatible REST server
bitrun serve falcon3-1b
```

---

## The BitRun Cockpit

BitRun v0.2.0-beta includes **Cockpit**, an interactive full-screen terminal interface built with Go and Bubble Tea:

```text
 ◆ BitRun Cockpit • model: bonsai-1.7b • quant: PrismML Q1_0 • port: 11435       ⚡ Pure CPU SIMD (-ngl 0) 
╭────────────────────────────────────────────────────────╮╭────────────────────────────────────────────╮
│❯ User 12:11:37                                         ││ LIVE HARDWARE TELEMETRY                    │
│Explain how 1-bit models run on commodity CPUs.         ││ Gen Speed     : 73.7 tok/s (peak: 77.9)    │
│                                                        ││ Prompt Speed  : 433.5 tok/s                │
│◆ BitRun 12:11:37                                       ││ Prompt TTFT   : 138 ms                     │
│Ternary models constrain weight values to {-1, 0, +1}.  ││ Context       : 124/4096 (3%)              │
│This transforms matrix multiplication from floating-    ││                 [■□□□□□□□□□□□□□□□]         │
│point math into simple integer additions and            ││ P-Cores ( 8)  : [■■■■■■■■■□□□] 75%         │
│subtractions. Because weights take only 1.58 bits,      ││ E-Cores ( 4)  : [■□□□□□□□□□□□] 11%         │
│memory bandwidth drops by roughly 10x, allowing laptops ││ RAM Available : 24.0 GB (Host)             │
│to run inference at high speed while drawing minimal    ││ SIMD Engine   : ARM NEON DotProd (Native)  │
│power...                                                ││                                            │
│                                                        │╰────────────────────────────────────────────╯
│                                                        │╭────────────────────────────────────────────╮
│                                                        ││ POWER & ENERGY TELEMETRY                   │
│                                                        ││ Package Power : 11.7 W  [■■■□□□□□□□□□]     │
│                                                        ││ Energy/Token  : 0.1587 Joules              │
│                                                        ││ Efficiency    : 6.3 tokens/Joule           │
│                                                        ││ Session Energy: 216.125 mWh (778.0 J)      │
│                                                        ││ Power Source  : AC Attached                │
│                                                        ││ Battery Draw  : 0.0 W (AC Powered)         │
│                                                        ││ REST Endpoint : http://127.0.0.1:11435     │
│                                                        ││ Status        : [READY]                    │
│                                                        │╰────────────────────────────────────────────╯
╰────────────────────────────────────────────────────────╯                                             
 [Enter] Send Prompt • [↑/↓/PgUp/PgDn] Scroll Chat • [Ctrl+L] Clear Screen • [Esc/Ctrl+C] Quit Cockpit  
```

### Cockpit Capabilities
* **Dual-Pane Layout**: The left viewport maintains multi-turn conversation history with independent keyboard and mouse wheel scrolling.
* **Real-Time Streaming**: Tokens stream character-by-character via server-sent events (`/v1/chat/completions`) with an active cursor indicator.
* **Live Hardware Telemetry**: Samples prompt time-to-first-token (TTFT), instantaneous and peak generation speed, and P-core vs E-core utilization.
* **Energy Gauges**: Tracks package power in Watts, energy efficiency in Joules per token, cumulative milliwatt-hours consumed, and estimated battery runtime.
* **Clean Terminal Buffer**: Background inference engine logs redirect to `~/.bitrun/engine.log`, keeping the terminal display free of coordinate corruption.

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

Pre-compiled standalone archives are also available on the [Releases](https://github.com/IshekKhal/bitrun/releases) page for offline setup.

### 2. Basic Workflow

```bash
# Verify system topology and SIMD extensions
bitrun hardware

# Download a model from the registry
bitrun pull bonsai-1.7b

# Launch interactive Cockpit interface
bitrun cockpit bonsai-1.7b

# Or run a standard interactive terminal session
bitrun run bonsai-1.7b

# Run a one-shot query directly
bitrun run bonsai-1.7b "Summarize the advantages of ternary weight quantization."
```

---

## Supported Model Catalog

BitRun includes pre-configured registries for 6 ternary and compact models:

| Model Tag | Parameters | Quant Format | Download Size | Context Window | Best Suited For |
| :---: | :---: | :---: | :---: | :---: | :---: |
| **`bitnet-70m`** | 70 Million | `Q8_0` | 79 MB | 768 tokens | Micro smoke testing, pipeline verification, low-latency health checks. |
| **`bonsai-1.7b`**| 1.7 Billion | Group-128 `Q1_0` | 237 MB | 2,048 tokens | Fast instruction chat on battery (73 tok/s on Apple Silicon, 86 tok/s on Windows AVX2). |
| **`falcon3-1b`** | 1.0 Billion | Packed 2-bit `I2_S` | 1.30 GB | 4,096 tokens | TII Falcon3 instruction follower with low memory footprint. |
| **`bitnet-2b`**  | 2.4 Billion | Packed 2-bit `I2_S` | 1.13 GB | 2,048 tokens | Official Microsoft BitNet b1.58 2B4T foundation model with squared ReLU. |
| **`falcon3-3b`** | 3.0 Billion | Packed 2-bit `I2_S` | 2.07 GB | 4,096 tokens | Multi-step reasoning and coding tasks on commodity CPUs. |
| **`bonsai-8b`**  | 8.0 Billion | Group-128 `PQ2_0` | 2.08 GB | 8,192 tokens | High-parameter instruction chat on CPU (21.0 tok/s on Apple Silicon). |

---

## Verified Hardware Benchmarks

The figures below reflect physical hardware runs using native binaries with pure CPU SIMD execution (`--tokens 64`). Zero GPU offloading (`-ngl 0`).

### 1. Apple Silicon (M4 Pro CPU, 12 Cores: 8P + 4E, 24 GB Unified RAM)
| Model Tag | Size | Prompt Speed | Generation Speed | TTFT | Package Power | Energy / Token |
| :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| **`bitnet-70m`** | 79 MB | **11,549.2 t/s** | **1,619.8 t/s** | 78 ms | 11.70 W | 0.043 J/tok |
| **`bonsai-1.7b`**| 237 MB | **411.7 t/s** | **72.8 t/s** | 576 ms | 11.70 W | 0.316 J/tok |
| **`falcon3-1b`** | 1.30 GB | **170.8 t/s** | **73.2 t/s** | 628 ms | 11.70 W | 0.345 J/tok |
| **`bitnet-2b`**  | 1.13 GB | **94.4 t/s** | **31.6 t/s** | 881 ms | 11.70 W | 0.484 J/tok |
| **`falcon3-3b`** | 2.07 GB | **80.6 t/s** | **33.1 t/s** | 1.08 s | 11.70 W | 0.592 J/tok |
| **`bonsai-8b`**  | 2.08 GB | **30.1 t/s** | **21.0 t/s** | 1.62 s | 11.70 W | 0.886 J/tok |

### 2. Windows 10 Native (Intel Core i5-12600K, 16 Threads, 32 GB RAM, MSVC AVX2)
| Model Tag | Size | Prompt Speed | Generation Speed | TTFT | Package Power | Energy / Token |
| :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| **`bitnet-70m`** | 79 MB | **1,766.2 t/s** | **597.8 t/s** | 595 ms | 22.60 W | 0.038 J/tok |
| **`bonsai-1.7b`**| 237 MB | **152.4 t/s** | **85.6 t/s** | 1.42 s | 22.60 W | 0.264 J/tok |
| **`falcon3-1b`** | 1.30 GB | **14.9 t/s** | **11.1 t/s** | 5.48 s | 22.60 W | 2.036 J/tok |
| **`bitnet-2b`**  | 1.13 GB | **8.0 t/s** | **6.7 t/s** | 9.67 s | 22.60 W | 3.373 J/tok |
| **`falcon3-3b`** | 2.07 GB | **7.5 t/s** | **6.0 t/s** | 10.34 s | 22.60 W | 3.766 J/tok |
| **`bonsai-8b`**  | 2.08 GB | **4.7 t/s** | **4.0 t/s** | 15.38 s | 22.60 W | 5.650 J/tok |

### 3. Linux WSL2 (Intel Core i5-12600K, 4 vCPUs Allocated, GCC 13.3 AVX2)
| Model Tag | Size | Prompt Speed | Generation Speed | TTFT | Package Power | Energy / Token |
| :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| **`bitnet-70m`** | 79 MB | **333.1 t/s** | **134.1 t/s** | 159 ms | 8.50 W | 0.063 J/tok |
| **`bonsai-1.7b`**| 237 MB | **41.8 t/s** | **16.8 t/s** | 1.27 s | 8.50 W | 0.505 J/tok |
| **`falcon3-1b`** | 1.30 GB | **16.7 t/s** | **6.7 t/s** | 3.17 s | 8.50 W | 1.261 J/tok |
| **`bitnet-2b`**  | 1.13 GB | **10.9 t/s** | **4.4 t/s** | 4.86 s | 8.50 W | 1.938 J/tok |
| **`falcon3-3b`** | 2.07 GB | **8.7 t/s** | **3.5 t/s** | 6.12 s | 8.50 W | 2.439 J/tok |
| **`bonsai-8b`**  | 2.08 GB | **1.3 t/s** | **0.5 t/s** | 40.30 s | 8.50 W | 16.058 J/tok |

---

## Energy & Telemetry Monitoring

Ternary models run at low electrical power (8W to 22W total package draw) rather than 250W to 400W on dedicated graphics cards. BitRun incorporates hardware telemetry hooks across:
* **macOS**: Queries Apple Silicon SoC power counters and battery voltage/amperage via `ioreg`.
* **Linux**: Reads Intel and AMD processor energy directly from RAPL microjoule counters (`/sys/class/powercap/intel-rapl`).
* **Windows**: Measures CPU package power via Windows Processor Power Management (PPM).

```bash
# Run a prompt with real-time energy telemetry
bitrun run bonsai-1.7b "Explain quantum key distribution" --power

# Run a synthetic benchmark and report Joules/token
bitrun bench bonsai-1.7b --tokens 64 --threads 8
```

Example terminal telemetry box:
```text
┌────────────────────────────────────────────────────────┐
│               BitRun Performance & Power               │
├───────────────────────┬────────────────────────────────┤
│ Prompt Evaluation     │ 411.7 tokens/sec (12 tokens)   │
│ Token Generation      │ 72.8 tokens/sec (64 tokens)    │
│ Time to First Token   │ 576 ms                         │
│ Average Power         │ 11.70 W                        │
│ Total Energy Consumed │ 10.28 Joules                   │
│ Energy Efficiency     │ 0.316 Joules/token             │
│ Battery Drain Delta   │ 0.00%                          │
└───────────────────────┴────────────────────────────────┘
```

---

## Modelfiles & Custom Personas

You can define custom personas, temperature presets, and context windows using declarative Modelfiles:

```dockerfile
# Modelfile
FROM falcon3-1b
SYSTEM You are an expert systems engineer. Answer concisely with production-grade code.
PARAMETER temperature 0.2
PARAMETER context_size 4096
```

Build and run your custom persona:

```bash
# Create custom persona manifest
bitrun create gopher -f Modelfile

# Chat with your persona
bitrun run gopher

# Or launch it directly inside the Cockpit
bitrun cockpit gopher
```

---

## OpenAI-Compatible REST API

`bitrun serve` starts a background server on port `11435` (avoiding port collisions with Ollama on `11434`):

```bash
bitrun serve falcon3-1b --port 11435
```

### Direct HTTP Request
```bash
curl -s http://127.0.0.1:11435/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{
    "model": "falcon3-1b",
    "messages": [
      {"role": "user", "content": "How do 1-bit models conserve memory bandwidth?"}
    ],
    "temperature": 0.2
  }'
```

### Python SDK Example
```python
from openai import OpenAI

client = OpenAI(
    base_url="http://127.0.0.1:11435/v1",
    api_key="bitrun"  # Any string
)

response = client.chat.completions.create(
    model="falcon3-1b",
    messages=[{"role": "user", "content": "Write a concurrent queue in Go."}],
    stream=True
)

for chunk in response:
    content = chunk.choices[0].delta.content
    if content:
        print(content, end="", flush=True)
print()
```

Connects with Cursor, Continue.dev, Open WebUI, and LangChain by setting the base URL to `http://localhost:11435/v1`.

---

## Command Reference

| Command | Usage | Description |
| :---: | :---: | :---: |
| `bitrun cockpit <model>` | `bitrun cockpit bonsai-1.7b` | Opens the full-screen terminal dashboard with live hardware and energy gauges. |
| `bitrun run <model> [prompt]`| `bitrun run falcon3-1b` | Starts an interactive chat session or evaluates a single prompt. |
| `bitrun bench <model>` | `bitrun bench bitnet-2b --tokens 64` | Measures prompt speed, generation throughput, and Joules/token. |
| `bitrun pull <model>` | `bitrun pull bonsai-8b` | Downloads a model checkpoint from Hugging Face with resume support. |
| `bitrun serve <model>` | `bitrun serve falcon3-1b --port 11435` | Runs the local OpenAI-compatible REST server daemon. |
| `bitrun list` | `bitrun list` | Lists downloaded models and custom Modelfile personas. |
| `bitrun create <name> -f <file>`| `bitrun create gopher -f Modelfile` | Builds a custom persona manifest from a Modelfile. |
| `bitrun hardware` | `bitrun hardware` | Displays CPU cores, SIMD vector capabilities, and recommended thread allocations. |

---

## Native Engine & Kernel Architecture

BitRun pairs a high-level Go supervisor with statically linked C++ inference binaries:
* **The Go Supervisor Layer**: Manages CLI argument dispatch, model downloads, Modelfile manifests, hardware telemetry probing, and process supervision.
* **The Native Engine Tier**: Statically compiled C++ binaries (`llama-cli`, `llama-server`, `llama-quantize`) tracked on the sovereign `bitrun-engine` branch of modern upstream `llama.cpp` (b1395).

### Key Low-Level Engineering Fixes
1. **Dynamic GGUF Type 42 Remapping (`gguf.cpp`)**:
   PrismML's next-generation 8B ternary model (`bonsai-8b`) uses Group-128 quantization (`PQ2_0`, type 142), but legacy checkpoints write GGUF tensor type 42. BitRun dynamically remaps these tensors during initialization, eliminating the tensor offset calculation crash and enabling 8B ternary inference on commodity CPUs.
2. **Packed 2-Bit Memory Safety (`I2_S`)**:
   Global stride calculations are pinned to `.blck_size = 4` with 32-byte scale tail preservation, resolving the 4x stride overread that previously caused segmentation faults in forward `mul_mat` operations.
3. **Scott Guyton ARM NEON Kernel (`quants.c`)**:
   Uses `vdotq_s32` vector instructions to map `{0, 1, 2}` to true signed ternary values `{-1, 0, +1}`, paired with squared ReLU activations (`LLM_FFN_RELU_SQR`).
4. **x86_64 Fallback Macro Resolution (`arch-fallback.h`)**:
   Provides clean symbol mappings for generic scalar fallbacks under MSVC and GCC, allowing native builds on Windows and Linux without unresolved external linker errors.

---

## Acknowledgements & Open Source Credits

BitRun builds upon foundational open-source projects:
* **[llama.cpp](https://github.com/ggerganov/llama.cpp)** by Georgi Gerganov and contributors (MIT License) — Core CPU inference architecture and GGUF tensor specification.
* **[BitNet.cpp](https://github.com/microsoft/BitNet)** by Microsoft Research (MIT License) — 1-bit and 1.58-bit ternary matrix research.
* **[PrismML](https://prismml.com)** — The Bonsai ternary model family and Group-128 quantization designs.

Upstream contributions:
* **[microsoft/BitNet#642](https://github.com/microsoft/BitNet/pull/642)** — ARM64 interleaved weight unpacking, squared ReLU activation, and CMake submodule auto-patching.
* **[microsoft/BitNet#644](https://github.com/microsoft/BitNet/pull/644)** — Native MSVC compiler support and CLI tool targets on Windows.

---

## License

MIT License. See [LICENSE](LICENSE) for details.
