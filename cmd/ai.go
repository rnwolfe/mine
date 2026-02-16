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
	"golang.org/x/term"
)

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "AI-powered helpers",
	Long:  `Manage AI provider integrations and use AI helpers directly from the CLI.`,
}

var aiConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure AI provider settings",
	Long:  `Configure AI providers, API keys, and model preferences.`,
	RunE:  hook.Wrap("ai config", runAIConfig),
}

var aiAskCmd = &cobra.Command{
	Use:   "ask [question]",
	Short: "Ask a quick question to your AI",
	Long:  `Send a question to your configured AI provider and get a response.`,
	Args:  cobra.MinimumNArgs(1),
	RunE:  hook.Wrap("ai ask", runAIAsk),
}

var aiReviewCmd = &cobra.Command{
	Use:   "review",
	Short: "AI-powered code review of staged changes",
	Long:  `Review staged git changes using AI to catch bugs, suggest improvements, and ensure code quality.`,
	RunE:  hook.Wrap("ai review", runAIReview),
}

var aiCommitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Generate a commit message from diff",
	Long:  `Generate a conventional commit message from your staged changes using AI.`,
	RunE:  hook.Wrap("ai commit", runAICommit),
}

var (
	aiStreamFlag bool
	aiModelFlag  string
)

func init() {
	rootCmd.AddCommand(aiCmd)
	aiCmd.AddCommand(aiConfigCmd)
	aiCmd.AddCommand(aiAskCmd)
	aiCmd.AddCommand(aiReviewCmd)
	aiCmd.AddCommand(aiCommitCmd)

	aiAskCmd.Flags().BoolVarP(&aiStreamFlag, "stream", "s", true, "Stream the response")
	aiAskCmd.Flags().StringVarP(&aiModelFlag, "model", "m", "", "Override the configured model")
	aiReviewCmd.Flags().StringVarP(&aiModelFlag, "model", "m", "", "Override the configured model")
	aiCommitCmd.Flags().StringVarP(&aiModelFlag, "model", "m", "", "Override the configured model")
}

func runAIConfig(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	ui.Inf("AI Configuration")
	fmt.Println()

	// Show current settings
	ui.Kv("  Provider", cfg.AI.Provider)
	ui.Kv("  Model", cfg.AI.Model)
	fmt.Println()

	// Check API key status
	ks := ai.NewKeyStore()
	if err := ks.Load(); err != nil {
		return fmt.Errorf("loading keys: %w", err)
	}

	key, err := ks.Get(cfg.AI.Provider)
	if err != nil {
		ui.Warn(fmt.Sprintf("No API key configured for %s", cfg.AI.Provider))
		fmt.Println()
		fmt.Println("  Set your API key:")
		envVar := ai.EnvClaudeKey
		if cfg.AI.Provider == "openai" {
			envVar = ai.EnvOpenAIKey
		}
		fmt.Printf("    export %s=your-key-here\n", envVar)
		fmt.Println()
		fmt.Println("  Or enter it now (it will be stored securely):")
		fmt.Print("  API Key: ")

		// Read password without echoing to terminal
		keyBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println() // Print newline after password input
		if err != nil {
			return fmt.Errorf("reading API key: %w", err)
		}

		apiKey := strings.TrimSpace(string(keyBytes))
		if apiKey != "" {
			ks.Set(cfg.AI.Provider, apiKey)
			if err := ks.Save(); err != nil {
				return fmt.Errorf("saving key: %w", err)
			}
			ui.Ok("API key saved successfully")
		}
	} else {
		maskedKey := maskAPIKey(key)
		ui.Ok(fmt.Sprintf("API key configured (%s)", maskedKey))
	}

	fmt.Println()
	ui.Tip("Use `mine ai ask` to test your configuration")
	return nil
}

