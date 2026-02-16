package ai

import (
	"fmt"
	"io"

	"github.com/rnwolfe/mine/internal/config"
)

// GetProvider returns a configured provider based on the config.
func GetProvider(cfg *config.Config, writer io.Writer) (Provider, error) {
	if cfg.AI.Provider == "" {
		cfg.AI.Provider = "claude"
	}

	// Get API key from keystore
	ks := NewKeyStore()
	apiKey, err := ks.Get(cfg.AI.Provider)
	if err != nil {
		return nil, fmt.Errorf("get API key: %w", err)
	}

	if apiKey == "" {
		return nil, fmt.Errorf("no API key configured for %s. Run: mine ai config", cfg.AI.Provider)
	}

	providerCfg := ProviderConfig{
		APIKey: apiKey,
		Model:  cfg.AI.Model,
		Writer: writer,
	}

	switch cfg.AI.Provider {
	case "claude", "anthropic":
		return NewClaudeProvider(providerCfg), nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", cfg.AI.Provider)
	}
}
