package agents

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rnwolfe/mine/internal/config"
)

// Agent represents a detected coding agent.
type Agent struct {
	Name      string `json:"name"`      // claude, codex, gemini, opencode
	Detected  bool   `json:"detected"`  // found on system
	ConfigDir string `json:"configDir"` // e.g., ~/.claude/
	Binary    string `json:"binary"`    // e.g., claude
}

// LinkEntry represents a file link from the canonical store to an agent directory.
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

// Dir returns the canonical agent config store path.
func Dir() string {
	return filepath.Join(config.GetPaths().DataDir, "agents")
}

// ManifestPath returns the path to the agents manifest file.
func ManifestPath() string {
	return filepath.Join(Dir(), ".mine-agents")
}

// IsInitialized returns true if the canonical store directory exists.
func IsInitialized() bool {
	_, err := os.Stat(Dir())
	return err == nil
}

// IsGitRepo returns true if the agents store directory is a git repository.
func IsGitRepo() bool {
	_, err := os.Stat(filepath.Join(Dir(), ".git"))
	return err == nil
}

// Init creates the canonical agent config store with a scaffolded directory
// structure and initializes a git repo. Safe to call multiple times (idempotent).
func Init() error {
	dir := Dir()

	// Create the full directory scaffold.
	subdirs := []string{
		"instructions",
		"skills",
		"commands",
		"agents",
		"settings",
		"mcp",
		"rules",
	}
	for _, sub := range subdirs {
		if err := os.MkdirAll(filepath.Join(dir, sub), 0o755); err != nil {
			return fmt.Errorf("creating %s/: %w", sub, err)
		}
	}

	// Create starter AGENTS.md if it doesn't exist.
	agentsMD := filepath.Join(dir, "instructions", "AGENTS.md")
	if _, err := os.Stat(agentsMD); os.IsNotExist(err) {
		content := `# Agent Instructions

This file is managed by mine agents. It provides shared instructions to all
your coding agents (Claude, Codex, Gemini, OpenCode, etc.).

## Style Guidelines

- Write clean, readable code with clear variable names
- Add comments for non-obvious logic
- Follow the language's idiomatic conventions

## Project Context

<!-- Add project-specific context here -->
`
		if err := os.WriteFile(agentsMD, []byte(content), 0o644); err != nil {
			return fmt.Errorf("creating AGENTS.md: %w", err)
		}
	}

	// Create manifest if it doesn't exist.
	if _, err := os.Stat(ManifestPath()); os.IsNotExist(err) {
		m := &Manifest{
			Agents: []Agent{},
			Links:  []LinkEntry{},
		}
		if err := WriteManifest(m); err != nil {
			return fmt.Errorf("creating manifest: %w", err)
		}
	}

	// Initialize git repo.
	if err := InitGitRepo(); err != nil {
		return err
	}

	return nil
}

// ReadManifest parses the agents manifest file and returns the Manifest.
// Returns an empty Manifest (not nil) if the file doesn't exist.
func ReadManifest() (*Manifest, error) {
	data, err := os.ReadFile(ManifestPath())
	if err != nil {
		if os.IsNotExist(err) {
			return &Manifest{
				Agents: []Agent{},
				Links:  []LinkEntry{},
			}, nil
		}
		return nil, err
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

// WriteManifest serializes the Manifest to the manifest file.
func WriteManifest(m *Manifest) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding manifest: %w", err)
	}
	if err := os.MkdirAll(Dir(), 0o755); err != nil {
		return fmt.Errorf("creating agents dir: %w", err)
	}
	return os.WriteFile(ManifestPath(), append(data, '\n'), 0o644)
}

// gitCmd runs a git command in the agents store directory and returns stdout.
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
