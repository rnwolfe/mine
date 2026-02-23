package agents

import (
	"os"
	"path/filepath"
	"testing"
)

// setupAdoptEnv creates a temp environment with agents store initialized.
// Returns (storeDir, homeDir).
func setupAdoptEnv(t *testing.T) (string, string) {
	t.Helper()
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	homeDir := filepath.Join(tmpDir, "home")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_DATA_HOME", dataDir)
	t.Setenv("HOME", homeDir)

	if err := Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	return Dir(), homeDir
}

// writeAgentFile creates a file in the agent's config directory.
func writeAgentFile(t *testing.T, configDir, relPath, content string) {
	t.Helper()
	p := filepath.Join(configDir, relPath)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("writing agent file %s: %v", relPath, err)
	}
}

// --- Adopt ---

func TestAdopt_NotInitialized(t *testing.T) {
	setupEnv(t) // fresh XDG env but no Init()
	_, err := Adopt(AdoptOptions{})
	if err == nil {
		t.Error("Adopt() error = nil for uninitialized store, want error")
	}
}

func TestAdopt_NoDetectedAgents_ReturnsEmpty(t *testing.T) {
	storeDir, _ := setupAdoptEnv(t)
	_ = storeDir

	// Manifest has no detected agents.
	items, err := Adopt(AdoptOptions{})
	if err != nil {
		t.Fatalf("Adopt() error = %v", err)
	}
	if len(items) != 0 {
		t.Errorf("Adopt() returned %d items, want 0 (no detected agents with config)", len(items))
	}
}

func TestAdopt_ImportsInstructionFile(t *testing.T) {
	storeDir, homeDir := setupAdoptEnv(t)

	claudeConfigDir := filepath.Join(homeDir, ".claude")
	writeAgentFile(t, claudeConfigDir, "CLAUDE.md", "# Claude Instructions\n")
	makeDetectedAgent(t, "claude", claudeConfigDir)

	// Remove the starter AGENTS.md so the store doesn't already have content.
	starterFile := filepath.Join(storeDir, "instructions", "AGENTS.md")
	if err := os.Remove(starterFile); err != nil {
		t.Fatalf("removing starter AGENTS.md: %v", err)
	}

	items, err := Adopt(AdoptOptions{Copy: true}) // --copy to skip symlink creation
	if err != nil {
		t.Fatalf("Adopt() error = %v", err)
	}

	var instrItem *AdoptItem
	for i := range items {
		if items[i].Kind == "instruction" && items[i].Agent == "claude" {
			instrItem = &items[i]
			break
		}
	}
	if instrItem == nil {
		t.Fatal("no instruction item for claude")
	}
	if instrItem.Status != "imported" {
		t.Errorf("instruction item.Status = %q, want %q", instrItem.Status, "imported")
	}
	if instrItem.Err != nil {
		t.Errorf("instruction item.Err = %v, want nil", instrItem.Err)
	}

	// Verify content was copied to the store.
	data, err := os.ReadFile(filepath.Join(storeDir, "instructions", "AGENTS.md"))
	if err != nil {
		t.Fatalf("reading store instructions: %v", err)
	}
	if string(data) != "# Claude Instructions\n" {
		t.Errorf("store instructions content = %q, want %q", string(data), "# Claude Instructions\n")
	}
}

func TestAdopt_DryRun_NoChanges(t *testing.T) {
	storeDir, homeDir := setupAdoptEnv(t)

	claudeConfigDir := filepath.Join(homeDir, ".claude")
	writeAgentFile(t, claudeConfigDir, "CLAUDE.md", "# Claude Instructions\n")
	makeDetectedAgent(t, "claude", claudeConfigDir)

	starterFile := filepath.Join(storeDir, "instructions", "AGENTS.md")
	if err := os.Remove(starterFile); err != nil {
		t.Fatalf("removing starter AGENTS.md: %v", err)
	}

	items, err := Adopt(AdoptOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Adopt() with DryRun error = %v", err)
	}

	// Items should be returned but store should be unchanged.
	if len(items) == 0 {
		t.Error("Adopt(DryRun) returned no items, want at least one")
	}

	// Store instructions file should NOT exist.
	if _, err := os.Stat(starterFile); err == nil {
		t.Error("store instructions file was created during dry run, want no changes")
	}
}

