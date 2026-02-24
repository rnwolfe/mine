package cmd

import (
	"fmt"
	"strings"

	"github.com/rnwolfe/mine/internal/agents"
	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

var agentsListType string

var agentsListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show a categorized inventory of managed agent configs",
	Long: `Show a categorized inventory of managed agent configs in the canonical store.

Lists skills (with descriptions from SKILL.md frontmatter), commands, agent
definitions, rules, instructions, and settings files.

Use --type to filter to a specific content type.`,
	RunE: hook.Wrap("agents.list", runAgentsList),
}

// validListTypes is the set of accepted --type values.
var validListTypes = map[string]bool{
	"skills":       true,
	"commands":     true,
	"agents":       true,
	"rules":        true,
	"instructions": true,
	"settings":     true,
}

func runAgentsList(_ *cobra.Command, _ []string) error {
	if !agents.IsInitialized() {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No agents store yet."))
		fmt.Printf("  Run %s first.\n", ui.Accent.Render("mine agents init"))
		fmt.Println()
		return nil
	}

	t := strings.ToLower(strings.TrimSpace(agentsListType))
	if t != "" && !validListTypes[t] {
		return fmt.Errorf("unknown type %q — valid types: skills, commands, agents, rules, instructions, settings", agentsListType)
	}

	result, err := agents.List(agents.ListOptions{Type: t})
	if err != nil {
		return fmt.Errorf("listing agent configs: %w", err)
	}

	fmt.Println()
	fmt.Printf("  %s\n", ui.Title.Render("⛏  Agent Configs"))
	fmt.Println()

	total := 0

	if t == "" || t == "skills" {
		total += printListSection("Skills", result.Skills)
	}
	if t == "" || t == "commands" {
		total += printListSection("Commands", result.Commands)
	}
	if t == "" || t == "agents" {
		total += printListSection("Agents", result.Agents)
	}
	if t == "" || t == "rules" {
		total += printListSection("Rules", result.Rules)
	}
	if t == "" || t == "instructions" {
		total += printListSection("Instructions", result.Instructions)
	}
	if t == "" || t == "settings" {
		total += printListSection("Settings", result.Settings)
	}

	if total == 0 {
		if t != "" {
			fmt.Printf("  %s\n", ui.Muted.Render(fmt.Sprintf("No %s found.", t)))
			if t == "skills" || t == "commands" || t == "agents" || t == "rules" {
				fmt.Printf("  Add one with %s\n", ui.Accent.Render(fmt.Sprintf("mine agents add %s <name>", singularType(t))))
			}
		} else {
			fmt.Println(ui.Muted.Render("  Store is empty — add some content first."))
			fmt.Printf("  Try %s or %s\n",
				ui.Accent.Render("mine agents add skill <name>"),
				ui.Accent.Render("mine agents adopt"))
		}
		fmt.Println()
	}

	return nil
}

// printListSection prints a category header and its items.
// Returns the number of items printed.
func printListSection(title string, items []agents.ContentItem) int {
	if len(items) == 0 {
		return 0
	}

	fmt.Printf("  %s\n", ui.KeyStyle.Render(fmt.Sprintf("%s (%d):", title, len(items))))

	// Compute the max name length for column alignment.
	maxNameLen := 0
	for _, item := range items {
		if len(item.Name) > maxNameLen {
			maxNameLen = len(item.Name)
		}
	}
	if maxNameLen < 16 {
		maxNameLen = 16
	}

	for _, item := range items {
		namePadded := fmt.Sprintf("%-*s", maxNameLen, item.Name)
		if item.Description != "" {
			fmt.Printf("    %s  %s\n",
				namePadded,
				ui.Muted.Render(item.Description))
		} else {
			fmt.Printf("    %s\n", namePadded)
		}
	}

	fmt.Println()
	return len(items)
}

// singularType returns the singular form of a list type for use in suggestions.
func singularType(t string) string {
	switch t {
	case "skills":
		return "skill"
	case "commands":
		return "command"
	case "agents":
		return "agent"
	case "rules":
		return "rule"
	case "instructions":
		return "instruction"
	case "settings":
		return "setting"
	default:
		return t
	}
}
