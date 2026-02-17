package tmux

import (
	"os"
	"path/filepath"
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
