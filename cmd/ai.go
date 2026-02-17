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

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "AI-powered development helpers",
	Long:  `Configure AI providers and use AI helpers for code review, commit messages, and quick questions.`,
	RunE:  hook.Wrap("ai", runAIHelp),
}

var aiModel string

func init() {
	aiCmd.PersistentFlags().StringVarP(&aiModel, "model", "m", "", "Override the configured model")

	aiCmd.AddCommand(aiConfigCmd)
	aiCmd.AddCommand(aiAskCmd)
	aiCmd.AddCommand(aiReviewCmd)
	aiCmd.AddCommand(aiCommitCmd)
	aiCmd.AddCommand(aiModelsCmd)
}

func runAIHelp(_ *cobra.Command, _ []string) error {
	fmt.Println()
	fmt.Println(ui.Title.Render("  mine ai") + ui.Muted.Render(" — AI-powered development helpers"))
	fmt.Println()
	fmt.Println(ui.Muted.Render("  Configure AI providers and use AI to help with your work."))
	fmt.Println()
	fmt.Println(ui.Accent.Render("  Commands:"))
	fmt.Println()
	fmt.Printf("    %s  Configure AI provider and API key\n", ui.KeyStyle.Render("config"))
	fmt.Printf("    %s   List available providers and models\n", ui.KeyStyle.Render("models"))
	fmt.Printf("    %s     Ask a quick question\n", ui.KeyStyle.Render("ask"))
	fmt.Printf("    %s    Review staged changes\n", ui.KeyStyle.Render("review"))
	fmt.Printf("    %s   Generate commit message from diff\n", ui.KeyStyle.Render("commit"))
	fmt.Println()
	fmt.Println(ui.Accent.Render("  Supported Providers:"))
	fmt.Println()
	fmt.Printf("    %s       Anthropic Claude (env: %s)\n",
		ui.KeyStyle.Render("claude"), ui.Muted.Render("ANTHROPIC_API_KEY"))
	fmt.Printf("              %s\n", ui.Muted.Render("Get key: https://console.anthropic.com/settings/keys"))
	fmt.Printf("    %s       OpenAI GPT models (env: %s)\n",
		ui.KeyStyle.Render("openai"), ui.Muted.Render("OPENAI_API_KEY"))
	fmt.Printf("              %s\n", ui.Muted.Render("Get key: https://platform.openai.com/api-keys"))
	fmt.Printf("    %s       Google Gemini (env: %s)\n",
		ui.KeyStyle.Render("gemini"), ui.Muted.Render("GEMINI_API_KEY"))
	fmt.Printf("              %s\n", ui.Muted.Render("Get key: https://aistudio.google.com/app/apikey"))
	fmt.Printf("    %s  OpenRouter with free models (env: %s)\n",
		ui.KeyStyle.Render("openrouter"), ui.Muted.Render("OPENROUTER_API_KEY"))
	fmt.Printf("              %s\n", ui.Muted.Render("Free models available: z-ai/glm-4.5-air:free"))
	fmt.Printf("              %s\n", ui.Muted.Render("Get key: https://openrouter.ai/keys"))
	fmt.Println()
	fmt.Println(ui.Accent.Render("  Zero-Config Setup:"))
	fmt.Println()
	fmt.Println(ui.Muted.Render("  mine automatically detects API keys from standard environment variables."))
	fmt.Println(ui.Muted.Render("  Set ANTHROPIC_API_KEY, OPENAI_API_KEY, or GEMINI_API_KEY and you're ready to go!"))
	fmt.Println()
	fmt.Println(ui.Muted.Render("  Examples:"))
	fmt.Println()
	fmt.Printf("    %s\n", ui.Muted.Render(`mine ai config --provider claude --key sk-...`))
	fmt.Printf("    %s\n", ui.Muted.Render(`mine ai ask "What's the difference between defer and panic?"`))
	fmt.Printf("    %s\n", ui.Muted.Render(`mine ai review`))
	fmt.Printf("    %s\n", ui.Muted.Render(`mine ai commit`))
	fmt.Println()
	return nil
}

