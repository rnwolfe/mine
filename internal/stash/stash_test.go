package stash

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rnwolfe/mine/internal/gitutil"
)

// setupEnv creates a temp directory structure for stash tests and benchmarks.
// Accepts testing.TB so it works with both *testing.T and *testing.B.
func setupEnv(tb testing.TB) (stashDir string, homeDir string) {
	tb.Helper()

	tmpDir := tb.TempDir()
	homeDir = filepath.Join(tmpDir, "home")
	dataDir := filepath.Join(tmpDir, "data")

	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		tb.Fatal(err)
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		tb.Fatal(err)
	}

	// Point XDG dirs to our temp dirs.
	tb.Setenv("XDG_DATA_HOME", dataDir)
	tb.Setenv("HOME", homeDir)

	return Dir(), homeDir
}

// setupTestEnv is a backward-compatible alias for setupEnv for *testing.T callers.
func setupTestEnv(t *testing.T) (stashDir string, homeDir string) {
	t.Helper()
	return setupEnv(t)
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
	stashDir, homeDir := setupEnv(t)
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
	setupEnv(t)

	entries, err := ReadManifest()
	if err != nil {
		t.Fatalf("ReadManifest() error: %v", err)
	}
	if entries != nil {
		t.Errorf("ReadManifest() = %v, want nil", entries)
	}
}

func TestReadManifestSkipsComments(t *testing.T) {
	stashDir, homeDir := setupEnv(t)
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
	stashDir, _ := setupEnv(t)
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
	stashDir, homeDir := setupEnv(t)
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
	setupEnv(t)

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
	stashDir, homeDir := setupEnv(t)
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
	stashDir, homeDir := setupEnv(t)
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
	stashDir, homeDir := setupEnv(t)
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
	setupEnv(t)

	_, err := Log("")
	if err == nil {
		t.Error("Log() should error when no git repo exists")
	}
}

