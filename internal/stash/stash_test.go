package stash

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestEnv creates a temp directory structure for stash tests.
// Returns a cleanup function (called by t.Cleanup automatically).
func setupTestEnv(t *testing.T) (stashDir string, homeDir string) {
	t.Helper()

	tmpDir := t.TempDir()
	homeDir = filepath.Join(tmpDir, "home")
	dataDir := filepath.Join(tmpDir, "data")

	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Point XDG dirs to our temp dirs.
	t.Setenv("XDG_DATA_HOME", dataDir)
	t.Setenv("HOME", homeDir)

	return Dir(), homeDir
}

// createTestFile creates a file in the test home directory.
func createTestFile(t *testing.T, homeDir, name, content string) string {
	t.Helper()
	path := filepath.Join(homeDir, name)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// setupManifest creates a stash directory with a manifest and tracked file.
func setupManifest(t *testing.T, stashDir, source, safeName, content string) {
	t.Helper()
	if err := os.MkdirAll(stashDir, 0o755); err != nil {
		t.Fatal(err)
	}

	manifest := filepath.Join(stashDir, ".mine-stash")
	entry := source + " -> " + safeName + "\n"
	if err := os.WriteFile(manifest, []byte("# mine stash manifest\n"+entry), 0o644); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(stashDir, safeName)
	if err := os.WriteFile(dest, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestReadManifest(t *testing.T) {
	stashDir, homeDir := setupTestEnv(t)
	source := createTestFile(t, homeDir, ".zshrc", "export PATH=$PATH")
	safeName := ".zshrc"
	setupManifest(t, stashDir, source, safeName, "export PATH=$PATH")

	entries, err := ReadManifest()
	if err != nil {
		t.Fatalf("ReadManifest() error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("ReadManifest() returned %d entries, want 1", len(entries))
	}
	if entries[0].Source != source {
		t.Errorf("Source = %q, want %q", entries[0].Source, source)
	}
	if entries[0].SafeName != safeName {
		t.Errorf("SafeName = %q, want %q", entries[0].SafeName, safeName)
	}
}

func TestReadManifestNoFile(t *testing.T) {
	setupTestEnv(t)

	entries, err := ReadManifest()
	if err != nil {
		t.Fatalf("ReadManifest() error: %v", err)
	}
	if entries != nil {
		t.Errorf("ReadManifest() = %v, want nil", entries)
	}
}

func TestReadManifestSkipsComments(t *testing.T) {
	stashDir, homeDir := setupTestEnv(t)
	if err := os.MkdirAll(stashDir, 0o755); err != nil {
		t.Fatal(err)
	}

	source := createTestFile(t, homeDir, ".bashrc", "echo hi")
	content := "# mine stash manifest\n# comment\n\n" + source + " -> .bashrc\n"
	if err := os.WriteFile(ManifestPath(), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stashDir, ".bashrc"), []byte("echo hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := ReadManifest()
	if err != nil {
		t.Fatalf("ReadManifest() error: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
}

func TestReadManifestEmptyEntries(t *testing.T) {
	stashDir, _ := setupTestEnv(t)
	if err := os.MkdirAll(stashDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a manifest with only comments and blank lines — no tracked entries.
	content := "# mine stash manifest\n# each line: source_path -> safe_name\n\n"
	if err := os.WriteFile(ManifestPath(), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := ReadManifest()
	if err != nil {
		t.Fatalf("ReadManifest() error: %v", err)
	}
	if entries == nil {
		t.Error("ReadManifest() returned nil, want non-nil empty slice for existing manifest")
	}
	if len(entries) != 0 {
		t.Errorf("ReadManifest() returned %d entries, want 0", len(entries))
	}
}

func TestFindEntry(t *testing.T) {
	stashDir, homeDir := setupTestEnv(t)
	source := createTestFile(t, homeDir, ".zshrc", "content")
	setupManifest(t, stashDir, source, ".zshrc", "content")

	// Find by safe name.
	entry, err := FindEntry(".zshrc")
	if err != nil {
		t.Fatalf("FindEntry by safeName: %v", err)
	}
	if entry.Source != source {
		t.Errorf("Source = %q, want %q", entry.Source, source)
	}

	// Find by absolute path.
	entry, err = FindEntry(source)
	if err != nil {
		t.Fatalf("FindEntry by source path: %v", err)
	}
	if entry.SafeName != ".zshrc" {
		t.Errorf("SafeName = %q, want %q", entry.SafeName, ".zshrc")
	}

	// Not found.
	_, err = FindEntry("nonexistent")
	if err == nil {
		t.Error("FindEntry should return error for nonexistent file")
	}
}

func TestInitGitRepo(t *testing.T) {
	setupTestEnv(t)

	if IsGitRepo() {
		t.Error("IsGitRepo() should be false before init")
	}

	if err := InitGitRepo(); err != nil {
		t.Fatalf("InitGitRepo() error: %v", err)
	}

	if !IsGitRepo() {
		t.Error("IsGitRepo() should be true after init")
	}

	// Second init should be idempotent.
	if err := InitGitRepo(); err != nil {
		t.Fatalf("second InitGitRepo() error: %v", err)
	}
}

func TestCommitAndLog(t *testing.T) {
	stashDir, homeDir := setupTestEnv(t)
	source := createTestFile(t, homeDir, ".zshrc", "export FOO=bar")
	setupManifest(t, stashDir, source, ".zshrc", "export FOO=bar")

	// First commit.
	hash, err := Commit("initial snapshot")
	if err != nil {
		t.Fatalf("Commit() error: %v", err)
	}
	if hash == "" {
		t.Error("Commit() returned empty hash")
	}

	// Verify log shows the commit.
	logs, err := Log("")
	if err != nil {
		t.Fatalf("Log() error: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("Log() returned %d entries, want 1", len(logs))
	}
	if logs[0].Message != "initial snapshot" {
		t.Errorf("Message = %q, want %q", logs[0].Message, "initial snapshot")
	}
	if logs[0].Short != hash {
		t.Errorf("Short = %q, want %q", logs[0].Short, hash)
	}

	// Modify source and commit again.
	if err := os.WriteFile(source, []byte("export FOO=baz"), 0o644); err != nil {
		t.Fatal(err)
	}
	hash2, err := Commit("updated config")
	if err != nil {
		t.Fatalf("second Commit() error: %v", err)
	}
	if hash2 == hash {
		t.Error("second commit should have different hash")
	}

	// Log should show 2 entries.
	logs, err = Log("")
	if err != nil {
		t.Fatalf("Log() error: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("Log() returned %d entries, want 2", len(logs))
	}
	// Most recent first.
	if logs[0].Message != "updated config" {
		t.Errorf("first log message = %q, want %q", logs[0].Message, "updated config")
	}
}

func TestCommitNothingToCommit(t *testing.T) {
	stashDir, homeDir := setupTestEnv(t)
	source := createTestFile(t, homeDir, ".zshrc", "content")
	setupManifest(t, stashDir, source, ".zshrc", "content")

	// First commit succeeds.
	if _, err := Commit("first"); err != nil {
		t.Fatalf("first Commit() error: %v", err)
	}

	// Second commit with no changes should fail.
	_, err := Commit("same thing")
	if err == nil {
		t.Error("expected error for nothing-to-commit")
	}
	if !strings.Contains(err.Error(), "nothing to commit") {
		t.Errorf("error = %q, want containing 'nothing to commit'", err.Error())
	}
}

func TestLogFilteredByFile(t *testing.T) {
	stashDir, homeDir := setupTestEnv(t)
	source1 := createTestFile(t, homeDir, ".zshrc", "zsh content")
	source2 := createTestFile(t, homeDir, ".bashrc", "bash content")

	// Set up manifest with two files.
	if err := os.MkdirAll(stashDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := source1 + " -> .zshrc\n" + source2 + " -> .bashrc\n"
	if err := os.WriteFile(ManifestPath(), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stashDir, ".zshrc"), []byte("zsh content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stashDir, ".bashrc"), []byte("bash content"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Commit both.
	if _, err := Commit("both files"); err != nil {
		t.Fatalf("Commit() error: %v", err)
	}

	// Modify only zshrc and commit again.
	if err := os.WriteFile(source1, []byte("zsh v2"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Commit("zsh update"); err != nil {
		t.Fatalf("second Commit() error: %v", err)
	}

	// Log for .zshrc should show 2 commits.
	zshLogs, err := Log(".zshrc")
	if err != nil {
		t.Fatalf("Log(.zshrc) error: %v", err)
	}
	if len(zshLogs) != 2 {
		t.Fatalf("Log(.zshrc) returned %d entries, want 2", len(zshLogs))
	}

	// Log for .bashrc should show 1 commit (unchanged in second).
	bashLogs, err := Log(".bashrc")
	if err != nil {
		t.Fatalf("Log(.bashrc) error: %v", err)
	}
	if len(bashLogs) != 1 {
		t.Fatalf("Log(.bashrc) returned %d entries, want 1", len(bashLogs))
	}
}

func TestLogNoRepo(t *testing.T) {
	setupTestEnv(t)

	_, err := Log("")
	if err == nil {
		t.Error("Log() should error when no git repo exists")
	}
}

func TestRestore(t *testing.T) {
	stashDir, homeDir := setupTestEnv(t)
	source := createTestFile(t, homeDir, ".zshrc", "version 1")
	setupManifest(t, stashDir, source, ".zshrc", "version 1")

	// Commit v1.
	if _, err := Commit("v1"); err != nil {
		t.Fatalf("Commit v1: %v", err)
	}

	// Get v1 hash.
	logs, _ := Log("")
	v1Hash := logs[0].Short

	// Modify and commit v2.
	if err := os.WriteFile(source, []byte("version 2"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Commit("v2"); err != nil {
		t.Fatalf("Commit v2: %v", err)
	}

	// Restore to v1.
	content, err := Restore(".zshrc", v1Hash)
	if err != nil {
		t.Fatalf("Restore() error: %v", err)
	}
	if string(content) != "version 1" {
		t.Errorf("restored content = %q, want %q", string(content), "version 1")
	}

	// Restore latest (HEAD).
	content, err = Restore(".zshrc", "")
	if err != nil {
		t.Fatalf("Restore(HEAD) error: %v", err)
	}
	if string(content) != "version 2" {
		t.Errorf("HEAD content = %q, want %q", string(content), "version 2")
	}
}

func TestRestoreToSource(t *testing.T) {
	stashDir, homeDir := setupTestEnv(t)
	source := createTestFile(t, homeDir, ".zshrc", "original")
	setupManifest(t, stashDir, source, ".zshrc", "original")

	// Commit original.
	if _, err := Commit("original"); err != nil {
		t.Fatal(err)
	}
	logs, _ := Log("")
	origHash := logs[0].Short

	// Modify source and commit.
	if err := os.WriteFile(source, []byte("modified"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Commit("modified"); err != nil {
		t.Fatal(err)
	}

	// Restore original version to source.
	if err := RestoreToSource(".zshrc", origHash); err != nil {
		t.Fatalf("RestoreToSource() error: %v", err)
	}

	// Verify source file was restored.
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "original" {
		t.Errorf("source content = %q, want %q", string(data), "original")
	}
}

func TestRestoreNoRepo(t *testing.T) {
	stashDir, homeDir := setupTestEnv(t)
	source := createTestFile(t, homeDir, ".zshrc", "content")
	setupManifest(t, stashDir, source, ".zshrc", "content")

	_, err := Restore(".zshrc", "")
	if err == nil {
		t.Error("Restore() should error when no git repo exists")
	}
}

func TestRestoreInvalidVersion(t *testing.T) {
	stashDir, homeDir := setupTestEnv(t)
	source := createTestFile(t, homeDir, ".zshrc", "content")
	setupManifest(t, stashDir, source, ".zshrc", "content")

	if _, err := Commit("initial"); err != nil {
		t.Fatal(err)
	}

	_, err := Restore(".zshrc", "deadbeef")
	if err == nil {
		t.Error("Restore() should error for invalid version")
	}
}

func TestSyncSetRemote(t *testing.T) {
	setupTestEnv(t)

	// Should fail without git repo.
	err := SyncSetRemote("https://example.com/dotfiles.git")
	if err == nil {
		t.Error("SyncSetRemote() should error without git repo")
	}

	// Init repo.
	if err := InitGitRepo(); err != nil {
		t.Fatal(err)
	}

	// Set remote.
	if err := SyncSetRemote("https://example.com/dotfiles.git"); err != nil {
		t.Fatalf("SyncSetRemote() error: %v", err)
	}

	url := SyncRemoteURL()
	if url != "https://example.com/dotfiles.git" {
		t.Errorf("SyncRemoteURL() = %q, want %q", url, "https://example.com/dotfiles.git")
	}

	// Update remote.
	if err := SyncSetRemote("https://example.com/new-dotfiles.git"); err != nil {
		t.Fatalf("SyncSetRemote(update) error: %v", err)
	}

	url = SyncRemoteURL()
	if url != "https://example.com/new-dotfiles.git" {
		t.Errorf("SyncRemoteURL() = %q, want %q", url, "https://example.com/new-dotfiles.git")
	}
}

func TestSyncPushNoRemote(t *testing.T) {
	stashDir, homeDir := setupTestEnv(t)
	source := createTestFile(t, homeDir, ".zshrc", "content")
	setupManifest(t, stashDir, source, ".zshrc", "content")

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

func TestSyncPullNoRemote(t *testing.T) {
	stashDir, homeDir := setupTestEnv(t)
	source := createTestFile(t, homeDir, ".zshrc", "content")
	setupManifest(t, stashDir, source, ".zshrc", "content")

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

func TestSyncRemoteURLEmpty(t *testing.T) {
	setupTestEnv(t)

	url := SyncRemoteURL()
	if url != "" {
		t.Errorf("SyncRemoteURL() = %q, want empty", url)
	}
}

func TestCommitRefreshesSources(t *testing.T) {
	stashDir, homeDir := setupTestEnv(t)
	source := createTestFile(t, homeDir, ".zshrc", "original")
	setupManifest(t, stashDir, source, ".zshrc", "old stash content")

	// The source has "original" but the stash has "old stash content".
	// Commit should refresh the stash copy from source before committing.
	hash, err := Commit("refresh test")
	if err != nil {
		t.Fatalf("Commit() error: %v", err)
	}
	if hash == "" {
		t.Error("Commit() returned empty hash")
	}

	// Verify the stash file was updated.
	stashContent, err := os.ReadFile(filepath.Join(stashDir, ".zshrc"))
	if err != nil {
		t.Fatal(err)
	}
	if string(stashContent) != "original" {
		t.Errorf("stash content = %q, want %q", string(stashContent), "original")
	}
}

func TestGitCmd(t *testing.T) {
	dir := t.TempDir()

	// Valid command.
	out, err := gitCmd(dir, "init")
	if err != nil {
		t.Fatalf("gitCmd(init) error: %v", err)
	}
	if out == "" {
		t.Error("gitCmd(init) returned empty output")
	}

	// Invalid command.
	_, err = gitCmd(dir, "fakecmd")
	if err == nil {
		t.Error("gitCmd(fakecmd) should error")
	}
}
