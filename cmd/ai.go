package cmd

import (
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

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "AI-powered developer tools",
	Long:  `Interact with AI providers for code review, commit messages, and more.`,
	RunE:  hook.Wrap("ai", runAIHelp),
}

var (
	aiProvider string
	aiModel    string
	aiStream   bool = true
)

func init() {
	// Subcommands
	aiCmd.AddCommand(aiConfigCmd)
	aiCmd.AddCommand(aiAskCmd)
	aiCmd.AddCommand(aiReviewCmd)
	aiCmd.AddCommand(aiCommitCmd)

	// Global flags
	aiCmd.PersistentFlags().BoolVar(&aiStream, "stream", true, "Stream responses (disable for non-TTY)")
}

var aiConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure AI provider",
	Long:  `Set up API keys and choose your AI provider.`,
	RunE:  hook.Wrap("ai.config", runAIConfig),
}

var aiAskCmd = &cobra.Command{
	Use:   "ask <question>",
	Short: "Ask the AI a question",
	Long:  `Send a quick question to your configured AI provider.`,
	Args:  cobra.MinimumNArgs(1),
	RunE:  hook.Wrap("ai.ask", runAIAsk),
}

var aiReviewCmd = &cobra.Command{
	Use:   "review",
	Short: "AI code review of staged changes",
	Long:  `Get AI feedback on your staged git changes.`,
	RunE:  hook.Wrap("ai.review", runAIReview),
}

var aiCommitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Generate commit message from diff",
	Long:  `Use AI to generate a conventional commit message from staged changes.`,
	RunE:  hook.Wrap("ai.commit", runAICommit),
}

func runAIHelp(cmd *cobra.Command, args []string) error {
	return cmd.Help()
}

