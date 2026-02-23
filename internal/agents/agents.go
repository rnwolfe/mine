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
	Name      string `json:"name"`       // claude, codex, gemini, opencode
	Detected  bool   `json:"detected"`   // found on system
	ConfigDir string `json:"config_dir"` // e.g., ~/.claude/
	Binary    string `json:"binary"`     // e.g., claude
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
agents here. These instructions will be linked to each agent you register.

## About This File

Instructions placed here are symlinked to the expected locations for each coding
agent you use. Changes here are automatically reflected in each agent's configured
instructions directory.

## How to Use

1. Add global instructions, rules, or preferences below
2. Run ` + "`mine agents link`" + ` to sync your changes to all registered agents
3. Use ` + "`mine agents status`" + ` to see what's linked where

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

// IsInitialized returns true if the agents store directory exists.
func IsInitialized() bool {
	_, err := os.Stat(Dir())
	return err == nil
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

	// Create manifest only if it doesn't exist yet.
	if _, err := os.Stat(ManifestPath()); os.IsNotExist(err) {
		m := &Manifest{
			Agents: []Agent{},
			Links:  []LinkEntry{},
		}
		if err := WriteManifest(m); err != nil {
			return fmt.Errorf("creating manifest: %w", err)
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

// initGitRepo initializes a git repo in the agents directory if one doesn't exist.
func initGitRepo(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
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

// gitCmd runs a git command in dir and returns stdout.
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
