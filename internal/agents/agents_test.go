package agents

import (
	"os"
	"path/filepath"
	"testing"
)

// setupEnv creates a temp environment for agent tests and returns the agents dir.
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
	setupEnv(t)
	dir := Dir()
	if dir == "" {
		t.Error("Dir() returned empty string")
	}
	if filepath.Base(dir) != "agents" {
		t.Errorf("Dir() base = %q, want %q", filepath.Base(dir), "agents")
	}
}

func TestManifestPath(t *testing.T) {
	setupEnv(t)
	mp := ManifestPath()
	if filepath.Base(mp) != ".mine-agents" {
		t.Errorf("ManifestPath() base = %q, want %q", filepath.Base(mp), ".mine-agents")
	}
	if filepath.Dir(mp) != Dir() {
		t.Errorf("ManifestPath() dir = %q, want %q", filepath.Dir(mp), Dir())
	}
}

func TestIsInitialized_False(t *testing.T) {
	setupEnv(t)
	if IsInitialized() {
		t.Error("IsInitialized() = true before init, want false")
	}
}

func TestIsInitialized_True(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	if !IsInitialized() {
		t.Error("IsInitialized() = false after Init, want true")
	}
}

func TestInit_CreatesDirectoryScaffold(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	dir := Dir()
	subdirs := []string{"instructions", "skills", "commands", "agents", "settings", "mcp", "rules"}
	for _, sub := range subdirs {
		p := filepath.Join(dir, sub)
		if _, err := os.Stat(p); err != nil {
			t.Errorf("subdirectory %q missing after Init: %v", sub, err)
		}
	}
}

func TestInit_CreatesAgentsMD(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	agentsMDPath := filepath.Join(Dir(), "instructions", "AGENTS.md")
	data, err := os.ReadFile(agentsMDPath)
	if err != nil {
		t.Fatalf("AGENTS.md missing after Init: %v", err)
	}
	if len(data) == 0 {
		t.Error("AGENTS.md is empty, want starter content")
	}
}

func TestInit_InitializesGitRepo(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	gitDir := filepath.Join(Dir(), ".git")
	if _, err := os.Stat(gitDir); err != nil {
		t.Errorf(".git missing after Init: %v", err)
	}
}

func TestInit_CreatesManifest(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	if _, err := os.Stat(ManifestPath()); err != nil {
		t.Errorf("manifest missing after Init: %v", err)
	}
}

func TestInit_Idempotent(t *testing.T) {
	setupEnv(t)

	if err := Init(); err != nil {
		t.Fatalf("first Init() error: %v", err)
	}

	// Modify AGENTS.md to verify it is not overwritten on second Init.
	agentsMDPath := filepath.Join(Dir(), "instructions", "AGENTS.md")
	custom := "# My custom instructions\n"
	if err := os.WriteFile(agentsMDPath, []byte(custom), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Init(); err != nil {
		t.Fatalf("second Init() error: %v", err)
	}

	data, err := os.ReadFile(agentsMDPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != custom {
		t.Errorf("AGENTS.md was overwritten on second Init(); got %q, want %q", string(data), custom)
	}
}

func TestInit_Idempotent_GitRepo(t *testing.T) {
	setupEnv(t)

	if err := Init(); err != nil {
		t.Fatalf("first Init() error: %v", err)
	}

	// Second init should not fail even though .git already exists.
	if err := Init(); err != nil {
		t.Fatalf("second Init() error: %v", err)
	}
}

func TestReadManifest_NoFile(t *testing.T) {
	setupEnv(t)

	// Call ReadManifest before Init — should return empty manifest, not error.
	m, err := ReadManifest()
	if err != nil {
		t.Fatalf("ReadManifest() error: %v", err)
	}
	if m == nil {
		t.Error("ReadManifest() returned nil for missing file, want empty manifest")
	}
	if m.Agents == nil {
		t.Error("ReadManifest() Agents is nil, want non-nil empty slice")
	}
	if m.Links == nil {
		t.Error("ReadManifest() Links is nil, want non-nil empty slice")
	}
	if len(m.Agents) != 0 {
		t.Errorf("Agents count = %d, want 0", len(m.Agents))
	}
	if len(m.Links) != 0 {
		t.Errorf("Links count = %d, want 0", len(m.Links))
	}
}

func TestReadWriteManifest_RoundTrip(t *testing.T) {
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
		t.Fatalf("Agents count = %d, want 1", len(got.Agents))
	}
	if got.Agents[0].Name != "claude" {
		t.Errorf("Agent.Name = %q, want %q", got.Agents[0].Name, "claude")
	}
	if got.Agents[0].Detected != true {
		t.Errorf("Agent.Detected = %v, want true", got.Agents[0].Detected)
	}
	if got.Agents[0].ConfigDir != "/home/user/.claude" {
		t.Errorf("Agent.ConfigDir = %q, want %q", got.Agents[0].ConfigDir, "/home/user/.claude")
	}
	if got.Agents[0].Binary != "claude" {
		t.Errorf("Agent.Binary = %q, want %q", got.Agents[0].Binary, "claude")
	}

	if len(got.Links) != 1 {
		t.Fatalf("Links count = %d, want 1", len(got.Links))
	}
	if got.Links[0].Source != "instructions/AGENTS.md" {
		t.Errorf("LinkEntry.Source = %q, want %q", got.Links[0].Source, "instructions/AGENTS.md")
	}
	if got.Links[0].Mode != "symlink" {
		t.Errorf("LinkEntry.Mode = %q, want %q", got.Links[0].Mode, "symlink")
	}
}

func TestWriteManifest_EmptySlicesNonNilAfterRead(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	m := &Manifest{
		Agents: []Agent{},
		Links:  []LinkEntry{},
	}
	if err := WriteManifest(m); err != nil {
		t.Fatalf("WriteManifest() error: %v", err)
	}

	got, err := ReadManifest()
	if err != nil {
		t.Fatalf("ReadManifest() error: %v", err)
	}
	if got.Agents == nil {
		t.Error("ReadManifest() Agents is nil after writing empty slice")
	}
	if got.Links == nil {
		t.Error("ReadManifest() Links is nil after writing empty slice")
	}
}

func TestWriteManifest_MultipleAgentsAndLinks(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	m := &Manifest{
		Agents: []Agent{
			{Name: "claude", Detected: true, ConfigDir: "/home/user/.claude", Binary: "claude"},
			{Name: "gemini", Detected: false, ConfigDir: "/home/user/.gemini", Binary: "gemini"},
		},
		Links: []LinkEntry{
			{Source: "instructions/AGENTS.md", Target: "/home/user/.claude/CLAUDE.md", Agent: "claude", Mode: "symlink"},
			{Source: "rules/style.md", Target: "/home/user/.gemini/rules.md", Agent: "gemini", Mode: "copy"},
		},
	}

	if err := WriteManifest(m); err != nil {
		t.Fatalf("WriteManifest() error: %v", err)
	}

	got, err := ReadManifest()
	if err != nil {
		t.Fatalf("ReadManifest() error: %v", err)
	}

	if len(got.Agents) != 2 {
		t.Errorf("Agents count = %d, want 2", len(got.Agents))
	}
	if len(got.Links) != 2 {
		t.Errorf("Links count = %d, want 2", len(got.Links))
	}
	if got.Links[1].Mode != "copy" {
		t.Errorf("second Link.Mode = %q, want %q", got.Links[1].Mode, "copy")
	}
}
