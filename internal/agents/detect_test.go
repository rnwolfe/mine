package agents

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// setupDetectEnv creates a temp home directory for detection tests.
// It sets XDG_DATA_HOME for the store and HOME for config dir detection.
func setupDetectEnv(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmpDir, "data"))
	return tmpDir
}

// makeFakeBinary creates a minimal executable in dir named name.
func makeFakeBinary(t *testing.T, dir, name string) string {
	t.Helper()
	var script string
	if runtime.GOOS == "windows" {
		t.Skip("fake binary tests not supported on Windows")
	}
	script = "#!/bin/sh\necho fake\n"
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(script), 0o755); err != nil {
		t.Fatalf("creating fake binary %s: %v", name, err)
	}
	return p
}

func TestBuildRegistry_ContainsAllAgents(t *testing.T) {
	home := "/home/testuser"
	specs := buildRegistry(home)

	wantNames := []string{"claude", "codex", "gemini", "opencode"}
	if len(specs) != len(wantNames) {
		t.Fatalf("registry length = %d, want %d", len(specs), len(wantNames))
	}

	nameSet := make(map[string]bool, len(specs))
	for _, s := range specs {
		nameSet[s.Name] = true
	}
	for _, name := range wantNames {
		if !nameSet[name] {
			t.Errorf("registry missing agent %q", name)
		}
	}
}

func TestBuildRegistry_ConfigDirsUseHome(t *testing.T) {
	home := "/custom/home"
	specs := buildRegistry(home)

	for _, spec := range specs {
		if !filepath.IsAbs(spec.ConfigDir) {
			t.Errorf("agent %q ConfigDir %q is not absolute", spec.Name, spec.ConfigDir)
		}
		if !strings.HasPrefix(spec.ConfigDir, home+string(filepath.Separator)) {
			t.Errorf("agent %q ConfigDir %q does not start with home %q", spec.Name, spec.ConfigDir, home)
		}
	}
}

func TestDetectBinary_Found(t *testing.T) {
	binDir := t.TempDir()
	makeFakeBinary(t, binDir, "fakecli")
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))

	path, ok := detectBinary("fakecli")
	if !ok {
		t.Error("detectBinary() = false, want true for existing binary")
	}
	if path == "" {
		t.Error("detectBinary() path is empty, want non-empty path")
	}
}

func TestDetectBinary_NotFound(t *testing.T) {
	// Use a fresh PATH with only a temp dir that has no binaries.
	binDir := t.TempDir()
	t.Setenv("PATH", binDir)

	path, ok := detectBinary("this-binary-does-not-exist-xyz")
	if ok {
		t.Error("detectBinary() = true, want false for missing binary")
	}
	if path != "" {
		t.Errorf("detectBinary() path = %q, want empty string", path)
	}
}

func TestDetectConfigDir_Exists(t *testing.T) {
	dir := t.TempDir()
	if !detectConfigDir(dir) {
		t.Errorf("detectConfigDir(%q) = false, want true for existing dir", dir)
	}
}

func TestDetectConfigDir_Missing(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nonexistent")
	if detectConfigDir(missing) {
		t.Errorf("detectConfigDir(%q) = true, want false for missing dir", missing)
	}
}

func TestDetectConfigDir_File(t *testing.T) {
	// A file should not count as a config dir.
	tmp := t.TempDir()
	filePath := filepath.Join(tmp, "notadir")
	if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if detectConfigDir(filePath) {
		t.Errorf("detectConfigDir(%q) = true for a file, want false", filePath)
	}
}

