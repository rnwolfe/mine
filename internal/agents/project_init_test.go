package agents

import (
	"os"
	"path/filepath"
	"testing"
)

// setupProjectEnv creates a temp environment with an initialized agents store and
// a fresh project directory. Returns (storeDir, projectDir).
func setupProjectEnv(t *testing.T) (string, string) {
	t.Helper()
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	homeDir := filepath.Join(tmpDir, "home")
	projectDir := filepath.Join(tmpDir, "myproject")

	for _, d := range []string{dataDir, homeDir, projectDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	t.Setenv("XDG_DATA_HOME", dataDir)
	t.Setenv("HOME", homeDir)

	if err := Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	return Dir(), projectDir
}

// addDetectedAgent appends a detected agent entry to the manifest.
func addDetectedAgent(t *testing.T, name string) {
	t.Helper()
	m, err := ReadManifest()
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	m.Agents = append(m.Agents, Agent{Name: name, Detected: true})
	if err := WriteManifest(m); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}
}

// --- buildProjectSpecRegistry ---

func TestBuildProjectSpecRegistry_ContainsAllAgents(t *testing.T) {
	specs := buildProjectSpecRegistry()
	wantNames := []string{"claude", "codex", "gemini", "opencode"}

	if len(specs) != len(wantNames) {
		t.Fatalf("buildProjectSpecRegistry() returned %d specs, want %d", len(specs), len(wantNames))
	}

	nameSet := make(map[string]bool)
	for _, s := range specs {
		nameSet[s.Name] = true
	}
	for _, name := range wantNames {
		if !nameSet[name] {
			t.Errorf("registry missing agent %q", name)
		}
	}
}

func TestBuildProjectSpecRegistry_ClaudeHasCommands(t *testing.T) {
	specs := buildProjectSpecRegistry()
	for _, s := range specs {
		if s.Name == "claude" {
			if s.CommandsSubDir == "" {
				t.Error("claude.CommandsSubDir is empty, want non-empty")
			}
			return
		}
	}
	t.Fatal("claude spec not found in registry")
}

func TestBuildProjectSpecRegistry_CodexConfigDir(t *testing.T) {
	specs := buildProjectSpecRegistry()
	for _, s := range specs {
		if s.Name == "codex" {
			if s.ConfigDir != ".agents" {
				t.Errorf("codex.ConfigDir = %q, want %q", s.ConfigDir, ".agents")
			}
			return
		}
	}
	t.Fatal("codex spec not found in registry")
}

func TestBuildProjectSpecRegistry_InstructionFilenames(t *testing.T) {
	wantFilenames := map[string]string{
		"claude":   "CLAUDE.md",
		"codex":    "AGENTS.md",
		"gemini":   "GEMINI.md",
		"opencode": "AGENTS.md",
	}
	for _, s := range buildProjectSpecRegistry() {
		want, ok := wantFilenames[s.Name]
		if !ok {
			continue
		}
		if s.InstructionFile != want {
			t.Errorf("agent %q InstructionFile = %q, want %q", s.Name, s.InstructionFile, want)
		}
	}
}

// --- validateProjectPath ---

func TestValidateProjectPath_Missing(t *testing.T) {
	err := validateProjectPath(filepath.Join(t.TempDir(), "nonexistent"))
	if err == nil {
		t.Error("validateProjectPath() error = nil for missing path, want error")
	}
}

func TestValidateProjectPath_File(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "file.txt")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := validateProjectPath(f); err == nil {
		t.Error("validateProjectPath() error = nil for file path, want error")
	}
}

func TestValidateProjectPath_ValidDir(t *testing.T) {
	if err := validateProjectPath(t.TempDir()); err != nil {
		t.Errorf("validateProjectPath() error = %v, want nil for valid dir", err)
	}
}

// --- initProjectDir ---

func TestInitProjectDir_CreatesNew(t *testing.T) {
	path := filepath.Join(t.TempDir(), "newdir")
	a := initProjectDir(path)
	if a.Err != nil {
		t.Errorf("initProjectDir() error = %v, want nil", a.Err)
	}
	if a.Status != "created" {
		t.Errorf("initProjectDir() status = %q, want %q", a.Status, "created")
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("directory not created: %v", err)
	}
}

func TestInitProjectDir_ExistingDir(t *testing.T) {
	path := t.TempDir()
	a := initProjectDir(path)
	if a.Err != nil {
		t.Errorf("initProjectDir() error = %v for existing dir, want nil", a.Err)
	}
	if a.Status != "exists" {
		t.Errorf("initProjectDir() status = %q for existing dir, want %q", a.Status, "exists")
	}
}

// --- ProjectInit ---

