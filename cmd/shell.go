package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rnwolfe/mine/internal/config"
	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/shell"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Shell integration and enhancements",
	Long:  `Set up shell completions, aliases, functions, and prompt integration.`,
	RunE:  hook.Wrap("shell", runShellHelp),
}

func init() {
	rootCmd.AddCommand(shellCmd)
	shellCmd.AddCommand(shellInitCmd)
	shellCmd.AddCommand(shellCompletionsCmd)
	shellCmd.AddCommand(shellAliasesCmd)
	shellCmd.AddCommand(shellFunctionsCmd)
	shellCmd.AddCommand(shellPromptCmd)
}

// --- mine shell init ---

var shellInitCmd = &cobra.Command{
	Use:   "init [bash|zsh|fish]",
	Short: "Generate shell init script (eval-able)",
	Long: `Generate a complete shell initialization script.

Usage: eval "$(mine shell init zsh)"

This sets up aliases, utility functions, and prompt integration
in a single command. Add it to your shell config for persistent use.`,
	Args: cobra.MaximumNArgs(1),
	RunE: hook.Wrap("shell.init", runShellInit),
}

func runShellInit(_ *cobra.Command, args []string) error {
	sh := detectShell()
	if len(args) > 0 {
		sh = args[0]
	}

	script, err := shell.InitScript(sh)
	if err != nil {
		return err
	}

	fmt.Print(script)
	return nil
}

// --- mine shell completions ---

var shellCompletionsCmd = &cobra.Command{
	Use:   "completions [bash|zsh|fish]",
	Short: "Generate shell completions",
	Args:  cobra.MaximumNArgs(1),
	RunE:  hook.Wrap("shell.completions", runShellCompletions),
}

func runShellCompletions(_ *cobra.Command, args []string) error {
	sh := detectShell()
	if len(args) > 0 {
		sh = args[0]
	}

	if !shell.ValidShell(sh) {
		return shell.ShellError(sh)
	}

	completionDir := filepath.Join(config.GetPaths().ConfigDir, "completions")
	os.MkdirAll(completionDir, 0o755)

	switch sh {
	case "bash":
		file := filepath.Join(completionDir, "mine.bash")
		f, err := os.Create(file)
		if err != nil {
			return err
		}
		defer f.Close()
		rootCmd.GenBashCompletionV2(f, true)
		ui.Ok("Bash completions written to " + file)
		fmt.Println()
		fmt.Println(ui.Muted.Render("  Add to your ~/.bashrc:"))
		fmt.Printf("    %s\n", ui.Accent.Render(fmt.Sprintf("source %s", file)))

	case "zsh":
		file := filepath.Join(completionDir, "_mine")
		f, err := os.Create(file)
		if err != nil {
			return err
		}
		defer f.Close()
		rootCmd.GenZshCompletion(f)
		ui.Ok("Zsh completions written to " + file)
		fmt.Println()
		fmt.Println(ui.Muted.Render("  Add to your ~/.zshrc:"))
		fmt.Printf("    %s\n", ui.Accent.Render(fmt.Sprintf("fpath=(%s $fpath)", completionDir)))
		fmt.Printf("    %s\n", ui.Accent.Render("autoload -Uz compinit && compinit"))

	case "fish":
		file := filepath.Join(completionDir, "mine.fish")
		f, err := os.Create(file)
		if err != nil {
			return err
		}
		defer f.Close()
		rootCmd.GenFishCompletion(f, true)
		ui.Ok("Fish completions written to " + file)
		fmt.Println()
		fmt.Println(ui.Muted.Render("  Add to your fish config:"))
		fmt.Printf("    %s\n", ui.Accent.Render(fmt.Sprintf("source %s", file)))

	}

	fmt.Println()
	return nil
}

// --- mine shell aliases ---

var shellAliasesCmd = &cobra.Command{
	Use:   "aliases",
	Short: "Show recommended shell aliases",
	RunE:  hook.Wrap("shell.aliases", runShellAliases),
}

func runShellAliases(_ *cobra.Command, _ []string) error {
	fmt.Println()
	fmt.Println(ui.Title.Render("  Recommended Aliases"))
	fmt.Println()
	fmt.Println(ui.Muted.Render("  Add these to your shell config:"))
	fmt.Println()

	aliases := []struct{ alias, expansion, desc string }{
		{"m", "mine", "shortcut to mine"},
		{"mt", "mine todo", "quick todo access"},
		{"mta", "mine todo add", "add a todo fast"},
		{"mtd", "mine todo done", "complete a todo"},
		{"md", "mine dig", "start a focus session"},
		{"mc", "mine craft", "scaffold something"},
		{"ms", "mine stash", "dotfile management"},
		{"mx", "mine tmux", "tmux sessions"},
	}

	for _, a := range aliases {
		fmt.Printf("    %s  %s\n",
			ui.Accent.Render(fmt.Sprintf("alias %-4s='%s'", a.alias, a.expansion)),
			ui.Muted.Render("# "+a.desc),
		)
	}

	fmt.Println()
	return nil
}

