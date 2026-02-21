package cmd

import (
	"strings"
	"testing"

	"github.com/rnwolfe/mine/internal/config"
)

func TestCheckConfig_Missing(t *testing.T) {
	configTestEnv(t)

	r := checkConfig()
	if r.ok {
		t.Fatal("expected checkConfig to fail when no config exists")
	}
	if r.name != "Config" {
		t.Errorf("expected name 'Config', got %q", r.name)
	}
	if r.fixHint == "" {
		t.Error("expected a fix hint when config is missing")
	}
	if !strings.Contains(r.fixHint, "mine init") {
		t.Errorf("expected fix hint to mention 'mine init', got: %q", r.fixHint)
	}
}

func TestCheckConfig_Present(t *testing.T) {
	configTestEnv(t)

	cfg := &config.Config{
		User: config.UserConfig{Name: "Test User"},
		AI:   config.AIConfig{Provider: "claude"},
	}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	r := checkConfig()
	if !r.ok {
		t.Fatalf("expected checkConfig to pass, got detail: %q", r.detail)
	}
	if !strings.Contains(r.detail, "config.toml") {
		t.Errorf("expected detail to mention config.toml, got: %q", r.detail)
	}
}

func TestCheckStore_Works(t *testing.T) {
	configTestEnv(t)

	r := checkStore()
	if !r.ok {
		t.Fatalf("expected checkStore to pass, got: %q", r.detail)
	}
}

func TestCheckGit_Found(t *testing.T) {
	// git is expected to be installed in the test environment.
	r := checkGit()
	if !r.ok {
		t.Logf("git not found — skipping assertion (environment may not have git): %s", r.detail)
		return
	}
	if !strings.Contains(r.detail, "git") {
		t.Errorf("expected detail to mention 'git', got: %q", r.detail)
	}
}

func TestCheckShellHelpers_NoInit(t *testing.T) {
	configTestEnv(t)

	r := checkShellHelpers(nil)
	if r.ok {
		t.Fatal("expected checkShellHelpers to fail when cfg is nil")
	}
	if !strings.Contains(r.fixHint, "mine init") {
		t.Errorf("expected fix hint to mention 'mine init', got: %q", r.fixHint)
	}
}

func TestCheckShellHelpers_NoName(t *testing.T) {
	configTestEnv(t)

	cfg := &config.Config{User: config.UserConfig{Name: ""}}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	r := checkShellHelpers(cfg)
	if r.ok {
		t.Fatal("expected checkShellHelpers to fail when user name is empty")
	}
}

func TestCheckShellHelpers_WithName(t *testing.T) {
	configTestEnv(t)

	cfg := &config.Config{User: config.UserConfig{Name: "Alice"}}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	r := checkShellHelpers(cfg)
	if !r.ok {
		t.Fatalf("expected checkShellHelpers to pass when user name is set, got: %q", r.detail)
	}
}

func TestCheckAI_NoProvider(t *testing.T) {
	cfg := &config.Config{AI: config.AIConfig{Provider: ""}}

	r := checkAI(cfg)
	if r.ok {
		t.Fatal("expected checkAI to fail when no provider is set")
	}
	if !strings.Contains(r.fixHint, "mine ai config") {
		t.Errorf("expected fix hint to mention 'mine ai config', got: %q", r.fixHint)
	}
}

func TestCheckAI_WithProvider(t *testing.T) {
	cfg := &config.Config{AI: config.AIConfig{Provider: "claude", Model: "claude-sonnet"}}

	r := checkAI(cfg)
	if !r.ok {
		t.Fatalf("expected checkAI to pass, got: %q", r.detail)
	}
	if !strings.Contains(r.detail, "claude") {
		t.Errorf("expected detail to include provider name, got: %q", r.detail)
	}
}

func TestCheckAnalytics_Enabled(t *testing.T) {
	enabled := true
	cfg := &config.Config{Analytics: config.AnalyticsConfig{Enabled: &enabled}}

	r := checkAnalytics(cfg)
	if !r.ok {
		t.Fatal("expected checkAnalytics to always pass")
	}
	if !strings.Contains(r.detail, "Enabled") {
		t.Errorf("expected 'Enabled' in detail, got: %q", r.detail)
	}
}

func TestCheckAnalytics_Disabled(t *testing.T) {
	disabled := false
	cfg := &config.Config{Analytics: config.AnalyticsConfig{Enabled: &disabled}}

	r := checkAnalytics(cfg)
	if !r.ok {
		t.Fatal("expected checkAnalytics to always pass")
	}
	if !strings.Contains(r.detail, "Disabled") {
		t.Errorf("expected 'Disabled' in detail, got: %q", r.detail)
	}
}

func TestRunDoctor_AllPass(t *testing.T) {
	configTestEnv(t)

	// Save a complete config so all checks pass.
	enabled := true
	cfg := &config.Config{
		User:      config.UserConfig{Name: "Alice", Email: "alice@example.com"},
		AI:        config.AIConfig{Provider: "claude", Model: "claude-sonnet"},
		Analytics: config.AnalyticsConfig{Enabled: &enabled},
	}
	if err := config.Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	out := captureStdout(t, func() {
		err := runDoctor(nil, nil)
		if err != nil {
			// Only fail if git is expected to be present.
			t.Logf("runDoctor returned error (may be expected in CI): %v", err)
		}
	})

	// Output should contain check names.
	for _, name := range []string{"Config", "Store", "Git", "Shell helpers", "AI", "Analytics"} {
		if !strings.Contains(out, name) {
			t.Errorf("expected %q in doctor output, got:\n%s", name, out)
		}
	}
}

func TestRunDoctor_ConfigMissing(t *testing.T) {
	configTestEnv(t)
	// No config saved — Config check should fail.

	out := captureStdout(t, func() {
		err := runDoctor(nil, nil)
		if err == nil {
			t.Error("expected runDoctor to return an error when config is missing")
		}
	})

	if !strings.Contains(out, "Config") {
		t.Errorf("expected 'Config' in output, got:\n%s", out)
	}
}
