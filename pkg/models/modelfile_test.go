package models

import (
	"os"
	"strings"
	"testing"
)

func TestParseModelfileBasic(t *testing.T) {
	content := `
# Simple Modelfile for BitRun
FROM falcon3-1b
SYSTEM You are an expert Go systems engineer. Write dense, production-grade code.
PARAMETER temperature 0.2
PARAMETER context_size 4096
PARAMETER stop "<|endoftext|>"
`
	manifest, err := ParseModelfile(strings.NewReader(content))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if manifest.BaseModel != "falcon3-1b" {
		t.Errorf("expected base model 'falcon3-1b', got '%s'", manifest.BaseModel)
	}

	expectedSys := "You are an expert Go systems engineer. Write dense, production-grade code."
	if manifest.SystemPrompt != expectedSys {
		t.Errorf("expected system prompt %q, got %q", expectedSys, manifest.SystemPrompt)
	}

	if temp, ok := manifest.Parameters["temperature"].(float64); !ok || temp != 0.2 {
		t.Errorf("expected temperature 0.2, got %v (%T)", manifest.Parameters["temperature"], manifest.Parameters["temperature"])
	}

	if ctx, ok := manifest.Parameters["context_size"].(int); !ok || ctx != 4096 {
		t.Errorf("expected context_size 4096, got %v (%T)", manifest.Parameters["context_size"], manifest.Parameters["context_size"])
	}

	if stop, ok := manifest.Parameters["stop"].(string); !ok || stop != "<|endoftext|>" {
		t.Errorf("expected stop token '<|endoftext|>', got %v", manifest.Parameters["stop"])
	}
}

func TestParseModelfileMultiline(t *testing.T) {
	content := `
FROM bitnet-2b

SYSTEM """
You are a senior Linux kernel developer.
You prioritize deterministic memory layout and zero-alloc paths.
Always explain with exact C snippets.
"""

PARAMETER temp 0.1
PARAMETER num_ctx 2048
`
	manifest, err := ParseModelfile(strings.NewReader(content))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if manifest.BaseModel != "bitnet-2b" {
		t.Errorf("expected base model 'bitnet-2b', got '%s'", manifest.BaseModel)
	}

	if !strings.Contains(manifest.SystemPrompt, "Linux kernel developer") {
		t.Errorf("missing kernel developer prompt in: %s", manifest.SystemPrompt)
	}
	if !strings.Contains(manifest.SystemPrompt, "zero-alloc paths") {
		t.Errorf("missing zero-alloc prompt in: %s", manifest.SystemPrompt)
	}

	// Normalized parameter keys
	if temp, ok := manifest.Parameters["temperature"].(float64); !ok || temp != 0.1 {
		t.Errorf("expected temperature 0.1, got %v", manifest.Parameters["temperature"])
	}
	if ctx, ok := manifest.Parameters["context_size"].(int); !ok || ctx != 2048 {
		t.Errorf("expected context_size 2048, got %v", manifest.Parameters["context_size"])
	}
}

func TestParseModelfileMissingFrom(t *testing.T) {
	content := `
SYSTEM You are an assistant.
PARAMETER temperature 0.5
`
	_, err := ParseModelfile(strings.NewReader(content))
	if err == nil {
		t.Fatal("expected error for missing FROM, got nil")
	}
}

func TestSaveAndLoadManifest(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	t.Setenv("HOME", tmpDir)
	defer t.Setenv("HOME", origHome)

	m := &Manifest{
		Name:         "test-gopher",
		BaseModel:    "falcon3-1b",
		SystemPrompt: "Test system prompt.",
		Parameters: map[string]interface{}{
			"temperature":  0.2,
			"context_size": 4096,
		},
	}

	if err := SaveManifest(m); err != nil {
		t.Fatalf("failed to save manifest: %v", err)
	}

	loaded, err := LoadManifest("test-gopher")
	if err != nil {
		t.Fatalf("failed to load manifest: %v", err)
	}

	if loaded.Name != "test-gopher" {
		t.Errorf("expected name 'test-gopher', got '%s'", loaded.Name)
	}
	if loaded.BaseModel != "falcon3-1b" {
		t.Errorf("expected base model 'falcon3-1b', got '%s'", loaded.BaseModel)
	}
	if loaded.SystemPrompt != "Test system prompt." {
		t.Errorf("expected system prompt 'Test system prompt.', got '%s'", loaded.SystemPrompt)
	}

	manifests, err := ListManifests()
	if err != nil {
		t.Fatalf("failed to list manifests: %v", err)
	}
	if len(manifests) != 1 {
		t.Fatalf("expected 1 manifest, got %d", len(manifests))
	}

	if err := DeleteManifest("test-gopher"); err != nil {
		t.Fatalf("failed to delete manifest: %v", err)
	}

	if ManifestExists("test-gopher") {
		t.Errorf("expected manifest to be deleted")
	}
}
