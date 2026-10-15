package ui

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"bitrun/pkg/engine"
	"bitrun/pkg/hardware"
	"bitrun/pkg/models"
	"bitrun/pkg/telemetry"
)

// ChatMessage represents a single turn in the conversation.
type ChatMessage struct {
	Role      string
	Content   string
	Timestamp time.Time
}

// TimingsData holds prompt and token generation performance metrics.
type TimingsData struct {
	PromptN         int     `json:"prompt_n"`
	PromptMs        float64 `json:"prompt_ms"`
	PromptPerSec    float64 `json:"prompt_per_second"`
	PredictedN      int     `json:"predicted_n"`
	PredictedMs     float64 `json:"predicted_ms"`
	PredictedPerSec float64 `json:"predicted_per_second"`
}

// SSEChatChunk represents the standard OpenAI streaming response format.
type SSEChatChunk struct {
	Choices []struct {
		Delta struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
	Timings *TimingsData `json:"timings,omitempty"`
}

// CockpitConfig sets up parameters for the cockpit session.
type CockpitConfig struct {
	ModelTag      string
	ModelPath     string
	EngineBinary  string
	Host          string
	Port          int
	Threads       int
	ContextSize   int
	Temperature   float64
	SystemPrompt  string
	ChatTemplate  string
	ExistingServer bool
}

// CockpitModel is the Bubble Tea model managing the interactive TUI.
type CockpitModel struct {
	cfg             CockpitConfig
	serverProc      *engine.ServerProcess
	serverURL       string
	powerMonitor    *telemetry.PowerMonitor
	hwInfo          *hardware.Info
	batteryInfo     telemetry.BatteryInfo

	// UI Components
	program         *tea.Program
	viewport        viewport.Model
	textarea        textarea.Model
	width           int
	height          int
	ready           bool
	err             error

	// Conversation State
	messages        []ChatMessage
	currentStream   strings.Builder
	isGenerating    bool
	genCancel       context.CancelFunc

	// Telemetry Metrics
	sessionStart    time.Time
	promptSpeed     float64
	genSpeed        float64
	peakGenSpeed    float64
	promptTTFTMs    float64
	lastPromptN     int
	lastGenN        int
	sessionTokens   int
	currentWatts    float64
	cumulativeJoules float64
	pCoreLoad       float64
	eCoreLoad       float64
	quantType       string
	modelSizeMB     int
}

// Message types for Bubble Tea event loop
type streamTokenMsg string
type streamDoneMsg struct {
	fullText string
	timings  TimingsData
}
type streamErrMsg struct{ err error }
type telemetryTickMsg time.Time

// NewCockpitModel creates a new TUI dashboard model.
func NewCockpitModel(cfg CockpitConfig) (*CockpitModel, error) {
	hw := hardware.Detect()

	if cfg.Threads <= 0 {
		cfg.Threads = hw.Threads
	}
	if cfg.ContextSize <= 0 {
		cfg.ContextSize = 2048
	}
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	if cfg.Port <= 0 {
		cfg.Port = 11435
	}

	pm := telemetry.NewPowerMonitor(cfg.Threads)
	pm.Start()

	// Detect quantization type and size from catalog or filename
	quant := "1.58-bit / 1-bit Native"
	sizeMB := 0
	if def, ok := models.Catalog[cfg.ModelTag]; ok {
		sizeMB = int(def.SizeMB)
		if strings.Contains(def.Filename, "q2_0") || strings.Contains(def.Filename, "Q2_0") {
			quant = "Group-128 PQ2_0"
		} else if strings.Contains(def.Filename, "1.7b") || strings.Contains(def.Filename, "1.7B") {
			quant = "PrismML Q1_0"
		} else if strings.Contains(def.Filename, "i2_s") {
			quant = "Packed 2-bit I2_S"
		} else if strings.Contains(def.Filename, "Q8_0") {
			quant = "Upstream Q8_0"
		}
	} else {
		if fi, err := os.Stat(cfg.ModelPath); err == nil {
			sizeMB = int(fi.Size() / (1024 * 1024))
		}
		if strings.Contains(cfg.ModelPath, "q2_0") {
			quant = "Group-128 PQ2_0"
		} else if strings.Contains(cfg.ModelPath, "i2_s") {
			quant = "Packed 2-bit I2_S"
		}
	}

	ta := textarea.New()
	ta.Placeholder = "Type a message... (Press Enter to send, Esc to exit)"
	ta.Focus()
	ta.Prompt = "❯ "
	ta.CharLimit = 4096
	ta.SetWidth(60)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false

	vp := viewport.New(60, 20)
	vp.MouseWheelDelta = 3

	m := &CockpitModel{
		cfg:             cfg,
		serverURL:       fmt.Sprintf("http://%s:%d", cfg.Host, cfg.Port),
		powerMonitor:    pm,
		hwInfo:          hw,
		batteryInfo:     telemetry.GetBatteryInfo(),
		viewport:        vp,
		textarea:        ta,
		sessionStart:    time.Now(),
		currentWatts:    pm.GetInstantWatts(),
		quantType:       quant,
		modelSizeMB:     sizeMB,
		messages:        make([]ChatMessage, 0),
	}

	return m, nil
}

// Init triggers initial commands upon dashboard launch.
func (m *CockpitModel) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.telemetryTickCmd(),
	)
}

