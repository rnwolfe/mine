package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
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

var initResetFlag bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Set up mine for the first time",
	Long:  `Initialize mine with your preferences. Creates config and data directories.`,
	RunE:  hook.Wrap("init", runInit),
}

func init() {
	initCmd.Flags().BoolVar(&initResetFlag, "reset", false, "Overwrite config from scratch (replaces existing config)")
}

func runInit(_ *cobra.Command, _ []string) error {
	return runInitWithReader(bufio.NewReader(os.Stdin), initResetFlag)
}

// runInitWithReader is the testable entry point for mine init.
// reset=true triggers the --reset path (full overwrite after confirmation).
func runInitWithReader(reader *bufio.Reader, reset bool) error {
	initialized := config.Initialized()

	// --reset path: warn and confirm before overwriting.
	if reset {
		if initialized {
			fmt.Println()
			fmt.Printf("  %s This will overwrite your current configuration.\n",
				ui.Warning.Render(ui.IconWarn))
			fmt.Println()
			fmt.Printf("  Proceed? %s ", ui.Muted.Render("(y/N)"))
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(strings.ToLower(input))
			fmt.Println()
			if input != "y" && input != "yes" {
				ui.Inf("No changes made.")
				return nil
			}
		}
		return runFreshInit(reader, nil)
	}

	// Re-init path: config already exists.
	if initialized {
		return runReInit(reader)
	}

	// Fresh install path.
	return runFreshInit(reader, nil)
}

// runReInit handles re-running mine init when config already exists.
// It shows the current settings and asks whether to update them.
func runReInit(reader *bufio.Reader) error {
	existing, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	fmt.Println(ui.Title.Render(ui.IconMine + " mine is already set up."))
	fmt.Println()
	fmt.Println("  Your current configuration:")

	name := existing.User.Name
	fmt.Printf("    %-8s %s\n", "Name", ui.Accent.Render(name))

	aiDisplay := existing.AI.Provider
	if aiDisplay != "" && existing.AI.Model != "" {
		aiDisplay = fmt.Sprintf("%s (%s)", existing.AI.Provider, existing.AI.Model)
	}
	if aiDisplay == "" {
		aiDisplay = ui.Muted.Render("not configured")
	}
	fmt.Printf("    %-8s %s\n", "AI", aiDisplay)

	shell := existing.Shell.DefaultShell
	if shell == "" {
		shell = os.Getenv("SHELL")
	}
	fmt.Printf("    %-8s %s\n", "Shell", shell)
	fmt.Println()

	fmt.Printf("  Update your configuration? %s ", ui.Muted.Render("(y/N)"))
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	fmt.Println()

	if input != "y" && input != "yes" {
		ui.Inf("No changes made.")
		return nil
	}

	return runFreshInit(reader, existing)
}

