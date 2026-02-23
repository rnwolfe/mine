package agents

import "path/filepath"

// linkSpec defines how a single agent receives linked files from the canonical store.
type linkSpec struct {
	Name                string // agent identifier, e.g. "claude"
	ConfigDir           string // agent's config directory, e.g. ~/.claude
	InstructionFilename string // filename for instructions file, e.g. "CLAUDE.md"
	SkillsDir           string // symlink target for skills/, empty if not supported
	CommandsDir         string // symlink target for commands/, empty if not supported
	SettingsFilename    string // filename for settings JSON, e.g. "settings.json"
	MCPConfigPath       string // absolute path for .mcp.json, empty if not applicable
}

// buildLinkRegistry returns the canonical per-agent link spec list.
// home is the user's home directory.
func buildLinkRegistry(home string) []linkSpec {
	return []linkSpec{
		{
			Name:                "claude",
			ConfigDir:           filepath.Join(home, ".claude"),
			InstructionFilename: "CLAUDE.md",
			SkillsDir:           filepath.Join(home, ".claude", "skills"),
			CommandsDir:         filepath.Join(home, ".claude", "commands"),
			SettingsFilename:    "settings.json",
			MCPConfigPath:       filepath.Join(home, ".claude", ".mcp.json"),
		},
		{
			Name:                "codex",
			ConfigDir:           filepath.Join(home, ".codex"),
			InstructionFilename: "AGENTS.md",
			SkillsDir:           filepath.Join(home, ".codex", "skills"),
			CommandsDir:         "",
			SettingsFilename:    "settings.json",
			MCPConfigPath:       "",
		},
		{
			Name:                "gemini",
			ConfigDir:           filepath.Join(home, ".gemini"),
			InstructionFilename: "GEMINI.md",
			SkillsDir:           filepath.Join(home, ".gemini", "skills"),
			CommandsDir:         "",
			SettingsFilename:    "settings.json",
			MCPConfigPath:       "",
		},
		{
			Name:                "opencode",
			ConfigDir:           filepath.Join(home, ".config", "opencode"),
			InstructionFilename: "AGENTS.md",
			SkillsDir:           filepath.Join(home, ".config", "opencode", "skills"),
			CommandsDir:         "",
			SettingsFilename:    "settings.json",
			MCPConfigPath:       "",
		},
	}
}
