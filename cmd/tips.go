package cmd

import (
	"fmt"
	"time"

	"github.com/rnwolfe/mine/internal/tips"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

var tipsShowAll bool

var tipsCmd = &cobra.Command{
	Use:   "tips",
	Short: "Discover what mine can do",
	Long:  `Show actionable tips for getting the most out of mine.`,
	RunE:  runTips,
}

func init() {
	tipsCmd.Flags().BoolVarP(&tipsShowAll, "all", "a", false, "List all tips")
}

func runTips(_ *cobra.Command, _ []string) error {
	if tipsShowAll {
		fmt.Println()
		fmt.Println(ui.Title.Render("  mine tips"))
		fmt.Println()
		for _, tip := range tips.All() {
			fmt.Printf("  %s %s\n", ui.Accent.Render("✦"), ui.Muted.Render(tip))
		}
		fmt.Println()
		return nil
	}

	tip := tips.Daily(time.Now())
	fmt.Println()
	ui.Tip(tip)
	fmt.Println()
	fmt.Printf("  %s\n", ui.Muted.Render("Run `mine tips --all` to see all tips."))
	fmt.Println()
	return nil
}
