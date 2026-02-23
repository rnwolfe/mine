package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rnwolfe/mine/internal/agents"
	"github.com/spf13/cobra"
)

// agentsTestEnv sets up an isolated XDG environment for agents cmd tests.
func agentsTestEnv(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmpDir, "data"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmpDir, "cache"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmpDir, "state"))
	t.Setenv("HOME", filepath.Join(tmpDir, "home"))
}

// agentsCmdWithFlags returns a stub cobra.Command with agents flags registered,
// matching what init() registers on the real command objects.
func agentsCommitStub() *cobra.Command {
	cmd := &cobra.Command{Use: "commit"}
	cmd.Flags().StringP("message", "m", "", "Commit message")
	return cmd
}

func agentsRestoreStub() *cobra.Command {
	cmd := &cobra.Command{Use: "restore"}
	cmd.Flags().StringP("version", "v", "", "Version hash to restore")
	return cmd
}

// TestRunAgentsInit_FirstTime verifies that init on a fresh env creates the store.
func TestRunAgentsInit_FirstTime(t *testing.T) {
	agentsTestEnv(t)

	out := captureStdout(t, func() {
		err := runAgentsInit(stubCmd(), nil)
		if err != nil {
			t.Errorf("runAgentsInit: %v", err)
		}
	})

	if !agents.IsInitialized() {
		t.Error("agents store not initialized after runAgentsInit")
	}
	if !strings.Contains(out, "Agents store ready") {
		t.Errorf("expected 'Agents store ready' in output, got: %q", out)
	}
}

// TestRunAgentsInit_AlreadyInitialized verifies idempotent re-init shows correct message.
func TestRunAgentsInit_AlreadyInitialized(t *testing.T) {
	agentsTestEnv(t)

	// Initialize first.
	if err := agents.Init(); err != nil {
		t.Fatalf("agents.Init: %v", err)
	}

	out := captureStdout(t, func() {
		err := runAgentsInit(stubCmd(), nil)
		if err != nil {
			t.Errorf("runAgentsInit re-init: %v", err)
		}
	})

	if !strings.Contains(out, "already initialized") {
		t.Errorf("expected 'already initialized' in output, got: %q", out)
	}
}

// TestRunAgentsCommit_NotInitialized verifies that commit returns error when store is absent.
func TestRunAgentsCommit_NotInitialized(t *testing.T) {
	agentsTestEnv(t)

	err := runAgentsCommit(agentsCommitStub(), nil)
	if err == nil {
		t.Fatal("expected error when store not initialized, got nil")
	}
	if !strings.Contains(err.Error(), "mine agents init") {
		t.Errorf("expected 'mine agents init' hint in error, got: %v", err)
	}
}

// TestRunAgentsCommit_NothingToCommit verifies the friendly message path.
func TestRunAgentsCommit_NothingToCommit(t *testing.T) {
	agentsTestEnv(t)

	if err := agents.Init(); err != nil {
		t.Fatalf("agents.Init: %v", err)
	}

	// First commit to establish HEAD.
	if _, err := agents.Commit("initial"); err != nil {
		t.Fatalf("agents.Commit initial: %v", err)
	}

	// Now commit again with no changes — should be friendly, not an error.
	out := captureStdout(t, func() {
		err := runAgentsCommit(agentsCommitStub(), nil)
		if err != nil {
			t.Errorf("runAgentsCommit nothing-to-commit: unexpected error %v", err)
		}
	})

	if !strings.Contains(out, "Nothing to commit") {
		t.Errorf("expected 'Nothing to commit' in output, got: %q", out)
	}
}

// TestRunAgentsCommit_Success verifies a successful commit shows the hash.
func TestRunAgentsCommit_Success(t *testing.T) {
	agentsTestEnv(t)

	if err := agents.Init(); err != nil {
		t.Fatalf("agents.Init: %v", err)
	}

	out := captureStdout(t, func() {
		err := runAgentsCommit(agentsCommitStub(), nil)
		if err != nil {
			t.Errorf("runAgentsCommit: %v", err)
		}
	})

	if !strings.Contains(out, "Snapshot saved") {
		t.Errorf("expected 'Snapshot saved' in output, got: %q", out)
	}
}

// TestRunAgentsLog_NotInitialized verifies that log returns error when store is absent.
func TestRunAgentsLog_NotInitialized(t *testing.T) {
	agentsTestEnv(t)

	err := runAgentsLog(stubCmd(), nil)
	if err == nil {
		t.Fatal("expected error when store not initialized, got nil")
	}
}

// TestRunAgentsLog_NoHistory verifies the friendly "no history yet" message.
func TestRunAgentsLog_NoHistory(t *testing.T) {
	agentsTestEnv(t)

	if err := agents.Init(); err != nil {
		t.Fatalf("agents.Init: %v", err)
	}

	// Initialized but no commits yet — Log() returns nil,nil (empty history).
	out := captureStdout(t, func() {
		err := runAgentsLog(stubCmd(), nil)
		if err != nil {
			t.Errorf("runAgentsLog no-history: unexpected error %v", err)
		}
	})

	if !strings.Contains(out, "No history yet") {
		t.Errorf("expected 'No history yet' in output, got: %q", out)
	}
}

