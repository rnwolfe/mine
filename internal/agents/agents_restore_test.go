package agents

import (
	"os"
	"path/filepath"
	"testing"
)

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
	_, _, err := RestoreToStore("instructions/AGENTS.md", "")
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
	updated, failed, err := RestoreToStore("instructions/AGENTS.md", "")
	if err != nil {
		t.Fatalf("RestoreToStore() error: %v", err)
	}
	if len(updated) != 1 {
		t.Errorf("RestoreToStore() updated %d links, want 1", len(updated))
	}
	if len(failed) != 0 {
		t.Errorf("RestoreToStore() failed %d links, want 0", len(failed))
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

func TestRestoreToStore_FailedLinks(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
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

	// Set up a copy-mode link pointing to a read-only directory.
	roDir := filepath.Join(tmpDir, "readonly")
	if err := os.MkdirAll(roDir, 0o555); err != nil {
		t.Fatal(err)
	}
	badTarget := filepath.Join(roDir, "subdir", "AGENTS.md")

	m := &Manifest{
		Agents: []Agent{},
		Links: []LinkEntry{
			{
				Source: "instructions/AGENTS.md",
				Target: badTarget,
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

	// Restore should succeed for the store file but report the failed link.
	updated, failed, err := RestoreToStore("instructions/AGENTS.md", "")
	if err != nil {
		t.Fatalf("RestoreToStore() unexpected hard error: %v", err)
	}
	if len(updated) != 0 {
		t.Errorf("RestoreToStore() updated %d links, want 0", len(updated))
	}
	if len(failed) != 1 {
		t.Errorf("RestoreToStore() failed %d links, want 1", len(failed))
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
	_, _, err = RestoreToStore("instructions/AGENTS.md", hash1)
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
