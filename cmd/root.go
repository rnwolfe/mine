package cmd

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/rnwolfe/mine/internal/analytics"
	"github.com/rnwolfe/mine/internal/config"
	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/plugin"
	"github.com/rnwolfe/mine/internal/proj"
	"github.com/rnwolfe/mine/internal/store"
	"github.com/rnwolfe/mine/internal/tips"
	"github.com/rnwolfe/mine/internal/todo"
	"github.com/rnwolfe/mine/internal/tui"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/rnwolfe/mine/internal/version"
	"github.com/spf13/cobra"
)

var dashPlain bool

var rootCmd = &cobra.Command{
	Use:   "mine",
	Short: "Your personal developer supercharger",
	Long:  `mine — todos, secrets, env profiles, dotfiles, git helpers, and more. All in one binary.`,
	RunE:  hook.Wrap("mine", runDashboard),
	// NOTE: Cobra's PersistentPostRun on rootCmd fires for ALL subcommands.
	// If any subcommand defines its own PersistentPostRun, it will shadow this one
	// and analytics will not fire for that subtree. Avoid this pattern on subcommands.
	PersistentPostRun: func(cmd *cobra.Command, _ []string) {
		fireAnalytics(topLevelCommand(cmd))
	},
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd: true,
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() {
	// Register user-local hooks and plugin hooks at startup.
	// Errors are non-fatal — the CLI should work without hooks/plugins.
	if err := hook.RegisterUserHooks(); err != nil {
		log.Printf("warning: loading user hooks: %v", err)
	}
	if err := plugin.RegisterPluginHooks(); err != nil {
		log.Printf("warning: loading plugin hooks: %v", err)
	}

	if err := rootCmd.Execute(); err != nil {
		ui.Err(err.Error())
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(todoCmd)
	rootCmd.AddCommand(aiCmd)
	rootCmd.AddCommand(vaultCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(aboutCmd)
	rootCmd.AddCommand(tipsCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(growCmd)
	rootCmd.Flags().BoolVar(&dashPlain, "plain", false, "Print static text dashboard instead of launching the TUI")
}

// fireAnalytics sends an anonymous analytics ping synchronously.
// It's a no-op if config is not initialized, analytics are disabled,
// or the store can't be opened.
func fireAnalytics(command string) {
	if !config.Initialized() {
		return
	}

	cfg, err := config.Load()
	if err != nil {
		return
	}

	if !cfg.Analytics.IsEnabled() {
		return
	}

	db, err := store.Open()
	if err != nil {
		return
	}

	endpoint := os.Getenv("MINE_ANALYTICS_ENDPOINT")
	if endpoint == "" {
		endpoint = analytics.DefaultEndpoint
	}

	// Show one-time privacy notice if needed (stderr to avoid contaminating stdout)
	if analytics.ShouldShowNotice(db.Conn()) {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, ui.Muted.Render("  mine sends anonymous usage stats (command names, version, OS) to help"))
		fmt.Fprintln(os.Stderr, ui.Muted.Render("  improve the tool. No personal data is ever collected."))
		fmt.Fprintf(os.Stderr, "  Opt out anytime: %s\n", ui.Accent.Render("mine config set analytics false"))
		fmt.Fprintln(os.Stderr)
		analytics.MarkNoticeShown(db.Conn())
	}

	// Synchronous call — Ping uses a 2s HTTP timeout and daily dedup means it
	// almost never hits the network. Running synchronously avoids a race between
	// the goroutine and process exit that could lose the dedup write or leave
	// the SQLite connection in a bad state.
	analytics.Ping(db.Conn(), command, cfg.Analytics.IsEnabled(), endpoint)
	db.Close()
}

// topLevelCommand extracts the top-level command name from a Cobra command.
// For example, "mine todo add" returns "todo", and "mine" returns "mine".
func topLevelCommand(cmd *cobra.Command) string {
	parts := strings.Fields(cmd.CommandPath())
	switch {
	case len(parts) >= 2:
		return parts[1] // First word after "mine"
	case len(parts) == 1:
		return parts[0] // Root command itself
	default:
		return "unknown"
	}
}

// runDashboard shows the at-a-glance status when you just type `mine`.
// In a TTY with config initialized and without --plain, it launches the TUI dashboard.
func runDashboard(_ *cobra.Command, _ []string) error {
	// Launch TUI when: connected to a terminal, config is initialized, --plain not set.
	if tui.IsTTY() && !dashPlain && config.Initialized() {
		return runDashTUI()
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if !config.Initialized() {
		fmt.Println(ui.Greet(""))
		fmt.Println()
		fmt.Println("  " + ui.Subtitle.Render("Your personal developer supercharger. Here's what's waiting:"))
		fmt.Println()
		features := []struct{ cmd, desc string }{
			{"mine todo", "Capture and track tasks without leaving the terminal"},
			{"mine proj", "Jump between projects instantly with the p helper"},
			{"mine ai ask", "Get AI answers and code review without browser tabs"},
			{"mine stash", "Version-control any file, not just code"},
			{"mine vault", "Encrypted secrets — no more plaintext .env files"},
		}
		for _, f := range features {
			fmt.Printf("  %s %-14s %s\n",
				ui.Accent.Render("✦"),
				ui.Accent.Render(f.cmd),
				ui.Muted.Render(f.desc),
			)
		}
		fmt.Println()
		fmt.Printf("  Run %s to get set up in about 30 seconds.\n", ui.Accent.Render("mine init"))
		fmt.Println()
		return nil
	}

	// Greeting
	name := cfg.User.Name
	fmt.Println(ui.Greet(name))
	fmt.Println()

	// Todo summary
	db, err := store.Open()
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	defer db.Close()

	ps := proj.NewStore(db.Conn())
	currentProject, _ := ps.FindForCWD()

	// Scope todo count to the current working directory's project when inside one.
	var projPath *string
	if currentProject != nil {
		projPath = &currentProject.Path
	}

	ts := todo.NewStore(db.Conn())
	open, total, overdue, err := ts.Count(projPath)
	if err != nil {
		return fmt.Errorf("counting todos: %w", err)
	}

	todoSummary := fmt.Sprintf("%d open", open)
	if total > 0 {
		todoSummary += fmt.Sprintf(" / %d total", total)
	}
	if overdue > 0 {
		todoSummary += ui.Error.Render(fmt.Sprintf(" (%d overdue!)", overdue))
	}
	ui.Kv(ui.IconTodo+" Todos", todoSummary)

	if currentProject != nil {
		projectSummary := currentProject.Name
		if currentProject.Branch != "" {
			projectSummary += fmt.Sprintf(" (%s)", currentProject.Branch)
		}
		ui.Kv("  "+ui.IconProject+" Project", projectSummary)
	}

	// Date/time
	now := time.Now()
	ui.Kv("  "+ui.IconCalendar+" Today", now.Format("Monday, January 2"))

	// Version
	ui.Kv("  "+ui.IconSettings+" Mine", version.Short())

	// Tip
	if open > 0 && overdue > 0 {
		ui.Tip("`mine todo` to tackle that overdue task.")
	} else if open > 0 {
		ui.Tip("`mine todo` to see what's on your plate.")
	} else {
		ui.Tip(dashboardTip(now))
	}

	fmt.Println()
	return nil
}

// dashboardTip returns a daily rotating tip for the dashboard.
func dashboardTip(t time.Time) string {
	return tips.Daily(t)
}
