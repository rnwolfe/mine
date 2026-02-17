package tmux

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteAndReadLayout(t *testing.T) {
	// Use a temp directory for layout storage.
	tmp := t.TempDir()
	origConfigDir := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmp)
	defer os.Setenv("XDG_CONFIG_HOME", origConfigDir)

	layout := &Layout{
		Name: "dev",
		Windows: []WindowLayout{
			{
				Name:      "editor",
				Layout:    "main-vertical",
				PaneCount: 2,
				Panes: []PaneLayout{
					{Dir: "/home/user/project", Command: "vim"},
					{Dir: "/home/user/project"},
				},
			},
			{
				Name:      "server",
				Layout:    "even-horizontal",
				PaneCount: 1,
				Panes: []PaneLayout{
					{Dir: "/home/user/project", Command: "go run ."},
				},
			},
		},
	}

	// Write
	if err := WriteLayout(layout); err != nil {
		t.Fatalf("WriteLayout failed: %v", err)
	}

	// Verify the file exists
	path := filepath.Join(tmp, "mine", "tmux", "layouts", "dev.toml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("layout file should exist at %s: %v", path, err)
	}

	// Read back
	got, err := ReadLayout("dev")
	if err != nil {
		t.Fatalf("ReadLayout failed: %v", err)
	}

	if got.Name != "dev" {
		t.Fatalf("expected name 'dev', got %q", got.Name)
	}
	if len(got.Windows) != 2 {
		t.Fatalf("expected 2 windows, got %d", len(got.Windows))
	}

	w0 := got.Windows[0]
	if w0.Name != "editor" {
		t.Fatalf("expected window name 'editor', got %q", w0.Name)
	}
	if w0.Layout != "main-vertical" {
		t.Fatalf("expected layout 'main-vertical', got %q", w0.Layout)
	}
	if w0.PaneCount != 2 {
		t.Fatalf("expected 2 panes, got %d", w0.PaneCount)
	}
	if len(w0.Panes) != 2 {
		t.Fatalf("expected 2 pane layouts, got %d", len(w0.Panes))
	}
	if w0.Panes[0].Dir != "/home/user/project" {
		t.Fatalf("expected dir '/home/user/project', got %q", w0.Panes[0].Dir)
	}
	if w0.Panes[0].Command != "vim" {
		t.Fatalf("expected command 'vim', got %q", w0.Panes[0].Command)
	}
	if w0.Panes[1].Command != "" {
		t.Fatalf("expected empty command, got %q", w0.Panes[1].Command)
	}

	w1 := got.Windows[1]
	if w1.Name != "server" {
		t.Fatalf("expected window name 'server', got %q", w1.Name)
	}
	if w1.Panes[0].Command != "go run ." {
		t.Fatalf("expected command 'go run .', got %q", w1.Panes[0].Command)
	}
}

func TestReadLayout_NotFound(t *testing.T) {
	tmp := t.TempDir()
	origConfigDir := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmp)
	defer os.Setenv("XDG_CONFIG_HOME", origConfigDir)

	_, err := ReadLayout("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent layout")
	}
}

func TestListLayouts(t *testing.T) {
	tmp := t.TempDir()
	origConfigDir := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmp)
	defer os.Setenv("XDG_CONFIG_HOME", origConfigDir)

	// No layouts directory yet
	names, err := ListLayouts()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if names != nil {
		t.Fatalf("expected nil for nonexistent dir, got %v", names)
	}

	// Write two layouts
	l1 := &Layout{Name: "alpha", Windows: []WindowLayout{{Name: "w1", PaneCount: 1}}}
	l2 := &Layout{Name: "beta", Windows: []WindowLayout{{Name: "w1", PaneCount: 1}}}
	WriteLayout(l1)
	WriteLayout(l2)

	names, err = ListLayouts()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(names) != 2 {
		t.Fatalf("expected 2 layouts, got %d: %v", len(names), names)
	}

	// Should be sorted alphabetically by ReadDir.
	if names[0] != "alpha" || names[1] != "beta" {
		t.Fatalf("expected [alpha, beta], got %v", names)
	}
}

func TestDeleteLayout(t *testing.T) {
	tmp := t.TempDir()
	origConfigDir := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmp)
	defer os.Setenv("XDG_CONFIG_HOME", origConfigDir)

	l := &Layout{Name: "todelete", Windows: []WindowLayout{{Name: "w1", PaneCount: 1}}}
	WriteLayout(l)

	if err := DeleteLayout("todelete"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be gone.
	_, err := ReadLayout("todelete")
	if err == nil {
		t.Fatal("expected error after deletion")
	}

	// Delete again should error.
	err = DeleteLayout("todelete")
	if err == nil {
		t.Fatal("expected error for double delete")
	}
}

