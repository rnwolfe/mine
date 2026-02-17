package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// testItem implements Item for testing.
type testItem struct {
	name string
	desc string
}

func (t testItem) FilterValue() string { return t.name }
func (t testItem) Title() string       { return t.name }
func (t testItem) Description() string { return t.desc }

func items(names ...string) []Item {
	out := make([]Item, len(names))
	for i, n := range names {
		out[i] = testItem{name: n}
	}
	return out
}

func TestNewPicker_Defaults(t *testing.T) {
	p := NewPicker(items("a", "b", "c"))
	if p.prompt != "> " {
		t.Fatalf("default prompt should be '> ', got %q", p.prompt)
	}
	if len(p.filtered) != 3 {
		t.Fatalf("all items should be visible initially, got %d", len(p.filtered))
	}
}

func TestNewPicker_Options(t *testing.T) {
	p := NewPicker(items("a"), WithTitle("Pick"), WithPrompt("? "), WithHeight(5))
	if p.title != "Pick" {
		t.Fatalf("title should be 'Pick', got %q", p.title)
	}
	if p.prompt != "? " {
		t.Fatalf("prompt should be '? ', got %q", p.prompt)
	}
	if p.height != 5 {
		t.Fatalf("height should be 5, got %d", p.height)
	}
}

func TestPicker_FilteringByQuery(t *testing.T) {
	p := NewPicker(items("alpha", "beta", "gamma", "delta"))

	// Type "a" — should match all 4 (alpha, beta, gamma, delta all contain 'a')
	p.query = "a"
	p.applyFilter()
	if len(p.filtered) != 4 {
		names := make([]string, len(p.filtered))
		for i, s := range p.filtered {
			names[i] = s.item.Title()
		}
		t.Fatalf("query 'a' should match 4 items, got %d: %v", len(p.filtered), names)
	}

	// Type "al" — should match alpha only
	p.query = "al"
	p.applyFilter()
	if len(p.filtered) != 1 || p.filtered[0].item.Title() != "alpha" {
		t.Fatalf("query 'al' should match only alpha, got %d items", len(p.filtered))
	}

	// Clear query — all items should reappear
	p.query = ""
	p.applyFilter()
	if len(p.filtered) != 4 {
		t.Fatalf("empty query should show all items, got %d", len(p.filtered))
	}
}

func TestPicker_Navigation(t *testing.T) {
	p := NewPicker(items("one", "two", "three"))

	if p.cursor != 0 {
		t.Fatalf("cursor should start at 0, got %d", p.cursor)
	}

	// Move down
	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.cursor != 1 {
		t.Fatalf("cursor should be 1 after down, got %d", p.cursor)
	}

	// Move down again
	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.cursor != 2 {
		t.Fatalf("cursor should be 2 after second down, got %d", p.cursor)
	}

	// Move down at bottom — should stay
	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	if p.cursor != 2 {
		t.Fatalf("cursor should stay at 2 at bottom, got %d", p.cursor)
	}

	// Move up
	p.Update(tea.KeyMsg{Type: tea.KeyUp})
	if p.cursor != 1 {
		t.Fatalf("cursor should be 1 after up, got %d", p.cursor)
	}
}

func TestPicker_EnterSelectsItem(t *testing.T) {
	p := NewPicker(items("one", "two", "three"))

	// Move to "two" and select
	p.Update(tea.KeyMsg{Type: tea.KeyDown})
	model, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := model.(*Picker)

	if result.chosen == nil {
		t.Fatal("chosen should not be nil after enter")
	}
	if result.chosen.Title() != "two" {
		t.Fatalf("chosen should be 'two', got %q", result.chosen.Title())
	}
	if result.canceled {
		t.Fatal("canceled should be false")
	}
	if cmd == nil {
		t.Fatal("enter should return tea.Quit cmd")
	}
}

func TestPicker_EscCancels(t *testing.T) {
	p := NewPicker(items("one", "two"))

	model, cmd := p.Update(tea.KeyMsg{Type: tea.KeyEsc})
	result := model.(*Picker)

	if !result.canceled {
		t.Fatal("esc should set canceled to true")
	}
	if result.chosen != nil {
		t.Fatal("chosen should be nil after cancel")
	}
	if cmd == nil {
		t.Fatal("esc should return tea.Quit cmd")
	}
}

func TestPicker_CtrlCCancels(t *testing.T) {
	p := NewPicker(items("one"))

	model, _ := p.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	result := model.(*Picker)

	if !result.canceled {
		t.Fatal("ctrl+c should set canceled to true")
	}
}

