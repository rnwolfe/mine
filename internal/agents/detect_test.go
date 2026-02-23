package agents

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDetectAgents_ReturnsAllAgents(t *testing.T) {
	home := t.TempDir()
	agents := detectAgents(home)

	wantNames := []string{"claude", "codex", "gemini", "opencode"}
	if len(agents) != len(wantNames) {
		t.Fatalf("detectAgents() returned %d agents, want %d", len(agents), len(wantNames))
	}

	nameSet := make(map[string]bool)
	for _, a := range agents {
		nameSet[a.Name] = true
	}
	for _, name := range wantNames {
		if !nameSet[name] {
			t.Errorf("detectAgents() missing agent %q", name)
		}
	}
}

func TestDetectAgents_ConfigDirExists(t *testing.T) {
	home := t.TempDir()
	claudeDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	agents := detectAgents(home)

	var claude Agent
	for _, a := range agents {
		if a.Name == "claude" {
			claude = a
			break
		}
	}

	if !claude.Detected {
		t.Error("claude.Detected = false, want true when config dir exists")
	}
	if claude.ConfigDir != claudeDir {
		t.Errorf("claude.ConfigDir = %q, want %q", claude.ConfigDir, claudeDir)
	}
}

func TestDetectAgents_NothingDetected(t *testing.T) {
	home := t.TempDir() // empty home dir, no config dirs

	// Use an empty bin dir to shadow any real agent binaries in PATH.
	emptyBinDir := t.TempDir()
	t.Setenv("PATH", emptyBinDir)

	agents := detectAgents(home)

	for _, a := range agents {
		if a.Detected {
			t.Errorf("agent %q.Detected = true, want false in empty home", a.Name)
		}
	}
}

func TestDetectAgents_BinaryInPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake binary test not supported on Windows")
	}

	home := t.TempDir()
	binDir := filepath.Join(home, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a fake "codex" binary.
	fakeBin := filepath.Join(binDir, "codex")
	if err := os.WriteFile(fakeBin, []byte("#!/bin/sh\necho fake\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))
	// Patch home to avoid picking up real config dirs.
	t.Setenv("HOME", home)

	agents := detectAgents(home)

	var codex Agent
	for _, a := range agents {
		if a.Name == "codex" {
			codex = a
			break
		}
	}

	if !codex.Detected {
		t.Error("codex.Detected = false, want true when binary is in PATH")
	}
	if codex.Binary == "" {
		t.Error("codex.Binary is empty, want full path")
	}
}

func TestDirExists_True(t *testing.T) {
	dir := t.TempDir()
	if !DirExists(dir) {
		t.Errorf("DirExists(%q) = false, want true for existing directory", dir)
	}
}

func TestDirExists_False(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nonexistent")
	if DirExists(dir) {
		t.Errorf("DirExists(%q) = true, want false for nonexistent directory", dir)
	}
}
