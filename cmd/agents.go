package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/rnwolfe/mine/internal/agents"
	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

var agentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "Manage and version-control your AI agent configurations",
	Long:  `Canonical store for AI agent configs — track, snapshot, and restore your agent setups.`,
	RunE:  hook.Wrap("agents", runAgentsStatus),
}

func init() {
	rootCmd.AddCommand(agentsCmd)
	agentsCmd.AddCommand(agentsInitCmd)
	agentsCmd.AddCommand(agentsCommitCmd)
	agentsCmd.AddCommand(agentsLogCmd)
	agentsCmd.AddCommand(agentsRestoreCmd)

	agentsCommitCmd.Flags().StringP("message", "m", "", "Commit message")
	agentsRestoreCmd.Flags().StringP("version", "v", "", "Version hash to restore (default: latest)")
}

var agentsInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Set up the canonical agent configuration store",
	Long:  `Create the canonical store at ~/.local/share/mine/agents/ with directory scaffold and git versioning.`,
	RunE:  hook.Wrap("agents.init", runAgentsInit),
}

var agentsCommitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Snapshot the current state of your agent configs",
	Long:  `Stage and commit all changes in the canonical store. Initializes git versioning on first use.`,
	RunE:  hook.Wrap("agents.commit", runAgentsCommit),
}

var agentsLogCmd = &cobra.Command{
	Use:   "log [file]",
	Short: "Show version history of agent configs",
	Args:  cobra.MaximumNArgs(1),
	RunE:  hook.Wrap("agents.log", runAgentsLog),
}

var agentsRestoreCmd = &cobra.Command{
	Use:   "restore <file>",
	Short: "Restore an agent config file to a previous version",
	Long: `Restore a file in the canonical store to a previous snapshot.
Symlinked agent directories reflect the change immediately.
Copy-mode links are re-synced automatically.

Example:
  mine agents restore instructions/AGENTS.md
  mine agents restore settings/claude.json --version abc1234`,
	Args: cobra.ExactArgs(1),
	RunE: hook.Wrap("agents.restore", runAgentsRestore),
}

func runAgentsInit(_ *cobra.Command, _ []string) error {
	dir := agents.Dir()
	alreadyInit := agents.IsInitialized()

	if err := agents.Init(); err != nil {
		return err
	}

	if alreadyInit {
		fmt.Println()
		ui.Ok("Agents store already initialized")
		fmt.Printf("  Location: %s\n", ui.Muted.Render(dir))
		fmt.Println()
		return nil
	}

	fmt.Println()
	ui.Ok("Agents store ready — your AI configs have a home")
	fmt.Printf("  Location: %s\n", ui.Muted.Render(dir))
	fmt.Println()
	fmt.Println(ui.Muted.Render("  Directory layout:"))
	subdirs := []string{"instructions/", "skills/", "commands/", "agents/", "settings/", "mcp/", "rules/"}
	for _, subdir := range subdirs {
		fmt.Printf("    %s %s\n", ui.Accent.Render("→"), subdir)
	}
	fmt.Println()
	fmt.Printf("  Edit %s to add shared instructions for all your agents.\n",
		ui.Accent.Render("instructions/AGENTS.md"))
	fmt.Printf("  Snapshot anytime: %s\n", ui.Accent.Render("mine agents commit"))
	fmt.Println()
	return nil
}

func runAgentsCommit(cmd *cobra.Command, _ []string) error {
	msg, _ := cmd.Flags().GetString("message")
	if msg == "" {
		msg = fmt.Sprintf("agents snapshot %s", time.Now().Format("2006-01-02 15:04"))
	}

	if !agents.IsInitialized() {
		return fmt.Errorf("agents store not initialized — run %s first", ui.Accent.Render("mine agents init"))
	}

	hash, err := agents.Commit(msg)
	if err != nil {
		// Surface "nothing to commit" as a friendly message, not an error.
		if strings.Contains(err.Error(), "nothing to commit") {
			fmt.Println()
			fmt.Println(ui.Muted.Render("  Nothing to commit — all agent configs are up to date."))
			fmt.Println()
			return nil
		}
		return err
	}

	fmt.Println()
	ui.Ok(fmt.Sprintf("Snapshot saved %s", ui.Muted.Render("["+hash+"]")))
	fmt.Printf("  %s\n", ui.Muted.Render(msg))
	fmt.Printf("  Restore anytime: %s\n", ui.Muted.Render("mine agents restore <file>"))
	fmt.Println()
	return nil
}

