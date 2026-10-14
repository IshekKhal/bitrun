package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"bitrun/pkg/engine"
	"bitrun/pkg/hardware"
	"bitrun/pkg/models"
	"bitrun/pkg/telemetry"
)

const version = "0.1.0-alpha"

func printUsage() {
	fmt.Printf("BitRun — The 1-bit LLM Runner (v%s)\n\n", version)
	fmt.Println("Usage:")
	fmt.Println("  bitrun <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  run <model> [prompt] [--power]  Run an interactive chat or prompt with energy telemetry")
	fmt.Println("  bench <model> [--tokens N]      Benchmark prompt processing, speed, and power consumption")
	fmt.Println("  pull <model>                    Download model weights to local storage")
	fmt.Println("  list                            List downloaded models and custom personas")
	fmt.Println("  create <name> -f <file>         Create a custom model persona from a Modelfile")
	fmt.Println("  serve <model>                   Start local OpenAI-compatible HTTP API server")
	fmt.Println("  hardware                        Display host CPU vector capabilities and thread tuning")
	fmt.Println("  version                         Print BitRun version")
	fmt.Println("\nAvailable base model tags:")
	for tag, def := range models.Catalog {
		fmt.Printf("  %-14s %s (~%d MB)\n", tag, def.Description, def.SizeMB)
	}
	fmt.Println()
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	if err := models.EnsureDirectories(); err != nil {
		fmt.Fprintf(os.Stderr, "[bitrun] Error: %v\n", err)
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "version", "--version", "-v":
		fmt.Printf("bitrun version %s (%s/%s)\n", version, hardware.Detect().OS, hardware.Detect().Arch)

	case "hardware":
		hw := hardware.Detect()
		fmt.Printf("Host Hardware Topology:\n")
		fmt.Printf("  Operating System:   %s\n", hw.OS)
		fmt.Printf("  Architecture:       %s\n", hw.Arch)
		fmt.Printf("  Logical CPU Cores:  %d\n", hw.NumCPU)
		fmt.Printf("  Recommended Threads:%d\n", hw.Threads)
		fmt.Printf("  Total Memory:       %.1f GB\n", hw.TotalRAMGB)
		fmt.Printf("  ARM DotProd SIMD:   %t\n", hw.HasDotProd)

	case "list":
		modelsDir := models.GetModelsDir()
		files, err := os.ReadDir(modelsDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[bitrun] Error listing models: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("DOWNLOADED BASE MODELS:")
		fmt.Printf("  %-32s %-12s %s\n", "NAME", "SIZE", "TAG")
		count := 0
		for _, f := range files {
			if !f.IsDir() && filepath.Ext(f.Name()) == ".gguf" {
				info, _ := f.Info()
				sizeMB := float64(info.Size()) / (1024 * 1024)
				tag := "-"
				for t, def := range models.Catalog {
					if def.Filename == f.Name() {
						tag = t
						break
					}
				}
				fmt.Printf("  %-32s %-12.1f %s\n", f.Name(), sizeMB, tag)
				count++
			}
		}
		if count == 0 {
			fmt.Println("  (No base models downloaded yet. Run 'bitrun pull falcon3-1b' to install one)")
		}

		manifests, _ := models.ListManifests()
		fmt.Println("\nCUSTOM MODELS (PERSONAS):")
		fmt.Printf("  %-18s %-14s %-12s %s\n", "NAME", "BASE MODEL", "CREATED", "SYSTEM PROMPT")
		if len(manifests) == 0 {
			fmt.Println("  (No custom models created yet. Run 'bitrun create <name> -f Modelfile')")
		} else {
			for _, m := range manifests {
				created := m.CreatedAt
				if len(created) > 10 {
					created = created[:10]
				}
				sysSnippet := strings.ReplaceAll(m.SystemPrompt, "\n", " ")
				if len(sysSnippet) > 38 {
					sysSnippet = sysSnippet[:35] + "..."
				}
				fmt.Printf("  %-18s %-14s %-12s %s\n", m.Name, m.BaseModel, created, sysSnippet)
			}
		}
		fmt.Println()

	case "create":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: bitrun create <name> -f <Modelfile>")
			os.Exit(1)
		}

		var name string
		var modelfilePath string
		for i := 2; i < len(os.Args); i++ {
			arg := os.Args[i]
			if arg == "-f" || arg == "--file" {
				if i+1 < len(os.Args) {
					modelfilePath = os.Args[i+1]
					i++
				}
			} else if name == "" && !strings.HasPrefix(arg, "-") {
				name = arg
			}
		}

		if name == "" || modelfilePath == "" {
			fmt.Fprintln(os.Stderr, "Usage: bitrun create <name> -f <Modelfile>")
			os.Exit(1)
		}

		manifest, err := models.ParseModelfilePath(modelfilePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[bitrun] Error parsing Modelfile: %v\n", err)
			os.Exit(1)
		}

		manifest.Name = name

		// Validate base model exists in Catalog or as direct file
		if _, ok := models.Catalog[manifest.BaseModel]; !ok {
			if _, err := os.Stat(manifest.BaseModel); os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "[bitrun] Unknown base model '%s' in %s.\nAvailable models: bitnet-70m, falcon3-1b, bitnet-2b, falcon3-3b, bonsai-8b, bonsai-1.7b\n", manifest.BaseModel, modelfilePath)
				os.Exit(1)
			}
		}

		if err := models.SaveManifest(manifest); err != nil {
			fmt.Fprintf(os.Stderr, "[bitrun] Failed to save model manifest: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("[bitrun] Created model '%s' from %s (base: %s)\n", name, modelfilePath, manifest.BaseModel)
		fmt.Printf("Run 'bitrun run %s' to start chatting.\n", name)

	case "pull":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: bitrun pull <model-tag>")
			os.Exit(1)
		}
		tag := os.Args[2]
		def, ok := models.Catalog[tag]
		if !ok {
			fmt.Fprintf(os.Stderr, "[bitrun] Unknown model '%s'. Run 'bitrun' to see available models.\n", tag)
			os.Exit(1)
		}
		_, err := models.DownloadModel(def, models.GetModelsDir())
		if err != nil {
			fmt.Fprintf(os.Stderr, "[bitrun] Download failed: %v\n", err)
			os.Exit(1)
		}

	case "run":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: bitrun run <model-tag-or-name>")
			os.Exit(1)
		}
		tag := os.Args[2]
		modelPath, def, manifest, err := resolveModel(tag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[bitrun] %v\n", err)
			os.Exit(1)
		}

		if _, statErr := os.Stat(modelPath); os.IsNotExist(statErr) {
			if def.Tag != "" && def.RemoteFilename != "" {
				fmt.Printf("[bitrun] Base model '%s' not found locally. Initiating download...\n", def.Tag)
				_, err = models.DownloadModel(def, models.GetModelsDir())
				if err != nil {
					fmt.Fprintf(os.Stderr, "[bitrun] Download failed: %v\n", err)
					os.Exit(1)
				}
			} else {
				fmt.Fprintf(os.Stderr, "[bitrun] Model file not found: %s\n", modelPath)
				os.Exit(1)
			}
		}

		engineBin := locateEngineBinary("llama-cli")
		if engineBin == "" {
			fmt.Fprintln(os.Stderr, "[bitrun] Error: Native engine 'llama-cli' not found. Run engine build first.")
			os.Exit(1)
		}

		cfg := engine.DefaultConfig(engineBin, modelPath)
		if def.ContextSize > 0 {
			cfg.ContextSize = def.ContextSize
		}

		if manifest != nil {
			if manifest.SystemPrompt != "" {
				cfg.SystemPrompt = manifest.SystemPrompt
			}
			if manifest.Template != "" {
				cfg.ChatTemplate = manifest.Template
			}
			if temp, ok := manifest.Parameters["temperature"].(float64); ok {
				cfg.Temperature = temp
			}
			if ctx, ok := manifest.Parameters["context_size"].(int); ok {
				cfg.ContextSize = ctx
			}
			if thr, ok := manifest.Parameters["threads"].(int); ok {
				cfg.Threads = thr
			}
			if stop, ok := manifest.Parameters["stop"].(string); ok {
				cfg.StopTokens = append(cfg.StopTokens, stop)
			}
		}

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		if manifest != nil {
			fmt.Printf("[bitrun] Loading persona '%s' (base: %s) on %d CPU threads...\n", manifest.Name, def.Name, cfg.Threads)
		} else {
			fmt.Printf("[bitrun] Loading %s on %d CPU threads...\n", def.Name, cfg.Threads)
		}

		enablePower := false
		var cleanPromptArgs []string
		for _, arg := range os.Args[3:] {
			if arg == "--power" || arg == "-w" {
				enablePower = true
			} else {
				cleanPromptArgs = append(cleanPromptArgs, arg)
			}
		}

		var pm *telemetry.PowerMonitor
		if enablePower {
			pm = telemetry.NewPowerMonitor(cfg.Threads)
			pm.Start()
		}

		if len(cleanPromptArgs) > 0 {
			cfg.Prompt = strings.Join(cleanPromptArgs, " ")
			startTime := time.Now()
			if err := engine.RunOneShot(ctx, cfg, os.Stdout); err != nil {
				fmt.Fprintf(os.Stderr, "[bitrun] Session error: %v\n", err)
			}
			if enablePower && pm != nil {
				promptTokens := len(strings.Fields(cfg.Prompt)) * 4 / 3
				genDur := time.Since(startTime)
				metrics := pm.Stop(promptTokens, 64, 20*time.Millisecond, genDur)
				fmt.Print(telemetry.FormatTelemetryBox(metrics))
			}
		} else {
			if err := engine.RunInteractive(ctx, cfg); err != nil {
				fmt.Fprintf(os.Stderr, "[bitrun] Session error: %v\n", err)
			}
			if enablePower && pm != nil {
				metrics := pm.Stop(0, 0, 0, 0)
				fmt.Print(telemetry.FormatTelemetryBox(metrics))
			}
		}

	case "bench":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: bitrun bench <model-tag-or-name> [--tokens 64] [--threads N] [--prompt \"...\"]")
			os.Exit(1)
		}
		tag := os.Args[2]
		modelPath, def, manifest, err := resolveModel(tag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[bitrun] %v\n", err)
			os.Exit(1)
		}

		if _, statErr := os.Stat(modelPath); os.IsNotExist(statErr) {
			if def.Tag != "" && def.RemoteFilename != "" {
				fmt.Printf("[bitrun] Base model '%s' not found locally. Initiating download...\n", def.Tag)
				_, err = models.DownloadModel(def, models.GetModelsDir())
				if err != nil {
					fmt.Fprintf(os.Stderr, "[bitrun] Download failed: %v\n", err)
					os.Exit(1)
				}
			} else {
				fmt.Fprintf(os.Stderr, "[bitrun] Model file not found: %s\n", modelPath)
				os.Exit(1)
			}
		}

		engineBin := locateEngineBinary("llama-cli")
		if engineBin == "" {
			fmt.Fprintln(os.Stderr, "[bitrun] Error: Native engine 'llama-cli' not found.")
			os.Exit(1)
		}

		cfg := engine.DefaultConfig(engineBin, modelPath)
		if def.ContextSize > 0 {
			cfg.ContextSize = def.ContextSize
		}
		cfg.PredictTokens = 64
		cfg.Prompt = "The architectural breakthrough of 1-bit ternary neural networks is that weights are constrained to minus one, zero, and plus one. This allows matrix multiplications to be replaced by simple additions and subtractions. Explain why this fundamentally changes CPU inference efficiency."

		for i := 3; i < len(os.Args); i++ {
			arg := os.Args[i]
			if (arg == "--tokens" || arg == "-n") && i+1 < len(os.Args) {
				if n, err := strconv.Atoi(os.Args[i+1]); err == nil && n > 0 {
					cfg.PredictTokens = n
					i++
				}
			} else if (arg == "--threads" || arg == "-t") && i+1 < len(os.Args) {
				if t, err := strconv.Atoi(os.Args[i+1]); err == nil && t > 0 {
					cfg.Threads = t
					i++
				}
			} else if (arg == "--prompt" || arg == "-p") && i+1 < len(os.Args) {
				cfg.Prompt = os.Args[i+1]
				i++
			}
		}

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		displayName := def.Name
		if manifest != nil {
			displayName = fmt.Sprintf("%s (persona: %s)", def.Name, manifest.Name)
		}
		fmt.Printf("[bitrun] Benchmarking %s (%d tokens, %d CPU threads)...\n\n", displayName, cfg.PredictTokens, cfg.Threads)

		metrics, err := engine.RunBenchmark(ctx, cfg, os.Stdout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\n[bitrun] Benchmark error: %v\n", err)
			os.Exit(1)
		}

		if metrics != nil {
			fmt.Print(telemetry.FormatTelemetryBox(*metrics))
		}

	case "serve":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: bitrun serve <model-tag-or-name> [--port 11435]")
			os.Exit(1)
		}
		tag := os.Args[2]
		modelPath, def, manifest, err := resolveModel(tag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[bitrun] %v\n", err)
			os.Exit(1)
		}

		if _, statErr := os.Stat(modelPath); os.IsNotExist(statErr) {
			if def.Tag != "" && def.RemoteFilename != "" {
				fmt.Printf("[bitrun] Base model '%s' not found locally. Initiating download...\n", def.Tag)
				_, err = models.DownloadModel(def, models.GetModelsDir())
				if err != nil {
					fmt.Fprintf(os.Stderr, "[bitrun] Download failed: %v\n", err)
					os.Exit(1)
				}
			} else {
				fmt.Fprintf(os.Stderr, "[bitrun] Model file not found: %s\n", modelPath)
				os.Exit(1)
			}
		}

		engineBin := locateEngineBinary("llama-server")
		if engineBin == "" {
			fmt.Fprintln(os.Stderr, "[bitrun] Error: Native engine 'llama-server' not found.")
			os.Exit(1)
		}

		cfg := engine.DefaultConfig(engineBin, modelPath)
		if def.ContextSize > 0 {
			cfg.ContextSize = def.ContextSize
		}

		if manifest != nil {
			cfg.ModelAlias = manifest.Name
			if manifest.SystemPrompt != "" {
				cfg.SystemPrompt = manifest.SystemPrompt
			}
			if manifest.Template != "" {
				cfg.ChatTemplate = manifest.Template
			}
			if temp, ok := manifest.Parameters["temperature"].(float64); ok {
				cfg.Temperature = temp
			}
			if ctx, ok := manifest.Parameters["context_size"].(int); ok {
				cfg.ContextSize = ctx
			}
			if thr, ok := manifest.Parameters["threads"].(int); ok {
				cfg.Threads = thr
			}
		} else {
			cfg.ModelAlias = tag
		}

		for i := 3; i < len(os.Args); i++ {
			if (os.Args[i] == "--port" || os.Args[i] == "-p") && i+1 < len(os.Args) {
				if p, err := strconv.Atoi(os.Args[i+1]); err == nil {
					cfg.Port = p
				}
			}
		}

		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		displayName := def.Name
		if manifest != nil {
			displayName = fmt.Sprintf("%s (persona: %s)", def.Name, manifest.Name)
		}
		fmt.Printf("[bitrun] Starting OpenAI-compatible server for %s on http://%s:%d...\n", displayName, cfg.Host, cfg.Port)
		proc, err := engine.StartServer(ctx, cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[bitrun] Failed to start server: %v\n", err)
			os.Exit(1)
		}
		defer proc.Stop()

		<-ctx.Done()
		fmt.Println("\n[bitrun] Server stopped.")

	default:
		printUsage()
		os.Exit(1)
	}
}

