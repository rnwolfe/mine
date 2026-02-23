package agents

import (
	"os"
	"path/filepath"
	"testing"
)

// setupLinkHealthEnv creates a temp environment and initializes the store,
// returning (storeDir, home).
func setupLinkHealthEnv(t *testing.T) (storeDir, home string) {
	t.Helper()
	tmp := t.TempDir()
	home = tmp
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, "data"))

	storeDir = Dir()
	if err := Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	return storeDir, home
}

// writeStoreFileHealth writes content to a file in the store.
func writeStoreFileHealth(t *testing.T, storeDir, rel, content string) string {
	t.Helper()
	p := filepath.Join(storeDir, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return p
}

// --- CheckLinkHealth unit tests ---

func TestCheckLinkHealth_Linked_Symlink(t *testing.T) {
	storeDir, _ := setupLinkHealthEnv(t)

	// Create canonical source file.
	sourcePath := writeStoreFileHealth(t, storeDir, "instructions/AGENTS.md", "hello")

	// Create a symlink pointing to the canonical source.
	target := filepath.Join(t.TempDir(), "CLAUDE.md")
	if err := os.Symlink(sourcePath, target); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	entry := LinkEntry{
		Source: "instructions/AGENTS.md",
		Target: target,
		Agent:  "claude",
		Mode:   "symlink",
	}

	h := CheckLinkHealth(entry, storeDir)
	if h.State != LinkHealthLinked {
		t.Errorf("State = %q, want %q", h.State, LinkHealthLinked)
	}
}

func TestCheckLinkHealth_Broken_DanglingSymlink(t *testing.T) {
	storeDir, _ := setupLinkHealthEnv(t)

	targetDir := t.TempDir()
	target := filepath.Join(targetDir, "CLAUDE.md")

	// Create a symlink pointing to a non-existent path.
	if err := os.Symlink("/nonexistent/canonical/path/AGENTS.md", target); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	entry := LinkEntry{
		Source: "instructions/AGENTS.md",
		Target: target,
		Agent:  "claude",
		Mode:   "symlink",
	}

	h := CheckLinkHealth(entry, storeDir)
	if h.State != LinkHealthBroken {
		t.Errorf("State = %q, want %q", h.State, LinkHealthBroken)
	}
}

func TestCheckLinkHealth_Replaced_RegularFile(t *testing.T) {
	storeDir, _ := setupLinkHealthEnv(t)

	// Target exists as a regular file (not a symlink).
	target := filepath.Join(t.TempDir(), "CLAUDE.md")
	if err := os.WriteFile(target, []byte("standalone content"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	entry := LinkEntry{
		Source: "instructions/AGENTS.md",
		Target: target,
		Agent:  "claude",
		Mode:   "symlink",
	}

	h := CheckLinkHealth(entry, storeDir)
	if h.State != LinkHealthReplaced {
		t.Errorf("State = %q, want %q", h.State, LinkHealthReplaced)
	}
}

func TestCheckLinkHealth_Unlinked_Missing(t *testing.T) {
	storeDir, _ := setupLinkHealthEnv(t)

	entry := LinkEntry{
		Source: "instructions/AGENTS.md",
		Target: filepath.Join(t.TempDir(), "nonexistent.md"),
		Agent:  "claude",
		Mode:   "symlink",
	}

	h := CheckLinkHealth(entry, storeDir)
	if h.State != LinkHealthUnlinked {
		t.Errorf("State = %q, want %q", h.State, LinkHealthUnlinked)
	}
}

func TestCheckLinkHealth_Linked_CopyMode_Matches(t *testing.T) {
	storeDir, _ := setupLinkHealthEnv(t)

	content := "shared instructions content"
	sourcePath := writeStoreFileHealth(t, storeDir, "instructions/AGENTS.md", content)

	// Create a copy of the file at target.
	target := filepath.Join(t.TempDir(), "CLAUDE.md")
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	entry := LinkEntry{
		Source: "instructions/AGENTS.md",
		Target: target,
		Agent:  "claude",
		Mode:   "copy",
	}

	_ = sourcePath // used via storeDir resolution
	h := CheckLinkHealth(entry, storeDir)
	if h.State != LinkHealthLinked {
		t.Errorf("State = %q, want %q", h.State, LinkHealthLinked)
	}
}

func TestCheckLinkHealth_Diverged_CopyMode_DifferentContent(t *testing.T) {
	storeDir, _ := setupLinkHealthEnv(t)

	writeStoreFileHealth(t, storeDir, "instructions/AGENTS.md", "canonical content")

	// Target has different content.
	target := filepath.Join(t.TempDir(), "CLAUDE.md")
	if err := os.WriteFile(target, []byte("modified content"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	entry := LinkEntry{
		Source: "instructions/AGENTS.md",
		Target: target,
		Agent:  "claude",
		Mode:   "copy",
	}

	h := CheckLinkHealth(entry, storeDir)
	if h.State != LinkHealthDiverged {
		t.Errorf("State = %q, want %q", h.State, LinkHealthDiverged)
	}
}

func TestCheckLinkHealth_Replaced_WrongSymlink(t *testing.T) {
	storeDir, _ := setupLinkHealthEnv(t)

	writeStoreFileHealth(t, storeDir, "instructions/AGENTS.md", "canonical")

	// Symlink points to a different file that exists.
	otherFile := filepath.Join(t.TempDir(), "other.md")
	if err := os.WriteFile(otherFile, []byte("other"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	target := filepath.Join(t.TempDir(), "CLAUDE.md")
	if err := os.Symlink(otherFile, target); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	entry := LinkEntry{
		Source: "instructions/AGENTS.md",
		Target: target,
		Agent:  "claude",
		Mode:   "symlink",
	}

	h := CheckLinkHealth(entry, storeDir)
	if h.State != LinkHealthReplaced {
		t.Errorf("State = %q, want %q", h.State, LinkHealthReplaced)
	}
}

// --- contentMatches unit tests ---

func TestContentMatches_Files_Equal(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	b := filepath.Join(dir, "b.txt")
	content := []byte("hello world")
	if err := os.WriteFile(a, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if !contentMatches(a, b) {
		t.Error("contentMatches = false for identical files, want true")
	}
}

func TestContentMatches_Files_Different(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	b := filepath.Join(dir, "b.txt")
	if err := os.WriteFile(a, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("world"), 0o644); err != nil {
		t.Fatal(err)
	}
	if contentMatches(a, b) {
		t.Error("contentMatches = true for different files, want false")
	}
}

func TestContentMatches_Dirs_Equal(t *testing.T) {
	base := t.TempDir()
	a := filepath.Join(base, "a")
	b := filepath.Join(base, "b")
	if err := os.MkdirAll(a, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(b, 0o755); err != nil {
		t.Fatal(err)
	}
	// Write identical files in both.
	if err := os.WriteFile(filepath.Join(a, "f.txt"), []byte("same"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(b, "f.txt"), []byte("same"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !contentMatches(a, b) {
		t.Error("contentMatches = false for identical dirs, want true")
	}
}

func TestContentMatches_Dirs_Different(t *testing.T) {
	base := t.TempDir()
	a := filepath.Join(base, "a")
	b := filepath.Join(base, "b")
	if err := os.MkdirAll(a, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(b, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(a, "f.txt"), []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(b, "f.txt"), []byte("modified"), 0o644); err != nil {
		t.Fatal(err)
	}
	if contentMatches(a, b) {
		t.Error("contentMatches = true for dirs with different file content, want false")
	}
}

// --- CheckStatus integration test ---

func TestCheckStatus_NotInitialized(t *testing.T) {
	setupEnv(t)
	// Do NOT call Init() — store should not exist.
	// CheckStatus returns (empty results, nil) even when uninitialized;
	// the caller (runAgentsStatus) is responsible for checking IsInitialized().
	result, err := CheckStatus()
	if err != nil {
		t.Errorf("CheckStatus() on uninitialized store: unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("CheckStatus() returned nil result, want empty StatusResult")
	}
	if len(result.Links) != 0 {
		t.Errorf("Links len = %d on uninitialized store, want 0", len(result.Links))
	}
}

func TestCheckStatus_InitializedEmpty(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	result, err := CheckStatus()
	if err != nil {
		t.Fatalf("CheckStatus: %v", err)
	}

	if result.Store.Dir == "" {
		t.Error("Store.Dir is empty")
	}
	if len(result.Links) != 0 {
		t.Errorf("Links len = %d, want 0 for empty store", len(result.Links))
	}
}

func TestCheckStatus_WithLinks(t *testing.T) {
	storeDir, _ := setupLinkHealthEnv(t)

	// Write a canonical source file.
	sourcePath := writeStoreFileHealth(t, storeDir, "instructions/AGENTS.md", "content")

	// Create a valid symlink at target.
	target := filepath.Join(t.TempDir(), "CLAUDE.md")
	if err := os.Symlink(sourcePath, target); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	m := &Manifest{
		Agents: []Agent{{Name: "claude", Detected: true, ConfigDir: t.TempDir()}},
		Links: []LinkEntry{
			{Source: "instructions/AGENTS.md", Target: target, Agent: "claude", Mode: "symlink"},
		},
	}
	if err := WriteManifest(m); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}

	result, err := CheckStatus()
	if err != nil {
		t.Fatalf("CheckStatus: %v", err)
	}

	if len(result.Links) != 1 {
		t.Fatalf("Links len = %d, want 1", len(result.Links))
	}
	if result.Links[0].State != LinkHealthLinked {
		t.Errorf("link[0].State = %q, want %q", result.Links[0].State, LinkHealthLinked)
	}
}
