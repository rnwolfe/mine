package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rnwolfe/mine/internal/proj"
	"github.com/rnwolfe/mine/internal/todo"
)

// makeDashData creates a populated DashData for testing.
func makeDashData() DashData {
	now := time.Now()
	todos := []todo.Todo{
		{ID: 1, Title: "Fix critical bug", Priority: todo.PrioCrit, Schedule: todo.ScheduleToday, CreatedAt: now, UpdatedAt: now},
		{ID: 2, Title: "Write tests", Priority: todo.PrioHigh, Schedule: todo.ScheduleSoon, CreatedAt: now, UpdatedAt: now},
		{ID: 3, Title: "Update docs", Priority: todo.PrioMedium, Schedule: todo.ScheduleLater, CreatedAt: now, UpdatedAt: now},
	}
	return DashData{
		Todos:        todos,
		TodoOpen:     3,
		TodoOverdue:  0,
		Streak:       5,
		WeekDone:     7,
		TotalFocus:   3 * time.Hour,
		HasFocusData: true,
		Project: &proj.Project{
			Name:   "myapp",
			Path:   "/home/user/myapp",
			Branch: "main",
		},
	}
}

// newLoadedModel creates a DashModel with pre-loaded data (no DB needed).
func newLoadedModel(data DashData, width, height int) *DashModel {
	m := &DashModel{
		data:    data,
		width:   width,
		height:  height,
		loading: false,
	}
	return m
}

// --- Panel render function tests ---

func TestRenderTodosPanel_NonEmpty(t *testing.T) {
	for _, width := range []int{80, 120, 200} {
		todos := makeDashData().Todos
		out := renderTodosPanel(todos, 3, 0, width)
		if strings.TrimSpace(out) == "" {
			t.Errorf("renderTodosPanel returned empty string at width %d", width)
		}
		if !strings.Contains(out, "Todos") {
			t.Errorf("renderTodosPanel should contain 'Todos' at width %d", width)
		}
	}
}

func TestRenderTodosPanel_ShowsTop5(t *testing.T) {
	now := time.Now()
	todos := make([]todo.Todo, 8)
	for i := range todos {
		todos[i] = todo.Todo{ID: i + 1, Title: "todo", Priority: todo.PrioMedium, CreatedAt: now, UpdatedAt: now}
	}
	out := renderTodosPanel(todos, 8, 0, 80)
	if !strings.Contains(out, "and 3 more") {
		t.Errorf("expected '...and 3 more' for 8 todos, got: %s", out)
	}
}

func TestRenderTodosPanel_EmptyList(t *testing.T) {
	out := renderTodosPanel(nil, 0, 0, 80)
	if !strings.Contains(out, "All clear") {
		t.Errorf("empty todos panel should say 'All clear', got: %s", out)
	}
}

func TestRenderTodosPanel_OverdueCount(t *testing.T) {
	todos := makeDashData().Todos
	out := renderTodosPanel(todos, 3, 2, 80)
	if !strings.Contains(out, "overdue") {
		t.Errorf("panel should show overdue count, got: %s", out)
	}
}

func TestRenderFocusPanel_NonEmpty(t *testing.T) {
	data := makeDashData()
	for _, width := range []int{80, 120, 200} {
		out := renderFocusPanel(data, width)
		if strings.TrimSpace(out) == "" {
			t.Errorf("renderFocusPanel returned empty string at width %d", width)
		}
		if !strings.Contains(out, "Focus") {
			t.Errorf("renderFocusPanel should contain 'Focus' at width %d", width)
		}
	}
}

func TestRenderFocusPanel_ShowsStreak(t *testing.T) {
	data := makeDashData()
	out := renderFocusPanel(data, 80)
	if !strings.Contains(out, "5-day streak") {
		t.Errorf("focus panel should show streak, got: %s", out)
	}
}

func TestRenderFocusPanel_NoStreak(t *testing.T) {
	data := makeDashData()
	data.Streak = 0
	out := renderFocusPanel(data, 80)
	if !strings.Contains(out, "No streak") {
		t.Errorf("focus panel should show no-streak message, got: %s", out)
	}
}

func TestRenderFocusPanel_ShowsFocusTime(t *testing.T) {
	data := makeDashData()
	out := renderFocusPanel(data, 80)
	if !strings.Contains(out, "Total focus") {
		t.Errorf("focus panel should show total focus time, got: %s", out)
	}
}

func TestRenderProjectPanel_NonEmpty(t *testing.T) {
	p := &proj.Project{Name: "myapp", Path: "/home/user/myapp", Branch: "main"}
	for _, width := range []int{80, 120, 200} {
		out := renderProjectPanel(p, 3, width)
		if strings.TrimSpace(out) == "" {
			t.Errorf("renderProjectPanel returned empty string at width %d", width)
		}
		if !strings.Contains(out, "Project") {
			t.Errorf("renderProjectPanel should contain 'Project' at width %d", width)
		}
	}
}

func TestRenderProjectPanel_ShowsName(t *testing.T) {
	p := &proj.Project{Name: "myapp", Path: "/home/user/myapp", Branch: "feature/x"}
	out := renderProjectPanel(p, 2, 80)
	if !strings.Contains(out, "myapp") {
		t.Errorf("project panel should show project name, got: %s", out)
	}
	if !strings.Contains(out, "feature/x") {
		t.Errorf("project panel should show branch, got: %s", out)
	}
}

