package ai

import (
	"os"
	"path/filepath"
	"testing"
)

func TestKeyStore(t *testing.T) {
	// Create temp directory for testing
	tmpDir := t.TempDir()
	keysFile := filepath.Join(tmpDir, "ai_keys")

	ks := &KeyStore{
		keysFile: keysFile,
	}

	// Test Set
	if err := ks.Set("claude", "test-key-123"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Verify file permissions
	info, err := os.Stat(keysFile)
	if err != nil {
		t.Fatalf("stat keys file: %v", err)
	}
	mode := info.Mode().Perm()
	if mode != 0o600 {
		t.Errorf("expected file mode 0600, got %o", mode)
	}

	// Test Get
	key, err := ks.Get("claude")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if key != "test-key-123" {
		t.Errorf("expected 'test-key-123', got %q", key)
	}

	// Test Get non-existent key
	key, err = ks.Get("openai")
	if err != nil {
		t.Fatalf("Get for non-existent key should not error: %v", err)
	}
	if key != "" {
		t.Errorf("expected empty string for non-existent key, got %q", key)
	}

	// Test Set multiple providers
	if err := ks.Set("openai", "openai-key-456"); err != nil {
		t.Fatalf("Set openai failed: %v", err)
	}

	claudeKey, _ := ks.Get("claude")
	openaiKey, _ := ks.Get("openai")

	if claudeKey != "test-key-123" {
		t.Errorf("claude key changed unexpectedly: %q", claudeKey)
	}
	if openaiKey != "openai-key-456" {
		t.Errorf("expected 'openai-key-456', got %q", openaiKey)
	}

	// Test Delete
	if err := ks.Delete("claude"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	key, err = ks.Get("claude")
	if err != nil {
		t.Fatalf("Get after delete should not error: %v", err)
	}
	if key != "" {
		t.Errorf("expected empty string after delete, got %q", key)
	}

	// Verify openai key still exists
	openaiKey, _ = ks.Get("openai")
	if openaiKey != "openai-key-456" {
		t.Errorf("openai key should still exist, got %q", openaiKey)
	}
}

func TestKeyStoreEnvironmentVariable(t *testing.T) {
	// Set environment variable
	os.Setenv("MINE_AI_claude_KEY", "env-key-789")
	defer os.Unsetenv("MINE_AI_claude_KEY")

	tmpDir := t.TempDir()
	ks := &KeyStore{
		keysFile: filepath.Join(tmpDir, "ai_keys"),
	}

	// Environment variable should take precedence
	key, err := ks.Get("claude")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if key != "env-key-789" {
		t.Errorf("expected env var 'env-key-789', got %q", key)
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", []string{}},
		{"line1", []string{"line1"}},
		{"line1\nline2", []string{"line1", "line2"}},
		{"line1\nline2\n", []string{"line1", "line2"}},
		{"\n\nline1\n\nline2\n\n", []string{"line1", "line2"}},
	}

	for _, tc := range tests {
		result := splitLines(tc.input)
		if len(result) != len(tc.expected) {
			t.Errorf("input %q: expected %d lines, got %d", tc.input, len(tc.expected), len(result))
			continue
		}
		for i, line := range result {
			if line != tc.expected[i] {
				t.Errorf("input %q: line %d: expected %q, got %q", tc.input, i, tc.expected[i], line)
			}
		}
	}
}

func TestFindByte(t *testing.T) {
	tests := []struct {
		input    string
		b        byte
		expected int
	}{
		{"hello=world", '=', 5},
		{"no equals", '=', -1},
		{"", '=', -1},
		{"=start", '=', 0},
	}

	for _, tc := range tests {
		result := findByte(tc.input, tc.b)
		if result != tc.expected {
			t.Errorf("input %q, byte %q: expected %d, got %d", tc.input, tc.b, tc.expected, result)
		}
	}
}
