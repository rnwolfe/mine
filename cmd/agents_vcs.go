package cmd

import (
	"errors"
	"fmt"
	"time"

	"github.com/rnwolfe/mine/internal/agents"
	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

var agentsCommitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Snapshot the current state of the canonical store",
	Long:  `Stage and commit all changes in the canonical store. Initializes git versioning on first use.`,
	RunE:  hook.Wrap("agents.commit", runAgentsCommit),
}

var agentsLogCmd = &cobra.Command{
	Use:   "log [file]",
	Short: "Show version history of the canonical store",
	RunE:  hook.Wrap("agents.log", runAgentsLog),
}

var agentsRestoreCmd = &cobra.Command{
	Use:   "restore <file>",
	Short: "Restore an agent config file to a previous version",
	Long: `Restore a file in the canonical store to a previous snapshot.

The file argument must be a path relative to the canonical store, e.g.:

  mine agents restore instructions/AGENTS.md
  mine agents restore settings/claude.json --version abc1234`,
	Args: cobra.ExactArgs(1),
	RunE: hook.Wrap("agents.restore", runAgentsRestore),
}

func init() {
	agentsCmd.AddCommand(agentsCommitCmd)
	agentsCmd.AddCommand(agentsLogCmd)
	agentsCmd.AddCommand(agentsRestoreCmd)

	agentsCommitCmd.Flags().StringP("message", "m", "", "Commit message")
	agentsRestoreCmd.Flags().StringP("version", "v", "", "Version hash to restore (default: latest)")
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
		if errors.Is(err, agents.ErrNothingToCommit) {
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
		if errors.Is(err, agents.ErrNoVersionHistory) {
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

	updated, failed, err := agents.RestoreToStore(file, version)
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
	if len(failed) > 0 {
		fmt.Printf("  Warning: %d copy-mode link(s) could not be re-synced:\n", len(failed))
		for _, link := range failed {
			fmt.Printf("    %s %s\n", ui.Muted.Render("✗"), link.Target)
		}
	}
	fmt.Println()
	return nil
}

// agentsFormatAge formats a time as a human-readable age string.
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
