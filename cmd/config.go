package cmd

import (
	"fmt"
	"strings"

	"github.com/rnwolfe/mine/internal/config"
	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View and manage configuration",
	RunE:  hook.Wrap("config", runConfigShow),
}

func init() {
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print configuration file path",
	Run: func(_ *cobra.Command, _ []string) {
		paths := config.GetPaths()
		fmt.Println(paths.ConfigFile)
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value. Supported keys:
  analytics    Enable/disable anonymous usage analytics (true/false)
  user.name    Your display name
  ai.provider  AI provider (claude, openai, gemini, openrouter)
  ai.model     AI model name`,
	Args: cobra.ExactArgs(2),
	RunE: hook.Wrap("config.set", runConfigSet),
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE:  hook.Wrap("config.get", runConfigGet),
}

// configKeys maps user-facing key names to getter/setter pairs.
var configKeys = map[string]struct {
	get func(*config.Config) string
	set func(*config.Config, string) error
}{
	"analytics": {
		get: func(cfg *config.Config) string {
			return fmt.Sprintf("%t", cfg.Analytics.IsEnabled())
		},
		set: func(cfg *config.Config, val string) error {
			switch strings.ToLower(val) {
			case "true", "1", "yes", "on":
				cfg.Analytics.Enabled = config.BoolPtr(true)
			case "false", "0", "no", "off":
				cfg.Analytics.Enabled = config.BoolPtr(false)
			default:
				return fmt.Errorf("invalid value %q for analytics (use true/false)", val)
			}
			return nil
		},
	},
	"user.name": {
		get: func(cfg *config.Config) string { return cfg.User.Name },
		set: func(cfg *config.Config, val string) error {
			cfg.User.Name = val
			return nil
		},
	},
	"ai.provider": {
		get: func(cfg *config.Config) string { return cfg.AI.Provider },
		set: func(cfg *config.Config, val string) error {
			cfg.AI.Provider = val
			return nil
		},
	},
	"ai.model": {
		get: func(cfg *config.Config) string { return cfg.AI.Model },
		set: func(cfg *config.Config, val string) error {
			cfg.AI.Model = val
			return nil
		},
	},
}

func runConfigSet(_ *cobra.Command, args []string) error {
	key, value := args[0], args[1]

	entry, ok := configKeys[key]
	if !ok {
		return fmt.Errorf("unknown config key %q (run %s to see available keys)",
			key, ui.Accent.Render("mine config set --help"))
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if err := entry.set(cfg, value); err != nil {
		return err
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	ui.Ok(fmt.Sprintf("%s = %s", key, value))
	return nil
}

func runConfigGet(_ *cobra.Command, args []string) error {
	key := args[0]

	entry, ok := configKeys[key]
	if !ok {
		return fmt.Errorf("unknown config key %q", key)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	fmt.Println(entry.get(cfg))
	return nil
}

func runConfigShow(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	paths := config.GetPaths()

	ui.Header("Configuration")
	fmt.Println()
	ui.Kv("Name", cfg.User.Name)
	ui.Kv("Shell", cfg.Shell.DefaultShell)
	ui.Kv("AI", fmt.Sprintf("%s / %s", cfg.AI.Provider, cfg.AI.Model))
	ui.Kv("Analytics", fmt.Sprintf("%t", cfg.Analytics.IsEnabled()))
	fmt.Println()
	ui.Kv("Config", paths.ConfigFile)
	ui.Kv("Data", paths.DBFile)
	ui.Kv("Cache", paths.CacheDir)
	fmt.Println()
	ui.Tip(fmt.Sprintf("Edit directly: %s", ui.Accent.Render("$EDITOR "+paths.ConfigFile)))
	fmt.Println()

	return nil
}
