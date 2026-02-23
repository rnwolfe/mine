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
	t.Setenv("OPENROUTER_API_KEY", "")
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
		if err := runInitWithReader(reader, false); err != nil {
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
		if err := runInitWithReader(reader, false); err != nil {
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
		if err := runInitWithReader(reader, false); err != nil {
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
		if err := runInitWithReader(reader, false); err != nil {
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
		if err := runInitWithReader(reader, false); err != nil {
			t.Errorf("runInitWithReader: %v", err)
		}
	})

	// proj row should show mine proj list (the ready command)
	if !strings.Contains(out, "mine proj list") {
		t.Errorf("expected 'mine proj list' in output after registration, got:\n%s", out)
	}
}

func TestRunInit_AlreadyRegisteredProject(t *testing.T) {
	runInitEnv(t)

	repoDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(repoDir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(repoDir)

	// Pre-register the project so init encounters ErrProjectExists
	db, err := store.Open()
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	ps := proj.NewStore(db.Conn())
	if _, err := ps.Add(repoDir); err != nil {
		t.Fatalf("pre-register: %v", err)
	}
	db.Close()

	// Answer "y" to the registration prompt — it's already registered
	reader := makeInitStdin("", "n", "y")

	out := captureStdout(t, func() {
		if err := runInitWithReader(reader, false); err != nil {
			t.Errorf("runInitWithReader: %v", err)
		}
	})

	// proj row should show as ready (already-registered counts as registered)
	if !strings.Contains(out, "mine proj list") {
		t.Errorf("expected 'mine proj list' for already-registered project, got:\n%s", out)
	}
	// Should not have shown an error
	if strings.Contains(out, "Could not register") {
		t.Errorf("unexpected error output for already-registered project:\n%s", out)
	}
}

func TestRunInit_ProjRow_NotRegisteredShowsHint(t *testing.T) {
	runInitEnv(t)

	plainDir := t.TempDir()
	t.Chdir(plainDir)

	reader := makeInitStdin("", "n")

	out := captureStdout(t, func() {
		if err := runInitWithReader(reader, false); err != nil {
			t.Errorf("runInitWithReader: %v", err)
		}
	})

	// proj row should show the add hint
	if !strings.Contains(out, "mine proj add") {
		t.Errorf("expected 'mine proj add' hint in output, got:\n%s", out)
	}
}

// ---- unit tests: rcFileForShell ----

func TestRcFileForShell_Zsh(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	got := rcFileForShell("/bin/zsh")
	want := filepath.Join(tmp, ".zshrc")
	if got != want {
		t.Errorf("zsh: got %q, want %q", got, want)
	}
}

func TestRcFileForShell_Bash_WithBashrc(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	// Create .bashrc so it takes priority over .bash_profile.
	bashrc := filepath.Join(tmp, ".bashrc")
	if err := os.WriteFile(bashrc, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	got := rcFileForShell("/bin/bash")
	if got != bashrc {
		t.Errorf("bash with .bashrc: got %q, want %q", got, bashrc)
	}
}

func TestRcFileForShell_Bash_FallbackProfile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	// No .bashrc — should fall back to .bash_profile.

	got := rcFileForShell("/bin/bash")
	want := filepath.Join(tmp, ".bash_profile")
	if got != want {
		t.Errorf("bash without .bashrc: got %q, want %q", got, want)
	}
}

func TestRcFileForShell_Fish(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	got := rcFileForShell("/usr/bin/fish")
	want := filepath.Join(tmp, ".config", "fish", "config.fish")
	if got != want {
		t.Errorf("fish: got %q, want %q", got, want)
	}
}

func TestRcFileForShell_Unknown(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	got := rcFileForShell("/bin/tcsh")
	if got != "" {
		t.Errorf("unknown shell: expected empty string, got %q", got)
	}
}

// ---- unit tests: alreadyInstalled ----

func TestAlreadyInstalled_Present(t *testing.T) {
	tmp := t.TempDir()
	rc := filepath.Join(tmp, ".zshrc")
	content := `# existing config
eval "$(mine shell init)"
`
	if err := os.WriteFile(rc, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if !alreadyInstalled(rc) {
		t.Error("expected alreadyInstalled to return true when snippet present")
	}
}

func TestAlreadyInstalled_Absent(t *testing.T) {
	tmp := t.TempDir()
	rc := filepath.Join(tmp, ".zshrc")
	if err := os.WriteFile(rc, []byte("# empty config\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if alreadyInstalled(rc) {
		t.Error("expected alreadyInstalled to return false when snippet absent")
	}
}

func TestAlreadyInstalled_MissingFile(t *testing.T) {
	tmp := t.TempDir()
	rc := filepath.Join(tmp, ".zshrc") // does not exist
	if alreadyInstalled(rc) {
		t.Error("expected alreadyInstalled to return false for missing file")
	}
}

// ---- integration tests: runInit with shell integration ----

// initTestEnv sets up a complete temp environment for runInit tests.
// It returns the temp home directory and a cleanup via t.Cleanup.
func initTestEnv(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, "data"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmp, "cache"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmp, "state"))
	// Clear API keys so the AI section takes the "no keys" branch.
	t.Setenv("ANTHROPIC_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("GEMINI_API_KEY", "")
	t.Setenv("OPENROUTER_API_KEY", "")
	// Use a fake git config so guessName returns "".
	t.Setenv("USER", "testuser")
	return tmp
}

// pipeStdin replaces os.Stdin with a pipe containing the given input string.
// The previous os.Stdin is restored via t.Cleanup.
func pipeStdin(t *testing.T, input string) {
	t.Helper()
	original := os.Stdin
	t.Cleanup(func() { os.Stdin = original })

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdin = r

	if _, err := w.WriteString(input); err != nil {
		t.Fatalf("write stdin: %v", err)
	}
	w.Close()
}

// runInitInputs returns stdin content for runInit that:
//   - Accepts default name ("\n")
//   - Skips OpenRouter ("\n", default N)
//   - Provides the given shellAnswer for the shell integration prompt
func runInitInputs(shellAnswer string) string {
	return "\n\n" + shellAnswer + "\n"
}

func TestRunInit_ShellIntegration_WritesRCFile(t *testing.T) {
	tmp := initTestEnv(t)
	t.Setenv("SHELL", "/bin/zsh")

	// Create the RC file so it's writable.
	rc := filepath.Join(tmp, ".zshrc")
	if err := os.WriteFile(rc, []byte("# existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Answer "y" to the shell integration prompt.
	pipeStdin(t, runInitInputs("y"))
	captureStdout(t, func() {
		if err := runInit(nil, nil); err != nil {
			t.Fatalf("runInit: %v", err)
		}
	})

	data, err := os.ReadFile(rc)
	if err != nil {
		t.Fatalf("read RC: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "mine shell init") {
		t.Errorf("RC file should contain 'mine shell init', got:\n%s", content)
	}
	// Verify the exact snippet was appended once.
	count := strings.Count(content, "mine shell init")
	if count != 1 {
		t.Errorf("expected exactly 1 occurrence of 'mine shell init', got %d", count)
	}
}

func TestRunInit_ShellIntegration_DefaultYes(t *testing.T) {
	tmp := initTestEnv(t)
	t.Setenv("SHELL", "/bin/zsh")

	rc := filepath.Join(tmp, ".zshrc")
	if err := os.WriteFile(rc, []byte("# existing\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Press Enter (empty) at shell integration prompt → default Y.
	pipeStdin(t, runInitInputs(""))
	captureStdout(t, func() {
		if err := runInit(nil, nil); err != nil {
			t.Fatalf("runInit: %v", err)
		}
	})

	data, err := os.ReadFile(rc)
	if err != nil {
		t.Fatalf("read RC: %v", err)
	}
	if !strings.Contains(string(data), "mine shell init") {
		t.Error("RC file should contain 'mine shell init' after default-yes")
	}
}

func TestRunInit_ShellIntegration_AlreadyInstalled_NoDuplicate(t *testing.T) {
	tmp := initTestEnv(t)
	t.Setenv("SHELL", "/bin/zsh")

	// RC file already has the eval line.
	rc := filepath.Join(tmp, ".zshrc")
	existing := "# existing\neval \"$(mine shell init)\"\n"
	if err := os.WriteFile(rc, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	// runShellIntegration should skip entirely — no prompt needed.
	// Still pipe 2 lines for name + openrouter.
	pipeStdin(t, "\n\n")
	captureStdout(t, func() {
		if err := runInit(nil, nil); err != nil {
			t.Fatalf("runInit: %v", err)
		}
	})

	data, err := os.ReadFile(rc)
	if err != nil {
		t.Fatalf("read RC: %v", err)
	}
	count := strings.Count(string(data), "mine shell init")
	if count != 1 {
		t.Errorf("expected exactly 1 occurrence of 'mine shell init', got %d", count)
	}
}

func TestRunInit_ShellIntegration_NonWritableRCFile_NoError(t *testing.T) {
	tmp := initTestEnv(t)
	t.Setenv("SHELL", "/bin/zsh")

	// Create RC file and make it read-only.
	rc := filepath.Join(tmp, ".zshrc")
	if err := os.WriteFile(rc, []byte("# existing\n"), 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(rc, 0o644) }) // restore perms for cleanup

	// Answer "y" — append will fail, but runInit must not return an error.
	pipeStdin(t, runInitInputs("y"))
	captureStdout(t, func() {
		if err := runInit(nil, nil); err != nil {
			t.Fatalf("runInit returned error on non-writable RC: %v", err)
		}
	})
}

func TestRunInit_ShellIntegration_Fish_WritesFishSnippet(t *testing.T) {
	tmp := initTestEnv(t)
	t.Setenv("SHELL", "/usr/bin/fish")

	// Create the fish config directory so the write succeeds.
	fishDir := filepath.Join(tmp, ".config", "fish")
	if err := os.MkdirAll(fishDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Answer "y" to the shell integration prompt.
	pipeStdin(t, runInitInputs("y"))
	captureStdout(t, func() {
		if err := runInit(nil, nil); err != nil {
			t.Fatalf("runInit: %v", err)
		}
	})

	rc := filepath.Join(fishDir, "config.fish")
	data, err := os.ReadFile(rc)
	if err != nil {
		t.Fatalf("read fish RC: %v", err)
	}
	content := string(data)

	// Must contain the fish-compatible syntax.
	if !strings.Contains(content, "mine shell init | source") {
		t.Errorf("fish RC should contain 'mine shell init | source', got:\n%s", content)
	}
	// Must NOT contain bash $(…) syntax.
	if strings.Contains(content, "$(mine shell init)") {
		t.Errorf("fish RC must not contain bash $(...) syntax, got:\n%s", content)
	}
}

func TestRunInit_ShellIntegration_FishConfigDirCreationFails_FallbackToManual(t *testing.T) {
	tmp := initTestEnv(t)
	t.Setenv("SHELL", "/usr/bin/fish")

	// Simulate a failure to create ~/.config/fish by making ~/.config a file.
	configPath := filepath.Join(tmp, ".config")
	if err := os.WriteFile(configPath, []byte("not a directory"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Answer "y" to shell integration. Directory creation for fish should fail,
	// but runInit must still succeed and fall back to manual instructions.
	pipeStdin(t, runInitInputs("y"))
	out := captureStdout(t, func() {
		if err := runInit(nil, nil); err != nil {
			t.Fatalf("runInit returned error when fish config dir creation failed: %v", err)
		}
	})

	// Fallback manual instructions should still mention how to run shell init.
	if !strings.Contains(out, "mine shell init") {
		t.Error("expected manual instructions containing 'mine shell init' when fish config dir creation fails")
	}
}

// ---- re-init and --reset tests ----

// preConfigureInit runs a silent fresh init to create an existing config.
func preConfigureInit(t *testing.T, name, provider, model string) {
	t.Helper()
	cfg := &config.Config{}
	cfg.User.Name = name
	cfg.AI.Provider = provider
	cfg.AI.Model = model
	cfg.Shell.DefaultShell = "/bin/bash"
	cfg.Analytics.Enabled = config.BoolPtr(true)
	if err := config.Save(cfg); err != nil {
		t.Fatalf("preConfigureInit save: %v", err)
	}
}

func TestRunInit_ExistingConfig_ShowsCurrentSettings(t *testing.T) {
	runInitEnv(t)
	plainDir := t.TempDir()
	t.Chdir(plainDir)

	// Set up existing config.
	preConfigureInit(t, "Alice", "claude", "claude-sonnet-4-5-20250929")

	// Answer "n" to "Update your configuration?" — no changes wanted.
	reader := makeInitStdin("n")
	out := captureStdout(t, func() {
		if err := runInitWithReader(reader, false); err != nil {
			t.Errorf("runInitWithReader: %v", err)
		}
	})

	// Should show existing settings summary.
	if !strings.Contains(out, "mine is already set up") {
		t.Errorf("expected 'mine is already set up' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Alice") {
		t.Errorf("expected name 'Alice' in current config summary, got:\n%s", out)
	}
	if !strings.Contains(out, "claude") {
		t.Errorf("expected 'claude' in current config summary, got:\n%s", out)
	}
}

func TestRunInit_ExistingConfig_DenyUpdate_NoChanges(t *testing.T) {
	runInitEnv(t)
	plainDir := t.TempDir()
	t.Chdir(plainDir)

	preConfigureInit(t, "Alice", "claude", "claude-sonnet-4-5-20250929")

	// Deny the update prompt.
	reader := makeInitStdin("n")
	captureStdout(t, func() {
		if err := runInitWithReader(reader, false); err != nil {
			t.Errorf("runInitWithReader: %v", err)
		}
	})

	// Config should be unchanged.
	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if loaded.User.Name != "Alice" {
		t.Errorf("expected name 'Alice' after denied update, got %q", loaded.User.Name)
	}
}

func TestRunInit_ExistingConfig_AcceptDefaults_PreservesName(t *testing.T) {
	runInitEnv(t)
	plainDir := t.TempDir()
	t.Chdir(plainDir)

	preConfigureInit(t, "Alice", "claude", "claude-sonnet-4-5-20250929")

	// Accept update, then press Enter on all prompts (keep current values).
	// Inputs: "y" (update?), "" (name=Alice), "" (model=current)
	// No keys detected so re-init simple AI path: "" (provider), "" (model)
	reader := makeInitStdin("y", "", "", "")
	captureStdout(t, func() {
		if err := runInitWithReader(reader, false); err != nil {
			t.Errorf("runInitWithReader: %v", err)
		}
	})

	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if loaded.User.Name != "Alice" {
		t.Errorf("expected name 'Alice' after pressing Enter, got %q", loaded.User.Name)
	}
}

func TestRunInit_ExistingConfig_AcceptUpdate_SaysConfigurationUpdated(t *testing.T) {
	runInitEnv(t)
	plainDir := t.TempDir()
	t.Chdir(plainDir)

	preConfigureInit(t, "Alice", "claude", "claude-sonnet-4-5-20250929")

	// Accept update, keep all defaults.
	reader := makeInitStdin("y", "", "", "")
	out := captureStdout(t, func() {
		if err := runInitWithReader(reader, false); err != nil {
			t.Errorf("runInitWithReader: %v", err)
		}
	})

	if !strings.Contains(out, "Configuration updated") {
		t.Errorf("expected 'Configuration updated' in output after re-init, got:\n%s", out)
	}
	// Should NOT say "All set!" on re-init.
	if strings.Contains(out, "All set") {
		t.Errorf("expected 'All set!' to be absent on re-init, got:\n%s", out)
	}
}

func TestRunInit_ExistingConfig_PreservesAnalytics(t *testing.T) {
	runInitEnv(t)
	plainDir := t.TempDir()
	t.Chdir(plainDir)

	// Set analytics to explicitly disabled.
	cfg := &config.Config{}
	cfg.User.Name = "Alice"
	cfg.AI.Provider = "claude"
	cfg.AI.Model = "claude-sonnet-4-5-20250929"
	cfg.Shell.DefaultShell = "/bin/bash"
	cfg.Analytics.Enabled = config.BoolPtr(false)
	if err := config.Save(cfg); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Re-init accepting all defaults.
	reader := makeInitStdin("y", "", "", "")
	captureStdout(t, func() {
		if err := runInitWithReader(reader, false); err != nil {
			t.Errorf("runInitWithReader: %v", err)
		}
	})

	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if loaded.Analytics.Enabled == nil || *loaded.Analytics.Enabled != false {
		t.Errorf("expected analytics.enabled=false to be preserved after re-init")
	}
}

func TestRunInit_ExistingConfig_PreservesNonPromptedFields(t *testing.T) {
	runInitEnv(t)
	t.Setenv("SHELL", "") // suppress shell integration prompt for determinism
	plainDir := t.TempDir()
	t.Chdir(plainDir)

	// Seed a config with fields that are NOT surfaced in init prompts.
	cfg := &config.Config{}
	cfg.User.Name = "Alice"
	cfg.User.Email = "alice@example.com"
	cfg.AI.Provider = "claude"
	cfg.AI.Model = "claude-sonnet-4-5-20250929"
	cfg.AI.SystemInstructions = "You are a helpful assistant."
	cfg.AI.AskSystemInstructions = "Answer precisely and concisely."
	cfg.Shell.DefaultShell = "/bin/zsh"
	cfg.Shell.Aliases = []string{"ll=ls -la", "g=git"}
	cfg.Analytics.Enabled = config.BoolPtr(true)
	if err := config.Save(cfg); err != nil {
		t.Fatalf("save: %v", err)
	}

	// Re-init accepting all defaults: update=y, name=Enter, provider=Enter, model=Enter.
	reader := makeInitStdin("y", "", "", "")
	captureStdout(t, func() {
		if err := runInitWithReader(reader, false); err != nil {
			t.Errorf("runInitWithReader: %v", err)
		}
	})

	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}

	// Non-prompted fields must be preserved.
	if loaded.User.Email != "alice@example.com" {
		t.Errorf("user.email: want %q, got %q", "alice@example.com", loaded.User.Email)
	}
	if loaded.AI.SystemInstructions != "You are a helpful assistant." {
		t.Errorf("ai.system_instructions: want %q, got %q",
			"You are a helpful assistant.", loaded.AI.SystemInstructions)
	}
	if loaded.AI.AskSystemInstructions != "Answer precisely and concisely." {
		t.Errorf("ai.ask_system_instructions: want %q, got %q",
			"Answer precisely and concisely.", loaded.AI.AskSystemInstructions)
	}
	if len(loaded.Shell.Aliases) != 2 {
		t.Errorf("shell.aliases: want len 2, got %v", loaded.Shell.Aliases)
	}
	if loaded.Shell.DefaultShell != "/bin/zsh" {
		t.Errorf("shell.default_shell: want %q, got %q", "/bin/zsh", loaded.Shell.DefaultShell)
	}
	// Prompted fields should still reflect accepted defaults.
	if loaded.User.Name != "Alice" {
		t.Errorf("user.name: want %q, got %q", "Alice", loaded.User.Name)
	}
}

func TestRunInit_Reset_Confirmed_ReplacesConfig(t *testing.T) {
	runInitEnv(t)
	plainDir := t.TempDir()
	t.Chdir(plainDir)

	preConfigureInit(t, "Alice", "claude", "claude-sonnet-4-5-20250929")

	// Confirm reset, then provide new name "Bob", skip AI/shell.
	reader := makeInitStdin("y", "Bob", "n")
	out := captureStdout(t, func() {
		if err := runInitWithReader(reader, true); err != nil {
			t.Errorf("runInitWithReader --reset: %v", err)
		}
	})

	// Should show fresh init output.
	if !strings.Contains(out, "Welcome to mine") {
		t.Errorf("expected 'Welcome to mine' in reset output, got:\n%s", out)
	}

	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if loaded.User.Name != "Bob" {
		t.Errorf("expected name 'Bob' after reset, got %q", loaded.User.Name)
	}
}

func TestRunInit_Reset_Denied_ConfigUnchanged(t *testing.T) {
	runInitEnv(t)
	plainDir := t.TempDir()
	t.Chdir(plainDir)

	preConfigureInit(t, "Alice", "claude", "claude-sonnet-4-5-20250929")

	// Deny reset confirmation.
	reader := makeInitStdin("n")
	captureStdout(t, func() {
		if err := runInitWithReader(reader, true); err != nil {
			t.Errorf("runInitWithReader --reset: %v", err)
		}
	})

	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	if loaded.User.Name != "Alice" {
		t.Errorf("expected name 'Alice' unchanged after denied reset, got %q", loaded.User.Name)
	}
}

func TestRunInit_Reset_DatabaseUntouched(t *testing.T) {
	runInitEnv(t)
	plainDir := t.TempDir()
	t.Chdir(plainDir)

	// First fresh init to create DB.
	reader := makeInitStdin("", "n")
	captureStdout(t, func() {
		if err := runInitWithReader(reader, false); err != nil {
			t.Fatalf("first init: %v", err)
		}
	})

	// Add a todo to the DB to verify it survives reset.
	db, err := store.Open()
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	_, err = db.Conn().Exec("INSERT INTO todos (title) VALUES ('survive-reset')")
	if err != nil {
		t.Fatalf("insert todo: %v", err)
	}
	db.Close()

	// Reset confirmed.
	reader2 := makeInitStdin("y", "", "n")
	captureStdout(t, func() {
		if err := runInitWithReader(reader2, true); err != nil {
			t.Errorf("runInitWithReader --reset: %v", err)
		}
	})

	// DB should still have our todo.
	db2, err := store.Open()
	if err != nil {
		t.Fatalf("store.Open after reset: %v", err)
	}
	defer db2.Close()

	var count int
	row := db2.Conn().QueryRow("SELECT COUNT(*) FROM todos WHERE title='survive-reset'")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("query: %v", err)
	}
	if count != 1 {
		t.Errorf("expected todo to survive --reset, got count=%d", count)
	}
}
