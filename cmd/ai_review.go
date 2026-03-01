package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/rnwolfe/mine/internal/ai"
	"github.com/rnwolfe/mine/internal/config"
	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

// ai review command
var (
	aiReviewCmd = &cobra.Command{
		Use:   "review",
		Short: "Get AI eyes on your staged changes",
		Long:  `Get an AI-powered code review of your staged git changes.`,
		RunE:  hook.Wrap("ai.review", runAIReview),
	}
	aiReviewRaw    bool
	aiReviewSystem string
	aiCommitSystem string
)

// ai commit command
var aiCommitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Let AI draft your commit message",
	Long:  `Analyze your staged changes and generate a clear commit message.`,
	RunE:  hook.Wrap("ai.commit", runAICommit),
}

func init() {
	aiReviewCmd.Flags().StringVar(&aiReviewSystem, "system", "", "Override system instructions for this invocation (empty string disables system instructions)")
	aiReviewCmd.Flags().BoolVar(&aiReviewRaw, "raw", false, "Output raw markdown without terminal rendering")
	aiCommitCmd.Flags().StringVar(&aiCommitSystem, "system", "", "Override system instructions for this invocation (empty string disables system instructions)")
}

func runAIReview(cmd *cobra.Command, _ []string) error {
	// Get git diff of staged changes
	gitCmd := exec.Command("git", "diff", "--cached")
	output, err := gitCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get git diff: %w\nGit output:\n%s", err, strings.TrimSpace(string(output)))
	}

	diff := string(output)
	if strings.TrimSpace(diff) == "" {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No staged changes to review."))
		fmt.Println()
		fmt.Printf("  Stage changes: %s\n", ui.Accent.Render("git add <files>"))
		fmt.Println()
		return nil
	}

	// Truncate large diffs to avoid exceeding provider context limits
	const maxDiffBytes = 50000 // ~50KB, conservative limit
	truncated := false
	if len(diff) > maxDiffBytes {
		diff = diff[:maxDiffBytes]
		truncated = true
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	provider, err := getConfiguredProviderFromConfig(cfg)
	if err != nil {
		return err
	}

	prompt := fmt.Sprintf(`Review the following code changes and provide feedback on:
- Potential bugs or issues
- Code quality and best practices
- Security concerns
- Performance improvements

Here's the diff%s:

%s`, func() string {
		if truncated {
			return " (truncated to 50KB - review may be incomplete)"
		}
		return ""
	}(), diff)

	const reviewBuiltinSystem = "You are an expert code reviewer. Provide constructive, specific feedback."
	req := ai.NewRequest(prompt)
	req.System = resolveSystemInstructions(&cfg.AI, "review", aiReviewSystem, cmd.Flags().Changed("system"), reviewBuiltinSystem)
	if aiModel != "" {
		req.Model = aiModel
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	fmt.Println()
	fmt.Println(ui.Title.Render("  Code Review"))
	fmt.Println()
	fmt.Println(ui.Muted.Render(fmt.Sprintf("  Analyzing staged changes with %s...", provider.Name())))
	fmt.Println()

	// Stream the review through a markdown-aware writer.
	mdw := ui.NewMarkdownWriter(os.Stdout, aiReviewRaw)
	if err := provider.Stream(ctx, req, mdw); err != nil {
		return err
	}
	if err := mdw.Flush(); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println()
	return nil
}

func runAICommit(cmd *cobra.Command, _ []string) error {
	// Get git diff of staged changes
	gitCmd := exec.Command("git", "diff", "--cached")
	output, err := gitCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to get git diff: %w; output:\n%s", err, string(output))
	}

	diff := string(output)
	if strings.TrimSpace(diff) == "" {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No staged changes to commit."))
		fmt.Println()
		fmt.Printf("  Stage changes: %s\n", ui.Accent.Render("git add <files>"))
		fmt.Println()
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	provider, err := getConfiguredProviderFromConfig(cfg)
	if err != nil {
		return err
	}

	// Truncate diff to avoid exceeding model context/MaxTokens limits (50KB cap).
	const maxCommitDiffBytes = 50 * 1024
	diffBytes := []byte(diff)
	truncatedDiff := diff
	truncationNote := ""
	if len(diffBytes) > maxCommitDiffBytes {
		truncatedDiff = string(diffBytes[:maxCommitDiffBytes])
		truncationNote = "\n\n[Diff truncated to 50KB to fit model limits. Review the full diff locally if needed.]\n"
	}

	prompt := fmt.Sprintf(`Generate a clear, concise git commit message for the following changes.
Follow conventional commit format (feat:, fix:, docs:, refactor:, test:, chore:).
Keep the first line under 70 characters.
If needed, add a blank line and then a more detailed explanation.

Here's the diff (it may be truncated to 50KB):

%s%s`, truncatedDiff, truncationNote)

	const commitBuiltinSystem = "You are a git commit message expert. Write clear, professional commit messages."
	req := ai.NewRequest(prompt)
	req.System = resolveSystemInstructions(&cfg.AI, "commit", aiCommitSystem, cmd.Flags().Changed("system"), commitBuiltinSystem)
	req.MaxTokens = 500
	if aiModel != "" {
		req.Model = aiModel
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println()
	fmt.Println(ui.Muted.Render(fmt.Sprintf("  Generating commit message with %s...", provider.Name())))
	fmt.Println()

	resp, err := provider.Complete(ctx, req)
	if err != nil {
		return err
	}

	message := strings.TrimSpace(resp.Content)

	// Display the message
	fmt.Println(ui.Success.Render("  Suggested commit message:"))
	fmt.Println()
	for _, line := range strings.Split(message, "\n") {
		fmt.Printf("    %s\n", line)
	}
	fmt.Println()

	// Ask if they want to use it
	fmt.Print(ui.Muted.Render("  Use this message? [y/N] "))
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))

	if answer != "y" && answer != "yes" {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  Commit cancelled."))
		fmt.Println()
		return nil
	}

	// Create the commit
	gitCmd = exec.Command("git", "commit", "-m", message)
	gitCmd.Stdout = os.Stdout
	gitCmd.Stderr = os.Stderr
	if err := gitCmd.Run(); err != nil {
		return fmt.Errorf("failed to create commit: %w", err)
	}

	fmt.Println()
	ui.Ok("Commit created successfully!")
	fmt.Println()

	return nil
}
