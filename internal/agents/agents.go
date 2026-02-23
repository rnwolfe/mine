package agents

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rnwolfe/mine/internal/config"
	"github.com/rnwolfe/mine/internal/gitutil"
)

// Sentinel errors for version history operations.
var (
	ErrNothingToCommit = errors.New("nothing to commit — all agent configs are up to date")
	ErrNoVersionHistory = errors.New("no version history yet — run `mine agents init` first")
)

// Agent represents a detected coding agent.
type Agent struct {
	Name      string `json:"name"`       // claude, codex, gemini, opencode
	Detected  bool   `json:"detected"`   // found on system
	ConfigDir string `json:"config_dir"` // e.g., ~/.claude/
	Binary    string `json:"binary"`     // full path if found, empty if not
}

// LinkEntry represents a symlink mapping.
type LinkEntry struct {
	Source string `json:"source"` // Path in canonical store (relative)
	Target string `json:"target"` // Absolute path in agent's expected location
	Agent  string `json:"agent"`  // Which agent this serves
	Mode   string `json:"mode"`   // "symlink" or "copy"
}

// Manifest holds the state of the agents store.
type Manifest struct {
	Agents []Agent     `json:"agents"`
	Links  []LinkEntry `json:"links"`
}

// agentsMD is the starter content for instructions/AGENTS.md.
const agentsMD = `# Agent Instructions

This file is managed by mine agents. Add shared instructions for your coding
agents here.

## About This File

Instructions placed here are the central source of truth for your coding agent
configurations. Keep rules, preferences, and shared context here.

## How to Use

1. Add global instructions, rules, or preferences below
2. Run ` + "`mine agents`" + ` to see which agents were detected and how they're configured

## Shared Instructions

<!-- Add your shared agent instructions below this line -->
`

// Dir returns the canonical agents store path.
func Dir() string {
	return filepath.Join(config.GetPaths().DataDir, "agents")
}

// ManifestPath returns the path to the manifest file.
func ManifestPath() string {
	return filepath.Join(Dir(), ".mine-agents")
}

// IsInitialized returns true if the agents store directory exists and has a manifest.
func IsInitialized() bool {
	info, err := os.Stat(Dir())
	if err != nil || !info.IsDir() {
		return false
	}

	if _, err := os.Stat(ManifestPath()); err != nil {
		return false
	}

	return true
}

// Init creates the canonical agents store with a full directory scaffold.
// It is idempotent: running it twice produces the same result without error.
func Init() error {
	dir := Dir()

	// Create main directory and all subdirectories.
	subdirs := []string{
		"",
		"instructions",
		"skills",
		"commands",
		"agents",
		"settings",
		"mcp",
		"rules",
	}
	for _, sub := range subdirs {
		p := filepath.Join(dir, sub)
		if err := os.MkdirAll(p, 0o755); err != nil {
			return fmt.Errorf("creating directory %s: %w", p, err)
		}
	}

	// Create starter AGENTS.md only if it doesn't exist yet.
	agentsMDPath := filepath.Join(dir, "instructions", "AGENTS.md")
	if _, err := os.Stat(agentsMDPath); os.IsNotExist(err) {
		if err := os.WriteFile(agentsMDPath, []byte(agentsMD), 0o644); err != nil {
			return fmt.Errorf("creating AGENTS.md: %w", err)
		}
	}

	// Initialize git repo (no-op if already exists).
	if err := initGitRepo(dir); err != nil {
		return err
	}

	// Create manifest and make initial commit only on fresh init.
	if _, err := os.Stat(ManifestPath()); os.IsNotExist(err) {
		m := &Manifest{
			Agents: []Agent{},
			Links:  []LinkEntry{},
		}
		if err := WriteManifest(m); err != nil {
			return fmt.Errorf("creating manifest: %w", err)
		}

		// Snapshot the initial store state so git history is non-empty.
		if _, err := Commit("init: initialize agents store"); err != nil {
			return fmt.Errorf("initial commit: %w", err)
		}
	}

	return nil
}

// ReadManifest reads and parses the manifest file.
// Returns an empty manifest if the file doesn't exist.
func ReadManifest() (*Manifest, error) {
	data, err := os.ReadFile(ManifestPath())
	if err != nil {
		if os.IsNotExist(err) {
			return &Manifest{Agents: []Agent{}, Links: []LinkEntry{}}, nil
		}
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}

	// Ensure slices are non-nil for consistent behavior.
	if m.Agents == nil {
		m.Agents = []Agent{}
	}
	if m.Links == nil {
		m.Links = []LinkEntry{}
	}

	return &m, nil
}

