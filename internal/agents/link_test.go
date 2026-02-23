package agents

import (
	"os"
	"path/filepath"
	"testing"
)

// setupLinkEnv initializes a temp agents store and home dir for link tests.
// Returns (agentsDir, homeDir).
func setupLinkEnv(t *testing.T) (string, string) {
	t.Helper()
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, "data")
	homeDir := filepath.Join(tmpDir, "home")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_DATA_HOME", dataDir)
	t.Setenv("HOME", homeDir)

	// Initialize agents store.
	if err := Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	return Dir(), homeDir
}

// writeStoreFile creates a file in the canonical store at relPath.
func writeStoreFile(t *testing.T, storeDir, relPath, content string) {
	t.Helper()
	p := filepath.Join(storeDir, relPath)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("writing store file %s: %v", relPath, err)
	}
}

// makeDetectedAgent adds a detected agent to the manifest.
func makeDetectedAgent(t *testing.T, name, configDir string) {
	t.Helper()
	m, err := ReadManifest()
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	m.Agents = append(m.Agents, Agent{
		Name:      name,
		Detected:  true,
		ConfigDir: configDir,
	})
	if err := WriteManifest(m); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}
}

// --- buildLinkRegistry ---

func TestBuildLinkRegistry_ContainsAllAgents(t *testing.T) {
	specs := buildLinkRegistry("/home/testuser")
	wantNames := []string{"claude", "codex", "gemini", "opencode"}

	if len(specs) != len(wantNames) {
		t.Fatalf("buildLinkRegistry() returned %d specs, want %d", len(specs), len(wantNames))
	}

	nameSet := make(map[string]bool)
	for _, s := range specs {
		nameSet[s.Name] = true
	}
	for _, name := range wantNames {
		if !nameSet[name] {
			t.Errorf("registry missing agent %q", name)
		}
	}
}

func TestBuildLinkRegistry_ClaudeHasCommandsAndMCP(t *testing.T) {
	specs := buildLinkRegistry("/home/testuser")
	for _, s := range specs {
		if s.Name == "claude" {
			if s.CommandsDir == "" {
				t.Error("claude.CommandsDir is empty, want non-empty")
			}
			if s.MCPConfigPath == "" {
				t.Error("claude.MCPConfigPath is empty, want non-empty")
			}
			return
		}
	}
	t.Fatal("claude spec not found in registry")
}

func TestBuildLinkRegistry_InstructionFilenames(t *testing.T) {
	home := "/home/testuser"
	specs := buildLinkRegistry(home)

	wantFilenames := map[string]string{
		"claude":   "CLAUDE.md",
		"codex":    "AGENTS.md",
		"gemini":   "GEMINI.md",
		"opencode": "AGENTS.md",
	}

	for _, s := range specs {
		want, ok := wantFilenames[s.Name]
		if !ok {
			continue
		}
		if s.InstructionFilename != want {
			t.Errorf("agent %q InstructionFilename = %q, want %q", s.Name, s.InstructionFilename, want)
		}
	}
}

// --- checkFileSafety ---

func TestCheckFileSafety_TargetMissing(t *testing.T) {
	target := filepath.Join(t.TempDir(), "nonexistent.md")
	existed, alreadyLinked, err := checkFileSafety("/some/source", target, false)
	if err != nil {
		t.Errorf("checkFileSafety() error = %v, want nil for missing target", err)
	}
	if existed {
		t.Error("checkFileSafety() existed = true, want false for missing target")
	}
	if alreadyLinked {
		t.Error("checkFileSafety() alreadyLinked = true, want false for missing target")
	}
}

func TestCheckFileSafety_ExistingRegularFile_NoForce(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "existing.md")
	if err := os.WriteFile(target, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	existed, _, err := checkFileSafety("/some/source", target, false)
	if err == nil {
		t.Error("checkFileSafety() error = nil, want error for existing regular file without --force")
	}
	if !existed {
		t.Error("checkFileSafety() existed = false, want true for existing file")
	}
}

func TestCheckFileSafety_ExistingRegularFile_WithForce(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "existing.md")
	if err := os.WriteFile(target, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	existed, alreadyLinked, err := checkFileSafety("/some/source", target, true)
	if err != nil {
		t.Errorf("checkFileSafety() with force error = %v, want nil", err)
	}
	if !existed {
		t.Error("checkFileSafety() existed = false, want true")
	}
	if alreadyLinked {
		t.Error("checkFileSafety() alreadyLinked = true, want false for regular file")
	}
}

