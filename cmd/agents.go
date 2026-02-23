package cmd

import (
	"fmt"

	"github.com/rnwolfe/mine/internal/agents"
	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "Manage coding agent configurations from a canonical store",
	Long:  `Manage your coding agent configurations with a single canonical store of instructions, rules, and skills. Linking configs to individual agents will be available in a future release.`,
	RunE:  hook.Wrap("agents", runAgentsStatus),
}

func init() {
	rootCmd.AddCommand(agentsCmd)
	agentsCmd.AddCommand(agentsInitCmd)
}

var agentsInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create the canonical agents store with a starter directory structure",
	RunE:  hook.Wrap("agents.init", runAgentsInit),
}

func runAgentsInit(_ *cobra.Command, _ []string) error {
	if err := agents.Init(); err != nil {
		return err
	}

	dir := agents.Dir()
	ui.Ok("Agents store ready — one place for all your agent configs")
	fmt.Printf("  Location: %s\n", ui.Muted.Render(dir))
	fmt.Println()
	fmt.Printf("  Edit shared instructions: %s\n", ui.Accent.Render("instructions/AGENTS.md"))
	fmt.Printf("  Store path:               %s\n", ui.Accent.Render(dir))
	fmt.Println()
	return nil
}

func runAgentsStatus(_ *cobra.Command, _ []string) error {
	if !agents.IsInitialized() {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No agents store yet."))
		fmt.Printf("  Run %s to get started.\n", ui.Accent.Render("mine agents init"))
		fmt.Println()
		return nil
	}

	dir := agents.Dir()
	m, err := agents.ReadManifest()
	if err != nil {
		return fmt.Errorf("reading manifest: %w", err)
	}

	fmt.Println()
	ui.Kv(ui.IconTools+" Store", dir)

	detected := 0
	for _, a := range m.Agents {
		if a.Detected {
			detected++
		}
	}

	if len(m.Agents) == 0 {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No agents registered yet."))
	} else {
		ui.Kv("  Agents", fmt.Sprintf("%d registered, %d detected", len(m.Agents), detected))
	}

	if len(m.Links) > 0 {
		ui.Kv("  Links", fmt.Sprintf("%d active", len(m.Links)))
	}

	fmt.Println()
	fmt.Printf("  Re-initialize: %s\n", ui.Accent.Render("mine agents init"))
	fmt.Println()
	return nil
}
