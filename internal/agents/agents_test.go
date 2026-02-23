package agents

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// setupEnv creates a temp directory structure for agents tests.
func setupEnv(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("XDG_DATA_HOME", dataDir)
	t.Setenv("HOME", filepath.Join(tmpDir, "home"))

	return Dir()
}

func TestDir(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)
	got := Dir()
	want := filepath.Join(dataDir, "mine", "agents")
	if got != want {
		t.Errorf("Dir() = %q, want %q", got, want)
	}
}

func TestManifestPath(t *testing.T) {
	dataDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataDir)
	got := ManifestPath()
	want := filepath.Join(dataDir, "mine", "agents", ".mine-agents")
	if got != want {
		t.Errorf("ManifestPath() = %q, want %q", got, want)
	}
}

func TestIsInitialized_False(t *testing.T) {
	setupEnv(t)
	if IsInitialized() {
		t.Error("IsInitialized() = true before init, want false")
	}
}

func TestIsInitialized_True(t *testing.T) {
	agentsDir := setupEnv(t)
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if !IsInitialized() {
		t.Error("IsInitialized() = false after creating directory, want true")
	}
}

func TestInit_CreatesStructure(t *testing.T) {
	agentsDir := setupEnv(t)

	if err := Init(); err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	// Check subdirectories exist.
	subdirs := []string{"instructions", "skills", "commands", "agents", "settings", "mcp", "rules"}
	for _, subdir := range subdirs {
		if _, err := os.Stat(filepath.Join(agentsDir, subdir)); err != nil {
			t.Errorf("subdirectory %q not created: %v", subdir, err)
		}
	}

	// Check manifest was created.
	if _, err := os.Stat(ManifestPath()); err != nil {
		t.Errorf("manifest file not created: %v", err)
	}

	// Check starter AGENTS.md was created.
	agentsMD := filepath.Join(agentsDir, "instructions", "AGENTS.md")
	if _, err := os.Stat(agentsMD); err != nil {
		t.Errorf("AGENTS.md not created: %v", err)
	}

	// Check git repo was initialized.
	if !IsGitRepo() {
		t.Error("git repo not initialized after Init()")
	}
}

func TestInit_Idempotent(t *testing.T) {
	setupEnv(t)

	if err := Init(); err != nil {
		t.Fatalf("first Init() error: %v", err)
	}
	if err := Init(); err != nil {
		t.Fatalf("second Init() error: %v", err)
	}
}

func TestReadWriteManifest(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	want := &Manifest{
		Agents: []Agent{
			{Name: "claude", Detected: true, ConfigDir: "~/.claude/", Binary: "claude"},
		},
		Links: []LinkEntry{
			{Source: "instructions/AGENTS.md", Target: "~/.claude/CLAUDE.md", Agent: "claude", Mode: "symlink"},
		},
	}

	if err := WriteManifest(want); err != nil {
		t.Fatalf("WriteManifest() error: %v", err)
	}

	got, err := ReadManifest()
	if err != nil {
		t.Fatalf("ReadManifest() error: %v", err)
	}
	if got == nil {
		t.Fatal("ReadManifest() returned nil")
	}
	if len(got.Agents) != 1 {
		t.Errorf("Agents count = %d, want 1", len(got.Agents))
	}
	if got.Agents[0].Name != "claude" {
		t.Errorf("Agents[0].Name = %q, want %q", got.Agents[0].Name, "claude")
	}
	if len(got.Links) != 1 {
		t.Errorf("Links count = %d, want 1", len(got.Links))
	}
	if got.Links[0].Mode != "symlink" {
		t.Errorf("Links[0].Mode = %q, want %q", got.Links[0].Mode, "symlink")
	}
}

func TestReadManifest_NoFile(t *testing.T) {
	setupEnv(t)

	m, err := ReadManifest()
	if err != nil {
		t.Fatalf("ReadManifest() error: %v", err)
	}
	if m != nil {
		t.Errorf("ReadManifest() = %v, want nil when no manifest exists", m)
	}
}

func TestWriteManifest_ValidJSON(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	m := &Manifest{Agents: []Agent{}, Links: []LinkEntry{}}
	if err := WriteManifest(m); err != nil {
		t.Fatalf("WriteManifest() error: %v", err)
	}

	data, err := os.ReadFile(ManifestPath())
	if err != nil {
		t.Fatal(err)
	}
	var check Manifest
	if err := json.Unmarshal(data, &check); err != nil {
		t.Errorf("manifest is not valid JSON: %v", err)
	}
}