func TestCheckFileSafety_SymlinkToOurStore(t *testing.T) {
	tmp := t.TempDir()
	sourcePath := filepath.Join(tmp, "source.md")
	target := filepath.Join(tmp, "target.md")
	if err := os.WriteFile(sourcePath, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(sourcePath, target); err != nil {
		t.Fatal(err)
	}

	existed, alreadyLinked, err := checkFileSafety(sourcePath, target, false)
	if err != nil {
		t.Errorf("checkFileSafety() error = %v, want nil for symlink to our store", err)
	}
	if !existed {
		t.Error("checkFileSafety() existed = false, want true")
	}
	if !alreadyLinked {
		t.Error("checkFileSafety() alreadyLinked = false, want true for symlink to our store")
	}
}

func TestCheckFileSafety_SymlinkElsewhere_NoForce(t *testing.T) {
	tmp := t.TempDir()
	otherTarget := filepath.Join(tmp, "other.md")
	target := filepath.Join(tmp, "target.md")
	if err := os.WriteFile(otherTarget, []byte("other"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(otherTarget, target); err != nil {
		t.Fatal(err)
	}

	_, _, err := checkFileSafety("/some/canonical/source", target, false)
	if err == nil {
		t.Error("checkFileSafety() error = nil, want error for symlink pointing elsewhere without --force")
	}
}

func TestCheckFileSafety_SymlinkElsewhere_WithForce(t *testing.T) {
	tmp := t.TempDir()
	otherTarget := filepath.Join(tmp, "other.md")
	target := filepath.Join(tmp, "target.md")
	if err := os.WriteFile(otherTarget, []byte("other"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(otherTarget, target); err != nil {
		t.Fatal(err)
	}

	existed, alreadyLinked, err := checkFileSafety("/some/canonical/source", target, true)
	if err != nil {
		t.Errorf("checkFileSafety() with force error = %v, want nil", err)
	}
	if !existed {
		t.Error("checkFileSafety() existed = false, want true")
	}
	if alreadyLinked {
		t.Error("checkFileSafety() alreadyLinked = true, want false for symlink to different path")
	}
}

// --- checkDirSafety ---

func TestCheckDirSafety_TargetMissing(t *testing.T) {
	target := filepath.Join(t.TempDir(), "nonexistent")
	existed, alreadyLinked, err := checkDirSafety("/some/source", target, false)
	if err != nil {
		t.Errorf("checkDirSafety() error = %v, want nil for missing target", err)
	}
	if existed || alreadyLinked {
		t.Error("checkDirSafety() returned existed/alreadyLinked=true for missing target")
	}
}

func TestCheckDirSafety_ExistingDir_NoForce(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "existing")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}

	_, _, err := checkDirSafety("/some/source", target, false)
	if err == nil {
		t.Error("checkDirSafety() error = nil, want error for existing dir without --force")
	}
}

func TestCheckDirSafety_ExistingDir_WithForce(t *testing.T) {
	tmp := t.TempDir()
	target := filepath.Join(tmp, "existing")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}

	existed, alreadyLinked, err := checkDirSafety("/some/source", target, true)
	if err != nil {
		t.Errorf("checkDirSafety() with force error = %v, want nil", err)
	}
	if !existed {
		t.Error("checkDirSafety() existed = false, want true")
	}
	if alreadyLinked {
		t.Error("checkDirSafety() alreadyLinked = true, want false for regular dir")
	}
}

func TestCheckDirSafety_SymlinkToOurStore(t *testing.T) {
	tmp := t.TempDir()
	sourcePath := filepath.Join(tmp, "source")
	target := filepath.Join(tmp, "target")
	if err := os.MkdirAll(sourcePath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(sourcePath, target); err != nil {
		t.Fatal(err)
	}

	existed, alreadyLinked, err := checkDirSafety(sourcePath, target, false)
	if err != nil {
		t.Errorf("checkDirSafety() error = %v, want nil for symlink to our store", err)
	}
	if !existed || !alreadyLinked {
		t.Error("checkDirSafety() expected existed=true, alreadyLinked=true for symlink to our store")
	}
}

// --- upsertManifestLink ---

func TestUpsertManifestLink_AddsNewEntry(t *testing.T) {
	m := &Manifest{Links: []LinkEntry{}}
	upsertManifestLink(m, "instructions/AGENTS.md", "/target/CLAUDE.md", "claude", "symlink")

	if len(m.Links) != 1 {
		t.Fatalf("Links count = %d, want 1", len(m.Links))
	}
	if m.Links[0].Source != "instructions/AGENTS.md" {
		t.Errorf("Source = %q, want %q", m.Links[0].Source, "instructions/AGENTS.md")
	}
	if m.Links[0].Agent != "claude" {
		t.Errorf("Agent = %q, want %q", m.Links[0].Agent, "claude")
	}
}

func TestUpsertManifestLink_UpdatesExistingEntry(t *testing.T) {
	m := &Manifest{
		Links: []LinkEntry{
			{Source: "instructions/AGENTS.md", Target: "/target/CLAUDE.md", Agent: "claude", Mode: "symlink"},
		},
	}
	upsertManifestLink(m, "instructions/AGENTS.md", "/target/CLAUDE.md", "claude", "copy")

	if len(m.Links) != 1 {
		t.Fatalf("Links count = %d after update, want 1 (no duplication)", len(m.Links))
	}
	if m.Links[0].Mode != "copy" {
		t.Errorf("Mode = %q after update, want %q", m.Links[0].Mode, "copy")
	}
}

// --- fileExists ---

func TestFileExists_RegularFile(t *testing.T) {
	tmp := t.TempDir()
	p := filepath.Join(tmp, "file.txt")
	if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !fileExists(p) {
		t.Error("fileExists() = false for existing regular file, want true")
	}
}

func TestFileExists_Directory(t *testing.T) {
	if fileExists(t.TempDir()) {
		t.Error("fileExists() = true for directory, want false")
	}
}

func TestFileExists_Missing(t *testing.T) {
	if fileExists(filepath.Join(t.TempDir(), "nonexistent")) {
		t.Error("fileExists() = true for missing path, want false")
	}
}

// --- dirNonEmpty ---

func TestDirNonEmpty_EmptyDir(t *testing.T) {
	if dirNonEmpty(t.TempDir()) {
		t.Error("dirNonEmpty() = true for empty dir, want false")
	}
}

func TestDirNonEmpty_NonEmptyDir(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "file.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !dirNonEmpty(tmp) {
		t.Error("dirNonEmpty() = false for non-empty dir, want true")
	}
}

func TestDirNonEmpty_Missing(t *testing.T) {
	if dirNonEmpty(filepath.Join(t.TempDir(), "nonexistent")) {
		t.Error("dirNonEmpty() = true for missing path, want false")
	}
}

// --- Link ---

func TestLink_NotInitialized(t *testing.T) {
	setupEnv(t) // fresh XDG env but no Init()
	_, err := Link(LinkOptions{})
	if err == nil {
		t.Error("Link() error = nil for uninitialized store, want error")
	}
}

func TestLink_NoDetectedAgents_ReturnsEmpty(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}
	// Manifest has no detected agents.
	actions, err := Link(LinkOptions{})
	if err != nil {
		t.Fatalf("Link() error = %v", err)
	}
	if len(actions) != 0 {
		t.Errorf("Link() returned %d actions, want 0 (no detected agents)", len(actions))
	}
}

func TestLink_CreatesInstructionSymlink(t *testing.T) {
	storeDir, homeDir := setupLinkEnv(t)

	// Write instructions file in store.
	writeStoreFile(t, storeDir, "instructions/AGENTS.md", "# Instructions\n")

	// Register claude as detected.
	claudeConfigDir := filepath.Join(homeDir, ".claude")
	makeDetectedAgent(t, "claude", claudeConfigDir)

	actions, err := Link(LinkOptions{})
	if err != nil {
		t.Fatalf("Link() error = %v", err)
	}

	// Find the instruction link action.
	var instrAction *LinkAction
	for i := range actions {
		if actions[i].Source == "instructions/AGENTS.md" && actions[i].Agent == "claude" {
			instrAction = &actions[i]
			break
		}
	}
	if instrAction == nil {
		t.Fatal("no action for instructions/AGENTS.md for claude")
	}
	if instrAction.Err != nil {
		t.Errorf("action.Err = %v, want nil", instrAction.Err)
	}
	if instrAction.Status != "created" {
		t.Errorf("action.Status = %q, want %q", instrAction.Status, "created")
	}

	// Verify symlink exists and points to store.
	target := filepath.Join(claudeConfigDir, "CLAUDE.md")
	info, err := os.Lstat(target)
	if err != nil {
		t.Fatalf("Lstat(%q) error = %v, want symlink to exist", target, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Errorf("target %q is not a symlink", target)
	}

	dest, err := os.Readlink(target)
	if err != nil {
		t.Fatal(err)
	}
	expected := filepath.Join(storeDir, "instructions", "AGENTS.md")
	if dest != expected {
		t.Errorf("symlink points to %q, want %q", dest, expected)
	}
}

func TestLink_CopyMode_CreatesFileCopy(t *testing.T) {
	storeDir, homeDir := setupLinkEnv(t)

	writeStoreFile(t, storeDir, "instructions/AGENTS.md", "# Instructions\n")

	claudeConfigDir := filepath.Join(homeDir, ".claude")
	makeDetectedAgent(t, "claude", claudeConfigDir)

	actions, err := Link(LinkOptions{Copy: true})
	if err != nil {
		t.Fatalf("Link() with Copy=true error = %v", err)
	}

	var instrAction *LinkAction
	for i := range actions {
		if actions[i].Source == "instructions/AGENTS.md" && actions[i].Agent == "claude" {
			instrAction = &actions[i]
			break
		}
	}
	if instrAction == nil {
		t.Fatal("no instruction action for claude")
	}
	if instrAction.Mode != "copy" {
		t.Errorf("action.Mode = %q, want %q", instrAction.Mode, "copy")
	}
	if instrAction.Err != nil {
		t.Errorf("action.Err = %v, want nil", instrAction.Err)
	}

	// Must be a regular file, not a symlink.
	target := filepath.Join(claudeConfigDir, "CLAUDE.md")
	info, err := os.Lstat(target)
	if err != nil {
		t.Fatalf("Lstat target: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("target is a symlink, want regular file when --copy is used")
	}
}

func TestLink_AgentFilter_OnlyLinksSpecifiedAgent(t *testing.T) {
	storeDir, homeDir := setupLinkEnv(t)

	writeStoreFile(t, storeDir, "instructions/AGENTS.md", "# Instructions\n")

	claudeConfigDir := filepath.Join(homeDir, ".claude")
	codexConfigDir := filepath.Join(homeDir, ".codex")
	makeDetectedAgent(t, "claude", claudeConfigDir)
	makeDetectedAgent(t, "codex", codexConfigDir)

	actions, err := Link(LinkOptions{Agent: "claude"})
	if err != nil {
		t.Fatalf("Link() error = %v", err)
	}

	for _, a := range actions {
		if a.Agent != "claude" {
			t.Errorf("action for agent %q found, want only claude", a.Agent)
		}
	}
}

func TestLink_SkipsEmptyStore(t *testing.T) {
	storeDir, homeDir := setupLinkEnv(t)

	// Remove the starter AGENTS.md so the store has no linkable content.
	starterFile := filepath.Join(storeDir, "instructions", "AGENTS.md")
	if err := os.Remove(starterFile); err != nil {
		t.Fatalf("removing starter AGENTS.md: %v", err)
	}

	claudeConfigDir := filepath.Join(homeDir, ".claude")
	makeDetectedAgent(t, "claude", claudeConfigDir)

	actions, err := Link(LinkOptions{})
	if err != nil {
		t.Fatalf("Link() error = %v", err)
	}

	// No actions — nothing in the store to link.
	if len(actions) != 0 {
		t.Errorf("Link() returned %d actions for empty store, want 0", len(actions))
	}
}

func TestLink_ExistingRegularFile_SafetyRefusal(t *testing.T) {
	storeDir, homeDir := setupLinkEnv(t)

	writeStoreFile(t, storeDir, "instructions/AGENTS.md", "# Instructions\n")

	claudeConfigDir := filepath.Join(homeDir, ".claude")
	if err := os.MkdirAll(claudeConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Pre-existing regular file at the target location.
	existingFile := filepath.Join(claudeConfigDir, "CLAUDE.md")
	if err := os.WriteFile(existingFile, []byte("existing content"), 0o644); err != nil {
		t.Fatal(err)
	}

	makeDetectedAgent(t, "claude", claudeConfigDir)

	actions, err := Link(LinkOptions{})
	if err != nil {
		t.Fatalf("Link() error = %v", err)
	}

	var instrAction *LinkAction
	for i := range actions {
		if actions[i].Source == "instructions/AGENTS.md" {
			instrAction = &actions[i]
			break
		}
	}
	if instrAction == nil {
		t.Fatal("no action for instructions/AGENTS.md")
	}
	if instrAction.Err == nil {
		t.Error("action.Err = nil, want error when target is existing regular file without --force")
	}
	if instrAction.Status != "skipped" {
		t.Errorf("action.Status = %q, want %q", instrAction.Status, "skipped")
	}

	// Verify file was NOT overwritten.
	data, err := os.ReadFile(existingFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "existing content" {
		t.Error("existing file was overwritten without --force, want original content preserved")
	}
}

func TestLink_ExistingRegularFile_ForceOverwrites(t *testing.T) {
	storeDir, homeDir := setupLinkEnv(t)

	writeStoreFile(t, storeDir, "instructions/AGENTS.md", "# Instructions\n")

	claudeConfigDir := filepath.Join(homeDir, ".claude")
	if err := os.MkdirAll(claudeConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}

	existingFile := filepath.Join(claudeConfigDir, "CLAUDE.md")
	if err := os.WriteFile(existingFile, []byte("existing content"), 0o644); err != nil {
		t.Fatal(err)
	}

	makeDetectedAgent(t, "claude", claudeConfigDir)

	actions, err := Link(LinkOptions{Force: true})
	if err != nil {
		t.Fatalf("Link() with Force=true error = %v", err)
	}

	var instrAction *LinkAction
	for i := range actions {
		if actions[i].Source == "instructions/AGENTS.md" {
			instrAction = &actions[i]
			break
		}
	}
	if instrAction == nil {
		t.Fatal("no action for instructions/AGENTS.md")
	}
	if instrAction.Err != nil {
		t.Errorf("action.Err = %v, want nil with --force", instrAction.Err)
	}

	// Verify target is now a symlink.
	info, err := os.Lstat(existingFile)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("target is not a symlink after --force link, want symlink")
	}
}

func TestLink_ExistingSymlinkToOurStore_UpdatedSilently(t *testing.T) {
	storeDir, homeDir := setupLinkEnv(t)

	writeStoreFile(t, storeDir, "instructions/AGENTS.md", "# Instructions\n")

	claudeConfigDir := filepath.Join(homeDir, ".claude")
	if err := os.MkdirAll(claudeConfigDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Pre-create symlink pointing to our store.
	instrSource := filepath.Join(storeDir, "instructions", "AGENTS.md")
	target := filepath.Join(claudeConfigDir, "CLAUDE.md")
	if err := os.Symlink(instrSource, target); err != nil {
		t.Fatal(err)
	}

	makeDetectedAgent(t, "claude", claudeConfigDir)

	actions, err := Link(LinkOptions{})
	if err != nil {
		t.Fatalf("Link() error = %v", err)
	}

	var instrAction *LinkAction
	for i := range actions {
		if actions[i].Source == "instructions/AGENTS.md" {
			instrAction = &actions[i]
			break
		}
	}
	if instrAction == nil {
		t.Fatal("no action for instructions/AGENTS.md")
	}
	if instrAction.Err != nil {
		t.Errorf("action.Err = %v, want nil for existing symlink to our store", instrAction.Err)
	}
	if instrAction.Status != "updated" {
		t.Errorf("action.Status = %q, want %q", instrAction.Status, "updated")
	}
}

func TestLink_CreatesParentDirectories(t *testing.T) {
	storeDir, homeDir := setupLinkEnv(t)

	writeStoreFile(t, storeDir, "instructions/AGENTS.md", "# Instructions\n")

	// Config dir does NOT exist yet — link should create it.
	claudeConfigDir := filepath.Join(homeDir, ".claude")
	makeDetectedAgent(t, "claude", claudeConfigDir)

	actions, err := Link(LinkOptions{})
	if err != nil {
		t.Fatalf("Link() error = %v", err)
	}

	var instrAction *LinkAction
	for i := range actions {
		if actions[i].Source == "instructions/AGENTS.md" {
			instrAction = &actions[i]
			break
		}
	}
	if instrAction == nil {
		t.Fatal("no action for instructions/AGENTS.md")
	}
	if instrAction.Err != nil {
		t.Errorf("action.Err = %v, want nil (parent dir should be created)", instrAction.Err)
	}

	// Parent directory must now exist.
	if _, err := os.Stat(claudeConfigDir); err != nil {
		t.Errorf("claude config dir not created: %v", err)
	}
}

func TestLink_PersistsToManifest(t *testing.T) {
	storeDir, homeDir := setupLinkEnv(t)

	writeStoreFile(t, storeDir, "instructions/AGENTS.md", "# Instructions\n")

	claudeConfigDir := filepath.Join(homeDir, ".claude")
	makeDetectedAgent(t, "claude", claudeConfigDir)

	if _, err := Link(LinkOptions{}); err != nil {
		t.Fatalf("Link() error = %v", err)
	}

	m, err := ReadManifest()
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}

	if len(m.Links) == 0 {
		t.Error("manifest.Links is empty after Link(), want at least one entry")
	}

	var found bool
	for _, l := range m.Links {
		if l.Agent == "claude" && l.Source == "instructions/AGENTS.md" {
			found = true
			if l.Mode != "symlink" {
				t.Errorf("link mode = %q, want %q", l.Mode, "symlink")
			}
			break
		}
	}
	if !found {
		t.Error("claude instruction link not found in manifest after Link()")
	}
}

func TestLink_DirectorySymlink(t *testing.T) {
	storeDir, homeDir := setupLinkEnv(t)

	// Write a file in the store's skills/ directory.
	writeStoreFile(t, storeDir, "skills/my-skill.md", "# My Skill\n")

	claudeConfigDir := filepath.Join(homeDir, ".claude")
	makeDetectedAgent(t, "claude", claudeConfigDir)

	actions, err := Link(LinkOptions{})
	if err != nil {
		t.Fatalf("Link() error = %v", err)
	}

	var skillsAction *LinkAction
	for i := range actions {
		if actions[i].Source == "skills" && actions[i].Agent == "claude" {
			skillsAction = &actions[i]
			break
		}
	}
	if skillsAction == nil {
		t.Fatal("no action for skills dir for claude")
	}
	if skillsAction.Err != nil {
		t.Errorf("skills action.Err = %v, want nil", skillsAction.Err)
	}

	// Verify skills dir symlink.
	target := filepath.Join(claudeConfigDir, "skills")
	info, err := os.Lstat(target)
	if err != nil {
		t.Fatalf("Lstat skills target: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("skills target is not a symlink, want symlink")
	}
}

// --- Unlink ---

func TestUnlink_NotInitialized(t *testing.T) {
	setupEnv(t)
	_, err := Unlink(UnlinkOptions{})
	if err == nil {
		t.Error("Unlink() error = nil for uninitialized store, want error")
	}
}

func TestUnlink_NoLinks_ReturnsEmpty(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	actions, err := Unlink(UnlinkOptions{})
	if err != nil {
		t.Fatalf("Unlink() error = %v", err)
	}
	if len(actions) != 0 {
		t.Errorf("Unlink() returned %d actions, want 0 (no links)", len(actions))
	}
}

func TestUnlink_ReplacesSymlinkWithStandaloneFile(t *testing.T) {
	storeDir, homeDir := setupLinkEnv(t)

	content := "# My Instructions\n"
	writeStoreFile(t, storeDir, "instructions/AGENTS.md", content)

	claudeConfigDir := filepath.Join(homeDir, ".claude")
	makeDetectedAgent(t, "claude", claudeConfigDir)

	// Link first.
	if _, err := Link(LinkOptions{}); err != nil {
		t.Fatalf("Link() error = %v", err)
	}

	target := filepath.Join(claudeConfigDir, "CLAUDE.md")

	// Verify it's a symlink.
	info, _ := os.Lstat(target)
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("target is not a symlink after Link()")
	}

	// Now unlink.
	actions, err := Unlink(UnlinkOptions{})
	if err != nil {
		t.Fatalf("Unlink() error = %v", err)
	}

	if len(actions) == 0 {
		t.Fatal("Unlink() returned no actions, want at least one")
	}

	var ua *UnlinkAction
	for i := range actions {
		if actions[i].Target == target {
			ua = &actions[i]
			break
		}
	}
	if ua == nil {
		t.Fatalf("no unlink action for target %q", target)
	}
	if ua.Err != nil {
		t.Errorf("unlink action.Err = %v, want nil", ua.Err)
	}
	if ua.Status != "unlinked" {
		t.Errorf("unlink action.Status = %q, want %q", ua.Status, "unlinked")
	}

	// Target must now be a regular file with original content.
	info, err = os.Lstat(target)
	if err != nil {
		t.Fatalf("Lstat target after unlink: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("target is still a symlink after Unlink(), want regular file")
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading target after unlink: %v", err)
	}
	if string(data) != content {
		t.Errorf("target content after unlink = %q, want %q", string(data), content)
	}
}

func TestUnlink_RemovesFromManifest(t *testing.T) {
	storeDir, homeDir := setupLinkEnv(t)

	writeStoreFile(t, storeDir, "instructions/AGENTS.md", "# Instructions\n")

	claudeConfigDir := filepath.Join(homeDir, ".claude")
	makeDetectedAgent(t, "claude", claudeConfigDir)

	if _, err := Link(LinkOptions{}); err != nil {
		t.Fatalf("Link() error = %v", err)
	}

	// Verify link is in manifest.
	m, _ := ReadManifest()
	if len(m.Links) == 0 {
		t.Fatal("expected links in manifest before Unlink()")
	}

	if _, err := Unlink(UnlinkOptions{}); err != nil {
		t.Fatalf("Unlink() error = %v", err)
	}

	// Verify manifest has no links.
	m, err := ReadManifest()
	if err != nil {
		t.Fatalf("ReadManifest after unlink: %v", err)
	}
	if len(m.Links) != 0 {
		t.Errorf("manifest.Links count = %d after Unlink(), want 0", len(m.Links))
	}
}

func TestUnlink_AgentFilter(t *testing.T) {
	storeDir, homeDir := setupLinkEnv(t)

	writeStoreFile(t, storeDir, "instructions/AGENTS.md", "# Instructions\n")

	claudeConfigDir := filepath.Join(homeDir, ".claude")
	codexConfigDir := filepath.Join(homeDir, ".codex")
	makeDetectedAgent(t, "claude", claudeConfigDir)
	makeDetectedAgent(t, "codex", codexConfigDir)

	if _, err := Link(LinkOptions{}); err != nil {
		t.Fatalf("Link() error = %v", err)
	}

	// Verify both agents have links.
	m, _ := ReadManifest()
	initialLinks := len(m.Links)
	if initialLinks < 2 {
		t.Fatalf("expected at least 2 links before unlink, got %d", initialLinks)
	}

	// Unlink only claude.
	actions, err := Unlink(UnlinkOptions{Agent: "claude"})
	if err != nil {
		t.Fatalf("Unlink() error = %v", err)
	}

	// All unlink actions must be for claude.
	for _, a := range actions {
		if a.Agent != "claude" {
			t.Errorf("unlink action for agent %q, want only claude", a.Agent)
		}
	}

	// Codex links must remain in manifest.
	m, err = ReadManifest()
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range m.Links {
		if l.Agent == "claude" {
			t.Error("claude link still in manifest after unlink --agent claude")
		}
	}
	var codexFound bool
	for _, l := range m.Links {
		if l.Agent == "codex" {
			codexFound = true
			break
		}
	}
	if !codexFound {
		t.Error("codex link missing from manifest after unlink --agent claude")
	}
}

func TestUnlink_CopyMode_OnlyRemovesTracking(t *testing.T) {
	setupEnv(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}

	// Write a copy-mode link entry directly in the manifest.
	tmp := t.TempDir()
	target := filepath.Join(tmp, "CLAUDE.md")
	if err := os.WriteFile(target, []byte("standalone"), 0o644); err != nil {
		t.Fatal(err)
	}

	m, _ := ReadManifest()
	m.Links = []LinkEntry{
		{Source: "instructions/AGENTS.md", Target: target, Agent: "claude", Mode: "copy"},
	}
	if err := WriteManifest(m); err != nil {
		t.Fatal(err)
	}

	actions, err := Unlink(UnlinkOptions{})
	if err != nil {
		t.Fatalf("Unlink() error = %v", err)
	}

	if len(actions) != 1 {
		t.Fatalf("Unlink() returned %d actions, want 1", len(actions))
	}
	if actions[0].Err != nil {
		t.Errorf("action.Err = %v, want nil for copy-mode unlink", actions[0].Err)
	}

	// The file should still exist as-is.
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "standalone" {
		t.Errorf("copy file content changed after unlink, want original content preserved")
	}
}

func TestUnlink_DirSymlink_ReplacesWithCopy(t *testing.T) {
	storeDir, homeDir := setupLinkEnv(t)

	// Write a file in the store's skills/ directory.
	skillContent := "# Skill\n"
	writeStoreFile(t, storeDir, "skills/my-skill.md", skillContent)

	claudeConfigDir := filepath.Join(homeDir, ".claude")
	makeDetectedAgent(t, "claude", claudeConfigDir)

	// Link the skills dir.
	if _, err := Link(LinkOptions{}); err != nil {
		t.Fatalf("Link() error = %v", err)
	}

	skillsTarget := filepath.Join(claudeConfigDir, "skills")
	info, _ := os.Lstat(skillsTarget)
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("skills target is not a symlink after Link()")
	}

	// Unlink.
	if _, err := Unlink(UnlinkOptions{}); err != nil {
		t.Fatalf("Unlink() error = %v", err)
	}

	// Skills target must now be a real directory.
	info, err := os.Lstat(skillsTarget)
	if err != nil {
		t.Fatalf("skills target missing after Unlink(): %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("skills target is still a symlink after Unlink(), want real directory")
	}
	if !info.IsDir() {
		t.Error("skills target is not a directory after Unlink(), want directory")
	}

	// File inside must still be accessible.
	data, err := os.ReadFile(filepath.Join(skillsTarget, "my-skill.md"))
	if err != nil {
		t.Fatalf("reading skill file after unlink: %v", err)
	}
	if string(data) != skillContent {
		t.Errorf("skill file content = %q, want %q", string(data), skillContent)
	}
}

// --- Integration: full link → unlink cycle ---

func TestLinkUnlink_FullCycle(t *testing.T) {
	storeDir, homeDir := setupLinkEnv(t)

	content := "# Shared Instructions\n"
	writeStoreFile(t, storeDir, "instructions/AGENTS.md", content)

	claudeConfigDir := filepath.Join(homeDir, ".claude")
	makeDetectedAgent(t, "claude", claudeConfigDir)

	// 1. Link.
	linkActions, err := Link(LinkOptions{})
	if err != nil {
		t.Fatalf("Link() error = %v", err)
	}
	if len(linkActions) == 0 {
		t.Fatal("Link() returned no actions")
	}

	target := filepath.Join(claudeConfigDir, "CLAUDE.md")

	// 2. Verify symlink.
	info, err := os.Lstat(target)
	if err != nil {
		t.Fatalf("Lstat target after link: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("target is not a symlink after Link()")
	}

	// 3. Unlink.
	unlinkActions, err := Unlink(UnlinkOptions{})
	if err != nil {
		t.Fatalf("Unlink() error = %v", err)
	}
	if len(unlinkActions) == 0 {
		t.Fatal("Unlink() returned no actions")
	}

	// 4. Verify regular file with original content.
	info, err = os.Lstat(target)
	if err != nil {
		t.Fatalf("Lstat target after unlink: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("target is still a symlink after Unlink()")
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading target after unlink: %v", err)
	}
	if string(data) != content {
		t.Errorf("content after unlink = %q, want %q", string(data), content)
	}

	// 5. Verify manifest has no links.
	m, err := ReadManifest()
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Links) != 0 {
		t.Errorf("manifest links count = %d after full cycle, want 0", len(m.Links))
	}
}

func TestUnlink_PreservesFilePermissions(t *testing.T) {
	storeDir, homeDir := setupLinkEnv(t)

	// Write source file, then explicitly chmod to 0755 to bypass the umask.
	p := filepath.Join(storeDir, "instructions", "AGENTS.md")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("# Instructions\n"), 0o644); err != nil {
		t.Fatalf("writing store file: %v", err)
	}
	if err := os.Chmod(p, 0o755); err != nil {
		t.Fatalf("chmod source file: %v", err)
	}

	claudeConfigDir := filepath.Join(homeDir, ".claude")
	makeDetectedAgent(t, "claude", claudeConfigDir)

	// Link (symlink mode).
	if _, err := Link(LinkOptions{}); err != nil {
		t.Fatalf("Link() error = %v", err)
	}

	target := filepath.Join(claudeConfigDir, "CLAUDE.md")

	// Unlink — should produce a standalone file with preserved permissions.
	if _, err := Unlink(UnlinkOptions{}); err != nil {
		t.Fatalf("Unlink() error = %v", err)
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("Stat target after unlink: %v", err)
	}

	// Permissions must match the source (0755). Use Chmod in the fix to bypass
	// the umask so the write matches the source mode exactly.
	got := info.Mode().Perm()
	want := os.FileMode(0o755)
	if got != want {
		t.Errorf("file permissions after unlink = %04o, want %04o", got, want)
	}
}
