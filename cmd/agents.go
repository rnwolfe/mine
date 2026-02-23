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

var (
	agentsLinkAgent string
	agentsLinkCopy  bool
	agentsLinkForce bool

	agentsUnlinkAgent string
)

func init() {
	rootCmd.AddCommand(agentsCmd)
	agentsCmd.AddCommand(agentsInitCmd)
	agentsCmd.AddCommand(agentsDetectCmd)
	agentsCmd.AddCommand(agentsLinkCmd)
	agentsCmd.AddCommand(agentsUnlinkCmd)

	agentsLinkCmd.Flags().StringVar(&agentsLinkAgent, "agent", "", "Link only a specific agent (e.g. claude, codex)")
	agentsLinkCmd.Flags().BoolVar(&agentsLinkCopy, "copy", false, "Copy files instead of creating symlinks")
	agentsLinkCmd.Flags().BoolVar(&agentsLinkForce, "force", false, "Overwrite existing files without requiring adopt first")

	agentsUnlinkCmd.Flags().StringVar(&agentsUnlinkAgent, "agent", "", "Unlink only a specific agent (e.g. claude, codex)")
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

var agentsLinkCmd = &cobra.Command{
	Use:   "link",
	Short: "Create symlinks from the canonical store to each detected agent's config locations",
	Long: `Create symlinks from the canonical agents store to each detected agent's
expected configuration locations. Only config types that exist in the store are linked
(e.g. skips skills/ if it is empty). Use --copy to create file copies instead of
symlinks. Use --force to overwrite existing non-symlink files.`,
	RunE: hook.Wrap("agents.link", runAgentsLink),
}

var agentsUnlinkCmd = &cobra.Command{
	Use:   "unlink",
	Short: "Replace symlinks with standalone copies, restoring independent configs",
	Long: `Replace agent config symlinks with standalone file copies, restoring each
agent's configuration to an independent state. After unlinking, changes to the
canonical store will no longer propagate to the agent configs.`,
	RunE: hook.Wrap("agents.unlink", runAgentsUnlink),
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
	fmt.Printf("  %s %s %s %s\n",
		ui.KeyStyle.Render(fmt.Sprintf("%-14s", "Agent")),
		ui.KeyStyle.Render(fmt.Sprintf("%-30s", "Binary")),
		ui.KeyStyle.Render(fmt.Sprintf("%-28s", "Config Dir")),
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

	// Pad plain text before rendering so ANSI codes don't inflate column widths.
	var binaryDisplay string
	if a.Binary == "" {
		binaryDisplay = ui.Muted.Render(fmt.Sprintf("%-30s", "not in PATH"))
	} else {
		binaryDisplay = fmt.Sprintf("%-30s", a.Binary)
	}

	var configDisplay string
	if !agents.DirExists(a.ConfigDir) {
		configDisplay = ui.Muted.Render(fmt.Sprintf("%-28s", "not found"))
	} else {
		configDisplay = fmt.Sprintf("%-28s", a.ConfigDir)
	}

	fmt.Printf("  %-14s %s %s %s\n", a.Name, binaryDisplay, configDisplay, statusStr)
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
		return fmt.Errorf("reading manifest: %w", err)
	}

	// Replace agents list entirely — re-running detect is always idempotent.
	m.Agents = detected

	return agents.WriteManifest(m)
}

func runAgentsLink(_ *cobra.Command, _ []string) error {
	if !agents.IsInitialized() {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No agents store yet."))
		fmt.Printf("  Run %s first.\n", ui.Accent.Render("mine agents init"))
		fmt.Println()
		return nil
	}

	opts := agents.LinkOptions{
		Agent: agentsLinkAgent,
		Copy:  agentsLinkCopy,
		Force: agentsLinkForce,
	}

	actions, err := agents.Link(opts)
	if err != nil {
		return err
	}

	fmt.Println()
	if len(actions) == 0 {
		fmt.Println(ui.Muted.Render("  No links created — run " + ui.Accent.Render("mine agents detect") + ui.Muted.Render(" to register detected agents.")))
		fmt.Println()
		return nil
	}

	createdCount := 0
	for _, a := range actions {
		printLinkAction(a)
		if a.Err == nil {
			createdCount++
		}
	}

	fmt.Println()
	if createdCount > 0 {
		ui.Ok(fmt.Sprintf("%d link(s) configured", createdCount))
	}
	fmt.Println()
	return nil
}

func runAgentsUnlink(_ *cobra.Command, _ []string) error {
	if !agents.IsInitialized() {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No agents store yet."))
		fmt.Printf("  Run %s first.\n", ui.Accent.Render("mine agents init"))
		fmt.Println()
		return nil
	}

	opts := agents.UnlinkOptions{
		Agent: agentsUnlinkAgent,
	}

	actions, err := agents.Unlink(opts)
	if err != nil {
		return err
	}

	fmt.Println()
	if len(actions) == 0 {
		fmt.Println(ui.Muted.Render("  No links to remove."))
		fmt.Println()
		return nil
	}

	unlinkedCount := 0
	for _, a := range actions {
		printUnlinkAction(a)
		if a.Err == nil {
			unlinkedCount++
		}
	}

	fmt.Println()
	if unlinkedCount > 0 {
		ui.Ok(fmt.Sprintf("%d link(s) removed — configs are now standalone", unlinkedCount))
	}
	fmt.Println()
	return nil
}

// printLinkAction prints a single link result row.
func printLinkAction(a agents.LinkAction) {
	var statusStr string
	switch {
	case a.Err != nil:
		statusStr = ui.Warning.Render(ui.IconWarn + a.Err.Error())
		fmt.Printf("  %-10s %-10s %s\n", a.Agent, a.Source, statusStr)
	case a.Status == "updated":
		statusStr = ui.Muted.Render(ui.IconOk + "already linked")
		fmt.Printf("  %-10s %-10s %s\n", a.Agent, a.Source, statusStr)
	default:
		modeStr := "symlink"
		if a.Mode == "copy" {
			modeStr = "copy"
		}
		statusStr = ui.Success.Render(ui.IconOk + a.Status + " (" + modeStr + ")")
		fmt.Printf("  %-10s %-10s %s %s\n", a.Agent, a.Source, ui.Muted.Render(ui.IconArrow), statusStr)
	}
}

// printUnlinkAction prints a single unlink result row.
func printUnlinkAction(a agents.UnlinkAction) {
	if a.Err != nil {
		fmt.Printf("  %-10s %s %s\n", a.Agent, ui.Warning.Render(ui.IconWarn), ui.Warning.Render(a.Err.Error()))
		return
	}
	fmt.Printf("  %-10s %s %s\n", a.Agent, ui.Muted.Render(a.Target), ui.Success.Render(ui.IconOk+"unlinked"))
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