func TestValidateRelativePath(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		wantErr bool
	}{
		{"valid relative path", "instructions/AGENTS.md", false},
		{"valid simple name", "AGENTS.md", false},
		{"valid nested path", "settings/claude/config.json", false},
		{"empty string", "", true},
		{"absolute path", "/etc/passwd", true},
		{"parent traversal", "../secret", true},
		{"embedded traversal", "a/../../b", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRelativePath(tc.file)
			if (err != nil) != tc.wantErr {
				t.Errorf("validateRelativePath(%q) error = %v, wantErr %v", tc.file, err, tc.wantErr)
			}
		})
	}
}

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

	// Second commit with no changes should error.
	_, err := Commit("empty")
	if err == nil {
		t.Error("Commit() expected error when nothing to commit, got nil")
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

	_, err := Log("")
	if err == nil {
		t.Error("Log() expected error when no git repo commits, got nil")
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

func TestRestore_NoGitRepo(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	if err := os.RemoveAll(filepath.Join(Dir(), ".git")); err != nil {
		t.Fatal(err)
	}

	_, err := Restore("instructions/AGENTS.md", "")
	if err == nil {
		t.Error("Restore() expected error when no git repo, got nil")
	}
}

func TestRestore_InvalidPath(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}
	if _, err := Commit("initial"); err != nil {
		t.Fatal(err)
	}

	_, err := Restore("../../../etc/passwd", "")
	if err == nil {
		t.Error("Restore() expected error for traversal path, got nil")
	}
}

func TestRestore_Success(t *testing.T) {
	agentsDir := setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	targetFile := filepath.Join(agentsDir, "instructions", "AGENTS.md")
	originalContent := "original content"
	if err := os.WriteFile(targetFile, []byte(originalContent), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Commit("initial"); err != nil {
		t.Fatalf("Commit() error: %v", err)
	}

	// Modify the file.
	if err := os.WriteFile(targetFile, []byte("modified content"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Restore from HEAD.
	content, err := Restore("instructions/AGENTS.md", "")
	if err != nil {
		t.Fatalf("Restore() error: %v", err)
	}
	if string(content) != originalContent {
		t.Errorf("Restore() content = %q, want %q", string(content), originalContent)
	}
}

func TestRestoreToStore_Success(t *testing.T) {
	agentsDir := setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	targetFile := filepath.Join(agentsDir, "instructions", "AGENTS.md")
	originalContent := "original content\n"
	if err := os.WriteFile(targetFile, []byte(originalContent), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Commit("initial"); err != nil {
		t.Fatalf("Commit() error: %v", err)
	}

	// Modify the file.
	if err := os.WriteFile(targetFile, []byte("modified content\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Restore to store.
	_, err := RestoreToStore("instructions/AGENTS.md", "")
	if err != nil {
		t.Fatalf("RestoreToStore() error: %v", err)
	}

	got, err := os.ReadFile(targetFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != originalContent {
		t.Errorf("RestoreToStore() file content = %q, want %q", string(got), originalContent)
	}
}

func TestRestoreToStore_CopyModeLinks(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	targetDir := filepath.Join(tmpDir, "agent-config")

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("XDG_DATA_HOME", dataDir)
	t.Setenv("HOME", filepath.Join(tmpDir, "home"))

	agentsDir := Dir()
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	// Write initial file content and commit.
	targetFile := filepath.Join(agentsDir, "instructions", "AGENTS.md")
	originalContent := "original content\n"
	if err := os.WriteFile(targetFile, []byte(originalContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Set up a copy-mode link.
	copyTarget := filepath.Join(targetDir, "AGENTS.md")
	m := &Manifest{
		Agents: []Agent{},
		Links: []LinkEntry{
			{
				Source: "instructions/AGENTS.md",
				Target: copyTarget,
				Agent:  "testAgent",
				Mode:   "copy",
			},
		},
	}
	if err := WriteManifest(m); err != nil {
		t.Fatal(err)
	}

	if _, err := Commit("initial"); err != nil {
		t.Fatalf("Commit() error: %v", err)
	}

	// Modify the canonical file.
	if err := os.WriteFile(targetFile, []byte("modified content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Also update the copy target to simulate diverged state.
	if err := os.WriteFile(copyTarget, []byte("diverged content\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Restore should update canonical store AND re-copy to the copy-mode target.
	updated, err := RestoreToStore("instructions/AGENTS.md", "")
	if err != nil {
		t.Fatalf("RestoreToStore() error: %v", err)
	}
	if len(updated) != 1 {
		t.Errorf("RestoreToStore() updated %d links, want 1", len(updated))
	}

	// Check copy target was updated.
	got, err := os.ReadFile(copyTarget)
	if err != nil {
		t.Fatalf("reading copy target: %v", err)
	}
	if string(got) != originalContent {
		t.Errorf("copy target content = %q, want %q", string(got), originalContent)
	}
}

// TestIntegration_InitCommitModifyRestoreVerify is the end-to-end integration test.
func TestIntegration_InitCommitModifyRestoreVerify(t *testing.T) {
	agentsDir := setupEnv(t)

	// Step 1: Init.
	if err := Init(); err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	// Step 2: Write a file and commit.
	testFile := filepath.Join(agentsDir, "instructions", "AGENTS.md")
	v1 := "version 1 content\n"
	if err := os.WriteFile(testFile, []byte(v1), 0o644); err != nil {
		t.Fatal(err)
	}
	hash1, err := Commit("v1: initial content")
	if err != nil {
		t.Fatalf("first Commit() error: %v", err)
	}

	// Step 3: Modify and commit again.
	v2 := "version 2 content\n"
	if err := os.WriteFile(testFile, []byte(v2), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Commit("v2: updated content"); err != nil {
		t.Fatalf("second Commit() error: %v", err)
	}

	// Step 4: Log should show 2 entries.
	entries, err := Log("")
	if err != nil {
		t.Fatalf("Log() error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("Log() returned %d entries, want 2", len(entries))
	}

	// Step 5: Restore to v1 using hash.
	_, err = RestoreToStore("instructions/AGENTS.md", hash1)
	if err != nil {
		t.Fatalf("RestoreToStore() error: %v", err)
	}

	// Step 6: Verify file content is v1.
	got, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != v1 {
		t.Errorf("restored content = %q, want %q", string(got), v1)
	}
}
