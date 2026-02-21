package cmd

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/rnwolfe/mine/internal/tmux"
)

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

	newSessionFunc = func(name string) (string, error) {
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
