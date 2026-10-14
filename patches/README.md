# Upstream C++ Engine Patches & Kernel Fixes

This directory archives the exact C++ kernel math, build system patches, and runtime fixes developed for Microsoft's `bitnet.cpp` and submitted upstream via Pull Requests **#642** and **#644**.

These files are tracked directly in BitRun so that any build environment (macOS, Windows, Linux) has complete access to the fixes without relying on external git submodules.

---

## 1. Patches Manifest

| File | Upstream PR | Target Subsystem | Core Functionality |
| :--- | :--- | :--- | :--- |
| `0001-fix-arm64-2b-inference.patch` | [#642](https://github.com/microsoft/BitNet/pull/642) | `3rdparty/llama.cpp` | Unpacks 32-stride block-interleaved weights on ARM64 (`vdotq_s32`), unifies `QK_I2_S 128`, sets `LLM_FFN_RELU_SQR`, links MAD kernels, fixes column evaluation loops, bypasses BLAS I2_S crash, and aligns `ggml_row_size` to `ne / 4`. |
| `0002-bitrun-ascii-branding.patch` | BitRun Release | `3rdparty/llama.cpp` | Customizes `LLAMA_ASCII_LOGO` in `tools/cli/cli.cpp` with the official solid mixed-case BitRun terminal banner (capital B & R, small i-t-u-n). |
| `0003-fix-windows-msvc-build.patch` | [#644](https://github.com/microsoft/BitNet/pull/644) | Root & `src/` | Removes hardcoded MSVC compiler fatal error check and explicitly enables `LLAMA_BUILD_COMMON` and `LLAMA_BUILD_TOOLS` so Windows builds create CLI binaries. |
| `ggml-bitnet-mad.cpp` | [#642](https://github.com/microsoft/BitNet/pull/642) | `src/` | Fixes batch output stride overwrites (`s[(row + rb) * bs]`) across `1x1`, `1x4`, and `1xN` SIMD kernels for AVX2 and ARM NEON. |
| `CMakeLists.txt` | [#642](https://github.com/microsoft/BitNet/pull/642) & [#644](https://github.com/microsoft/BitNet/pull/644) | Root | Automatically detects and applies `0001-fix-arm64-2b-inference.patch` via `git apply --check`, enables `LLAMA_BUILD_COMMON`, `LLAMA_BUILD_TOOLS`, and `LLAMA_BUILD_SERVER`. |
| `src_CMakeLists.txt` | [#642](https://github.com/microsoft/BitNet/pull/642) & [#644](https://github.com/microsoft/BitNet/pull/644) | `src/` | Defines both `ggml-bitnet-mad.cpp` and `ggml-bitnet-lut.cpp` in `GGML_SOURCES_BITNET` and enables MSVC compiler support for Windows builds. |

---

## 2. Technical Context & Resolved Bugs

### Bug 1: ARM64 Interleaved Weight Permutation (51M Perplexity Bug)
* **Reported by**: Matheus Feitosa ([comment #6173981334](https://github.com/microsoft/BitNet/pull/642#issuecomment-6173981334)) on Linux aarch64 (Neoverse-N1).
* **Root Cause**: GGUF `I2_S` ternary weights are packed in 32-byte blocks interleaved across 128 activations (32-stride). The unpatched loop in `quants.c` unpacked bytes sequentially (`i / 4`, bit shift `6 - 2*(i % 4)`), multiplying weights against the wrong activation channels.
* **Resolution**: Integrated Scott Guyton's ARM NEON dot-product kernel (`vdotq_s32`) and 32-stride interleaved scalar fallback (`isHuangXin/llama.cpp#2`) into `quants.c`. Combined with squared ReLU (`LLM_FFN_RELU_SQR`), this dropped perplexity from **51,732,149** to **18.84** on `wiki.test.raw`, matching the Hugging Face reference baseline (**18.79**).

### Bug 2: Missing Submodule Patch Application in Default Builds
* **Reported by**: Matheus Feitosa.
* **Root Cause**: Default builds (`cmake -B build` or `python setup_env.py`) built unmodified submodules because patches were not auto-applied by CMake.
* **Resolution**: Added automated patch detection and application logic directly into top-level `CMakeLists.txt`.

### Bug 3: Batch Output Stride Overwrite in MAD Kernels
* **Root Cause**: Output pointers were written to `s[row]` rather than `s[row * bs]`, causing subsequent batch rows to clobber earlier rows in memory.
* **Resolution**: Standardized all output indexing across AVX2 and ARM NEON to `s[(row + rb) * bs]`.

### Bug 4: BLAS Multi-Turn Segmentation Fault
* **Root Cause**: BLAS claimed `MUL_MAT` on context lengths $\ge 32$ tokens, calling `dequantize_row_i2_s` with an unaligned offset that crashed with `EXC_BAD_ACCESS`.
* **Resolution**: Excluded `GGML_TYPE_I2_S` from `ggml_backend_blas_device_supports_op` and updated `ggml_row_size()` to return `(size_t)ne / 4`.

### Bug 5: Windows MSVC Hardcoded Compiler Rejection & Missing CLI Tools (PR #644)
* **Root Cause**: Upstream `src/CMakeLists.txt` explicitly threw `FATAL_ERROR "Clang or GCC is required"` when compiled on Windows using Microsoft Visual Studio C++ (MSVC). Furthermore, CLI tools (`llama-cli`) were omitted because `LLAMA_BUILD_TOOLS` was disabled in subdirectory mode.
* **Resolution**: Added `CMAKE_C_COMPILER_ID STREQUAL "MSVC"` to `src/CMakeLists.txt` and explicitly enabled `LLAMA_BUILD_COMMON=ON` and `LLAMA_BUILD_TOOLS=ON` in `CMakeLists.txt`.


---

## 3. How to Apply Patches Manually (If Not Using Auto-CMake)

```bash
# Apply engine patch to 3rdparty/llama.cpp
cd upstream/bitnet/3rdparty/llama.cpp
git apply ../../../patches/0001-fix-arm64-2b-inference.patch

# Copy updated MAD kernel and CMake files
cd ../..
cp ../../patches/ggml-bitnet-mad.cpp src/
cp ../../patches/CMakeLists.txt ./
cp ../../patches/src_CMakeLists.txt src/CMakeLists.txt

# Rebuild engine
cmake -B build -DBUILD_SHARED_LIBS=OFF -DLLAMA_BUILD_COMMON=ON -DLLAMA_BUILD_TOOLS=ON
cmake --build build --target llama-cli -j 8
```
