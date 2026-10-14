package telemetry

import (
	"strings"
	"testing"
	"time"
)

func TestPowerMonitor(t *testing.T) {
	pm := NewPowerMonitor(4)
	pm.Start()

	// Simulate work for 250ms
	time.Sleep(250 * time.Millisecond)

	metrics := pm.Stop(32, 64, 50*time.Millisecond, 200*time.Millisecond)

	if metrics.PromptTokens != 32 {
		t.Errorf("expected 32 prompt tokens, got %d", metrics.PromptTokens)
	}
	if metrics.GenTokens != 64 {
		t.Errorf("expected 64 gen tokens, got %d", metrics.GenTokens)
	}
	if metrics.AvgWatts <= 0 {
		t.Errorf("expected AvgWatts > 0, got %f", metrics.AvgWatts)
	}
	if metrics.TotalJoules <= 0 {
		t.Errorf("expected TotalJoules > 0, got %f", metrics.TotalJoules)
	}
	if metrics.GenSpeed <= 0 {
		t.Errorf("expected GenSpeed > 0, got %f", metrics.GenSpeed)
	}
	if metrics.JoulesPerToken <= 0 {
		t.Errorf("expected JoulesPerToken > 0, got %f", metrics.JoulesPerToken)
	}

	box := FormatTelemetryBox(metrics)
	if !strings.Contains(box, "BitRun Power & Performance Telemetry") {
		t.Errorf("expected telemetry box header, got:\n%s", box)
	}
	if !strings.Contains(box, "Energy Efficiency") {
		t.Errorf("expected Energy Efficiency row, got:\n%s", box)
	}
}
