package cmd

import (
	"fmt"
	"strings"

	"github.com/rnwolfe/mine/internal/agents"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

func runAgentsDiff(_ *cobra.Command, _ []string) error {
	if !agents.IsInitialized() {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No agents store yet."))
		fmt.Printf("  Run %s first.\n", ui.Accent.Render("mine agents init"))
		fmt.Println()
		return nil
	}

	opts := agents.DiffOptions{
		Agent: agentsDiffAgent,
	}

	entries, err := agents.Diff(opts)
	if err != nil {
		return err
	}

	fmt.Println()

	if len(entries) == 0 {
		fmt.Println(ui.Muted.Render("  No links to diff."))
		fmt.Printf("  Run %s to create links first.\n", ui.Accent.Render("mine agents link"))
		fmt.Println()
		return nil
	}

	diffCount := 0
	for _, e := range entries {
		printDiffEntry(e)
		if len(e.Lines) > 0 {
			diffCount++
		}
	}

	fmt.Println()
	if diffCount == 0 {
		ui.Ok("All links match canonical store — nothing to sync")
	} else {
		fmt.Printf("  %s %d link(s) differ from canonical store\n",
			ui.Warning.Render(ui.IconWarn), diffCount)
		fmt.Printf("  Run %s to restore symlinks.\n", ui.Accent.Render("mine agents link --force"))
	}
	fmt.Println()
	return nil
}

// printDiffEntry prints the diff output for a single link entry.
func printDiffEntry(e agents.DiffEntry) {
	switch e.State {
	case agents.LinkHealthLinked:
		fmt.Printf("  %s %s %s %s\n",
			ui.Success.Render(ui.IconOk),
			e.Link.Source,
			ui.Muted.Render(ui.IconArrow),
			ui.Muted.Render(e.Link.Target+" (linked, no diff)"))
	case agents.LinkHealthBroken, agents.LinkHealthUnlinked:
		fmt.Printf("  %s %s %s %s\n",
			ui.Muted.Render("○ "),
			e.Link.Source,
			ui.Muted.Render(ui.IconArrow),
			ui.Muted.Render(e.Link.Target))
		if e.Err != nil {
			fmt.Printf("      %s\n", ui.Muted.Render(e.Err.Error()))
		}
	case agents.LinkHealthDiverged, agents.LinkHealthReplaced:
		stateLabel := "diverged"
		prefixSymbol := "~ "
		if e.State == agents.LinkHealthReplaced {
			stateLabel = "replaced"
			prefixSymbol = "! "
		}
		fmt.Printf("  %s %s %s %s\n",
			ui.Warning.Render(prefixSymbol),
			e.Link.Source,
			ui.Muted.Render(ui.IconArrow),
			ui.Warning.Render(e.Link.Target+" ("+stateLabel+")"))
		if e.Err != nil {
			fmt.Printf("      %s\n", ui.Warning.Render(e.Err.Error()))
		} else if len(e.Lines) == 0 {
			fmt.Printf("      %s\n", ui.Muted.Render("(no textual diff available)"))
		} else {
			for _, line := range e.Lines {
				fmt.Printf("      %s\n", formatDiffLine(line))
			}
		}
	}
}

// formatDiffLine applies color to a single diff line.
func formatDiffLine(line string) string {
	if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
		return ui.Success.Render(line)
	}
	if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
		return ui.Error.Render(line)
	}
	if strings.HasPrefix(line, "@@") {
		return ui.Info.Render(line)
	}
	return ui.Muted.Render(line)
}
