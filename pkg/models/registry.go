package models

import (
	"fmt"
	"os"
	"path/filepath"
)

// ModelDefinition contains catalog metadata for a supported 1-bit model.
type ModelDefinition struct {
	Tag            string `json:"tag"`
	Name           string `json:"name"`
	Repo           string `json:"repo"`
	RemoteFilename string `json:"remote_filename"`
	Filename       string `json:"filename"`
	SizeMB         int64  `json:"size_mb"`
	ContextSize    int    `json:"context_size"`
	Description    string `json:"description"`
}

// Catalog contains known model tags and their upstream locations.
var Catalog = map[string]ModelDefinition{
	"bitnet-70m": {
		Tag:            "bitnet-70m",
		Name:           "Bitnet-M7-70m",
		Repo:           "gate369/Bitnet-M7-70m-Q8_0-GGUF",
		RemoteFilename: "bitnet-m7-70m.Q8_0.gguf",
		Filename:       "bitnet-m7-70m.Q8_0.gguf",
		SizeMB:         79,
		ContextSize:    768,
		Description:    "Micro 70M test model (79 MB)",
	},
	"falcon3-1b": {
		Tag:            "falcon3-1b",
		Name:           "Falcon3-1B-Instruct-1.58bit",
		Repo:           "tiiuae/Falcon3-1B-Instruct-1.58bit-GGUF",
		RemoteFilename: "ggml-model-i2_s.gguf",
		Filename:       "falcon3-1b-instruct-i2_s.gguf",
		SizeMB:         1361,
		ContextSize:    4096,
		Description:    "TII Falcon3 1B Instruct conversational 1.58-bit model (~1.36 GB)",
	},
	"bitnet-2b": {
		Tag:            "bitnet-2b",
		Name:           "BitNet-b1.58-2B-4T",
		Repo:           "microsoft/bitnet-b1.58-2B-4T-gguf",
		RemoteFilename: "ggml-model-i2_s.gguf",
		Filename:       "bitnet-b1.58-2B-4T-i2_s.gguf",
		SizeMB:         1187,
		ContextSize:    2048,
		Description:    "Microsoft BitNet b1.58 2.4B ternary model (~1.18 GB)",
	},
	"falcon3-3b": {
		Tag:            "falcon3-3b",
		Name:           "Falcon3-3B-Instruct-1.58bit",
		Repo:           "tiiuae/Falcon3-3B-Instruct-1.58bit-GGUF",
		RemoteFilename: "ggml-model-i2_s.gguf",
		Filename:       "falcon3-3b-instruct-i2_s.gguf",
		SizeMB:         2800,
		ContextSize:    4096,
		Description:    "TII Falcon3 3B Instruct 1.58-bit model (~2.8 GB)",
	},
}

// GetRootDir returns the standard root configuration directory ~/.bitrun
func GetRootDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".bitrun"
	}
	return filepath.Join(home, ".bitrun")
}

// GetModelsDir returns the path where model weights are stored: ~/.bitrun/models
func GetModelsDir() string {
	return filepath.Join(GetRootDir(), "models")
}

// GetBinDir returns the path where engine binaries are placed: ~/.bitrun/bin
func GetBinDir() string {
	return filepath.Join(GetRootDir(), "bin")
}

// EnsureDirectories ensures that the ~/.bitrun directory tree exists.
func EnsureDirectories() error {
	dirs := []string{
		GetRootDir(),
		GetModelsDir(),
		GetBinDir(),
		filepath.Join(GetRootDir(), "manifests"),
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}
