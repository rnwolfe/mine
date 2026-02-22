package tmux

import (
	"fmt"
	"testing"
)

func TestParseWindows(t *testing.T) {
	raw := "0\teditor\t1\n1\tserver\t0\n2\ttests\t0\n"
	windows := parseWindows(raw)

	if len(windows) != 3 {
		t.Fatalf("expected 3 windows, got %d", len(windows))
	}

	w := windows[0]
	if w.Index != 0 {
		t.Errorf("expected index 0, got %d", w.Index)
	}
	if w.Name != "editor" {
		t.Errorf("expected name 'editor', got %q", w.Name)
	}
	if !w.Active {
		t.Error("expected active=true for first window")
	}

	w2 := windows[1]
	if w2.Index != 1 {
		t.Errorf("expected index 1, got %d", w2.Index)
	}
	if w2.Name != "server" {
		t.Errorf("expected name 'server', got %q", w2.Name)
	}
	if w2.Active {
		t.Error("expected active=false for second window")
	}
}

func TestParseWindows_Empty(t *testing.T) {
	windows := parseWindows("")
	if windows != nil {
		t.Fatalf("expected nil for empty input, got %v", windows)
	}
}

func TestParseWindows_Whitespace(t *testing.T) {
	windows := parseWindows("  \n  \n")
	if windows != nil {
		t.Fatalf("expected nil for whitespace-only input, got %v", windows)
	}
}

func TestParseWindows_MalformedLine(t *testing.T) {
	// Lines with fewer than 3 tab-separated fields should be skipped.
	raw := "bad-line\n0\tgood\t0\n"
	windows := parseWindows(raw)

	if len(windows) != 1 {
		t.Fatalf("expected 1 window, got %d", len(windows))
	}
	if windows[0].Name != "good" {
		t.Fatalf("expected name 'good', got %q", windows[0].Name)
	}
}

func TestWindowItem(t *testing.T) {
	w := Window{
		Index:  2,
		Name:   "editor",
		Active: true,
	}

	if w.FilterValue() != "editor" {
		t.Fatalf("FilterValue should return name, got %q", w.FilterValue())
	}
	if w.Title() != "editor" {
		t.Fatalf("Title should return name, got %q", w.Title())
	}

	desc := w.Description()
	if desc != "index 2  (active)" {
		t.Fatalf("unexpected description: %q", desc)
	}

	// Test without active flag.
	w.Active = false
	desc = w.Description()
	if desc != "index 2" {
		t.Fatalf("unexpected description for inactive window: %q", desc)
	}
}

func TestWindowItem_ZeroIndex(t *testing.T) {
	w := Window{Index: 0, Name: "main", Active: false}
	if w.Description() != "index 0" {
		t.Fatalf("unexpected description: %q", w.Description())
	}
}

func TestListWindows_Stubbed(t *testing.T) {
	original := tmuxCmd
	defer func() { tmuxCmd = original }()

	var capturedArgs []string
	tmuxCmd = func(args ...string) (string, error) {
		capturedArgs = args
		return "0\teditor\t1\n1\tserver\t0\n", nil
	}

	windows, err := ListWindows("mysession")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(windows) != 2 {
		t.Fatalf("expected 2 windows, got %d", len(windows))
	}

	// Verify tmux was called with correct args.
	if len(capturedArgs) < 3 || capturedArgs[0] != "list-windows" {
		t.Fatalf("unexpected tmux args: %v", capturedArgs)
	}
	// -t mysession should be in args.
	found := false
	for i, a := range capturedArgs {
		if a == "-t" && i+1 < len(capturedArgs) && capturedArgs[i+1] == "mysession" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected -t mysession in args, got: %v", capturedArgs)
	}
}

func TestListWindows_Error(t *testing.T) {
	original := tmuxCmd
	defer func() { tmuxCmd = original }()

	tmuxCmd = func(args ...string) (string, error) {
		return "", fmt.Errorf("session not found")
	}

	_, err := ListWindows("nosuchsession")
	if err == nil {
		t.Fatal("expected error for failed list-windows, got nil")
	}
}

