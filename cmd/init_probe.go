package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/rnwolfe/mine/internal/config"
	"github.com/rnwolfe/mine/internal/ui"
)

// envProbe holds detected environment capabilities.
type envProbe struct {
	gitInstalled  bool
	tmuxInstalled bool
	aiConfigured  bool
	aiProvider    string
	inGitRepo     bool
	cwd           string
}

// probeEnvironment detects which mine capabilities are ready to use.
func probeEnvironment(cfg *config.Config) envProbe {
	probe := envProbe{}

	_, err := exec.LookPath("git")
	probe.gitInstalled = err == nil

	_, err = exec.LookPath("tmux")
	probe.tmuxInstalled = err == nil

	if cfg != nil && cfg.AI.Provider != "" {
		probe.aiConfigured = true
		probe.aiProvider = cfg.AI.Provider
	}

	cwd, err := os.Getwd()
	if err == nil {
		probe.cwd = cwd
		_, statErr := os.Stat(filepath.Join(cwd, ".git"))
		probe.inGitRepo = statErr == nil
	}

	return probe
}

// printCapabilityRow prints a single row in the capability table.
// Ready rows show a concrete command example; unready rows show a setup hint.
func printCapabilityRow(feature string, ready bool, readyExample, notReadyHint string) {
	label := fmt.Sprintf("%-14s", feature)
	if ready {
		fmt.Printf("    %s %s — %s\n",
			ui.Success.Render(ui.IconOk),
			ui.KeyStyle.Render(label),
			ui.Accent.Render(readyExample),
		)
	} else {
		fmt.Printf("    %s  %s — %s\n",
			ui.Muted.Render(ui.IconDot),
			ui.Muted.Render(label),
			ui.Muted.Render(notReadyHint),
		)
	}
}

// detectAIKeys checks environment for standard AI provider API keys.
func detectAIKeys() map[string]bool {
	detected := make(map[string]bool)
	envVars := map[string]string{
		"claude":     "ANTHROPIC_API_KEY",
		"openai":     "OPENAI_API_KEY",
		"gemini":     "GEMINI_API_KEY",
		"openrouter": "OPENROUTER_API_KEY",
	}

	for provider, envVar := range envVars {
		if os.Getenv(envVar) != "" {
			detected[provider] = true
		}
	}

	return detected
}

// getEnvVarForProvider returns the env var name for a provider.
func getEnvVarForProvider(provider string) string {
	envVars := map[string]string{
		"claude":     "ANTHROPIC_API_KEY",
		"openai":     "OPENAI_API_KEY",
		"gemini":     "GEMINI_API_KEY",
		"openrouter": "OPENROUTER_API_KEY",
	}
	return envVars[provider]
}

// getDefaultModelForProvider returns a sensible default model for a provider.
func getDefaultModelForProvider(provider string) string {
	defaults := map[string]string{
		"claude":     config.DefaultModel,
		"openai":     "gpt-5.2",
		"gemini":     "gemini-3-flash-preview",
		"openrouter": "z-ai/glm-4.5-air:free",
	}
	return defaults[provider]
}
