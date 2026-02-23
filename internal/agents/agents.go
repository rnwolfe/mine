package agents

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rnwolfe/mine/internal/config"
)

// ErrNothingToCommit is returned by Commit when there are no staged changes.
var ErrNothingToCommit = errors.New("nothing to commit — all files up to date")

// ErrNoVersionHistory is returned by Log when the store has no git repository.
var ErrNoVersionHistory = errors.New("no version history yet — run `mine agents commit` first")

// Agent represents a detected coding agent.
type Agent struct {
	Name      string `json:"name"`       // claude, codex, gemini, opencode
	Detected  bool   `json:"detected"`   // found on system
	ConfigDir string `json:"config_dir"` // e.g., ~/.claude/
	Binary    string `json:"binary"`     // e.g., claude
}

// LinkEntry represents a file distribution mapping from canonical store to an agent directory.
type LinkEntry struct {
	Source string `json:"source"` // Relative path in canonical store
	Target string `json:"target"` // Absolute path in agent's expected location
	Agent  string `json:"agent"`  // Which agent this serves
	Mode   string `json:"mode"`   // "symlink" or "copy"
}

// Manifest holds the state of the agents store.
type Manifest struct {
	Agents []Agent     `json:"agents"`
	Links  []LinkEntry `json:"links"`
}

// LogEntry represents a single commit in the agents store history.
type LogEntry struct {
	Hash    string
	Short   string
	Date    time.Time
	Message string
}

// Dir returns the canonical agent store directory path.
func Dir() string {
	return filepath.Join(config.GetPaths().DataDir, "agents")
}

// ManifestPath returns the path to the manifest file.
func ManifestPath() string {
	return filepath.Join(Dir(), ".mine-agents")
}

// IsInitialized returns true if the canonical store directory exists and is a directory.
func IsInitialized() bool {
	info, err := os.Stat(Dir())
	if err != nil {
		return false
	}
	return info.IsDir()
}

// IsGitRepo returns true if the canonical store is a git repository.
func IsGitRepo() bool {
	_, err := os.Stat(filepath.Join(Dir(), ".git"))
	return err == nil
}

// InitGitRepo initializes a git repo in the canonical store if one doesn't exist.
func InitGitRepo() error {
	dir := Dir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating agents directory: %w", err)
	}

	if IsGitRepo() {
		return nil
	}

	if _, err := gitCmd(dir, "init"); err != nil {
		return fmt.Errorf("git init: %w", err)
	}

	if _, err := gitCmd(dir, "config", "user.name", "mine-agents"); err != nil {
		return fmt.Errorf("git config user.name: %w", err)
	}
	if _, err := gitCmd(dir, "config", "user.email", "agents@mine.local"); err != nil {
		return fmt.Errorf("git config user.email: %w", err)
	}

	return nil
}

// Init creates the canonical store with full directory scaffold.
// Safe to call multiple times — idempotent.
func Init() error {
	dir := Dir()

	// Create the directory scaffold.
	subdirs := []string{"instructions", "skills", "commands", "agents", "settings", "mcp", "rules"}
	for _, subdir := range subdirs {
		if err := os.MkdirAll(filepath.Join(dir, subdir), 0o755); err != nil {
			return fmt.Errorf("creating %s directory: %w", subdir, err)
		}
	}

	// Initialize git repo.
	if err := InitGitRepo(); err != nil {
		return err
	}

	// Create manifest if it doesn't exist.
	manifestPath := ManifestPath()
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		m := &Manifest{
			Agents: []Agent{},
			Links:  []LinkEntry{},
		}
		if err := WriteManifest(m); err != nil {
			return fmt.Errorf("writing manifest: %w", err)
		}
	}

	// Create starter AGENTS.md if it doesn't exist.
	agentsMD := filepath.Join(dir, "instructions", "AGENTS.md")
	if _, err := os.Stat(agentsMD); os.IsNotExist(err) {
		if err := os.WriteFile(agentsMD, []byte(starterAgentsMD()), 0o644); err != nil {
			return fmt.Errorf("creating AGENTS.md: %w", err)
		}
	}

	return nil
}

// starterAgentsMD returns the starter content for the shared instructions file.
func starterAgentsMD() string {
	return `# Agent Instructions

This file contains shared instructions for all your AI coding agents.
Add your coding preferences, conventions, and project context here.

Snapshot your changes with: mine agents commit

## Coding Style

- ...

## Project Context

- ...
`
}

// ReadManifest parses the manifest file and returns the agents manifest.
// Returns nil (no error) if the manifest file does not exist.
func ReadManifest() (*Manifest, error) {
	data, err := os.ReadFile(ManifestPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}
	return &m, nil
}

