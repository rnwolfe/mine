package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rnwolfe/mine/internal/ai"
	"github.com/rnwolfe/mine/internal/config"
	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/rnwolfe/mine/internal/vault"
	"github.com/spf13/cobra"
)

var aiCmd = &cobra.Command{
	Use:   "ai",
	Short: "Your AI co-pilot for code review, commits, and questions",
	Long:  `Configure AI providers and use AI helpers for code review, commit messages, and quick questions.`,
	RunE:  hook.Wrap("ai", runAIHelp),
}

var (
	aiModel     string
	aiAskSystem string
	aiAskRaw    bool
)

func init() {
	aiCmd.PersistentFlags().StringVarP(&aiModel, "model", "m", "", "Override the configured model")

	aiAskCmd.Flags().StringVar(&aiAskSystem, "system", "", "Override system instructions for this invocation (empty string disables system instructions)")
	aiAskCmd.Flags().BoolVar(&aiAskRaw, "raw", false, "Output raw markdown without terminal rendering")

	aiCmd.AddCommand(aiConfigCmd)
	aiCmd.AddCommand(aiAskCmd)
	aiCmd.AddCommand(aiReviewCmd)
	aiCmd.AddCommand(aiCommitCmd)
	aiCmd.AddCommand(aiModelsCmd)
}

