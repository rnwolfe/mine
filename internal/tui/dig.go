package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rnwolfe/mine/internal/ui"
)

// DigResult is returned when the dig TUI session ends.
type DigResult struct {
	Elapsed   time.Duration
	Completed bool // true if the full duration was reached
	Canceled  bool // true if user quit early
}

// DigModel is a full-screen Bubbletea timer model for focus sessions.
type DigModel struct {
	duration  time.Duration
	label     string
	start     time.Time
	elapsed   time.Duration
	width     int
	height    int
	quitting  bool
	completed bool
	canceled  bool
}

type digTickMsg time.Time

// NewDigModel creates a new DigModel.
func NewDigModel(duration time.Duration, label string) *DigModel {
	return &DigModel{
		duration: duration,
		label:    label,
		start:    time.Now(),
		width:    80,
		height:   24,
	}
}

// RunDig launches the full-screen dig timer TUI.
func RunDig(duration time.Duration, label string) (DigResult, error) {
	m := NewDigModel(duration, label)
	prog := tea.NewProgram(m, tea.WithAltScreen())
	result, err := prog.Run()
	if err != nil {
		return DigResult{}, fmt.Errorf("dig tui: %w", err)
	}
	final := result.(*DigModel)
	return DigResult{
		Elapsed:   final.elapsed,
		Completed: final.completed,
		Canceled:  final.canceled,
	}, nil
}

func digTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return digTickMsg(t)
	})
}

func (m *DigModel) Init() tea.Cmd {
	return digTick()
}

func (m *DigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case digTickMsg:
		m.elapsed = time.Since(m.start).Round(time.Second)
		if m.elapsed >= m.duration {
			m.elapsed = m.duration
			m.completed = true
			m.quitting = true
			return m, tea.Quit
		}
		return m, digTick()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.elapsed = time.Since(m.start).Round(time.Second)
			m.canceled = true
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *DigModel) View() string {
	var b strings.Builder

	remaining := m.duration - m.elapsed
	if remaining < 0 {
		remaining = 0
	}

	mins := int(remaining.Minutes())
	secs := int(remaining.Seconds()) % 60

	// Center content vertically
	contentLines := 10
	topPad := (m.height - contentLines) / 2
	if topPad < 0 {
		topPad = 0
	}

	// Top padding
	for i := 0; i < topPad; i++ {
		b.WriteString("\n")
	}

	// Title
	titleText := fmt.Sprintf("%s  Deep Focus", ui.IconDig)
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(ui.Gold).
		Width(m.width).
		Align(lipgloss.Center).
		Render(titleText)
	b.WriteString(title + "\n\n")

	// Session label
	labelText := fmt.Sprintf("Session: %s", m.label)
	labelLine := ui.Muted.Copy().
		Width(m.width).
		Align(lipgloss.Center).
		Render(labelText)
	b.WriteString(labelLine + "\n\n")

	// Big timer
	timerText := fmt.Sprintf("%02d:%02d", mins, secs)
	timerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ui.Gold).
		Width(m.width).
		Align(lipgloss.Center)

	if remaining <= 5*time.Minute && remaining > 0 {
		timerStyle = timerStyle.Foreground(ui.Amber)
	}
	if remaining <= time.Minute && remaining > 0 {
		timerStyle = timerStyle.Foreground(ui.Ruby)
	}

	b.WriteString(timerStyle.Render(timerText) + "\n\n")

	// Progress bar (fill terminal width minus margins)
	barWidth := m.width - 8
	if barWidth < 10 {
		barWidth = 10
	}
	if barWidth > 60 {
		barWidth = 60
	}

	pct := float64(m.elapsed) / float64(m.duration)
	filled := int(pct * float64(barWidth))
	if filled > barWidth {
		filled = barWidth
	}

	bar := ui.Success.Render(strings.Repeat("█", filled)) +
		ui.Muted.Render(strings.Repeat("░", barWidth-filled))

	barLine := lipgloss.NewStyle().
		Width(m.width).
		Align(lipgloss.Center).
		Render(bar)
	b.WriteString(barLine + "\n\n")

	// Elapsed / remaining info
	elapsedText := fmt.Sprintf("%s elapsed · %s remaining",
		m.elapsed.Round(time.Second),
		remaining.Round(time.Second),
	)
	infoLine := ui.Muted.Copy().
		Width(m.width).
		Align(lipgloss.Center).
		Render(elapsedText)
	b.WriteString(infoLine + "\n\n")

	// Help
	var helpText string
	if m.completed {
		helpText = ui.Success.Copy().
			Width(m.width).
			Align(lipgloss.Center).
			Render("Session complete!")
	} else {
		helpText = ui.Muted.Copy().
			Width(m.width).
			Align(lipgloss.Center).
			Render("q / Ctrl+C to end early")
	}
	b.WriteString(helpText + "\n")

	return b.String()
}
