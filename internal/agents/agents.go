package agents

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rnwolfe/mine/internal/config"
	"github.com/rnwolfe/mine/internal/gitutil"
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

