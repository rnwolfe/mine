package cmd

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/rnwolfe/mine/internal/git"
	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/tui"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

var gitCmd = &cobra.Command{
	Use:     "git",
	Aliases: []string{"g"},
	Short:   "Git workflow supercharger",
	Long:    `Fast convenience layer over the git operations you run 20+ times a day.`,
	RunE:    hook.Wrap("git", runGitBare),
}

func init() {
	rootCmd.AddCommand(gitCmd)

	gitCmd.AddCommand(gitSweepCmd)
	gitCmd.AddCommand(gitUndoCmd)
	gitCmd.AddCommand(gitWipCmd)
	gitCmd.AddCommand(gitUnwipCmd)
	gitCmd.AddCommand(gitPRCmd)
	gitCmd.AddCommand(gitLogCmd)
	gitCmd.AddCommand(gitChangelogCmd)
	gitCmd.AddCommand(gitAliasesCmd)

	gitChangelogCmd.Flags().StringP("from", "f", "", "Start ref (default: auto-detected base branch)")
	gitChangelogCmd.Flags().StringP("to", "t", "HEAD", "End ref")
}

// --- mine git (bare) — fuzzy branch picker ---

func runGitBare(_ *cobra.Command, _ []string) error {
	if !git.Available() {
		return fmt.Errorf("git not found in PATH — install git first")
	}

	branches, err := git.ListBranches()
	if err != nil {
		return err
	}

	if len(branches) == 0 {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No branches found."))
		fmt.Println()
		return nil
	}

	// Non-TTY fallback: plain list.
	if !tui.IsTTY() {
		return printBranchList(branches)
	}

	// Interactive fuzzy picker.
	items := make([]tui.Item, len(branches))
	for i := range branches {
		items[i] = branches[i]
	}

	chosen, err := tui.Run(items,
		tui.WithTitle(ui.IconPick+"Switch branch"),
		tui.WithHeight(14),
	)
	if err != nil {
		return err
	}
	if chosen == nil {
		return nil // user canceled
	}

	name := chosen.FilterValue()

	if err := git.SwitchBranch(name); err != nil {
		return err
	}

	ui.Ok(fmt.Sprintf("Switched to %s", ui.Accent.Render(name)))
	fmt.Println()
	return nil
}

// --- mine git sweep ---

var gitSweepCmd = &cobra.Command{
	Use:   "sweep",
	Short: "Delete merged local branches and prune stale remotes",
	RunE:  hook.Wrap("git.sweep", runGitSweep),
}

func runGitSweep(_ *cobra.Command, _ []string) error {
	if !git.Available() {
		return fmt.Errorf("git not found in PATH")
	}

	branches, err := git.MergedBranches()
	if err != nil {
		return err
	}

	if len(branches) == 0 {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No merged branches to delete."))
		fmt.Println()
		return nil
	}

	fmt.Println()
	fmt.Println(ui.Title.Render("  Merged branches to delete:"))
	fmt.Println()
	for _, b := range branches {
		fmt.Printf("    %s\n", ui.Accent.Render(b.Name))
	}
	fmt.Println()

	if !confirmPrompt("Delete these branches and prune remotes?") {
		fmt.Println(ui.Muted.Render("  Aborted."))
		fmt.Println()
		return nil
	}

	deleted := 0
	for _, b := range branches {
		if err := git.DeleteBranch(b.Name); err != nil {
			fmt.Printf("  %s %s: %v\n", ui.Warning.Render("warn"), b.Name, err)
		} else {
			fmt.Printf("  %s %s\n", ui.Success.Render("deleted"), ui.Accent.Render(b.Name))
			deleted++
		}
	}

	// Prune stale remote-tracking branches.
	if err := git.PruneRemote(); err != nil {
		fmt.Printf("  %s prune: %v\n", ui.Warning.Render("warn"), err)
	} else {
		fmt.Printf("  %s pruned stale remote refs\n", ui.Success.Render("ok"))
	}

	fmt.Println()
	ui.Ok(fmt.Sprintf("Swept %d branch(es)", deleted))
	fmt.Println()
	return nil
}