// ai config command
var (
	aiConfigProvider string
	aiConfigKey      string
	aiConfigModel    string
	aiConfigList     bool
)

var aiConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Configure AI provider settings",
	Long:  `Set up your AI provider (Claude, OpenAI, etc.) and store your API key securely.`,
	RunE:  hook.Wrap("ai.config", runAIConfig),
}

func init() {
	aiConfigCmd.Flags().StringVarP(&aiConfigProvider, "provider", "p", "", "Provider name (claude, openai, ollama)")
	aiConfigCmd.Flags().StringVarP(&aiConfigKey, "key", "k", "", "API key")
	aiConfigCmd.Flags().StringVar(&aiConfigModel, "default-model", "", "Default model")
	aiConfigCmd.Flags().BoolVarP(&aiConfigList, "list", "l", false, "List configured providers")
}

func runAIConfig(_ *cobra.Command, _ []string) error {
	ks, err := ai.NewKeystore()
	if err != nil {
		return err
	}

	// List mode
	if aiConfigList {
		providers, err := ks.List()
		if err != nil {
			return err
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		fmt.Println()
		fmt.Println(ui.Title.Render("  Configured AI Providers"))
		fmt.Println()

		if len(providers) == 0 {
			fmt.Println(ui.Muted.Render("  No providers configured yet."))
			fmt.Println()
			fmt.Printf("  Get started: %s\n", ui.Accent.Render(`mine ai config --provider claude --key sk-...`))
			fmt.Println()
			return nil
		}

		for _, p := range providers {
			marker := "  "
			if cfg.AI.Provider == p {
				marker = ui.Success.Render("✓ ")
			}
			fmt.Printf("%s %s", marker, ui.KeyStyle.Render(p))
			if cfg.AI.Provider == p && cfg.AI.Model != "" {
				fmt.Printf(" %s", ui.Muted.Render(fmt.Sprintf("(model: %s)", cfg.AI.Model)))
			}
			fmt.Println()
		}
		fmt.Println()
		return nil
	}

	// Set provider
	if aiConfigProvider == "" {
		return fmt.Errorf("--provider is required (use --list to see configured providers)")
	}

	// Store API key
	if aiConfigKey != "" {
		if err := ks.Set(aiConfigProvider, aiConfigKey); err != nil {
			return err
		}
		fmt.Println()
		fmt.Printf("%s API key stored securely for %s\n", ui.IconVault, ui.Accent.Render(aiConfigProvider))
	}

	// Update config
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	cfg.AI.Provider = aiConfigProvider
	if aiConfigModel != "" {
		cfg.AI.Model = aiConfigModel
	}

	if err := config.Save(cfg); err != nil {
		return err
	}

	fmt.Printf("%s Default provider set to %s\n", ui.IconOk, ui.Accent.Render(aiConfigProvider))
	if cfg.AI.Model != "" {
		fmt.Printf("%s Default model: %s\n", ui.IconOk, ui.Muted.Render(cfg.AI.Model))
	}
	fmt.Println()

	return nil
}

// ai ask command
var aiAskCmd = &cobra.Command{
	Use:   "ask <question>",
	Short: "Ask a quick question",
	Long:  `Send a question to your configured AI provider and get an answer.`,
	Args:  cobra.MinimumNArgs(1),
	RunE:  hook.Wrap("ai.ask", runAIAsk),
}

func runAIAsk(_ *cobra.Command, args []string) error {
	question := strings.Join(args, " ")

	provider, err := getConfiguredProvider()
	if err != nil {
		return err
	}

	req := ai.NewRequest(question)
	if aiModel != "" {
		req.Model = aiModel
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Println()
	fmt.Println(ui.Muted.Render(fmt.Sprintf("  Asking %s...", provider.Name())))
	fmt.Println()

	// Stream the response
	if err := provider.Stream(ctx, req, os.Stdout); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println()
	return nil
}

// ai review command
var aiReviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Review staged changes with AI",
	Long:  `Get an AI-powered code review of your staged git changes.`,
	RunE:  hook.Wrap("ai.review", runAIReview),
}

func runAIReview(_ *cobra.Command, _ []string) error {
	// Get git diff of staged changes
	cmd := exec.Command("git", "diff", "--cached")
	output, err := cmd.CombinedOutput()
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

	provider, err := getConfiguredProvider()
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

	req := ai.NewRequest(prompt)
	req.System = "You are an expert code reviewer. Provide constructive, specific feedback."
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

	if err := provider.Stream(ctx, req, os.Stdout); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println()
	return nil
}

// ai commit command
var aiCommitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Generate a commit message from diff",
	Long:  `Analyze your staged changes and generate a clear commit message.`,
	RunE:  hook.Wrap("ai.commit", runAICommit),
}