func TestDetectAgents_NoneInstalled(t *testing.T) {
	home := setupDetectEnv(t)
	// Use a fresh PATH with only an empty dir (no agent binaries).
	binDir := t.TempDir()
	t.Setenv("PATH", binDir)
	// home has no .claude, .codex etc. subdirs.
	_ = home

	agents := detectAgents(home)
	if len(agents) != 4 {
		t.Fatalf("detectAgents() returned %d entries, want 4 entries (one per registry agent)", len(agents))
	}
	for _, a := range agents {
		if a.Detected {
			t.Errorf("agent %q detected = true, want false (no binaries or config dirs)", a.Name)
		}
		if a.Binary != "" {
			t.Errorf("agent %q binary = %q, want empty (not in PATH)", a.Name, a.Binary)
		}
	}
}

func TestDetectAgents_BinaryFound(t *testing.T) {
	home := setupDetectEnv(t)
	binDir := t.TempDir()
	makeFakeBinary(t, binDir, "claude")
	t.Setenv("PATH", binDir)

	agents := detectAgents(home)

	var claudeAgent *Agent
	for i := range agents {
		if agents[i].Name == "claude" {
			claudeAgent = &agents[i]
			break
		}
	}
	if claudeAgent == nil {
		t.Fatal("claude not found in detectAgents() result")
	}
	if !claudeAgent.Detected {
		t.Error("claude.Detected = false, want true (binary in PATH)")
	}
	if claudeAgent.Binary == "" {
		t.Error("claude.Binary is empty, want full path")
	}
}

func TestDetectAgents_ConfigDirFound(t *testing.T) {
	home := setupDetectEnv(t)
	// No binaries in PATH.
	binDir := t.TempDir()
	t.Setenv("PATH", binDir)
	// Create ~/.gemini config dir.
	geminiDir := filepath.Join(home, ".gemini")
	if err := os.MkdirAll(geminiDir, 0o755); err != nil {
		t.Fatal(err)
	}

	agents := detectAgents(home)

	var geminiAgent *Agent
	for i := range agents {
		if agents[i].Name == "gemini" {
			geminiAgent = &agents[i]
			break
		}
	}
	if geminiAgent == nil {
		t.Fatal("gemini not found in detectAgents() result")
	}
	if !geminiAgent.Detected {
		t.Error("gemini.Detected = false, want true (config dir exists)")
	}
}

func TestDetectAgents_AllFourAgentsPresent(t *testing.T) {
	home := setupDetectEnv(t)
	binDir := t.TempDir()
	t.Setenv("PATH", binDir)

	agents := detectAgents(home)

	wantNames := map[string]bool{
		"claude":   false,
		"codex":    false,
		"gemini":   false,
		"opencode": false,
	}
	for _, a := range agents {
		wantNames[a.Name] = true
	}
	for name, seen := range wantNames {
		if !seen {
			t.Errorf("agent %q missing from detectAgents() result", name)
		}
	}
}

func TestDetectAgents_ConfigDirStoredInAgent(t *testing.T) {
	home := setupDetectEnv(t)
	binDir := t.TempDir()
	t.Setenv("PATH", binDir)

	agents := detectAgents(home)

	for _, a := range agents {
		if a.ConfigDir == "" {
			t.Errorf("agent %q ConfigDir is empty, want non-empty path", a.Name)
		}
	}
}

func TestDetectAgents_BothBinaryAndConfigDir(t *testing.T) {
	home := setupDetectEnv(t)
	binDir := t.TempDir()
	makeFakeBinary(t, binDir, "codex")
	t.Setenv("PATH", binDir)
	// Also create ~/.codex config dir.
	if err := os.MkdirAll(filepath.Join(home, ".codex"), 0o755); err != nil {
		t.Fatal(err)
	}

	agents := detectAgents(home)

	var codexAgent *Agent
	for i := range agents {
		if agents[i].Name == "codex" {
			codexAgent = &agents[i]
			break
		}
	}
	if codexAgent == nil {
		t.Fatal("codex not found in detectAgents() result")
	}
	if !codexAgent.Detected {
		t.Error("codex.Detected = false, want true")
	}
	if codexAgent.Binary == "" {
		t.Error("codex.Binary is empty, want full path")
	}
}
