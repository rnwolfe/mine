package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetPaths(t *testing.T) {
	paths := GetPaths()

	if paths.ConfigDir == "" {
		t.Fatal("ConfigDir should not be empty")
	}
	if paths.DataDir == "" {
		t.Fatal("DataDir should not be empty")
	}
	if paths.ConfigFile == "" {
		t.Fatal("ConfigFile should not be empty")
	}
	if paths.ProjectsFile == "" {
		t.Fatal("ProjectsFile should not be empty")
	}
	if paths.DBFile == "" {
		t.Fatal("DBFile should not be empty")
	}
	if paths.EnvDir == "" {
		t.Fatal("EnvDir should not be empty")
	}
}

func TestGetPathsRespectsXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/testxdg/config")
	t.Setenv("XDG_DATA_HOME", "/tmp/testxdg/data")

	paths := GetPaths()

	if paths.ConfigDir != "/tmp/testxdg/config/mine" {
		t.Fatalf("expected /tmp/testxdg/config/mine, got %s", paths.ConfigDir)
	}
	if paths.DataDir != "/tmp/testxdg/data/mine" {
		t.Fatalf("expected /tmp/testxdg/data/mine, got %s", paths.DataDir)
	}
	if paths.EnvDir != "/tmp/testxdg/data/mine/envs" {
		t.Fatalf("expected /tmp/testxdg/data/mine/envs, got %s", paths.EnvDir)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()

	if cfg.AI.Provider != "claude" {
		t.Fatalf("expected provider 'claude', got %q", cfg.AI.Provider)
	}
	if cfg.Shell.DefaultShell == "" {
		t.Fatal("DefaultShell should not be empty")
	}
}

func TestAIConfigSystemInstructionsRoundtrip(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir+"/config")
	t.Setenv("XDG_DATA_HOME", tmpDir+"/data")
	t.Setenv("XDG_CACHE_HOME", tmpDir+"/cache")
	t.Setenv("XDG_STATE_HOME", tmpDir+"/state")

	cfg := &Config{
		AI: AIConfig{
			Provider:                 "claude",
			Model:                    "claude-sonnet-4-5-20250929",
			SystemInstructions:       "global system instructions",
			AskSystemInstructions:    "ask-specific instructions",
			ReviewSystemInstructions: "review-specific instructions",
			CommitSystemInstructions: "commit-specific instructions",
		},
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.AI.SystemInstructions != "global system instructions" {
		t.Errorf("SystemInstructions = %q, want %q", loaded.AI.SystemInstructions, "global system instructions")
	}
	if loaded.AI.AskSystemInstructions != "ask-specific instructions" {
		t.Errorf("AskSystemInstructions = %q, want %q", loaded.AI.AskSystemInstructions, "ask-specific instructions")
	}
	if loaded.AI.ReviewSystemInstructions != "review-specific instructions" {
		t.Errorf("ReviewSystemInstructions = %q, want %q", loaded.AI.ReviewSystemInstructions, "review-specific instructions")
	}
	if loaded.AI.CommitSystemInstructions != "commit-specific instructions" {
		t.Errorf("CommitSystemInstructions = %q, want %q", loaded.AI.CommitSystemInstructions, "commit-specific instructions")
	}
}

func TestAIConfigSystemInstructionsOmitEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir+"/config")
	t.Setenv("XDG_DATA_HOME", tmpDir+"/data")
	t.Setenv("XDG_CACHE_HOME", tmpDir+"/cache")
	t.Setenv("XDG_STATE_HOME", tmpDir+"/state")

	cfg := &Config{
		AI: AIConfig{
			Provider: "claude",
			Model:    "claude-sonnet-4-5-20250929",
			// System instruction fields intentionally empty
		},
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Verify the file does not contain the omitempty fields.
	paths := GetPaths()
	data, err := os.ReadFile(filepath.Join(paths.ConfigDir, "config.toml"))
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	content := string(data)
	for _, field := range []string{"system_instructions", "ask_system_instructions", "review_system_instructions", "commit_system_instructions"} {
		if strings.Contains(content, field) {
			t.Errorf("config file should not contain %q when empty, but got:\n%s", field, content)
		}
	}
}

func TestEnsureDirs(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir+"/config")
	t.Setenv("XDG_DATA_HOME", tmpDir+"/data")
	t.Setenv("XDG_CACHE_HOME", tmpDir+"/cache")
	t.Setenv("XDG_STATE_HOME", tmpDir+"/state")

	paths := GetPaths()
	if err := paths.EnsureDirs(); err != nil {
		t.Fatalf("EnsureDirs failed: %v", err)
	}

	// Check dirs exist
	for _, dir := range []string{paths.ConfigDir, paths.DataDir, paths.CacheDir, paths.StateDir} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("dir %s not created: %v", dir, err)
		}
		if !info.IsDir() {
			t.Fatalf("%s is not a directory", dir)
		}
	}
}
