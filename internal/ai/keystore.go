package ai

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rnwolfe/mine/internal/config"
)

// KeyStore handles secure storage of API keys.
type KeyStore struct {
	keysFile string
}

// NewKeyStore creates a new key store.
func NewKeyStore() *KeyStore {
	paths := config.GetPaths()
	return &KeyStore{
		keysFile: filepath.Join(paths.ConfigDir, "ai_keys"),
	}
}

// Get retrieves an API key for a provider.
// Returns empty string if not found.
func (k *KeyStore) Get(provider string) (string, error) {
	// Try environment variable first
	envKey := fmt.Sprintf("MINE_AI_%s_KEY", provider)
	if key := os.Getenv(envKey); key != "" {
		return key, nil
	}

	// Try secure file
	data, err := os.ReadFile(k.keysFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("read keys file: %w", err)
	}

	// Simple line-based format: provider=key
	lines := splitLines(string(data))
	prefix := provider + "="
	for _, line := range lines {
		if len(line) > len(prefix) && line[:len(prefix)] == prefix {
			return line[len(prefix):], nil
		}
	}

	return "", nil
}

// Set stores an API key for a provider.
func (k *KeyStore) Set(provider, key string) error {
	// Read existing keys
	existing := make(map[string]string)
	data, err := os.ReadFile(k.keysFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read keys file: %w", err)
	}

	if len(data) > 0 {
		lines := splitLines(string(data))
		for _, line := range lines {
			if idx := findByte(line, '='); idx != -1 {
				existing[line[:idx]] = line[idx+1:]
			}
		}
	}

	// Update the key
	existing[provider] = key

	// Write back
	var content string
	for p, k := range existing {
		content += p + "=" + k + "\n"
	}

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(k.keysFile), 0o700); err != nil {
		return fmt.Errorf("create keys dir: %w", err)
	}

	// Write with restricted permissions (only owner can read)
	if err := os.WriteFile(k.keysFile, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write keys file: %w", err)
	}

	return nil
}

// Delete removes a stored API key.
func (k *KeyStore) Delete(provider string) error {
	// Read existing keys
	data, err := os.ReadFile(k.keysFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read keys file: %w", err)
	}

	// Filter out the provider
	lines := splitLines(string(data))
	prefix := provider + "="
	var newLines []string
	for _, line := range lines {
		if len(line) <= len(prefix) || line[:len(prefix)] != prefix {
			newLines = append(newLines, line)
		}
	}

	// Write back
	var content string
	for _, line := range newLines {
		content += line + "\n"
	}

	return os.WriteFile(k.keysFile, []byte(content), 0o600)
}

// splitLines splits a string into lines, trimming empty ones.
func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 {
				lines = append(lines, line)
			}
			start = i + 1
		}
	}
	if start < len(s) {
		line := s[start:]
		if len(line) > 0 {
			lines = append(lines, line)
		}
	}
	return lines
}

// findByte finds the first occurrence of a byte in a string.
func findByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}
