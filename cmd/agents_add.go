package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/rnwolfe/mine/internal/agents"
	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

var agentsAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a skill, command, agent, or rule to your store",
	Long: `Scaffold new content in the canonical agents store.

  mine agents add skill <name>     Scaffold a new Agent Skill directory
  mine agents add command <name>   Create a new custom command file
  mine agents add agent <name>     Create a new agent definition file
  mine agents add rule <name>      Create a new rule file`,
	RunE: hook.Wrap("agents.add", func(_ *cobra.Command, _ []string) error {
		fmt.Println()
		fmt.Println("  Scaffold new content in the agents store.")
		fmt.Println()
		fmt.Printf("  %s\n", ui.Accent.Render("mine agents add skill <name>"))
		fmt.Printf("  %s\n", ui.Accent.Render("mine agents add command <name>"))
		fmt.Printf("  %s\n", ui.Accent.Render("mine agents add agent <name>"))
		fmt.Printf("  %s\n", ui.Accent.Render("mine agents add rule <name>"))
		fmt.Println()
		return nil
	}),
}

var agentsAddSkillCmd = &cobra.Command{
	Use:   "skill <name>",
	Short: "Scaffold a new skill in your agents store",
	Long: `Scaffold a new Agent Skill in the canonical agents store.

Creates the following structure:

  skills/<name>/
  ├── SKILL.md        YAML frontmatter + instruction placeholder
  ├── scripts/        Executable scripts
  ├── references/     Documentation and references
  └── assets/         Templates and data files

Name must be lowercase letters, digits, and hyphens (1-64 chars).`,
	Args: cobra.ExactArgs(1),
	RunE: hook.Wrap("agents.add.skill", runAgentsAddSkill),
}

var agentsAddCommandCmd = &cobra.Command{
	Use:   "command <name>",
	Short: "Add a new custom command to your agents store",
	Long: `Create a new custom command file in the canonical agents store.

Creates:

  commands/<name>.md

The .md extension is added automatically if not provided.
Name must be lowercase letters, digits, and hyphens (1-64 chars).`,
	Args: cobra.ExactArgs(1),
	RunE: hook.Wrap("agents.add.command", runAgentsAddCommand),
}

var agentsAddAgentCmd = &cobra.Command{
	Use:   "agent <name>",
	Short: "Add a new agent definition to your store",
	Long: `Create a new agent definition file in the canonical agents store.

Creates:

  agents/<name>.md

Name must be lowercase letters, digits, and hyphens (1-64 chars).`,
	Args: cobra.ExactArgs(1),
	RunE: hook.Wrap("agents.add.agent", runAgentsAddAgent),
}

var agentsAddRuleCmd = &cobra.Command{
	Use:   "rule <name>",
	Short: "Add a new rule to your agents store",
	Long: `Create a new rule file in the canonical agents store.

Creates:

  rules/<name>.md

Name must be lowercase letters, digits, and hyphens (1-64 chars).`,
	Args: cobra.ExactArgs(1),
	RunE: hook.Wrap("agents.add.rule", runAgentsAddRule),
}

func runAgentsAddSkill(_ *cobra.Command, args []string) error {
	name := args[0]

	if !agents.IsInitialized() {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No agents store yet."))
		fmt.Printf("  Run %s first.\n", ui.Accent.Render("mine agents init"))
		fmt.Println()
		return nil
	}

	result, err := agents.AddSkill(name)
	if err != nil {
		return fmt.Errorf("adding skill: %w", err)
	}

	rel, err := filepath.Rel(agents.Dir(), result.Dir)
	if err != nil {
		rel = result.Dir
	}

	fmt.Println()
	ui.Ok(fmt.Sprintf("Skill %s created", ui.Accent.Render(name)))
	fmt.Printf("  Location: %s\n", ui.Muted.Render(rel))
	fmt.Println()
	fmt.Printf("  Next: edit %s to describe the skill\n", ui.Accent.Render(rel+"/SKILL.md"))
	fmt.Println()
	return nil
}

func runAgentsAddCommand(_ *cobra.Command, args []string) error {
	name := args[0]

	if !agents.IsInitialized() {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No agents store yet."))
		fmt.Printf("  Run %s first.\n", ui.Accent.Render("mine agents init"))
		fmt.Println()
		return nil
	}

	result, err := agents.AddCommand(name)
	if err != nil {
		return fmt.Errorf("adding command: %w", err)
	}

	rel, err := filepath.Rel(agents.Dir(), result.File)
	if err != nil {
		rel = result.File
	}

	fmt.Println()
	ui.Ok(fmt.Sprintf("Command %s created", ui.Accent.Render(name)))
	fmt.Printf("  Location: %s\n", ui.Muted.Render(rel))
	fmt.Println()
	fmt.Printf("  Next: edit %s to add command instructions\n", ui.Accent.Render(rel))
	fmt.Println()
	return nil
}

func runAgentsAddAgent(_ *cobra.Command, args []string) error {
	name := args[0]

	if !agents.IsInitialized() {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No agents store yet."))
		fmt.Printf("  Run %s first.\n", ui.Accent.Render("mine agents init"))
		fmt.Println()
		return nil
	}

	result, err := agents.AddAgent(name)
	if err != nil {
		return fmt.Errorf("adding agent: %w", err)
	}

	rel, err := filepath.Rel(agents.Dir(), result.File)
	if err != nil {
		rel = result.File
	}

	fmt.Println()
	ui.Ok(fmt.Sprintf("Agent %s created", ui.Accent.Render(name)))
	fmt.Printf("  Location: %s\n", ui.Muted.Render(rel))
	fmt.Println()
	fmt.Printf("  Next: edit %s to define the agent's role and instructions\n", ui.Accent.Render(rel))
	fmt.Println()
	return nil
}

func runAgentsAddRule(_ *cobra.Command, args []string) error {
	name := args[0]

	if !agents.IsInitialized() {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No agents store yet."))
		fmt.Printf("  Run %s first.\n", ui.Accent.Render("mine agents init"))
		fmt.Println()
		return nil
	}

	result, err := agents.AddRule(name)
	if err != nil {
		return fmt.Errorf("adding rule: %w", err)
	}

	rel, err := filepath.Rel(agents.Dir(), result.File)
	if err != nil {
		rel = result.File
	}

	fmt.Println()
	ui.Ok(fmt.Sprintf("Rule %s created", ui.Accent.Render(name)))
	fmt.Printf("  Location: %s\n", ui.Muted.Render(rel))
	fmt.Println()
	fmt.Printf("  Next: edit %s to add the rule content\n", ui.Accent.Render(rel))
	fmt.Println()
	return nil
}
