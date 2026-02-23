package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rnwolfe/mine/internal/ui"
)

// shellIntegrationSnippet is the exact content appended to bash/zsh RC files.
const shellIntegrationSnippet = "\n# added by mine\neval \"$(mine shell init)\"\n"

// fishIntegrationSnippet is the fish-compatible equivalent (fish does not support $(...) syntax).
const fishIntegrationSnippet = "\n# added by mine\nmine shell init | source\n"

// rcFileForShell returns the RC file path for a given shell binary path or name.
// Returns "" for unrecognized shells.
func rcFileForShell(shell string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	switch {
	case strings.Contains(shell, "zsh"):
		return filepath.Join(home, ".zshrc")
	case strings.Contains(shell, "bash"):
		rc := filepath.Join(home, ".bashrc")
		if _, err := os.Stat(rc); err == nil {
			return rc
		}
		return filepath.Join(home, ".bash_profile")
	case strings.Contains(shell, "fish"):
		return filepath.Join(home, ".config", "fish", "config.fish")
	default:
		return ""
	}
}

// alreadyInstalled reports whether rcPath already contains the mine shell init eval line.
func alreadyInstalled(rcPath string) bool {
	data, err := os.ReadFile(rcPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "mine shell init")
}

// appendToRC appends snippet to the file at rcPath, creating it (and any parent
// directories) if necessary.
func appendToRC(rcPath, snippet string) error {
	if err := os.MkdirAll(filepath.Dir(rcPath), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(rcPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(snippet)
	return err
}

// runShellIntegration runs the interactive shell RC-file setup section of mine init.
func runShellIntegration(reader *bufio.Reader) {
	shell := os.Getenv("SHELL")
	rcPath := rcFileForShell(shell)

	// Silently skip if already installed.
	if rcPath != "" && alreadyInstalled(rcPath) {
		return
	}

	fmt.Println(ui.Subtitle.Render("  Shell Integration"))
	fmt.Println()

	// Pick the shell-appropriate eval line and RC snippet.
	isFish := strings.Contains(shell, "fish")
	var evalLine, snippet string
	if isFish {
		evalLine = "mine shell init | source"
		snippet = fishIntegrationSnippet
	} else {
		evalLine = `eval "$(mine shell init)"`
		snippet = shellIntegrationSnippet
	}

	if rcPath == "" {
		// Unrecognized shell — print fallback instructions.
		fmt.Println(ui.Muted.Render("  Unrecognized shell. Add this line to your shell config manually:"))
		fmt.Println()
		fmt.Printf("    %s\n", ui.Accent.Render(evalLine))
		fmt.Println()
		fmt.Println(ui.Muted.Render("  This enables p, pp, and menv in your shell."))
		fmt.Println()
		return
	}

	fmt.Printf(ui.Muted.Render("  Adding this line to %s enables p, pp, and menv:\n"), rcPath)
	fmt.Println()
	fmt.Printf("    %s\n", ui.Accent.Render(evalLine))
	fmt.Println()
	fmt.Printf("  Add it now? %s ", ui.Muted.Render("(Y/n)"))

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))
	fmt.Println()

	if input == "n" || input == "no" {
		fmt.Println(ui.Muted.Render("  No problem. Add this line to your shell config manually:"))
		fmt.Println()
		fmt.Printf("    %s\n", ui.Accent.Render(evalLine))
		fmt.Println()
		fmt.Printf("  Then restart your shell or run: %s\n", ui.Accent.Render("source "+rcPath))
		fmt.Println()
		return
	}

	// User said yes (or pressed Enter for default Y).
	if err := appendToRC(rcPath, snippet); err != nil {
		// Non-fatal: fall back to manual instructions.
		fmt.Println(ui.Muted.Render("  Could not write to " + rcPath + ". Add this line manually:"))
		fmt.Println()
		fmt.Printf("    %s\n", ui.Accent.Render(evalLine))
		fmt.Println()
		return
	}

	ui.Ok("Added to " + rcPath + ". Restart your shell or run: " + ui.Accent.Render("source "+rcPath))
	fmt.Println()
}
