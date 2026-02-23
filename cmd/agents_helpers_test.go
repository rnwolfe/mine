package cmd

import (
	"os"
	"path/filepath"
	"runtime"
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

// setupAgentsLinkEnv initializes an agents store with a detected agent and
// an instruction file in the store, ready for link tests.
// Returns (storeDir, claudeConfigDir).
func setupAgentsLinkEnv(t *testing.T) (string, string) {
	t.Helper()
	agentsTestEnv(t)

	// Ensure the agents store is initialized.
	captureStdout(t, func() {
		if err := runAgentsInit(nil, nil); err != nil {
			t.Fatalf("runAgentsInit: %v", err)
		}
	})

	storeDir := agents.Dir()
	home := os.Getenv("HOME")
	claudeConfigDir := filepath.Join(home, ".claude")

	// Write the instructions file.
	instrFile := filepath.Join(storeDir, "instructions", "AGENTS.md")
	if err := os.WriteFile(instrFile, []byte("# Shared Instructions\n"), 0o644); err != nil {
		t.Fatalf("writing instructions file: %v", err)
	}

	// Register claude as detected in the manifest.
	m, err := agents.ReadManifest()
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	m.Agents = []agents.Agent{
		{Name: "claude", Detected: true, ConfigDir: claudeConfigDir},
	}
	if err := agents.WriteManifest(m); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}

	return storeDir, claudeConfigDir
}

// setupAgentsAdoptEnv initializes an agents store for adopt tests.
// Returns (storeDir, homeDir).
func setupAgentsAdoptEnv(t *testing.T) (string, string) {
	t.Helper()
	agentsTestEnv(t)

	captureStdout(t, func() {
		if err := runAgentsInit(nil, nil); err != nil {
			t.Fatalf("runAgentsInit: %v", err)
		}
	})

	return agents.Dir(), os.Getenv("HOME")
}
