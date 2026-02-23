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

	return Dir()
}

func TestDir(t *testing.T) {
	agentsDir := setupEnv(t)
	if agentsDir == "" {
		t.Error("Dir() returned empty string")
	}
	if filepath.Base(agentsDir) != "agents" {
		t.Errorf("Dir() base = %q, want %q", filepath.Base(agentsDir), "agents")
	}
}

func TestManifestPath(t *testing.T) {
	setupEnv(t)
	mp := ManifestPath()
	if filepath.Base(mp) != ".mine-agents" {
		t.Errorf("ManifestPath() base = %q, want %q", filepath.Base(mp), ".mine-agents")
	}
}

func TestIsInitialized(t *testing.T) {
	setupEnv(t)

	if IsInitialized() {
		t.Error("IsInitialized() = true before Init(), want false")
	}

	if err := Init(); err != nil {
		t.Fatal(err)
	}

	if !IsInitialized() {
		t.Error("IsInitialized() = false after Init(), want true")
	}
}

func TestInit_Idempotent(t *testing.T) {
	setupEnv(t)

	if err := Init(); err != nil {
		t.Fatalf("first Init() error: %v", err)
	}

	if err := Init(); err != nil {
		t.Fatalf("second Init() error (should be idempotent): %v", err)
	}
}

func TestInit_CreatesSubdirs(t *testing.T) {
	agentsDir := setupEnv(t)

	if err := Init(); err != nil {
		t.Fatal(err)
	}

	expectedDirs := []string{"instructions", "skills", "commands", "agents", "settings", "mcp", "rules"}
	for _, sub := range expectedDirs {
		path := filepath.Join(agentsDir, sub)
		if info, err := os.Stat(path); err != nil {
			t.Errorf("subdir %s not created: %v", sub, err)
		} else if !info.IsDir() {
			t.Errorf("subdir %s is not a directory", sub)
		}
	}
}

func TestInit_CreatesAgentsMD(t *testing.T) {
	agentsDir := setupEnv(t)

	if err := Init(); err != nil {
		t.Fatal(err)
	}

	agentsMD := filepath.Join(agentsDir, "instructions", "AGENTS.md")
	if _, err := os.Stat(agentsMD); err != nil {
		t.Errorf("AGENTS.md not created: %v", err)
	}
}

func TestReadWriteManifest(t *testing.T) {
	setupEnv(t)

	if err := Init(); err != nil {
		t.Fatal(err)
	}

	m := &Manifest{
		Agents: []Agent{
			{Name: "claude", Detected: true, ConfigDir: "/home/user/.claude", Binary: "claude"},
		},
		Links: []LinkEntry{
			{Source: "instructions/AGENTS.md", Target: "/home/user/.claude/CLAUDE.md", Agent: "claude", Mode: "symlink"},
		},
	}

	if err := WriteManifest(m); err != nil {
		t.Fatalf("WriteManifest() error: %v", err)
	}

	got, err := ReadManifest()
	if err != nil {
		t.Fatalf("ReadManifest() error: %v", err)
	}

	if len(got.Agents) != 1 {
		t.Errorf("Agents count = %d, want 1", len(got.Agents))
	}
	if got.Agents[0].Name != "claude" {
		t.Errorf("Agent.Name = %q, want %q", got.Agents[0].Name, "claude")
	}
	if len(got.Links) != 1 {
		t.Errorf("Links count = %d, want 1", len(got.Links))
	}
	if got.Links[0].Mode != "symlink" {
		t.Errorf("Link.Mode = %q, want %q", got.Links[0].Mode, "symlink")
	}
}

func TestReadManifest_Empty(t *testing.T) {
	setupEnv(t)

	m, err := ReadManifest()
	if err != nil {
		t.Fatalf("ReadManifest() on missing file error: %v", err)
	}
	if m == nil {
		t.Fatal("ReadManifest() returned nil for missing file")
	}
	if len(m.Agents) != 0 {
		t.Errorf("Agents = %v, want empty", m.Agents)
	}
	if len(m.Links) != 0 {
		t.Errorf("Links = %v, want empty", m.Links)
	}
}

func TestReadManifest_InvalidJSON(t *testing.T) {
	agentsDir := setupEnv(t)
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(ManifestPath(), []byte("not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ReadManifest()
	if err == nil {
		t.Error("ReadManifest() should error on invalid JSON")
	}
}

func TestWriteManifest_ValidJSON(t *testing.T) {
	setupEnv(t)

	if err := Init(); err != nil {
		t.Fatal(err)
	}

	m := &Manifest{
		Agents: []Agent{{Name: "test", Detected: false}},
		Links:  []LinkEntry{},
	}

	if err := WriteManifest(m); err != nil {
		t.Fatalf("WriteManifest() error: %v", err)
	}

	data, err := os.ReadFile(ManifestPath())
	if err != nil {
		t.Fatal(err)
	}

	var parsed Manifest
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Errorf("WriteManifest produced invalid JSON: %v", err)
	}
}
