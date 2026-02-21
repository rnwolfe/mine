package cmd

import (
	"fmt"

	"github.com/rnwolfe/mine/internal/ui"
	"github.com/rnwolfe/mine/internal/version"
	"github.com/spf13/cobra"
)

var aboutCmd = &cobra.Command{
	Use:   "about",
	Short: "The story, the version, the vibe",
	Long:  "Everything you ever wanted to know about mine — who made it, what it does, and why it exists.",
	Run:   runAbout,
}

func runAbout(_ *cobra.Command, _ []string) {
	fmt.Println()
	fmt.Println(ui.Title.Render("  " + ui.IconGem + " mine"))
	fmt.Println(ui.Muted.Render("  ────────────────────────────────────────────"))
	fmt.Println()
	fmt.Println("  " + ui.Subtitle.Render("The developer CLI that has your back."))
	fmt.Println()
	fmt.Println(ui.Muted.Render("  Todos. Encrypted env profiles. Secrets vault. Dotfile stashing."))
	fmt.Println(ui.Muted.Render("  Git helpers. Tmux sessions. AI code review. All in one binary."))
	fmt.Println()
	ui.Kv("  Version", version.Full())
	ui.Kv("  Repo   ", "https://github.com/rnwolfe/mine")
	ui.Kv("  License", "MIT")
	fmt.Println()
	fmt.Println(ui.Muted.Render("  Built for developers who got tired of juggling twelve different tools."))
	fmt.Println(ui.Muted.Render("  One binary. Zero runtime dependencies. Radically yours."))
	fmt.Println()
	ui.Tip("run `mine help` to explore everything mine can do")
	fmt.Println()
}