func runAgentsLog(_ *cobra.Command, args []string) error {
	file := ""
	if len(args) > 0 {
		file = args[0]
	}

	if !agents.IsInitialized() {
		return fmt.Errorf("agents store not initialized — run %s first", ui.Accent.Render("mine agents init"))
	}

	logs, err := agents.Log(file)
	if err != nil {
		// Surface "no version history" as a friendly message.
		if strings.Contains(err.Error(), "no version history") {
			fmt.Println()
			fmt.Println(ui.Muted.Render("  No history yet."))
			fmt.Printf("  Run %s to create a snapshot.\n", ui.Accent.Render("mine agents commit"))
			fmt.Println()
			return nil
		}
		return err
	}

	if len(logs) == 0 {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No history yet."))
		fmt.Printf("  Run %s to create a snapshot.\n", ui.Accent.Render("mine agents commit"))
		fmt.Println()
		return nil
	}

	fmt.Println()
	for _, entry := range logs {
		age := agentsFormatAge(entry.Date)
		fmt.Printf("  %s %s %s\n",
			ui.Accent.Render(entry.Short),
			entry.Message,
			ui.Muted.Render("("+age+")"),
		)
	}
	fmt.Println()
	fmt.Println(ui.Muted.Render(fmt.Sprintf("  %d snapshots", len(logs))))
	if file != "" {
		fmt.Println(ui.Muted.Render(fmt.Sprintf("  filtered to: %s", file)))
	}
	fmt.Println()
	return nil
}

func runAgentsRestore(cmd *cobra.Command, args []string) error {
	file := args[0]
	version, _ := cmd.Flags().GetString("version")

	if !agents.IsInitialized() {
		return fmt.Errorf("agents store not initialized — run %s first", ui.Accent.Render("mine agents init"))
	}

	updated, err := agents.RestoreToStore(file, version)
	if err != nil {
		return err
	}

	versionLabel := "latest"
	if version != "" {
		versionLabel = version
	}

	fmt.Println()
	ui.Ok(fmt.Sprintf("Restored %s to %s", ui.Accent.Render(file), versionLabel))
	if len(updated) > 0 {
		fmt.Printf("  Re-synced %d copy-mode link(s):\n", len(updated))
		for _, link := range updated {
			fmt.Printf("    %s %s\n", ui.Muted.Render("→"), link.Target)
		}
	}
	fmt.Println()
	return nil
}

func runAgentsStatus(_ *cobra.Command, _ []string) error {
	dir := agents.Dir()

	if !agents.IsInitialized() {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No agents store yet."))
		fmt.Printf("  Run %s to get started.\n", ui.Accent.Render("mine agents init"))
		fmt.Println()
		return nil
	}

	fmt.Println()
	fmt.Printf("  %s %s\n", ui.Success.Render("●"), ui.Muted.Render(dir))

	if agents.IsGitRepo() {
		logs, err := agents.Log("")
		if err == nil && len(logs) > 0 {
			fmt.Println(ui.Muted.Render(fmt.Sprintf("  %d snapshots", len(logs))))
			fmt.Printf("  Latest: %s %s\n",
				ui.Accent.Render(logs[0].Short),
				ui.Muted.Render(logs[0].Message),
			)
		}
	}

	fmt.Println()
	fmt.Printf("  %s  %s  %s\n",
		ui.Muted.Render("mine agents commit"),
		ui.Muted.Render("mine agents log"),
		ui.Muted.Render("mine agents restore <file>"),
	)
	fmt.Println()
	return nil
}

// agentsFormatAge formats a time as a human-readable age string.
// Mirrors the formatAge function used by stash but avoids cross-file function sharing.
func agentsFormatAge(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		m := int(d.Minutes())
		if m == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", m)
	case d < 24*time.Hour:
		h := int(d.Hours())
		if h == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", h)
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}
