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
	"github.com/rnwolfe/mine/internal/store"
	"github.com/rnwolfe/mine/internal/todo"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/rnwolfe/mine/internal/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mine",
	Short: "Your personal developer supercharger",
	Long:  `mine — everything you need, nothing you don't. Radically yours.`,
	RunE:  hook.Wrap("mine", runDashboard),
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
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(aboutCmd)
}

// fireAnalytics sends an anonymous analytics ping in the background.
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
	if analytics.ShowNotice(db.Conn()) {
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, ui.Muted.Render("  mine sends anonymous usage stats (command names, version, OS) to help"))
		fmt.Fprintln(os.Stderr, ui.Muted.Render("  improve the tool. No personal data is ever collected."))
		fmt.Fprintf(os.Stderr, "  Opt out anytime: %s\n", ui.Accent.Render("mine config set analytics false"))
		fmt.Fprintln(os.Stderr)
	}

	// Fire-and-forget: the goroutine outlives this function but is bounded by
	// the HTTP client timeout (2s). The main process exits normally.
	go func() {
		defer db.Close()
		analytics.Ping(db.Conn(), command, cfg.Analytics.IsEnabled(), endpoint)
	}()
}

// topLevelCommand extracts the top-level command name from a Cobra command.
// For example, "mine todo add" returns "todo", and "mine" returns "mine".
func topLevelCommand(cmd *cobra.Command) string {
	parts := strings.Fields(cmd.CommandPath())
	if len(parts) >= 2 {
		return parts[1] // First word after "mine"
	}
	return parts[0] // Root command itself
}

// runDashboard shows the at-a-glance status when you just type `mine`.
func runDashboard(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if !config.Initialized() {
		fmt.Println(ui.Greet(""))
		fmt.Println()
		fmt.Println("  Looks like this is your first time. Let's set things up!")
		fmt.Println()
		fmt.Printf("  Run %s to get started.\n", ui.Accent.Render("mine init"))
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

	ts := todo.NewStore(db.Conn())
	open, total, overdue, err := ts.Count()
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

	// Date/time
	now := time.Now()
	ui.Kv("  📅 Today", now.Format("Monday, January 2"))

	// Version
	ui.Kv("  ⚙️  Mine", version.Short())

	// Tip
	if open > 0 && overdue > 0 {
		ui.Tip("`mine todo` to tackle that overdue task.")
	} else if open > 0 {
		ui.Tip("`mine todo` to see what's on your plate.")
	} else {
		ui.Tip("`mine todo add \"something awesome\"` to capture an idea.")
	}

	fmt.Println()
	return nil
}