// --- mine shell functions ---

var shellFunctionsCmd = &cobra.Command{
	Use:   "functions",
	Short: "List available shell utility functions",
	RunE:  hook.Wrap("shell.functions", runShellFunctions),
}

func runShellFunctions(_ *cobra.Command, _ []string) error {
	fmt.Println()
	fmt.Println(ui.Title.Render("  Shell Functions"))
	fmt.Println()
	fmt.Println(ui.Muted.Render("  These are included when you run: eval \"$(mine shell init)\""))
	fmt.Println()

	for _, fn := range shell.Functions() {
		fmt.Printf("    %s  %s\n",
			ui.Accent.Render(fmt.Sprintf("%-10s", fn.Name)),
			ui.Muted.Render(fn.Desc),
		)
	}

	fmt.Println()
	ui.Tip("Run `mine shell init " + detectShell() + "` to see the generated script.")
	fmt.Println()
	return nil
}

// --- mine shell prompt ---

var shellPromptCmd = &cobra.Command{
	Use:   "prompt",
	Short: "Show prompt integration setup",
	RunE:  hook.Wrap("shell.prompt", runShellPrompt),
}

func runShellPrompt(_ *cobra.Command, _ []string) error {
	fmt.Println()
	fmt.Println(ui.Title.Render("  Prompt Integration"))
	fmt.Println()

	fmt.Println(ui.Subtitle.Render("  Automatic (via mine shell init)"))
	fmt.Println()
	fmt.Println(ui.Muted.Render("  Prompt integration is included in the init script:"))
	fmt.Println()
	fmt.Printf("    %s\n", ui.Accent.Render("eval \"$(mine shell init "+detectShell()+")\""))
	fmt.Println()

	fmt.Println(ui.Subtitle.Render("  Starship"))
	fmt.Println()
	fmt.Println(ui.Muted.Render("  Add to ~/.config/starship.toml:"))
	fmt.Println()
	for _, line := range strings.Split(shell.StarshipConfig(), "\n") {
		if line != "" {
			fmt.Printf("    %s\n", ui.Muted.Render(line))
		}
	}
	fmt.Println()

	fmt.Println(ui.Subtitle.Render("  Data Commands"))
	fmt.Println()
	fmt.Printf("    %s  %s\n", ui.Accent.Render("mine status --json"), ui.Muted.Render("Full JSON status"))
	fmt.Printf("    %s  %s\n", ui.Accent.Render("mine status --prompt"), ui.Muted.Render("Compact prompt segment"))
	fmt.Println()
	return nil
}

// --- help ---

func runShellHelp(_ *cobra.Command, _ []string) error {
	fmt.Println()
	fmt.Println(ui.Title.Render("  Shell Integration"))
	fmt.Println()
	fmt.Println(ui.Muted.Render("  Quick start:"))
	fmt.Printf("    %s\n", ui.Accent.Render(fmt.Sprintf("eval \"$(mine shell init %s)\"", detectShell())))
	fmt.Println()
	fmt.Println(ui.Muted.Render("  Commands:"))
	fmt.Printf("    %s  %s\n", ui.Accent.Render("mine shell init"), ui.Muted.Render("Generate eval-able init script"))
	fmt.Printf("    %s  %s\n", ui.Accent.Render("mine shell completions"), ui.Muted.Render("Generate tab completions"))
	fmt.Printf("    %s  %s\n", ui.Accent.Render("mine shell aliases"), ui.Muted.Render("Show handy aliases"))
	fmt.Printf("    %s  %s\n", ui.Accent.Render("mine shell functions"), ui.Muted.Render("List utility functions"))
	fmt.Printf("    %s  %s\n", ui.Accent.Render("mine shell prompt"), ui.Muted.Render("Prompt integration setup"))
	fmt.Println()
	return nil
}

func detectShell() string {
	sh := os.Getenv("SHELL")
	if strings.Contains(sh, "zsh") {
		return "zsh"
	}
	if strings.Contains(sh, "fish") {
		return "fish"
	}
	return "bash"
}