func TestPicker_BackspaceRemovesChar(t *testing.T) {
	p := NewPicker(items("alpha", "beta"))
	p.query = "al"
	p.applyFilter()

	if len(p.filtered) != 1 {
		t.Fatalf("expected 1 match for 'al', got %d", len(p.filtered))
	}

	p.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if p.query != "a" {
		t.Fatalf("backspace should remove last char, got %q", p.query)
	}
	// Filter should have re-applied and include more items
	if len(p.filtered) < 1 {
		t.Fatal("filter should have re-applied after backspace")
	}
}

func TestPicker_BackspaceUTF8(t *testing.T) {
	p := NewPicker(items("alpha"))
	// Set a query with multi-byte UTF-8 characters (e.g. "café")
	p.query = "caf\u00e9"
	p.applyFilter()

	p.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if p.query != "caf" {
		t.Fatalf("backspace should remove last rune 'é', got %q", p.query)
	}

	// Also test with CJK character
	p.query = "ab\u4e16"
	p.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if p.query != "ab" {
		t.Fatalf("backspace should remove last rune '世', got %q", p.query)
	}
}

func TestPicker_TypingFilters(t *testing.T) {
	p := NewPicker(items("alpha", "beta", "gamma"))

	// Type 'b'
	p.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	if p.query != "b" {
		t.Fatalf("typing 'b' should set query to 'b', got %q", p.query)
	}
	if len(p.filtered) != 1 {
		t.Fatalf("only 'beta' should match 'b', got %d", len(p.filtered))
	}
}

func TestPicker_EmptyListEnter(t *testing.T) {
	p := NewPicker(items("alpha"))
	p.query = "zzz"
	p.applyFilter()

	model, _ := p.Update(tea.KeyMsg{Type: tea.KeyEnter})
	result := model.(*Picker)

	if result.chosen != nil {
		t.Fatal("enter on empty list should not select anything")
	}
}

func TestPicker_ViewContainsElements(t *testing.T) {
	p := NewPicker(items("alpha", "beta"), WithTitle("Pick one"))
	view := p.View()

	if !strings.Contains(view, "Pick one") {
		t.Fatal("view should contain title")
	}
	if !strings.Contains(view, "alpha") {
		t.Fatal("view should contain item 'alpha'")
	}
	if !strings.Contains(view, "beta") {
		t.Fatal("view should contain item 'beta'")
	}
	if !strings.Contains(view, "2/2") {
		t.Fatal("view should contain count '2/2'")
	}
}

func TestPicker_ViewNoMatches(t *testing.T) {
	p := NewPicker(items("alpha"))
	p.query = "zzz"
	p.applyFilter()
	view := p.View()

	if !strings.Contains(view, "No matches") {
		t.Fatal("view should show 'No matches' when nothing matches")
	}
}

func TestPicker_WindowSizeMsg(t *testing.T) {
	p := NewPicker(items("a"))
	p.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	if p.termWidth != 120 {
		t.Fatalf("termWidth should be 120, got %d", p.termWidth)
	}
	if p.termHeight != 40 {
		t.Fatalf("termHeight should be 40, got %d", p.termHeight)
	}
}

func TestPicker_Description(t *testing.T) {
	itm := []Item{testItem{name: "session", desc: "3 windows"}}
	p := NewPicker(itm)
	view := p.View()

	if !strings.Contains(view, "3 windows") {
		t.Fatal("view should contain item description")
	}
}

func TestSortScored(t *testing.T) {
	items := []scored{
		{item: testItem{name: "low"}, score: 1},
		{item: testItem{name: "high"}, score: 10},
		{item: testItem{name: "mid"}, score: 5},
	}
	sortScored(items)

	if items[0].item.Title() != "high" {
		t.Fatalf("first item should be 'high', got %q", items[0].item.Title())
	}
	if items[1].item.Title() != "mid" {
		t.Fatalf("second item should be 'mid', got %q", items[1].item.Title())
	}
	if items[2].item.Title() != "low" {
		t.Fatalf("third item should be 'low', got %q", items[2].item.Title())
	}
}

func TestPicker_ScrollViewport(t *testing.T) {
	// Create more items than visible height.
	p := NewPicker(items("a", "b", "c", "d", "e", "f", "g", "h"), WithHeight(3))

	// Navigate down past the visible area
	for i := 0; i < 4; i++ {
		p.Update(tea.KeyMsg{Type: tea.KeyDown})
	}

	if p.cursor != 4 {
		t.Fatalf("cursor should be 4, got %d", p.cursor)
	}
	// Offset should have scrolled
	if p.offset < 2 {
		t.Fatalf("offset should have scrolled, got %d", p.offset)
	}

	// Navigate back up
	for i := 0; i < 4; i++ {
		p.Update(tea.KeyMsg{Type: tea.KeyUp})
	}
	if p.cursor != 0 {
		t.Fatalf("cursor should be 0 after navigating up, got %d", p.cursor)
	}
	if p.offset != 0 {
		t.Fatalf("offset should be 0 after navigating up, got %d", p.offset)
	}
}
