package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rnwolfe/mine/internal/agents"
)

// setupAgentsProjectEnv initializes a store with a detected agent and creates a
// project directory. Returns (storeDir, projectDir).
func setupAgentsProjectEnv(t *testing.T) (string, string) {
	t.Helper()
	agentsTestEnv(t)

	captureStdout(t, func() {
		if err := runAgentsInit(nil, nil); err != nil {
			t.Fatalf("runAgentsInit: %v", err)
		}
	})

	home := os.Getenv("HOME")
	projectDir := filepath.Join(home, "myproject")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatalf("creating project dir: %v", err)
	}

	// Register claude as detected.
	m, err := agents.ReadManifest()
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	m.Agents = []agents.Agent{
		{Name: "claude", Detected: true, ConfigDir: filepath.Join(home, ".claude")},
	}
	if err := agents.WriteManifest(m); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}

	return agents.Dir(), projectDir
}

func TestRunAgentsProjectInit_CreatesStructure(t *testing.T) {
	_, projectDir := setupAgentsProjectEnv(t)
	agentsProjectInitForce = false
	t.Cleanup(func() { agentsProjectInitForce = false })

	out := captureStdout(t, func() {
		if err := runAgentsProjectInit(nil, []string{projectDir}); err != nil {
			t.Errorf("runAgentsProjectInit: %v", err)
		}
	})

	if !strings.Contains(out, "created") {
		t.Errorf("expected 'created' in output, got:\n%s", out)
	}

	if _, err := os.Stat(filepath.Join(projectDir, ".claude")); err != nil {
		t.Error(".claude/ not created after project init")
	}
}

func TestRunAgentsProjectInit_Idempotent(t *testing.T) {
	_, projectDir := setupAgentsProjectEnv(t)
	agentsProjectInitForce = false
	t.Cleanup(func() { agentsProjectInitForce = false })

	captureStdout(t, func() {
		if err := runAgentsProjectInit(nil, []string{projectDir}); err != nil {
			t.Fatalf("first init: %v", err)
		}
	})

	// Second run must succeed without error.
	captureStdout(t, func() {
		if err := runAgentsProjectInit(nil, []string{projectDir}); err != nil {
			t.Errorf("second init (idempotency): %v", err)
		}
	})
}

func TestRunAgentsProjectInit_NoDetectedAgents(t *testing.T) {
	agentsTestEnv(t)

	captureStdout(t, func() {
		if err := runAgentsInit(nil, nil); err != nil {
			t.Fatalf("runAgentsInit: %v", err)
		}
	})

	agentsProjectInitForce = false
	t.Cleanup(func() { agentsProjectInitForce = false })

	out := captureStdout(t, func() {
		if err := runAgentsProjectInit(nil, []string{t.TempDir()}); err != nil {
			t.Errorf("runAgentsProjectInit: %v", err)
		}
	})

	if !strings.Contains(out, "No agents detected") {
		t.Errorf("expected 'No agents detected' in output, got:\n%s", out)
	}
}

func TestRunAgentsProjectInit_SeedsSettings(t *testing.T) {
	storeDir, projectDir := setupAgentsProjectEnv(t)

	// Add settings template to store.
	settingsFile := filepath.Join(storeDir, "settings", "claude.json")
	if err := os.WriteFile(settingsFile, []byte(`{"model":"claude"}`), 0o644); err != nil {
		t.Fatalf("writing settings: %v", err)
	}

	agentsProjectInitForce = false
	t.Cleanup(func() { agentsProjectInitForce = false })

	captureStdout(t, func() {
		if err := runAgentsProjectInit(nil, []string{projectDir}); err != nil {
			t.Fatalf("runAgentsProjectInit: %v", err)
		}
	})

	// settings.json must be seeded in .claude/.
	if _, err := os.Stat(filepath.Join(projectDir, ".claude", "settings.json")); err != nil {
		t.Error("settings.json not seeded in .claude/")
	}

	// CLAUDE.md must exist at project root.
	if _, err := os.Stat(filepath.Join(projectDir, "CLAUDE.md")); err != nil {
		t.Error("CLAUDE.md not created at project root")
	}
}

func TestRunAgentsProjectLink_RequiresAgentsInit(t *testing.T) {
	agentsTestEnv(t)
	// Don't call runAgentsInit — store is not initialized.

	agentsProjectLinkCopy = false
	agentsProjectLinkForce = false
	agentsProjectLinkAgent = ""
	t.Cleanup(func() {
		agentsProjectLinkCopy = false
		agentsProjectLinkForce = false
		agentsProjectLinkAgent = ""
	})

	out := captureStdout(t, func() {
		if err := runAgentsProjectLink(nil, []string{t.TempDir()}); err != nil {
			t.Errorf("runAgentsProjectLink: %v", err)
		}
	})

	if !strings.Contains(out, "mine agents init") {
		t.Errorf("expected 'mine agents init' suggestion in output, got:\n%s", out)
	}
}

