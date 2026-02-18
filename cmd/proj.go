package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/proj"
	"github.com/rnwolfe/mine/internal/store"
	"github.com/rnwolfe/mine/internal/tui"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

var (
	projRmYes      bool
	projScanDepth  int
	projUsePrev    bool
	projPrintPath  bool
	projConfigName string
)

var projCmd = &cobra.Command{
	Use:     "proj",
	Aliases: []string{"project"},
	Short:   "Project switcher and context manager",
	Long:    `Register projects, jump between them quickly, and manage per-project context.`,
	RunE:    hook.Wrap("proj", runProj),
}

func init() {
	rootCmd.AddCommand(projCmd)

	projCmd.AddCommand(projAddCmd)
	projCmd.AddCommand(projRmCmd)
	projCmd.AddCommand(projListCmd)
	projCmd.AddCommand(projOpenCmd)
	projCmd.AddCommand(projScanCmd)
	projCmd.AddCommand(projConfigCmd)

	projRmCmd.Flags().BoolVarP(&projRmYes, "yes", "y", false, "Skip confirmation prompt")
	projScanCmd.Flags().IntVar(&projScanDepth, "depth", 3, "Scan recursion depth")
	projOpenCmd.Flags().BoolVar(&projUsePrev, "previous", false, "Open previously active project")
	projOpenCmd.Flags().BoolVar(&projPrintPath, "print-path", false, "Print resolved path only")
	projCmd.Flags().BoolVar(&projPrintPath, "print-path", false, "Print selected path only (for shell helpers)")
	projCmd.Flags().MarkHidden("print-path")
	projConfigCmd.Flags().StringVarP(&projConfigName, "project", "p", "", "Project name (defaults to current project)")
}

func runProj(_ *cobra.Command, _ []string) error {
	db, err := store.Open()
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	defer db.Close()

	ps := proj.NewStore(db.Conn())
	projects, err := ps.List()
	if err != nil {
		return err
	}
	if len(projects) == 0 {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No projects registered yet."))
		fmt.Printf("  Add one: %s\n", ui.Accent.Render("mine proj add ."))
		fmt.Println()
		return nil
	}

	if !tui.IsTTY() {
		if projPrintPath {
			return fmt.Errorf("--print-path requires a terminal when no project name is provided")
		}
		return printProjectList(projects)
	}

	items := make([]tui.Item, len(projects))
	for i := range projects {
		items[i] = projects[i]
	}
	chosen, err := tui.Run(items,
		tui.WithTitle(ui.IconPick+"Select project"),
		tui.WithHeight(12),
	)
	if err != nil {
		return err
	}
	if chosen == nil {
		return nil
	}

	res, err := ps.Open(chosen.Title())
	if err != nil {
		return err
	}

	if projPrintPath {
		fmt.Print(res.Project.Path)
		return nil
	}

	ui.Ok(fmt.Sprintf("Selected %s", ui.Accent.Render(res.Project.Name)))
	fmt.Printf("  Switch now: %s\n", ui.Muted.Render("cd "+res.Project.Path))
	fmt.Println()
	return nil
}

var projAddCmd = &cobra.Command{
	Use:   "add [path]",
	Short: "Register a project path",
	Args:  cobra.MaximumNArgs(1),
	RunE:  hook.Wrap("proj.add", runProjAdd),
}

func runProjAdd(_ *cobra.Command, args []string) error {
	path := ""
	if len(args) > 0 {
		path = args[0]
	}

	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	ps := proj.NewStore(db.Conn())
	p, err := ps.Add(path)
	if err != nil {
		return err
	}

	ui.Ok(fmt.Sprintf("Registered project %s", ui.Accent.Render(p.Name)))
	fmt.Printf("  Path: %s\n", ui.Muted.Render(p.Path))
	fmt.Println()
	return nil
}

var projRmCmd = &cobra.Command{
	Use:   "rm <name>",
	Short: "Remove a registered project",
	Args:  cobra.ExactArgs(1),
	RunE:  hook.Wrap("proj.rm", runProjRm),
}

func runProjRm(_ *cobra.Command, args []string) error {
	name := args[0]
	if !projRmYes {
		if !tui.IsTTY() {
			return fmt.Errorf("non-interactive remove requires --yes")
		}
		if !confirmRemove(name) {
			ui.Warn("Cancelled.")
			return nil
		}
	}

	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	ps := proj.NewStore(db.Conn())
	if err := ps.Remove(name); err != nil {
		return err
	}

	ui.Ok(fmt.Sprintf("Removed project %s", ui.Accent.Render(name)))
	fmt.Println()
	return nil
}

func confirmRemove(name string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("  %s [y/N] ", ui.Warning.Render(fmt.Sprintf("Remove project %q?", name)))
	line, _ := reader.ReadString('\n')
	answer := strings.TrimSpace(strings.ToLower(line))
	return answer == "y" || answer == "yes"
}

var projListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List registered projects",
	RunE:    hook.Wrap("proj.list", runProjList),
}

func runProjList(_ *cobra.Command, _ []string) error {
	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	ps := proj.NewStore(db.Conn())
	projects, err := ps.List()
	if err != nil {
		return err
	}
	if len(projects) == 0 {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No projects registered yet."))
		fmt.Println()
		return nil
	}

	return printProjectList(projects)
}

var projOpenCmd = &cobra.Command{
	Use:   "open [name]",
	Short: "Open a project context (shell-aware with --print-path)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  hook.Wrap("proj.open", runProjOpen),
}

func runProjOpen(_ *cobra.Command, args []string) error {
	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	ps := proj.NewStore(db.Conn())
	var result *proj.OpenResult
	if projUsePrev {
		result, err = ps.OpenPrevious()
		if err != nil {
			return err
		}
	} else {
		if len(args) == 0 {
			return fmt.Errorf("project name is required unless --previous is set")
		}
		result, err = ps.Open(args[0])
		if err != nil {
			return err
		}
	}

	if projPrintPath {
		fmt.Print(result.Project.Path)
		return nil
	}

	ui.Ok(fmt.Sprintf("Project %s ready", ui.Accent.Render(result.Project.Name)))
	fmt.Printf("  Path: %s\n", ui.Muted.Render(result.Project.Path))
	if result.Project.Branch != "" {
		fmt.Printf("  Branch: %s\n", ui.Muted.Render(result.Project.Branch))
	}

	if result.Previous != "" && result.Previous != result.Project.Name {
		fmt.Printf("  Previous: %s\n", ui.Muted.Render(result.Previous))
	}
	fmt.Printf("  Shell switch: %s\n", ui.Accent.Render("p "+result.Project.Name))
	fmt.Println()
	return nil
}

var projScanCmd = &cobra.Command{
	Use:   "scan [dir]",
	Short: "Recursively discover git repos and register them",
	Args:  cobra.MaximumNArgs(1),
	RunE:  hook.Wrap("proj.scan", runProjScan),
}

func runProjScan(_ *cobra.Command, args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	ps := proj.NewStore(db.Conn())
	added, err := ps.Scan(dir, projScanDepth)
	if err != nil {
		return err
	}

	if len(added) == 0 {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No new git projects discovered."))
		fmt.Println()
		return nil
	}

	ui.Ok(fmt.Sprintf("Added %d projects", len(added)))
	for _, p := range added {
		fmt.Printf("  %s %s\n", ui.Success.Render("●"), ui.Muted.Render(p.Path))
	}
	fmt.Println()
	return nil
}

var projConfigCmd = &cobra.Command{
	Use:   "config [key] [value]",
	Short: "Get/set per-project settings",
	Args:  cobra.RangeArgs(0, 2),
	RunE:  hook.Wrap("proj.config", runProjConfig),
}

func runProjConfig(_ *cobra.Command, args []string) error {
	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	ps := proj.NewStore(db.Conn())
	projectName := projConfigName
	if projectName == "" {
		current, err := ps.CurrentName()
		if err != nil {
			return err
		}
		if current == "" {
			return fmt.Errorf("no current project set; use --project or run mine proj open <name>")
		}
		projectName = current
	}

	switch len(args) {
	case 0:
		fmt.Println()
		fmt.Println(ui.Title.Render("  Project Settings"))
		ui.Kv("Project", projectName)
		for _, key := range proj.SupportedConfigKeys() {
			value, err := ps.GetSetting(projectName, key)
			if err != nil {
				return err
			}
			if value == "" {
				value = "(unset)"
			}
			ui.Kv("  "+key, value)
		}
		fmt.Println()
		return nil
	case 1:
		value, err := ps.GetSetting(projectName, args[0])
		if err != nil {
			return err
		}
		fmt.Print(value)
		return nil
	default:
		if err := ps.SetSetting(projectName, args[0], args[1]); err != nil {
			return err
		}
		ui.Ok(fmt.Sprintf("Saved %s for %s", args[0], ui.Accent.Render(projectName)))
		fmt.Println()
		return nil
	}
}

func printProjectList(projects []proj.Project) error {
	fmt.Println()
	fmt.Println(ui.Title.Render("  Registered Projects"))
	fmt.Println()

	for _, p := range projects {
		last := "never"
		if !p.LastAccessed.IsZero() {
			last = p.LastAccessed.Local().Format(time.RFC822)
		}
		branch := "-"
		if p.Branch != "" {
			branch = p.Branch
		}
		fmt.Printf("  %s  %s\n", ui.Accent.Render(fmt.Sprintf("%-18s", p.Name)), ui.Muted.Render(p.Path))
		fmt.Printf("      last: %s  branch: %s\n", ui.Muted.Render(last), ui.Muted.Render(branch))
	}
	fmt.Println()
	return nil
}