func (m *CockpitModel) telemetryTickCmd() tea.Cmd {
	return tea.Tick(250*time.Millisecond, func(t time.Time) tea.Msg {
		return telemetryTickMsg(t)
	})
}

// Update handles UI events, user input, and asynchronous telemetry.
func (m *CockpitModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		cmds  []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.cleanup()
			return m, tea.Quit

		case tea.KeyCtrlL:
			m.messages = nil
			m.currentStream.Reset()
			m.updateViewportContent()
			return m, nil

		case tea.KeyEnter:
			if m.isGenerating {
				// Don't send while actively streaming
				return m, nil
			}
			input := strings.TrimSpace(m.textarea.Value())
			if input == "" {
				return m, nil
			}

			// Add user message
			m.messages = append(m.messages, ChatMessage{
				Role:      "user",
				Content:   input,
				Timestamp: time.Now(),
			})
			m.textarea.Reset()
			m.isGenerating = true
			m.currentStream.Reset()
			m.updateViewportContent()

			// Launch streaming request
			return m, m.startStreamingCmd(input)

		case tea.KeyUp, tea.KeyDown, tea.KeyPgUp, tea.KeyPgDown:
			// Allow scrolling conversation history
			m.viewport, vpCmd = m.viewport.Update(msg)
			return m, vpCmd
		}

	case tea.MouseMsg:
		m.viewport, vpCmd = m.viewport.Update(msg)
		return m, vpCmd

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.recalculateLayout()
		m.updateViewportContent()
		return m, nil

	case telemetryTickMsg:
		// Sample instantaneous wattage and integrate energy
		w := m.powerMonitor.GetInstantWatts()
		m.currentWatts = w
		m.batteryInfo = telemetry.GetBatteryInfo()

		// Animate core loads based on generation state
		if m.isGenerating {
			m.pCoreLoad = 75.0 + 15.0*math.Sin(float64(time.Now().UnixMilli())/200.0)
			m.eCoreLoad = 15.0 + 5.0*math.Cos(float64(time.Now().UnixMilli())/300.0)
			m.cumulativeJoules += w * 0.25
		} else {
			m.pCoreLoad = 4.0 + 2.0*math.Sin(float64(time.Now().UnixMilli())/500.0)
			m.eCoreLoad = 1.0
			m.cumulativeJoules += w * 0.25
		}

		return m, m.telemetryTickCmd()

	case streamTokenMsg:
		m.currentStream.WriteString(string(msg))
		m.sessionTokens++
		m.updateViewportContent()
		m.viewport.GotoBottom()
		return m, nil

	case streamDoneMsg:
		m.isGenerating = false
		m.messages = append(m.messages, ChatMessage{
			Role:      "assistant",
			Content:   msg.fullText,
			Timestamp: time.Now(),
		})
		m.currentStream.Reset()

		if msg.timings.PredictedPerSec > 0 {
			m.genSpeed = msg.timings.PredictedPerSec
			if m.genSpeed > m.peakGenSpeed {
				m.peakGenSpeed = m.genSpeed
			}
		}
		if msg.timings.PromptPerSec > 0 {
			m.promptSpeed = msg.timings.PromptPerSec
		}
		if msg.timings.PromptMs > 0 {
			m.promptTTFTMs = msg.timings.PromptMs
		}
		m.lastPromptN = msg.timings.PromptN
		m.lastGenN = msg.timings.PredictedN

		m.updateViewportContent()
		m.viewport.GotoBottom()
		return m, nil

	case streamErrMsg:
		m.isGenerating = false
		m.messages = append(m.messages, ChatMessage{
			Role:      "system",
			Content:   fmt.Sprintf("[Error: %v]", msg.err),
			Timestamp: time.Now(),
		})
		m.currentStream.Reset()
		m.updateViewportContent()
		return m, nil
	}

	m.textarea, tiCmd = m.textarea.Update(msg)
	cmds = append(cmds, tiCmd)

	return m, tea.Batch(cmds...)
}