// WriteManifest writes the manifest to disk.
func WriteManifest(m *Manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding manifest: %w", err)
	}
	return os.WriteFile(ManifestPath(), append(data, '\n'), 0o644)
}

// validateRelativePath checks that a path is safe to use within the canonical store.
func validateRelativePath(file string) error {
	if file == "" {
		return fmt.Errorf("empty file path")
	}
	if filepath.IsAbs(file) {
		return fmt.Errorf("file path must be relative to the agents store, not absolute: %q", file)
	}
	// Reject Windows volume-qualified paths (e.g. "C:foo") which can escape the
	// base directory on Windows when passed to filepath.Join.
	if filepath.VolumeName(file) != "" {
		return fmt.Errorf("file path must not contain a volume name: %q", file)
	}
	clean := filepath.Clean(file)
	// Reject paths that resolve to the current directory (e.g. "." or "a/..").
	if clean == "." {
		return fmt.Errorf("file path must refer to a specific file, not the current directory: %q", file)
	}
	if clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("unsafe file path (directory traversal): %q", file)
	}
	return nil
}

// Commit snapshots the current state of the canonical store with a message.
// Initializes the git repo on first commit.
func Commit(message string) (string, error) {
	dir := Dir()

	if !IsInitialized() {
		return "", fmt.Errorf("agents store not initialized — run `mine agents init` first")
	}

	if err := InitGitRepo(); err != nil {
		return "", err
	}

	// Stage all changes in the canonical store.
	if _, err := gitCmd(dir, "add", "-A"); err != nil {
		return "", fmt.Errorf("git add: %w", err)
	}

	// Check if there's anything to commit.
	status, err := gitCmd(dir, "status", "--porcelain")
	if err != nil {
		return "", fmt.Errorf("git status: %w", err)
	}
	if strings.TrimSpace(status) == "" {
		return "", ErrNothingToCommit
	}

	if _, err := gitCmd(dir, "commit", "-m", message); err != nil {
		return "", fmt.Errorf("git commit: %w", err)
	}

	hash, err := gitCmd(dir, "rev-parse", "--short", "HEAD")
	if err != nil {
		return "", fmt.Errorf("getting commit hash: %w", err)
	}
	return strings.TrimSpace(hash), nil
}

// Log returns the commit history, optionally filtered to a specific relative file path.
// Returns nil (no error) when there are no commits yet.
func Log(file string) ([]LogEntry, error) {
	dir := Dir()

	if !IsGitRepo() {
		return nil, ErrNoVersionHistory
	}

	if file != "" {
		if err := validateRelativePath(file); err != nil {
			return nil, err
		}
	}

	// Return empty history if the repo has no commits yet (empty repo).
	if _, err := gitCmd(dir, "rev-parse", "--verify", "HEAD"); err != nil {
		return nil, nil
	}

	args := []string{"log", "--format=%H|%h|%aI|%s"}
	if file != "" {
		args = append(args, "--", file)
	}

	out, err := gitCmd(dir, args...)
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}

	if strings.TrimSpace(out) == "" {
		return nil, nil
	}

	var entries []LogEntry
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "|", 4)
		if len(parts) != 4 {
			continue
		}
		t, _ := time.Parse(time.RFC3339, parts[2])
		entries = append(entries, LogEntry{
			Hash:    parts[0],
			Short:   parts[1],
			Date:    t,
			Message: parts[3],
		})
	}
	return entries, scanner.Err()
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

// RestoreToStore restores a file to the canonical store and re-distributes
// to any copy-mode link targets. Symlink targets are updated automatically
// since they point directly to the canonical store.
//
// Returns (updated, failed, err): updated contains successfully re-synced copy-mode
// links; failed contains links that could not be re-synced (caller should warn the user).
func RestoreToStore(file string, version string) (updated []LinkEntry, failed []LinkEntry, err error) {
	if err := validateRelativePath(file); err != nil {
		return nil, nil, err
	}

	content, err := Restore(file, version)
	if err != nil {
		return nil, nil, err
	}

	destPath := filepath.Join(Dir(), file)

	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return nil, nil, fmt.Errorf("creating directory for %s: %w", file, err)
	}

	// Preserve existing file permissions.
	mode := os.FileMode(0o644)
	if info, err := os.Stat(destPath); err == nil {
		mode = info.Mode().Perm()
	}

	if err := os.WriteFile(destPath, content, mode); err != nil {
		return nil, nil, fmt.Errorf("writing %s: %w", file, err)
	}

	// Re-distribute to copy-mode link targets.
	manifest, readErr := ReadManifest()
	if readErr != nil {
		return nil, nil, fmt.Errorf("reading manifest: %w", readErr)
	}
	if manifest == nil {
		return nil, nil, nil
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

// gitCmd runs a git command in the given directory and returns stdout.
func gitCmd(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s", msg)
	}
	return stdout.String(), nil
}