func runAICommit(_ *cobra.Command, _ []string) error {
	// Get git diff of staged changes
	cmd := exec.Command("git", "diff", "--cached")
	output, err := cmd.CombinedOutput()
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

	provider, err := getConfiguredProvider()
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

	req := ai.NewRequest(prompt)
	req.System = "You are a git commit message expert. Write clear, professional commit messages."
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
	cmd = exec.Command("git", "commit", "-m", message)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create commit: %w", err)
	}

	fmt.Println()
	fmt.Println(ui.Success.Render("  Commit created successfully!"))
	fmt.Println()

	return nil
}

// Helper to get a configured provider
func getConfiguredProvider() (ai.Provider, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	if cfg.AI.Provider == "" {
		return nil, fmt.Errorf(`no AI provider configured

Get started:
  • Set environment variable (e.g., ANTHROPIC_API_KEY, OPENAI_API_KEY)
  • Or run: mine ai config --provider claude --key sk-...
  • Or use free model: mine ai config --provider openrouter --default-model z-ai/glm-4.5-air:free

See providers: mine ai --help`)
	}

	ks, err := ai.NewKeystore()
	if err != nil {
		return nil, err
	}

	apiKey, err := ks.Get(cfg.AI.Provider)
	if err != nil {
		// Provide provider-specific help for where to get API keys
		helpMsg := ""
		switch cfg.AI.Provider {
		case "claude":
			helpMsg = fmt.Sprintf(`API key not found for Claude.

Options:
  • Set environment variable: export ANTHROPIC_API_KEY=sk-...
  • Get a key: https://console.anthropic.com/settings/keys
  • Or run: mine ai config --provider claude --key <your-key>`)
		case "openai":
			helpMsg = fmt.Sprintf(`API key not found for OpenAI.

Options:
  • Set environment variable: export OPENAI_API_KEY=sk-...
  • Get a key: https://platform.openai.com/api-keys
  • Or run: mine ai config --provider openai --key <your-key>`)
		case "gemini":
			helpMsg = fmt.Sprintf(`API key not found for Gemini.

Options:
  • Set environment variable: export GEMINI_API_KEY=AIza...
  • Get a key: https://aistudio.google.com/app/apikey
  • Or run: mine ai config --provider gemini --key <your-key>`)
		case "openrouter":
			helpMsg = fmt.Sprintf(`API key not found for OpenRouter.

OpenRouter provides access to free AI models. Get your free API key:
  • Visit: https://openrouter.ai/keys (sign up free, no credit card)
  • Set: export OPENROUTER_API_KEY=sk-or-v1-...
  • Or run: mine ai config --provider openrouter --key <your-key>
  • Free models: z-ai/glm-4.5-air:free, google/gemini-flash-1.5`)
		default:
			helpMsg = fmt.Sprintf("Run: mine ai config --provider %s --key <your-key>", cfg.AI.Provider)
		}
		return nil, fmt.Errorf("%s", helpMsg)
	}

	provider, err := ai.GetProvider(cfg.AI.Provider, apiKey)
	if err != nil {
		return nil, err
	}

	return provider, nil
}

// ai models command
var aiModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List available AI providers and models",
	Long:  `Show configured providers, available providers, and suggested models for each.`,
	RunE:  hook.Wrap("ai.models", runAIModels),
}

