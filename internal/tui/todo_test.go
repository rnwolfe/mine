package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rnwolfe/mine/internal/todo"
)

func makeTodos(titles ...string) []todo.Todo {
	out := make([]todo.Todo, len(titles))
	now := time.Now()
	for i, t := range titles {
		out[i] = todo.Todo{
			ID:        i + 1,
			Title:     t,
			Priority:  todo.PrioMedium,
			CreatedAt: now,
			UpdatedAt: now,
		}
	}
	return out
}

func TestNewTodoModel_Defaults(t *testing.T) {
	todos := makeTodos("buy milk", "write tests", "ship it")
	m := NewTodoModel(todos)

	if m.cursor != 0 {
		t.Fatalf("cursor should start at 0, got %d", m.cursor)
	}
	if len(m.filtered) != 3 {
		t.Fatalf("all todos should be visible initially, got %d", len(m.filtered))
	}
	if m.mode != todoModeNormal {
		t.Fatalf("initial mode should be normal, got %d", m.mode)
	}
}

func TestTodoModel_NavigateDownUp(t *testing.T) {
	todos := makeTodos("one", "two", "three")
	m := NewTodoModel(todos)

	// Move down
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.cursor != 1 {
		t.Fatalf("cursor should be 1 after j, got %d", m.cursor)
	}

	// Move down again
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.cursor != 2 {
		t.Fatalf("cursor should be 2, got %d", m.cursor)
	}

	// At bottom, j should clamp
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if m.cursor != 2 {
		t.Fatalf("cursor should stay at 2, got %d", m.cursor)
	}

	// Move up with k
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if m.cursor != 1 {
		t.Fatalf("cursor should be 1 after k, got %d", m.cursor)
	}
}

