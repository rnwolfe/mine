package config

import (
	"os"
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
	if paths.DBFile == "" {
		t.Fatal("DBFile should not be empty")
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