func runAIConfig(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Prompt for provider
	fmt.Println()
	fmt.Printf("  %s AI Provider Configuration\n", ui.IconGem)
	fmt.Println()
	fmt.Println("  Available providers:")
	fmt.Println("    1. Claude (Anthropic)")
	fmt.Println("    2. OpenAI (coming soon)")
	fmt.Println("    3. Ollama (coming soon)")
	fmt.Println()
	fmt.Print("  Select provider [1]: ")

	var choice string
	fmt.Scanln(&choice)
	if choice == "" {
		choice = "1"
	}

	var provider string
	switch choice {
	case "1", "claude", "anthropic":
		provider = "claude"
	default:
		return fmt.Errorf("invalid provider choice")
	}

	// Prompt for API key
	fmt.Println()
	fmt.Printf("  Enter your %s API key: ", provider)
	var apiKey string
	fmt.Scanln(&apiKey)
	if apiKey == "" {
		return fmt.Errorf("API key required")
	}

	// Save to keystore
	ks := ai.NewKeyStore()
	if err := ks.Set(provider, apiKey); err != nil {
		return fmt.Errorf("save API key: %w", err)
	}

	// Update config
	cfg.AI.Provider = provider
	if provider == "claude" && cfg.AI.Model == "" {
		cfg.AI.Model = "claude-sonnet-4-5-20250929"
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	fmt.Println()
	fmt.Printf("  %s Configured %s as default provider\n", ui.Success.Render("✓"), provider)
	fmt.Printf("  %s Model: %s\n", ui.IconArrow, ui.Muted.Render(cfg.AI.Model))
	fmt.Println()
	fmt.Printf("  Try it: %s\n", ui.Accent.Render(`mine ai ask "what is a monad?"`))
	fmt.Println()

	return nil
}

func runAIAsk(_ *cobra.Command, args []string) error {
	question := strings.Join(args, " ")

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	provider, err := ai.GetProvider(cfg, os.Stdout)
	if err != nil {
		return err
	}

	req := ai.CompletionRequest{
		Messages: []ai.Message{
			{Role: "user", Content: question},
		},
		MaxTokens: 4096,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Println()
	fmt.Printf("  %s Asking %s...\n", ui.IconGem, provider.Name())
	fmt.Println()

	if aiStream {
		first := true
		_, err = provider.Stream(ctx, req, func(chunk string) error {
			if first {
				fmt.Print("  ")
				first = false
			}
			fmt.Print(chunk)
			return nil
		})
		fmt.Println()
	} else {
		resp, err := provider.Complete(ctx, req)
		if err != nil {
			return err
		}
		fmt.Printf("  %s\n", resp.Content)
	}

	fmt.Println()
	return err
}

func runAIReview(_ *cobra.Command, _ []string) error {
	// Get staged diff
	diff, err := getGitDiff(true)
	if err != nil {
		return err
	}

	if diff == "" {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No staged changes to review"))
		fmt.Println()
		fmt.Printf("  Stage changes: %s\n", ui.Accent.Render("git add <files>"))
		fmt.Println()
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	provider, err := ai.GetProvider(cfg, os.Stdout)
	if err != nil {
		return err
	}

	prompt := fmt.Sprintf(`Review this code diff and provide feedback on:
- Potential bugs or issues
- Code quality and best practices
- Security concerns
- Suggestions for improvement

Diff:
%s`, diff)

	req := ai.CompletionRequest{
		Messages: []ai.Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens: 4096,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Println()
	fmt.Printf("  %s Reviewing staged changes with %s...\n", ui.IconGem, provider.Name())
	fmt.Println()

	if aiStream {
		first := true
		_, err = provider.Stream(ctx, req, func(chunk string) error {
			if first {
				fmt.Print("  ")
				first = false
			}
			fmt.Print(chunk)
			return nil
		})
		fmt.Println()
	} else {
		resp, err := provider.Complete(ctx, req)
		if err != nil {
			return err
		}
		fmt.Printf("  %s\n", resp.Content)
	}

	fmt.Println()
	return err
}

func runAICommit(_ *cobra.Command, _ []string) error {
	// Get staged diff
	diff, err := getGitDiff(true)
	if err != nil {
		return err
	}

	if diff == "" {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No staged changes"))
		fmt.Println()
		fmt.Printf("  Stage changes: %s\n", ui.Accent.Render("git add <files>"))
		fmt.Println()
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	provider, err := ai.GetProvider(cfg, os.Stdout)
	if err != nil {
		return err
	}

	prompt := fmt.Sprintf(`Generate a conventional commit message for this diff.
Follow the format: <type>(<scope>): <description>

Types: feat, fix, docs, style, refactor, test, chore
Keep the description under 72 characters.
Add a body if needed to explain the "why".

Return ONLY the commit message, no extra commentary.

Diff:
%s`, diff)

	req := ai.CompletionRequest{
		Messages: []ai.Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens: 512,
		Temperature: 0.7,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println()
	fmt.Printf("  %s Generating commit message with %s...\n", ui.IconGem, provider.Name())
	fmt.Println()

	resp, err := provider.Complete(ctx, req)
	if err != nil {
		return err
	}

	message := strings.TrimSpace(resp.Content)
	fmt.Println(ui.Muted.Render("  Suggested commit message:"))
	fmt.Println()
	for _, line := range strings.Split(message, "\n") {
		fmt.Printf("    %s\n", line)
	}
	fmt.Println()

	// Ask if user wants to commit
	fmt.Print("  Use this message? [y/N]: ")
	var answer string
	fmt.Scanln(&answer)
	if strings.ToLower(answer) != "y" {
		fmt.Println(ui.Muted.Render("  Commit cancelled"))
		return nil
	}

	// Create the commit
	cmd := exec.Command("git", "commit", "-m", message)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git commit failed: %w", err)
	}

	fmt.Println()
	fmt.Printf("  %s Committed!\n", ui.Success.Render("✓"))
	fmt.Println()

	return nil
}

// getGitDiff returns the git diff output.
func getGitDiff(staged bool) (string, error) {
	args := []string{"diff"}
	if staged {
		args = append(args, "--staged")
	}

	cmd := exec.Command("git", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git diff failed: %w", err)
	}

	return string(output), nil
}
