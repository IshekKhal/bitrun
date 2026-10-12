package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"bitrun/pkg/engine"
	"bitrun/pkg/hardware"
	"bitrun/pkg/models"
)

const version = "0.1.0-alpha"

func printUsage() {
	fmt.Printf("BitRun — The 1-bit LLM Runner (v%s)\n\n", version)
	fmt.Println("Usage:")
	fmt.Println("  bitrun <command> [arguments]\n")
	fmt.Println("Commands:")
	fmt.Println("  run <model>      Run an interactive chat session with a 1-bit model")
	fmt.Println("  pull <model>     Download model weights to local storage")
	fmt.Println("  list             List downloaded models in ~/.bitrun/models")
	fmt.Println("  serve <model>    Start local OpenAI-compatible HTTP API server")
	fmt.Println("  hardware         Display host CPU vector capabilities and thread tuning")
	fmt.Println("  version          Print BitRun version")
	fmt.Println("\nAvailable model tags:")
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
		fmt.Println("NAME            SIZE       PATH")
		count := 0
		for _, f := range files {
			if !f.IsDir() && filepath.Ext(f.Name()) == ".gguf" {
				info, _ := f.Info()
				sizeMB := float64(info.Size()) / (1024 * 1024)
				fmt.Printf("%-15s %-10.1f %s\n", f.Name(), sizeMB, filepath.Join(modelsDir, f.Name()))
				count++
			}
		}
		if count == 0 {
			fmt.Println("(No models downloaded yet. Run 'bitrun pull bitnet-2b' to install one)")
		}

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
			fmt.Fprintln(os.Stderr, "Usage: bitrun run <model-tag>")
			os.Exit(1)
		}
		tag := os.Args[2]
		def, ok := models.Catalog[tag]
		if !ok {
			fmt.Fprintf(os.Stderr, "[bitrun] Unknown model '%s'.\n", tag)
			os.Exit(1)
		}

		modelFile := filepath.Join(models.GetModelsDir(), def.Filename)
		if _, err := os.Stat(modelFile); os.IsNotExist(err) {
			fmt.Printf("[bitrun] Model '%s' not found locally. Initiating download...\n", tag)
			_, err = models.DownloadModel(def, models.GetModelsDir())
			if err != nil {
				fmt.Fprintf(os.Stderr, "[bitrun] Download failed: %v\n", err)
				os.Exit(1)
			}
		}

		engineBin := locateEngineBinary("llama-cli")
		if engineBin == "" {
			fmt.Fprintln(os.Stderr, "[bitrun] Error: Native engine 'llama-cli' not found. Run engine build first.")
			os.Exit(1)
		}

		cfg := engine.DefaultConfig(engineBin, modelFile)
		if def.ContextSize > 0 {
			cfg.ContextSize = def.ContextSize
		}
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		fmt.Printf("[bitrun] Loading %s on %d CPU threads...\n", def.Name, cfg.Threads)
		if err := engine.RunInteractive(ctx, cfg); err != nil {
			fmt.Fprintf(os.Stderr, "[bitrun] Session error: %v\n", err)
		}

	case "serve":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: bitrun serve <model-tag> [--port 11435]")
			os.Exit(1)
		}
		tag := os.Args[2]
		def, ok := models.Catalog[tag]
		if !ok {
			fmt.Fprintf(os.Stderr, "[bitrun] Unknown model '%s'.\n", tag)
			os.Exit(1)
		}

		modelFile := filepath.Join(models.GetModelsDir(), def.Filename)
		if _, err := os.Stat(modelFile); os.IsNotExist(err) {
			fmt.Printf("[bitrun] Model '%s' not found locally. Initiating download...\n", tag)
			_, err = models.DownloadModel(def, models.GetModelsDir())
			if err != nil {
				fmt.Fprintf(os.Stderr, "[bitrun] Download failed: %v\n", err)
				os.Exit(1)
			}
		}

		engineBin := locateEngineBinary("llama-server")
		if engineBin == "" {
			fmt.Fprintln(os.Stderr, "[bitrun] Error: Native engine 'llama-server' not found.")
			os.Exit(1)
		}

		cfg := engine.DefaultConfig(engineBin, modelFile)
		if def.ContextSize > 0 {
			cfg.ContextSize = def.ContextSize
		}
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()

		fmt.Printf("[bitrun] Starting OpenAI-compatible server for %s on http://%s:%d...\n", def.Name, cfg.Host, cfg.Port)
		proc, err := engine.StartServer(ctx, cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[bitrun] Failed to start server: %v\n", err)
			os.Exit(1)
		}
		defer proc.Stop()

		<-ctx.Done()
		fmt.Println("\n[bitrun] Server stopped.")

	default:
		flag.Usage()
		os.Exit(1)
	}
}

// locateEngineBinary checks local development build paths or ~/.bitrun/bin
func locateEngineBinary(name string) string {
	candidates := []string{
		filepath.Join(models.GetBinDir(), name),
		filepath.Join(".", "bin", name),
		filepath.Join("upstream", "bitnet", "build", "bin", name),
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}
