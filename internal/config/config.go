package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds the top-level mine configuration.
type Config struct {
	User      UserConfig      `toml:"user"`
	Shell     ShellConfig     `toml:"shell"`
	AI        AIConfig        `toml:"ai"`
	Analytics AnalyticsConfig `toml:"analytics"`
}

// AnalyticsConfig controls anonymous usage analytics.
type AnalyticsConfig struct {
	// Enabled controls whether anonymous analytics are sent.
	// Defaults to true when not set in config (opt-out model).
	Enabled *bool `toml:"enabled,omitempty"`
}

// IsEnabled returns whether analytics are enabled.
// Treats nil (missing from config) as true — opt-out, not opt-in.
func (a AnalyticsConfig) IsEnabled() bool {
	if a.Enabled == nil {
		return true
	}
	return *a.Enabled
}

type UserConfig struct {
	Name  string `toml:"name"`
	Email string `toml:"email"`
}

type ShellConfig struct {
	DefaultShell string   `toml:"default_shell"`
	Aliases      []string `toml:"aliases"`
}

type AIConfig struct {
	Provider string `toml:"provider"` // claude, openai, ollama, etc.
	Model    string `toml:"model"`
}

// Paths returns standard XDG-compliant paths.
type Paths struct {
	ConfigDir string
	DataDir   string
	CacheDir  string
	StateDir  string
	ConfigFile string
	DBFile    string
}

// GetPaths returns the resolved paths, respecting XDG env vars.
func GetPaths() Paths {
	home, _ := os.UserHomeDir()

	configDir := envOr("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	dataDir := envOr("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	cacheDir := envOr("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	stateDir := envOr("XDG_STATE_HOME", filepath.Join(home, ".local", "state"))

	mineConfig := filepath.Join(configDir, "mine")
	mineData := filepath.Join(dataDir, "mine")

	return Paths{
		ConfigDir:  mineConfig,
		DataDir:    mineData,
		CacheDir:   filepath.Join(cacheDir, "mine"),
		StateDir:   filepath.Join(stateDir, "mine"),
		ConfigFile: filepath.Join(mineConfig, "config.toml"),
		DBFile:     filepath.Join(mineData, "mine.db"),
	}
}

// EnsureDirs creates all required directories.
func (p Paths) EnsureDirs() error {
	dirs := []string{p.ConfigDir, p.DataDir, p.CacheDir, p.StateDir}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	return nil
}

// Load reads config from disk, returning defaults if not found.
func Load() (*Config, error) {
	paths := GetPaths()
	cfg := &Config{}

	data, err := os.ReadFile(paths.ConfigFile)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultConfig(), nil
		}
		return nil, err
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Save writes config to disk.
func Save(cfg *Config) error {
	paths := GetPaths()
	if err := paths.EnsureDirs(); err != nil {
		return err
	}

	f, err := os.Create(paths.ConfigFile)
	if err != nil {
		return err
	}
	defer f.Close()

	return toml.NewEncoder(f).Encode(cfg)
}

// Initialized returns true if mine has been set up.
func Initialized() bool {
	paths := GetPaths()
	_, err := os.Stat(paths.ConfigFile)
	return err == nil
}

// BoolPtr returns a pointer to a bool value.
func BoolPtr(v bool) *bool {
	return &v
}

func defaultConfig() *Config {
	return &Config{
		Shell: ShellConfig{
			DefaultShell: envOr("SHELL", "/bin/bash"),
		},
		AI: AIConfig{
			Provider: "claude",
			Model:    "claude-sonnet-4-5-20250929",
		},
		Analytics: AnalyticsConfig{
			Enabled: BoolPtr(true),
		},
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
