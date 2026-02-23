package cmd

import (
	"strings"
	"testing"

	"github.com/rnwolfe/mine/internal/agents"
)

// agentsTestEnv sets up a temp XDG environment for agents cmd tests.
func agentsTestEnv(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir+"/config")
	t.Setenv("XDG_DATA_HOME", tmpDir+"/data")
	t.Setenv("XDG_CACHE_HOME", tmpDir+"/cache")
	t.Setenv("XDG_STATE_HOME", tmpDir+"/state")
}

func TestRunAgentsInit_PrintsLocationAndTip(t *testing.T) {
	agentsTestEnv(t)

	out := captureStdout(t, func() {
		if err := runAgentsInit(nil, nil); err != nil {
			t.Errorf("runAgentsInit: %v", err)
		}
	})

	dir := agents.Dir()
	if !strings.Contains(out, dir) {
		t.Errorf("expected store location %q in output, got:\n%s", dir, out)
	}
	if !strings.Contains(out, "instructions/AGENTS.md") {
		t.Errorf("expected 'instructions/AGENTS.md' tip in output, got:\n%s", out)
	}
}

func TestRunAgentsInit_CreatesStore(t *testing.T) {
	agentsTestEnv(t)

	captureStdout(t, func() {
		if err := runAgentsInit(nil, nil); err != nil {
			t.Fatalf("runAgentsInit: %v", err)
		}
	})

	if !agents.IsInitialized() {
		t.Error("expected agents store to be initialized after runAgentsInit")
	}
}

func TestRunAgentsInit_Idempotent(t *testing.T) {
	agentsTestEnv(t)

	captureStdout(t, func() {
		if err := runAgentsInit(nil, nil); err != nil {
			t.Fatalf("first runAgentsInit: %v", err)
		}
	})
	captureStdout(t, func() {
		if err := runAgentsInit(nil, nil); err != nil {
			t.Errorf("second runAgentsInit (idempotency): %v", err)
		}
	})
}

func TestRunAgentsStatus_NotInitialized(t *testing.T) {
	agentsTestEnv(t)

	out := captureStdout(t, func() {
		if err := runAgentsStatus(nil, nil); err != nil {
			t.Errorf("runAgentsStatus: %v", err)
		}
	})

	if !strings.Contains(out, "No agents store yet") {
		t.Errorf("expected 'No agents store yet' in not-initialized output, got:\n%s", out)
	}
	if !strings.Contains(out, "mine agents init") {
		t.Errorf("expected 'mine agents init' hint in not-initialized output, got:\n%s", out)
	}
}

func TestRunAgentsStatus_Initialized_Empty(t *testing.T) {
	agentsTestEnv(t)

	captureStdout(t, func() {
		if err := runAgentsInit(nil, nil); err != nil {
			t.Fatalf("runAgentsInit: %v", err)
		}
	})

	out := captureStdout(t, func() {
		if err := runAgentsStatus(nil, nil); err != nil {
			t.Errorf("runAgentsStatus: %v", err)
		}
	})

	dir := agents.Dir()
	if !strings.Contains(out, dir) {
		t.Errorf("expected store dir %q in status output, got:\n%s", dir, out)
	}
	if !strings.Contains(out, "No agents registered yet") {
		t.Errorf("expected 'No agents registered yet' in empty status output, got:\n%s", out)
	}
	if !strings.Contains(out, "No links configured yet") {
		t.Errorf("expected 'No links configured yet' in empty status output, got:\n%s", out)
	}
}

func TestRunAgentsStatus_Initialized_WithAgents(t *testing.T) {
	agentsTestEnv(t)

	captureStdout(t, func() {
		if err := runAgentsInit(nil, nil); err != nil {
			t.Fatalf("runAgentsInit: %v", err)
		}
	})

	m := &agents.Manifest{
		Agents: []agents.Agent{
			{Name: "claude", Detected: true, ConfigDir: "/home/user/.claude", Binary: "claude"},
			{Name: "gemini", Detected: false, ConfigDir: "/home/user/.gemini", Binary: "gemini"},
		},
		Links: []agents.LinkEntry{
			{Source: "instructions/AGENTS.md", Target: "/home/user/.claude/CLAUDE.md", Agent: "claude", Mode: "symlink"},
		},
	}
	if err := agents.WriteManifest(m); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}

	out := captureStdout(t, func() {
		if err := runAgentsStatus(nil, nil); err != nil {
			t.Errorf("runAgentsStatus: %v", err)
		}
	})

	if !strings.Contains(out, "2 registered") {
		t.Errorf("expected '2 registered' in status output, got:\n%s", out)
	}
	if !strings.Contains(out, "1 detected") {
		t.Errorf("expected '1 detected' in status output, got:\n%s", out)
	}
	if !strings.Contains(out, "1 active") {
		t.Errorf("expected '1 active' in links status output, got:\n%s", out)
	}
}
