package cmd

import (
	"fmt"

	"github.com/rnwolfe/mine/internal/config"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View and manage configuration",
	RunE:  runConfigShow,
}

func init() {
	configCmd.AddCommand(configPathCmd)
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print configuration file path",
	Run: func(_ *cobra.Command, _ []string) {
		paths := config.GetPaths()
		fmt.Println(paths.ConfigFile)
	},
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
	fmt.Println()
	ui.Kv("Config", paths.ConfigFile)
	ui.Kv("Data", paths.DBFile)
	ui.Kv("Cache", paths.CacheDir)
	fmt.Println()
	ui.Tip(fmt.Sprintf("Edit directly: %s", ui.Accent.Render("$EDITOR "+paths.ConfigFile)))
	fmt.Println()

	return nil
}
