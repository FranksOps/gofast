package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// UIState represents the aggregated state for the TUI
type UIState struct {
	TotalFiles     int64
	TotalBytes     int64
	CompletedFiles int64
	CompletedBytes int64
	ActiveStreams  []*ActiveStream
	ActiveWorkers  int
	MaxWorkers     int
	ThroughputBPms float64 // bytes per millisecond
	IsRunning      bool
	Done           bool
}

// ActiveStream represents a current running transfer
type ActiveStream struct {
	JobID    string
	FilePath string
	Progress float64 // 0.0 to 1.0
	BytesSec float64 // bytes per second for this stream
}

// TUIModel implements the tea.Model interface
type TUIModel struct {
	engineState *UIState
	spinner     spinner.Model
	progress    progress.Model
	viewport    viewport.Model

	width  int
	height int

	// Styles
	titleStyle   lipgloss.Style
	infoStyle    lipgloss.Style
	streamStyle  lipgloss.Style
	helpStyle    lipgloss.Style
	errorStyle   lipgloss.Style
	successStyle lipgloss.Style
}

// TUIUpdateMsg is sent periodically to update the UI state
type TUIUpdateMsg struct {
	State *UIState
}

// WorkerCountMsg is sent when modifying the worker count
type WorkerCountMsg int

func NewTUIModel(initialState *UIState) TUIModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	prog := progress.New(progress.WithDefaultGradient())

	return TUIModel{
		engineState:  initialState,
		spinner:      s,
		progress:     prog,
		titleStyle:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205")).Padding(0, 1),
		infoStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("241")),
		streamStyle:  lipgloss.NewStyle().Foreground(lipgloss.Color("78")),
		helpStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("241")).MarginTop(1),
		errorStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		successStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true),
	}
}

func (m TUIModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
	)
}

func (m TUIModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.engineState.IsRunning = false
			return m, tea.Quit
		case "+", "=":
			// Increase workers
			return m, func() tea.Msg { return WorkerCountMsg(1) }
		case "-":
			// Decrease workers
			return m, func() tea.Msg { return WorkerCountMsg(-1) }
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = msg.Width - 14

		headerHeight := 5
		footerHeight := 2
		m.viewport = viewport.New(msg.Width, msg.Height-headerHeight-footerHeight)

	case TUIUpdateMsg:
		m.engineState = msg.State
		if m.engineState.Done {
			return m, tea.Quit
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m TUIModel) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	var sb strings.Builder

	// Header
	header := fmt.Sprintf("%s Gofast %s", m.spinner.View(), m.titleStyle.Render("Concurrent Migration Engine"))
	sb.WriteString(header + "\n")

	// Global Progress
	var percent float64 = 0
	if m.engineState.TotalBytes > 0 {
		percent = float64(m.engineState.CompletedBytes) / float64(m.engineState.TotalBytes)
	}

	totalTB := float64(m.engineState.TotalBytes) / (1024 * 1024 * 1024 * 1024)
	compTB := float64(m.engineState.CompletedBytes) / (1024 * 1024 * 1024 * 1024)

	opsInfo := fmt.Sprintf("ETA: %s | Workers: %d/%d | %.2f TB / %.2f TB",
		formatETA(percent, m.engineState.ThroughputBPms, m.engineState.TotalBytes, m.engineState.CompletedBytes),
		m.engineState.ActiveWorkers, m.engineState.MaxWorkers,
		compTB, totalTB)

	sb.WriteString(m.infoStyle.Render(opsInfo) + "\n")
	sb.WriteString(m.progress.ViewAs(percent) + "\n\n")

	// Active Streams
	sb.WriteString("Active Streams:\n")
	var streamContent strings.Builder

	if len(m.engineState.ActiveStreams) == 0 {
		streamContent.WriteString(m.infoStyle.Render("No active streams..."))
	} else {
		for _, s := range m.engineState.ActiveStreams {
			speedStr := formatSpeed(s.BytesSec)
			bar := m.progress.ViewAs(s.Progress)
			truncatePath := s.FilePath
			if len(truncatePath) > 40 {
				truncatePath = "..." + truncatePath[len(truncatePath)-37:]
			}

			// Format: [===       ] 30% | 45 MB/s | /path/to/file
			streamContent.WriteString(fmt.Sprintf("%s | %-10s | %s\n",
				bar, m.streamStyle.Render(speedStr), truncatePath))
		}
	}

	m.viewport.SetContent(streamContent.String())
	sb.WriteString(m.viewport.View())

	// Footer
	help := m.helpStyle.Render("q/ctrl+c: quit • +/-: adjust workers")
	if m.engineState.Done {
		help = m.successStyle.Render("Migration Complete!") + " Press 'q' to exit."
	}
	sb.WriteString("\n" + help)

	return sb.String()
}

func formatSpeed(bytesPerSec float64) string {
	if bytesPerSec >= 1024*1024*1024 {
		return fmt.Sprintf("%.2f GB/s", bytesPerSec/(1024*1024*1024))
	} else if bytesPerSec >= 1024*1024 {
		return fmt.Sprintf("%.2f MB/s", bytesPerSec/(1024*1024))
	} else if bytesPerSec >= 1024 {
		return fmt.Sprintf("%.2f KB/s", bytesPerSec/1024)
	}
	return fmt.Sprintf("%.0f B/s", bytesPerSec)
}

func formatETA(progress float64, bytesPerMs float64, totalBytes, completedBytes int64) string {
	if progress == 0 || bytesPerMs <= 0 || totalBytes == 0 {
		return "Calculating..."
	}

	remainingBytes := totalBytes - completedBytes
	if remainingBytes <= 0 {
		return "0s"
	}

	remainingMs := float64(remainingBytes) / bytesPerMs
	d := time.Duration(remainingMs) * time.Millisecond

	if d.Hours() > 24 {
		return fmt.Sprintf("> 1d")
	}

	return d.Round(time.Second).String()
}
