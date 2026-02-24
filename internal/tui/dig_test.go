package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewDigModel_Defaults(t *testing.T) {
	m := NewDigModel(25*time.Minute, "25m", "")

	if m.duration != 25*time.Minute {
		t.Fatalf("duration should be 25m, got %v", m.duration)
	}
	if m.label != "25m" {
		t.Fatalf("label should be '25m', got %q", m.label)
	}
	if m.completed {
		t.Fatal("should not be completed initially")
	}
	if m.canceled {
		t.Fatal("should not be canceled initially")
	}
}

func TestDigModel_QuitSetsCancel(t *testing.T) {
	m := NewDigModel(25*time.Minute, "25m", "")

	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	result := model.(*DigModel)

	if !result.canceled {
		t.Fatal("q should set canceled")
	}
	if !result.quitting {
		t.Fatal("q should set quitting")
	}
	if cmd == nil {
		t.Fatal("q should return tea.Quit cmd")
	}
}

func TestDigModel_CtrlCCancels(t *testing.T) {
	m := NewDigModel(25*time.Minute, "25m", "")

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	result := model.(*DigModel)

	if !result.canceled {
		t.Fatal("ctrl+c should set canceled")
	}
}

func TestDigModel_EscNoOp(t *testing.T) {
	m := NewDigModel(25*time.Minute, "25m", "")

	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	result := model.(*DigModel)

	// Esc is a no-op in the dig TUI; only q and Ctrl+C cancel.
	if result.canceled {
		t.Fatal("esc should not cancel the dig session")
	}
	if result.quitting {
		t.Fatal("esc should not quit the dig session")
	}
	if cmd != nil {
		t.Fatal("esc should return nil cmd")
	}
}

func TestDigModel_TickUpdatesElapsed(t *testing.T) {
	m := NewDigModel(25*time.Minute, "25m", "")
	// Start slightly in the past to simulate elapsed time
	m.start = time.Now().Add(-5 * time.Second)

	m.Update(digTickMsg(time.Now()))

	if m.elapsed < 4*time.Second {
		t.Fatalf("elapsed should be ~5s, got %v", m.elapsed)
	}
}

func TestDigModel_TickCompletesAtDuration(t *testing.T) {
	m := NewDigModel(5*time.Second, "5s", "")
	// Simulate session already past its duration
	m.start = time.Now().Add(-10 * time.Second)

	model, cmd := m.Update(digTickMsg(time.Now()))
	result := model.(*DigModel)

	if !result.completed {
		t.Fatal("should be completed when elapsed >= duration")
	}
	if result.canceled {
		t.Fatal("completed session should not be canceled")
	}
	if cmd == nil {
		t.Fatal("completion should return tea.Quit cmd")
	}
}

func TestDigModel_WindowSizeMsg(t *testing.T) {
	m := NewDigModel(25*time.Minute, "25m", "")
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	if m.width != 120 {
		t.Fatalf("width should be 120, got %d", m.width)
	}
	if m.height != 40 {
		t.Fatalf("height should be 40, got %d", m.height)
	}
}

func TestDigModel_ViewContainsTimer(t *testing.T) {
	m := NewDigModel(25*time.Minute, "25m", "")
	m.elapsed = 5 * time.Minute
	view := m.View()

	// Should show remaining time: 20:00
	if !strings.Contains(view, "20:00") {
		t.Fatalf("view should show remaining time 20:00, got:\n%s", view)
	}
}

func TestDigModel_ViewContainsLabel(t *testing.T) {
	m := NewDigModel(45*time.Minute, "45m", "")
	view := m.View()

	if !strings.Contains(view, "45m") {
		t.Fatal("view should contain session label")
	}
}

func TestDigModel_ViewContainsHelp(t *testing.T) {
	m := NewDigModel(25*time.Minute, "25m", "")
	view := m.View()

	if !strings.Contains(view, "Ctrl+C") {
		t.Fatal("view should contain help text")
	}
}

func TestDigModel_ViewShowsCompletedMsg(t *testing.T) {
	m := NewDigModel(5*time.Second, "5s", "")
	m.completed = true
	view := m.View()

	if !strings.Contains(view, "Session complete") {
		t.Fatalf("completed view should show completion message, got:\n%s", view)
	}
}

func TestDigModel_ViewProgressBar(t *testing.T) {
	m := NewDigModel(25*time.Minute, "25m", "")
	m.elapsed = 12*time.Minute + 30*time.Second // ~50% done
	view := m.View()

	// Progress bar should contain both filled and empty chars
	if !strings.Contains(view, "█") {
		t.Fatal("view should contain filled bar characters")
	}
	if !strings.Contains(view, "░") {
		t.Fatal("view should contain empty bar characters")
	}
}

func TestDigModel_InitReturnsCmd(t *testing.T) {
	m := NewDigModel(25*time.Minute, "25m", "")
	cmd := m.Init()

	if cmd == nil {
		t.Fatal("Init should return a tick command")
	}
}

func TestDigModel_TaskLabel_ShownInView(t *testing.T) {
	m := NewDigModel(25*time.Minute, "25m", "Refactor auth module")
	view := m.View()

	if !strings.Contains(view, "Refactor auth module") {
		t.Fatal("view should show task label when set")
	}
	if !strings.Contains(view, "Focusing on:") {
		t.Fatal("view should show 'Focusing on:' prefix for task label")
	}
}

func TestDigModel_NoTaskLabel_NoFocusingOn(t *testing.T) {
	m := NewDigModel(25*time.Minute, "25m", "")
	view := m.View()

	if strings.Contains(view, "Focusing on:") {
		t.Fatal("view should not show 'Focusing on:' when no task label set")
	}
}

func TestDigResult_Fields(t *testing.T) {
	r := DigResult{
		Elapsed:   10 * time.Minute,
		Completed: true,
		Canceled:  false,
	}

	if r.Elapsed != 10*time.Minute {
		t.Fatalf("elapsed should be 10m, got %v", r.Elapsed)
	}
	if !r.Completed {
		t.Fatal("should be completed")
	}
	if r.Canceled {
		t.Fatal("should not be canceled")
	}
}
