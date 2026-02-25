package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/rnwolfe/mine/internal/agents"
	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

var (
	agentsProjectInitForce bool

	agentsProjectLinkAgent string
	agentsProjectLinkCopy  bool
	agentsProjectLinkForce bool
)

var agentsProjectCmd = &cobra.Command{
	Use:   "project",
	Short: "Set up and link agent configs for the current project",
	Long: `Manage project-level agent configurations.

  mine agents project init [path]          Scaffold agent config dirs in a project
  mine agents project link [path]          Link canonical configs to project dirs
  mine agents project link --copy [path]   Copy instead of symlink`,
	RunE: hook.Wrap("agents.project", runAgentsProjectHelp),
}

var agentsProjectInitCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Create agent config directories in the current project",
	Long: `Scaffold project-level agent configuration directories for all detected agents.

Creates agent config directories (e.g. .claude/, .agents/, .gemini/, .opencode/)
along with skills subdirectories and starter instruction files at the project root.

Only creates directories for detected coding agents. Settings files from the
canonical store are seeded when available.

Defaults to the current working directory when no path is given. Re-running is
safe — existing files and directories are preserved unless --force is given.`,
	Args: cobra.MaximumNArgs(1),
	RunE: hook.Wrap("agents.project.init", runAgentsProjectInit),
}

var agentsProjectLinkCmd = &cobra.Command{
	Use:   "link [path]",
	Short: "Link canonical skills and configs into this project",
	Long: `Create symlinks (or copies) from the canonical agents store into the
project-level agent config directories.

Links the following from the canonical store (when present):
  skills/        → <project>/<config-dir>/skills/   (all agents)
  commands/      → <project>/.claude/commands/       (claude only)
  settings/<a>.json → <project>/<config-dir>/settings.json (all agents)

Useful for sharing global agent configs across projects without duplication.
Use --copy for projects where symlinks to external paths are not appropriate
(e.g. to check configs into the repository).

If mine agents project init was run first (creating empty directories),
use --force to replace those empty dirs with symlinks to the canonical store.

Defaults to the current working directory when no path is given.`,
	Args: cobra.MaximumNArgs(1),
	RunE: hook.Wrap("agents.project.link", runAgentsProjectLink),
}

func init() {
	agentsCmd.AddCommand(agentsProjectCmd)
	agentsProjectCmd.AddCommand(agentsProjectInitCmd)
	agentsProjectCmd.AddCommand(agentsProjectLinkCmd)

	agentsProjectInitCmd.Flags().BoolVar(&agentsProjectInitForce, "force", false, "Overwrite existing files")

	agentsProjectLinkCmd.Flags().StringVar(&agentsProjectLinkAgent, "agent", "", "Link only a specific agent (e.g. claude, codex)")
	agentsProjectLinkCmd.Flags().BoolVar(&agentsProjectLinkCopy, "copy", false, "Copy files instead of creating symlinks")
	agentsProjectLinkCmd.Flags().BoolVar(&agentsProjectLinkForce, "force", false, "Overwrite existing files")
}

func runAgentsProjectHelp(_ *cobra.Command, _ []string) error {
	fmt.Println()
	fmt.Println("  Scaffold and manage agent configs at the project level.")
	fmt.Println()
	fmt.Printf("  %s   Scaffold project agent config directories\n", ui.Accent.Render("mine agents project init"))
	fmt.Printf("  %s   Link canonical skills to project skill dirs\n", ui.Accent.Render("mine agents project link"))
	fmt.Println()
	return nil
}

func runAgentsProjectInit(_ *cobra.Command, args []string) error {
	projectPath := ""
	if len(args) > 0 {
		projectPath = args[0]
	}

	opts := agents.ProjectInitOptions{Force: agentsProjectInitForce}

	actions, err := agents.ProjectInit(projectPath, opts)
	if err != nil {
		return err
	}

	fmt.Println()

	if len(actions) == 0 {
		fmt.Println(ui.Muted.Render("  No agents detected — run " + ui.Accent.Render("mine agents detect") + ui.Muted.Render(" to register agents.")))
		fmt.Println()
		return nil
	}

	createdCount := 0
	for _, a := range actions {
		printProjectInitAction(a)
		if a.Status == "created" {
			createdCount++
		}
	}

	fmt.Println()
	if createdCount > 0 {
		target := "current directory"
		if projectPath != "" {
			target = filepath.Base(projectPath)
		}
		ui.Ok(fmt.Sprintf("Project scaffolded — %d item(s) created in %s", createdCount, target))
	} else {
		fmt.Println(ui.Muted.Render("  Project already scaffolded — nothing new to create."))
		fmt.Printf("  Use %s to overwrite existing files.\n", ui.Accent.Render("--force"))
	}

	fmt.Println()
	return nil
}

func runAgentsProjectLink(_ *cobra.Command, args []string) error {
	if !agents.IsInitialized() {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No agents store yet."))
		fmt.Printf("  Run %s first.\n", ui.Accent.Render("mine agents init"))
		fmt.Println()
		return nil
	}

	projectPath := ""
	if len(args) > 0 {
		projectPath = args[0]
	}

	opts := agents.ProjectLinkOptions{
		Agent: agentsProjectLinkAgent,
		Copy:  agentsProjectLinkCopy,
		Force: agentsProjectLinkForce,
	}

	actions, err := agents.ProjectLink(projectPath, opts)
	if err != nil {
		return err
	}

	fmt.Println()

	if len(actions) == 0 {
		fmt.Println(ui.Muted.Render("  Nothing to link — store skills/ is empty or no agents detected."))
		fmt.Printf("  Add skills to %s and re-run.\n", ui.Accent.Render(agents.Dir()+"/skills/"))
		fmt.Println()
		return nil
	}

	createdCount := 0
	for _, a := range actions {
		printLinkAction(a)
		if a.Err == nil {
			createdCount++
		}
	}

	fmt.Println()
	if createdCount > 0 {
		ui.Ok(fmt.Sprintf("%d project link(s) configured", createdCount))
	}
	fmt.Println()
	return nil
}

// printProjectInitAction prints a single project init action row.
func printProjectInitAction(a agents.ProjectInitAction) {
	switch {
	case a.Err != nil:
		fmt.Printf("  %-6s %s %s\n",
			a.Kind,
			ui.Warning.Render(ui.IconWarn+a.Err.Error()),
			ui.Muted.Render(a.Path))
	case a.Status == "exists":
		fmt.Printf("  %-6s %s %s\n",
			a.Kind,
			ui.Muted.Render(ui.IconOk+"exists"),
			ui.Muted.Render(a.Path))
	default:
		fmt.Printf("  %-6s %s %s\n",
			a.Kind,
			ui.Success.Render(ui.IconOk+a.Status),
			a.Path)
	}
}
