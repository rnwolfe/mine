package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsGitRepo(t *testing.T) {
	setupEnv(t)

	if IsGitRepo() {
		t.Error("IsGitRepo() = true before InitGitRepo(), want false")
	}

	if err := Init(); err != nil {
		t.Fatal(err)
	}

	if !IsGitRepo() {
		t.Error("IsGitRepo() = false after Init(), want true")
	}
}

func TestInitGitRepo_Idempotent(t *testing.T) {
	setupEnv(t)

	if err := Init(); err != nil {
		t.Fatal(err)
	}

	// InitGitRepo a second time should not fail.
	if err := InitGitRepo(); err != nil {
		t.Fatalf("second InitGitRepo() error: %v", err)
	}
}

func TestCommit(t *testing.T) {
	agentsDir := setupEnv(t)

	if err := Init(); err != nil {
		t.Fatal(err)
	}

	// Write a file so there's something to commit.
	if err := os.WriteFile(filepath.Join(agentsDir, "test.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	hash, err := Commit("test commit")
	if err != nil {
		t.Fatalf("Commit() error: %v", err)
	}
	if hash == "" {
		t.Error("Commit() returned empty hash")
	}
}

func TestCommit_NothingToCommit(t *testing.T) {
	agentsDir := setupEnv(t)

	if err := Init(); err != nil {
		t.Fatal(err)
	}

	// First commit with content.
	if err := os.WriteFile(filepath.Join(agentsDir, "test.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Commit("initial"); err != nil {
		t.Fatal(err)
	}

	// Second commit with no changes should error.
	_, err := Commit("nothing changed")
	if err == nil {
		t.Error("Commit() should error when nothing to commit")
	}
	if !strings.Contains(err.Error(), "nothing to commit") {
		t.Errorf("error = %q, want containing 'nothing to commit'", err.Error())
	}
}

func TestLog(t *testing.T) {
	agentsDir := setupEnv(t)

	if err := Init(); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(agentsDir, "test.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Commit("first commit"); err != nil {
		t.Fatal(err)
	}

	logs, err := Log("")
	if err != nil {
		t.Fatalf("Log() error: %v", err)
	}
	if len(logs) == 0 {
		t.Error("Log() returned no entries after commit")
	}
	if logs[0].Message != "first commit" {
		t.Errorf("Log()[0].Message = %q, want %q", logs[0].Message, "first commit")
	}
}

func TestLog_NoHistory(t *testing.T) {
	setupEnv(t)

	_, err := Log("")
	if err == nil {
		t.Error("Log() should error when no git repo")
	}
}

func TestHasCommits(t *testing.T) {
	setupEnv(t)

	if HasCommits() {
		t.Error("HasCommits() = true before any repo, want false")
	}

	if err := Init(); err != nil {
		t.Fatal(err)
	}

	// Init creates an initial commit.
	if !HasCommits() {
		t.Error("HasCommits() = false after Init(), want true")
	}
}
