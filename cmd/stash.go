package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rnwolfe/mine/internal/config"
	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

var stashCmd = &cobra.Command{
	Use:   "stash",
	Short: "Manage your dotfiles and environment",
	Long:  `Track, backup, and sync your dotfiles. Your environment, version controlled.`,
	RunE:  hook.Wrap("stash", runStashStatus),
}

func init() {
	rootCmd.AddCommand(stashCmd)
	stashCmd.AddCommand(stashTrackCmd)
	stashCmd.AddCommand(stashListCmd)
	stashCmd.AddCommand(stashInitCmd)
	stashCmd.AddCommand(stashDiffCmd)
}

var stashInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize dotfile tracking",
	RunE:  hook.Wrap("stash.init", runStashInit),
}

var stashTrackCmd = &cobra.Command{
	Use:   "track <file>",
	Short: "Start tracking a dotfile",
	Args:  cobra.ExactArgs(1),
	RunE:  hook.Wrap("stash.track", runStashTrack),
}

var stashListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tracked dotfiles",
	RunE:  hook.Wrap("stash.list", runStashList),
}

var stashDiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show changes in tracked dotfiles",
	RunE:  hook.Wrap("stash.diff", runStashDiff),
}

func stashDir() string {
	return filepath.Join(config.GetPaths().DataDir, "stash")
}

func runStashInit(_ *cobra.Command, _ []string) error {
	dir := stashDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// Initialize as git repo for version tracking
	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		// Create a simple manifest file
		manifest := filepath.Join(dir, ".mine-stash")
		if err := os.WriteFile(manifest, []byte("# mine stash manifest\n# each line: source_path\n"), 0o644); err != nil {
			return err
		}
		ui.Ok("Stash initialized at " + dir)
	} else {
		ui.Ok("Stash already initialized")
	}

	fmt.Println()
	fmt.Printf("  Track a file: %s\n", ui.Accent.Render("mine stash track ~/.zshrc"))
	fmt.Println()
	return nil
}

func runStashTrack(_ *cobra.Command, args []string) error {
	source := args[0]

	// Expand ~ to home dir
	if strings.HasPrefix(source, "~") {
		home, _ := os.UserHomeDir()
		source = filepath.Join(home, source[1:])
	}

	// Make absolute
	source, err := filepath.Abs(source)
	if err != nil {
		return err
	}

	// Check file exists
	info, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("can't find %s", source)
	}
	if info.IsDir() {
		return fmt.Errorf("can't track directories yet (coming soon)")
	}

	dir := stashDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// Create a safe filename from the path
	home, _ := os.UserHomeDir()
	relPath := strings.TrimPrefix(source, home+"/")
	safeName := strings.ReplaceAll(relPath, "/", "__")

	dest := filepath.Join(dir, safeName)

	// Copy file to stash
	data, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("reading %s: %w", source, err)
	}
	if err := os.WriteFile(dest, data, info.Mode()); err != nil {
		return fmt.Errorf("writing to stash: %w", err)
	}

	// Update manifest
	manifestPath := filepath.Join(dir, ".mine-stash")
	manifest, _ := os.ReadFile(manifestPath)
	entry := source + " -> " + safeName + "\n"
	if !strings.Contains(string(manifest), source) {
		f, err := os.OpenFile(manifestPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		defer f.Close()
		f.WriteString(entry)
	}

	ui.Ok(fmt.Sprintf("Tracking %s", relPath))
	fmt.Printf("  Stashed to: %s\n", ui.Muted.Render(dest))
	fmt.Println()
	return nil
}

func runStashList(_ *cobra.Command, _ []string) error {
	dir := stashDir()
	manifestPath := filepath.Join(dir, ".mine-stash")

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println()
			fmt.Println(ui.Muted.Render("  No stash yet."))
			fmt.Printf("  Run %s first.\n", ui.Accent.Render("mine stash init"))
			fmt.Println()
			return nil
		}
		return err
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	tracked := 0

	fmt.Println()
	for _, line := range lines {
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		parts := strings.SplitN(line, " -> ", 2)
		if len(parts) == 2 {
			tracked++
			home, _ := os.UserHomeDir()
			display := strings.Replace(parts[0], home, "~", 1)
			fmt.Printf("  %s %s\n", ui.Success.Render("●"), display)
		}
	}

	if tracked == 0 {
		fmt.Println(ui.Muted.Render("  No files tracked yet."))
		fmt.Printf("  Try: %s\n", ui.Accent.Render("mine stash track ~/.zshrc"))
	} else {
		fmt.Println()
		fmt.Println(ui.Muted.Render(fmt.Sprintf("  %d files tracked", tracked)))
	}
	fmt.Println()
	return nil
}

func runStashDiff(_ *cobra.Command, _ []string) error {
	dir := stashDir()
	manifestPath := filepath.Join(dir, ".mine-stash")

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("no stash found — run `mine stash init` first")
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	changes := 0

	fmt.Println()
	for _, line := range lines {
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		parts := strings.SplitN(line, " -> ", 2)
		if len(parts) != 2 {
			continue
		}

		source := parts[0]
		stashed := filepath.Join(dir, parts[1])

		sourceData, err := os.ReadFile(source)
		if err != nil {
			fmt.Printf("  %s %s (missing!)\n", ui.Error.Render("✗"), source)
			changes++
			continue
		}

		stashedData, err := os.ReadFile(stashed)
		if err != nil {
			continue
		}

		if string(sourceData) != string(stashedData) {
			home, _ := os.UserHomeDir()
			display := strings.Replace(source, home, "~", 1)
			fmt.Printf("  %s %s (modified)\n", ui.Warning.Render("~"), display)
			changes++
		}
	}

	if changes == 0 {
		fmt.Println(ui.Success.Render("  Everything in sync."))
	} else {
		fmt.Println()
		fmt.Println(ui.Muted.Render(fmt.Sprintf("  %d files changed since last stash", changes)))
	}
	fmt.Println()
	return nil
}

func runStashStatus(_ *cobra.Command, _ []string) error {
	return runStashList(nil, nil)
}
