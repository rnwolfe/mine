package cmd

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/rnwolfe/mine/internal/plugin"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

var pluginCmd = &cobra.Command{
	Use:   "plugin",
	Short: "Manage plugins",
	Long:  `Install, remove, and manage mine plugins from GitHub repositories.`,
	RunE:  runPluginList,
}

var pluginListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed plugins",
	RunE:  runPluginList,
}

var pluginInfoCmd = &cobra.Command{
	Use:   "info <name>",
	Short: "Show detailed plugin info",
	Args:  cobra.ExactArgs(1),
	RunE:  runPluginInfo,
}

var pluginInstallCmd = &cobra.Command{
	Use:   "install <path>",
	Short: "Install a plugin from a local directory",
	Long: `Install a mine plugin from a local directory containing mine-plugin.toml.

Examples:
  mine plugin install ./my-plugin
  mine plugin install /path/to/mine-plugin-obsidian`,
	Args: cobra.ExactArgs(1),
	RunE: runPluginInstall,
}

var pluginRemoveCmd = &cobra.Command{
	Use:     "remove <name>",
	Aliases: []string{"rm", "uninstall"},
	Short:   "Remove an installed plugin",
	Args:    cobra.ExactArgs(1),
	RunE:    runPluginRemove,
}

var pluginSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search GitHub for mine plugins",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runPluginSearch,
}

var pluginSearchTag string

func init() {
	rootCmd.AddCommand(pluginCmd)
	pluginCmd.AddCommand(pluginListCmd)
	pluginCmd.AddCommand(pluginInfoCmd)
	pluginCmd.AddCommand(pluginInstallCmd)
	pluginCmd.AddCommand(pluginRemoveCmd)
	pluginCmd.AddCommand(pluginSearchCmd)

	pluginSearchCmd.Flags().StringVar(&pluginSearchTag, "tag", "", "Filter by GitHub topic")
}

func runPluginList(_ *cobra.Command, _ []string) error {
	plugins, err := plugin.List()
	if err != nil {
		return err
	}

	if len(plugins) == 0 {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No plugins installed."))
		fmt.Println()
		fmt.Printf("  Search: %s\n", ui.Accent.Render("mine plugin search <query>"))
		fmt.Printf("  Install: %s\n", ui.Accent.Render("mine plugin install <path>"))
		fmt.Println()
		return nil
	}

	fmt.Println()
	fmt.Println(ui.Title.Render("  Installed Plugins"))
	fmt.Println()

	for _, p := range plugins {
		status := ui.Success.Render("●")
		if !p.Enabled {
			status = ui.Muted.Render("○")
		}

		name := ui.Accent.Render(p.Manifest.Plugin.Name)
		version := ui.Muted.Render("v" + p.Manifest.Plugin.Version)
		hookCount := len(p.Manifest.Hooks)
		cmdCount := len(p.Manifest.Commands)

		meta := ui.Muted.Render(fmt.Sprintf("%d hooks, %d commands", hookCount, cmdCount))

		fmt.Printf("  %s %-20s %s  %s\n", status, name, version, meta)
	}

	fmt.Println()
	fmt.Printf("  %s\n", ui.Muted.Render(fmt.Sprintf("%d plugins installed", len(plugins))))
	fmt.Println()
	return nil
}

func runPluginInfo(_ *cobra.Command, args []string) error {
	p, err := plugin.Get(args[0])
	if err != nil {
		return err
	}

	m := p.Manifest

	fmt.Println()
	fmt.Println(ui.Title.Render("  " + m.Plugin.Name))
	fmt.Println()

	ui.Kv("Version", m.Plugin.Version)
	ui.Kv("Author", m.Plugin.Author)
	ui.Kv("Description", m.Plugin.Description)
	if m.Plugin.License != "" {
		ui.Kv("License", m.Plugin.License)
	}
	ui.Kv("Protocol", m.Plugin.ProtocolVersion)
	ui.Kv("Directory", p.Dir)
	ui.Kv("Enabled", fmt.Sprintf("%v", p.Enabled))

	if len(m.Hooks) > 0 {
		fmt.Println()
		fmt.Println(ui.Subtitle.Render("  Hooks"))
		for _, h := range m.Hooks {
			fmt.Printf("    %s  %s  %s\n",
				ui.Accent.Render(h.Command),
				ui.Muted.Render(h.Stage),
				ui.Muted.Render(h.Mode),
			)
		}
	}

	if len(m.Commands) > 0 {
		fmt.Println()
		fmt.Println(ui.Subtitle.Render("  Commands"))
		for _, c := range m.Commands {
			fmt.Printf("    %s  %s\n",
				ui.Accent.Render(fmt.Sprintf("mine %s %s", m.Plugin.Name, c.Name)),
				ui.Muted.Render(c.Description),
			)
		}
	}

	fmt.Println()
	fmt.Println(ui.Subtitle.Render("  Permissions"))
	for _, line := range plugin.PermissionSummary(m.Permissions) {
		fmt.Printf("    %s\n", ui.Muted.Render(line))
	}

	fmt.Println()
	return nil
}

