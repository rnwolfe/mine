package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rnwolfe/mine/internal/tmux"
)

func TestLayoutItemFilterValue(t *testing.T) {
	item := layoutItem{name: "dev-setup", description: "3 windows"}
	if got := item.FilterValue(); got != "dev-setup" {
		t.Errorf("FilterValue: want %q, got %q", "dev-setup", got)
	}
}

func TestLayoutItemTitle(t *testing.T) {
	item := layoutItem{name: "dev-setup", description: "3 windows"}
	if got := item.Title(); got != "dev-setup" {
		t.Errorf("Title: want %q, got %q", "dev-setup", got)
	}
}

func TestLayoutItemDescription(t *testing.T) {
	item := layoutItem{name: "dev-setup", description: "3 windows"}
	if got := item.Description(); got != "3 windows" {
		t.Errorf("Description: want %q, got %q", "3 windows", got)
	}
}

func TestLayoutItemEmptyDescription(t *testing.T) {
	item := layoutItem{name: "minimal"}
	if got := item.Description(); got != "" {
		t.Errorf("Description: want empty, got %q", got)
	}
}

func TestTmuxLayoutLoadCmdAcceptsZeroArgs(t *testing.T) {
	if err := tmuxLayoutLoadCmd.Args(tmuxLayoutLoadCmd, []string{}); err != nil {
		t.Errorf("expected 0 args to be valid, got: %v", err)
	}
}

func TestTmuxLayoutLoadCmdAcceptsOneArg(t *testing.T) {
	if err := tmuxLayoutLoadCmd.Args(tmuxLayoutLoadCmd, []string{"dev-setup"}); err != nil {
		t.Errorf("expected 1 arg to be valid, got: %v", err)
	}
}

func TestTmuxLayoutLoadCmdRejectsTwoArgs(t *testing.T) {
	if err := tmuxLayoutLoadCmd.Args(tmuxLayoutLoadCmd, []string{"dev-setup", "extra"}); err == nil {
		t.Error("expected 2 args to be rejected, but no error")
	}
}

// setupTmuxEnv creates a minimal tmux stub in PATH and sets TMUX so that
// tmux.Available() and tmux.InsideTmux() return true during the test.
func setupTmuxEnv(t *testing.T) {
	t.Helper()

	stubDir := t.TempDir()
	stub := filepath.Join(stubDir, "tmux")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", stubDir+":"+os.Getenv("PATH"))
	t.Setenv("TMUX", "/tmp/tmux-test,12345,0")
}

// TestRunTmuxLayoutLoadNoLayouts verifies that when no layouts are saved,
// the command prints guidance and returns nil (no error).
func TestRunTmuxLayoutLoadNoLayouts(t *testing.T) {
	setupTmuxEnv(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // empty config dir → no layouts

	err := runTmuxLayoutLoad(nil, []string{})
	if err != nil {
		t.Errorf("expected nil error when no layouts exist, got: %v", err)
	}
}

// TestRunTmuxLayoutLoadNonTTYListsAndErrors verifies that when layouts exist
// and stdin is not a TTY, the command lists them and returns an actionable error.
func TestRunTmuxLayoutLoadNonTTYListsAndErrors(t *testing.T) {
	setupTmuxEnv(t)
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	// Write a layout file so ListLayouts() returns at least one entry.
	layout := &tmux.Layout{
		Name:    "test-layout",
		SavedAt: time.Now(),
		Windows: []tmux.WindowLayout{
			{Name: "main", Layout: "even-horizontal", PaneCount: 1},
		},
	}
	if err := tmux.WriteLayout(layout); err != nil {
		t.Fatal(err)
	}

	// IsTTY() returns false in tests (no terminal attached), so the non-TTY
	// listing branch runs and returns the actionable error.
	err := runTmuxLayoutLoad(nil, []string{})
	if err == nil {
		t.Fatal("expected error for non-TTY path with layouts, got nil")
	}
	if !strings.Contains(err.Error(), "no layout name given") {
		t.Errorf("expected error to mention 'no layout name given', got: %v", err)
	}
	if !strings.Contains(err.Error(), "mine tmux layout load") {
		t.Errorf("expected error to include usage hint, got: %v", err)
	}
}

// TestLayoutItemDescriptionErrorReading verifies that the "(error reading)"
// sentinel value is accepted as a valid description for the picker item.
func TestLayoutItemDescriptionErrorReading(t *testing.T) {
	item := layoutItem{name: "broken", description: "(error reading)"}
	if got := item.Description(); got != "(error reading)" {
		t.Errorf("Description: want %q, got %q", "(error reading)", got)
	}
}
