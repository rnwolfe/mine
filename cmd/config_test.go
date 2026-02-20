package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/rnwolfe/mine/internal/config"
)

// configTestEnv sets up a temp XDG environment and returns a cleanup function.
func configTestEnv(t *testing.T) {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir+"/config")
	t.Setenv("XDG_DATA_HOME", tmpDir+"/data")
	t.Setenv("XDG_CACHE_HOME", tmpDir+"/cache")
	t.Setenv("XDG_STATE_HOME", tmpDir+"/state")
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w
	defer func() {
		os.Stdout = old
		r.Close()
	}()

	fn()

	w.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("io.Copy: %v", err)
	}
	return buf.String()
}

func TestRunConfigGet_KnownKey(t *testing.T) {
	configTestEnv(t)

	cfg := &config.Config{
		AI: config.AIConfig{Provider: "openai", Model: "gpt-4"},
	}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	out := captureStdout(t, func() {
		err := runConfigGet(nil, []string{"ai.provider"})
		if err != nil {
			t.Errorf("runConfigGet: %v", err)
		}
	})

	if !strings.Contains(out, "openai") {
		t.Fatalf("expected 'openai' in output, got: %q", out)
	}
}

func TestRunConfigGet_UnknownKey(t *testing.T) {
	configTestEnv(t)

	err := runConfigGet(nil, []string{"not.a.real.key"})
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
	if !strings.Contains(err.Error(), "unknown config key") {
		t.Errorf("expected 'unknown config key' in error, got: %v", err)
	}
	// Error should include list of valid keys.
	if !strings.Contains(err.Error(), "user.name") {
		t.Errorf("expected valid key hint in error, got: %v", err)
	}
}

func TestRunConfigSet_KnownKey(t *testing.T) {
	configTestEnv(t)

	out := captureStdout(t, func() {
		err := runConfigSet(nil, []string{"user.name", "Bob"})
		if err != nil {
			t.Errorf("runConfigSet: %v", err)
		}
	})
	if !strings.Contains(out, "user.name") {
		t.Errorf("expected key name in output, got: %q", out)
	}

	// Verify persistence.
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.User.Name != "Bob" {
		t.Fatalf("expected User.Name='Bob', got %q", cfg.User.Name)
	}
}

func TestRunConfigSet_UnknownKey(t *testing.T) {
	configTestEnv(t)

	err := runConfigSet(nil, []string{"fake.key", "value"})
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
	if !strings.Contains(err.Error(), "unknown config key") {
		t.Errorf("expected 'unknown config key' error, got: %v", err)
	}
}

func TestRunConfigSet_BoolTypeMismatch(t *testing.T) {
	configTestEnv(t)

	err := runConfigSet(nil, []string{"analytics", "notabool"})
	if err == nil {
		t.Fatal("expected type mismatch error")
	}
}

func TestRunConfigUnset_KnownKey(t *testing.T) {
	configTestEnv(t)

	// First set a value.
	cfg := &config.Config{AI: config.AIConfig{Provider: "openai"}}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	out := captureStdout(t, func() {
		err := runConfigUnset(nil, []string{"ai.provider"})
		if err != nil {
			t.Errorf("runConfigUnset: %v", err)
		}
	})
	if !strings.Contains(out, "ai.provider") {
		t.Errorf("expected key name in output, got: %q", out)
	}

	// Verify value was reset to default.
	loaded, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.AI.Provider != "claude" {
		t.Fatalf("expected AI.Provider='claude' after unset, got %q", loaded.AI.Provider)
	}
}

func TestRunConfigUnset_UnknownKey(t *testing.T) {
	configTestEnv(t)

	err := runConfigUnset(nil, []string{"ghost.key"})
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
	if !strings.Contains(err.Error(), "unknown config key") {
		t.Errorf("expected 'unknown config key' error, got: %v", err)
	}
}

func TestRunConfigList_ShowsKeys(t *testing.T) {
	configTestEnv(t)

	out := captureStdout(t, func() {
		err := runConfigList(nil, nil)
		if err != nil {
			t.Errorf("runConfigList: %v", err)
		}
	})

	for _, key := range []string{"user.name", "ai.provider", "analytics"} {
		if !strings.Contains(out, key) {
			t.Errorf("expected key %q in list output, got:\n%s", key, out)
		}
	}
}

func TestRunConfigPath_PrintsPath(t *testing.T) {
	configTestEnv(t)

	out := captureStdout(t, func() {
		err := runConfigPath(nil, nil)
		if err != nil {
			t.Errorf("runConfigPath: %v", err)
		}
	})

	if !strings.Contains(out, "config.toml") {
		t.Fatalf("expected 'config.toml' in path output, got: %q", out)
	}
}

func TestRunConfigEdit_NoEditor(t *testing.T) {
	configTestEnv(t)
	t.Setenv("EDITOR", "")

	err := runConfigEdit(nil, nil)
	if err == nil {
		t.Fatal("expected error when $EDITOR is not set")
	}
	if !strings.Contains(err.Error(), "$EDITOR") {
		t.Errorf("expected $EDITOR mention in error, got: %v", err)
	}
}

func TestRunConfigSet_BoolKey_ValidValues(t *testing.T) {
	for _, val := range []string{"true", "false", "1", "0", "yes", "no"} {
		t.Run(val, func(t *testing.T) {
			configTestEnv(t)
			err := runConfigSet(nil, []string{"analytics", val})
			if err != nil {
				t.Errorf("runConfigSet analytics=%q: %v", val, err)
			}
		})
	}
}

func TestRunConfigGet_Analytics_DefaultTrue(t *testing.T) {
	configTestEnv(t)
	// No config file — should default to true.

	out := captureStdout(t, func() {
		err := runConfigGet(nil, []string{"analytics"})
		if err != nil {
			t.Errorf("runConfigGet: %v", err)
		}
	})

	if !strings.Contains(out, "true") {
		t.Fatalf("expected 'true' for default analytics, got: %q", out)
	}
}
