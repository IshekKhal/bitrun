package engine

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"bitrun/pkg/hardware"
	"bitrun/pkg/telemetry"
)

// RunnerConfig specifies execution parameters for the native BitNet engine.
type RunnerConfig struct {
	EngineBinary  string
	ModelPath     string
	Prompt        string
	SystemPrompt  string
	ChatTemplate  string
	Threads       int
	ContextSize   int
	PredictTokens int
	Temperature   float64
	StopTokens    []string
	Host          string
	Port          int
	ModelAlias    string
	EnablePower   bool
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
		"-ngl", "0",            // Run on CPU for pure 1-bit SIMD execution
		"--no-op-offload",      // Disable Metal offload attempts on Apple Silicon
		"-cnv",                 // Conversation / chat mode
	}

	if cfg.SystemPrompt != "" {
		args = append(args, "-sys", cfg.SystemPrompt)
	}
	if cfg.ChatTemplate != "" {
		args = append(args, "--chat-template", cfg.ChatTemplate)
	}

	stopTokens := []string{
		"<|end of answer|>",
		"<|endoftext|>",
		"<|eot_id|>",
		"<|im_end|>",
	}
	if len(cfg.StopTokens) > 0 {
		stopTokens = append(stopTokens, cfg.StopTokens...)
	}
	for _, st := range stopTokens {
		args = append(args, "-r", st)
	}
	args = append(args, "--log-disable")

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
		"-c", strconv.Itoa(cfg.ContextSize),
		"-t", strconv.Itoa(cfg.Threads),
		"--temp", fmt.Sprintf("%.2f", cfg.Temperature),
		"-ngl", "0",
		"--no-op-offload",
		"-cnv",
		"-st",
		"-p", cfg.Prompt,
	}

	if cfg.PredictTokens > 0 {
		args = append(args, "-n", strconv.Itoa(cfg.PredictTokens))
	}

	if cfg.SystemPrompt != "" {
		args = append(args, "-sys", cfg.SystemPrompt)
	}
	if cfg.ChatTemplate != "" {
		args = append(args, "--chat-template", cfg.ChatTemplate)
	}

	stopTokens := []string{
		"<|end of answer|>",
		"<|endoftext|>",
		"<|eot_id|>",
		"<|im_end|>",
	}
	if len(cfg.StopTokens) > 0 {
		stopTokens = append(stopTokens, cfg.StopTokens...)
	}
	for _, st := range stopTokens {
		args = append(args, "-r", st)
	}
	args = append(args, "--log-disable")

	cmd := exec.CommandContext(ctx, cfg.EngineBinary, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = out
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// RunBenchmark executes a standardized benchmark and captures telemetry metrics.
func RunBenchmark(ctx context.Context, cfg *RunnerConfig, out io.Writer) (*telemetry.PowerMetrics, error) {
	pm := telemetry.NewPowerMonitor(cfg.Threads)
	pm.Start()

	var buf bytes.Buffer
	multiOut := io.MultiWriter(out, &buf)

	startTime := time.Now()
	err := RunOneShot(ctx, cfg, multiOut)
	totalDur := time.Since(startTime)
	if err != nil {
		return nil, err
	}

	output := buf.String()

	// Parse llama timing stats if printed: [ Prompt: 83.3 t/s | Generation: 49.3 t/s ]
	var promptSpeed float64
	var genSpeed float64

	promptRe := regexp.MustCompile(`Prompt:\s*([\d\.]+)\s*t/s`)
	if m := promptRe.FindStringSubmatch(output); len(m) > 1 {
		promptSpeed, _ = strconv.ParseFloat(m[1], 64)
	}

	genRe := regexp.MustCompile(`Generation:\s*([\d\.]+)\s*t/s`)
	if m := genRe.FindStringSubmatch(output); len(m) > 1 {
		genSpeed, _ = strconv.ParseFloat(m[1], 64)
	}

	// Approximate token counts
	promptTokens := len(strings.Fields(cfg.Prompt)) * 4 / 3
	genTokens := cfg.PredictTokens
	if genTokens <= 0 {
		genTokens = 64
	}

	var genDur time.Duration
	if genSpeed > 0 {
		genDur = time.Duration(float64(genTokens)/genSpeed*1000) * time.Millisecond
	} else {
		genDur = totalDur * 3 / 4
	}
	promptDur := totalDur - genDur
	if promptDur < 0 {
		promptDur = 10 * time.Millisecond
	}

	metrics := pm.Stop(promptTokens, genTokens, promptDur, genDur)
	if promptSpeed > 0 {
		metrics.PromptSpeed = promptSpeed
	}
	if genSpeed > 0 {
		metrics.GenSpeed = genSpeed
	}

	return &metrics, nil
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
		"--no-op-offload",
		"-cb", // continuous batching
	}

	if cfg.ModelAlias != "" {
		args = append(args, "-a", cfg.ModelAlias)
	}
	if cfg.ChatTemplate != "" {
		args = append(args, "--chat-template", cfg.ChatTemplate)
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