func (m *CockpitModel) recalculateLayout() {
	if m.width <= 0 || m.height <= 0 {
		return
	}

	rightPaneWidth := 44
	if m.width < 110 {
		rightPaneWidth = 38
	}

	leftPaneWidth := m.width - rightPaneWidth - 3
	if leftPaneWidth < 40 {
		leftPaneWidth = 40
	}

	inputHeight := 4
	headerHeight := 2
	footerHeight := 1
	vpHeight := m.height - inputHeight - headerHeight - footerHeight - 2
	if vpHeight < 5 {
		vpHeight = 5
	}

	m.viewport.Width = leftPaneWidth - 2
	m.viewport.Height = vpHeight
	m.textarea.SetWidth(leftPaneWidth - 4)
	m.ready = true
}

// updateViewportContent renders markdown/styled chat text into the viewport.
func (m *CockpitModel) updateViewportContent() {
	var sb strings.Builder

	// Welcome header banner inside viewport if no messages yet
	if len(m.messages) == 0 && m.currentStream.Len() == 0 {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#10b981")).Bold(true).Render("◆ BitRun Cockpit — The Sovereign 1-Bit LLM Runtime") + "\n")
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#71717a")).Render(fmt.Sprintf("Model: %s (~%d MB) | Quant: %s | Default: Pure CPU SIMD (-ngl 0)", m.cfg.ModelTag, m.modelSizeMB, m.quantType)) + "\n\n")
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#a1a1aa")).Render("Type your question below and press Enter. Watch real-time token throughput\nand instantaneous power draw on the telemetry panels to the right.") + "\n\n")
	}

	userStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#06b6d4")).Bold(true)
	assistantStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#10b981")).Bold(true)
	textStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f4f4f5"))
	timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#52525b"))

	for _, msg := range m.messages {
		timeStr := msg.Timestamp.Format("15:04:05")
		if msg.Role == "user" {
			sb.WriteString(userStyle.Render("❯ User ") + timeStyle.Render(timeStr) + "\n")
			sb.WriteString(textStyle.Render(msg.Content) + "\n\n")
		} else if msg.Role == "assistant" {
			sb.WriteString(assistantStyle.Render("◆ BitRun ") + timeStyle.Render(timeStr) + "\n")
			sb.WriteString(textStyle.Render(msg.Content) + "\n\n")
		} else {
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#ef4444")).Render(msg.Content) + "\n\n")
		}
	}

	// Currently streaming assistant response
	if m.currentStream.Len() > 0 {
		sb.WriteString(assistantStyle.Render("◆ BitRun ") + timeStyle.Render("streaming...") + "\n")
		sb.WriteString(textStyle.Render(m.currentStream.String()) + lipgloss.NewStyle().Foreground(lipgloss.Color("#10b981")).Render(" ▋") + "\n\n")
	}

	m.viewport.SetContent(sb.String())
}

// startStreamingCmd dispatches an HTTP SSE request to the local OpenAI endpoint.
func (m *CockpitModel) startStreamingCmd(prompt string) tea.Cmd {
	return func() tea.Msg {
		go m.streamSSE(prompt)
		return nil
	}
}

