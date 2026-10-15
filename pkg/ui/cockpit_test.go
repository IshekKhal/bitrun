package ui

import (
	"strings"
	"testing"

	"bitrun/pkg/hardware"
	"bitrun/pkg/telemetry"
)

func TestCockpitModelInitialization(t *testing.T) {
	cfg := CockpitConfig{
		ModelTag:     "bonsai-1.7b",
		ModelPath:    "/tmp/dummy.gguf",
		EngineBinary: "/tmp/dummy-engine",
		Host:         "127.0.0.1",
		Port:         11435,
		Threads:      4,
		ContextSize:  2048,
	}

	model, err := NewCockpitModel(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize CockpitModel: %v", err)
	}

	if model.cfg.ModelTag != "bonsai-1.7b" {
		t.Errorf("Expected model tag 'bonsai-1.7b', got '%s'", model.cfg.ModelTag)
	}

	if model.quantType != "PrismML Q1_0" {
		t.Errorf("Expected quant 'PrismML Q1_0', got '%s'", model.quantType)
	}
}

func TestRenderBar(t *testing.T) {
	bar0 := renderBar(0, 10)
	if !strings.Contains(bar0, "□□□□□□□□□□") {
		t.Errorf("renderBar(0) failed: %s", bar0)
	}

	bar100 := renderBar(100, 10)
	if !strings.Contains(bar100, "■■■■■■■■■■") {
		t.Errorf("renderBar(100) failed: %s", bar100)
	}

	bar50 := renderBar(50, 10)
	if !strings.Contains(bar50, "■■■■■") || !strings.Contains(bar50, "□□□□□") {
		t.Errorf("renderBar(50) failed: %s", bar50)
	}
}

func TestRenderDurationMs(t *testing.T) {
	if s := renderDurationMs(0); s != "0 ms" {
		t.Errorf("Expected '0 ms', got '%s'", s)
	}
	if s := renderDurationMs(250); s != "250 ms" {
		t.Errorf("Expected '250 ms', got '%s'", s)
	}
	if s := renderDurationMs(1500); s != "1.50s" {
		t.Errorf("Expected '1.50s', got '%s'", s)
	}
}

func TestRenderSpeed(t *testing.T) {
	if s := renderSpeed(0, 0); s != "idle (ready)" {
		t.Errorf("Expected 'idle (ready)', got '%s'", s)
	}
	if s := renderSpeed(42.5, 48.0); s != "42.5 tok/s (peak: 48.0)" {
		t.Errorf("Expected '42.5 tok/s (peak: 48.0)', got '%s'", s)
	}
}

func TestRecalculateLayout(t *testing.T) {
	cfg := CockpitConfig{
		ModelTag:     "bitnet-2b",
		ModelPath:    "/tmp/dummy.gguf",
		EngineBinary: "/tmp/dummy-engine",
	}

	model, err := NewCockpitModel(cfg)
	if err != nil {
		t.Fatalf("Failed to initialize CockpitModel: %v", err)
	}

	model.width = 120
	model.height = 36
	model.recalculateLayout()

	if !model.ready {
		t.Error("Expected model.ready to be true after recalculateLayout")
	}

	if model.viewport.Width <= 0 || model.viewport.Height <= 0 {
		t.Errorf("Invalid viewport dimensions: %dx%d", model.viewport.Width, model.viewport.Height)
	}

	// Verify view rendering does not crash
	view := model.View()
	if !strings.Contains(view, "BitRun Cockpit") {
		t.Error("View output missing 'BitRun Cockpit'")
	}
	if !strings.Contains(view, "LIVE HARDWARE TELEMETRY") {
		t.Error("View output missing 'LIVE HARDWARE TELEMETRY'")
	}
	if !strings.Contains(view, "POWER & ENERGY TELEMETRY") {
		t.Error("View output missing 'POWER & ENERGY TELEMETRY'")
	}
}

func TestRenderVectorUnit(t *testing.T) {
	hw := &hardware.Info{HasDotProd: true}
	if s := renderVectorUnit(hw); !strings.Contains(s, "ARM NEON DotProd") {
		t.Errorf("Expected ARM NEON DotProd, got '%s'", s)
	}

	hw2 := &hardware.Info{HasAVX512: true}
	if s := renderVectorUnit(hw2); !strings.Contains(s, "AVX-512") {
		t.Errorf("Expected AVX-512, got '%s'", s)
	}

	hw3 := &hardware.Info{HasAVX2: true}
	if s := renderVectorUnit(hw3); !strings.Contains(s, "AVX2") {
		t.Errorf("Expected AVX2, got '%s'", s)
	}
}

func TestBatteryInfoStructure(t *testing.T) {
	info := telemetry.GetBatteryInfo()
	if info.Percentage < 0 || info.Percentage > 100 {
		t.Errorf("Invalid battery percentage: %.2f", info.Percentage)
	}
	if info.State == "" {
		t.Error("Battery state should not be empty")
	}
}
