package ai

import (
	"os"
	"path/filepath"
	"testing"
)

func TestKeystoreSetAndGet(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()

	// Override config paths for testing
	ks := &Keystore{
		path: filepath.Join(tmpDir, "keystore.enc"),
		key:  []byte("01234567890123456789012345678901"), // 32 bytes
	}

	// Test Set
	err := ks.Set("test-provider", "sk-test-key-12345")
	if err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Test Get
	key, err := ks.Get("test-provider")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if key != "sk-test-key-12345" {
		t.Errorf("expected key 'sk-test-key-12345', got '%s'", key)
	}

	// Test Get non-existent
	_, err = ks.Get("non-existent")
	if err == nil {
		t.Error("expected error for non-existent provider")
	}
}

func TestKeystoreMultipleProviders(t *testing.T) {
	tmpDir := t.TempDir()
	ks := &Keystore{
		path: filepath.Join(tmpDir, "keystore.enc"),
		key:  []byte("01234567890123456789012345678901"),
	}

	// Set multiple keys
	providers := map[string]string{
		"claude": "sk-claude-key",
		"openai": "sk-openai-key",
		"ollama": "ollama-key",
	}

	for provider, key := range providers {
		if err := ks.Set(provider, key); err != nil {
			t.Fatalf("Set failed for %s: %v", provider, err)
		}
	}

	// Verify all keys
	for provider, expectedKey := range providers {
		key, err := ks.Get(provider)
		if err != nil {
			t.Fatalf("Get failed for %s: %v", provider, err)
		}
		if key != expectedKey {
			t.Errorf("provider %s: expected key '%s', got '%s'", provider, expectedKey, key)
		}
	}

	// Test List
	list, err := ks.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(list) != 3 {
		t.Errorf("expected 3 providers, got %d", len(list))
	}
}

func TestKeystoreDelete(t *testing.T) {
	tmpDir := t.TempDir()
	ks := &Keystore{
		path: filepath.Join(tmpDir, "keystore.enc"),
		key:  []byte("01234567890123456789012345678901"),
	}

	// Set a key
	if err := ks.Set("test", "key123"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Delete it
	if err := ks.Delete("test"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's gone
	_, err := ks.Get("test")
	if err == nil {
		t.Error("expected error after deletion")
	}

	// Delete non-existent (should not error)
	if err := ks.Delete("non-existent"); err != nil {
		t.Errorf("Delete of non-existent should not error: %v", err)
	}
}

func TestKeystoreEncryption(t *testing.T) {
	tmpDir := t.TempDir()
	ks := &Keystore{
		path: filepath.Join(tmpDir, "keystore.enc"),
		key:  []byte("01234567890123456789012345678901"),
	}

	// Set a key
	originalKey := "super-secret-api-key-12345"
	if err := ks.Set("test", originalKey); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Read the file directly
	data, err := os.ReadFile(ks.path)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	// The file should NOT contain the plaintext key
	if contains(data, []byte(originalKey)) {
		t.Error("keystore file contains plaintext API key (not encrypted!)")
	}

	// But Get should still return the correct key
	key, err := ks.Get("test")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if key != originalKey {
		t.Errorf("expected key '%s', got '%s'", originalKey, key)
	}
}

func TestKeystoreFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	ks := &Keystore{
		path: filepath.Join(tmpDir, "keystore.enc"),
		key:  []byte("01234567890123456789012345678901"),
	}

	if err := ks.Set("test", "key"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	info, err := os.Stat(ks.path)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	mode := info.Mode().Perm()
	expected := os.FileMode(0o600)

	if mode != expected {
		t.Errorf("expected file permissions %o, got %o", expected, mode)
	}
}

func TestKeystoreGetFromEnv_WithEnvVar(t *testing.T) {
	tmpDir := t.TempDir()
	ks := &Keystore{
		path: tmpDir + "/keystore.enc", // no file created
		key:  []byte("01234567890123456789012345678901"),
	}

	// Set env var for each known provider
	tests := []struct {
		provider string
		envVar   string
	}{
		{"claude", "ANTHROPIC_API_KEY"},
		{"openai", "OPENAI_API_KEY"},
		{"gemini", "GEMINI_API_KEY"},
		{"openrouter", "OPENROUTER_API_KEY"},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			t.Setenv(tt.envVar, "env-key-for-"+tt.provider)

			key, err := ks.Get(tt.provider)
			if err != nil {
				t.Fatalf("Get(%s) failed: %v", tt.provider, err)
			}
			if key != "env-key-for-"+tt.provider {
				t.Errorf("expected env key, got '%s'", key)
			}
		})
	}
}

func TestKeystoreGetFromEnv_OpenRouterEmptyAllowed(t *testing.T) {
	tmpDir := t.TempDir()
	ks := &Keystore{
		path: tmpDir + "/keystore.enc", // no file
		key:  []byte("01234567890123456789012345678901"),
	}

	// openrouter returns empty key (not an error) when env var is unset
	t.Setenv("OPENROUTER_API_KEY", "")

	key, err := ks.Get("openrouter")
	if err != nil {
		t.Fatalf("expected no error for openrouter with empty env, got: %v", err)
	}
	if key != "" {
		t.Errorf("expected empty key for openrouter, got '%s'", key)
	}
}

func TestKeystoreGetFromEnv_UnknownProvider(t *testing.T) {
	tmpDir := t.TempDir()
	ks := &Keystore{
		path: tmpDir + "/keystore.enc", // no file
		key:  []byte("01234567890123456789012345678901"),
	}

	_, err := ks.Get("unknown-provider-xyz")
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
	if !contains([]byte(err.Error()), []byte("unknown-provider-xyz")) {
		t.Errorf("expected error message to contain provider name, got: %v", err)
	}
}

func TestKeystoreGetFromEnv_KnownProviderMissingEnv(t *testing.T) {
	tmpDir := t.TempDir()
	ks := &Keystore{
		path: tmpDir + "/keystore.enc", // no file
		key:  []byte("01234567890123456789012345678901"),
	}

	// Ensure the env var is not set
	t.Setenv("ANTHROPIC_API_KEY", "")

	_, err := ks.Get("claude")
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestKeystoreNewKeystore(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)

	ks, err := NewKeystore()
	if err != nil {
		t.Fatalf("NewKeystore failed: %v", err)
	}
	if ks == nil {
		t.Fatal("expected non-nil keystore")
	}
	if ks.path == "" {
		t.Error("expected non-empty keystore path")
	}
	if len(ks.key) != 32 {
		t.Errorf("expected 32-byte key, got %d bytes", len(ks.key))
	}
}

func TestKeystoreDeleteNonExistentFile(t *testing.T) {
	tmpDir := t.TempDir()
	ks := &Keystore{
		path: tmpDir + "/no-such-file.enc",
		key:  []byte("01234567890123456789012345678901"),
	}

	// Delete when no keystore file exists should not error
	if err := ks.Delete("any-provider"); err != nil {
		t.Errorf("Delete on missing file should not error: %v", err)
	}
}

func TestKeystoreListEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	ks := &Keystore{
		path: tmpDir + "/no-such-file.enc",
		key:  []byte("01234567890123456789012345678901"),
	}

	providers, err := ks.List()
	if err != nil {
		t.Fatalf("List on empty keystore failed: %v", err)
	}
	if len(providers) != 0 {
		t.Errorf("expected empty list, got %d providers", len(providers))
	}
}

// Helper function to check if a byte slice contains a subsequence
func contains(data, substr []byte) bool {
	for i := 0; i <= len(data)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if data[i+j] != substr[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
