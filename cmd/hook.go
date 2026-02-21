package cmd

import (
	"fmt"

	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Automate mine with event-driven scripts",
	Long:  `Create scripts that fire before or after any mine command. Drop them in ~/.config/mine/hooks/.`,
	RunE:  hook.Wrap("hook", runHookList),
}

var hookListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all active hook scripts",
	RunE:  hook.Wrap("hook.list", runHookList),
}

var hookCreateCmd = &cobra.Command{
	Use:   "create <command-pattern> <stage>",
	Short: "Scaffold a new hook script",
	Long: `Create a starter hook script.

Examples:
  mine hook create todo.add preexec
  mine hook create "todo.*" notify
  mine hook create "*" postexec`,
	Args: cobra.ExactArgs(2),
	RunE: hook.Wrap("hook.create", runHookCreate),
}

var hookTestCmd = &cobra.Command{
	Use:   "test <file>",
	Short: "Dry-run a hook with sample input to verify it works",
	Args:  cobra.ExactArgs(1),
	RunE:  hook.Wrap("hook.test", runHookTest),
}

func init() {
	rootCmd.AddCommand(hookCmd)
	hookCmd.AddCommand(hookListCmd)
	hookCmd.AddCommand(hookCreateCmd)
	hookCmd.AddCommand(hookTestCmd)
}

func runHookList(_ *cobra.Command, _ []string) error {
	hooks, err := hook.Discover()
	if err != nil {
		return err
	}

	if len(hooks) == 0 {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No hooks found."))
		fmt.Println()
		fmt.Printf("  Hooks directory: %s\n", ui.Accent.Render(hook.HooksDir()))
		fmt.Println()
		fmt.Printf("  Create one: %s\n", ui.Accent.Render("mine hook create todo.add preexec"))
		fmt.Println()
		return nil
	}

	fmt.Println()
	fmt.Println(ui.Title.Render("  User Hooks"))
	fmt.Println()

	for _, h := range hooks {
		stageLabel := ui.Muted.Render(string(h.Stage))
		mode := "transform"
		if h.Stage == hook.StageNotify {
			mode = "notify"
		}
		modeLabel := ui.Muted.Render(mode)

		fmt.Printf("  %s %-20s %s  %s\n",
			ui.Success.Render("●"),
			ui.Accent.Render(h.Pattern),
			stageLabel,
			modeLabel,
		)
	}

	fmt.Println()
	fmt.Printf("  %s\n", ui.Muted.Render(fmt.Sprintf("%d hooks in %s", len(hooks), hook.HooksDir())))
	fmt.Println()
	return nil
}

func runHookCreate(_ *cobra.Command, args []string) error {
	pattern := args[0]
	stage, err := hook.ParseStageStr(args[1])
	if err != nil {
		return err
	}

	path, err := hook.CreateHookScript(pattern, stage)
	if err != nil {
		return err
	}

	ui.Ok(fmt.Sprintf("Created hook: %s", path))
	fmt.Println()
	fmt.Printf("  Pattern: %s\n", ui.Accent.Render(pattern))
	fmt.Printf("  Stage:   %s\n", ui.Accent.Render(string(stage)))
	fmt.Println()
	fmt.Printf("  Edit:    %s\n", ui.Accent.Render("$EDITOR "+path))
	fmt.Printf("  Test:    %s\n", ui.Accent.Render("mine hook test "+path))
	fmt.Println()
	return nil
}

func runHookTest(_ *cobra.Command, args []string) error {
	path := args[0]

	fmt.Println()
	fmt.Printf("  Testing: %s\n", ui.Accent.Render(path))
	fmt.Println()

	output, err := hook.TestHook(path)
	if err != nil {
		return err
	}

	ui.Ok("Hook executed successfully")
	if output != "" {
		fmt.Println()
		fmt.Printf("  Output:\n  %s\n", ui.Muted.Render(output))
	}
	fmt.Println()
	return nil
}
