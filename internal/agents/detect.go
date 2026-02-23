package agents

import (
	"os"
	"os/exec"
	"path/filepath"
)

// agentSpec defines the detection parameters for a single coding agent.
// Adding a new agent to the registry slice requires no changes to detection logic.
type agentSpec struct {
	Name      string // unique agent identifier, e.g. "claude"
	Binary    string // executable name to search in PATH, e.g. "claude"
	ConfigDir string // full path to agent config directory
	SkillsDir string // full path to agent skills directory
}

// buildRegistry returns the canonical list of supported coding agents.
// home is the user's home directory used to expand config paths.
func buildRegistry(home string) []agentSpec {
	return []agentSpec{
		{
			Name:      "claude",
			Binary:    "claude",
			ConfigDir: filepath.Join(home, ".claude"),
			SkillsDir: filepath.Join(home, ".claude", "skills"),
		},
		{
			Name:      "codex",
			Binary:    "codex",
			ConfigDir: filepath.Join(home, ".codex"),
			SkillsDir: filepath.Join(home, ".agents", "skills"),
		},
		{
			Name:      "gemini",
			Binary:    "gemini",
			ConfigDir: filepath.Join(home, ".gemini"),
			SkillsDir: filepath.Join(home, ".gemini", "skills"),
		},
		{
			Name:      "opencode",
			Binary:    "opencode",
			ConfigDir: filepath.Join(home, ".config", "opencode"),
			SkillsDir: filepath.Join(home, ".config", "opencode", "skills"),
		},
	}
}

// DetectAgents scans the system for installed coding agents.
// An agent is considered detected if its binary is in PATH and/or its config
// directory exists. Results can be persisted via WriteManifest.
func DetectAgents() []Agent {
	home, _ := os.UserHomeDir()
	return detectAgents(home)
}

// detectAgents is the testable core of DetectAgents with an injectable home dir.
func detectAgents(home string) []Agent {
	specs := buildRegistry(home)
	result := make([]Agent, len(specs))
	for i, spec := range specs {
		binaryPath, hasBinary := detectBinary(spec.Binary)
		hasConfigDir := detectConfigDir(spec.ConfigDir)

		result[i] = Agent{
			Name:      spec.Name,
			Detected:  hasBinary || hasConfigDir,
			ConfigDir: spec.ConfigDir,
			Binary:    binaryPath,
		}
	}
	return result
}

// detectBinary checks if a binary exists in PATH.
// Returns the resolved full path and true if found, empty string and false otherwise.
func detectBinary(name string) (string, bool) {
	path, err := exec.LookPath(name)
	return path, err == nil
}

// detectConfigDir checks if an agent's config directory exists.
func detectConfigDir(dir string) bool {
	info, err := os.Stat(dir)
	return err == nil && info.IsDir()
}

// DirExists reports whether path exists and is a directory.
// Used by callers that need to check config dir existence for display purposes.
func DirExists(path string) bool {
	return detectConfigDir(path)
}
