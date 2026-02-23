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

func TestIsInitialized_RegularFile(t *testing.T) {
	agentsDir := setupEnv(t)
	// Create a regular file where the store directory should be.
	if err := os.MkdirAll(filepath.Dir(agentsDir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(agentsDir, []byte("not a dir"), 0o644); err != nil {
		t.Fatal(err)
	}
	if IsInitialized() {
		t.Error("IsInitialized() = true when store path is a regular file, want false")
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
		{"current directory dot", ".", true},
		{"path resolves to dot", "a/..", true},
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
