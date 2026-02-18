package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/rnwolfe/mine/internal/contrib"
	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/tmux"
	"github.com/rnwolfe/mine/internal/tui"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

var (
	contribRepo    string
	contribIssue   int
	contribUseTmux bool
	contribList    bool
)

var contribCmd = &cobra.Command{
	Use:   "contrib",
	Short: "Start an AI-assisted contribution workflow for any GitHub repo",
	Long: `Turbo-start an AI-assisted contribution workflow for any GitHub repo.
Handles fork/clone/issue-selection orchestration and hands off to your
local agentic environment.`,
	RunE: hook.Wrap("contrib", runContrib),
}

func init() {
	rootCmd.AddCommand(contribCmd)

	contribCmd.Flags().StringVar(&contribRepo, "repo", "", "Target GitHub repo (owner/name)")
	contribCmd.Flags().IntVarP(&contribIssue, "issue", "i", 0, "Issue number to work on")
	contribCmd.Flags().BoolVar(&contribUseTmux, "tmux", false, "Start a two-pane tmux workspace")
	contribCmd.Flags().BoolVar(&contribList, "list", false, "List candidate issues without starting flow")
}

func runContrib(_ *cobra.Command, _ []string) error {
	repo := strings.TrimSpace(contribRepo)

	if repo == "" {
		return fmt.Errorf("--repo is required — use: mine contrib --repo owner/name")
	}

	if err := contrib.ValidateRepo(repo); err != nil {
		return err
	}

	if err := contrib.CheckGH(); err != nil {
		return err
	}

	// --list: just show candidate issues and exit.
	if contribList {
		return runContribList(repo)
	}

	return runContribFlow(repo, contribIssue, contribUseTmux)
}

// runContribList prints candidate issues for the given repo without starting the flow.
func runContribList(repo string) error {
	fmt.Println()
	fmt.Printf("  %s %s\n", ui.Title.Render("Candidate issues for"), ui.Accent.Render(repo))
	fmt.Println()

	issues, agentReady, err := contrib.FetchCandidateIssues(repo)
	if err != nil {
		return err
	}

	if len(issues) == 0 {
		fmt.Println(ui.Muted.Render("  No open issues found."))
		fmt.Println()
		return nil
	}

	if agentReady {
		fmt.Println(ui.Muted.Render("  Showing agent-ready issues:"))
	} else {
		fmt.Println(ui.Muted.Render("  No agent-ready issues — showing all open issues:"))
	}
	fmt.Println()

	for _, issue := range issues {
		labels := issue.Description()
		labelStr := ""
		if labels != "" {
			labelStr = "  " + ui.Muted.Render("["+labels+"]")
		}
		fmt.Printf("  %s  %s%s\n",
			ui.Accent.Render(fmt.Sprintf("#%-4d", issue.Number)),
			issue.IssueTitle,
			labelStr,
		)
	}

	fmt.Println()
	ui.Tip(fmt.Sprintf("mine contrib --repo %s --issue <number>", repo))
	fmt.Println()
	return nil
}

