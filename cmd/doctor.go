package cmd

import (
	"fmt"
	"os/exec"

	"github.com/rnwolfe/mine/internal/config"
	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/store"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check your mine setup for problems",
	Long:  `Run a suite of health checks and report what's working (and what isn't).`,
	RunE:  hook.Wrap("doctor", runDoctor),
}

// checkResult holds the outcome of a single health check.
type checkResult struct {
	name    string
	ok      bool
	detail  string
	fixHint string
}

func runDoctor(_ *cobra.Command, _ []string) error {
	cfg, _ := config.Load()

	results := []checkResult{
		checkConfig(),
		checkStore(),
		checkGit(),
		checkShellHelpers(cfg),
		checkAI(cfg),
		checkAnalytics(cfg),
	}

	fmt.Println()

	allPassed := true
	for _, r := range results {
		printCheck(r)
		if !r.ok {
			allPassed = false
		}
	}

	fmt.Println()

	if !allPassed {
		return fmt.Errorf("one or more checks failed — see suggestions above")
	}
	return nil
}

func printCheck(r checkResult) {
	label := fmt.Sprintf("%-16s", r.name)
	if r.ok {
		icon := ui.Success.Render(ui.IconOk)
		fmt.Printf("  %s %s %s\n", icon, ui.KeyStyle.Render(label), ui.Muted.Render(r.detail))
	} else {
		icon := ui.Error.Render(ui.IconError)
		fmt.Printf("  %s %s %s\n", icon, ui.KeyStyle.Render(label), r.detail)
		if r.fixHint != "" {
			fmt.Printf("  %s %s %s\n", "  ", "                ", ui.Muted.Render("→ "+r.fixHint))
		}
	}
}

func checkConfig() checkResult {
	if !config.Initialized() {
		return checkResult{
			name:    "Config",
			ok:      false,
			detail:  "config file not found",
			fixHint: fmt.Sprintf("Run %s to create it", ui.Accent.Render("mine init")),
		}
	}
	paths := config.GetPaths()
	_, err := config.Load()
	if err != nil {
		return checkResult{
			name:    "Config",
			ok:      false,
			detail:  fmt.Sprintf("parse error: %v", err),
			fixHint: fmt.Sprintf("Check %s for syntax errors", paths.ConfigFile),
		}
	}
	return checkResult{
		name:   "Config",
		ok:     true,
		detail: paths.ConfigFile + " found and valid",
	}
}

func checkStore() checkResult {
	db, err := store.Open()
	if err != nil {
		return checkResult{
			name:    "Store",
			ok:      false,
			detail:  fmt.Sprintf("cannot open database: %v", err),
			fixHint: fmt.Sprintf("Try re-running %s or check available disk space", ui.Accent.Render("mine init")),
		}
	}
	db.Close()
	return checkResult{
		name:   "Store",
		ok:     true,
		detail: "SQLite database opens and responds",
	}
}

func checkGit() checkResult {
	path, err := exec.LookPath("git")
	if err != nil {
		return checkResult{
			name:    "Git",
			ok:      false,
			detail:  "git not found in PATH",
			fixHint: "Install git from https://git-scm.com or your package manager",
		}
	}
	out, err := exec.Command("git", "--version").Output()
	if err != nil {
		return checkResult{
			name:    "Git",
			ok:      true,
			detail:  path + " found",
		}
	}
	// Trim trailing newline from version string.
	version := string(out)
	for len(version) > 0 && (version[len(version)-1] == '\n' || version[len(version)-1] == '\r') {
		version = version[:len(version)-1]
	}
	return checkResult{
		name:   "Git",
		ok:     true,
		detail: version + " found in PATH",
	}
}

func checkShellHelpers(cfg *config.Config) checkResult {
	if cfg == nil || !config.Initialized() {
		return checkResult{
			name:    "Shell helpers",
			ok:      false,
			detail:  "shell integration not detected",
			fixHint: fmt.Sprintf("Run %s to install shell helpers (p, pp, menv)", ui.Accent.Render("mine init")),
		}
	}
	// Shell helpers are installed as part of mine init and require
	// the user name to have been set (indicates init completed).
	if cfg.User.Name == "" {
		return checkResult{
			name:    "Shell helpers",
			ok:      false,
			detail:  "shell integration not detected",
			fixHint: fmt.Sprintf("Run %s to install shell helpers (p, pp, menv)", ui.Accent.Render("mine init")),
		}
	}
	return checkResult{
		name:   "Shell helpers",
		ok:     true,
		detail: "shell integration detected (p, pp, menv available)",
	}
}

func checkAI(cfg *config.Config) checkResult {
	if cfg == nil || cfg.AI.Provider == "" {
		return checkResult{
			name:    "AI",
			ok:      false,
			detail:  "no AI provider configured",
			fixHint: fmt.Sprintf("Run %s to configure an AI provider", ui.Accent.Render("mine ai config")),
		}
	}
	detail := "Provider: " + cfg.AI.Provider
	if cfg.AI.Model != "" {
		detail += " (" + cfg.AI.Model + ")"
	}
	return checkResult{
		name:   "AI",
		ok:     true,
		detail: detail,
	}
}

func checkAnalytics(cfg *config.Config) checkResult {
	if cfg == nil {
		return checkResult{
			name:   "Analytics",
			ok:     true,
			detail: "status unknown",
		}
	}
	if cfg.Analytics.IsEnabled() {
		return checkResult{
			name:   "Analytics",
			ok:     true,
			detail: fmt.Sprintf("Enabled (opt out: %s)", ui.Accent.Render("mine config set analytics false")),
		}
	}
	return checkResult{
		name:   "Analytics",
		ok:     true,
		detail: "Disabled",
	}
}
