package shell

import (
	"strings"
	"testing"
)

func TestValidShell(t *testing.T) {
	tests := []struct {
		name  string
		shell string
		want  bool
	}{
		{"bash valid", "bash", true},
		{"zsh valid", "zsh", true},
		{"fish valid", "fish", true},
		{"unknown invalid", "powershell", false},
		{"empty invalid", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidShell(tt.shell); got != tt.want {
				t.Errorf("ValidShell(%q) = %v, want %v", tt.shell, got, tt.want)
			}
		})
	}
}

func TestShellError(t *testing.T) {
	err := ShellError("powershell")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "powershell") {
		t.Errorf("error should mention the shell name, got: %s", err.Error())
	}
}

func TestFunctions(t *testing.T) {
	funcs := Functions()
	if len(funcs) < 5 {
		t.Errorf("expected at least 5 functions, got %d", len(funcs))
	}

	names := make(map[string]bool)
	for _, fn := range funcs {
		if fn.Name == "" {
			t.Error("function has empty name")
		}
		if fn.Desc == "" {
			t.Errorf("function %q has empty description", fn.Name)
		}
		if fn.Bash == "" {
			t.Errorf("function %q has no bash implementation", fn.Name)
		}
		if fn.Zsh == "" {
			t.Errorf("function %q has no zsh implementation", fn.Name)
		}
		if fn.Fish == "" {
			t.Errorf("function %q has no fish implementation", fn.Name)
		}
		if names[fn.Name] {
			t.Errorf("duplicate function name: %s", fn.Name)
		}
		names[fn.Name] = true
	}

	// Verify expected functions exist.
	expected := []string{"mkcd", "extract", "ports", "gitroot", "serve", "backup", "tre"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected function %q not found", name)
		}
	}
}

func TestFunctionsScript(t *testing.T) {
	shells := []string{Bash, Zsh, Fish}
	for _, sh := range shells {
		t.Run(sh, func(t *testing.T) {
			script, err := FunctionsScript(sh)
			if err != nil {
				t.Fatalf("FunctionsScript(%q) error: %v", sh, err)
			}
			if script == "" {
				t.Fatalf("FunctionsScript(%q) returned empty script", sh)
			}
			// Should contain header comment.
			if !strings.Contains(script, "mine shell functions") {
				t.Error("script missing header comment")
			}
			// Should contain each function name.
			for _, fn := range Functions() {
				if !strings.Contains(script, fn.Name) {
					t.Errorf("script missing function %q", fn.Name)
				}
			}
		})
	}

	// Invalid shell should error.
	_, err := FunctionsScript("powershell")
	if err == nil {
		t.Error("expected error for invalid shell")
	}
}

func TestInitScript(t *testing.T) {
	shells := []string{Bash, Zsh, Fish}
	for _, sh := range shells {
		t.Run(sh, func(t *testing.T) {
			script, err := InitScript(sh)
			if err != nil {
				t.Fatalf("InitScript(%q) error: %v", sh, err)
			}

			// Should contain aliases.
			if !strings.Contains(script, "alias") {
				t.Error("init script missing aliases")
			}

			// Should contain functions.
			if !strings.Contains(script, "mkcd") {
				t.Error("init script missing mkcd function")
			}

			// Should contain prompt integration.
			if !strings.Contains(script, "__mine_prompt") {
				t.Error("init script missing prompt integration")
			}

			// Should contain the header.
			if !strings.Contains(script, "mine shell init") {
				t.Error("init script missing header")
			}
		})
	}

	// Invalid shell should error.
	_, err := InitScript("powershell")
	if err == nil {
		t.Error("expected error for invalid shell")
	}
}

func TestPromptScript(t *testing.T) {
	shells := []string{Bash, Zsh, Fish}
	for _, sh := range shells {
		t.Run(sh, func(t *testing.T) {
			script, err := PromptScript(sh)
			if err != nil {
				t.Fatalf("PromptScript(%q) error: %v", sh, err)
			}
			if script == "" {
				t.Fatalf("PromptScript(%q) returned empty script", sh)
			}
			// Should reference mine status --json.
			if !strings.Contains(script, "mine status --json") {
				t.Error("prompt script should reference mine status --json")
			}
			// Should define the prompt function.
			if !strings.Contains(script, "__mine_prompt") {
				t.Error("prompt script should define __mine_prompt")
			}
		})
	}

	_, err := PromptScript("powershell")
	if err == nil {
		t.Error("expected error for invalid shell")
	}
}

func TestStarshipConfig(t *testing.T) {
	cfg := StarshipConfig()
	if !strings.Contains(cfg, "starship.toml") {
		t.Error("starship config should reference starship.toml")
	}
	if !strings.Contains(cfg, "mine status --prompt") {
		t.Error("starship config should use mine status --prompt")
	}
	if !strings.Contains(cfg, "[custom.mine]") {
		t.Error("starship config should define [custom.mine] section")
	}
}

func TestInitScriptBashAliasFormat(t *testing.T) {
	script, _ := InitScript(Bash)
	// Bash aliases use = without spaces.
	if !strings.Contains(script, "alias m='mine'") {
		t.Error("bash alias should use format: alias m='mine'")
	}
}

func TestInitScriptFishAliasFormat(t *testing.T) {
	script, _ := InitScript(Fish)
	// Fish aliases use space, not =.
	if !strings.Contains(script, "alias m 'mine'") {
		t.Error("fish alias should use format: alias m 'mine'")
	}
}
