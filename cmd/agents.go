package cmd

import (
	"fmt"
	"strings"

	"github.com/rnwolfe/mine/internal/agents"
	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "Manage coding agent configurations from a canonical store",
	Long:  `Manage your coding agent configurations with a single canonical store of instructions, rules, and skills.`,
	RunE:  hook.Wrap("agents", runAgentsStatus),
}

func init() {
	rootCmd.AddCommand(agentsCmd)
	agentsCmd.AddCommand(agentsInitCmd)
	agentsCmd.AddCommand(agentsDetectCmd)
}

var agentsInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create the canonical agents store with a starter directory structure",
	RunE:  hook.Wrap("agents.init", runAgentsInit),
}

var agentsDetectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Scan for installed coding agents and persist results to the manifest",
	RunE:  hook.Wrap("agents.detect", runAgentsDetect),
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
	fmt.Println()
	return nil
}

func runAgentsDetect(_ *cobra.Command, _ []string) error {
	detected := agents.DetectAgents()

	fmt.Println()
	fmt.Printf("  %-14s %-30s %-28s %s\n",
		ui.KeyStyle.Render("Agent"),
		ui.KeyStyle.Render("Binary"),
		ui.KeyStyle.Render("Config Dir"),
		ui.KeyStyle.Render("Status"),
	)
	fmt.Println(ui.Muted.Render("  " + strings.Repeat("─", 82)))

	detectedCount := 0
	for _, a := range detected {
		printAgentRow(a)
		if a.Detected {
			detectedCount++
		}
	}

	fmt.Println()

	if detectedCount == 0 {
		fmt.Println(ui.Muted.Render("  No coding agents detected on this system."))
		fmt.Printf("  Install an agent and re-run %s to register it.\n", ui.Accent.Render("mine agents detect"))
	} else {
		ui.Ok(fmt.Sprintf("%d agent(s) detected", detectedCount))
	}

	// Persist results to manifest (initializes the store if needed).
	if err := persistDetectionResults(detected); err != nil {
		return fmt.Errorf("saving detection results: %w", err)
	}

	fmt.Println(ui.Muted.Render("  Manifest updated."))
	fmt.Println()
	return nil
}

// printAgentRow prints a single agent detection row.
func printAgentRow(a agents.Agent) {
	var statusStr string
	if a.Detected {
		statusStr = ui.Success.Render(ui.IconOk + "detected")
	} else {
		statusStr = ui.Muted.Render(ui.IconError + "not found")
	}

	binaryDisplay := a.Binary
	if binaryDisplay == "" {
		binaryDisplay = ui.Muted.Render("not in PATH")
	}

	configDisplay := a.ConfigDir
	if !agents.DirExists(a.ConfigDir) {
		configDisplay = ui.Muted.Render("not found")
	}

	fmt.Printf("  %-14s %-30s %-28s %s\n", a.Name, binaryDisplay, configDisplay, statusStr)
}

// persistDetectionResults saves the detection results to the manifest.
// It initializes the store first if needed, and replaces the agents list (no duplication).
func persistDetectionResults(detected []agents.Agent) error {
	if !agents.IsInitialized() {
		if err := agents.Init(); err != nil {
			return fmt.Errorf("initializing agents store: %w", err)
		}
	}

	m, err := agents.ReadManifest()
	if err != nil {
		return err
	}

	// Replace agents list entirely — re-running detect is always idempotent.
	m.Agents = detected

	return agents.WriteManifest(m)
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

	detectedCount := 0
	for _, a := range m.Agents {
		if a.Detected {
			detectedCount++
		}
	}

	if len(m.Agents) == 0 {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No agents registered yet."))
		fmt.Printf("  Run %s to scan for installed agents.\n", ui.Accent.Render("mine agents detect"))
	} else {
		ui.Kv("  Agents", fmt.Sprintf("%d registered, %d detected", len(m.Agents), detectedCount))
	}

	if len(m.Links) == 0 {
		fmt.Println(ui.Muted.Render("  No links configured yet."))
	} else {
		ui.Kv("  Links", fmt.Sprintf("%d active", len(m.Links)))
	}

	fmt.Println()
	return nil
}

