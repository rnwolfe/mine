package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/stash"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

var stashCmd = &cobra.Command{
	Use:   "stash",
	Short: "Git-backed dotfile snapshots — your configs, version controlled",
	Long:  `Track, snapshot, and sync your dotfiles. Never lose your carefully tuned configs again.`,
	RunE:  hook.Wrap("stash", runStashStatus),
}

func init() {
	rootCmd.AddCommand(stashCmd)
	stashCmd.AddCommand(stashTrackCmd)
	stashCmd.AddCommand(stashListCmd)
	stashCmd.AddCommand(stashInitCmd)
	stashCmd.AddCommand(stashDiffCmd)
	stashCmd.AddCommand(stashCommitCmd)
	stashCmd.AddCommand(stashLogCmd)
	stashCmd.AddCommand(stashRestoreCmd)
	stashCmd.AddCommand(stashSyncCmd)

	stashCommitCmd.Flags().StringP("message", "m", "", "Commit message")
	stashRestoreCmd.Flags().StringP("version", "v", "", "Version hash to restore (default: latest)")
	stashRestoreCmd.Flags().BoolP("force", "f", false, "Override destination file permissions with stash-recorded permissions")
}

var stashInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Set up dotfile tracking in this directory",
	RunE:  hook.Wrap("stash.init", runStashInit),
}

var stashTrackCmd = &cobra.Command{
	Use:   "track <file>",
	Short: "Start tracking a dotfile",
	Args:  cobra.ExactArgs(1),
	RunE:  hook.Wrap("stash.track", runStashTrack),
}

var stashListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show all tracked dotfiles",
	RunE:  hook.Wrap("stash.list", runStashList),
}

var stashDiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "See what's changed since the last snapshot",
	RunE:  hook.Wrap("stash.diff", runStashDiff),
}

var stashCommitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Snapshot all tracked dotfiles",
	Long:  `Create a versioned snapshot of all tracked dotfiles. Initializes git tracking on first use.`,
	RunE:  hook.Wrap("stash.commit", runStashCommit),
}

var stashLogCmd = &cobra.Command{
	Use:   "log [file]",
	Short: "Browse snapshot history",
	Args:  cobra.MaximumNArgs(1),
	RunE:  hook.Wrap("stash.log", runStashLog),
}

var stashRestoreCmd = &cobra.Command{
	Use:   "restore <file>",
	Short: "Restore a dotfile to a previous snapshot",
	Args:  cobra.ExactArgs(1),
	RunE:  hook.Wrap("stash.restore", runStashRestore),
}

var stashSyncCmd = &cobra.Command{
	Use:   "sync <push|pull|remote>",
	Short: "Back up your stash to a git remote",
	Long: `Sync your stash with a git remote. Opt-in cloud backup.

  mine stash sync remote <url>   Set the remote repository URL
  mine stash sync push           Push stash to remote
  mine stash sync pull           Pull stash from remote`,
	Args: cobra.RangeArgs(1, 2),
	RunE: hook.Wrap("stash.sync", runStashSync),
}

func runStashInit(_ *cobra.Command, _ []string) error {
	dir := stash.Dir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// Create manifest if it doesn't exist.
	manifestPath := stash.ManifestPath()
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		if err := os.WriteFile(manifestPath, []byte("# mine stash manifest\n# each line: source_path -> safe_name (e.g. ~/.zshrc -> zshrc)\n"), 0o644); err != nil {
			return err
		}
	}

	ui.Ok("Stash ready — your dotfile vault is open")
	fmt.Printf("  Location: %s\n", ui.Muted.Render(dir))
	fmt.Println()
	fmt.Printf("  Start tracking: %s\n", ui.Accent.Render("mine stash track ~/.zshrc"))
	fmt.Println()
	return nil
}

func runStashTrack(_ *cobra.Command, args []string) error {
	source := args[0]

	// Expand ~ to home dir.
	if strings.HasPrefix(source, "~") {
		home, _ := os.UserHomeDir()
		source = filepath.Join(home, source[1:])
	}

	source, err := filepath.Abs(source)
	if err != nil {
		return err
	}

	entry, err := stash.TrackFile(source)
	if err != nil {
		return err
	}

	home, _ := os.UserHomeDir()
	relPath := strings.TrimPrefix(entry.Source, home+"/")
	dest := filepath.Join(stash.Dir(), entry.SafeName)

	ui.Ok(fmt.Sprintf("Tracking %s", relPath))
	fmt.Printf("  Stashed to: %s\n", ui.Muted.Render(dest))
	fmt.Println()
	return nil
}

func runStashList(_ *cobra.Command, _ []string) error {
	entries, err := stash.ReadManifest()
	if err != nil {
		return fmt.Errorf("failed to read stash manifest: %w", err)
	}

	// entries == nil means manifest doesn't exist (no stash initialized).
	// len(entries) == 0 means manifest exists but no files tracked yet.
	if entries == nil {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No stash yet."))
		fmt.Printf("  Run %s first.\n", ui.Accent.Render("mine stash init"))
		fmt.Println()
		return nil
	}

	if len(entries) == 0 {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No files tracked yet."))
		fmt.Printf("  Try: %s\n", ui.Accent.Render("mine stash track ~/.zshrc"))
		fmt.Println()
		return nil
	}

	home, _ := os.UserHomeDir()
	fmt.Println()
	for _, e := range entries {
		display := strings.Replace(e.Source, home, "~", 1)
		fmt.Printf("  %s %s\n", ui.Success.Render("●"), display)
	}
	fmt.Println()
	fmt.Println(ui.Muted.Render(fmt.Sprintf("  %d files tracked", len(entries))))

	if stash.IsGitRepo() {
		logs, err := stash.Log("")
		if err == nil && len(logs) > 0 {
			fmt.Println(ui.Muted.Render(fmt.Sprintf("  %d snapshots", len(logs))))
		}
	}

	fmt.Println()
	return nil
}

