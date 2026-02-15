package cmd

import (
	"fmt"
	"log"
	"os"
	"time"

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
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)
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
