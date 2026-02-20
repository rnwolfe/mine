package cmd

import (
	"fmt"
	"os"
	"os/exec"
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
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configUnsetCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configPathCmd)
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all known config keys and their current values",
	Args:  cobra.NoArgs,
	RunE:  hook.Wrap("config.list", runConfigList),
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE:  hook.Wrap("config.get", runConfigGet),
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a known configuration value with type-aware validation.

Keys use dot-notation (e.g. user.name, ai.provider).
Run 'mine config list' to see all available keys and their types.`,
	Args: cobra.ExactArgs(2),
	RunE: hook.Wrap("config.set", runConfigSet),
}

var configUnsetCmd = &cobra.Command{
	Use:   "unset <key>",
	Short: "Reset a config key to its default value",
	Args:  cobra.ExactArgs(1),
	RunE:  hook.Wrap("config.unset", runConfigUnset),
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open config file in $EDITOR",
	Args:  cobra.NoArgs,
	RunE:  hook.Wrap("config.edit", runConfigEdit),
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print configuration file path",
	Args:  cobra.NoArgs,
	RunE:  hook.Wrap("config.path", runConfigPath),
}

func runConfigList(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	keys := config.ValidKeyNames()
	ui.Header("Configuration Keys")
	fmt.Println()
	for _, key := range keys {
		entry := config.SchemaKeys[key]
		value := entry.Get(cfg)
		display := value
		if display == "" {
			display = ui.Muted.Render("(unset)")
		}
		typeTag := ui.Muted.Render("[" + string(entry.Type) + "]")
		fmt.Printf("  %-32s %s  %s\n", ui.KeyStyle.Render(key), display, typeTag)
	}
	fmt.Println()
	ui.Tip(fmt.Sprintf("Use %s to change a value.", ui.Accent.Render("mine config set <key> <value>")))
	fmt.Println()
	return nil
}

func runConfigGet(_ *cobra.Command, args []string) error {
	key := args[0]
	entry, ok := config.LookupKey(key)
	if !ok {
		return fmt.Errorf("unknown config key %q\n\nValid keys:\n  %s",
			key, strings.Join(config.ValidKeyNames(), "\n  "))
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	fmt.Println(entry.Get(cfg))
	return nil
}

func runConfigSet(_ *cobra.Command, args []string) error {
	key, value := args[0], args[1]
	entry, ok := config.LookupKey(key)
	if !ok {
		return fmt.Errorf("unknown config key %q\n\nValid keys:\n  %s",
			key, strings.Join(config.ValidKeyNames(), "\n  "))
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if err := entry.Set(cfg, value); err != nil {
		return fmt.Errorf("%w\n\nExpected type: %s", err, entry.Type)
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	ui.Ok(fmt.Sprintf("%s = %s", key, value))
	return nil
}

func runConfigUnset(_ *cobra.Command, args []string) error {
	key := args[0]
	entry, ok := config.LookupKey(key)
	if !ok {
		return fmt.Errorf("unknown config key %q\n\nValid keys:\n  %s",
			key, strings.Join(config.ValidKeyNames(), "\n  "))
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	entry.Unset(cfg)

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	defaultDisplay := entry.DefaultStr
	if defaultDisplay == "" {
		defaultDisplay = "(empty)"
	}
	ui.Ok(fmt.Sprintf("%s reset to default (%s)", key, defaultDisplay))
	return nil
}

func runConfigEdit(_ *cobra.Command, _ []string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		paths := config.GetPaths()
		return fmt.Errorf(
			"$EDITOR is not set\n\nEdit the config file manually:\n  %s\n\nOr set EDITOR in your shell profile (e.g. export EDITOR=vim)",
			ui.Accent.Render(paths.ConfigFile),
		)
	}

	paths := config.GetPaths()
	cmd := exec.Command(editor, paths.ConfigFile)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runConfigPath(_ *cobra.Command, _ []string) error {
	paths := config.GetPaths()
	fmt.Println(paths.ConfigFile)
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
	ui.Tip(fmt.Sprintf("Run %s to see all available keys.", ui.Accent.Render("mine config list")))
	fmt.Println()

	return nil
}