// runFreshInit runs the full interactive init flow.
// When existing is non-nil, prompts are pre-filled with its values and its
// analytics preference is preserved in the saved config (re-init mode).
func runFreshInit(reader *bufio.Reader, existing *config.Config) error {
	isReInit := existing != nil

	if !isReInit {
		fmt.Println(ui.Title.Render(ui.IconMine + " Welcome to mine!"))
		fmt.Println()
		ui.Inf("Let's get you set up. This takes about 30 seconds.")
		fmt.Println()
	}

	// Name prompt: prefer existing name over git/USER guess.
	nameDefault := guessName()
	if existing != nil && existing.User.Name != "" {
		nameDefault = existing.User.Name
	}
	name := prompt(reader, "  What should I call you?", nameDefault)
	fmt.Println()

	// Build config from scratch; we'll carry forward preserved fields below.
	cfg := &config.Config{}
	cfg.User.Name = name

	// Detect shell from environment.
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}
	cfg.Shell.DefaultShell = shell

	// AI setup.
	fmt.Println(ui.Subtitle.Render("  AI Setup (optional)"))
	fmt.Println()
	fmt.Println(ui.Muted.Render("  mine can use AI to help with code review, commit messages, and questions."))
	fmt.Println()

	detectedKeys := detectAIKeys()
	if len(detectedKeys) > 0 {
		ui.Ok(fmt.Sprintf("Detected %d API key(s) in environment:", len(detectedKeys)))
		for provider := range detectedKeys {
			envVar := getEnvVarForProvider(provider)
			fmt.Printf("    %s %s\n", ui.KeyStyle.Render(provider), ui.Muted.Render(fmt.Sprintf("(%s)", envVar)))
		}
		fmt.Println()

		if len(detectedKeys) == 1 {
			for p := range detectedKeys {
				cfg.AI.Provider = p
			}
		} else {
			providerList := make([]string, 0, len(detectedKeys))
			for p := range detectedKeys {
				providerList = append(providerList, p)
			}
			// Pre-select existing provider if it's among detected keys.
			providerDefault := strings.Join(providerList, ", ")
			if existing != nil && detectedKeys[existing.AI.Provider] {
				providerDefault = existing.AI.Provider
			}
			fmt.Printf("  Which provider would you like to use by default? %s ", ui.Muted.Render(fmt.Sprintf("(%s)", providerDefault)))
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(strings.ToLower(input))
			switch {
			case input != "" && detectedKeys[input]:
				cfg.AI.Provider = input
			case existing != nil && detectedKeys[existing.AI.Provider]:
				cfg.AI.Provider = existing.AI.Provider
			default:
				cfg.AI.Provider = providerList[0]
			}
			fmt.Println()
		}

		if cfg.AI.Provider != "" {
			modelDefault := getDefaultModelForProvider(cfg.AI.Provider)
			if existing != nil && existing.AI.Model != "" && existing.AI.Provider == cfg.AI.Provider {
				modelDefault = existing.AI.Model
			}
			modelInput := prompt(reader, "  Default model? (press Enter to skip)", modelDefault)
			if modelInput != "" {
				cfg.AI.Model = modelInput
			}
			fmt.Println()
		}
	} else if isReInit && existing.AI.Provider != "" {
		// Re-init with existing AI config but no env keys: simple update prompts.
		fmt.Println(ui.Muted.Render("  No API keys detected in environment."))
		fmt.Println()
		providerInput := prompt(reader, "  AI provider?", existing.AI.Provider)
		cfg.AI.Provider = providerInput
		if cfg.AI.Provider != "" {
			modelDefault := existing.AI.Model
			if modelDefault == "" {
				modelDefault = getDefaultModelForProvider(cfg.AI.Provider)
			}
			modelInput := prompt(reader, "  Default model?", modelDefault)
			cfg.AI.Model = modelInput
		}
		fmt.Println()
	} else {
		// No API keys detected — offer OpenRouter with free models.
		fmt.Println(ui.Muted.Render("  No API keys detected in environment."))
		fmt.Println()
		fmt.Printf("  Would you like to use OpenRouter for free AI models? %s ", ui.Muted.Render("(y/N, or 's' to skip)"))
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		fmt.Println()

		if input == "y" || input == "yes" {
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

	// Shell integration (idempotent — skips silently if already installed).
	runShellIntegration(reader)

	// Preserve analytics preference from existing config during re-init.
	// Fresh installs default to enabled.
	if existing != nil {
		cfg.Analytics.Enabled = existing.Analytics.Enabled
	} else {
		cfg.Analytics.Enabled = config.BoolPtr(true)
	}

	// Save config.
	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	// Initialize database (idempotent).
	db, err := store.Open()
	if err != nil {
		return fmt.Errorf("initializing database: %w", err)
	}
	db.Close()

	// Generate analytics installation ID (idempotent — returns existing ID if present).
	if _, err := analytics.GetOrCreateID(); err != nil {
		fmt.Println(ui.Muted.Render("  (could not generate analytics ID — will retry later)"))
	}

	paths := config.GetPaths()

	if isReInit {
		if name != "" {
			ui.Ok("Configuration updated, " + name + "!")
		} else {
			ui.Ok("Configuration updated.")
		}
	} else {
		if name != "" {
			ui.Ok("All set, " + name + "! " + ui.IconParty)
		} else {
			ui.Ok("All set! " + ui.IconParty)
		}
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

	// Probe environment for capability table.
	probe := probeEnvironment(cfg)

	// Project registration prompt (only inside a git repo).
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

	// Dynamic capability table.
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
	if name := gitUserName(); name != "" {
		return name
	}
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	return ""
}

func gitUserName() string {
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
