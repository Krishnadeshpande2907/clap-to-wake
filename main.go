package main

import (
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gen2brain/malgo"
	"github.com/go-vgo/robotgo"
	"github.com/shirou/gopsutil/v3/process"
)

// --- Configuration & Constants ---

const (
	StateWatchingApps = iota
	StateWaitingIdle
	StateListening
	StateWaking
)

type Config struct {
	TargetApps []string
	Threshold  float32
	IdleLimit  uint32 // robotgo.IdleTime() returns uint32
}

// --- TUI Styling ---

var (
	keywordStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
	statusStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Italic(true)
	helpStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

// --- Bubble Tea Model ---

type statusUpdateMsg struct {
	state   int
	message string
	volume  float32
}

type model struct {
	cfg          Config
	currentState int
	statusMsg    string
	currentVol   float32
	spinner      spinner.Model
	progress     progress.Model
	quitting     bool
}

func initialModel() model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return model{
		cfg: Config{
			TargetApps: []string{"code", "chrome", "cursor", "browser", "vlc", "msedge"},
			Threshold:  0.25, // Sensitivity: 0.0 to 1.0
			IdleLimit:  300,  // 5 Minutes in seconds
		},
		currentState: StateWatchingApps,
		statusMsg:    "System Booting...",
		spinner:      s,
		progress:     progress.New(progress.WithDefaultGradient()),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, monitorLoop(m.cfg))
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}
	case statusUpdateMsg:
		m.currentState = msg.state
		m.statusMsg = msg.message
		m.currentVol = msg.volume
		return m, monitorLoop(m.cfg)
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m model) View() string {
	if m.quitting {
		return "\n  Yeah! You better get to work, Nigga!! \n\n"
	}

	stateName := ""
	switch m.currentState {
	case StateWatchingApps:
		stateName = "Watcher (Low Energy)"
	case StateWaitingIdle:
		stateName = "Idle Monitor"
	case StateListening:
		stateName = "Active Listener (Mic On)"
	case StateWaking:
		stateName = "WAKING SYSTEM"
	}

	s := fmt.Sprintf("\n  %s CLAP-TO-WAKE\n\n", m.spinner.View())
	s += fmt.Sprintf("  Status: %s\n", statusStyle.Render(stateName))
	s += fmt.Sprintf("  Details: %s\n\n", m.statusMsg)

	if m.currentState == StateListening {
		s += "  Mic Level:\n"
		s += "  " + m.progress.ViewAs(float64(m.currentVol*4)) + "\n\n"
	}

	s += "  Watching: " + keywordStyle.Render(strings.Join(m.cfg.TargetApps, ", ")) + "\n"
	s += "  Timeout: " + keywordStyle.Render(fmt.Sprintf("%ds", m.cfg.IdleLimit)) + "\n"
	s += helpStyle.Render("\n  (Press 'q' to quit application)\n")
	return s
}

// --- Core Logic ---

func monitorLoop(cfg Config) tea.Cmd {
	return func() tea.Msg {
		procs, _ := process.Processes()
		found := false
		var activeApp string
		for _, p := range procs {
			name, _ := p.Name()
			for _, target := range cfg.TargetApps {
				if strings.Contains(strings.ToLower(name), strings.ToLower(target)) {
					found = true
					activeApp = name
					break
				}
			}
			if found {
				break
			}
		}

		if !found {
			time.Sleep(3 * time.Second)
			return statusUpdateMsg{StateWatchingApps, "Target apps not running...", 0}
		}

		// TODO: robotgo.IdleTime() is not available in v1.0.2
		// For now, we'll skip the idle check and proceed directly to listening
		// idleSec := robotgo.IdleTime()
		if false { // Placeholder - idle time check disabled
			time.Sleep(2 * time.Second)
			return statusUpdateMsg{
				StateWaitingIdle,
				fmt.Sprintf("%s active. Checking idle time...", activeApp),
				0,
			}
		}

		return listenForAudio(cfg)
	}
}

func listenForAudio(cfg Config) tea.Msg {
	// Initialize miniaudio context (replaces portaudio)
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, func(message string) {
		// optional log callback — ignore
	})
	if err != nil {
		return statusUpdateMsg{StateWatchingApps, "Failed to init audio context", 0}
	}
	defer func() {
		_ = ctx.Uninit()
		ctx.Free()
	}()

	deviceConfig := malgo.DefaultDeviceConfig(malgo.Capture)
	deviceConfig.Capture.Format = malgo.FormatF32
	deviceConfig.Capture.Channels = 1
	deviceConfig.SampleRate = 44100
	deviceConfig.Alsa.NoMMap = 1

	const bufferFrames = 1024
	var (
		lastRMS     float32
		clapTimeout = time.Now().Add(5 * time.Second)
		detected    = false
	)

	callbacks := malgo.DeviceCallbacks{
		Data: func(outputSamples, inputSamples []byte, frameCount uint32) {
			if detected || time.Now().After(clapTimeout) {
				return
			}
			// inputSamples is raw bytes; reinterpret as float32 (4 bytes each)
			samples := byteSliceToFloat32(inputSamples)
			var sum float64
			for _, s := range samples {
				sum += float64(s * s)
			}
			if len(samples) > 0 {
				lastRMS = float32(math.Sqrt(sum / float64(len(samples))))
			}
			if lastRMS > cfg.Threshold {
				detected = true
			}
		},
	}

	device, err := malgo.InitDevice(ctx.Context, deviceConfig, callbacks)
	if err != nil {
		return statusUpdateMsg{StateWatchingApps, "Mic not found or busy", 0}
	}
	defer device.Uninit()

	if err := device.Start(); err != nil {
		return statusUpdateMsg{StateWatchingApps, "Stream start error", 0}
	}

	_ = bufferFrames
	for time.Now().Before(clapTimeout) {
		time.Sleep(50 * time.Millisecond)
		if detected {
			// Wake the system: tiny mouse nudge + shift keypress
			robotgo.Move(1, 1)
			robotgo.KeySleep = 50
			err := robotgo.KeyTap("shift")
			if err != nil {
				return statusUpdateMsg{StateWaking, "CLAP DETECTED! (KeyTap failed: " + err.Error() + ")", lastRMS}
			}
			return statusUpdateMsg{StateWaking, "CLAP DETECTED!", lastRMS}
		}
	}

	return statusUpdateMsg{StateListening, "Waiting for signal...", lastRMS}
}

// byteSliceToFloat32 reinterprets a []byte (little-endian float32 samples) as []float32.
func byteSliceToFloat32(b []byte) []float32 {
	count := len(b) / 4
	out := make([]float32, count)
	for i := 0; i < count; i++ {
		bits := uint32(b[i*4]) |
			uint32(b[i*4+1])<<8 |
			uint32(b[i*4+2])<<16 |
			uint32(b[i*4+3])<<24
		out[i] = math.Float32frombits(bits)
	}
	return out
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