func TestCaptureLayout_Stubbed(t *testing.T) {
	tmp := t.TempDir()
	orig := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmp)
	defer os.Setenv("XDG_CONFIG_HOME", orig)

	origCmd := tmuxCmd
	defer func() { tmuxCmd = origCmd }()

	tmuxCmd = func(args ...string) (string, error) {
		if len(args) == 0 {
			return "", fmt.Errorf("no args")
		}
		switch args[0] {
		case "list-windows":
			return "editor\tmain-vertical\t2\nserver\teven-horizontal\t1", nil
		case "list-panes":
			// Return panes for the target window.
			if len(args) >= 3 && args[2] == "editor" {
				return "/home/user/project\tvim\n/home/user/project\tbash", nil
			}
			return "/var/log\ttail -f syslog", nil
		default:
			return "", fmt.Errorf("unexpected command: %s", args[0])
		}
	}

	// SaveLayout calls captureLayout internally.
	if err := SaveLayout("test-layout"); err != nil {
		t.Fatalf("SaveLayout failed: %v", err)
	}

	// Verify persisted file.
	got, err := ReadLayout("test-layout")
	if err != nil {
		t.Fatalf("ReadLayout failed: %v", err)
	}
	if got.Name != "test-layout" {
		t.Fatalf("expected name 'test-layout', got %q", got.Name)
	}
	if len(got.Windows) != 2 {
		t.Fatalf("expected 2 windows, got %d", len(got.Windows))
	}
	if got.Windows[0].Name != "editor" {
		t.Fatalf("expected window 'editor', got %q", got.Windows[0].Name)
	}
	if len(got.Windows[0].Panes) != 2 {
		t.Fatalf("expected 2 panes in editor, got %d", len(got.Windows[0].Panes))
	}
	if got.Windows[0].Panes[0].Command != "vim" {
		t.Fatalf("expected command 'vim', got %q", got.Windows[0].Panes[0].Command)
	}
}

func TestApplyLayout_Stubbed(t *testing.T) {
	origCmd := tmuxCmd
	defer func() { tmuxCmd = origCmd }()

	var commands []string
	tmuxCmd = func(args ...string) (string, error) {
		commands = append(commands, strings.Join(args, " "))
		return "", nil
	}

	layout := &Layout{
		Name: "dev",
		Windows: []WindowLayout{
			{
				Name:      "editor",
				Layout:    "main-vertical",
				PaneCount: 2,
				Panes: []PaneLayout{
					{Dir: "/home/user/project"},
					{Dir: "/tmp/logs"},
				},
			},
			{
				Name:      "server",
				Layout:    "even-horizontal",
				PaneCount: 1,
				Panes:     []PaneLayout{{Dir: "/var/www"}},
			},
		},
	}

	if err := applyLayout(layout); err != nil {
		t.Fatalf("applyLayout failed: %v", err)
	}

	// Verify the commands issued.
	expected := []string{
		"rename-window editor",
		"split-window -t editor",
		"select-layout -t editor main-vertical",
	}
	for _, exp := range expected {
		found := false
		for _, cmd := range commands {
			if cmd == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected command %q not found in: %v", exp, commands)
		}
	}

	// Verify cd commands use %q quoting.
	cdCount := 0
	for _, cmd := range commands {
		if strings.HasPrefix(cmd, "send-keys") && strings.Contains(cmd, "cd ") {
			cdCount++
			// The dir should be quoted (Go %q produces "..." with escapes).
			if !strings.Contains(cmd, `cd "`) {
				t.Errorf("cd command should use quoted path, got: %s", cmd)
			}
		}
	}
	if cdCount != 3 {
		t.Errorf("expected 3 cd send-keys commands, got %d: %v", cdCount, commands)
	}

	// Verify select-window at end.
	last := commands[len(commands)-1]
	if last != "select-window -t editor" {
		t.Errorf("last command should be select-window, got: %s", last)
	}
}

func TestApplyLayout_ErrorPropagation(t *testing.T) {
	origCmd := tmuxCmd
	defer func() { tmuxCmd = origCmd }()

	tmuxCmd = func(args ...string) (string, error) {
		if args[0] == "new-window" {
			return "", fmt.Errorf("tmux: session not found")
		}
		return "", nil
	}

	layout := &Layout{
		Name: "fail",
		Windows: []WindowLayout{
			{Name: "first", PaneCount: 1},
			{Name: "second", PaneCount: 1},
		},
	}

	err := applyLayout(layout)
	if err == nil {
		t.Fatal("expected error from applyLayout")
	}
	if !strings.Contains(err.Error(), "creating window") {
		t.Fatalf("error should mention creating window, got: %v", err)
	}
}
