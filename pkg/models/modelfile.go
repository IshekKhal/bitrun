package models

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Manifest defines the persistent JSON metadata for custom models created from Modelfiles.
type Manifest struct {
	Name         string                 `json:"name"`
	BaseModel    string                 `json:"base_model"`
	SystemPrompt string                 `json:"system_prompt,omitempty"`
	Template     string                 `json:"template,omitempty"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
	CreatedAt    string                 `json:"created_at,omitempty"`
}

// GetManifestsDir returns the path where custom model manifests are stored: ~/.bitrun/manifests
func GetManifestsDir() string {
	return filepath.Join(GetRootDir(), "manifests")
}

// SaveManifest writes the manifest JSON to ~/.bitrun/manifests/<name>.json.
func SaveManifest(m *Manifest) error {
	if m.Name == "" {
		return fmt.Errorf("manifest name cannot be empty")
	}
	if err := EnsureDirectories(); err != nil {
		return err
	}
	if m.CreatedAt == "" {
		m.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	manifestPath := filepath.Join(GetManifestsDir(), m.Name+".json")
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode manifest: %w", err)
	}

	return os.WriteFile(manifestPath, data, 0644)
}

// LoadManifest reads and parses a manifest JSON from ~/.bitrun/manifests/<name>.json.
func LoadManifest(name string) (*Manifest, error) {
	manifestPath := filepath.Join(GetManifestsDir(), name+".json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, err
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("invalid manifest JSON in %s: %w", manifestPath, err)
	}

	return &m, nil
}

// ManifestExists returns true if a manifest with the specified name exists.
func ManifestExists(name string) bool {
	manifestPath := filepath.Join(GetManifestsDir(), name+".json")
	_, err := os.Stat(manifestPath)
	return err == nil
}

// ListManifests returns all custom model manifests saved in ~/.bitrun/manifests.
func ListManifests() ([]*Manifest, error) {
	dir := GetManifestsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var manifests []*Manifest
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			name := strings.TrimSuffix(entry.Name(), ".json")
			m, err := LoadManifest(name)
			if err == nil && m != nil {
				manifests = append(manifests, m)
			}
		}
	}
	return manifests, nil
}

// DeleteManifest removes a manifest file by name.
func DeleteManifest(name string) error {
	manifestPath := filepath.Join(GetManifestsDir(), name+".json")
	return os.Remove(manifestPath)
}

// ParseModelfile reads a Modelfile format from an io.Reader and returns a Manifest.
func ParseModelfile(r io.Reader) (*Manifest, error) {
	scanner := bufio.NewScanner(r)
	manifest := &Manifest{
		Parameters: make(map[string]interface{}),
	}

	var inBlock string // "SYSTEM" or "TEMPLATE"
	var blockDelim string
	var blockBuilder strings.Builder

	for scanner.Scan() {
		line := scanner.Text()

		// Handle multi-line block accumulation (e.g. triple quotes)
		if inBlock != "" {
			if strings.Contains(line, blockDelim) {
				idx := strings.Index(line, blockDelim)
				blockBuilder.WriteString(line[:idx])
				content := strings.TrimSpace(blockBuilder.String())
				if inBlock == "SYSTEM" {
					manifest.SystemPrompt = content
				} else if inBlock == "TEMPLATE" {
					manifest.Template = content
				}
				inBlock = ""
				blockDelim = ""
				blockBuilder.Reset()
				continue
			}
			blockBuilder.WriteString(line)
			blockBuilder.WriteString("\n")
			continue
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		parts := strings.SplitN(trimmed, " ", 2)
		cmd := strings.ToUpper(parts[0])
		var rest string
		if len(parts) > 1 {
			rest = strings.TrimSpace(parts[1])
		}

		switch cmd {
		case "FROM":
			manifest.BaseModel = strings.Trim(rest, "\"'")

		case "SYSTEM":
			if strings.HasPrefix(rest, `"""`) || strings.HasPrefix(rest, `'''`) {
				delim := rest[:3]
				rest = rest[3:]
				if idx := strings.Index(rest, delim); idx != -1 {
					// Closed on the same line
					manifest.SystemPrompt = strings.TrimSpace(rest[:idx])
				} else {
					inBlock = "SYSTEM"
					blockDelim = delim
					blockBuilder.WriteString(rest)
					blockBuilder.WriteString("\n")
				}
			} else {
				if manifest.SystemPrompt != "" {
					manifest.SystemPrompt += "\n" + rest
				} else {
					manifest.SystemPrompt = rest
				}
			}

		case "TEMPLATE":
			if strings.HasPrefix(rest, `"""`) || strings.HasPrefix(rest, `'''`) {
				delim := rest[:3]
				rest = rest[3:]
				if idx := strings.Index(rest, delim); idx != -1 {
					manifest.Template = strings.TrimSpace(rest[:idx])
				} else {
					inBlock = "TEMPLATE"
					blockDelim = delim
					blockBuilder.WriteString(rest)
					blockBuilder.WriteString("\n")
				}
			} else {
				manifest.Template = rest
			}

		case "PARAMETER":
			paramParts := strings.SplitN(rest, " ", 2)
			if len(paramParts) < 2 {
				return nil, fmt.Errorf("malformed PARAMETER directive: %q", line)
			}
			key := strings.ToLower(strings.TrimSpace(paramParts[0]))
			valStr := strings.TrimSpace(paramParts[1])
			valStr = strings.Trim(valStr, "\"'")

			// Normalize parameter keys
			switch key {
			case "num_ctx", "context", "ctx_size":
				key = "context_size"
			case "temp":
				key = "temperature"
			}

			// Value type parsing
			if intVal, err := strconv.Atoi(valStr); err == nil {
				manifest.Parameters[key] = intVal
			} else if floatVal, err := strconv.ParseFloat(valStr, 64); err == nil {
				manifest.Parameters[key] = floatVal
			} else if boolVal, err := strconv.ParseBool(valStr); err == nil {
				manifest.Parameters[key] = boolVal
			} else {
				manifest.Parameters[key] = valStr
			}
		}
	}

	if inBlock != "" {
		return nil, fmt.Errorf("unclosed %s block delimiter %s", inBlock, blockDelim)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading Modelfile: %w", err)
	}

	if manifest.BaseModel == "" {
		return nil, fmt.Errorf("missing required 'FROM' instruction in Modelfile")
	}

	return manifest, nil
}

// ParseModelfilePath opens a Modelfile at the given path and parses it.
func ParseModelfilePath(path string) (*Manifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open Modelfile at %s: %w", path, err)
	}
	defer file.Close()

	return ParseModelfile(file)
}
