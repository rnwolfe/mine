package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rnwolfe/mine/internal/agents"
)

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
