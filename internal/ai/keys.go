package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rnwolfe/mine/internal/config"
)

const (
	// EnvClaudeKey is the environment variable for Claude API key.
	EnvClaudeKey = "ANTHROPIC_API_KEY"
	// EnvOpenAIKey is the environment variable for OpenAI API key.
	EnvOpenAIKey = "OPENAI_API_KEY"
)

// KeyStore manages secure API key storage.
// For simplicity, keys are stored in a JSON file with 0600 permissions
// in the config directory. A future enhancement could use OS keyring.
type KeyStore struct {
	path string
	keys map[string]string
}

// NewKeyStore creates a new KeyStore instance.
func NewKeyStore() *KeyStore {
	paths := config.GetPaths()
	return &KeyStore{
		path: filepath.Join(paths.ConfigDir, "keys.json"),
		keys: make(map[string]string),
	}
}

// Load reads keys from disk.
func (ks *KeyStore) Load() error {
	data, err := os.ReadFile(ks.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No keys stored yet
		}
		return fmt.Errorf("reading keys: %w", err)
	}

	if err := json.Unmarshal(data, &ks.keys); err != nil {
		return fmt.Errorf("parsing keys: %w", err)
	}
	return nil
}

// Save writes keys to disk with restricted permissions.
func (ks *KeyStore) Save() error {
	paths := config.GetPaths()
	if err := paths.EnsureDirs(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(ks.keys, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding keys: %w", err)
	}

	// Write with 0600 permissions (owner read/write only)
	if err := os.WriteFile(ks.path, data, 0o600); err != nil {
		return fmt.Errorf("writing keys: %w", err)
	}

	// Ensure permissions are 0600 even if the file already existed with different mode.
	if err := os.Chmod(ks.path, 0o600); err != nil {
		return fmt.Errorf("setting key file permissions: %w", err)
	}
	return nil
}

// Get retrieves an API key for the given provider.
// It checks environment variables first, then the key store.
func (ks *KeyStore) Get(provider string) (string, error) {
	// Check environment variables first
	envKey := envKeyForProvider(provider)
	if key := os.Getenv(envKey); key != "" {
		return key, nil
	}

	// Check stored keys
	key, ok := ks.keys[provider]
	if !ok || key == "" {
		return "", &ProviderError{
			Provider: provider,
			Message:  fmt.Sprintf("API key not found (set %s or run `mine ai config`)", envKey),
		}
	}
	return key, nil
}

// Set stores an API key for the given provider.
func (ks *KeyStore) Set(provider, key string) {
	ks.keys[provider] = key
}

// Delete removes an API key for the given provider.
func (ks *KeyStore) Delete(provider string) {
	delete(ks.keys, provider)
}

// envKeyForProvider returns the environment variable name for the given provider.
func envKeyForProvider(provider string) string {
	switch provider {
	case "claude":
		return EnvClaudeKey
	case "openai":
		return EnvOpenAIKey
	default:
		return ""
	}
}
