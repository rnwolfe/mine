package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rnwolfe/mine/internal/agents"
)

func TestRunAgentsDiff_NotInitialized(t *testing.T) {
	agentsTestEnv(t)

	out := captureStdout(t, func() {
		if err := runAgentsDiff(nil, nil); err != nil {
			t.Errorf("runAgentsDiff: %v", err)
		}
	})

	if !strings.Contains(out, "No agents store yet") {
		t.Errorf("expected 'No agents store yet' in not-initialized output, got:\n%s", out)
	}
}

func TestRunAgentsDiff_NoLinks(t *testing.T) {
	agentsTestEnv(t)

	captureStdout(t, func() {
		if err := runAgentsInit(nil, nil); err != nil {
			t.Fatalf("runAgentsInit: %v", err)
		}
	})

	out := captureStdout(t, func() {
		agentsDiffAgent = ""
		if err := runAgentsDiff(nil, nil); err != nil {
			t.Errorf("runAgentsDiff: %v", err)
		}
	})

	if !strings.Contains(out, "No links to diff") {
		t.Errorf("expected 'No links to diff' in output, got:\n%s", out)
	}
}

func TestRunAgentsDiff_LinkedSymlink_NoOutput(t *testing.T) {
	agentsTestEnv(t)

	// Init the store.
	captureStdout(t, func() {
		if err := runAgentsInit(nil, nil); err != nil {
			t.Fatalf("runAgentsInit: %v", err)
		}
	})

	storeDir := agents.Dir()

	// Write a canonical source file.
	sourcePath := filepath.Join(storeDir, "instructions", "AGENTS.md")
	if err := os.WriteFile(sourcePath, []byte("canonical content\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Create a valid symlink pointing to the canonical source.
	claudeDir := t.TempDir()
	target := filepath.Join(claudeDir, "CLAUDE.md")
	if err := os.Symlink(sourcePath, target); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	m := &agents.Manifest{
		Agents: []agents.Agent{{Name: "claude", Detected: true, ConfigDir: claudeDir}},
		Links: []agents.LinkEntry{
			{Source: "instructions/AGENTS.md", Target: target, Agent: "claude", Mode: "symlink"},
		},
	}
	if err := agents.WriteManifest(m); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}

	out := captureStdout(t, func() {
		agentsDiffAgent = ""
		if err := runAgentsDiff(nil, nil); err != nil {
			t.Errorf("runAgentsDiff: %v", err)
		}
	})

	// Symlink matches canonical — should report "linked, no diff" and success.
	if !strings.Contains(out, "linked") {
		t.Errorf("expected 'linked' in diff output for healthy symlink, got:\n%s", out)
	}
	if !strings.Contains(out, "All links match") {
		t.Errorf("expected 'All links match' in diff success summary, got:\n%s", out)
	}
}

func TestRunAgentsDiff_DivergentCopy_ShowsDiff(t *testing.T) {
	agentsTestEnv(t)

	// Init the store.
	captureStdout(t, func() {
		if err := runAgentsInit(nil, nil); err != nil {
			t.Fatalf("runAgentsInit: %v", err)
		}
	})

	storeDir := agents.Dir()

	// Write canonical source.
	sourcePath := filepath.Join(storeDir, "instructions", "AGENTS.md")
	if err := os.WriteFile(sourcePath, []byte("canonical content\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Create a copy-mode target with different content.
	claudeDir := t.TempDir()
	target := filepath.Join(claudeDir, "CLAUDE.md")
	if err := os.WriteFile(target, []byte("modified content\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	m := &agents.Manifest{
		Agents: []agents.Agent{{Name: "claude", Detected: true, ConfigDir: claudeDir}},
		Links: []agents.LinkEntry{
			{Source: "instructions/AGENTS.md", Target: target, Agent: "claude", Mode: "copy"},
		},
	}
	if err := agents.WriteManifest(m); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}

	out := captureStdout(t, func() {
		agentsDiffAgent = ""
		if err := runAgentsDiff(nil, nil); err != nil {
			t.Errorf("runAgentsDiff: %v", err)
		}
	})

	// Should report diverged state.
	if !strings.Contains(out, "diverged") {
		t.Errorf("expected 'diverged' in diff output for divergent copy, got:\n%s", out)
	}
}

func TestRunAgentsDiff_AgentFilter(t *testing.T) {
	agentsTestEnv(t)

	captureStdout(t, func() {
		if err := runAgentsInit(nil, nil); err != nil {
			t.Fatalf("runAgentsInit: %v", err)
		}
	})

	storeDir := agents.Dir()
	sourcePath := filepath.Join(storeDir, "instructions", "AGENTS.md")
	if err := os.WriteFile(sourcePath, []byte("canonical\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	claudeDir := t.TempDir()
	geminiDir := t.TempDir()

	claudeTarget := filepath.Join(claudeDir, "CLAUDE.md")
	geminiTarget := filepath.Join(geminiDir, "GEMINI.md")
	if err := os.WriteFile(claudeTarget, []byte("claude copy\n"), 0o644); err != nil {
		t.Fatalf("WriteFile claude: %v", err)
	}
	if err := os.WriteFile(geminiTarget, []byte("gemini copy\n"), 0o644); err != nil {
		t.Fatalf("WriteFile gemini: %v", err)
	}

	m := &agents.Manifest{
		Agents: []agents.Agent{
			{Name: "claude", Detected: true, ConfigDir: claudeDir},
			{Name: "gemini", Detected: true, ConfigDir: geminiDir},
		},
		Links: []agents.LinkEntry{
			{Source: "instructions/AGENTS.md", Target: claudeTarget, Agent: "claude", Mode: "copy"},
			{Source: "instructions/AGENTS.md", Target: geminiTarget, Agent: "gemini", Mode: "copy"},
		},
	}
	if err := agents.WriteManifest(m); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}

	out := captureStdout(t, func() {
		agentsDiffAgent = "claude"
		defer func() { agentsDiffAgent = "" }()
		if err := runAgentsDiff(nil, nil); err != nil {
			t.Errorf("runAgentsDiff --agent claude: %v", err)
		}
	})

	// Should show claude's target but not gemini's.
	if !strings.Contains(out, claudeTarget) {
		t.Errorf("expected claude target in filtered diff output, got:\n%s", out)
	}
	if strings.Contains(out, geminiTarget) {
		t.Errorf("expected gemini target to be absent from filtered diff output, got:\n%s", out)
	}
}
