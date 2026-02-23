package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/rnwolfe/mine/internal/agents"
)

// agentsTestEnv sets up a temp XDG + HOME environment for agents cmd tests.
func agentsTestEnv(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", tmpDir+"/config")
	t.Setenv("XDG_DATA_HOME", tmpDir+"/data")
	t.Setenv("XDG_CACHE_HOME", tmpDir+"/cache")
	t.Setenv("XDG_STATE_HOME", tmpDir+"/state")
}

// makeFakeBinaryCmd creates a minimal executable in dir named name.
func makeFakeBinaryCmd(t *testing.T, dir, name string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake binary tests not supported on Windows")
	}
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte("#!/bin/sh\necho fake\n"), 0o755); err != nil {
		t.Fatalf("creating fake binary %s: %v", name, err)
	}
	return p
}

// --- mine agents init ---

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

// --- mine agents (status) ---

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
			{Name: "claude", Detected: true, ConfigDir: "/home/user/.claude", Binary: "/usr/local/bin/claude"},
			{Name: "gemini", Detected: false, ConfigDir: "/home/user/.gemini", Binary: ""},
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

// --- mine agents detect ---

func TestRunAgentsDetect_NoAgentsFound(t *testing.T) {
	agentsTestEnv(t)
	// Prepend an empty dir to PATH so system tools (git) remain accessible,
	// but no agent binaries are present in that dir. HOME is a fresh temp
	// dir with no agent config directories, so no agents will be detected.
	binDir := t.TempDir()
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	out := captureStdout(t, func() {
		if err := runAgentsDetect(nil, nil); err != nil {
			t.Errorf("runAgentsDetect: %v", err)
		}
	})

	if !strings.Contains(out, "Manifest updated") {
		t.Errorf("expected 'Manifest updated' in output, got:\n%s", out)
	}
}

func TestRunAgentsDetect_AgentBinaryFound(t *testing.T) {
	agentsTestEnv(t)
	binDir := t.TempDir()
	makeFakeBinaryCmd(t, binDir, "claude")
	// Prepend binDir so fake claude is found first; system tools stay accessible.
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	out := captureStdout(t, func() {
		if err := runAgentsDetect(nil, nil); err != nil {
			t.Errorf("runAgentsDetect: %v", err)
		}
	})

	if !strings.Contains(out, "claude") {
		t.Errorf("expected 'claude' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "detected") {
		t.Errorf("expected 'detected' in output, got:\n%s", out)
	}
}

func TestRunAgentsDetect_PersistsToManifest(t *testing.T) {
	agentsTestEnv(t)
	binDir := t.TempDir()
	makeFakeBinaryCmd(t, binDir, "claude")
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	captureStdout(t, func() {
		if err := runAgentsDetect(nil, nil); err != nil {
			t.Fatalf("runAgentsDetect: %v", err)
		}
	})

	m, err := agents.ReadManifest()
	if err != nil {
		t.Fatalf("ReadManifest after detect: %v", err)
	}
	if len(m.Agents) == 0 {
		t.Error("manifest.Agents is empty after detect, want agents persisted")
	}

	var claudeFound bool
	for _, a := range m.Agents {
		if a.Name == "claude" && a.Detected {
			claudeFound = true
			break
		}
	}
	if !claudeFound {
		t.Error("claude not found as detected in manifest after runAgentsDetect")
	}
}

func TestRunAgentsDetect_Idempotent(t *testing.T) {
	agentsTestEnv(t)
	binDir := t.TempDir()
	makeFakeBinaryCmd(t, binDir, "claude")
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	captureStdout(t, func() {
		if err := runAgentsDetect(nil, nil); err != nil {
			t.Fatalf("first runAgentsDetect: %v", err)
		}
	})
	captureStdout(t, func() {
		if err := runAgentsDetect(nil, nil); err != nil {
			t.Errorf("second runAgentsDetect (idempotency): %v", err)
		}
	})

	// Manifest should still have exactly 4 agents (one per registry entry), not duplicates.
	m, err := agents.ReadManifest()
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if len(m.Agents) != 4 {
		t.Errorf("manifest.Agents count = %d after two detects, want 4 (no duplicates)", len(m.Agents))
	}
}

func TestRunAgentsDetect_InitializesStoreIfNeeded(t *testing.T) {
	agentsTestEnv(t)
	// Prepend empty binDir; git remains accessible via system PATH.
	binDir := t.TempDir()
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	// Do not call runAgentsInit first — detect should auto-init.
	if agents.IsInitialized() {
		t.Fatal("store should not be initialized at start of test")
	}

	captureStdout(t, func() {
		if err := runAgentsDetect(nil, nil); err != nil {
			t.Fatalf("runAgentsDetect: %v", err)
		}
	})

	if !agents.IsInitialized() {
		t.Error("agents store not initialized after runAgentsDetect, expected auto-init")
	}
}

func TestRunAgentsDetect_ShowsAllFourAgents(t *testing.T) {
	agentsTestEnv(t)
	binDir := t.TempDir()
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	out := captureStdout(t, func() {
		if err := runAgentsDetect(nil, nil); err != nil {
			t.Fatalf("runAgentsDetect: %v", err)
		}
	})

	for _, name := range []string{"claude", "codex", "gemini", "opencode"} {
		if !strings.Contains(out, name) {
			t.Errorf("expected agent %q in detect output, got:\n%s", name, out)
		}
	}
}

func TestRunAgentsDetect_ConfigDirDetected(t *testing.T) {
	agentsTestEnv(t)
	// Prepend empty binDir; git remains accessible.
	binDir := t.TempDir()
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	// Create ~/.gemini config dir to trigger config-dir-based detection.
	home := os.Getenv("HOME")
	geminiDir := filepath.Join(home, ".gemini")
	if err := os.MkdirAll(geminiDir, 0o755); err != nil {
		t.Fatal(err)
	}

	captureStdout(t, func() {
		if err := runAgentsDetect(nil, nil); err != nil {
			t.Fatalf("runAgentsDetect: %v", err)
		}
	})

	m, err := agents.ReadManifest()
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	var geminiDetected bool
	for _, a := range m.Agents {
		if a.Name == "gemini" && a.Detected {
			geminiDetected = true
			break
		}
	}
	if !geminiDetected {
		t.Error("gemini not detected in manifest via config dir, expected detection")
	}
}