func runAIModels(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ks, err := ai.NewKeystore()
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println(ui.Title.Render("  AI Providers & Models"))
	fmt.Println()

	// Get all registered providers
	allProviders := ai.ListProviders()

	// Get providers with stored keys
	configuredProviders, err := ks.List()
	if err != nil {
		return err
	}

	// Create a map for faster lookup
	configuredMap := make(map[string]bool)
	for _, p := range configuredProviders {
		configuredMap[p] = true
	}

	// Check which providers have env vars
	envProviders := detectAIKeys()

	// Provider details with suggested models and env var info
	providerInfo := map[string]struct {
		envVar         string
		models         []string
		apiKeyURL      string
		requiresAPIKey bool
	}{
		"claude": {
			envVar:         "ANTHROPIC_API_KEY",
			models:         []string{"claude-sonnet-4-5-20250929", "claude-opus-4-6"},
			apiKeyURL:      "https://console.anthropic.com/settings/keys",
			requiresAPIKey: true,
		},
		"openai": {
			envVar:         "OPENAI_API_KEY",
			models:         []string{"gpt-5.2", "gpt-4o", "o3-mini"},
			apiKeyURL:      "https://platform.openai.com/api-keys",
			requiresAPIKey: true,
		},
		"gemini": {
			envVar:         "GEMINI_API_KEY",
			models:         []string{"gemini-3-flash-preview", "gemini-3-pro-preview"},
			apiKeyURL:      "https://aistudio.google.com/app/apikey",
			requiresAPIKey: true,
		},
		"openrouter": {
			envVar:         "OPENROUTER_API_KEY",
			models:         []string{"z-ai/glm-4.5-air:free (free model)", "google/gemini-flash-1.5", "anthropic/claude-3.5-sonnet"},
			apiKeyURL:      "https://openrouter.ai/keys",
			requiresAPIKey: true,
		},
	}

	for _, provider := range allProviders {
		info, ok := providerInfo[provider]
		if !ok {
			continue // Skip unknown providers
		}

		// Determine status
		status := ""
		isDefault := cfg.AI.Provider == provider
		hasKey := configuredMap[provider] || envProviders[provider]

		if isDefault {
			status = ui.Success.Render("✓ DEFAULT")
		} else if hasKey {
			status = ui.Success.Render("✓ Ready")
		} else {
			status = ui.Muted.Render("○ Not configured")
		}

		// Print provider header
		fmt.Printf("  %s %s\n", ui.KeyStyle.Render(provider), status)

		// Show env var status
		if envProviders[provider] {
			fmt.Printf("    %s\n", ui.Success.Render(fmt.Sprintf("API key detected in %s", info.envVar)))
		} else if configuredMap[provider] {
			fmt.Printf("    %s\n", ui.Muted.Render("API key stored in keystore"))
		} else {
			fmt.Printf("    %s %s\n",
				ui.Muted.Render(fmt.Sprintf("No API key (set %s or use:", info.envVar)),
				ui.Accent.Render(fmt.Sprintf("mine ai config --provider %s --key <key>", provider)))
			fmt.Printf("    %s\n", ui.Muted.Render(fmt.Sprintf("Get key: %s", info.apiKeyURL)))
		}

		// Show model if this is the default provider
		if isDefault && cfg.AI.Model != "" {
			fmt.Printf("    %s %s\n", ui.Muted.Render("Default model:"), ui.Accent.Render(cfg.AI.Model))
		}

		// Show suggested models
		fmt.Printf("    %s\n", ui.Muted.Render("Suggested models:"))
		for _, model := range info.models {
			fmt.Printf("      • %s\n", ui.Muted.Render(model))
		}

		fmt.Println()
	}

	// Show helpful examples
	fmt.Println(ui.Accent.Render("  Examples:"))
	fmt.Println()
	fmt.Printf("    %s\n", ui.Muted.Render("mine ai config --provider claude --key sk-..."))
	fmt.Printf("    %s\n", ui.Muted.Render("mine ai ask \"explain Go interfaces\" --model gemini-3-flash-preview"))
	fmt.Printf("    %s\n", ui.Muted.Render("export ANTHROPIC_API_KEY=sk-...  # Zero-config setup"))
	fmt.Println()

	return nil
}
