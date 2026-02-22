package cmd

import (
	"fmt"
	"os"
	"os/exec"
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

// TestRunTmuxLayoutHelp_NotInsideTmux_ShowsHelp verifies that the help text is shown
// when not inside a tmux session (TMUX env var unset).
func TestRunTmuxLayoutHelp_NotInsideTmux_ShowsHelp(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("TMUX", "") // ensure not inside tmux

	output := captureStdout(t, func() {
		if err := runTmuxLayoutHelp(nil, []string{}); err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
	})

	for _, want := range []string{
		"mine tmux layout save",
		"mine tmux layout load",
		"mine tmux layout ls",
		"mine tmux layout delete",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("help output missing %q\nGot:\n%s", want, output)
		}
	}
}

// TestRunTmuxLayoutHelp_InsideTmuxTmuxNotAvailable verifies that an error is returned
// when TMUX is set but the tmux binary is not in PATH.
func TestRunTmuxLayoutHelp_InsideTmuxTmuxNotAvailable(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("TMUX", "/tmp/tmux-test,12345,0") // inside tmux
	// Do NOT add tmux to PATH — Available() must return false.
	t.Setenv("PATH", t.TempDir())

	err := runTmuxLayoutHelp(nil, []string{})
	if err == nil {
		t.Fatal("expected error when tmux not in PATH, got nil")
	}
	if !strings.Contains(err.Error(), "tmux not found in PATH") {
		t.Errorf("expected 'tmux not found in PATH' error, got: %v", err)
	}
}

// TestRunTmuxLayoutHelp_InsideTmuxNoLayouts verifies that when inside tmux but no
// layouts are saved, an informative message is printed and nil is returned.
func TestRunTmuxLayoutHelp_InsideTmuxNoLayouts(t *testing.T) {
	setupTmuxEnv(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // empty config dir → no layouts

	output := captureStdout(t, func() {
		if err := runTmuxLayoutHelp(nil, []string{}); err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
	})

	if !strings.Contains(output, "No saved layouts") {
		t.Errorf("expected 'No saved layouts' message, got:\n%s", output)
	}
	if !strings.Contains(output, "mine tmux layout save") {
		t.Errorf("expected save hint in output, got:\n%s", output)
	}
}

// TestRunTmuxLayoutHelp_InsideTmuxNonTTY_ShowsHelp verifies that when inside tmux
// but not a TTY, the help text is shown (the picker requires a TTY).
func TestRunTmuxLayoutHelp_InsideTmuxNonTTY_ShowsHelp(t *testing.T) {
	setupTmuxEnv(t)
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	// Write a layout so the no-layouts branch is not triggered.
	layout := &tmux.Layout{
		Name:    "dev-setup",
		Windows: []tmux.WindowLayout{{Name: "editor", PaneCount: 1}},
	}
	if err := tmux.WriteLayout(layout); err != nil {
		t.Fatal(err)
	}

	// IsTTY() returns false in tests (no terminal attached), so the help text
	// branch should run even though we are "inside" tmux.
	output := captureStdout(t, func() {
		if err := runTmuxLayoutHelp(nil, []string{}); err != nil {
			t.Errorf("expected nil error, got: %v", err)
		}
	})

	for _, want := range []string{
		"mine tmux layout save",
		"mine tmux layout load",
		"mine tmux layout ls",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("help output missing %q\nGot:\n%s", want, output)
		}
	}
}

// TestTmuxLayoutPreviewCmdArgs verifies arg count enforcement on the preview command.
func TestTmuxLayoutPreviewCmdArgs(t *testing.T) {
	if err := tmuxLayoutPreviewCmd.Args(tmuxLayoutPreviewCmd, []string{"dev-setup"}); err != nil {
		t.Errorf("expected 1 arg to be valid, got: %v", err)
	}
	if err := tmuxLayoutPreviewCmd.Args(tmuxLayoutPreviewCmd, []string{}); err == nil {
		t.Error("expected 0 args to be rejected, but got nil error")
	}
	if err := tmuxLayoutPreviewCmd.Args(tmuxLayoutPreviewCmd, []string{"a", "b"}); err == nil {
		t.Error("expected 2 args to be rejected, but got nil error")
	}
}

// TestRunTmuxLayoutPreviewNotFound verifies that previewing a missing layout returns an error.
func TestRunTmuxLayoutPreviewNotFound(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	err := runTmuxLayoutPreview(nil, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for nonexistent layout, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("expected error to mention layout name, got: %v", err)
	}
}

