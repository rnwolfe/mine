package agents

import (
	"os"
	"path/filepath"
	"testing"
)

// --- ProjectLink ---

func TestProjectLink_NotInitialized(t *testing.T) {
	setupEnv(t)
	_, err := ProjectLink(t.TempDir(), ProjectLinkOptions{})
	if err == nil {
		t.Error("ProjectLink() error = nil for uninitialized store, want error")
	}
}

func TestProjectLink_CreatesSkillsSymlink(t *testing.T) {
	storeDir, projectDir := setupProjectEnv(t)
	addDetectedAgent(t, "claude")

	// Add a skill to the canonical store.
	writeStoreFile(t, storeDir, "skills/my-skill.md", "# My Skill\n")

	actions, err := ProjectLink(projectDir, ProjectLinkOptions{})
	if err != nil {
		t.Fatalf("ProjectLink() error = %v", err)
	}
	if len(actions) == 0 {
		t.Fatal("ProjectLink() returned no actions")
	}

	// Find the skills link action.
	var skillsAction *LinkAction
	for i := range actions {
		if actions[i].Source == "skills" && actions[i].Agent == "claude" {
			skillsAction = &actions[i]
			break
		}
	}
	if skillsAction == nil {
		t.Fatal("no skills action for claude")
	}
	if skillsAction.Err != nil {
		t.Errorf("skills action.Err = %v, want nil", skillsAction.Err)
	}

	// Verify symlink exists in project's .claude/skills.
	target := filepath.Join(projectDir, ".claude", "skills")
	info, err := os.Lstat(target)
	if err != nil {
		t.Fatalf(".claude/skills not created: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error(".claude/skills is not a symlink, want symlink")
	}

	dest, _ := os.Readlink(target)
	want := filepath.Join(storeDir, "skills")
	if dest != want {
		t.Errorf("skills symlink points to %q, want %q", dest, want)
	}
}

func TestProjectLink_CopyMode(t *testing.T) {
	storeDir, projectDir := setupProjectEnv(t)
	addDetectedAgent(t, "claude")

	writeStoreFile(t, storeDir, "skills/my-skill.md", "# My Skill\n")

	actions, err := ProjectLink(projectDir, ProjectLinkOptions{Copy: true})
	if err != nil {
		t.Fatalf("ProjectLink() error = %v", err)
	}

	var skillsAction *LinkAction
	for i := range actions {
		if actions[i].Source == "skills" && actions[i].Agent == "claude" {
			skillsAction = &actions[i]
			break
		}
	}
	if skillsAction == nil {
		t.Fatal("no skills action for claude in copy mode")
	}
	if skillsAction.Mode != "copy" {
		t.Errorf("action.Mode = %q, want %q", skillsAction.Mode, "copy")
	}

	// Must be a real directory, not a symlink.
	target := filepath.Join(projectDir, ".claude", "skills")
	info, err := os.Lstat(target)
	if err != nil {
		t.Fatalf(".claude/skills not created in copy mode: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error(".claude/skills is a symlink in copy mode, want real directory")
	}
}

func TestProjectLink_AgentFilter(t *testing.T) {
	storeDir, projectDir := setupProjectEnv(t)
	addDetectedAgent(t, "claude")
	addDetectedAgent(t, "codex")

	writeStoreFile(t, storeDir, "skills/my-skill.md", "# My Skill\n")

	actions, err := ProjectLink(projectDir, ProjectLinkOptions{Agent: "claude"})
	if err != nil {
		t.Fatalf("ProjectLink() error = %v", err)
	}

	for _, a := range actions {
		if a.Agent != "claude" {
			t.Errorf("action for agent %q found, want only claude", a.Agent)
		}
	}
}

func TestProjectLink_EmptyStoreSkills_NoSkillsActions(t *testing.T) {
	_, projectDir := setupProjectEnv(t)
	addDetectedAgent(t, "claude")
	// Store skills/ is empty (default after Init).

	actions, err := ProjectLink(projectDir, ProjectLinkOptions{})
	if err != nil {
		t.Fatalf("ProjectLink() error = %v", err)
	}

	// No skills actions (store is empty).
	for _, a := range actions {
		if a.Source == "skills" {
			t.Errorf("skills action found for empty store, want none")
		}
	}
}

func TestProjectLink_DefaultsToCWD(t *testing.T) {
	storeDir, projectDir := setupProjectEnv(t)
	addDetectedAgent(t, "claude")

	writeStoreFile(t, storeDir, "skills/my-skill.md", "# My Skill\n")

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(projectDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldDir) })

	actions, err := ProjectLink("", ProjectLinkOptions{})
	if err != nil {
		t.Fatalf("ProjectLink with empty path: %v", err)
	}
	if len(actions) == 0 {
		t.Fatal("ProjectLink returned no actions for CWD")
	}
}