// resolveModel resolves a model tag or custom persona name to its disk path and definition.
func resolveModel(tag string) (string, models.ModelDefinition, *models.Manifest, error) {
	if models.ManifestExists(tag) {
		manifest, err := models.LoadManifest(tag)
		if err != nil {
			return "", models.ModelDefinition{}, nil, fmt.Errorf("failed to load custom model '%s': %w", tag, err)
		}
		baseDef, ok := models.Catalog[manifest.BaseModel]
		if ok {
			modelPath := filepath.Join(models.GetModelsDir(), baseDef.Filename)
			return modelPath, baseDef, manifest, nil
		}
		if _, statErr := os.Stat(manifest.BaseModel); statErr == nil {
			modelPath := manifest.BaseModel
			def := models.ModelDefinition{
				Tag:         manifest.Name,
				Name:        manifest.Name,
				ContextSize: 2048,
			}
			return modelPath, def, manifest, nil
		}
		return "", models.ModelDefinition{}, nil, fmt.Errorf("custom model '%s' specifies unknown base model '%s'", tag, manifest.BaseModel)
	}

	baseDef, ok := models.Catalog[tag]
	if !ok {
		return "", models.ModelDefinition{}, nil, fmt.Errorf("unknown model '%s'. Run 'bitrun list' or 'bitrun' for options", tag)
	}
	modelPath := filepath.Join(models.GetModelsDir(), baseDef.Filename)
	return modelPath, baseDef, nil, nil
}

// locateEngineBinary checks local development build paths or ~/.bitrun/bin
func locateEngineBinary(name string) string {
	exts := []string{""}
	if runtime.GOOS == "windows" {
		exts = []string{".exe", ""}
	}

	for _, ext := range exts {
		binName := name + ext
		candidates := []string{
			filepath.Join(models.GetBinDir(), binName),
			filepath.Join(".", "bin", binName),
			filepath.Join("upstream", "bitnet", "build", "bin", binName),
			filepath.Join("upstream", "bitnet", "build", "bin", "Release", binName),
			filepath.Join("upstream", "bitnet", "build", "bin", "Debug", binName),
		}
		for _, path := range candidates {
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}
	return ""
}