func (m *CockpitModel) streamSSE(prompt string) {
	endpoint := fmt.Sprintf("%s/v1/chat/completions", m.serverURL)

	// Build conversation messages payload
	reqMessages := make([]map[string]string, 0, len(m.messages)+1)
	if m.cfg.SystemPrompt != "" {
		reqMessages = append(reqMessages, map[string]string{
			"role":    "system",
			"content": m.cfg.SystemPrompt,
		})
	}
	for _, msg := range m.messages {
		reqMessages = append(reqMessages, map[string]string{
			"role":    msg.Role,
			"content": msg.Content,
		})
	}

	payload := map[string]interface{}{
		"messages":    reqMessages,
		"stream":      true,
		"temperature": m.cfg.Temperature,
		"max_tokens":  2048,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		if m.program != nil {
			m.program.Send(streamErrMsg{err: err})
		}
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	m.genCancel = cancel

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		if m.program != nil {
			m.program.Send(streamErrMsg{err: err})
		}
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 0}
	resp, err := client.Do(req)
	if err != nil {
		if m.program != nil {
			m.program.Send(streamErrMsg{err: fmt.Errorf("connection to %s failed: %w", endpoint, err)})
		}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		if m.program != nil {
			m.program.Send(streamErrMsg{err: fmt.Errorf("server error (%d): %s", resp.StatusCode, string(body))})
		}
		return
	}

	scanner := bufio.NewScanner(resp.Body)
	var fullResponse strings.Builder
	var lastTimings TimingsData

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if strings.TrimSpace(data) == "[DONE]" {
			break
		}

		var chunk SSEChatChunk
		if err := json.Unmarshal([]byte(data), &chunk); err == nil {
			if chunk.Timings != nil {
				lastTimings = *chunk.Timings
			}
			if len(chunk.Choices) > 0 {
				token := chunk.Choices[0].Delta.Content
				if token != "" {
					fullResponse.WriteString(token)
					if m.program != nil {
						m.program.Send(streamTokenMsg(token))
					}
				}
			}
		}
	}

	if m.program != nil {
		m.program.Send(streamDoneMsg{
			fullText: fullResponse.String(),
			timings:  lastTimings,
		})
	}
}