// runContribFlow runs the full contribution orchestration flow.
func runContribFlow(repo string, issueNumber int, useTmux bool) error {
	reader := bufio.NewReader(os.Stdin)

	// Resolve the target issue.
	issue, err := resolveIssue(repo, issueNumber, reader)
	if err != nil {
		return err
	}
	if issue == nil {
		// User canceled in TTY picker.
		fmt.Println()
		ui.Warn("Cancelled.")
		return nil
	}

	// Show the opt-in prompt.
	if !confirmContrib(reader, repo, issue) {
		fmt.Println()
		ui.Warn("Cancelled.")
		return nil
	}

	// Check fork/clone state.
	state, err := contrib.CheckForkState(repo)
	if err != nil {
		return fmt.Errorf("checking fork state: %w", err)
	}

	// Handle existing clone gracefully.
	if state.CloneExists {
		fmt.Println()
		ui.Warn(fmt.Sprintf("Local directory already exists: %s", state.CloneDir))
		fmt.Printf("  %s\n", ui.Muted.Render("Remove it or change directories, then retry."))
		return fmt.Errorf("clone directory already exists: %s", state.CloneDir)
	}

	// Fork the repo (or reuse existing fork).
	fmt.Println()
	if state.ForkExists {
		fmt.Printf("  %s  Using existing fork %s\n", ui.Muted.Render(ui.IconOk), ui.Accent.Render(state.ForkSlug))
	} else {
		fmt.Printf("  %s  Forking %s...\n", ui.Muted.Render(ui.IconArrow), ui.Accent.Render(repo))
	}

	forkSlug, err := contrib.EnsureFork(repo)
	if err != nil {
		return fmt.Errorf("fork failed: %w", err)
	}

	// Clone the fork.
	fmt.Printf("  %s  Cloning %s...\n", ui.Muted.Render(ui.IconArrow), ui.Accent.Render(forkSlug))
	cloneDir, err := contrib.CloneRepo(repo, forkSlug)
	if err != nil {
		return fmt.Errorf("clone failed: %w", err)
	}

	// Create a branch for the issue.
	branch := contrib.BranchName(issue.Number, issue.IssueTitle)
	fmt.Printf("  %s  Creating branch %s...\n", ui.Muted.Render(ui.IconArrow), ui.Accent.Render(branch))
	if err := contrib.CreateBranch(cloneDir, branch); err != nil {
		return fmt.Errorf("branch creation failed: %w", err)
	}

	ui.Ok(fmt.Sprintf("Workspace ready at %s", ui.Accent.Render(cloneDir)))
	fmt.Println()

	printWorkspaceInfo(repo, issue, forkSlug, cloneDir, branch)

	// Optional tmux workspace.
	if useTmux {
		if err := launchTmuxWorkspace(cloneDir, repo, issue.Number); err != nil {
			ui.Warn(fmt.Sprintf("Could not create tmux workspace: %v", err))
			fmt.Printf("  %s\n", ui.Muted.Render("Continuing without tmux."))
		}
	}

	return nil
}

// resolveIssue determines the target issue via flag, agent-ready picker, or interactive picker.
func resolveIssue(repo string, issueNumber int, reader *bufio.Reader) (*contrib.Issue, error) {
	// Explicit issue number provided.
	if issueNumber > 0 {
		fmt.Printf("  %s  Fetching issue #%d...\n", ui.Muted.Render(ui.IconArrow), issueNumber)
		issue, err := contrib.FetchIssue(repo, issueNumber)
		if err != nil {
			return nil, err
		}
		return issue, nil
	}

	// Fetch candidate issues.
	fmt.Printf("  %s  Fetching candidate issues from %s...\n", ui.Muted.Render(ui.IconArrow), ui.Accent.Render(repo))
	issues, agentReady, err := contrib.FetchCandidateIssues(repo)
	if err != nil {
		return nil, err
	}

	if len(issues) == 0 {
		return nil, fmt.Errorf("no open issues found in %s — use --issue/-i to specify one", repo)
	}

	if agentReady {
		fmt.Printf("  %s  Found %d agent-ready issue(s)\n", ui.Muted.Render(ui.IconOk), len(issues))
	} else {
		fmt.Printf("  %s  No agent-ready issues — showing all open issues\n", ui.Muted.Render(ui.IconArrow))
	}

	// Non-TTY mode: require --issue flag.
	if !tui.IsTTY() {
		return nil, fmt.Errorf("not in a terminal — use --issue/-i to specify an issue number")
	}

	// Interactive picker.
	items := make([]tui.Item, len(issues))
	for i := range issues {
		items[i] = issues[i]
	}

	chosen, err := tui.Run(items,
		tui.WithTitle(ui.IconPick+"Select an issue to work on"),
		tui.WithHeight(14),
	)
	if err != nil {
		return nil, err
	}
	if chosen == nil {
		return nil, nil // user canceled
	}

	selected := chosen.(contrib.Issue)
	return &selected, nil
}