func TestRunAgentsProjectLink_LinksSkills(t *testing.T) {
	storeDir, projectDir := setupAgentsProjectEnv(t)

	// Add a skill to the canonical store.
	skillsFile := filepath.Join(storeDir, "skills", "my-skill.md")
	if err := os.WriteFile(skillsFile, []byte("# My Skill\n"), 0o644); err != nil {
		t.Fatalf("writing skill: %v", err)
	}

	agentsProjectLinkCopy = false
	agentsProjectLinkForce = false
	agentsProjectLinkAgent = ""
	t.Cleanup(func() {
		agentsProjectLinkCopy = false
		agentsProjectLinkForce = false
		agentsProjectLinkAgent = ""
	})

	out := captureStdout(t, func() {
		if err := runAgentsProjectLink(nil, []string{projectDir}); err != nil {
			t.Errorf("runAgentsProjectLink: %v", err)
		}
	})

	if !strings.Contains(out, "link(s) configured") {
		t.Errorf("expected success message in output, got:\n%s", out)
	}

	// .claude/skills must be a symlink in the project.
	claudeSkills := filepath.Join(projectDir, ".claude", "skills")
	info, err := os.Lstat(claudeSkills)
	if err != nil {
		t.Fatalf(".claude/skills not created: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error(".claude/skills is not a symlink after project link")
	}
}

func TestRunAgentsProjectLink_CopyMode(t *testing.T) {
	storeDir, projectDir := setupAgentsProjectEnv(t)

	skillsFile := filepath.Join(storeDir, "skills", "my-skill.md")
	if err := os.WriteFile(skillsFile, []byte("# My Skill\n"), 0o644); err != nil {
		t.Fatalf("writing skill: %v", err)
	}

	agentsProjectLinkCopy = true
	agentsProjectLinkForce = false
	agentsProjectLinkAgent = ""
	t.Cleanup(func() {
		agentsProjectLinkCopy = false
		agentsProjectLinkForce = false
		agentsProjectLinkAgent = ""
	})

	captureStdout(t, func() {
		if err := runAgentsProjectLink(nil, []string{projectDir}); err != nil {
			t.Errorf("runAgentsProjectLink: %v", err)
		}
	})

	// .claude/skills must be a real directory (not a symlink) in copy mode.
	claudeSkills := filepath.Join(projectDir, ".claude", "skills")
	info, err := os.Lstat(claudeSkills)
	if err != nil {
		t.Fatalf(".claude/skills not created in copy mode: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error(".claude/skills is a symlink in copy mode, want real directory")
	}
}

func TestRunAgentsProjectLink_EmptyStore(t *testing.T) {
	_, projectDir := setupAgentsProjectEnv(t)
	// Store skills/ is empty (default after init).

	agentsProjectLinkCopy = false
	agentsProjectLinkForce = false
	agentsProjectLinkAgent = ""
	t.Cleanup(func() {
		agentsProjectLinkCopy = false
		agentsProjectLinkForce = false
		agentsProjectLinkAgent = ""
	})

	out := captureStdout(t, func() {
		if err := runAgentsProjectLink(nil, []string{projectDir}); err != nil {
			t.Errorf("runAgentsProjectLink: %v", err)
		}
	})

	if !strings.Contains(out, "Nothing to link") {
		t.Errorf("expected 'Nothing to link' in output for empty store, got:\n%s", out)
	}
}

// TestRunAgentsProject_FullCycle tests the complete init → link workflow.
func TestRunAgentsProject_FullCycle(t *testing.T) {
	storeDir, projectDir := setupAgentsProjectEnv(t)

	// Populate canonical store.
	skillsFile := filepath.Join(storeDir, "skills", "shared-skill.md")
	if err := os.WriteFile(skillsFile, []byte("# Shared Skill\n"), 0o644); err != nil {
		t.Fatalf("writing skill: %v", err)
	}
	settingsFile := filepath.Join(storeDir, "settings", "claude.json")
	if err := os.WriteFile(settingsFile, []byte(`{"theme":"dark"}`), 0o644); err != nil {
		t.Fatalf("writing settings: %v", err)
	}

	// Step 1: Init project.
	agentsProjectInitForce = false
	t.Cleanup(func() { agentsProjectInitForce = false })
	captureStdout(t, func() {
		if err := runAgentsProjectInit(nil, []string{projectDir}); err != nil {
			t.Fatalf("runAgentsProjectInit: %v", err)
		}
	})

	// Verify init output.
	for _, expected := range []string{
		filepath.Join(projectDir, ".claude"),
		filepath.Join(projectDir, ".claude", "skills"),
		filepath.Join(projectDir, "CLAUDE.md"),
	} {
		if _, err := os.Stat(expected); err != nil {
			t.Errorf("expected path %q missing after init: %v", expected, err)
		}
	}

	// Step 2: Link skills.
	// --force is needed because project init created the skills dirs as real dirs.
	agentsProjectLinkCopy = false
	agentsProjectLinkForce = true
	agentsProjectLinkAgent = ""
	t.Cleanup(func() {
		agentsProjectLinkCopy = false
		agentsProjectLinkForce = false
		agentsProjectLinkAgent = ""
	})
	captureStdout(t, func() {
		if err := runAgentsProjectLink(nil, []string{projectDir}); err != nil {
			t.Fatalf("runAgentsProjectLink: %v", err)
		}
	})

	// .claude/skills must be a symlink.
	info, err := os.Lstat(filepath.Join(projectDir, ".claude", "skills"))
	if err != nil {
		t.Fatal(".claude/skills missing after project link")
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error(".claude/skills is not a symlink after full cycle")
	}
}