func TestTodoModel_ArrowKeysNavigate(t *testing.T) {
	todos := makeTodos("one", "two")
	m := NewTodoModel(todos)

	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursor != 1 {
		t.Fatalf("cursor should be 1 after down arrow, got %d", m.cursor)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.cursor != 0 {
		t.Fatalf("cursor should be 0 after up arrow, got %d", m.cursor)
	}
}

func TestTodoModel_GotoTopBottom(t *testing.T) {
	todos := makeTodos("a", "b", "c", "d")
	m := NewTodoModel(todos)

	// Move to bottom with G
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if m.cursor != 3 {
		t.Fatalf("G should move to last item, got %d", m.cursor)
	}

	// Move to top with g
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if m.cursor != 0 {
		t.Fatalf("g should move to first item, got %d", m.cursor)
	}
}

func TestTodoModel_ToggleAction(t *testing.T) {
	todos := makeTodos("buy milk")
	m := NewTodoModel(todos)

	// Press x to toggle
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	if len(m.Actions) != 1 {
		t.Fatalf("expected 1 action after toggle, got %d", len(m.Actions))
	}
	if m.Actions[0].Type != "toggle" {
		t.Fatalf("expected toggle action, got %q", m.Actions[0].Type)
	}
	if m.Actions[0].ID != 1 {
		t.Fatalf("expected ID 1, got %d", m.Actions[0].ID)
	}

	// Local state should be toggled
	if !m.todos[0].Done {
		t.Fatal("todo should be marked done locally after toggle")
	}
}

func TestTodoModel_SpaceToggles(t *testing.T) {
	todos := makeTodos("test item")
	m := NewTodoModel(todos)

	m.Update(tea.KeyMsg{Type: tea.KeySpace})

	if len(m.Actions) != 1 || m.Actions[0].Type != "toggle" {
		t.Fatalf("space should produce toggle action, got %+v", m.Actions)
	}
}

func TestTodoModel_DeleteAction(t *testing.T) {
	todos := makeTodos("item one", "item two")
	m := NewTodoModel(todos)

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

	if len(m.Actions) != 1 || m.Actions[0].Type != "delete" {
		t.Fatalf("expected delete action, got %+v", m.Actions)
	}
	if len(m.todos) != 1 {
		t.Fatalf("todos should have 1 item after delete, got %d", len(m.todos))
	}
}

func TestTodoModel_AddMode(t *testing.T) {
	todos := makeTodos("existing")
	m := NewTodoModel(todos)

	// Enter add mode
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if m.mode != todoModeAdd {
		t.Fatalf("mode should be add, got %d", m.mode)
	}

	// Type a title
	for _, r := range "new task" {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if m.addInput != "new task" {
		t.Fatalf("addInput should be 'new task', got %q", m.addInput)
	}

	// Confirm with enter
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.mode != todoModeNormal {
		t.Fatalf("mode should return to normal after enter, got %d", m.mode)
	}
	if len(m.Actions) != 1 || m.Actions[0].Type != "add" {
		t.Fatalf("expected add action, got %+v", m.Actions)
	}
	if m.Actions[0].Text != "new task" {
		t.Fatalf("expected text 'new task', got %q", m.Actions[0].Text)
	}
}

func TestTodoModel_AddModeEscapesCancels(t *testing.T) {
	todos := makeTodos("existing")
	m := NewTodoModel(todos)

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	for _, r := range "partial" {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if m.mode != todoModeNormal {
		t.Fatalf("esc should exit add mode, got %d", m.mode)
	}
	if len(m.Actions) != 0 {
		t.Fatalf("canceled add should not produce actions, got %+v", m.Actions)
	}
}

func TestTodoModel_AddModeBackspace(t *testing.T) {
	todos := makeTodos("x")
	m := NewTodoModel(todos)

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	for _, r := range "hello" {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.addInput != "hell" {
		t.Fatalf("backspace should remove last char, got %q", m.addInput)
	}
}

func TestTodoModel_FilterMode(t *testing.T) {
	todos := makeTodos("buy milk", "write tests", "book flight")
	m := NewTodoModel(todos)

	// Enter filter mode
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if m.mode != todoModeFilter {
		t.Fatalf("/ should enter filter mode, got %d", m.mode)
	}

	// Type filter
	for _, r := range "milk" {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if m.filter != "milk" {
		t.Fatalf("filter should be 'milk', got %q", m.filter)
	}
	if len(m.filtered) != 1 {
		t.Fatalf("filter 'milk' should match 1 item, got %d", len(m.filtered))
	}

	// Confirm
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.mode != todoModeNormal {
		t.Fatalf("enter should confirm filter, got %d", m.mode)
	}

	// Filter remains active
	if len(m.filtered) != 1 {
		t.Fatalf("filter should still be active, got %d items", len(m.filtered))
	}
}

func TestTodoModel_FilterModeClear(t *testing.T) {
	todos := makeTodos("a", "b", "c")
	m := NewTodoModel(todos)

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	for _, r := range "zzz" {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	if len(m.filtered) != 0 {
		t.Fatalf("'zzz' should match nothing, got %d", len(m.filtered))
	}

	// Esc clears filter
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.mode != todoModeNormal {
		t.Fatalf("esc should return to normal, got %d", m.mode)
	}
	if len(m.filtered) != 3 {
		t.Fatalf("cleared filter should show all items, got %d", len(m.filtered))
	}
}

func TestTodoModel_FilterBackspace(t *testing.T) {
	todos := makeTodos("buy milk")
	m := NewTodoModel(todos)

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	for _, r := range "buy" {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.filter != "bu" {
		t.Fatalf("backspace should remove last filter char, got %q", m.filter)
	}
}

func TestTodoModel_QuitAction(t *testing.T) {
	todos := makeTodos("item")
	m := NewTodoModel(todos)

	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	result := model.(*TodoModel)

	if !result.quitting {
		t.Fatal("q should set quitting")
	}
	if cmd == nil {
		t.Fatal("q should return tea.Quit cmd")
	}
}

func TestTodoModel_EscQuits(t *testing.T) {
	todos := makeTodos("item")
	m := NewTodoModel(todos)

	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	result := model.(*TodoModel)

	if !result.quitting {
		t.Fatal("esc should set quitting in normal mode")
	}
	if cmd == nil {
		t.Fatal("esc should return tea.Quit cmd")
	}
}

func TestTodoModel_CtrlCQuits(t *testing.T) {
	todos := makeTodos("item")
	m := NewTodoModel(todos)

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	result := model.(*TodoModel)

	if !result.quitting {
		t.Fatal("ctrl+c should set quitting")
	}
}

func TestTodoModel_WindowSizeMsg(t *testing.T) {
	m := NewTodoModel(makeTodos("x"))
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	if m.width != 120 {
		t.Fatalf("width should be 120, got %d", m.width)
	}
	if m.height != 40 {
		t.Fatalf("height should be 40, got %d", m.height)
	}
}

func TestTodoModel_ViewContainsTodos(t *testing.T) {
	todos := makeTodos("buy milk", "write tests")
	m := NewTodoModel(todos)
	view := m.View()

	if !strings.Contains(view, "buy milk") {
		t.Fatal("view should contain 'buy milk'")
	}
	if !strings.Contains(view, "write tests") {
		t.Fatal("view should contain 'write tests'")
	}
}

func TestTodoModel_ViewShowsHelp(t *testing.T) {
	m := NewTodoModel(makeTodos("x"))
	view := m.View()

	if !strings.Contains(view, "j/k") {
		t.Fatal("view should show navigation help")
	}
	if !strings.Contains(view, "toggle") {
		t.Fatal("view should mention toggle")
	}
}

func TestTodoModel_ViewFilterMode(t *testing.T) {
	m := NewTodoModel(makeTodos("x"))
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	view := m.View()

	if !strings.Contains(view, "esc clear") {
		t.Fatal("filter mode view should show filter help")
	}
}

func TestTodoModel_ViewAddMode(t *testing.T) {
	m := NewTodoModel(makeTodos("x"))
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	view := m.View()

	if !strings.Contains(view, "add:") {
		t.Fatal("add mode view should show 'add:' prompt")
	}
}

func TestTodoModel_ViewEmptyList(t *testing.T) {
	m := NewTodoModel([]todo.Todo{})
	view := m.View()

	if !strings.Contains(view, "No todos") {
		t.Fatal("empty list view should say 'No todos'")
	}
}

func TestTodoModel_DeleteClampscursor(t *testing.T) {
	todos := makeTodos("a", "b")
	m := NewTodoModel(todos)

	// Move to last item
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	// Delete it
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

	if m.cursor != 0 {
		t.Fatalf("cursor should clamp to 0 after last item deleted, got %d", m.cursor)
	}
}

func TestTodoModel_AddEmptyTextSkipped(t *testing.T) {
	todos := makeTodos("existing")
	m := NewTodoModel(todos)

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	// Enter with empty input
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if len(m.Actions) != 0 {
		t.Fatalf("empty add should not produce action, got %+v", m.Actions)
	}
}

func TestTodoModel_DoneItemsMuted(t *testing.T) {
	now := time.Now()
	todos := []todo.Todo{
		{ID: 1, Title: "done item", Done: true, Priority: todo.PrioMedium, CreatedAt: now, UpdatedAt: now},
	}
	m := NewTodoModel(todos)
	view := m.View()

	if !strings.Contains(view, "done item") {
		t.Fatal("view should contain done item title")
	}
	if !strings.Contains(view, "✓") {
		t.Fatal("view should show check mark for done item")
	}
}
