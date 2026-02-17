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