func TestAdopt_InstructionConflict_Skipped(t *testing.T) {
	storeDir, homeDir := setupAdoptEnv(t)

	// Store already has different instructions.
	storeInstr := filepath.Join(storeDir, "instructions", "AGENTS.md")
	if err := os.WriteFile(storeInstr, []byte("# Existing Store Instructions\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	claudeConfigDir := filepath.Join(homeDir, ".claude")
	writeAgentFile(t, claudeConfigDir, "CLAUDE.md", "# Different Claude Instructions\n")
	makeDetectedAgent(t, "claude", claudeConfigDir)

	items, err := Adopt(AdoptOptions{})
	if err != nil {
		t.Fatalf("Adopt() error = %v", err)
	}

	var instrItem *AdoptItem
	for i := range items {
		if items[i].Kind == "instruction" && items[i].Agent == "claude" {
			instrItem = &items[i]
			break
		}
	}
	if instrItem == nil {
		t.Fatal("no instruction item for claude")
	}
	if !instrItem.Conflict {
		t.Error("instruction item.Conflict = false, want true when content differs")
	}
	if instrItem.Status != "conflict" {
		t.Errorf("instruction item.Status = %q, want %q", instrItem.Status, "conflict")
	}

	// Store content must be unchanged.
	data, err := os.ReadFile(storeInstr)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "# Existing Store Instructions\n" {
		t.Error("store instructions were overwritten during conflict, want original content preserved")
	}
}

func TestAdopt_SameContent_AlreadyManaged(t *testing.T) {
	storeDir, homeDir := setupAdoptEnv(t)

	content := "# Shared Instructions\n"

	// Store has identical content.
	storeInstr := filepath.Join(storeDir, "instructions", "AGENTS.md")
	if err := os.WriteFile(storeInstr, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	claudeConfigDir := filepath.Join(homeDir, ".claude")
	writeAgentFile(t, claudeConfigDir, "CLAUDE.md", content)
	makeDetectedAgent(t, "claude", claudeConfigDir)

	items, err := Adopt(AdoptOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Adopt() error = %v", err)
	}

	var instrItem *AdoptItem
	for i := range items {
		if items[i].Kind == "instruction" && items[i].Agent == "claude" {
			instrItem = &items[i]
			break
		}
	}
	if instrItem == nil {
		t.Fatal("no instruction item for claude")
	}
	if instrItem.Conflict {
		t.Error("instruction item.Conflict = true, want false when content is identical")
	}
	if instrItem.Status != "already-managed" {
		t.Errorf("instruction item.Status = %q, want %q", instrItem.Status, "already-managed")
	}
}

func TestAdopt_SkipsAlreadyManagedSymlink(t *testing.T) {
	storeDir, homeDir := setupAdoptEnv(t)

	claudeConfigDir := filepath.Join(homeDir, ".claude")
	if err := os.MkdirAll(claudeConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create CLAUDE.md as a symlink pointing to the store.
	storeInstr := filepath.Join(storeDir, "instructions", "AGENTS.md")
	claudeFile := filepath.Join(claudeConfigDir, "CLAUDE.md")
	if err := os.Symlink(storeInstr, claudeFile); err != nil {
		t.Fatal(err)
	}

	makeDetectedAgent(t, "claude", claudeConfigDir)

	items, err := Adopt(AdoptOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Adopt() error = %v", err)
	}

	for _, item := range items {
		if item.Kind == "instruction" && item.Agent == "claude" {
			t.Error("instruction item returned for file already managed by store symlink, want it skipped")
		}
	}
}

func TestAdopt_AgentFilter(t *testing.T) {
	storeDir, homeDir := setupAdoptEnv(t)

	// Remove starter AGENTS.md.
	if err := os.Remove(filepath.Join(storeDir, "instructions", "AGENTS.md")); err != nil {
		t.Fatalf("removing starter AGENTS.md: %v", err)
	}

	claudeConfigDir := filepath.Join(homeDir, ".claude")
	codexConfigDir := filepath.Join(homeDir, ".codex")
	writeAgentFile(t, claudeConfigDir, "CLAUDE.md", "# Claude\n")
	writeAgentFile(t, codexConfigDir, "AGENTS.md", "# Codex\n")
	makeDetectedAgent(t, "claude", claudeConfigDir)
	makeDetectedAgent(t, "codex", codexConfigDir)

	items, err := Adopt(AdoptOptions{Agent: "claude", DryRun: true})
	if err != nil {
		t.Fatalf("Adopt() error = %v", err)
	}

	for _, item := range items {
		if item.Agent != "claude" {
			t.Errorf("item for agent %q found, want only claude with --agent claude", item.Agent)
		}
	}
}

func TestAdopt_ImportsSkillsDirectory(t *testing.T) {
	storeDir, homeDir := setupAdoptEnv(t)

	claudeConfigDir := filepath.Join(homeDir, ".claude")
	writeAgentFile(t, claudeConfigDir, "skills/my-skill.md", "# My Skill\n")
	makeDetectedAgent(t, "claude", claudeConfigDir)

	items, err := Adopt(AdoptOptions{Copy: true})
	if err != nil {
		t.Fatalf("Adopt() error = %v", err)
	}

	var skillsItem *AdoptItem
	for i := range items {
		if items[i].Kind == "skills" && items[i].Agent == "claude" {
			skillsItem = &items[i]
			break
		}
	}
	if skillsItem == nil {
		t.Fatal("no skills item for claude")
	}
	if skillsItem.Status != "imported" {
		t.Errorf("skills item.Status = %q, want %q", skillsItem.Status, "imported")
	}

	// Verify skill file was copied to the store.
	data, err := os.ReadFile(filepath.Join(storeDir, "skills", "my-skill.md"))
	if err != nil {
		t.Fatalf("reading store skill: %v", err)
	}
	if string(data) != "# My Skill\n" {
		t.Errorf("store skill content = %q, want %q", string(data), "# My Skill\n")
	}
}

func TestAdopt_ImportsSettings(t *testing.T) {
	storeDir, homeDir := setupAdoptEnv(t)

	claudeConfigDir := filepath.Join(homeDir, ".claude")
	writeAgentFile(t, claudeConfigDir, "settings.json", `{"theme":"dark"}`)
	makeDetectedAgent(t, "claude", claudeConfigDir)

	items, err := Adopt(AdoptOptions{Copy: true})
	if err != nil {
		t.Fatalf("Adopt() error = %v", err)
	}

	var settingsItem *AdoptItem
	for i := range items {
		if items[i].Kind == "settings" && items[i].Agent == "claude" {
			settingsItem = &items[i]
			break
		}
	}
	if settingsItem == nil {
		t.Fatal("no settings item for claude")
	}
	if settingsItem.Status != "imported" {
		t.Errorf("settings item.Status = %q, want %q", settingsItem.Status, "imported")
	}

	// Verify settings were copied to settings/claude.json in the store.
	data, err := os.ReadFile(filepath.Join(storeDir, "settings", "claude.json"))
	if err != nil {
		t.Fatalf("reading store settings: %v", err)
	}
	if string(data) != `{"theme":"dark"}` {
		t.Errorf("store settings content = %q, want %q", string(data), `{"theme":"dark"}`)
	}
}

func TestAdopt_CreatesSymplinksAfterImport(t *testing.T) {
	storeDir, homeDir := setupAdoptEnv(t)

	claudeConfigDir := filepath.Join(homeDir, ".claude")
	writeAgentFile(t, claudeConfigDir, "CLAUDE.md", "# Instructions\n")
	makeDetectedAgent(t, "claude", claudeConfigDir)

	// Remove starter AGENTS.md so content is imported from claude.
	if err := os.Remove(filepath.Join(storeDir, "instructions", "AGENTS.md")); err != nil {
		t.Fatalf("removing starter AGENTS.md: %v", err)
	}

	// Run adopt WITHOUT --copy, so symlinks should be created.
	_, err := Adopt(AdoptOptions{})
	if err != nil {
		t.Fatalf("Adopt() error = %v", err)
	}

	// CLAUDE.md should now be a symlink pointing to the store.
	claudeFile := filepath.Join(claudeConfigDir, "CLAUDE.md")
	info, err := os.Lstat(claudeFile)
	if err != nil {
		t.Fatalf("Lstat CLAUDE.md after adopt: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("CLAUDE.md is not a symlink after adopt, want symlink to store")
	}

	dest, err := os.Readlink(claudeFile)
	if err != nil {
		t.Fatal(err)
	}
	expected := filepath.Join(storeDir, "instructions", "AGENTS.md")
	if dest != expected {
		t.Errorf("CLAUDE.md symlink points to %q, want %q", dest, expected)
	}
}

func TestAdopt_CopyMode_OriginalUnchanged(t *testing.T) {
	storeDir, homeDir := setupAdoptEnv(t)

	claudeConfigDir := filepath.Join(homeDir, ".claude")
	writeAgentFile(t, claudeConfigDir, "CLAUDE.md", "# Instructions\n")
	makeDetectedAgent(t, "claude", claudeConfigDir)

	if err := os.Remove(filepath.Join(storeDir, "instructions", "AGENTS.md")); err != nil {
		t.Fatalf("removing starter AGENTS.md: %v", err)
	}

	_, err := Adopt(AdoptOptions{Copy: true})
	if err != nil {
		t.Fatalf("Adopt() error = %v", err)
	}

	// CLAUDE.md should remain a regular file (not a symlink).
	claudeFile := filepath.Join(claudeConfigDir, "CLAUDE.md")
	info, err := os.Lstat(claudeFile)
	if err != nil {
		t.Fatalf("Lstat CLAUDE.md after adopt with --copy: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("CLAUDE.md is a symlink after adopt --copy, want original regular file kept")
	}
}

// --- fileConflict ---

func TestFileConflict_NoDst(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "source.md")
	dst := filepath.Join(tmp, "nonexistent.md")

	if err := os.WriteFile(src, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	if fileConflict(src, dst) {
		t.Error("fileConflict() = true when dst doesn't exist, want false")
	}
}

func TestFileConflict_SameContent(t *testing.T) {
	tmp := t.TempDir()
	content := "# Instructions\n"
	src := filepath.Join(tmp, "source.md")
	dst := filepath.Join(tmp, "dest.md")

	if err := os.WriteFile(src, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if fileConflict(src, dst) {
		t.Error("fileConflict() = true for identical content, want false")
	}
}

func TestFileConflict_DifferentContent(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "source.md")
	dst := filepath.Join(tmp, "dest.md")

	if err := os.WriteFile(src, []byte("# Source\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("# Destination\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if !fileConflict(src, dst) {
		t.Error("fileConflict() = false for different content, want true")
	}
}

// --- isAlreadyManagedByStore ---

func TestIsAlreadyManagedByStore_RegularFile(t *testing.T) {
	storeDir := t.TempDir()
	tmp := t.TempDir()
	p := filepath.Join(tmp, "file.md")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	if isAlreadyManagedByStore(p, storeDir) {
		t.Error("isAlreadyManagedByStore() = true for regular file, want false")
	}
}

func TestIsAlreadyManagedByStore_SymlinkInsideStore(t *testing.T) {
	storeDir := t.TempDir()
	storeFile := filepath.Join(storeDir, "instructions", "AGENTS.md")
	if err := os.MkdirAll(filepath.Dir(storeFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(storeFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	tmp := t.TempDir()
	link := filepath.Join(tmp, "CLAUDE.md")
	if err := os.Symlink(storeFile, link); err != nil {
		t.Fatal(err)
	}

	if !isAlreadyManagedByStore(link, storeDir) {
		t.Error("isAlreadyManagedByStore() = false for symlink pointing inside store, want true")
	}
}

func TestIsAlreadyManagedByStore_SymlinkOutsideStore(t *testing.T) {
	storeDir := t.TempDir()
	outsideFile := filepath.Join(t.TempDir(), "other.md")
	if err := os.WriteFile(outsideFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	tmp := t.TempDir()
	link := filepath.Join(tmp, "CLAUDE.md")
	if err := os.Symlink(outsideFile, link); err != nil {
		t.Fatal(err)
	}

	if isAlreadyManagedByStore(link, storeDir) {
		t.Error("isAlreadyManagedByStore() = true for symlink pointing outside store, want false")
	}
}

// --- appendUniq ---

func TestAppendUniq_AddsNewElement(t *testing.T) {
	result := appendUniq([]string{"a", "b"}, "c")
	if len(result) != 3 {
		t.Errorf("appendUniq() len = %d, want 3", len(result))
	}
	if result[2] != "c" {
		t.Errorf("appendUniq() last = %q, want %q", result[2], "c")
	}
}

func TestAppendUniq_SkipsDuplicate(t *testing.T) {
	result := appendUniq([]string{"a", "b"}, "a")
	if len(result) != 2 {
		t.Errorf("appendUniq() len = %d after duplicate, want 2", len(result))
	}
}

// --- Integration test: full adopt workflow ---

func TestAdopt_Integration_MockAgentDirs(t *testing.T) {
	storeDir, homeDir := setupAdoptEnv(t)

	// Set up mock claude config directory with various content types.
	claudeConfigDir := filepath.Join(homeDir, ".claude")
	writeAgentFile(t, claudeConfigDir, "CLAUDE.md", "# Claude Instructions\n")
	writeAgentFile(t, claudeConfigDir, "skills/my-skill.md", "# My Skill\n")
	writeAgentFile(t, claudeConfigDir, "commands/my-command.md", "# My Command\n")
	writeAgentFile(t, claudeConfigDir, "settings.json", `{"theme":"dark"}`)
	writeAgentFile(t, claudeConfigDir, "agents/my-agent.json", `{"name":"test"}`)
	writeAgentFile(t, claudeConfigDir, "rules/style.md", "# Style Rules\n")
	makeDetectedAgent(t, "claude", claudeConfigDir)

	// Remove starter AGENTS.md to allow adoption.
	if err := os.Remove(filepath.Join(storeDir, "instructions", "AGENTS.md")); err != nil {
		t.Fatalf("removing starter AGENTS.md: %v", err)
	}

	items, err := Adopt(AdoptOptions{Copy: true})
	if err != nil {
		t.Fatalf("Adopt() error = %v", err)
	}

	// Check that we got items for each content type.
	kindsSeen := make(map[string]bool)
	for _, item := range items {
		if item.Status == "imported" {
			kindsSeen[item.Kind] = true
		}
	}

	for _, kind := range []string{"instruction", "skills", "commands", "settings", "agents", "rules"} {
		if !kindsSeen[kind] {
			t.Errorf("no %q item imported, want one", kind)
		}
	}

	// Verify store has the canonical files.
	checks := []struct {
		path    string
		content string
	}{
		{"instructions/AGENTS.md", "# Claude Instructions\n"},
		{"skills/my-skill.md", "# My Skill\n"},
		{"commands/my-command.md", "# My Command\n"},
		{"settings/claude.json", `{"theme":"dark"}`},
		{"agents/my-agent.json", `{"name":"test"}`},
		{"rules/style.md", "# Style Rules\n"},
	}

	for _, c := range checks {
		data, err := os.ReadFile(filepath.Join(storeDir, c.path))
		if err != nil {
			t.Errorf("store %s: %v", c.path, err)
			continue
		}
		if string(data) != c.content {
			t.Errorf("store %s = %q, want %q", c.path, string(data), c.content)
		}
	}
}