func TestProjectInit_NoDetectedAgents_ReturnsEmpty(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}
	// Manifest has no detected agents.
	projectDir := t.TempDir()
	actions, err := ProjectInit(projectDir, ProjectInitOptions{})
	if err != nil {
		t.Fatalf("ProjectInit() error = %v", err)
	}
	if len(actions) != 0 {
		t.Errorf("ProjectInit() returned %d actions, want 0 for no detected agents", len(actions))
	}
}

func TestProjectInit_CreatesAgentConfigDirs(t *testing.T) {
	_, projectDir := setupProjectEnv(t)
	addDetectedAgent(t, "claude")

	actions, err := ProjectInit(projectDir, ProjectInitOptions{})
	if err != nil {
		t.Fatalf("ProjectInit() error = %v", err)
	}
	if len(actions) == 0 {
		t.Fatal("ProjectInit() returned no actions for detected agent")
	}

	// .claude/ must be created.
	claudeDir := filepath.Join(projectDir, ".claude")
	if _, err := os.Stat(claudeDir); err != nil {
		t.Errorf(".claude/ not created: %v", err)
	}

	// .claude/skills/ must be created.
	if _, err := os.Stat(filepath.Join(claudeDir, "skills")); err != nil {
		t.Errorf(".claude/skills/ not created: %v", err)
	}

	// .claude/commands/ must be created (claude-specific).
	if _, err := os.Stat(filepath.Join(claudeDir, "commands")); err != nil {
		t.Errorf(".claude/commands/ not created: %v", err)
	}
}

func TestProjectInit_CodexUsesAgentsDir(t *testing.T) {
	_, projectDir := setupProjectEnv(t)
	addDetectedAgent(t, "codex")

	if _, err := ProjectInit(projectDir, ProjectInitOptions{}); err != nil {
		t.Fatalf("ProjectInit() error = %v", err)
	}

	// Codex uses .agents/ at project level.
	if _, err := os.Stat(filepath.Join(projectDir, ".agents")); err != nil {
		t.Errorf(".agents/ not created for codex: %v", err)
	}

	// .codex/ must NOT be created.
	if _, err := os.Stat(filepath.Join(projectDir, ".codex")); err == nil {
		t.Error(".codex/ was created, want .agents/ for codex project-level config")
	}
}

