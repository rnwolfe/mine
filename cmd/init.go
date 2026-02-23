package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rnwolfe/mine/internal/analytics"
	"github.com/rnwolfe/mine/internal/config"
	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/proj"
	"github.com/rnwolfe/mine/internal/store"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/rnwolfe/mine/internal/vault"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Set up mine for the first time",
	Long:  `Initialize mine with your preferences. Creates config and data directories.`,
	RunE:  hook.Wrap("init", runInit),
}

func runInit(_ *cobra.Command, _ []string) error {
	return runInitWithReader(bufio.NewReader(os.Stdin))
}

func runInitWithReader(reader *bufio.Reader) error {
	fmt.Println(ui.Title.Render(ui.IconMine + " Welcome to mine!"))
	fmt.Println()
	ui.Inf("Let's get you set up. This takes about 30 seconds.")
	fmt.Println()

	// Name
	name := prompt(reader, "  What should I call you?", guessName())
	fmt.Println()

	// Create config
	cfg := &config.Config{}
	cfg.User.Name = name
	cfg.Shell.DefaultShell = config.GetPaths().ConfigDir // will fix below

	// Detect shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}
	cfg.Shell.DefaultShell = shell

	// AI setup
	fmt.Println(ui.Subtitle.Render("  AI Setup (optional)"))
	fmt.Println()
	fmt.Println(ui.Muted.Render("  mine can use AI to help with code review, commit messages, and questions."))
	fmt.Println()

	// Detect available API keys from environment
	detectedKeys := detectAIKeys()
	if len(detectedKeys) > 0 {
		ui.Ok(fmt.Sprintf("Detected %d API key(s) in environment:", len(detectedKeys)))
		for provider := range detectedKeys {
			envVar := getEnvVarForProvider(provider)
			fmt.Printf("    %s %s\n", ui.KeyStyle.Render(provider), ui.Muted.Render(fmt.Sprintf("(%s)", envVar)))
		}
		fmt.Println()

		// Ask which provider to use as default
		defaultProvider := ""
		if len(detectedKeys) == 1 {
			// Only one provider, use it as default
			for p := range detectedKeys {
				defaultProvider = p
			}
			cfg.AI.Provider = defaultProvider
		} else {
			// Multiple providers, ask which to use
			providerList := make([]string, 0, len(detectedKeys))
			for p := range detectedKeys {
				providerList = append(providerList, p)
			}
			fmt.Printf("  Which provider would you like to use by default? %s ", ui.Muted.Render(fmt.Sprintf("(%s)", strings.Join(providerList, ", "))))
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(strings.ToLower(input))
			if input != "" && detectedKeys[input] {
				cfg.AI.Provider = input
			} else if len(providerList) > 0 {
				cfg.AI.Provider = providerList[0] // Default to first
			}
			fmt.Println()
		}

		// Ask for default model
		if cfg.AI.Provider != "" {
			defaultModel := getDefaultModelForProvider(cfg.AI.Provider)
			modelInput := prompt(reader, "  Default model? (press Enter to skip)", defaultModel)
			if modelInput != "" {
				cfg.AI.Model = modelInput
			}
			fmt.Println()
		}
	} else {
		// No API keys detected, offer OpenRouter with free models
		fmt.Println(ui.Muted.Render("  No API keys detected in environment."))
		fmt.Println()
		fmt.Printf("  Would you like to use OpenRouter for free AI models? %s ", ui.Muted.Render("(y/N, or 's' to skip)"))
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		fmt.Println()

		if input == "y" || input == "yes" {
			// Guide user through getting OpenRouter API key
			fmt.Println(ui.Muted.Render("  OpenRouter provides access to free AI models, but requires an API key."))
			fmt.Println()
			fmt.Println(ui.Muted.Render("  Steps to get your free API key:"))
			fmt.Printf("    1. Visit: %s\n", ui.Accent.Render("https://openrouter.ai/keys"))
			fmt.Println(ui.Muted.Render("    2. Sign up (free, no credit card required)"))
			fmt.Println(ui.Muted.Render("    3. Copy your API key"))
			fmt.Println()
			fmt.Printf("  Paste your OpenRouter API key (or press Enter to skip): ")
			keyInput, _ := reader.ReadString('\n')
			keyInput = strings.TrimSpace(keyInput)
			fmt.Println()

			if keyInput != "" {
				// Store the API key in vault.
				passphrase, err := readPassphrase(false)
				if err != nil {
					ui.Warn(fmt.Sprintf("Could not read vault passphrase: %v", err))
					ui.Tip("set your key later with: mine ai config --provider openrouter --key <your-key>")
					fmt.Println()
				} else {
					v := vault.New(passphrase)
					if err := v.Set(aiVaultKey("openrouter"), keyInput); err != nil {
						ui.Warn(fmt.Sprintf("Could not save API key to vault: %v", err))
						ui.Tip("set your key later with: mine ai config --provider openrouter --key <your-key>")
						fmt.Println()
					} else {
						cfg.AI.Provider = "openrouter"
						cfg.AI.Model = "z-ai/glm-4.5-air:free"
						ui.Ok("OpenRouter API key saved and configured")
						fmt.Println(ui.Muted.Render("    Using free model: z-ai/glm-4.5-air:free"))
						fmt.Println()
					}
				}
			} else {
				fmt.Println(ui.Muted.Render("  Skipped. You can configure AI later with:"))
				fmt.Printf("    %s\n", ui.Accent.Render("mine ai config --provider openrouter --key <your-key>"))
				fmt.Println()
			}
		} else {
			fmt.Println(ui.Muted.Render("  You can configure AI later with:"))
			fmt.Printf("    %s\n", ui.Accent.Render("mine ai config --provider claude --key sk-..."))
			fmt.Printf("    %s\n", ui.Muted.Render("Or visit https://openrouter.ai/keys for a free OpenRouter key"))
			fmt.Println()
		}
	}

	// Set analytics defaults (enabled by default, opt-out)
	cfg.Analytics.Enabled = config.BoolPtr(true)

	// Save config
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	// Initialize database
	db, err := store.Open()
	if err != nil {
		return fmt.Errorf("initializing database: %w", err)
	}
	db.Close()

	// Generate analytics installation ID
	if _, err := analytics.GetOrCreateID(); err != nil {
		// Non-fatal — analytics can generate the ID later
		fmt.Println(ui.Muted.Render("  (could not generate analytics ID — will retry later)"))
	}

	paths := config.GetPaths()

	if name != "" {
		ui.Ok("All set, " + name + "! " + ui.IconParty)
	} else {
		ui.Ok("All set! " + ui.IconParty)
	}
	fmt.Println()
	fmt.Println(ui.Muted.Render("  Created:"))
	fmt.Printf("    Config  %s\n", ui.Muted.Render(paths.ConfigFile))
	fmt.Printf("    Data    %s\n", ui.Muted.Render(paths.DBFile))
	fmt.Println()
	if name != "" {
		fmt.Printf("  Hey %s — you're ready to go. Type %s to see your dashboard.\n",
			ui.Accent.Render(name),
			ui.Accent.Render("mine"),
		)
	} else {
		fmt.Printf("  You're ready to go. Type %s to see your dashboard.\n",
			ui.Accent.Render("mine"),
		)
	}
	fmt.Println()

	// Probe environment for capability table
	probe := probeEnvironment(cfg)

	// Project registration prompt (only inside a git repo)
	projRegistered := false
	if probe.inGitRepo && probe.cwd != "" {
		fmt.Printf("  Register %s as a mine project? %s ",
			ui.Accent.Render(probe.cwd),
			ui.Muted.Render("(Y/n)"),
		)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		fmt.Println()

		if input == "" || input == "y" || input == "yes" {
			db2, err := store.Open()
			if err != nil {
				fmt.Printf("  %s Could not open store for project registration: %v\n",
					ui.Warning.Render(ui.IconWarn), err)
			} else {
				defer db2.Close()
				ps := proj.NewStore(db2.Conn())
				p, err := ps.Add(probe.cwd)
				switch {
				case err == nil:
					fmt.Printf("  %s Registered project %s\n",
						ui.Success.Render(ui.IconOk),
						ui.Accent.Render(p.Name),
					)
					projRegistered = true
				case errors.Is(err, proj.ErrProjectExists):
					projRegistered = true
				default:
					fmt.Printf("  %s Could not register project: %v\n",
						ui.Warning.Render(ui.IconWarn), err)
				}
			}
			fmt.Println()
		}
	}

	// Dynamic capability table
	fmt.Println(ui.Muted.Render("  What you've got:"))
	fmt.Println()
	printCapabilityRow("todos", true,
		`mine todo add "ship it"`, "")
	printCapabilityRow("stash", true,
		"mine stash add <url>", "")
	printCapabilityRow("env", true,
		"mine env init", "")
	printCapabilityRow("git", probe.gitInstalled,
		"mine git log",
		"install git, then mine git log")
	printCapabilityRow("tmux", probe.tmuxInstalled,
		"mine tmux new",
		"install tmux, then mine tmux new")
	aiLabel := "AI"
	if probe.aiProvider != "" {
		aiLabel = "AI (" + probe.aiProvider + ")"
	}
	printCapabilityRow(aiLabel, probe.aiConfigured,
		`mine ai ask "explain this diff"`,
		"mine ai config --provider claude --key sk-...")
	printCapabilityRow("proj", projRegistered,
		"mine proj list",
		"mine proj add <path>")
	fmt.Println()

	return nil
}

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

