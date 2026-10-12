package engine

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	"bitrun/pkg/hardware"
)

// RunnerConfig specifies execution parameters for the native BitNet engine.
type RunnerConfig struct {
	EngineBinary string
	ModelPath    string
	Prompt       string
	Threads      int
	ContextSize  int
	Temperature  float64
	Host         string
	Port         int
}

// DefaultConfig generates a recommended configuration based on host hardware.
func DefaultConfig(engineBinary, modelPath string) *RunnerConfig {
	hw := hardware.Detect()
	return &RunnerConfig{
		EngineBinary: engineBinary,
		ModelPath:    modelPath,
		Threads:      hw.Threads,
		ContextSize:  2048,
		Temperature:  0.7,
		Host:         "127.0.0.1",
		Port:         11435,
	}
}

// RunInteractive executes an interactive chat session using the native engine.
func RunInteractive(ctx context.Context, cfg *RunnerConfig) error {
	if _, err := os.Stat(cfg.EngineBinary); os.IsNotExist(err) {
		return fmt.Errorf("engine binary not found at %s", cfg.EngineBinary)
	}
	if _, err := os.Stat(cfg.ModelPath); os.IsNotExist(err) {
		return fmt.Errorf("model file not found at %s", cfg.ModelPath)
	}

	args := []string{
		"-m", cfg.ModelPath,
		"-c", strconv.Itoa(cfg.ContextSize),
		"-t", strconv.Itoa(cfg.Threads),
		"--temp", fmt.Sprintf("%.2f", cfg.Temperature),
		"-ngl", "0", // Run on CPU for pure 1-bit SIMD execution
		"-cnv",      // Conversation / chat mode
		"--log-disable",
	}

	cmd := exec.CommandContext(ctx, cfg.EngineBinary, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// RunOneShot executes a single prompt and streams output to the provided writer.
func RunOneShot(ctx context.Context, cfg *RunnerConfig, out io.Writer) error {
	args := []string{
		"-m", cfg.ModelPath,
		"-p", cfg.Prompt,
		"-c", strconv.Itoa(cfg.ContextSize),
		"-t", strconv.Itoa(cfg.Threads),
		"--temp", fmt.Sprintf("%.2f", cfg.Temperature),
		"-ngl", "0",
	}

	cmd := exec.CommandContext(ctx, cfg.EngineBinary, args...)
	cmd.Stdout = out
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// ServerProcess manages the background llama-server instance.
type ServerProcess struct {
	cmd  *exec.Cmd
	host string
	port int
}

// StartServer launches the native engine as an HTTP API server.
func StartServer(ctx context.Context, cfg *RunnerConfig) (*ServerProcess, error) {
	args := []string{
		"-m", cfg.ModelPath,
		"-c", strconv.Itoa(cfg.ContextSize),
		"-t", strconv.Itoa(cfg.Threads),
		"--host", cfg.Host,
		"--port", strconv.Itoa(cfg.Port),
		"--temp", fmt.Sprintf("%.2f", cfg.Temperature),
		"-ngl", "0",
		"-cb", // continuous batching
	}

	cmd := exec.CommandContext(ctx, cfg.EngineBinary, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start server: %w", err)
	}

	proc := &ServerProcess{
		cmd:  cmd,
		host: cfg.Host,
		port: cfg.Port,
	}

	// Probe server health until ready
	healthURL := fmt.Sprintf("http://%s:%d/health", cfg.Host, cfg.Port)
	client := &http.Client{Timeout: 500 * time.Millisecond}

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get(healthURL)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return proc, nil
		}
		time.Sleep(200 * time.Millisecond)
	}

	return proc, nil
}

// Stop gracefully terminates the server process.
func (sp *ServerProcess) Stop() error {
	if sp.cmd != nil && sp.cmd.Process != nil {
		return sp.cmd.Process.Kill()
	}
	return nil
}
