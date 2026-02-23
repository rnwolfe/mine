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

// --- mine agents link ---

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

func TestRunAgentsLink_NotInitialized(t *testing.T) {
	agentsTestEnv(t)
	// Do NOT run init.

	out := captureStdout(t, func() {
		agentsLinkAgent = ""
		agentsLinkCopy = false
		agentsLinkForce = false
		if err := runAgentsLink(nil, nil); err != nil {
			t.Errorf("runAgentsLink: %v", err)
		}
	})

	if !strings.Contains(out, "No agents store yet") {
		t.Errorf("expected 'No agents store yet' in output, got:\n%s", out)
	}
}

func TestRunAgentsLink_CreatesSymlinks(t *testing.T) {
	_, claudeConfigDir := setupAgentsLinkEnv(t)

	agentsLinkAgent = ""
	agentsLinkCopy = false
	agentsLinkForce = false
	out := captureStdout(t, func() {
		if err := runAgentsLink(nil, nil); err != nil {
			t.Errorf("runAgentsLink: %v", err)
		}
	})

	if !strings.Contains(out, "link") && !strings.Contains(out, "created") && !strings.Contains(out, "claude") {
		t.Errorf("expected link output, got:\n%s", out)
	}

	// Verify symlink created at ~/.claude/CLAUDE.md.
	target := filepath.Join(claudeConfigDir, "CLAUDE.md")
	info, err := os.Lstat(target)
	if err != nil {
		t.Fatalf("Lstat target: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("CLAUDE.md is not a symlink, want symlink")
	}
}

func TestRunAgentsLink_CopyFlag(t *testing.T) {
	_, claudeConfigDir := setupAgentsLinkEnv(t)

	agentsLinkAgent = ""
	agentsLinkCopy = true
	agentsLinkForce = false
	defer func() { agentsLinkCopy = false }()

	captureStdout(t, func() {
		if err := runAgentsLink(nil, nil); err != nil {
			t.Errorf("runAgentsLink --copy: %v", err)
		}
	})

	target := filepath.Join(claudeConfigDir, "CLAUDE.md")
	info, err := os.Lstat(target)
	if err != nil {
		t.Fatalf("Lstat target: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("CLAUDE.md is a symlink, want regular file with --copy")
	}
}

func TestRunAgentsLink_AgentFilter(t *testing.T) {
	storeDir, _ := setupAgentsLinkEnv(t)
	home := os.Getenv("HOME")

	// Add a second detected agent (codex) to the manifest.
	codexConfigDir := filepath.Join(home, ".codex")
	m, _ := agents.ReadManifest()
	m.Agents = append(m.Agents, agents.Agent{
		Name:      "codex",
		Detected:  true,
		ConfigDir: codexConfigDir,
	})
	if err := agents.WriteManifest(m); err != nil {
		t.Fatal(err)
	}

	agentsLinkAgent = "claude"
	agentsLinkCopy = false
	agentsLinkForce = false
	defer func() { agentsLinkAgent = "" }()

	captureStdout(t, func() {
		if err := runAgentsLink(nil, nil); err != nil {
			t.Errorf("runAgentsLink --agent claude: %v", err)
		}
	})

	_ = storeDir
	// Only claude should have a link; codex should not.
	claudeTarget := filepath.Join(home, ".claude", "CLAUDE.md")
	if _, err := os.Lstat(claudeTarget); err != nil {
		t.Errorf("claude CLAUDE.md not found after --agent claude link: %v", err)
	}

	codexTarget := filepath.Join(codexConfigDir, "AGENTS.md")
	if _, err := os.Lstat(codexTarget); err == nil {
		t.Error("codex AGENTS.md exists after --agent claude link, want no codex links")
	}
}

func TestRunAgentsLink_ForceOverwrites(t *testing.T) {
	_, claudeConfigDir := setupAgentsLinkEnv(t)

	// Create an existing file at the target.
	if err := os.MkdirAll(claudeConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(claudeConfigDir, "CLAUDE.md")
	if err := os.WriteFile(target, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	agentsLinkAgent = ""
	agentsLinkCopy = false
	agentsLinkForce = true
	defer func() { agentsLinkForce = false }()

	captureStdout(t, func() {
		if err := runAgentsLink(nil, nil); err != nil {
			t.Errorf("runAgentsLink --force: %v", err)
		}
	})

	info, err := os.Lstat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("target is not a symlink after --force link, want symlink")
	}
}

func TestRunAgentsLink_PersistsManifest(t *testing.T) {
	setupAgentsLinkEnv(t)

	agentsLinkAgent = ""
	agentsLinkCopy = false
	agentsLinkForce = false

	captureStdout(t, func() {
		if err := runAgentsLink(nil, nil); err != nil {
			t.Fatalf("runAgentsLink: %v", err)
		}
	})

	m, err := agents.ReadManifest()
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if len(m.Links) == 0 {
		t.Error("manifest.Links is empty after link, want at least one entry")
	}
}

// --- mine agents unlink ---

func TestRunAgentsUnlink_NotInitialized(t *testing.T) {
	agentsTestEnv(t)

	out := captureStdout(t, func() {
		agentsUnlinkAgent = ""
		if err := runAgentsUnlink(nil, nil); err != nil {
			t.Errorf("runAgentsUnlink: %v", err)
		}
	})

	if !strings.Contains(out, "No agents store yet") {
		t.Errorf("expected 'No agents store yet' in output, got:\n%s", out)
	}
}

func TestRunAgentsUnlink_NoLinks(t *testing.T) {
	agentsTestEnv(t)
	captureStdout(t, func() {
		if err := runAgentsInit(nil, nil); err != nil {
			t.Fatalf("runAgentsInit: %v", err)
		}
	})

	out := captureStdout(t, func() {
		agentsUnlinkAgent = ""
		if err := runAgentsUnlink(nil, nil); err != nil {
			t.Errorf("runAgentsUnlink: %v", err)
		}
	})

	if !strings.Contains(out, "No links to remove") {
		t.Errorf("expected 'No links to remove' in output, got:\n%s", out)
	}
}

func TestRunAgentsUnlink_ReplacesSymlinkWithFile(t *testing.T) {
	_, claudeConfigDir := setupAgentsLinkEnv(t)

	// Link first.
	agentsLinkAgent = ""
	agentsLinkCopy = false
	agentsLinkForce = false
	captureStdout(t, func() {
		if err := runAgentsLink(nil, nil); err != nil {
			t.Fatalf("runAgentsLink: %v", err)
		}
	})

	target := filepath.Join(claudeConfigDir, "CLAUDE.md")
	info, _ := os.Lstat(target)
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("target not a symlink after link")
	}

	// Unlink.
	out := captureStdout(t, func() {
		agentsUnlinkAgent = ""
		if err := runAgentsUnlink(nil, nil); err != nil {
			t.Errorf("runAgentsUnlink: %v", err)
		}
	})

	if !strings.Contains(out, "unlinked") {
		t.Errorf("expected 'unlinked' in output, got:\n%s", out)
	}

	info, err := os.Lstat(target)
	if err != nil {
		t.Fatalf("Lstat after unlink: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("target is still a symlink after unlink, want regular file")
	}
}

func TestRunAgentsUnlink_AgentFilter(t *testing.T) {
	_, claudeConfigDir := setupAgentsLinkEnv(t)
	home := os.Getenv("HOME")

	// Add codex as detected.
	codexConfigDir := filepath.Join(home, ".codex")
	m, _ := agents.ReadManifest()
	m.Agents = append(m.Agents, agents.Agent{
		Name:      "codex",
		Detected:  true,
		ConfigDir: codexConfigDir,
	})
	if err := agents.WriteManifest(m); err != nil {
		t.Fatal(err)
	}

	// Link all agents.
	agentsLinkAgent = ""
	agentsLinkCopy = false
	agentsLinkForce = false
	captureStdout(t, func() {
		if err := runAgentsLink(nil, nil); err != nil {
			t.Fatalf("runAgentsLink: %v", err)
		}
	})

	// Unlink only claude.
	captureStdout(t, func() {
		agentsUnlinkAgent = "claude"
		defer func() { agentsUnlinkAgent = "" }()
		if err := runAgentsUnlink(nil, nil); err != nil {
			t.Errorf("runAgentsUnlink --agent claude: %v", err)
		}
	})

	// Claude target should be a regular file now.
	claudeTarget := filepath.Join(claudeConfigDir, "CLAUDE.md")
	info, err := os.Lstat(claudeTarget)
	if err != nil {
		t.Fatalf("Lstat claude target: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("claude target is still a symlink after unlink --agent claude")
	}

	// Codex target should still be a symlink.
	codexTarget := filepath.Join(codexConfigDir, "AGENTS.md")
	info, err = os.Lstat(codexTarget)
	if err != nil {
		t.Fatalf("Lstat codex target: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("codex target is not a symlink after unlink --agent claude, want symlink preserved")
	}
}

func TestRunAgentsUnlink_ClearsManifestLinks(t *testing.T) {
	setupAgentsLinkEnv(t)

	agentsLinkAgent = ""
	agentsLinkCopy = false
	agentsLinkForce = false
	captureStdout(t, func() {
		if err := runAgentsLink(nil, nil); err != nil {
			t.Fatalf("runAgentsLink: %v", err)
		}
	})

	m, _ := agents.ReadManifest()
	if len(m.Links) == 0 {
		t.Fatal("no links in manifest before unlink")
	}

	captureStdout(t, func() {
		agentsUnlinkAgent = ""
		if err := runAgentsUnlink(nil, nil); err != nil {
			t.Errorf("runAgentsUnlink: %v", err)
		}
	})

	m, err := agents.ReadManifest()
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if len(m.Links) != 0 {
		t.Errorf("manifest.Links count = %d after unlink, want 0", len(m.Links))
	}
}

// TestRunAgentsLinkUnlink_FullCycle is an integration test covering the full
// link → verify symlinks → unlink → verify standalone cycle via the cmd handlers.
func TestRunAgentsLinkUnlink_FullCycle(t *testing.T) {
	_, claudeConfigDir := setupAgentsLinkEnv(t)

	// 1. Link.
	agentsLinkAgent = ""
	agentsLinkCopy = false
	agentsLinkForce = false
	captureStdout(t, func() {
		if err := runAgentsLink(nil, nil); err != nil {
			t.Fatalf("runAgentsLink: %v", err)
		}
	})

	// 2. Verify symlink exists.
	target := filepath.Join(claudeConfigDir, "CLAUDE.md")
	info, err := os.Lstat(target)
	if err != nil {
		t.Fatalf("Lstat after link: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("target not a symlink after link step")
	}

	// 3. Unlink.
	agentsUnlinkAgent = ""
	captureStdout(t, func() {
		if err := runAgentsUnlink(nil, nil); err != nil {
			t.Fatalf("runAgentsUnlink: %v", err)
		}
	})

	// 4. Verify standalone file.
	info, err = os.Lstat(target)
	if err != nil {
		t.Fatalf("Lstat after unlink: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("target is still a symlink after unlink step")
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading target after unlink: %v", err)
	}
	if !strings.Contains(string(data), "Shared Instructions") {
		t.Errorf("target content after unlink = %q, want original instruction content", string(data))
	}

	// 5. Verify manifest has no links.
	m, err := agents.ReadManifest()
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Links) != 0 {
		t.Errorf("manifest links = %d after full cycle, want 0", len(m.Links))
	}
}