func TestRenderProjectPanel_NilProject(t *testing.T) {
	out := renderProjectPanel(nil, 0, 80)
	if !strings.Contains(out, "Not in a registered project") {
		t.Errorf("nil project panel should show not-in-project message, got: %s", out)
	}
}

// --- Model state tests ---

func TestDashModel_Defaults(t *testing.T) {
	m := &DashModel{width: 80, height: 24, loading: true}
	if m.width != 80 {
		t.Fatalf("default width should be 80, got %d", m.width)
	}
	if !m.loading {
		t.Fatal("model should start in loading state")
	}
	if m.action != DashActionQuit {
		t.Fatalf("default action should be DashActionQuit, got %d", m.action)
	}
}

func TestDashModel_WindowSizeMsg(t *testing.T) {
	m := newLoadedModel(makeDashData(), 80, 24)
	result, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	m = result.(*DashModel)
	if m.width != 160 {
		t.Fatalf("width should be 160 after WindowSizeMsg, got %d", m.width)
	}
	if m.height != 50 {
		t.Fatalf("height should be 50 after WindowSizeMsg, got %d", m.height)
	}
}

func TestDashModel_QuitKey(t *testing.T) {
	m := newLoadedModel(makeDashData(), 80, 24)
	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	fm := result.(*DashModel)
	if fm.action != DashActionQuit {
		t.Fatalf("q should set action to DashActionQuit, got %d", fm.action)
	}
	if cmd == nil {
		t.Fatal("q should return tea.Quit cmd")
	}
}

func TestDashModel_CtrlCQuits(t *testing.T) {
	m := newLoadedModel(makeDashData(), 80, 24)
	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	fm := result.(*DashModel)
	if fm.action != DashActionQuit {
		t.Fatalf("ctrl+c should set DashActionQuit, got %d", fm.action)
	}
	if cmd == nil {
		t.Fatal("ctrl+c should return tea.Quit cmd")
	}
}

func TestDashModel_TKeyOpensTodo(t *testing.T) {
	m := newLoadedModel(makeDashData(), 80, 24)
	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	fm := result.(*DashModel)
	if fm.action != DashActionOpenTodo {
		t.Fatalf("t should set DashActionOpenTodo, got %d", fm.action)
	}
	if cmd == nil {
		t.Fatal("t should return tea.Quit cmd")
	}
}

func TestDashModel_DKeyStartsDig(t *testing.T) {
	m := newLoadedModel(makeDashData(), 80, 24)
	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	fm := result.(*DashModel)
	if fm.action != DashActionStartDig {
		t.Fatalf("d should set DashActionStartDig, got %d", fm.action)
	}
	if cmd == nil {
		t.Fatal("d should return tea.Quit cmd")
	}
}

func TestDashModel_TKeyIgnoredWhileLoading(t *testing.T) {
	m := &DashModel{width: 80, height: 24, loading: true}
	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	fm := result.(*DashModel)
	if fm.action != DashActionQuit {
		t.Fatalf("t while loading should not change action, got %d", fm.action)
	}
	if cmd != nil {
		t.Fatal("t while loading should return nil cmd")
	}
}

func TestDashModel_RKeyRefreshes(t *testing.T) {
	m := newLoadedModel(makeDashData(), 80, 24)
	result, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	fm := result.(*DashModel)
	if !fm.loading {
		t.Fatal("r should set loading = true")
	}
	if cmd == nil {
		t.Fatal("r should return a load command")
	}
}

// --- View layout tests ---

func TestDashModel_ViewTwoColumn(t *testing.T) {
	m := newLoadedModel(makeDashData(), 120, 40)
	view := m.View()
	if strings.TrimSpace(view) == "" {
		t.Fatal("two-column view should be non-empty")
	}
	if !strings.Contains(view, "Todos") {
		t.Fatal("two-column view should contain 'Todos'")
	}
	if !strings.Contains(view, "Focus") {
		t.Fatal("two-column view should contain 'Focus'")
	}
	if !strings.Contains(view, "q quit") {
		t.Fatal("two-column view should contain help bar")
	}
}

func TestDashModel_ViewStacked(t *testing.T) {
	m := newLoadedModel(makeDashData(), 80, 40)
	view := m.View()
	if strings.TrimSpace(view) == "" {
		t.Fatal("stacked view should be non-empty")
	}
	if !strings.Contains(view, "Todos") {
		t.Fatal("stacked view should contain 'Todos'")
	}
}

func TestDashModel_ViewWideWidth(t *testing.T) {
	m := newLoadedModel(makeDashData(), 200, 50)
	view := m.View()
	if strings.TrimSpace(view) == "" {
		t.Fatal("wide view (200 cols) should be non-empty")
	}
}

func TestDashModel_ViewMinimal(t *testing.T) {
	m := newLoadedModel(makeDashData(), 40, 24)
	view := m.View()
	if strings.TrimSpace(view) == "" {
		t.Fatal("minimal view should be non-empty")
	}
	if !strings.Contains(view, "q quit") {
		t.Fatal("minimal view should contain quit hint")
	}
	if !strings.Contains(view, "d dig") {
		t.Fatal("minimal view should contain dig hint")
	}
}

func TestDashModel_ViewLoading(t *testing.T) {
	m := &DashModel{width: 80, height: 24, loading: true}
	view := m.View()
	if !strings.Contains(view, "Loading") {
		t.Fatal("loading view should say 'Loading'")
	}
}
