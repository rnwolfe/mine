package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rnwolfe/mine/internal/agents"
)

func TestRunAgentsAdopt_NotInitialized(t *testing.T) {
	agentsTestEnv(t)
	// Do NOT run init.

	out := captureStdout(t, func() {
		agentsAdoptAgent = ""
		agentsAdoptDryRun = false
		agentsAdoptCopy = false
		if err := runAgentsAdopt(nil, nil); err != nil {
			t.Errorf("runAgentsAdopt: %v", err)
		}
	})

	if !strings.Contains(out, "No agents store yet") {
		t.Errorf("expected 'No agents store yet' in output, got:\n%s", out)
	}
}

func TestRunAgentsAdopt_NothingToAdopt(t *testing.T) {
	setupAgentsAdoptEnv(t)

	out := captureStdout(t, func() {
		agentsAdoptAgent = ""
		agentsAdoptDryRun = false
		agentsAdoptCopy = false
		if err := runAgentsAdopt(nil, nil); err != nil {
			t.Errorf("runAgentsAdopt: %v", err)
		}
	})

	if !strings.Contains(out, "Nothing to adopt") {
		t.Errorf("expected 'Nothing to adopt' in output, got:\n%s", out)
	}
}

func TestRunAgentsAdopt_ImportsInstructionFile(t *testing.T) {
	storeDir, homeDir := setupAgentsAdoptEnv(t)

	claudeConfigDir := filepath.Join(homeDir, ".claude")
	if err := os.MkdirAll(claudeConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}
	claudeFile := filepath.Join(claudeConfigDir, "CLAUDE.md")
	if err := os.WriteFile(claudeFile, []byte("# My Instructions\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Register claude as detected.
	m, _ := agents.ReadManifest()
	m.Agents = []agents.Agent{
		{Name: "claude", Detected: true, ConfigDir: claudeConfigDir},
	}
	if err := agents.WriteManifest(m); err != nil {
		t.Fatal(err)
	}

	// Remove starter AGENTS.md to allow adoption.
	if err := os.Remove(filepath.Join(storeDir, "instructions", "AGENTS.md")); err != nil {
		t.Fatalf("removing starter AGENTS.md: %v", err)
	}

	out := captureStdout(t, func() {
		agentsAdoptAgent = ""
		agentsAdoptDryRun = false
		agentsAdoptCopy = true // --copy to skip symlink creation
		defer func() { agentsAdoptCopy = false }()
		if err := runAgentsAdopt(nil, nil); err != nil {
			t.Errorf("runAgentsAdopt: %v", err)
		}
	})

	if !strings.Contains(out, "imported") {
		t.Errorf("expected 'imported' in output, got:\n%s", out)
	}

	// Verify instruction was copied to store.
	data, err := os.ReadFile(filepath.Join(storeDir, "instructions", "AGENTS.md"))
	if err != nil {
		t.Fatalf("reading store instructions: %v", err)
	}
	if !strings.Contains(string(data), "My Instructions") {
		t.Errorf("store instructions content = %q, want '# My Instructions'", string(data))
	}
}

func TestRunAgentsAdopt_DryRun_NoChanges(t *testing.T) {
	storeDir, homeDir := setupAgentsAdoptEnv(t)

	claudeConfigDir := filepath.Join(homeDir, ".claude")
	if err := os.MkdirAll(claudeConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeConfigDir, "CLAUDE.md"), []byte("# Instructions\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m, _ := agents.ReadManifest()
	m.Agents = []agents.Agent{{Name: "claude", Detected: true, ConfigDir: claudeConfigDir}}
	if err := agents.WriteManifest(m); err != nil {
		t.Fatal(err)
	}

	starterFile := filepath.Join(storeDir, "instructions", "AGENTS.md")
	if err := os.Remove(starterFile); err != nil {
		t.Fatalf("removing starter AGENTS.md: %v", err)
	}

	captureStdout(t, func() {
		agentsAdoptAgent = ""
		agentsAdoptDryRun = true
		agentsAdoptCopy = false
		defer func() { agentsAdoptDryRun = false }()
		if err := runAgentsAdopt(nil, nil); err != nil {
			t.Errorf("runAgentsAdopt --dry-run: %v", err)
		}
	})

	// Store file should NOT have been created.
	if _, err := os.Stat(starterFile); err == nil {
		t.Error("store instructions created during dry run, want no changes")
	}
}

func TestRunAgentsAdopt_ConflictReported(t *testing.T) {
	storeDir, homeDir := setupAgentsAdoptEnv(t)

	claudeConfigDir := filepath.Join(homeDir, ".claude")
	if err := os.MkdirAll(claudeConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeConfigDir, "CLAUDE.md"), []byte("# Different Instructions\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Store already has different instructions.
	if err := os.WriteFile(filepath.Join(storeDir, "instructions", "AGENTS.md"), []byte("# Store Instructions\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m, _ := agents.ReadManifest()
	m.Agents = []agents.Agent{{Name: "claude", Detected: true, ConfigDir: claudeConfigDir}}
	if err := agents.WriteManifest(m); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() {
		agentsAdoptAgent = ""
		agentsAdoptDryRun = false
		agentsAdoptCopy = false
		if err := runAgentsAdopt(nil, nil); err != nil {
			t.Errorf("runAgentsAdopt: %v", err)
		}
	})

	if !strings.Contains(out, "conflict") {
		t.Errorf("expected 'conflict' in output for differing content, got:\n%s", out)
	}
}

func TestRunAgentsAdopt_AgentFilter(t *testing.T) {
	storeDir, homeDir := setupAgentsAdoptEnv(t)

	claudeConfigDir := filepath.Join(homeDir, ".claude")
	codexConfigDir := filepath.Join(homeDir, ".codex")

	for _, dir := range []string{claudeConfigDir, codexConfigDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(claudeConfigDir, "CLAUDE.md"), []byte("# Claude\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(codexConfigDir, "AGENTS.md"), []byte("# Codex\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m, _ := agents.ReadManifest()
	m.Agents = []agents.Agent{
		{Name: "claude", Detected: true, ConfigDir: claudeConfigDir},
		{Name: "codex", Detected: true, ConfigDir: codexConfigDir},
	}
	if err := agents.WriteManifest(m); err != nil {
		t.Fatal(err)
	}

	if err := os.Remove(filepath.Join(storeDir, "instructions", "AGENTS.md")); err != nil {
		t.Fatalf("removing starter AGENTS.md: %v", err)
	}

	captureStdout(t, func() {
		agentsAdoptAgent = "claude"
		agentsAdoptDryRun = true
		agentsAdoptCopy = false
		defer func() { agentsAdoptAgent = ""; agentsAdoptDryRun = false }()
		if err := runAgentsAdopt(nil, nil); err != nil {
			t.Errorf("runAgentsAdopt --agent claude: %v", err)
		}
	})

	items, err := agents.Adopt(agents.AdoptOptions{Agent: "claude", DryRun: true})
	if err != nil {
		t.Fatalf("Adopt DryRun: %v", err)
	}
	for _, item := range items {
		if item.Agent != "claude" {
			t.Errorf("item for agent %q found with --agent claude, want only claude", item.Agent)
		}
	}
}

func TestRunAgentsAdopt_IntegrationWithSymlinks(t *testing.T) {
	storeDir, homeDir := setupAgentsAdoptEnv(t)

	claudeConfigDir := filepath.Join(homeDir, ".claude")
	if err := os.MkdirAll(claudeConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeConfigDir, "CLAUDE.md"), []byte("# Instructions\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m, _ := agents.ReadManifest()
	m.Agents = []agents.Agent{{Name: "claude", Detected: true, ConfigDir: claudeConfigDir}}
	if err := agents.WriteManifest(m); err != nil {
		t.Fatal(err)
	}

	if err := os.Remove(filepath.Join(storeDir, "instructions", "AGENTS.md")); err != nil {
		t.Fatalf("removing starter AGENTS.md: %v", err)
	}

	// Adopt without --copy; should create symlinks.
	captureStdout(t, func() {
		agentsAdoptAgent = ""
		agentsAdoptDryRun = false
		agentsAdoptCopy = false
		if err := runAgentsAdopt(nil, nil); err != nil {
			t.Errorf("runAgentsAdopt: %v", err)
		}
	})

	// CLAUDE.md should now be a symlink.
	target := filepath.Join(claudeConfigDir, "CLAUDE.md")
	info, err := os.Lstat(target)
	if err != nil {
		t.Fatalf("Lstat CLAUDE.md after adopt: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("CLAUDE.md is not a symlink after adopt, want symlink to store")
	}
}