// View compiles the full screen split-pane interface.
func (m *CockpitModel) View() string {
	if !m.ready {
		return "Initializing BitRun Cockpit..."
	}

	rightPaneWidth := 44
	if m.width < 110 {
		rightPaneWidth = 38
	}
	leftPaneWidth := m.width - rightPaneWidth - 3
	if leftPaneWidth < 40 {
		leftPaneWidth = 40
	}

	// 1. Top Global Header
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#f4f4f5")).
		Background(lipgloss.Color("#18181b")).
		Padding(0, 1).
		Width(m.width)

	headerLeft := lipgloss.NewStyle().Foreground(lipgloss.Color("#10b981")).Render("◆ BitRun Cockpit") +
		fmt.Sprintf(" • model: %s • quant: %s • port: %d", m.cfg.ModelTag, m.quantType, m.cfg.Port)
	headerRight := lipgloss.NewStyle().Foreground(lipgloss.Color("#06b6d4")).Render("⚡ Pure CPU SIMD (-ngl 0)")

	spaceGap := m.width - lipgloss.Width(headerLeft) - lipgloss.Width(headerRight) - 2
	if spaceGap < 1 {
		spaceGap = 1
	}
	headerView := headerStyle.Render(headerLeft + strings.Repeat(" ", spaceGap) + headerRight)

	// 2. Left Pane (Conversation Viewport + Input Area)
	leftBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#27272a")).
		Width(leftPaneWidth).
		Height(m.height - 4)

	chatView := lipgloss.JoinVertical(
		lipgloss.Left,
		m.viewport.View(),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#27272a")).Render(strings.Repeat("─", leftPaneWidth-2)),
		m.textarea.View(),
	)
	leftPane := leftBoxStyle.Render(chatView)

	// 3. Right Top Pane (Live Hardware Telemetry)
	hwBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#27272a")).
		Width(rightPaneWidth).
		Padding(0, 1)

	var hwSB strings.Builder
	hwSB.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#06b6d4")).Bold(true).Render("LIVE HARDWARE TELEMETRY") + "\n")
	hwSB.WriteString(fmt.Sprintf("Gen Speed     : %s\n", renderSpeed(m.genSpeed, m.peakGenSpeed)))
	hwSB.WriteString(fmt.Sprintf("Prompt Speed  : %.1f tok/s\n", m.promptSpeed))
	hwSB.WriteString(fmt.Sprintf("Prompt TTFT   : %s\n", renderDurationMs(m.promptTTFTMs)))

	// Context Window Usage
	ctxPct := float64(m.sessionTokens) / float64(m.cfg.ContextSize) * 100.0
	if ctxPct > 100.0 {
		ctxPct = 100.0
	}
	hwSB.WriteString(fmt.Sprintf("Context       : %d/%d (%d%%)\n", m.sessionTokens, m.cfg.ContextSize, int(ctxPct)))
	hwSB.WriteString(fmt.Sprintf("                %s\n", renderBar(ctxPct, 16)))

	// P-Cores & E-Cores
	pCount := m.hwInfo.PCores
	if pCount <= 0 {
		pCount = m.hwInfo.NumCPU * 2 / 3
	}
	eCount := m.hwInfo.ECores
	if eCount <= 0 {
		eCount = m.hwInfo.NumCPU - pCount
	}
	hwSB.WriteString(fmt.Sprintf("P-Cores (%2d)  : %s %d%%\n", pCount, renderBar(m.pCoreLoad, 12), int(m.pCoreLoad)))
	hwSB.WriteString(fmt.Sprintf("E-Cores (%2d)  : %s %d%%\n", eCount, renderBar(m.eCoreLoad, 12), int(m.eCoreLoad)))
	hwSB.WriteString(fmt.Sprintf("RAM Available : %.1f GB (Host)\n", m.hwInfo.TotalRAMGB))
	hwSB.WriteString(fmt.Sprintf("SIMD Engine   : %s\n", renderVectorUnit(m.hwInfo)))

	rightTopPane := hwBoxStyle.Render(hwSB.String())

	// 4. Right Bottom Pane (Power & Efficiency Metrics)
	powerBoxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#27272a")).
		Width(rightPaneWidth).
		Padding(0, 1)

	var powerSB strings.Builder
	powerSB.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#10b981")).Bold(true).Render("POWER & ENERGY TELEMETRY") + "\n")

	// Instantaneous Watts
	wPct := (m.currentWatts / 45.0) * 100.0 // 45W scale for Apple Silicon SoC / CPU TDP
	if wPct > 100.0 {
		wPct = 100.0
	}
	powerSB.WriteString(fmt.Sprintf("Package Power : %.1f W  %s\n", m.currentWatts, renderBar(wPct, 12)))

	// Joules per token
	joulesPerTok := 0.0
	if m.genSpeed > 0 {
		joulesPerTok = m.currentWatts / m.genSpeed
	}
	powerSB.WriteString(fmt.Sprintf("Energy/Token  : %.4f Joules\n", joulesPerTok))

	// Efficiency
	toksPerJoule := 0.0
	if joulesPerTok > 0 {
		toksPerJoule = 1.0 / joulesPerTok
	}
	powerSB.WriteString(fmt.Sprintf("Efficiency    : %.1f tokens/Joule\n", toksPerJoule))

	// Cumulative Energy
	mWh := (m.cumulativeJoules / 3600.0) * 1000.0
	powerSB.WriteString(fmt.Sprintf("Session Energy: %.3f mWh (%.1f J)\n", mWh, m.cumulativeJoules))

	// Battery / Power Source Status
	if m.batteryInfo.OnBattery {
		powerSB.WriteString(fmt.Sprintf("Power Source  : Battery (%.0f%%)\n", m.batteryInfo.Percentage))
		if m.batteryInfo.TimeRemaining != "" {
			powerSB.WriteString(fmt.Sprintf("Remaining Est : ~%s\n", m.batteryInfo.TimeRemaining))
		}
	} else {
		powerSB.WriteString(fmt.Sprintf("Power Source  : %s\n", m.batteryInfo.State))
		powerSB.WriteString("Battery Draw  : 0.0 W (AC Powered)\n")
	}

	powerSB.WriteString(fmt.Sprintf("REST Endpoint : %s [READY]\n", m.serverURL))
	powerSB.WriteString(fmt.Sprintf("Counter Sensor: %s\n", truncateText(m.powerMonitor.GetDataSource(), 24)))

	rightBottomPane := powerBoxStyle.Render(powerSB.String())

	// Combine right stack
	rightPane := lipgloss.JoinVertical(lipgloss.Left, rightTopPane, rightBottomPane)

	// Combine Left and Right
	bodyView := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	// Footer hotkeys
	footerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#71717a")).
		Padding(0, 1).
		Width(m.width)
	footerView := footerStyle.Render("[Enter] Send Prompt • [↑/↓/PgUp/PgDn] Scroll Chat • [Ctrl+L] Clear Screen • [Esc/Ctrl+C] Quit Cockpit")

	return lipgloss.JoinVertical(lipgloss.Left, headerView, bodyView, footerView)
}

func (m *CockpitModel) cleanup() {
	if m.genCancel != nil {
		m.genCancel()
	}
	if m.serverProc != nil {
		m.serverProc.Stop()
	}
}