func TestProjectInit_CreatesInstructionFile(t *testing.T) {
	_, projectDir := setupProjectEnv(t)
	addDetectedAgent(t, "claude")

	if _, err := ProjectInit(projectDir, ProjectInitOptions{}); err != nil {
		t.Fatalf("ProjectInit() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(projectDir, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("CLAUDE.md not created: %v", err)
	}
	if len(data) == 0 {
		t.Error("CLAUDE.md is empty, want starter content")
	}
}

func TestProjectInit_DeduplicatesSharedInstructionFile(t *testing.T) {
	_, projectDir := setupProjectEnv(t)
	// Both codex and opencode use AGENTS.md.
	addDetectedAgent(t, "codex")
	addDetectedAgent(t, "opencode")

	actions, err := ProjectInit(projectDir, ProjectInitOptions{})
	if err != nil {
		t.Fatalf("ProjectInit() error = %v", err)
	}

	// AGENTS.md must be created exactly once.
	agentsMDPath := filepath.Join(projectDir, "AGENTS.md")
	if _, err := os.ReadFile(agentsMDPath); err != nil {
		t.Fatalf("AGENTS.md not created: %v", err)
	}

	// Count how many actions refer to AGENTS.md.
	count := 0
	for _, a := range actions {
		if a.Path == agentsMDPath {
			count++
		}
	}
	if count != 1 {
		t.Errorf("AGENTS.md appears %d times in actions, want 1 (deduplicated)", count)
	}
}

func TestProjectInit_OnlyScaffoldsDetectedAgents(t *testing.T) {
	_, projectDir := setupProjectEnv(t)
	// Only claude is detected.
	addDetectedAgent(t, "claude")

	if _, err := ProjectInit(projectDir, ProjectInitOptions{}); err != nil {
		t.Fatalf("ProjectInit() error = %v", err)
	}

	// .claude/ must exist.
	if _, err := os.Stat(filepath.Join(projectDir, ".claude")); err != nil {
		t.Error(".claude/ not created for detected agent")
	}

	// .agents/ (codex) must NOT exist.
	if _, err := os.Stat(filepath.Join(projectDir, ".agents")); err == nil {
		t.Error(".agents/ was created for non-detected codex, want not created")
	}
}

func TestProjectInit_Idempotent(t *testing.T) {
	_, projectDir := setupProjectEnv(t)
	addDetectedAgent(t, "claude")

	// First init.
	actions1, err := ProjectInit(projectDir, ProjectInitOptions{})
	if err != nil {
		t.Fatalf("first ProjectInit() error = %v", err)
	}
	created := 0
	for _, a := range actions1 {
		if a.Status == "created" {
			created++
		}
	}
	if created == 0 {
		t.Fatal("first init created nothing")
	}

	// Write custom content to CLAUDE.md.
	claudeMD := filepath.Join(projectDir, "CLAUDE.md")
	customContent := "# My Custom Claude Instructions\n"
	if err := os.WriteFile(claudeMD, []byte(customContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Second init — must not overwrite.
	actions2, err := ProjectInit(projectDir, ProjectInitOptions{})
	if err != nil {
		t.Fatalf("second ProjectInit() error = %v", err)
	}
	for _, a := range actions2 {
		if a.Status == "created" {
			t.Errorf("second init created %q (should be idempotent), want status=exists", a.Path)
		}
	}

	// Custom content must be preserved.
	data, err := os.ReadFile(claudeMD)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != customContent {
		t.Errorf("CLAUDE.md overwritten on second init: got %q, want %q", string(data), customContent)
	}
}

func TestProjectInit_ForceOverwritesExisting(t *testing.T) {
	_, projectDir := setupProjectEnv(t)
	addDetectedAgent(t, "claude")

	// First init.
	if _, err := ProjectInit(projectDir, ProjectInitOptions{}); err != nil {
		t.Fatalf("first init: %v", err)
	}

	// Modify CLAUDE.md.
	claudeMD := filepath.Join(projectDir, "CLAUDE.md")
	if err := os.WriteFile(claudeMD, []byte("custom content"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Second init with force.
	actions, err := ProjectInit(projectDir, ProjectInitOptions{Force: true})
	if err != nil {
		t.Fatalf("force init: %v", err)
	}

	var fileRecreated bool
	for _, a := range actions {
		if a.Path == claudeMD && a.Status == "created" {
			fileRecreated = true
			break
		}
	}
	if !fileRecreated {
		t.Error("CLAUDE.md not recreated with --force, want recreated")
	}

	data, err := os.ReadFile(claudeMD)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == "custom content" {
		t.Error("CLAUDE.md still has custom content after --force init, want template")
	}
}

func TestProjectInit_SeedsSettingsFromCanonicalStore(t *testing.T) {
	storeDir, projectDir := setupProjectEnv(t)
	addDetectedAgent(t, "claude")

	// Place a settings template in the canonical store.
	settingsContent := `{"model": "claude-opus-4-6"}`
	settingsSrc := filepath.Join(storeDir, "settings", "claude.json")
	if err := os.WriteFile(settingsSrc, []byte(settingsContent), 0o644); err != nil {
		t.Fatalf("writing settings: %v", err)
	}

	if _, err := ProjectInit(projectDir, ProjectInitOptions{}); err != nil {
		t.Fatalf("ProjectInit: %v", err)
	}

	// settings.json must be seeded in .claude/.
	dst := filepath.Join(projectDir, ".claude", "settings.json")
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("settings.json not seeded: %v", err)
	}
	if string(data) != settingsContent {
		t.Errorf("settings.json content = %q, want %q", string(data), settingsContent)
	}
}

func TestProjectInit_SettingsNotSeedWhenNoTemplate(t *testing.T) {
	_, projectDir := setupProjectEnv(t)
	addDetectedAgent(t, "claude")

	// No settings/claude.json in the canonical store.
	if _, err := ProjectInit(projectDir, ProjectInitOptions{}); err != nil {
		t.Fatalf("ProjectInit: %v", err)
	}

	// settings.json must NOT be created.
	dst := filepath.Join(projectDir, ".claude", "settings.json")
	if _, err := os.Stat(dst); err == nil {
		t.Error("settings.json created without a canonical template, want not created")
	}
}

func TestProjectInit_DefaultsToCWD(t *testing.T) {
	_, projectDir := setupProjectEnv(t)
	addDetectedAgent(t, "claude")

	// Change CWD to projectDir.
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldDir) })

	// Empty path should use CWD.
	if _, err := ProjectInit("", ProjectInitOptions{}); err != nil {
		t.Fatalf("ProjectInit with empty path: %v", err)
	}

	if _, err := os.Stat(filepath.Join(projectDir, ".claude")); err != nil {
		t.Error(".claude/ not created in CWD when path is empty")
	}
}

func TestProjectInit_InvalidPath(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	_, err := ProjectInit(filepath.Join(t.TempDir(), "nonexistent"), ProjectInitOptions{})
	if err == nil {
		t.Error("ProjectInit() error = nil for nonexistent path, want error")
	}
}
