package ai

import (
	"os"
	"path/filepath"
	"testing"
)

func TestKeyStore(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()

	// Override the config path for testing
	originalHome := os.Getenv("HOME")
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Cleanup(func() {
		os.Setenv("HOME", originalHome)
		os.Unsetenv("XDG_CONFIG_HOME")
	})

	t.Run("new keystore is empty", func(t *testing.T) {
		ks := NewKeyStore()
		if err := ks.Load(); err != nil {
			t.Fatalf("unexpected error loading empty keystore: %v", err)
		}

		_, err := ks.Get("claude")
		if err == nil {
			t.Fatal("expected error for missing key, got nil")
		}
	})

	t.Run("set and get key", func(t *testing.T) {
		ks := NewKeyStore()
		testKey := "sk-test-key-12345"

		ks.Set("claude", testKey)
		if err := ks.Save(); err != nil {
			t.Fatalf("error saving keystore: %v", err)
		}

		// Create a new keystore to verify persistence
		ks2 := NewKeyStore()
		if err := ks2.Load(); err != nil {
			t.Fatalf("error loading keystore: %v", err)
		}

		got, err := ks2.Get("claude")
		if err != nil {
			t.Fatalf("unexpected error getting key: %v", err)
		}

		if got != testKey {
			t.Errorf("expected key %q, got %q", testKey, got)
		}
	})

	t.Run("delete key", func(t *testing.T) {
		ks := NewKeyStore()
		ks.Set("claude", "test-key")
		if err := ks.Save(); err != nil {
			t.Fatalf("error saving keystore: %v", err)
		}

		ks.Delete("claude")
		if err := ks.Save(); err != nil {
			t.Fatalf("error saving keystore: %v", err)
		}

		// Verify deletion
		ks2 := NewKeyStore()
		if err := ks2.Load(); err != nil {
			t.Fatalf("error loading keystore: %v", err)
		}

		_, err := ks2.Get("claude")
		if err == nil {
			t.Fatal("expected error for deleted key, got nil")
		}
	})

	t.Run("file permissions", func(t *testing.T) {
		ks := NewKeyStore()
		ks.Set("claude", "test-key")
		if err := ks.Save(); err != nil {
			t.Fatalf("error saving keystore: %v", err)
		}

		info, err := os.Stat(ks.path)
		if err != nil {
			t.Fatalf("error stating keystore file: %v", err)
		}

		perm := info.Mode().Perm()
		if perm != 0o600 {
			t.Errorf("expected permissions 0600, got %o", perm)
		}
	})

	t.Run("environment variable takes precedence", func(t *testing.T) {
		envKey := "sk-env-key-12345"
		os.Setenv(EnvClaudeKey, envKey)
		t.Cleanup(func() {
			os.Unsetenv(EnvClaudeKey)
		})

		ks := NewKeyStore()
		ks.Set("claude", "sk-stored-key")

		got, err := ks.Get("claude")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if got != envKey {
			t.Errorf("expected env key %q, got %q", envKey, got)
		}
	})
}

func TestEnvKeyForProvider(t *testing.T) {
	tests := []struct {
		provider string
		want     string
	}{
		{"claude", EnvClaudeKey},
		{"openai", EnvOpenAIKey},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			got := envKeyForProvider(tt.provider)
			if got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestKeyStoreInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	t.Cleanup(func() {
		os.Unsetenv("XDG_CONFIG_HOME")
	})

	ks := NewKeyStore()

	// Create directory structure
	configDir := filepath.Dir(ks.path)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("error creating config dir: %v", err)
	}

	// Write invalid JSON
	if err := os.WriteFile(ks.path, []byte("{invalid json"), 0o600); err != nil {
		t.Fatalf("error writing invalid JSON: %v", err)
	}

	err := ks.Load()
	if err == nil {
		t.Fatal("expected error loading invalid JSON, got nil")
	}
}
