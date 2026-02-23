package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncRemoteURL_Empty(t *testing.T) {
	setupEnv(t)

	// No git repo at all.
	url := SyncRemoteURL()
	if url != "" {
		t.Errorf("SyncRemoteURL() = %q, want empty when no repo", url)
	}
}

func TestSyncSetRemote_NoGitRepo(t *testing.T) {
	setupEnv(t)

	err := SyncSetRemote("https://example.com/agents.git")
	if err == nil {
		t.Error("SyncSetRemote() should error without git repo")
	}
}

func TestSyncSetRemote(t *testing.T) {
	agentsDir := setupEnv(t)

	if err := Init(); err != nil {
		t.Fatal(err)
	}

	// Need at least one commit for a valid git repo.
	if err := os.WriteFile(filepath.Join(agentsDir, "test.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Commit("initial"); err != nil {
		t.Fatal(err)
	}

	// Set remote.
	if err := SyncSetRemote("https://example.com/agents.git"); err != nil {
		t.Fatalf("SyncSetRemote() error: %v", err)
	}

	url := SyncRemoteURL()
	if url != "https://example.com/agents.git" {
		t.Errorf("SyncRemoteURL() = %q, want %q", url, "https://example.com/agents.git")
	}

	// Update remote (set-url path).
	if err := SyncSetRemote("https://example.com/new-agents.git"); err != nil {
		t.Fatalf("SyncSetRemote(update) error: %v", err)
	}

	url = SyncRemoteURL()
	if url != "https://example.com/new-agents.git" {
		t.Errorf("SyncRemoteURL() = %q, want %q", url, "https://example.com/new-agents.git")
	}
}

func TestSyncPush_NoRemote(t *testing.T) {
	agentsDir := setupEnv(t)

	if err := Init(); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(agentsDir, "test.txt"), []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Commit("initial"); err != nil {
		t.Fatal(err)
	}

	err := SyncPush()
	if err == nil {
		t.Error("SyncPush() should error without configured remote")
	}
	if !strings.Contains(err.Error(), "no remote configured") {
		t.Errorf("error = %q, want containing 'no remote configured'", err.Error())
	}
}

func TestSyncPull_NoRemote(t *testing.T) {
	agentsDir := setupEnv(t)

	if err := Init(); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(agentsDir, "test.txt"), []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Commit("initial"); err != nil {
		t.Fatal(err)
	}

	err := SyncPull()
	if err == nil {
		t.Error("SyncPull() should error without configured remote")
	}
	if !strings.Contains(err.Error(), "no remote configured") {
		t.Errorf("error = %q, want containing 'no remote configured'", err.Error())
	}
}

func TestSyncPush_NoGitRepo(t *testing.T) {
	setupEnv(t)

	err := SyncPush()
	if err == nil {
		t.Error("SyncPush() should error without git repo")
	}
}

func TestSyncPull_NoGitRepo(t *testing.T) {
	setupEnv(t)

	err := SyncPull()
	if err == nil {
		t.Error("SyncPull() should error without git repo")
	}
}

// TestSyncPullWithResult_CopyModeLinks verifies that copy-mode links are
// re-distributed after a pull. This test uses a local bare repo as the remote.
func TestSyncPullWithResult_CopyModeLinks(t *testing.T) {
	agentsDir := setupEnv(t)
	tmpDir := t.TempDir()

	if err := Init(); err != nil {
		t.Fatal(err)
	}

	// Create initial content and commit.
	if err := os.WriteFile(filepath.Join(agentsDir, "test.txt"), []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Commit("initial commit"); err != nil {
		t.Fatal(err)
	}

	// Set up a bare repo as the "remote".
	bareRepo := filepath.Join(tmpDir, "remote.git")
	if _, err := gitCmd(tmpDir, "init", "--bare", bareRepo); err != nil {
		t.Fatalf("creating bare repo: %v", err)
	}

	// Push to bare repo.
	if err := SyncSetRemote(bareRepo); err != nil {
		t.Fatalf("SyncSetRemote() error: %v", err)
	}
	if err := SyncPush(); err != nil {
		t.Fatalf("SyncPush() error: %v", err)
	}

	// Add a copy-mode link entry to the manifest, pointing to a temp target.
	targetDir := filepath.Join(tmpDir, "target")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	targetFile := filepath.Join(targetDir, "CLAUDE.md")

	// Write initial target file.
	if err := os.WriteFile(targetFile, []byte("old content"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Also need the source in the canonical store.
	srcFile := filepath.Join(agentsDir, "instructions", "AGENTS.md")
	if err := os.WriteFile(srcFile, []byte("new content from remote"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &Manifest{
		Agents: []Agent{},
		Links: []LinkEntry{
			{Source: "instructions/AGENTS.md", Target: targetFile, Agent: "claude", Mode: "copy"},
		},
	}
	if err := WriteManifest(m); err != nil {
		t.Fatal(err)
	}

	// Update the canonical store and push changes.
	if _, err := Commit("add link entry"); err != nil {
		t.Fatal(err)
	}
	if err := SyncPush(); err != nil {
		t.Fatalf("SyncPush() after manifest update: %v", err)
	}

	// Now simulate a pull: the manifest has been updated with copy-mode links.
	result, err := SyncPullWithResult()
	if err != nil {
		t.Fatalf("SyncPullWithResult() error: %v", err)
	}

	// Verify the target file was re-copied.
	if result.CopiedLinks != 1 {
		t.Errorf("CopiedLinks = %d, want 1", result.CopiedLinks)
	}

	data, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatalf("reading target file: %v", err)
	}
	if string(data) != "new content from remote" {
		t.Errorf("target file content = %q, want %q", string(data), "new content from remote")
	}
}

// TestSyncPullWithResult_SymlinkMode verifies that symlink-mode links are not
// actively processed (they're already up-to-date via the symlink pointer).
func TestSyncPullWithResult_SymlinkMode(t *testing.T) {
	agentsDir := setupEnv(t)
	tmpDir := t.TempDir()

	if err := Init(); err != nil {
		t.Fatal(err)
	}

	// Create initial content and commit.
	if err := os.WriteFile(filepath.Join(agentsDir, "test.txt"), []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Commit("initial"); err != nil {
		t.Fatal(err)
	}

	// Set up a bare repo as the "remote".
	bareRepo := filepath.Join(tmpDir, "remote.git")
	if _, err := gitCmd(tmpDir, "init", "--bare", bareRepo); err != nil {
		t.Fatalf("creating bare repo: %v", err)
	}

	if err := SyncSetRemote(bareRepo); err != nil {
		t.Fatal(err)
	}
	if err := SyncPush(); err != nil {
		t.Fatal(err)
	}

	// Add a symlink-mode link to the manifest.
	m := &Manifest{
		Agents: []Agent{},
		Links: []LinkEntry{
			{Source: "instructions/AGENTS.md", Target: "/tmp/some/path/CLAUDE.md", Agent: "claude", Mode: "symlink"},
		},
	}
	if err := WriteManifest(m); err != nil {
		t.Fatal(err)
	}

	if _, err := Commit("add symlink entry"); err != nil {
		t.Fatal(err)
	}
	if err := SyncPush(); err != nil {
		t.Fatal(err)
	}

	// Pull — symlink entries should not count as copied.
	result, err := SyncPullWithResult()
	if err != nil {
		t.Fatalf("SyncPullWithResult() error: %v", err)
	}

	if result.CopiedLinks != 0 {
		t.Errorf("CopiedLinks = %d, want 0 for symlink-mode links", result.CopiedLinks)
	}
}

// TestSyncIntegration_InitCommitRemotePush tests the full workflow:
// init → add content → commit → set remote → push.
func TestSyncIntegration_InitCommitRemotePush(t *testing.T) {
	agentsDir := setupEnv(t)
	tmpDir := t.TempDir()

	// Step 1: Init.
	if err := Init(); err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	// Step 2: Add content.
	if err := os.WriteFile(filepath.Join(agentsDir, "instructions", "AGENTS.md"), []byte("# My Instructions\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Step 3: Commit.
	hash, err := Commit("add instructions")
	if err != nil {
		t.Fatalf("Commit() error: %v", err)
	}
	if hash == "" {
		t.Error("Commit() returned empty hash")
	}

	// Step 4: Set remote (use a bare local repo as mock remote).
	bareRepo := filepath.Join(tmpDir, "agents.git")
	if _, err := gitCmd(tmpDir, "init", "--bare", bareRepo); err != nil {
		t.Fatalf("creating bare repo: %v", err)
	}

	if err := SyncSetRemote(bareRepo); err != nil {
		t.Fatalf("SyncSetRemote() error: %v", err)
	}

	// Verify remote was set.
	url := SyncRemoteURL()
	if url != bareRepo {
		t.Errorf("SyncRemoteURL() = %q, want %q", url, bareRepo)
	}

	// Step 5: Push.
	if err := SyncPush(); err != nil {
		t.Fatalf("SyncPush() error: %v", err)
	}
}
