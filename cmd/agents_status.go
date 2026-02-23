package cmd

import (
	"fmt"

	"github.com/rnwolfe/mine/internal/agents"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

func runAgentsStatus(_ *cobra.Command, _ []string) error {
	if !agents.IsInitialized() {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No agents store yet."))
		fmt.Printf("  Run %s to get started.\n", ui.Accent.Render("mine agents init"))
		fmt.Println()
		return nil
	}

	result, err := agents.CheckStatus()
	if err != nil {
		return fmt.Errorf("checking status: %w", err)
	}

	fmt.Println()

	// Store health.
	storeDesc := result.Store.Dir
	if result.Store.CommitCount > 0 {
		storeDesc = fmt.Sprintf("%s (%d commit(s))", result.Store.Dir, result.Store.CommitCount)
	}
	ui.Kv(ui.IconTools+" Store", storeDesc)

	// Sync state — only shown when a remote is configured.
	if result.Store.RemoteURL != "" {
		remoteDesc := result.Store.RemoteURL
		if result.Store.UnpushedCommits > 0 {
			remoteDesc = fmt.Sprintf("%s (%d unpushed)", result.Store.RemoteURL, result.Store.UnpushedCommits)
		}
		ui.Kv("  Remote", remoteDesc)
	}
	if result.Store.UncommittedFiles > 0 {
		ui.Kv("  Changes", ui.Warning.Render(fmt.Sprintf("%d uncommitted file(s)", result.Store.UncommittedFiles)))
	}

	fmt.Println()

	// Detected agents.
	fmt.Printf("  %s\n", ui.KeyStyle.Render("Detected Agents"))
	if len(result.Agents) == 0 {
		fmt.Println(ui.Muted.Render("    No agents registered yet."))
		fmt.Printf("    Run %s to scan for installed agents.\n", ui.Accent.Render("mine agents detect"))
	} else {
		detectedCount := 0
		for _, a := range result.Agents {
			printStatusAgentRow(a)
			if a.Detected {
				detectedCount++
			}
		}
		fmt.Println()
		ui.Kv("  Summary", fmt.Sprintf("%d registered, %d detected", len(result.Agents), detectedCount))
	}

	fmt.Println()

	// Link health.
	fmt.Printf("  %s\n", ui.KeyStyle.Render("Links"))
	if len(result.Links) == 0 {
		fmt.Println(ui.Muted.Render("    No links configured yet."))
		fmt.Printf("    Run %s to create links.\n", ui.Accent.Render("mine agents link"))
	} else {
		for _, lh := range result.Links {
			printLinkHealthRow(lh)
		}
		fmt.Println()
		printLinkHealthSummary(result.Links)
	}

	fmt.Println()
	return nil
}

// printStatusAgentRow prints a single agent row in the status output.
func printStatusAgentRow(a agents.Agent) {
	if a.Detected {
		binaryDisplay := a.Binary
		if binaryDisplay == "" {
			binaryDisplay = a.Name
		}
		fmt.Printf("    %s %-10s %s\n",
			ui.Success.Render(ui.IconOk),
			a.Name,
			ui.Muted.Render(binaryDisplay))
	} else {
		fmt.Printf("    %s %-10s %s\n",
			ui.Muted.Render("○ "),
			a.Name,
			ui.Muted.Render("(not installed)"))
	}
}

// printLinkHealthRow prints a single link health row.
func printLinkHealthRow(lh agents.LinkHealth) {
	sourceDisplay := lh.Entry.Source
	targetDisplay := lh.Entry.Target

	switch lh.State {
	case agents.LinkHealthLinked:
		fmt.Printf("    %s %s %s %s\n",
			ui.Success.Render(ui.IconOk),
			ui.Muted.Render(sourceDisplay),
			ui.Muted.Render(ui.IconArrow),
			ui.Muted.Render(targetDisplay))
	case agents.LinkHealthBroken:
		detail := ""
		if lh.Message != "" {
			detail = ui.Muted.Render(" (" + lh.Message + ")")
		}
		fmt.Printf("    %s %s %s %s%s\n",
			ui.Error.Render("✗ "),
			sourceDisplay,
			ui.Muted.Render(ui.IconArrow),
			targetDisplay,
			detail)
	case agents.LinkHealthReplaced:
		detail := ""
		if lh.Message != "" {
			detail = " (" + lh.Message + ")"
		}
		fmt.Printf("    %s %s %s %s%s\n",
			ui.Warning.Render("! "),
			sourceDisplay,
			ui.Muted.Render(ui.IconArrow),
			targetDisplay,
			ui.Warning.Render(" (replaced)"+detail))
	case agents.LinkHealthUnlinked:
		fmt.Printf("    %s %s %s %s\n",
			ui.Muted.Render("○ "),
			sourceDisplay,
			ui.Muted.Render(ui.IconArrow),
			ui.Muted.Render(targetDisplay+" (missing)"))
	case agents.LinkHealthDiverged:
		fmt.Printf("    %s %s %s %s\n",
			ui.Warning.Render("~ "),
			sourceDisplay,
			ui.Muted.Render(ui.IconArrow),
			ui.Warning.Render(targetDisplay+" (diverged)"))
	}
}

// printLinkHealthSummary prints counts for each link health state.
func printLinkHealthSummary(links []agents.LinkHealth) {
	counts := map[agents.LinkHealthState]int{}
	for _, lh := range links {
		counts[lh.State]++
	}

	linked := counts[agents.LinkHealthLinked]
	problems := len(links) - linked

	if problems == 0 {
		ui.Kv("  Summary", ui.Success.Render(fmt.Sprintf("%d/%d linked", linked, len(links))))
	} else {
		ui.Kv("  Summary", fmt.Sprintf("%d/%d linked, %s",
			linked, len(links),
			ui.Warning.Render(fmt.Sprintf("%d issue(s) — run %s for details",
				problems,
				"mine agents diff"))))
	}
}