func TestProjectLink_InvalidPath(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	_, err := ProjectLink(filepath.Join(t.TempDir(), "nonexistent"), ProjectLinkOptions{})
	if err == nil {
		t.Error("ProjectLink() error = nil for nonexistent path, want error")
	}
}

func TestProjectLink_PersistsToManifest(t *testing.T) {
	storeDir, projectDir := setupProjectEnv(t)
	addDetectedAgent(t, "claude")

	writeStoreFile(t, storeDir, "skills/my-skill.md", "# My Skill\n")

	if _, err := ProjectLink(projectDir, ProjectLinkOptions{}); err != nil {
		t.Fatalf("ProjectLink: %v", err)
	}

	m, err := ReadManifest()
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}

	var found bool
	for _, l := range m.Links {
		if l.Agent == "claude" && l.Source == "skills" {
			found = true
			break
		}
	}
	if !found {
		t.Error("project link not persisted to manifest")
	}
}

// --- Integration: init → link cycle ---

func TestProjectInitLink_Integration(t *testing.T) {
	storeDir, projectDir := setupProjectEnv(t)
	addDetectedAgent(t, "claude")
	addDetectedAgent(t, "codex")

	// Populate canonical store.
	writeStoreFile(t, storeDir, "skills/shared-skill.md", "# Shared Skill\n")
	settingsContent := `{"theme": "dark"}`
	if err := os.WriteFile(filepath.Join(storeDir, "settings", "claude.json"), []byte(settingsContent), 0o644); err != nil {
		t.Fatalf("writing settings: %v", err)
	}

	// Step 1: Init project.
	initActions, err := ProjectInit(projectDir, ProjectInitOptions{})
	if err != nil {
		t.Fatalf("ProjectInit: %v", err)
	}
	if len(initActions) == 0 {
		t.Fatal("ProjectInit returned no actions")
	}

	// Verify directory structure.
	expectedDirs := []string{
		".claude", ".claude/skills", ".claude/commands",
		".agents", ".agents/skills",
	}
	for _, d := range expectedDirs {
		if _, err := os.Stat(filepath.Join(projectDir, d)); err != nil {
			t.Errorf("expected dir %q missing after init: %v", d, err)
		}
	}

	// Instruction files must be created.
	for _, f := range []string{"CLAUDE.md", "AGENTS.md"} {
		if _, err := os.ReadFile(filepath.Join(projectDir, f)); err != nil {
			t.Errorf("%s not created", f)
		}
	}

	// Settings seeded for claude.
	settingsDst := filepath.Join(projectDir, ".claude", "settings.json")
	if data, err := os.ReadFile(settingsDst); err != nil {
		t.Error("claude settings.json not seeded")
	} else if string(data) != settingsContent {
		t.Errorf("settings.json content = %q, want %q", string(data), settingsContent)
	}

	// Step 2: Link canonical skills.
	// --force is required because project init already created the skills dirs;
	// project link replaces them with symlinks to the canonical store.
	linkActions, err := ProjectLink(projectDir, ProjectLinkOptions{Force: true})
	if err != nil {
		t.Fatalf("ProjectLink: %v", err)
	}
	if len(linkActions) == 0 {
		t.Fatal("ProjectLink returned no actions")
	}

	// Skills symlinks must be created.
	for _, target := range []string{
		filepath.Join(projectDir, ".claude", "skills"),
		filepath.Join(projectDir, ".agents", "skills"),
	} {
		info, err := os.Lstat(target)
		if err != nil {
			t.Errorf("skills link %q not created: %v", target, err)
			continue
		}
		if info.Mode()&os.ModeSymlink == 0 {
			t.Errorf("%q is not a symlink after project link", target)
		}
	}
}