// TestRunTmuxLayoutPreviewOutput verifies that previewing a known layout prints
// the layout name, save timestamp, window names, pane counts, and directories.
func TestRunTmuxLayoutPreviewOutput(t *testing.T) {
	configDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configDir)

	savedAt := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)
	layout := &tmux.Layout{
		Name:    "dev-setup",
		SavedAt: savedAt,
		Windows: []tmux.WindowLayout{
			{
				Name:      "editor",
				PaneCount: 2,
				Panes:     []tmux.PaneLayout{{Dir: "/home/user/code"}},
			},
			{
				Name:      "server",
				PaneCount: 1,
				Panes:     []tmux.PaneLayout{{Dir: "/home/user/code/api"}},
			},
		},
	}
	if err := tmux.WriteLayout(layout); err != nil {
		t.Fatal(err)
	}

	var runErr error
	output := captureStdout(t, func() {
		runErr = runTmuxLayoutPreview(nil, []string{"dev-setup"})
	})

	if runErr != nil {
		t.Fatalf("expected no error, got: %v", runErr)
	}

	for _, want := range []string{
		"dev-setup",
		"2024-01-15",
		"editor",
		"server",
		"/home/user/code",
		"/home/user/code/api",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q\nGot:\n%s", want, output)
		}
	}
}

// TestRunTmuxNew_InvalidLayoutErrors verifies that runTmuxNew returns an error
// when the requested layout does not exist, without creating a session.
// Also checks that the error message contains no embedded newline (regression
// against the original `\n` in the format string).
func TestRunTmuxNew_InvalidLayoutErrors(t *testing.T) {
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not in PATH")
	}

	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	orig := tmuxNewLayout
	tmuxNewLayout = "nonexistent-layout"
	defer func() { tmuxNewLayout = orig }()

	err := runTmuxNew(nil, []string{"test-session-invalid"})
	if err == nil {
		t.Fatal("expected error for nonexistent layout")
	}
	if !strings.Contains(err.Error(), "nonexistent-layout") {
		t.Errorf("error should mention layout name, got: %v", err)
	}
	if strings.Contains(err.Error(), "\n") {
		t.Errorf("error message should not contain embedded newline, got: %v", err)
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error should say 'not found', got: %v", err)
	}
}

// TestRunTmuxNew_CleanupOnLoadFailure verifies that when LoadLayout fails after
// a session is successfully created, KillSession is called to clean up the
// orphaned session — maintaining the "no side effects on error" invariant.
func TestRunTmuxNew_CleanupOnLoadFailure(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	// Write a valid layout file so ReadLayout succeeds.
	layout := &tmux.Layout{
		Name:    "dev",
		Windows: []tmux.WindowLayout{{Name: "editor", PaneCount: 1}},
	}
	if err := tmux.WriteLayout(layout); err != nil {
		t.Fatalf("WriteLayout: %v", err)
	}

	var killCalledFor string

	origNew := newSessionFunc
	origLoad := loadLayoutFunc
	origKill := killSessionFunc
	defer func() {
		newSessionFunc = origNew
		loadLayoutFunc = origLoad
		killSessionFunc = origKill
	}()

	newSessionFunc = func(name, dir string) (string, error) {
		return name, nil // session creation succeeds
	}
	loadLayoutFunc = func(_ string) error {
		return fmt.Errorf("simulated layout apply failure")
	}
	killSessionFunc = func(name string) error {
		killCalledFor = name
		return nil
	}

	orig := tmuxNewLayout
	tmuxNewLayout = "dev"
	defer func() { tmuxNewLayout = orig }()

	// Also bypass the tmux.Available() check by setting a no-op readLayout stub
	// that returns the pre-written layout from our temp dir — Available() still
	// calls the real exec.LookPath, so skip if tmux is not installed.
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not in PATH")
	}

	err := runTmuxNew(nil, []string{"test-cleanup-session"})
	if err == nil {
		t.Fatal("expected error when LoadLayout fails")
	}
	if killCalledFor == "" {
		t.Error("KillSession should have been called to clean up the orphaned session")
	}
	if killCalledFor != "test-cleanup-session" {
		t.Errorf("KillSession called for %q, expected %q", killCalledFor, "test-cleanup-session")
	}
	if !strings.Contains(err.Error(), "cleaned up") {
		t.Errorf("error should mention session was cleaned up, got: %v", err)
	}
}
