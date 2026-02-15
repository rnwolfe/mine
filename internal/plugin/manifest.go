// Package plugin implements the mine plugin system including manifest parsing,
// installation, subprocess runtime, permissions, and registry discovery.
package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/rnwolfe/mine/internal/config"
)

// validPluginName enforces kebab-case: lowercase letters, digits, and hyphens.
var validPluginName = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`)

// Manifest represents a parsed mine-plugin.toml file.
type Manifest struct {
	Plugin      PluginMeta   `toml:"plugin"`
	Hooks       []HookDef    `toml:"hooks"`
	Commands    []CommandDef `toml:"commands"`
	Permissions Permissions  `toml:"permissions"`
}

// PluginMeta holds plugin identification and compatibility info.
type PluginMeta struct {
	Name            string `toml:"name"`
	Version         string `toml:"version"`
	Description     string `toml:"description"`
	Author          string `toml:"author"`
	License         string `toml:"license"`
	MinMineVersion  string `toml:"min_mine_version"`
	ProtocolVersion string `toml:"protocol_version"`
	Entrypoint      string `toml:"entrypoint"`
}

// HookDef defines a hook registration in the manifest.
type HookDef struct {
	Command string `toml:"command"`
	Stage   string `toml:"stage"`
	Mode    string `toml:"mode"`
	Timeout string `toml:"timeout"`
}

// CommandDef defines a custom command registration.
type CommandDef struct {
	Name        string `toml:"name"`
	Description string `toml:"description"`
	Args        string `toml:"args"`
}

// Permissions declares what system resources a plugin needs.
type Permissions struct {
	Network     bool     `toml:"network"`
	Filesystem  []string `toml:"filesystem"`
	Store       bool     `toml:"store"`
	ConfigRead  bool     `toml:"config_read"`
	ConfigWrite bool     `toml:"config_write"`
	EnvVars     []string `toml:"env_vars"`
}

// InstalledPlugin represents a plugin on disk with its parsed manifest.
type InstalledPlugin struct {
	Manifest    Manifest
	Dir         string
	InstalledAt time.Time
	Enabled     bool
}

// PluginsDir returns the directory where plugins are installed.
func PluginsDir() string {
	return filepath.Join(config.GetPaths().DataDir, "plugins")
}

// PluginsConfigFile returns the path to the plugins registry file.
func PluginsConfigFile() string {
	return filepath.Join(config.GetPaths().ConfigDir, "plugins.toml")
}

// ParseManifest reads and parses a mine-plugin.toml file.
func ParseManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	var m Manifest
	if err := toml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}

	if err := m.Validate(); err != nil {
		return nil, fmt.Errorf("invalid manifest: %w", err)
	}

	return &m, nil
}

// Validate checks that required manifest fields are present.
func (m *Manifest) Validate() error {
	if m.Plugin.Name == "" {
		return fmt.Errorf("plugin.name is required")
	}
	if !validPluginName.MatchString(m.Plugin.Name) {
		return fmt.Errorf("plugin.name %q must be kebab-case (lowercase letters, digits, and hyphens)", m.Plugin.Name)
	}
	if m.Plugin.Version == "" {
		return fmt.Errorf("plugin.version is required")
	}
	if m.Plugin.Description == "" {
		return fmt.Errorf("plugin.description is required")
	}
	if m.Plugin.Author == "" {
		return fmt.Errorf("plugin.author is required")
	}
	if m.Plugin.ProtocolVersion == "" {
		return fmt.Errorf("plugin.protocol_version is required")
	}

	for i, h := range m.Hooks {
		if h.Command == "" {
			return fmt.Errorf("hooks[%d].command is required", i)
		}
		if h.Stage == "" {
			return fmt.Errorf("hooks[%d].stage is required", i)
		}
		if h.Mode == "" {
			return fmt.Errorf("hooks[%d].mode is required", i)
		}
		if h.Stage != "prevalidate" && h.Stage != "preexec" && h.Stage != "postexec" && h.Stage != "notify" {
			return fmt.Errorf("hooks[%d].stage %q is invalid", i, h.Stage)
		}
		if h.Mode != "transform" && h.Mode != "notify" {
			return fmt.Errorf("hooks[%d].mode %q is invalid", i, h.Mode)
		}
		// Validate stage/mode pairing: notify stage requires notify mode,
		// transform mode requires a non-notify stage.
		if h.Stage == "notify" && h.Mode != "notify" {
			return fmt.Errorf("hooks[%d]: notify stage requires notify mode, got %q", i, h.Mode)
		}
		if h.Stage != "notify" && h.Mode == "notify" {
			return fmt.Errorf("hooks[%d]: notify mode is only valid with notify stage, got stage %q", i, h.Stage)
		}
	}

	for i, c := range m.Commands {
		if c.Name == "" {
			return fmt.Errorf("commands[%d].name is required", i)
		}
		if c.Description == "" {
			return fmt.Errorf("commands[%d].description is required", i)
		}
	}

	return nil
}

// Entrypoint returns the binary name for this plugin.
func (m *Manifest) Entrypoint() string {
	if m.Plugin.Entrypoint != "" {
		return m.Plugin.Entrypoint
	}
	return "mine-plugin-" + m.Plugin.Name
}
