package agents

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCommit_NotInitialized(t *testing.T) {
	setupEnv(t)

	_, err := Commit("test")
	if err == nil {
		t.Error("Commit() expected error when store not initialized, got nil")
	}
}

func TestCommit_NothingToCommit(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	// First commit to establish HEAD.
	if _, err := Commit("initial"); err != nil {
		t.Fatalf("first Commit() error: %v", err)
	}

	// Second commit with no changes should return ErrNothingToCommit.
	_, err := Commit("empty")
	if err == nil {
		t.Error("Commit() expected error when nothing to commit, got nil")
	}
	if !errors.Is(err, ErrNothingToCommit) {
		t.Errorf("Commit() error = %v, want ErrNothingToCommit", err)
	}
}

func TestCommit_Success(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	hash, err := Commit("initial setup")
	if err != nil {
		t.Fatalf("Commit() error: %v", err)
	}
	if hash == "" {
		t.Error("Commit() returned empty hash")
	}
}

func TestLog_NoHistory(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}
	if err := InitGitRepo(); err != nil {
		t.Fatal(err)
	}

	entries, err := Log("")
	if err != nil {
		t.Errorf("Log() unexpected error when no git repo commits (expected empty history): %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Log() returned %d entries, want 0 for empty repo", len(entries))
	}
}

func TestLog_NoGitRepo(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	// Remove .git to simulate no repo.
	if err := os.RemoveAll(filepath.Join(Dir(), ".git")); err != nil {
		t.Fatal(err)
	}

	_, err := Log("")
	if err == nil {
		t.Error("Log() expected error when no git repo, got nil")
	}
	if !errors.Is(err, ErrNoVersionHistory) {
		t.Errorf("Log() error = %v, want ErrNoVersionHistory", err)
	}
}

func TestLog_AfterCommit(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	if _, err := Commit("initial setup"); err != nil {
		t.Fatalf("Commit() error: %v", err)
	}

	entries, err := Log("")
	if err != nil {
		t.Fatalf("Log() error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("Log() returned %d entries, want 1", len(entries))
	}
	if entries[0].Message != "initial setup" {
		t.Errorf("entries[0].Message = %q, want %q", entries[0].Message, "initial setup")
	}
	if entries[0].Short == "" {
		t.Error("entries[0].Short is empty")
	}
}

func TestLog_FilteredToFile(t *testing.T) {
	agentsDir := setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	// First commit (includes AGENTS.md from init).
	if _, err := Commit("initial"); err != nil {
		t.Fatalf("first Commit() error: %v", err)
	}

	// Create a new file and commit.
	newFile := filepath.Join(agentsDir, "settings", "config.json")
	if err := os.WriteFile(newFile, []byte(`{"key":"value"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Commit("add config"); err != nil {
		t.Fatalf("second Commit() error: %v", err)
	}

	// Log filtered to the new file should show 1 entry.
	entries, err := Log("settings/config.json")
	if err != nil {
		t.Fatalf("Log(settings/config.json) error: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("Log(settings/config.json) returned %d entries, want 1", len(entries))
	}

	// Log filtered to AGENTS.md should also show 1 entry.
	entries, err = Log("instructions/AGENTS.md")
	if err != nil {
		t.Fatalf("Log(instructions/AGENTS.md) error: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("Log(instructions/AGENTS.md) returned %d entries, want 1", len(entries))
	}
}