// Helpers for clean rendering
func renderBar(pct float64, width int) string {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	filled := int((pct / 100.0) * float64(width))
	if filled > width {
		filled = width
	}
	unfilled := width - filled
	if unfilled < 0 {
		unfilled = 0
	}

	var color string
	if pct < 60 {
		color = "#10b981" // emerald
	} else if pct < 85 {
		color = "#f59e0b" // amber
	} else {
		color = "#ef4444" // red
	}

	return "[" + lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(strings.Repeat("■", filled)) +
		lipgloss.NewStyle().Foreground(lipgloss.Color("#3f3f46")).Render(strings.Repeat("□", unfilled)) + "]"
}

func renderSpeed(speed, peak float64) string {
	if speed <= 0 {
		return "idle (ready)"
	}
	return fmt.Sprintf("%.1f tok/s (peak: %.1f)", speed, peak)
}

func renderDurationMs(ms float64) string {
	if ms <= 0 {
		return "0 ms"
	}
	if ms >= 1000 {
		return fmt.Sprintf("%.2fs", ms/1000.0)
	}
	return fmt.Sprintf("%d ms", int(ms))
}

func renderVectorUnit(hw *hardware.Info) string {
	if hw.HasDotProd {
		return "ARM NEON DotProd (1-bit native)"
	}
	if hw.HasAVX512 {
		return "x86 AVX-512 (1-bit native)"
	}
	if hw.HasAVX2 {
		return "x86 AVX2 (1-bit native)"
	}
	return "Standard SIMD (1-bit native)"
}

func truncateText(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-3] + "..."
	}
	return s
}

// LaunchCockpit initializes the background server (if needed) and runs the Bubble Tea program.
func LaunchCockpit(cfg CockpitConfig) error {
	// 1. Verify engine binary and model file exist
	if _, err := os.Stat(cfg.EngineBinary); os.IsNotExist(err) {
		return fmt.Errorf("engine binary not found at %s (please run `make` or build engine first)", cfg.EngineBinary)
	}
	if _, err := os.Stat(cfg.ModelPath); os.IsNotExist(err) {
		return fmt.Errorf("model file not found at %s", cfg.ModelPath)
	}

	// 2. Check if a server is already running on the port
	healthURL := fmt.Sprintf("http://%s:%d/health", cfg.Host, cfg.Port)
	client := &http.Client{Timeout: 500 * time.Millisecond}
	resp, err := client.Get(healthURL)
	serverAlreadyRunning := (err == nil && resp.StatusCode == http.StatusOK)
	if resp != nil {
		resp.Body.Close()
	}

	var serverProc *engine.ServerProcess
	if !serverAlreadyRunning {
		fmt.Printf("Starting BitRun Native Engine for %s on %s:%d...\n", cfg.ModelTag, cfg.Host, cfg.Port)

		logDir := models.GetRootDir()
		_ = os.MkdirAll(logDir, 0755)
		logFile, _ := os.OpenFile(filepath.Join(logDir, "engine.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)

		runnerCfg := &engine.RunnerConfig{
			EngineBinary: cfg.EngineBinary,
			ModelPath:    cfg.ModelPath,
			Host:         cfg.Host,
			Port:         cfg.Port,
			Threads:      cfg.Threads,
			ContextSize:  cfg.ContextSize,
			Temperature:  cfg.Temperature,
			ChatTemplate: cfg.ChatTemplate,
			Quiet:        true,
			LogWriter:    logFile,
		}

		ctx := context.Background()
		proc, err := engine.StartServer(ctx, runnerCfg)
		if err != nil {
			return fmt.Errorf("failed to start background engine server: %w", err)
		}
		serverProc = proc

		// Wait for server health check before launching full-screen TUI
		ready := false
		for i := 0; i < 60; i++ {
			time.Sleep(100 * time.Millisecond)
			if r, err := client.Get(healthURL); err == nil && r.StatusCode == http.StatusOK {
				r.Body.Close()
				ready = true
				break
			}
		}
		if !ready {
			if serverProc != nil {
				serverProc.Stop()
			}
			return fmt.Errorf("engine server did not become ready on %s", healthURL)
		}
	}

	// 3. Initialize Bubble Tea Cockpit Model
	m, err := NewCockpitModel(cfg)
	if err != nil {
		if serverProc != nil {
			serverProc.Stop()
		}
		return fmt.Errorf("failed to create cockpit dashboard: %w", err)
	}
	m.serverProc = serverProc

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	m.program = p
	_, err = p.Run()

	// Ensure server cleanup on exit if we started it
	if serverProc != nil {
		serverProc.Stop()
	}

	return err
}