func runAIAsk(_ *cobra.Command, args []string) error {
	question := strings.Join(args, " ")

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	provider, err := getProvider(cfg)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	req := &ai.Request{
		Prompt: question,
	}

	if aiStreamFlag {
		if err := provider.Stream(ctx, req, os.Stdout); err != nil {
			return fmt.Errorf("streaming response: %w", err)
		}
		fmt.Println() // Add newline after stream
	} else {
		resp, err := provider.Complete(ctx, req)
		if err != nil {
			return fmt.Errorf("getting response: %w", err)
		}
		fmt.Println(resp.Content)
	}

	return nil
}

func runAIReview(_ *cobra.Command, _ []string) error {
	// Ensure we are inside a git repository
	if err := exec.Command("git", "rev-parse", "--git-dir").Run(); err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	// Get staged diff
	cmd := exec.Command("git", "diff", "--cached")
	diffOutput, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("getting staged changes: %w", err)
	}

	diff := string(diffOutput)
	if diff == "" {
		ui.Warn("No staged changes to review")
		ui.Tip("Run `git add` to stage files first")
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	provider, err := getProvider(cfg)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	req := &ai.Request{
		System: `You are a code reviewer. Review the provided git diff and provide:
1. A summary of the changes
2. Potential bugs or issues
3. Suggestions for improvement
4. Security concerns (if any)

Be concise and actionable.`,
		Prompt: fmt.Sprintf("Review these changes:\n\n%s", diff),
	}

	ui.Inf("Reviewing staged changes...")
	fmt.Println()

	if err := provider.Stream(ctx, req, os.Stdout); err != nil {
		return fmt.Errorf("streaming review: %w", err)
	}
	fmt.Println()

	return nil
}

func runAICommit(_ *cobra.Command, _ []string) error {
	// Ensure we're inside a git repository
	checkCmd := exec.Command("git", "rev-parse", "--git-dir")
	if err := checkCmd.Run(); err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	// Get staged diff
	cmd := exec.Command("git", "diff", "--cached")
	diffOutput, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("getting staged changes: %w", err)
	}

	diff := string(diffOutput)
	if diff == "" {
		ui.Warn("No staged changes to commit")
		ui.Tip("Run `git add` to stage files first")
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	provider, err := getProvider(cfg)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	req := &ai.Request{
		System: `You are a commit message generator. Generate a conventional commit message following this format:

type: description

Valid types: feat, fix, chore, docs, refactor, test, ci

The description should be concise (under 70 characters) and explain WHAT and WHY, not HOW.
Only output the commit message, nothing else.`,
		Prompt: fmt.Sprintf("Generate a commit message for these changes:\n\n%s", diff),
	}

	ui.Inf("Generating commit message...")
	fmt.Println()

	resp, err := provider.Complete(ctx, req)
	if err != nil {
		return fmt.Errorf("generating message: %w", err)
	}

	message := strings.TrimSpace(resp.Content)
	fmt.Println(ui.Accent.Render(message))
	fmt.Println()

	// Offer to commit with this message
	fmt.Print("Use this message? [y/N]: ")
	var answer string
	fmt.Scanln(&answer)

	normalized := strings.ToLower(strings.TrimSpace(answer))
	if normalized == "y" || normalized == "yes" {
		cmd := exec.Command("git", "commit", "-m", message)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("committing changes: %w", err)
		}
		ui.Ok("Changes committed successfully")
	} else {
		ui.Inf("Commit cancelled")
	}

	return nil
}

func getProvider(cfg *config.Config) (ai.Provider, error) {
	ks := ai.NewKeyStore()
	if err := ks.Load(); err != nil {
		return nil, fmt.Errorf("loading keys: %w", err)
	}

	model := cfg.AI.Model
	if aiModelFlag != "" {
		model = aiModelFlag
	}

	switch cfg.AI.Provider {
	case "claude":
		apiKey, err := ks.Get("claude")
		if err != nil {
			return nil, err
		}
		return ai.NewClaudeProvider(apiKey, model), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s (only 'claude' is supported)", cfg.AI.Provider)
	}
}

func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "..." + key[len(key)-4:]
}