// confirmContrib shows the opt-in prompt explaining all actions about to happen.
func confirmContrib(reader *bufio.Reader, repo string, issue *contrib.Issue) bool {
	fmt.Println()
	fmt.Println(ui.Title.Render("  Contribution Flow"))
	fmt.Println()
	fmt.Printf("  %s\n", ui.Muted.Render("The following actions will be performed using your GitHub account:"))
	fmt.Println()
	fmt.Printf("    %s  Fork %s (if you don't already have one)\n", ui.Accent.Render(ui.IconArrow), ui.Accent.Render(repo))
	fmt.Printf("    %s  Clone your fork locally\n", ui.Accent.Render(ui.IconArrow))
	fmt.Printf("    %s  Create a branch for issue #%d\n", ui.Accent.Render(ui.IconArrow), issue.Number)
	fmt.Println()
	fmt.Printf("  %s %s\n", ui.Warning.Render(ui.IconWarn+"Issue:"), ui.Accent.Render(fmt.Sprintf("#%d — %s", issue.Number, issue.IssueTitle)))
	fmt.Println()
	fmt.Printf("  %s These actions use your own GitHub and API quota.\n",
		ui.Warning.Render(ui.IconWarn))
	fmt.Println()

	fmt.Printf("  %s ", ui.Accent.Render("Proceed? [y/N]"))
	line, _ := reader.ReadString('\n')
	ans := strings.TrimSpace(strings.ToLower(line))
	return ans == "y" || ans == "yes"
}

// printWorkspaceInfo shows the workspace summary after setup.
func printWorkspaceInfo(repo string, issue *contrib.Issue, forkSlug, cloneDir, branch string) {
	fmt.Println(ui.Title.Render("  Workspace"))
	fmt.Println()
	ui.Kv("  Repo", repo)
	ui.Kv("  Fork", forkSlug)
	ui.Kv("  Clone", cloneDir)
	ui.Kv("  Branch", branch)
	ui.Kv("  Issue", fmt.Sprintf("#%d — %s", issue.Number, issue.IssueTitle))
	fmt.Println()
	fmt.Printf("  Get started:\n")
	fmt.Printf("    %s\n", ui.Accent.Render("cd "+cloneDir))
	fmt.Println()
}

// launchTmuxWorkspace creates a two-pane tmux workspace: editor + shell.
func launchTmuxWorkspace(cloneDir, repo string, issueNumber int) error {
	if !tmux.Available() {
		return fmt.Errorf("tmux not found in PATH")
	}

	sessionName := fmt.Sprintf("contrib-%s-%d",
		strings.ReplaceAll(strings.Split(repo, "/")[1], ".", "-"),
		issueNumber,
	)

	// Create new session in the clone directory.
	newSess := exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-c", cloneDir)
	if out, err := newSess.CombinedOutput(); err != nil {
		return fmt.Errorf("creating session: %s", strings.TrimSpace(string(out)))
	}

	// Split into two panes: main (top) + shell (bottom).
	splitPane := exec.Command("tmux", "split-window", "-v", "-t", sessionName, "-c", cloneDir)
	if out, err := splitPane.CombinedOutput(); err != nil {
		// Non-fatal: session exists, just without the split.
		_ = out
	}

	// Select the top pane.
	exec.Command("tmux", "select-pane", "-t", sessionName+":0.0").Run() //nolint:errcheck

	ui.Ok(fmt.Sprintf("Tmux workspace %s created", ui.Accent.Render(sessionName)))
	fmt.Printf("  Attach: %s\n", ui.Accent.Render("mine tmux attach "+sessionName))
	fmt.Println()

	return tmux.AttachSession(sessionName)
}
