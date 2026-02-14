package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rnwolfe/mine/internal/config"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Shell integration and enhancements",
	Long:  `Set up shell completions, aliases, and environment helpers.`,
	RunE:  runShellHelp,
}

func init() {
	rootCmd.AddCommand(shellCmd)
	shellCmd.AddCommand(shellCompletionsCmd)
	shellCmd.AddCommand(shellAliasesCmd)
}

var shellCompletionsCmd = &cobra.Command{
	Use:   "completions [bash|zsh|fish]",
	Short: "Generate shell completions",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runShellCompletions,
}

var shellAliasesCmd = &cobra.Command{
	Use:   "aliases",
	Short: "Show recommended shell aliases",
	RunE:  runShellAliases,
}

func runShellHelp(_ *cobra.Command, _ []string) error {
	fmt.Println()
	fmt.Println(ui.Title.Render("  Shell Integration"))
	fmt.Println()
	fmt.Printf("    %s  %s\n", ui.Accent.Render("mine shell completions"), ui.Muted.Render("Generate tab completions"))
	fmt.Printf("    %s  %s\n", ui.Accent.Render("mine shell aliases"), ui.Muted.Render("Show handy aliases"))
	fmt.Println()
	return nil
}

func runShellCompletions(_ *cobra.Command, args []string) error {
	shell := detectShell()
	if len(args) > 0 {
		shell = args[0]
	}

	completionDir := filepath.Join(config.GetPaths().ConfigDir, "completions")
	os.MkdirAll(completionDir, 0o755)

	switch shell {
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

	default:
		return fmt.Errorf("unknown shell %q — try: bash, zsh, fish", shell)
	}

	fmt.Println()
	return nil
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

func detectShell() string {
	shell := os.Getenv("SHELL")
	if strings.Contains(shell, "zsh") {
		return "zsh"
	}
	if strings.Contains(shell, "fish") {
		return "fish"
	}
	return "bash"
}