func TestNewWindow_Stubbed(t *testing.T) {
	original := tmuxCmd
	defer func() { tmuxCmd = original }()

	var capturedArgs []string
	tmuxCmd = func(args ...string) (string, error) {
		capturedArgs = args
		return "", nil
	}

	if err := NewWindow("mysession", "editor"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify new-window args.
	if capturedArgs[0] != "new-window" {
		t.Fatalf("expected 'new-window', got %q", capturedArgs[0])
	}
	// Should include -t mysession and -n editor.
	hasTarget := false
	hasName := false
	for i, a := range capturedArgs {
		if a == "-t" && i+1 < len(capturedArgs) && capturedArgs[i+1] == "mysession" {
			hasTarget = true
		}
		if a == "-n" && i+1 < len(capturedArgs) && capturedArgs[i+1] == "editor" {
			hasName = true
		}
	}
	if !hasTarget {
		t.Fatalf("expected -t mysession in args, got: %v", capturedArgs)
	}
	if !hasName {
		t.Fatalf("expected -n editor in args, got: %v", capturedArgs)
	}
}

func TestNewWindow_EmptyName(t *testing.T) {
	original := tmuxCmd
	defer func() { tmuxCmd = original }()

	var capturedArgs []string
	tmuxCmd = func(args ...string) (string, error) {
		capturedArgs = args
		return "", nil
	}

	// Empty name should omit -n flag.
	if err := NewWindow("mysession", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for i, a := range capturedArgs {
		if a == "-n" {
			t.Fatalf("expected no -n flag for empty name, got args: %v (flag at index %d)", capturedArgs, i)
		}
	}
}

func TestKillWindow_Stubbed(t *testing.T) {
	original := tmuxCmd
	defer func() { tmuxCmd = original }()

	var capturedArgs []string
	tmuxCmd = func(args ...string) (string, error) {
		capturedArgs = args
		return "", nil
	}

	if err := KillWindow("mysession", "editor"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(capturedArgs) != 3 ||
		capturedArgs[0] != "kill-window" ||
		capturedArgs[1] != "-t" ||
		capturedArgs[2] != "mysession:editor" {
		t.Fatalf("unexpected tmux args: %v", capturedArgs)
	}
}

func TestKillWindow_Error(t *testing.T) {
	original := tmuxCmd
	defer func() { tmuxCmd = original }()

	tmuxCmd = func(args ...string) (string, error) {
		return "", fmt.Errorf("can't find window: editor")
	}

	err := KillWindow("mysession", "editor")
	if err == nil {
		t.Fatal("expected error for failed kill-window, got nil")
	}
}

func TestRenameWindow_Stubbed(t *testing.T) {
	original := tmuxCmd
	defer func() { tmuxCmd = original }()

	var capturedArgs []string
	tmuxCmd = func(args ...string) (string, error) {
		capturedArgs = args
		return "", nil
	}

	if err := RenameWindow("mysession", "old", "new"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(capturedArgs) != 4 ||
		capturedArgs[0] != "rename-window" ||
		capturedArgs[1] != "-t" ||
		capturedArgs[2] != "mysession:old" ||
		capturedArgs[3] != "new" {
		t.Fatalf("unexpected tmux args: %v", capturedArgs)
	}
}

func TestRenameWindow_EmptyNewName(t *testing.T) {
	err := RenameWindow("mysession", "old", "")
	if err == nil {
		t.Fatal("expected error for empty new name")
	}
	if err.Error() != "new window name cannot be empty" {
		t.Fatalf("unexpected error message: %q", err.Error())
	}
}

func TestRenameWindow_Error(t *testing.T) {
	original := tmuxCmd
	defer func() { tmuxCmd = original }()

	tmuxCmd = func(args ...string) (string, error) {
		return "", fmt.Errorf("can't find window: notexist")
	}

	err := RenameWindow("mysession", "notexist", "newname")
	if err == nil {
		t.Fatal("expected error when window not found")
	}
}

func TestFindWindowByName_Found(t *testing.T) {
	windows := []Window{
		{Index: 0, Name: "editor"},
		{Index: 1, Name: "server"},
		{Index: 2, Name: "tests"},
	}

	w := FindWindowByName("server", windows)
	if w == nil {
		t.Fatal("expected to find window 'server', got nil")
	}
	if w.Name != "server" {
		t.Fatalf("expected 'server', got %q", w.Name)
	}
}

func TestFindWindowByName_NotFound(t *testing.T) {
	windows := []Window{
		{Index: 0, Name: "editor"},
		{Index: 1, Name: "server"},
	}

	w := FindWindowByName("missing", windows)
	if w != nil {
		t.Fatalf("expected nil for missing window, got %v", w)
	}
}

func TestFindWindowByName_Empty(t *testing.T) {
	w := FindWindowByName("anything", nil)
	if w != nil {
		t.Fatalf("expected nil for empty window list, got %v", w)
	}
}