// resolveSystemInstructions returns the effective system instruction string according
// to the precedence rules:
//  1. --system flag (including empty string, which disables system instructions)
//  2. per-subcommand config default
//  3. global config default
//  4. builtinDefault (only when no custom value is configured/provided)
func resolveSystemInstructions(cfg *config.AIConfig, subcommand, flagValue string, flagChanged bool, builtinDefault string) string {
	if flagChanged {
		return flagValue
	}
	switch subcommand {
	case "ask":
		if cfg.AskSystemInstructions != "" {
			return cfg.AskSystemInstructions
		}
	case "review":
		if cfg.ReviewSystemInstructions != "" {
			return cfg.ReviewSystemInstructions
		}
	case "commit":
		if cfg.CommitSystemInstructions != "" {
			return cfg.CommitSystemInstructions
		}
	}
	if cfg.SystemInstructions != "" {
		return cfg.SystemInstructions
	}
	return builtinDefault
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

// ai ask command
var aiAskCmd = &cobra.Command{
	Use:   "ask <question>",
	Short: "Ask your AI anything",
	Long:  `Send a question to your configured AI provider and get an answer.`,
	Args:  cobra.MinimumNArgs(1),
	RunE:  hook.Wrap("ai.ask", runAIAsk),
}

func runAIAsk(cmd *cobra.Command, args []string) error {
	question := strings.Join(args, " ")

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	provider, err := getConfiguredProviderFromConfig(cfg)
	if err != nil {
		return err
	}

	req := ai.NewRequest(question)
	req.System = resolveSystemInstructions(&cfg.AI, "ask", aiAskSystem, cmd.Flags().Changed("system"), "")
	if aiModel != "" {
		req.Model = aiModel
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Println()
	fmt.Println(ui.Muted.Render(fmt.Sprintf("  Asking %s...", provider.Name())))
	fmt.Println()

	// Stream the response through a markdown-aware writer.
	mdw := ui.NewMarkdownWriter(os.Stdout, aiAskRaw)
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

// getConfiguredProvider loads config and returns the configured AI provider.
func getConfiguredProvider() (ai.Provider, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	return getConfiguredProviderFromConfig(cfg)
}

// getConfiguredProviderFromConfig returns the configured AI provider using a pre-loaded config.
func getConfiguredProviderFromConfig(cfg *config.Config) (ai.Provider, error) {
	if cfg.AI.Provider == "" {
		return nil, fmt.Errorf(`no AI provider configured

Get started:
  • Set environment variable (e.g., ANTHROPIC_API_KEY, OPENAI_API_KEY)
  • Or run: mine ai config --provider claude --key sk-...
  • Or use free model: mine ai config --provider openrouter --default-model z-ai/glm-4.5-air:free

See providers: mine ai --help`)
	}

	apiKey, err := getAIKey(cfg.AI.Provider)
	if err != nil {
		// Provide provider-specific help for where to get API keys
		helpMsg := ""
		switch cfg.AI.Provider {
		case "claude":
			helpMsg = `API key not found for Claude.

Options:
  • Set environment variable: export ANTHROPIC_API_KEY=sk-...
  • Get a key: https://console.anthropic.com/settings/keys
  • Or run: mine ai config --provider claude --key <your-key>`
		case "openai":
			helpMsg = `API key not found for OpenAI.

Options:
  • Set environment variable: export OPENAI_API_KEY=sk-...
  • Get a key: https://platform.openai.com/api-keys
  • Or run: mine ai config --provider openai --key <your-key>`
		case "gemini":
			helpMsg = `API key not found for Gemini.

Options:
  • Set environment variable: export GEMINI_API_KEY=AIza...
  • Get a key: https://aistudio.google.com/app/apikey
  • Or run: mine ai config --provider gemini --key <your-key>`
		case "openrouter":
			helpMsg = `API key not found for OpenRouter.

OpenRouter provides access to free AI models. Get your free API key:
  • Visit: https://openrouter.ai/keys (sign up free, no credit card)
  • Set: export OPENROUTER_API_KEY=sk-or-v1-...
  • Or run: mine ai config --provider openrouter --key <your-key>
  • Free models: z-ai/glm-4.5-air:free, google/gemini-flash-1.5`
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

// aiVaultKey returns the vault key for an AI provider's API key.
func aiVaultKey(provider string) string {
	return "ai." + provider + ".api_key"
}

// getAIKey retrieves an AI provider API key, checking env vars first, then vault.
func getAIKey(provider string) (string, error) {
	// Check env vars first (zero-config setup).
	envVars := map[string]string{
		"claude":     "ANTHROPIC_API_KEY",
		"openai":     "OPENAI_API_KEY",
		"gemini":     "GEMINI_API_KEY",
		"openrouter": "OPENROUTER_API_KEY",
	}
	if envVar, ok := envVars[provider]; ok {
		if key := os.Getenv(envVar); key != "" {
			return key, nil
		}
	}

	// Try vault (requires passphrase from env var or prompt).
	// Skip the prompt entirely if the vault file doesn't exist yet — no point
	// asking for a passphrase if there's nothing to unlock.
	paths := config.GetPaths()
	vaultPath := filepath.Join(paths.DataDir, "vault.age")
	if _, statErr := os.Stat(vaultPath); os.IsNotExist(statErr) {
		return vaultFallback(provider)
	}
	passphrase, err := readPassphrase(false)
	if err != nil {
		// Vault not configured — fall through to "not found" error.
		return vaultFallback(provider)
	}
	v := vault.New(passphrase)
	key, err := v.Get(aiVaultKey(provider))
	if err == nil {
		return key, nil
	}
	// Surface vault-specific errors directly so users get actionable feedback.
	if errors.Is(err, vault.ErrWrongPassphrase) || errors.Is(err, vault.ErrCorruptedVault) {
		return "", err
	}

	return vaultFallback(provider)
}

// vaultFallback returns the appropriate error for a missing AI key.
func vaultFallback(provider string) (string, error) {
	// OpenRouter free models don't require an API key.
	if provider == "openrouter" {
		return "", nil
	}
	return "", fmt.Errorf("no API key configured for %s", provider)
}

// aiVaultProviders returns AI provider names that have keys stored in vault.
func aiVaultProviders() ([]string, error) {
	passphrase := os.Getenv("MINE_VAULT_PASSPHRASE")
	if passphrase == "" {
		return nil, nil // No passphrase available — return empty list silently.
	}
	v := vault.New(passphrase)
	allKeys, err := v.List()
	if err != nil {
		return nil, err
	}

	var providers []string
	for _, k := range allKeys {
		// Extract provider from "ai.<provider>.api_key".
		if strings.HasPrefix(k, "ai.") && strings.HasSuffix(k, ".api_key") {
			provider := strings.TrimSuffix(strings.TrimPrefix(k, "ai."), ".api_key")
			providers = append(providers, provider)
		}
	}
	return providers, nil
}
