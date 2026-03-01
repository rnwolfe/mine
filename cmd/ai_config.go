package cmd

import (
	"fmt"
	"os"

	"github.com/rnwolfe/mine/internal/ai"
	"github.com/rnwolfe/mine/internal/config"
	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/rnwolfe/mine/internal/vault"
	"github.com/spf13/cobra"
)

// ai config command
var (
	aiConfigProvider string
	aiConfigKey      string
	aiConfigModel    string
	aiConfigList     bool
)

var aiConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Set up your AI provider and API key",
	Long:  `Set up your AI provider (Claude, OpenAI, etc.) and store your API key securely.`,
	RunE:  hook.Wrap("ai.config", runAIConfig),
}

func init() {
	aiConfigCmd.Flags().StringVarP(&aiConfigProvider, "provider", "p", "", "Provider name (claude, openai, gemini, openrouter; see 'mine ai models')")
	aiConfigCmd.Flags().StringVarP(&aiConfigKey, "key", "k", "", "API key")
	aiConfigCmd.Flags().StringVar(&aiConfigModel, "default-model", "", "Default model")
	aiConfigCmd.Flags().BoolVarP(&aiConfigList, "list", "l", false, "List configured providers")
}

func runAIConfig(_ *cobra.Command, _ []string) error {
	// List mode — read providers from vault.
	if aiConfigList {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		// Collect providers that have vault keys.
		providers, err := aiVaultProviders()
		if err != nil && !os.IsNotExist(err) {
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

	// Store API key in vault.
	if aiConfigKey != "" {
		passphrase, err := readPassphrase(false)
		if err != nil {
			return err
		}
		v := vault.New(passphrase)
		vaultKey := aiVaultKey(aiConfigProvider)
		if err := v.Set(vaultKey, aiConfigKey); err != nil {
			return fmt.Errorf("storing API key in vault: %w", err)
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

// ai models command
var aiModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "See what AI providers and models are available",
	Long:  `Show configured providers, available providers, and suggested models for each.`,
	RunE:  hook.Wrap("ai.models", runAIModels),
}

func runAIModels(_ *cobra.Command, _ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Println(ui.Title.Render("  AI Providers & Models"))
	fmt.Println()

	// Get all registered providers
	allProviders := ai.ListProviders()

	// Get providers with vault-stored keys (best-effort, silent if vault unavailable).
	vaultProviders, _ := aiVaultProviders()
	vaultMap := make(map[string]bool)
	for _, p := range vaultProviders {
		vaultMap[p] = true
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
		hasKey := vaultMap[provider] || envProviders[provider]

		if isDefault {
			status = ui.Success.Render("✓ DEFAULT")
		} else if hasKey {
			status = ui.Success.Render("✓ Ready")
		} else {
			status = ui.Muted.Render("○ Not configured")
		}

		// Print provider header
		fmt.Printf("  %s %s\n", ui.KeyStyle.Render(provider), status)

		// Show key source
		if envProviders[provider] {
			fmt.Printf("    %s\n", ui.Success.Render(fmt.Sprintf("API key detected in %s", info.envVar)))
		} else if vaultMap[provider] {
			fmt.Printf("    %s\n", ui.Muted.Render("API key stored in vault"))
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
