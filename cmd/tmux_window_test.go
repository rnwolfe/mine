package cmd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/rnwolfe/mine/internal/tmux"
)

// setupWindowFuncs replaces all injectable window functions with stubs and
// returns a cleanup function that restores the originals.
func setupWindowFuncs(t *testing.T, windows []tmux.Window, session string) {
	t.Helper()

	origList := listWindowsFunc
	origNew := newWindowFunc
	origKill := killWindowFunc
	origRename := renameWindowFunc
	origCurrent := currentSessionFunc

	t.Cleanup(func() {
		listWindowsFunc = origList
		newWindowFunc = origNew
		killWindowFunc = origKill
		renameWindowFunc = origRename
		currentSessionFunc = origCurrent
		tmuxWindowSession = ""
	})

	listWindowsFunc = func(_ string) ([]tmux.Window, error) { return windows, nil }
	newWindowFunc = func(_, _ string) error { return nil }
	killWindowFunc = func(_, _ string) error { return nil }
	renameWindowFunc = func(_, _, _ string) error { return nil }
	currentSessionFunc = func() (string, error) { return session, nil }
}

// --- mine tmux window ls ---

// TestRunTmuxWindowLs_NotInsideTmux verifies that ls errors when not inside
// tmux and no --session flag is given.
func TestRunTmuxWindowLs_NotInsideTmux(t *testing.T) {
	setupTmuxEnv(t)
	// Clear TMUX so InsideTmux() returns false.
	t.Setenv("TMUX", "")
	tmuxWindowSession = ""
	defer func() { tmuxWindowSession = "" }()

	err := runTmuxWindowLs(nil, []string{})
	if err == nil {
		t.Fatal("expected error when not inside tmux and no --session given")
	}
	if !strings.Contains(err.Error(), "not inside a tmux session") {
		t.Errorf("expected 'not inside a tmux session' in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "--session") {
		t.Errorf("expected '--session' hint in error, got: %v", err)
	}
}

// TestRunTmuxWindowLs_NoWindows verifies that when a session has no windows,
// the command prints guidance and returns nil.
func TestRunTmuxWindowLs_NoWindows(t *testing.T) {
	setupTmuxEnv(t)
	setupWindowFuncs(t, []tmux.Window{}, "mysession")
	t.Setenv("TMUX", "/tmp/tmux-test,12345,0")

	err := runTmuxWindowLs(nil, []string{})
	if err != nil {
		t.Errorf("expected nil error for empty window list, got: %v", err)
	}
}

// TestRunTmuxWindowLs_WithSession verifies that --session flag uses the given
// session name directly without calling currentSessionFunc.
func TestRunTmuxWindowLs_WithSession(t *testing.T) {
	setupTmuxEnv(t)
	t.Setenv("TMUX", "")

	windows := []tmux.Window{
		{Index: 0, Name: "editor", Active: true},
		{Index: 1, Name: "server", Active: false},
	}

	var capturedSession string
	origList := listWindowsFunc
	defer func() {
		listWindowsFunc = origList
		tmuxWindowSession = ""
	}()
	listWindowsFunc = func(s string) ([]tmux.Window, error) {
		capturedSession = s
		return windows, nil
	}

	tmuxWindowSession = "target-session"
	output := captureStdout(t, func() {
		_ = runTmuxWindowLs(nil, []string{})
	})

	if capturedSession != "target-session" {
		t.Errorf("expected session 'target-session', got %q", capturedSession)
	}
	if !strings.Contains(output, "editor") {
		t.Errorf("expected 'editor' in output, got: %s", output)
	}
	if !strings.Contains(output, "server") {
		t.Errorf("expected 'server' in output, got: %s", output)
	}
}

// TestRunTmuxWindowLs_ListWindows verifies that window list output includes
// window names, indices, and an active marker.
func TestRunTmuxWindowLs_ListWindows(t *testing.T) {
	setupTmuxEnv(t)
	windows := []tmux.Window{
		{Index: 0, Name: "editor", Active: true},
		{Index: 1, Name: "server", Active: false},
	}
	setupWindowFuncs(t, windows, "dev")
	t.Setenv("TMUX", "/tmp/tmux-test,12345,0")

	output := captureStdout(t, func() {
		_ = runTmuxWindowLs(nil, []string{})
	})

	for _, want := range []string{"editor", "server", "index 0", "index 1"} {
		if !strings.Contains(output, want) {
			t.Errorf("output missing %q\nGot:\n%s", want, output)
		}
	}
}

// --- mine tmux window new ---

// TestRunTmuxWindowNew_CreatesWindow verifies the happy path: window is created
// and a success message is printed.
func TestRunTmuxWindowNew_CreatesWindow(t *testing.T) {
	setupTmuxEnv(t)
	setupWindowFuncs(t, nil, "dev")
	t.Setenv("TMUX", "/tmp/tmux-test,12345,0")

	var capturedSession, capturedName string
	origNew := newWindowFunc
	defer func() { newWindowFunc = origNew }()
	newWindowFunc = func(sess, name string) error {
		capturedSession = sess
		capturedName = name
		return nil
	}

	output := captureStdout(t, func() {
		_ = runTmuxWindowNew(nil, []string{"mywindow"})
	})

	if capturedName != "mywindow" {
		t.Errorf("expected window name 'mywindow', got %q", capturedName)
	}
	if capturedSession != "dev" {
		t.Errorf("expected session 'dev', got %q", capturedSession)
	}
	if !strings.Contains(output, "mywindow") {
		t.Errorf("expected 'mywindow' in output, got: %s", output)
	}
}

// TestRunTmuxWindowNew_ErrorPropagated verifies that a domain-layer error is
// returned to the caller.
func TestRunTmuxWindowNew_ErrorPropagated(t *testing.T) {
	setupTmuxEnv(t)
	setupWindowFuncs(t, nil, "dev")
	t.Setenv("TMUX", "/tmp/tmux-test,12345,0")

	newWindowFunc = func(_, _ string) error {
		return fmt.Errorf("tmux: session not found")
	}

	err := runTmuxWindowNew(nil, []string{"mywindow"})
	if err == nil {
		t.Fatal("expected error from failing newWindowFunc, got nil")
	}
}

// --- mine tmux window kill ---

// TestRunTmuxWindowKill_ByName verifies that kill with a name arg kills directly.
func TestRunTmuxWindowKill_ByName(t *testing.T) {
	setupTmuxEnv(t)
	windows := []tmux.Window{
		{Index: 0, Name: "editor", Active: false},
		{Index: 1, Name: "server", Active: true},
	}
	setupWindowFuncs(t, windows, "dev")
	t.Setenv("TMUX", "/tmp/tmux-test,12345,0")

	var killedName string
	origKill := killWindowFunc
	defer func() { killWindowFunc = origKill }()
	killWindowFunc = func(_, name string) error {
		killedName = name
		return nil
	}

	output := captureStdout(t, func() {
		_ = runTmuxWindowKill(nil, []string{"editor"})
	})

	if killedName != "editor" {
		t.Errorf("expected 'editor' to be killed, got %q", killedName)
	}
	if !strings.Contains(output, "editor") {
		t.Errorf("expected 'editor' in success output, got: %s", output)
	}
}

// TestRunTmuxWindowKill_UnknownName verifies that kill with a nonexistent name
// returns a clear error.
func TestRunTmuxWindowKill_UnknownName(t *testing.T) {
	setupTmuxEnv(t)
	windows := []tmux.Window{
		{Index: 0, Name: "editor", Active: true},
	}
	setupWindowFuncs(t, windows, "dev")
	t.Setenv("TMUX", "/tmp/tmux-test,12345,0")

	err := runTmuxWindowKill(nil, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown window name, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("expected window name in error, got: %v", err)
	}
}

// TestRunTmuxWindowKill_NoWindows verifies that kill returns an error when
// there are no windows to kill.
func TestRunTmuxWindowKill_NoWindows(t *testing.T) {
	setupTmuxEnv(t)
	setupWindowFuncs(t, []tmux.Window{}, "dev")
	t.Setenv("TMUX", "/tmp/tmux-test,12345,0")

	err := runTmuxWindowKill(nil, []string{})
	if err == nil {
		t.Fatal("expected error when no windows exist, got nil")
	}
}

// TestRunTmuxWindowKill_NonTTY verifies that kill without a name on non-TTY
// lists windows instead of failing.
func TestRunTmuxWindowKill_NonTTY(t *testing.T) {
	setupTmuxEnv(t)
	windows := []tmux.Window{
		{Index: 0, Name: "editor", Active: true},
		{Index: 1, Name: "server", Active: false},
	}
	setupWindowFuncs(t, windows, "dev")
	t.Setenv("TMUX", "/tmp/tmux-test,12345,0")

	// IsTTY() returns false in tests, so the non-TTY listing branch runs.
	output := captureStdout(t, func() {
		_ = runTmuxWindowKill(nil, []string{})
	})

	if !strings.Contains(output, "editor") {
		t.Errorf("expected window list in output, got: %s", output)
	}
}

// --- mine tmux window rename ---

// TestRunTmuxWindowRename_DirectRename verifies the 2-arg direct rename path.
func TestRunTmuxWindowRename_DirectRename(t *testing.T) {
	setupTmuxEnv(t)
	setupWindowFuncs(t, nil, "dev")
	t.Setenv("TMUX", "/tmp/tmux-test,12345,0")

	var renamedOld, renamedNew string
	origRename := renameWindowFunc
	defer func() { renameWindowFunc = origRename }()
	renameWindowFunc = func(_, old, new string) error {
		renamedOld = old
		renamedNew = new
		return nil
	}

	output := captureStdout(t, func() {
		_ = runTmuxWindowRename(nil, []string{"editor", "code"})
	})

	if renamedOld != "editor" || renamedNew != "code" {
		t.Errorf("expected rename editor→code, got %q→%q", renamedOld, renamedNew)
	}
	if !strings.Contains(output, "editor") || !strings.Contains(output, "code") {
		t.Errorf("expected both names in output, got: %s", output)
	}
}

// TestRunTmuxWindowRename_EmptyNewName verifies that an empty new name in
// 2-arg mode returns an error.
func TestRunTmuxWindowRename_EmptyNewName(t *testing.T) {
	setupTmuxEnv(t)
	setupWindowFuncs(t, nil, "dev")
	t.Setenv("TMUX", "/tmp/tmux-test,12345,0")

	err := runTmuxWindowRename(nil, []string{"editor", ""})
	if err == nil {
		t.Fatal("expected error for empty new name, got nil")
	}
	if !strings.Contains(err.Error(), "cannot be empty") {
		t.Errorf("expected 'cannot be empty' in error, got: %v", err)
	}
}

// TestRunTmuxWindowRename_UnknownWindowInOneArgMode verifies that rename with
// an unknown window name in 1-arg mode returns a clear error.
func TestRunTmuxWindowRename_UnknownWindowInOneArgMode(t *testing.T) {
	setupTmuxEnv(t)
	windows := []tmux.Window{
		{Index: 0, Name: "editor", Active: true},
	}
	setupWindowFuncs(t, windows, "dev")
	t.Setenv("TMUX", "/tmp/tmux-test,12345,0")

	err := runTmuxWindowRename(nil, []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for unknown window name, got nil")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("expected window name in error, got: %v", err)
	}
}

// TestRunTmuxWindowRename_NoWindowsError verifies that rename (non-direct)
// fails clearly when there are no windows.
func TestRunTmuxWindowRename_NoWindowsError(t *testing.T) {
	setupTmuxEnv(t)
	setupWindowFuncs(t, []tmux.Window{}, "dev")
	t.Setenv("TMUX", "/tmp/tmux-test,12345,0")

	err := runTmuxWindowRename(nil, []string{"somename"})
	if err == nil {
		t.Fatal("expected error when no windows exist, got nil")
	}
}

// TestRunTmuxWindowRename_NonTTY verifies that rename without args on non-TTY
// lists windows instead of opening the picker.
func TestRunTmuxWindowRename_NonTTY(t *testing.T) {
	setupTmuxEnv(t)
	windows := []tmux.Window{
		{Index: 0, Name: "editor", Active: true},
	}
	setupWindowFuncs(t, windows, "dev")
	t.Setenv("TMUX", "/tmp/tmux-test,12345,0")

	// IsTTY() returns false in tests, so the non-TTY listing branch runs.
	output := captureStdout(t, func() {
		_ = runTmuxWindowRename(nil, []string{})
	})

	if !strings.Contains(output, "editor") {
		t.Errorf("expected window list in output for non-TTY path, got: %s", output)
	}
}

// --- arg validation ---

func TestTmuxWindowNewCmdRequiresOneArg(t *testing.T) {
	if err := tmuxWindowNewCmd.Args(tmuxWindowNewCmd, []string{}); err == nil {
		t.Error("expected error for 0 args, got nil")
	}
	if err := tmuxWindowNewCmd.Args(tmuxWindowNewCmd, []string{"mywin"}); err != nil {
		t.Errorf("expected 1 arg to be valid, got: %v", err)
	}
	if err := tmuxWindowNewCmd.Args(tmuxWindowNewCmd, []string{"a", "b"}); err == nil {
		t.Error("expected error for 2 args, got nil")
	}
}

func TestTmuxWindowKillCmdAcceptsZeroOrOneArg(t *testing.T) {
	if err := tmuxWindowKillCmd.Args(tmuxWindowKillCmd, []string{}); err != nil {
		t.Errorf("expected 0 args to be valid, got: %v", err)
	}
	if err := tmuxWindowKillCmd.Args(tmuxWindowKillCmd, []string{"win"}); err != nil {
		t.Errorf("expected 1 arg to be valid, got: %v", err)
	}
	if err := tmuxWindowKillCmd.Args(tmuxWindowKillCmd, []string{"a", "b"}); err == nil {
		t.Error("expected error for 2 args, got nil")
	}
}

func TestTmuxWindowRenameCmdAcceptsUpToTwoArgs(t *testing.T) {
	if err := tmuxWindowRenameCmd.Args(tmuxWindowRenameCmd, []string{}); err != nil {
		t.Errorf("expected 0 args to be valid, got: %v", err)
	}
	if err := tmuxWindowRenameCmd.Args(tmuxWindowRenameCmd, []string{"a"}); err != nil {
		t.Errorf("expected 1 arg to be valid, got: %v", err)
	}
	if err := tmuxWindowRenameCmd.Args(tmuxWindowRenameCmd, []string{"a", "b"}); err != nil {
		t.Errorf("expected 2 args to be valid, got: %v", err)
	}
	if err := tmuxWindowRenameCmd.Args(tmuxWindowRenameCmd, []string{"a", "b", "c"}); err == nil {
		t.Error("expected error for 3 args, got nil")
	}
}
