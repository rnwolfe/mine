package cmd

import (
	"fmt"

	"github.com/rnwolfe/mine/internal/ui"
	"github.com/rnwolfe/mine/internal/version"
	"github.com/spf13/cobra"
)

var aboutCmd = &cobra.Command{
	Use:   "about",
	Short: "Show project information",
	Long:  "Display version, repository URL, and license information for mine.",
	Run:   runAbout,
}

func runAbout(_ *cobra.Command, _ []string) {
	fmt.Println()
	ui.Header("About mine")
	fmt.Println()

	ui.Kv("  Version", version.Full())
	ui.Kv("  Repository", "https://github.com/rnwolfe/mine")
	ui.Kv("  License", "MIT")
	fmt.Println()
}