// TestRunAgentsLog_NoVersionHistory verifies friendly message when git repo is absent.
func TestRunAgentsLog_NoVersionHistory(t *testing.T) {
	agentsTestEnv(t)

	if err := agents.Init(); err != nil {
		t.Fatalf("agents.Init: %v", err)
	}

	// Remove .git to trigger ErrNoVersionHistory.
	if err := os.RemoveAll(filepath.Join(agents.Dir(), ".git")); err != nil {
		t.Fatalf("removing .git: %v", err)
	}

	out := captureStdout(t, func() {
		err := runAgentsLog(stubCmd(), nil)
		if err != nil {
			t.Errorf("runAgentsLog no-version-history: unexpected error %v", err)
		}
	})

	if !strings.Contains(out, "No history yet") {
		t.Errorf("expected 'No history yet' in output, got: %q", out)
	}
}

// TestRunAgentsLog_WithHistory verifies that entries are rendered when commits exist.
func TestRunAgentsLog_WithHistory(t *testing.T) {
	agentsTestEnv(t)

	if err := agents.Init(); err != nil {
		t.Fatalf("agents.Init: %v", err)
	}

	if _, err := agents.Commit("my test snapshot"); err != nil {
		t.Fatalf("agents.Commit: %v", err)
	}

	out := captureStdout(t, func() {
		err := runAgentsLog(stubCmd(), nil)
		if err != nil {
			t.Errorf("runAgentsLog: %v", err)
		}
	})

	if !strings.Contains(out, "my test snapshot") {
		t.Errorf("expected commit message in output, got: %q", out)
	}
	if !strings.Contains(out, "1 snapshots") {
		t.Errorf("expected '1 snapshots' in output, got: %q", out)
	}
}

// TestRunAgentsRestore_NotInitialized verifies that restore errors when store is absent.
func TestRunAgentsRestore_NotInitialized(t *testing.T) {
	agentsTestEnv(t)

	err := runAgentsRestore(agentsRestoreStub(), []string{"instructions/AGENTS.md"})
	if err == nil {
		t.Fatal("expected error when store not initialized, got nil")
	}
	if !strings.Contains(err.Error(), "mine agents init") {
		t.Errorf("expected 'mine agents init' hint in error, got: %v", err)
	}
}

// TestRunAgentsRestore_Success verifies restore output when it succeeds.
func TestRunAgentsRestore_Success(t *testing.T) {
	agentsDir := agentsTestEnvDir(t)

	if err := agents.Init(); err != nil {
		t.Fatalf("agents.Init: %v", err)
	}

	// Write a file and commit.
	testFile := filepath.Join(agentsDir, "instructions", "AGENTS.md")
	if err := os.WriteFile(testFile, []byte("v1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := agents.Commit("v1"); err != nil {
		t.Fatalf("agents.Commit: %v", err)
	}

	// Modify the file.
	if err := os.WriteFile(testFile, []byte("v2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() {
		err := runAgentsRestore(agentsRestoreStub(), []string{"instructions/AGENTS.md"})
		if err != nil {
			t.Errorf("runAgentsRestore: %v", err)
		}
	})

	if !strings.Contains(out, "Restored") {
		t.Errorf("expected 'Restored' in output, got: %q", out)
	}
}

// TestRunAgentsStatus_NotInitialized verifies that status shows guidance when no store.
func TestRunAgentsStatus_NotInitialized(t *testing.T) {
	agentsTestEnv(t)

	out := captureStdout(t, func() {
		err := runAgentsStatus(stubCmd(), nil)
		if err != nil {
			t.Errorf("runAgentsStatus: %v", err)
		}
	})

	if !strings.Contains(out, "mine agents init") {
		t.Errorf("expected 'mine agents init' hint in output, got: %q", out)
	}
}

// TestRunAgentsStatus_Initialized verifies that status shows store info when initialized.
func TestRunAgentsStatus_Initialized(t *testing.T) {
	agentsTestEnv(t)

	if err := agents.Init(); err != nil {
		t.Fatalf("agents.Init: %v", err)
	}

	out := captureStdout(t, func() {
		err := runAgentsStatus(stubCmd(), nil)
		if err != nil {
			t.Errorf("runAgentsStatus: %v", err)
		}
	})

	if !strings.Contains(out, agents.Dir()) {
		t.Errorf("expected store dir in output, got: %q", out)
	}
}

// agentsTestEnvDir sets up the test env and returns the agents store directory.
func agentsTestEnvDir(t *testing.T) string {
	t.Helper()
	agentsTestEnv(t)
	return agents.Dir()
}
