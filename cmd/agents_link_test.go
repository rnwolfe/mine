package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rnwolfe/mine/internal/agents"
)

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
