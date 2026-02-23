package cmd

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rnwolfe/mine/internal/config"
	"github.com/rnwolfe/mine/internal/proj"
	"github.com/rnwolfe/mine/internal/store"
)

// --- probeEnvironment unit tests ---

func TestProbeEnvironment_GitTmuxPresent(t *testing.T) {
	stubDir := t.TempDir()

	// Create fake git binary
	gitStub := filepath.Join(stubDir, "git")
	if err := os.WriteFile(gitStub, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Create fake tmux binary
	tmuxStub := filepath.Join(stubDir, "tmux")
	if err := os.WriteFile(tmuxStub, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv("PATH", stubDir+":"+os.Getenv("PATH"))

	cfg := &config.Config{AI: config.AIConfig{Provider: "claude"}}
	probe := probeEnvironment(cfg)

	if !probe.gitInstalled {
		t.Error("expected gitInstalled=true when git stub is on PATH")
	}
	if !probe.tmuxInstalled {
		t.Error("expected tmuxInstalled=true when tmux stub is on PATH")
	}
	if !probe.aiConfigured {
		t.Error("expected aiConfigured=true when cfg.AI.Provider is set")
	}
	if probe.aiProvider != "claude" {
		t.Errorf("expected aiProvider='claude', got %q", probe.aiProvider)
	}
}

func TestProbeEnvironment_NoBinaries(t *testing.T) {
	emptyDir := t.TempDir()
	t.Setenv("PATH", emptyDir)

	probe := probeEnvironment(nil)

	if probe.gitInstalled {
		t.Error("expected gitInstalled=false with empty PATH")
	}
	if probe.tmuxInstalled {
		t.Error("expected tmuxInstalled=false with empty PATH")
	}
	if probe.aiConfigured {
		t.Error("expected aiConfigured=false with nil cfg")
	}
}

func TestProbeEnvironment_AINotConfigured(t *testing.T) {
	cfg := &config.Config{AI: config.AIConfig{Provider: ""}}
	probe := probeEnvironment(cfg)

	if probe.aiConfigured {
		t.Error("expected aiConfigured=false when Provider is empty")
	}
	if probe.aiProvider != "" {
		t.Errorf("expected aiProvider='', got %q", probe.aiProvider)
	}
}

func TestProbeEnvironment_InGitRepo(t *testing.T) {
	repoDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoDir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	t.Chdir(repoDir)

	probe := probeEnvironment(nil)

	if !probe.inGitRepo {
		t.Error("expected inGitRepo=true when .git directory exists in cwd")
	}
	if probe.cwd != repoDir {
		t.Errorf("expected cwd=%q, got %q", repoDir, probe.cwd)
	}
}

func TestProbeEnvironment_NotInGitRepo(t *testing.T) {
	plainDir := t.TempDir()
	t.Chdir(plainDir)

	probe := probeEnvironment(nil)

	if probe.inGitRepo {
		t.Error("expected inGitRepo=false when no .git directory exists")
	}
}

// --- runInit integration tests ---

// makeInitStdin constructs a reader that simulates user input for runInit.
// answers are newline-separated responses in order: name, openrouter choice,
// and optionally a project registration response.
func makeInitStdin(answers ...string) *bufio.Reader {
	input := strings.Join(answers, "\n") + "\n"
	return bufio.NewReader(strings.NewReader(input))
}

// runInitEnv sets up a clean XDG environment for init tests.
func runInitEnv(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir+"/config")
	t.Setenv("XDG_DATA_HOME", tmpDir+"/data")
	t.Setenv("XDG_CACHE_HOME", tmpDir+"/cache")
	t.Setenv("XDG_STATE_HOME", tmpDir+"/state")
	// Suppress AI key detection so we get the predictable no-key branch
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")
	// Use a stable USER so guessName() doesn't rely on real git config
	t.Setenv("USER", "testuser")
	// Suppress keychain so readPassphrase never prompts for a passphrase
	t.Setenv("MINE_VAULT_PASSPHRASE", "testpassphrase")
}

func TestRunInit_CapabilityTable_WithBinaries(t *testing.T) {
	runInitEnv(t)

	// Put fake git and tmux stubs on PATH
	stubDir := t.TempDir()
	for _, bin := range []string{"git", "tmux"} {
		stub := filepath.Join(stubDir, bin)
		if err := os.WriteFile(stub, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", stubDir+":"+os.Getenv("PATH"))

	// Not in a git repo — no registration prompt
	plainDir := t.TempDir()
	t.Chdir(plainDir)

	// Input: name="\n" (use default), openrouter="n"
	reader := makeInitStdin("", "n")

	out := captureStdout(t, func() {
		if err := runInitWithReader(reader); err != nil {
			t.Errorf("runInitWithReader: %v", err)
		}
	})

	// Both git and tmux should show as ready (✓)
	if !strings.Contains(out, "git") {
		t.Errorf("expected 'git' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "tmux") {
		t.Errorf("expected 'tmux' in output, got:\n%s", out)
	}
	// Should show the ready command examples
	if !strings.Contains(out, "mine git log") {
		t.Errorf("expected 'mine git log' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "mine tmux new") {
		t.Errorf("expected 'mine tmux new' in output, got:\n%s", out)
	}
	// Always-ready features must be present
	if !strings.Contains(out, "todos") {
		t.Errorf("expected 'todos' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "stash") {
		t.Errorf("expected 'stash' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "env") {
		t.Errorf("expected 'env' in output, got:\n%s", out)
	}
}

func TestRunInit_CapabilityTable_NoBinaries(t *testing.T) {
	runInitEnv(t)

	// Only empty dir on PATH — no git or tmux
	emptyDir := t.TempDir()
	t.Setenv("PATH", emptyDir)

	plainDir := t.TempDir()
	t.Chdir(plainDir)

	reader := makeInitStdin("", "n")

	out := captureStdout(t, func() {
		if err := runInitWithReader(reader); err != nil {
			t.Errorf("runInitWithReader: %v", err)
		}
	})

	// Unready rows should show setup hints
	if !strings.Contains(out, "install git") {
		t.Errorf("expected git install hint in output, got:\n%s", out)
	}
	if !strings.Contains(out, "install tmux") {
		t.Errorf("expected tmux install hint in output, got:\n%s", out)
	}
}

func TestRunInit_ProjectRegistration_InGitRepo(t *testing.T) {
	runInitEnv(t)

	// Use real git binary if available so probeEnvironment works naturally
	// Also need PATH for store open (no external deps there)
	repoDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoDir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repoDir)

	// Input: name="\n" (default), openrouter="n", register="y"
	reader := makeInitStdin("", "n", "y")

	out := captureStdout(t, func() {
		if err := runInitWithReader(reader); err != nil {
			t.Errorf("runInitWithReader: %v", err)
		}
	})

	// Output should mention the registration prompt and success
	if !strings.Contains(out, "Register") {
		t.Errorf("expected registration prompt in output, got:\n%s", out)
	}

	// Verify project is in the store
	db, err := store.Open()
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	defer db.Close()

	ps := proj.NewStore(db.Conn())
	projects, err := ps.List()
	if err != nil {
		t.Fatalf("ps.List: %v", err)
	}

	found := false
	for _, p := range projects {
		if p.Path == repoDir {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected project at %q to be registered, got projects: %v", repoDir, projects)
	}
}

func TestRunInit_NoRegistrationPrompt_OutsideGitRepo(t *testing.T) {
	runInitEnv(t)

	// Plain directory — no .git
	plainDir := t.TempDir()
	t.Chdir(plainDir)

	// Only 2 answers needed: name and openrouter
	reader := makeInitStdin("", "n")

	out := captureStdout(t, func() {
		if err := runInitWithReader(reader); err != nil {
			t.Errorf("runInitWithReader: %v", err)
		}
	})

	if strings.Contains(out, "Register") {
		t.Errorf("expected no registration prompt outside a git repo, got:\n%s", out)
	}
}

func TestRunInit_ProjRow_RegisteredShowsReady(t *testing.T) {
	runInitEnv(t)

	repoDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoDir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repoDir)

	// Answer "y" to registration
	reader := makeInitStdin("", "n", "y")

	out := captureStdout(t, func() {
		if err := runInitWithReader(reader); err != nil {
			t.Errorf("runInitWithReader: %v", err)
		}
	})

	// proj row should show mine proj list (the ready command)
	if !strings.Contains(out, "mine proj list") {
		t.Errorf("expected 'mine proj list' in output after registration, got:\n%s", out)
	}
}

func TestRunInit_ProjRow_NotRegisteredShowsHint(t *testing.T) {
	runInitEnv(t)

	plainDir := t.TempDir()
	t.Chdir(plainDir)

	reader := makeInitStdin("", "n")

	out := captureStdout(t, func() {
		if err := runInitWithReader(reader); err != nil {
			t.Errorf("runInitWithReader: %v", err)
		}
	})

	// proj row should show the add hint
	if !strings.Contains(out, "mine proj add") {
		t.Errorf("expected 'mine proj add' hint in output, got:\n%s", out)
	}
}