// --- mine git undo ---

var gitUndoCmd = &cobra.Command{
	Use:   "undo",
	Short: "Undo last commit (soft reset, keeps changes staged)",
	RunE:  hook.Wrap("git.undo", runGitUndo),
}

func runGitUndo(_ *cobra.Command, _ []string) error {
	if !git.Available() {
		return fmt.Errorf("git not found in PATH")
	}

	msg, err := git.LastCommitMessage()
	if err != nil {
		return fmt.Errorf("no commits to undo: %w", err)
	}

	fmt.Println()
	fmt.Printf("  Last commit: %s\n", ui.Accent.Render(msg))
	fmt.Println()

	if !confirmPrompt("Undo this commit? (changes will remain staged)") {
		fmt.Println(ui.Muted.Render("  Aborted."))
		fmt.Println()
		return nil
	}

	if err := git.UndoLastCommit(); err != nil {
		return err
	}

	ui.Ok("Commit undone — changes are staged and ready to re-commit.")
	fmt.Println()
	return nil
}

// --- mine git wip ---

var gitWipCmd = &cobra.Command{
	Use:   "wip",
	Short: `Quick WIP commit (git add -A && git commit -m "wip")`,
	RunE:  hook.Wrap("git.wip", runGitWip),
}

func runGitWip(_ *cobra.Command, _ []string) error {
	if !git.Available() {
		return fmt.Errorf("git not found in PATH")
	}

	if err := git.WipCommit(); err != nil {
		return err
	}

	ui.Ok(`WIP committed. Use "mine git unwip" to undo.`)
	fmt.Println()
	return nil
}

// --- mine git unwip ---

var gitUnwipCmd = &cobra.Command{
	Use:   "unwip",
	Short: "Undo the last commit if it is a WIP commit",
	RunE:  hook.Wrap("git.unwip", runGitUnwip),
}

func runGitUnwip(_ *cobra.Command, _ []string) error {
	if !git.Available() {
		return fmt.Errorf("git not found in PATH")
	}

	ok, err := git.IsWipCommit()
	if err != nil {
		return fmt.Errorf("no commits to unwip: %w", err)
	}
	if !ok {
		msg, _ := git.LastCommitMessage()
		return fmt.Errorf("last commit is not a WIP commit (got: %q)", msg)
	}

	if err := git.UndoLastCommit(); err != nil {
		return err
	}

	ui.Ok("WIP commit undone — changes are staged.")
	fmt.Println()
	return nil
}

// --- mine git pr ---

var gitPRCmd = &cobra.Command{
	Use:   "pr",
	Short: "Create a PR from the current branch (uses gh CLI if available)",
	RunE:  hook.Wrap("git.pr", runGitPR),
}

func runGitPR(_ *cobra.Command, _ []string) error {
	if !git.Available() {
		return fmt.Errorf("git not found in PATH")
	}

	info, err := git.BuildPRInfo()
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("  %s %s\n", ui.Muted.Render("Branch:"), ui.Accent.Render(info.Branch))
	fmt.Printf("  %s %s\n", ui.Muted.Render("Base:  "), ui.Accent.Render(info.Base))
	fmt.Printf("  %s %s\n", ui.Muted.Render("Title: "), info.Title)
	fmt.Println()

	if !git.HasGhCLI() {
		fmt.Println(ui.Warning.Render("  gh CLI not found — cannot create PR automatically."))
		fmt.Println()
		fmt.Println(ui.Muted.Render("  Install gh: https://cli.github.com"))
		fmt.Println()
		fmt.Println(ui.Title.Render("  Generated PR body:"))
		fmt.Println()
		fmt.Println(info.Body)
		return nil
	}

	if !confirmPrompt(fmt.Sprintf("Create PR: %q", info.Title)) {
		fmt.Println(ui.Muted.Render("  Aborted."))
		fmt.Println()
		return nil
	}

	prURL, err := ghPRCreate(info)
	if err != nil {
		return fmt.Errorf("gh pr create: %w", err)
	}

	ui.Ok("PR created: " + prURL)
	fmt.Println()
	return nil
}