// detectAIKeys checks environment for standard AI provider API keys
func detectAIKeys() map[string]bool {
	detected := make(map[string]bool)
	envVars := map[string]string{
		"claude":  "ANTHROPIC_API_KEY",
		"openai":  "OPENAI_API_KEY",
		"gemini":  "GEMINI_API_KEY",
	}

	for provider, envVar := range envVars {
		if os.Getenv(envVar) != "" {
			detected[provider] = true
		}
	}

	return detected
}

// getEnvVarForProvider returns the env var name for a provider
func getEnvVarForProvider(provider string) string {
	envVars := map[string]string{
		"claude":     "ANTHROPIC_API_KEY",
		"openai":     "OPENAI_API_KEY",
		"gemini":     "GEMINI_API_KEY",
		"openrouter": "OPENROUTER_API_KEY",
	}
	return envVars[provider]
}

// getDefaultModelForProvider returns a sensible default model for a provider
func getDefaultModelForProvider(provider string) string {
	defaults := map[string]string{
		"claude":     "claude-sonnet-4-5-20250929",
		"openai":     "gpt-5.2",
		"gemini":     "gemini-3-flash-preview",
		"openrouter": "z-ai/glm-4.5-air:free",
	}
	return defaults[provider]
}

func prompt(reader *bufio.Reader, question, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("%s %s ", question, ui.Muted.Render(fmt.Sprintf("(%s)", defaultVal)))
	} else {
		fmt.Printf("%s ", question)
	}

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

func guessName() string {
	// Try git config first
	if name := gitUserName(); name != "" {
		return name
	}
	// Fall back to OS user
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	return ""
}

func gitUserName() string {
	// Simple: read git config for user.name
	// We'll keep this lightweight — no exec, just parse the file
	home, _ := os.UserHomeDir()
	data, err := os.ReadFile(home + "/.gitconfig")
	if err != nil {
		return ""
	}

	inUser := false
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "[user]" {
			inUser = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			inUser = false
			continue
		}
		if inUser && strings.HasPrefix(line, "name") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				return strings.Trim(strings.TrimSpace(parts[1]), `"`)
			}
		}
	}
	return ""
}
