package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/rnwolfe/mine/internal/config"
	"github.com/rnwolfe/mine/internal/store"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Set up mine for the first time",
	Long:  `Initialize mine with your preferences. Creates config and data directories.`,
	RunE:  runInit,
}

func runInit(_ *cobra.Command, _ []string) error {
	fmt.Println(ui.Title.Render("⛏  Welcome to mine!"))
	fmt.Println()
	fmt.Println("  Let's get you set up. This takes about 30 seconds.")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)

	// Name
	name := prompt(reader, "  What should I call you?", guessName())
	fmt.Println()

	// Create config
	cfg := &config.Config{}
	cfg.User.Name = name
	cfg.Shell.DefaultShell = config.GetPaths().ConfigDir // will fix below
	cfg.AI.Provider = "claude"
	cfg.AI.Model = "claude-sonnet-4-5-20250929"

	// Detect shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/bash"
	}
	cfg.Shell.DefaultShell = shell

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

	paths := config.GetPaths()

	fmt.Println(ui.Success.Render("  ✓ All set!"))
	fmt.Println()
	fmt.Println(ui.Muted.Render("  Created:"))
	fmt.Printf("    Config  %s\n", ui.Muted.Render(paths.ConfigFile))
	fmt.Printf("    Data    %s\n", ui.Muted.Render(paths.DBFile))
	fmt.Println()
	fmt.Printf("  Hey %s — you're ready to go. Type %s to see your dashboard.\n",
		ui.Accent.Render(name),
		ui.Accent.Render("mine"),
	)
	fmt.Println()
	fmt.Println(ui.Muted.Render("  Some things to try:"))
	fmt.Printf("    %s  %s\n", ui.Accent.Render("mine todo add \"ship feature X\""), ui.Muted.Render("— capture a task"))
	fmt.Printf("    %s                        %s\n", ui.Accent.Render("mine todo"), ui.Muted.Render("— see your tasks"))
	fmt.Printf("    %s                      %s\n", ui.Accent.Render("mine config"), ui.Muted.Render("— tweak settings"))
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