// ghPRCreate invokes gh to create a pull request and returns the PR URL.
var ghPRCreate = func(info *git.PRInfo) (string, error) {
	cmd := exec.Command("gh", "pr", "create",
		"--title", info.Title,
		"--body", info.Body,
		"--base", info.Base,
	)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", errors.New(msg)
	}
	return strings.TrimSpace(out.String()), nil
}

// --- mine git log ---

var gitLogCmd = &cobra.Command{
	Use:   "log",
	Short: "Pretty interactive commit log (compact, colored, graph)",
	RunE:  hook.Wrap("git.log", runGitLog),
}

func runGitLog(_ *cobra.Command, _ []string) error {
	if !git.Available() {
		return fmt.Errorf("git not found in PATH")
	}

	out, err := git.CommitLog(30)
	if err != nil {
		return err
	}

	if out == "" {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No commits yet."))
		fmt.Println()
		return nil
	}

	fmt.Println()
	fmt.Println(out)
	fmt.Println()
	return nil
}

// --- mine git changelog ---

var gitChangelogCmd = &cobra.Command{
	Use:   "changelog",
	Short: "Generate Markdown changelog from conventional commits",
	RunE:  hook.Wrap("git.changelog", runGitChangelog),
}

func runGitChangelog(cmd *cobra.Command, _ []string) error {
	if !git.Available() {
		return fmt.Errorf("git not found in PATH")
	}

	from, _ := cmd.Flags().GetString("from")
	to, _ := cmd.Flags().GetString("to")

	if from == "" {
		from = git.DefaultBase()
	}

	changelog, err := git.Changelog(from, to)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Print(changelog)
	fmt.Println()
	return nil
}

// --- mine git aliases ---

var gitAliasesCmd = &cobra.Command{
	Use:   "aliases",
	Short: "Install opinionated git aliases to global config",
	RunE:  hook.Wrap("git.aliases", runGitAliases),
}

func runGitAliases(_ *cobra.Command, _ []string) error {
	if !git.Available() {
		return fmt.Errorf("git not found in PATH")
	}

	aliases := git.GitAliases()

	fmt.Println()
	fmt.Println(ui.Title.Render("  Git Aliases to Install:"))
	fmt.Println()
	for _, a := range aliases {
		fmt.Printf("    %s  %s  %s\n",
			ui.Accent.Render(fmt.Sprintf("%-12s", "git "+a.Name)),
			ui.Muted.Render(fmt.Sprintf("→ %-42s", a.Value)),
			ui.Muted.Render("# "+a.Desc),
		)
	}
	fmt.Println()

	if !confirmPrompt("Install these aliases to ~/.gitconfig?") {
		fmt.Println(ui.Muted.Render("  Aborted."))
		fmt.Println()
		return nil
	}

	installed := 0
	for _, a := range aliases {
		if err := git.InstallAlias(a); err != nil {
			fmt.Printf("  %s %s: %v\n", ui.Warning.Render("warn"), a.Name, err)
		} else {
			fmt.Printf("  %s %s\n", ui.Success.Render("installed"), ui.Accent.Render("git "+a.Name))
			installed++
		}
	}

	fmt.Println()
	ui.Ok(fmt.Sprintf("Installed %d git alias(es)", installed))
	fmt.Println()
	return nil
}

// --- helpers ---

func printBranchList(branches []git.Branch) error {
	fmt.Println()
	for _, b := range branches {
		marker := "  "
		if b.Current {
			marker = ui.Success.Render("* ")
		}
		fmt.Printf("  %s%s\n", marker, ui.Accent.Render(b.Name))
	}
	fmt.Println()
	return nil
}

// confirmPrompt prompts the user for a yes/no answer and returns true for "y"/"yes".
func confirmPrompt(prompt string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("  %s [y/N] ", ui.Warning.Render(prompt))
	line, _ := reader.ReadString('\n')
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes"
}