func TestRestore(t *testing.T) {
	stashDir, homeDir := setupEnv(t)
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
	stashDir, homeDir := setupEnv(t)
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
	if _, err := RestoreToSource(".zshrc", origHash, false); err != nil {
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
	stashDir, homeDir := setupEnv(t)
	source := createTestFile(t, homeDir, ".zshrc", "content")
	setupManifest(t, stashDir, source, ".zshrc", "content")

	_, err := Restore(".zshrc", "")
	if err == nil {
		t.Error("Restore() should error when no git repo exists")
	}
}

func TestRestoreInvalidVersion(t *testing.T) {
	stashDir, homeDir := setupEnv(t)
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
	setupEnv(t)

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
	stashDir, homeDir := setupEnv(t)
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
	stashDir, homeDir := setupEnv(t)
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
	setupEnv(t)

	url := SyncRemoteURL()
	if url != "" {
		t.Errorf("SyncRemoteURL() = %q, want empty", url)
	}
}

func TestCommitRefreshesSources(t *testing.T) {
	stashDir, homeDir := setupEnv(t)
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

func TestValidateEntry(t *testing.T) {
	_, homeDir := setupEnv(t)

	tests := []struct {
		name    string
		entry   Entry
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid entry",
			entry:   Entry{SafeName: ".zshrc", Source: filepath.Join(homeDir, ".zshrc")},
			wantErr: false,
		},
		{
			name:    "empty SafeName",
			entry:   Entry{SafeName: "", Source: filepath.Join(homeDir, ".zshrc")},
			wantErr: true,
			errMsg:  "empty SafeName",
		},
		{
			name:    "traversal in SafeName (..)",
			entry:   Entry{SafeName: "../evil", Source: filepath.Join(homeDir, ".zshrc")},
			wantErr: true,
			errMsg:  "unsafe SafeName",
		},
		{
			name:    "traversal in SafeName (slash)",
			entry:   Entry{SafeName: "sub/file", Source: filepath.Join(homeDir, ".zshrc")},
			wantErr: true,
			errMsg:  "unsafe SafeName",
		},
		{
			name:    "empty Source",
			entry:   Entry{SafeName: ".zshrc", Source: ""},
			wantErr: true,
			errMsg:  "empty Source",
		},
		{
			name:    "relative Source",
			entry:   Entry{SafeName: ".zshrc", Source: "relative/path/.zshrc"},
			wantErr: true,
			errMsg:  "not absolute",
		},
		{
			name:    "Source outside home dir",
			entry:   Entry{SafeName: "passwd", Source: "/etc/passwd"},
			wantErr: true,
			errMsg:  "escapes home directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEntry(tt.entry)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateEntry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("ValidateEntry() error = %q, want containing %q", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestRestoreToSourcePermissions(t *testing.T) {
	tests := []struct {
		name     string
		// setup is called after the initial commit; source is the source file path.
		setup    func(t *testing.T, source string)
		wantPerm os.FileMode
	}{
		{
			name: "source does not exist uses 0644 default",
			setup: func(t *testing.T, source string) {
				// Remove source to simulate a first-time restore to a machine that
				// doesn't yet have the file.
				if err := os.Remove(source); err != nil {
					t.Fatal(err)
				}
			},
			wantPerm: 0o644,
		},
		{
			name: "source exists with 0755 inherits 0755",
			setup: func(t *testing.T, source string) {
				if err := os.Chmod(source, 0o755); err != nil {
					t.Fatal(err)
				}
			},
			wantPerm: 0o755,
		},
		{
			name: "source exists with 0600 inherits 0600",
			setup: func(t *testing.T, source string) {
				if err := os.Chmod(source, 0o600); err != nil {
					t.Fatal(err)
				}
			},
			wantPerm: 0o600,
		},
		{
			name: "source is read-only 0444 restore succeeds",
			setup: func(t *testing.T, source string) {
				if err := os.Chmod(source, 0o444); err != nil {
					t.Fatal(err)
				}
			},
			wantPerm: 0o444,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stashDir, homeDir := setupTestEnv(t)
			source := createTestFile(t, homeDir, ".zshrc", "content v1")
			setupManifest(t, stashDir, source, ".zshrc", "content v1")

			if _, err := Commit("initial"); err != nil {
				t.Fatalf("Commit() error: %v", err)
			}

			tt.setup(t, source)

			entry, err := RestoreToSource(".zshrc", "", false)
			if err != nil {
				t.Fatalf("RestoreToSource() error: %v", err)
			}
			if entry == nil {
				t.Fatal("RestoreToSource() returned nil entry")
			}

			info, err := os.Stat(source)
			if err != nil {
				t.Fatalf("os.Stat(source) error: %v", err)
			}
			if got := info.Mode().Perm(); got != tt.wantPerm {
				t.Errorf("source file mode = %04o, want %04o", got, tt.wantPerm)
			}

			data, err := os.ReadFile(source)
			if err != nil {
				t.Fatalf("os.ReadFile(source) error: %v", err)
			}
			if gotContent := string(data); gotContent != "content v1" {
				t.Errorf("source file content = %q, want %q", gotContent, "content v1")
			}
		})
	}
}

func TestRestoreToSourceForce(t *testing.T) {
	tests := []struct {
		name     string
		// srcPerm is the permission set on the source file before restore.
		srcPerm  os.FileMode
		// stashPerm is the permission to set on the stash copy (simulates commit-time mode).
		stashPerm os.FileMode
		force    bool
		wantPerm os.FileMode
	}{
		{
			name:      "force=false preserves source permissions",
			srcPerm:   0o444,
			stashPerm: 0o644,
			force:     false,
			wantPerm:  0o444,
		},
		{
			name:      "force=true uses stash-recorded permissions",
			srcPerm:   0o444,
			stashPerm: 0o644,
			force:     true,
			wantPerm:  0o644,
		},
		{
			name:      "force=true with executable stash mode",
			srcPerm:   0o600,
			stashPerm: 0o755,
			force:     true,
			wantPerm:  0o755,
		},
		{
			name:      "force=false with executable source mode",
			srcPerm:   0o755,
			stashPerm: 0o644,
			force:     false,
			wantPerm:  0o755,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stashDir, homeDir := setupTestEnv(t)
			source := createTestFile(t, homeDir, ".zshrc", "content v1")
			setupManifest(t, stashDir, source, ".zshrc", "content v1")

			if _, err := Commit("initial"); err != nil {
				t.Fatalf("Commit() error: %v", err)
			}

			// Set source file to the test permission.
			if err := os.Chmod(source, tt.srcPerm); err != nil {
				t.Fatal(err)
			}
			// Override stash copy permission to simulate the commit-time mode.
			stashCopy := filepath.Join(stashDir, ".zshrc")
			if err := os.Chmod(stashCopy, tt.stashPerm); err != nil {
				t.Fatal(err)
			}

			entry, err := RestoreToSource(".zshrc", "", tt.force)
			if err != nil {
				t.Fatalf("RestoreToSource(force=%v) error: %v", tt.force, err)
			}
			if entry == nil {
				t.Fatal("RestoreToSource() returned nil entry")
			}

			info, err := os.Stat(source)
			if err != nil {
				t.Fatalf("os.Stat(source) error: %v", err)
			}
			if got := info.Mode().Perm(); got != tt.wantPerm {
				t.Errorf("source file mode = %04o, want %04o (force=%v)", got, tt.wantPerm, tt.force)
			}

			data, err := os.ReadFile(source)
			if err != nil {
				t.Fatalf("os.ReadFile(source) error: %v", err)
			}
			if string(data) != "content v1" {
				t.Errorf("source content = %q, want %q", string(data), "content v1")
			}
		})
	}
}

func TestRestoreToSourceForce_NoSourceFile(t *testing.T) {
	stashDir, homeDir := setupTestEnv(t)

	source := createTestFile(t, homeDir, ".zshrc", "content v1")
	setupManifest(t, stashDir, source, ".zshrc", "content v1")

	if _, err := Commit("initial"); err != nil {
		t.Fatalf("Commit() error: %v", err)
	}

	// Remove source file to simulate first-time restore on a new machine.
	if err := os.Remove(source); err != nil {
		t.Fatalf("os.Remove(source) error: %v", err)
	}

	// Set a specific permission on the stash copy to verify it's used.
	stashCopy := filepath.Join(stashDir, ".zshrc")
	if err := os.Chmod(stashCopy, 0o755); err != nil {
		t.Fatalf("os.Chmod(stashCopy) error: %v", err)
	}

	entry, err := RestoreToSource(".zshrc", "", true)
	if err != nil {
		t.Fatalf("RestoreToSource(force=true, no source) error: %v", err)
	}
	if entry == nil {
		t.Fatal("RestoreToSource() returned nil entry")
	}

	info, err := os.Stat(source)
	if err != nil {
		t.Fatalf("os.Stat(source) error: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o755 {
		t.Errorf("source file mode = %04o, want %04o (force=true, no prior source)", got, 0o755)
	}

	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("os.ReadFile(source) error: %v", err)
	}
	if string(data) != "content v1" {
		t.Errorf("source content = %q, want %q", string(data), "content v1")
	}
}

func TestRunCmd(t *testing.T) {
	dir := t.TempDir()

	// Valid command.
	out, err := gitutil.RunCmd(dir, "init")
	if err != nil {
		t.Fatalf("gitutil.RunCmd(init) error: %v", err)
	}
	if out == "" {
		t.Error("gitutil.RunCmd(init) returned empty output")
	}

	// Invalid command.
	_, err = gitutil.RunCmd(dir, "fakecmd")
	if err == nil {
		t.Error("gitutil.RunCmd(fakecmd) should error")
	}
}