func runStashDiff(_ *cobra.Command, _ []string) error {
	dir := stash.Dir()
	entries, err := stash.ReadManifest()
	if err != nil {
		return err
	}
	if entries == nil {
		return fmt.Errorf("stash not initialized — run `mine stash init` to get started")
	}

	if len(entries) == 0 {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No files tracked yet."))
		fmt.Printf("  Try: %s\n", ui.Accent.Render("mine stash track ~/.zshrc"))
		fmt.Println()
		return nil
	}

	changes := 0
	fmt.Println()
	for _, e := range entries {
		sourceData, err := os.ReadFile(e.Source)
		if err != nil {
			fmt.Printf("  %s %s (missing!)\n", ui.Error.Render("✗"), e.Source)
			changes++
			continue
		}

		stashedData, err := os.ReadFile(filepath.Join(dir, e.SafeName))
		if err != nil {
			continue
		}

		if string(sourceData) != string(stashedData) {
			home, _ := os.UserHomeDir()
			display := strings.Replace(e.Source, home, "~", 1)
			fmt.Printf("  %s %s (modified)\n", ui.Warning.Render("~"), display)
			changes++
		}
	}

	if changes == 0 {
		fmt.Println(ui.Success.Render("  Everything in sync."))
	} else {
		fmt.Println()
		fmt.Println(ui.Muted.Render(fmt.Sprintf("  %d files changed since last stash", changes)))
	}
	fmt.Println()
	return nil
}

func runStashCommit(cmd *cobra.Command, _ []string) error {
	msg, _ := cmd.Flags().GetString("message")
	if msg == "" {
		msg = fmt.Sprintf("stash snapshot %s", time.Now().Format("2006-01-02 15:04"))
	}

	entries, err := stash.ReadManifest()
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return fmt.Errorf("nothing to snapshot — add a file first with `mine stash track ~/.zshrc`")
	}

	hash, err := stash.Commit(msg)
	if err != nil {
		return err
	}

	fmt.Println()
	ui.Ok(fmt.Sprintf("Snapshot saved %s", ui.Muted.Render("["+hash+"]")))
	fmt.Printf("  %s\n", ui.Muted.Render(msg))
	fmt.Printf("  Your dotfiles are safe. %s\n", ui.Muted.Render("restore anytime with `mine stash restore <file>`"))
	fmt.Println()
	return nil
}

func runStashLog(_ *cobra.Command, args []string) error {
	file := ""
	if len(args) > 0 {
		file = args[0]
	}

	logs, err := stash.Log(file)
	if err != nil {
		return err
	}

	if len(logs) == 0 {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No history yet."))
		fmt.Printf("  Run %s to create a snapshot.\n", ui.Accent.Render("mine stash commit"))
		fmt.Println()
		return nil
	}

	fmt.Println()
	for _, entry := range logs {
		age := formatAge(entry.Date)
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

func runStashRestore(cmd *cobra.Command, args []string) error {
	file := args[0]
	version, _ := cmd.Flags().GetString("version")
	force, _ := cmd.Flags().GetBool("force")

	// RestoreToSource returns the Entry, avoiding duplicate FindEntry calls
	entry, err := stash.RestoreToSource(file, version, force)
	if err != nil {
		return err
	}

	home, _ := os.UserHomeDir()
	display := strings.Replace(entry.Source, home, "~", 1)

	versionLabel := "latest"
	if version != "" {
		versionLabel = version
	}

	fmt.Println()
	ui.Ok(fmt.Sprintf("Restored %s to %s", display, versionLabel))
	fmt.Println()
	return nil
}

func runStashSync(_ *cobra.Command, args []string) error {
	action := args[0]
	switch action {
	case "remote":
		if len(args) < 2 {
			// Show current remote.
			url := stash.SyncRemoteURL()
			if url == "" {
				fmt.Println()
				fmt.Println(ui.Muted.Render("  No remote configured."))
				fmt.Printf("  Set one: %s\n", ui.Accent.Render("mine stash sync remote <url>"))
				fmt.Println()
			} else {
				fmt.Println()
				ui.Kv("remote", url)
				fmt.Println()
			}
			return nil
		}
		url := args[1]
		if err := stash.SyncSetRemote(url); err != nil {
			return err
		}
		fmt.Println()
		ui.Ok(fmt.Sprintf("Remote set to %s", url))
		fmt.Println()
		return nil

	case "push":
		if err := stash.SyncPush(); err != nil {
			return err
		}
		fmt.Println()
		ui.Ok("Stash backed up to remote — your configs are safe in the cloud")
		fmt.Println()
		return nil

	case "pull":
		if err := stash.SyncPull(); err != nil {
			return err
		}
		fmt.Println()
		ui.Ok("Stash pulled and restored — welcome back to your setup")
		fmt.Println()
		return nil

	default:
		return fmt.Errorf("unknown sync action %q — use push, pull, or remote", action)
	}
}

func runStashStatus(_ *cobra.Command, _ []string) error {
	return runStashList(nil, nil)
}

func formatAge(t time.Time) string {
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
