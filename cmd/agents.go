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
	Short: "Manage and sync coding agent configurations",
	Long:  `A canonical store for your coding agent configs — one source of truth, distributed everywhere.`,
	RunE:  hook.Wrap("agents", runAgentsStatus),
}

func init() {
	rootCmd.AddCommand(agentsCmd)
	agentsCmd.AddCommand(agentsInitCmd)
	agentsCmd.AddCommand(agentsSyncCmd)

	agentsSyncCmd.AddCommand(agentsSyncRemoteCmd)
	agentsSyncCmd.AddCommand(agentsSyncPushCmd)
	agentsSyncCmd.AddCommand(agentsSyncPullCmd)
}

var agentsInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Set up the canonical agent config store",
	Long: `Create the canonical agent configuration store at ~/.local/share/mine/agents/.

Scaffolds a directory structure for instructions, skills, commands, settings,
MCP configs, and rules. Initializes a git repository for version tracking.`,
	RunE: hook.Wrap("agents.init", runAgentsInit),
}

var agentsSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync agent configs with a git remote",
	Long: `Back up and sync your canonical agent config store with a git remote.

  mine agents sync remote <url>   Set the remote repository URL
  mine agents sync remote         Show the current remote URL
  mine agents sync push           Push store to remote
  mine agents sync pull           Pull from remote and re-distribute`,
	RunE: hook.Wrap("agents.sync", runAgentsSyncHelp),
}

var agentsSyncRemoteCmd = &cobra.Command{
	Use:   "remote [url]",
	Short: "Get or set the sync remote URL",
	Long: `Configure the upstream git remote for the agents store.

With no arguments, shows the current remote URL.
With a URL argument, sets (or updates) the remote.`,
	Args: cobra.MaximumNArgs(1),
	RunE: hook.Wrap("agents.sync.remote", runAgentsSyncRemote),
}

var agentsSyncPushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push agent configs to the remote",
	RunE:  hook.Wrap("agents.sync.push", runAgentsSyncPush),
}

var agentsSyncPullCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull agent configs from the remote",
	Long: `Pull from the configured remote with rebase.

After pulling, copy-mode links are automatically re-distributed to their target
agent directories. Symlink-mode links are already up-to-date via the symlink.`,
	RunE: hook.Wrap("agents.sync.pull", runAgentsSyncPull),
}

func runAgentsStatus(_ *cobra.Command, _ []string) error {
	if !agents.IsInitialized() {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  Agents store not initialized."))
		fmt.Printf("  Get started: %s\n", ui.Accent.Render("mine agents init"))
		fmt.Println()
		return nil
	}

	dir := agents.Dir()
	fmt.Println()
	ui.Kv("store", dir)

	if agents.IsGitRepo() {
		logs, err := agents.Log("")
		if err == nil && len(logs) > 0 {
			fmt.Printf("  %s\n", ui.Muted.Render(fmt.Sprintf("%d commits", len(logs))))
		}

		remote := agents.SyncRemoteURL()
		if remote != "" {
			ui.Kv("remote", remote)
		}
	}

	manifest, err := agents.ReadManifest()
	if err == nil {
		if len(manifest.Links) > 0 {
			fmt.Printf("  %s\n", ui.Muted.Render(fmt.Sprintf("%d links", len(manifest.Links))))
		}
	}

	fmt.Println()
	return nil
}

func runAgentsInit(_ *cobra.Command, _ []string) error {
	if err := agents.Init(); err != nil {
		return err
	}

	dir := agents.Dir()
	fmt.Println()
	ui.Ok("Agents store ready")
	fmt.Printf("  Location: %s\n", ui.Muted.Render(dir))
	fmt.Println()
	fmt.Println("  Next steps:")
	fmt.Printf("    1. Set a remote: %s\n", ui.Accent.Render("mine agents sync remote <url>"))
	fmt.Printf("    2. Push to it:   %s\n", ui.Accent.Render("mine agents sync push"))
	fmt.Println()
	return nil
}

func runAgentsSyncHelp(_ *cobra.Command, _ []string) error {
	fmt.Println()
	fmt.Println("  Sync your agent configs with a git remote.")
	fmt.Println()
	fmt.Printf("  %s   Set or show the remote URL\n", ui.Accent.Render("mine agents sync remote"))
	fmt.Printf("  %s          Push store to remote\n", ui.Accent.Render("mine agents sync push"))
	fmt.Printf("  %s          Pull from remote\n", ui.Accent.Render("mine agents sync pull"))
	fmt.Println()
	return nil
}

func runAgentsSyncRemote(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		// Show current remote.
		url := agents.SyncRemoteURL()
		if url == "" {
			fmt.Println()
			fmt.Println(ui.Muted.Render("  No remote configured."))
			fmt.Printf("  Set one: %s\n", ui.Accent.Render("mine agents sync remote <url>"))
			fmt.Println()
		} else {
			fmt.Println()
			ui.Kv("remote", url)
			fmt.Println()
		}
		return nil
	}

	url := args[0]
	if err := agents.SyncSetRemote(url); err != nil {
		return err
	}
	fmt.Println()
	ui.Ok(fmt.Sprintf("Remote set to %s", url))
	fmt.Println()
	return nil
}

func runAgentsSyncPush(_ *cobra.Command, _ []string) error {
	if err := agents.SyncPush(); err != nil {
		return err
	}
	fmt.Println()
	ui.Ok("Agent configs pushed to remote — your configs are safe in the cloud")
	fmt.Println()
	return nil
}

func runAgentsSyncPull(_ *cobra.Command, _ []string) error {
	result, err := agents.SyncPullWithResult()
	if err != nil {
		return err
	}
	fmt.Println()
	ui.Ok("Agent configs pulled from remote")
	if result.CopiedLinks > 0 {
		fmt.Printf("  %s\n", ui.Muted.Render(fmt.Sprintf("%d copy-mode link(s) re-distributed", result.CopiedLinks)))
	}
	fmt.Println()
	return nil
}