// WriteManifest serializes and writes the manifest file.
func WriteManifest(m *Manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling manifest: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(ManifestPath(), data, 0o644); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}
	return nil
}

// IsGitRepo returns true if the agents store directory is a git repository.
func IsGitRepo() bool {
	_, err := os.Stat(filepath.Join(Dir(), ".git"))
	return err == nil
}

// gitCmd runs a git command in the given directory and returns stdout.
// It is a thin wrapper around gitutil.RunCmd for use within this package.
func gitCmd(dir string, args ...string) (string, error) {
	return gitutil.RunCmd(dir, args...)
}

// initGitRepo initializes a git repo in the agents directory if one doesn't exist.
func initGitRepo(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
		return nil
	}

	if _, err := gitutil.RunCmd(dir, "init"); err != nil {
		return fmt.Errorf("git init: %w", err)
	}
	if _, err := gitutil.RunCmd(dir, "config", "user.name", "mine-agents"); err != nil {
		return fmt.Errorf("git config user.name: %w", err)
	}
	if _, err := gitutil.RunCmd(dir, "config", "user.email", "agents@mine.local"); err != nil {
		return fmt.Errorf("git config user.email: %w", err)
	}

	return nil
}

// validateRelativePath checks that a path is safe to use within the canonical store.
func validateRelativePath(file string) error {
	if file == "" {
		return fmt.Errorf("empty file path")
	}
	if filepath.IsAbs(file) {
		return fmt.Errorf("file path must be relative to the agents store, not absolute: %q", file)
	}
	if filepath.VolumeName(file) != "" {
		return fmt.Errorf("file path must not contain a volume name: %q", file)
	}
	clean := filepath.Clean(file)
	if clean == "." {
		return fmt.Errorf("file path must refer to a specific file, not the current directory: %q", file)
	}
	if clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("unsafe file path (directory traversal): %q", file)
	}
	return nil
}

// Restore retrieves file content from the canonical store at a specific version.
// file must be a path relative to the canonical store (e.g., "instructions/AGENTS.md").
// If version is empty, retrieves from the latest commit (HEAD).
func Restore(file string, version string) ([]byte, error) {
	dir := Dir()

	if !IsGitRepo() {
		return nil, fmt.Errorf("no version history yet — run `mine agents commit` first")
	}

	if err := validateRelativePath(file); err != nil {
		return nil, err
	}

	if version == "" {
		version = "HEAD"
	}

	content, err := gitCmd(dir, "show", version+":"+file)
	if err != nil {
		return nil, fmt.Errorf("version %s not found for %s", version, file)
	}

	return []byte(content), nil
}

// RestoreToStore restores a file to the canonical store and re-distributes to any
// copy-mode link targets. Symlink targets update automatically via the symlink itself.
// Returns (updated, failed, err): updated contains successfully re-synced copy-mode
// links; failed contains links that could not be re-synced.
func RestoreToStore(file string, version string) (updated []LinkEntry, failed []LinkEntry, err error) {
	if err := validateRelativePath(file); err != nil {
		return nil, nil, err
	}

	content, err := Restore(file, version)
	if err != nil {
		return nil, nil, err
	}

	destPath := filepath.Join(Dir(), file)

	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return nil, nil, fmt.Errorf("creating directory for %s: %w", file, err)
	}

	mode := os.FileMode(0o644)
	if info, statErr := os.Stat(destPath); statErr == nil {
		mode = info.Mode().Perm()
	}

	if err := os.WriteFile(destPath, content, mode); err != nil {
		return nil, nil, fmt.Errorf("writing %s: %w", file, err)
	}

	manifest, readErr := ReadManifest()
	if readErr != nil {
		return nil, nil, fmt.Errorf("reading manifest: %w", readErr)
	}

	for _, link := range manifest.Links {
		if link.Source != file || link.Mode != "copy" {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(link.Target), 0o755); err != nil {
			failed = append(failed, link)
			continue
		}
		if err := os.WriteFile(link.Target, content, mode); err != nil {
			failed = append(failed, link)
			continue
		}
		updated = append(updated, link)
	}

	return updated, failed, nil
}