func runPluginInstall(_ *cobra.Command, args []string) error {
	sourceDir := args[0]

	// Parse manifest first to show permissions
	manifestPath := sourceDir + "/mine-plugin.toml"
	manifest, err := plugin.ParseManifest(manifestPath)
	if err != nil {
		return err
	}

	// Show plugin info and permissions
	fmt.Println()
	fmt.Printf("  Installing %s v%s by %s\n",
		ui.Accent.Render(manifest.Plugin.Name),
		manifest.Plugin.Version,
		manifest.Plugin.Author,
	)
	fmt.Printf("  %s\n", ui.Muted.Render(manifest.Plugin.Description))
	fmt.Println()

	// Show permissions
	fmt.Println(ui.Subtitle.Render("  Permissions:"))
	for _, line := range plugin.PermissionSummary(manifest.Permissions) {
		fmt.Printf("    %s\n", line)
	}
	fmt.Println()

	// Confirm
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("  %s ", ui.Accent.Render("Install this plugin? [y/N]"))
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		// Align with other prompts: ignore read errors and treat as empty input.
		line = ""
	}
	ans := strings.TrimSpace(strings.ToLower(line))
	if ans != "y" && ans != "yes" {
		ui.Warn("Installation cancelled.")
		return nil
	}

	p, err := plugin.Install(sourceDir, sourceDir)
	if err != nil {
		return err
	}

	if err := plugin.AuditLog(p.Manifest.Plugin.Name, "install", "version="+p.Manifest.Plugin.Version); err != nil {
		log.Printf("warning: audit log: %v", err)
	}

	fmt.Println()
	ui.Ok(fmt.Sprintf("Installed %s v%s", p.Manifest.Plugin.Name, p.Manifest.Plugin.Version))
	fmt.Printf("  %d hooks registered, %d commands available\n",
		len(p.Manifest.Hooks), len(p.Manifest.Commands))
	fmt.Println()
	return nil
}

func runPluginRemove(_ *cobra.Command, args []string) error {
	name := args[0]

	// Verify it exists
	p, err := plugin.Get(name)
	if err != nil {
		return err
	}

	if err := plugin.Remove(name); err != nil {
		return err
	}

	if err := plugin.AuditLog(name, "remove", "version="+p.Manifest.Plugin.Version); err != nil {
		log.Printf("warning: audit log: %v", err)
	}

	ui.Ok(fmt.Sprintf("Removed %s", name))
	fmt.Println()
	return nil
}

func runPluginSearch(_ *cobra.Command, args []string) error {
	query := ""
	if len(args) > 0 {
		query = args[0]
	}

	fmt.Println()
	fmt.Printf("  Searching GitHub for mine plugins")
	if query != "" {
		fmt.Printf(" matching %q", query)
	}
	fmt.Println("...")
	fmt.Println()

	results, err := plugin.Search(query, pluginSearchTag)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		fmt.Println(ui.Muted.Render("  No plugins found."))
		fmt.Println()
		return nil
	}

	for _, r := range results {
		stars := ui.Muted.Render(fmt.Sprintf("★ %d", r.Stars))
		fmt.Printf("  %s  %s\n", ui.Accent.Render(r.FullName), stars)
		if r.Description != "" {
			fmt.Printf("    %s\n", ui.Muted.Render(r.Description))
		}
		fmt.Println()
	}

	fmt.Printf("  %s\n", ui.Muted.Render(fmt.Sprintf("%d results", len(results))))
	fmt.Println()
	return nil
}
