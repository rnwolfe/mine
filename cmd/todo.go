package cmd

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rnwolfe/mine/internal/config"
	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/proj"
	"github.com/rnwolfe/mine/internal/store"
	"github.com/rnwolfe/mine/internal/todo"
	"github.com/rnwolfe/mine/internal/tui"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

var todoCmd = &cobra.Command{
	Use:     "todo",
	Aliases: []string{"t"},
	Short:   "Fast, no-nonsense task management",
	Long: `Capture ideas, track work, and knock things out. Add, complete, and browse todos.

In an interactive terminal, launches a full-screen todo browser.
Pipe output or use subcommands for scripting.

Tasks are automatically scoped to the current project when run inside a
registered project directory. Use --all to view tasks across all projects.

Keyboard shortcuts (interactive mode):
  j / k        Move down / up
  x / space    Toggle done/undone
  a            Add new todo (type title, Enter to save)
  d            Delete selected todo
  s            Cycle schedule bucket (today → soon → later → someday)
  /            Filter todos (fuzzy search)
  g / G        Jump to top / bottom
  Esc          Clear active filter (or no-op)
  q / Ctrl+C   Quit`,
	RunE: hook.Wrap("todo.list", runTodoList),
}

var (
	todoPriority         string
	todoDue              string
	todoTags             string
	todoShowDone         bool
	todoShowAll          bool
	todoProjectName      string
	todoScheduleFlag     string
	todoIncludeSomeday   bool
	todoNoteFlag         string
	todoStatsProjectFlag string
	todoEveryFlag        string
)

func init() {
	// Subcommands
	todoCmd.AddCommand(todoAddCmd)
	todoCmd.AddCommand(todoDoneCmd)
	todoCmd.AddCommand(todoRmCmd)
	todoCmd.AddCommand(todoEditCmd)
	todoCmd.AddCommand(todoScheduleCmd)
	todoCmd.AddCommand(todoNextCmd)
	todoCmd.AddCommand(todoNoteCmd)
	todoCmd.AddCommand(todoShowCmd)
	todoCmd.AddCommand(todoStatsCmd)
	todoCmd.AddCommand(todoRecurringCmd)

	// Flags on stats subcommand
	todoStatsCmd.Flags().StringVar(&todoStatsProjectFlag, "project", "", "Scope stats to a named project")

	// Flags on the root todo command
	todoCmd.Flags().BoolVar(&todoShowDone, "done", false, "Show completed todos too")
	todoCmd.Flags().BoolVarP(&todoShowAll, "all", "a", false, "Show todos across all projects")
	todoCmd.Flags().StringVar(&todoProjectName, "project", "", "Scope to a named project")
	todoCmd.Flags().BoolVar(&todoIncludeSomeday, "someday", false, "Include someday tasks in output")

	// Flags on add subcommand
	todoAddCmd.Flags().StringVarP(&todoPriority, "priority", "p", "med", "Priority: low, med, high, crit")
	todoAddCmd.Flags().StringVarP(&todoDue, "due", "d", "", "Due date (YYYY-MM-DD, tomorrow, next-week)")
	todoAddCmd.Flags().StringVarP(&todoTags, "tags", "t", "", "Comma-separated tags")
	todoAddCmd.Flags().StringVar(&todoProjectName, "project", "", "Assign to a named project")
	todoAddCmd.Flags().StringVar(&todoScheduleFlag, "schedule", "later", "Schedule bucket: today, soon, later, someday")
	todoAddCmd.Flags().StringVar(&todoNoteFlag, "note", "", "Initial body/context for the task")
	todoAddCmd.Flags().StringVar(&todoEveryFlag, "every", "", "Recurrence frequency: day (d), weekday (wd), week (w), month (m)")
}

var todoAddCmd = &cobra.Command{
	Use:   "add <title>",
	Short: "Capture an idea before it escapes",
	Args:  cobra.MinimumNArgs(1),
	RunE:  hook.Wrap("todo.add", runTodoAdd),
}

var todoDoneCmd = &cobra.Command{
	Use:     "done <id>",
	Aliases: []string{"do", "complete", "x"},
	Short:   "Mark a todo complete — check it off",
	Args:    cobra.ExactArgs(1),
	RunE:    hook.Wrap("todo.done", runTodoDone),
}

var todoRmCmd = &cobra.Command{
	Use:     "rm <id>",
	Aliases: []string{"remove", "delete"},
	Short:   "Remove a todo from the list",
	Args:    cobra.ExactArgs(1),
	RunE:    hook.Wrap("todo.rm", runTodoRm),
}

// resolveTodoProject resolves the project path for todo operations.
// If projectName is set, it looks up that project in the registry.
// Otherwise, it auto-detects the project from the current working directory.
// Returns nil if not in any registered project (global context).
func resolveTodoProject(ps *proj.Store, projectName string) (*string, error) {
	if projectName != "" {
		p, err := ps.Get(projectName)
		if err != nil {
			if errors.Is(err, proj.ErrProjectNotFound) {
				return nil, fmt.Errorf("project %q not found in registry — use %s to list projects",
					projectName, ui.Accent.Render("mine proj list"))
			}
			return nil, fmt.Errorf("looking up project %q: %w", projectName, err)
		}
		return &p.Path, nil
	}

	// Auto-detect from cwd.
	p, err := ps.FindForCWD()
	if err != nil {
		return nil, err
	}
	if p != nil {
		return &p.Path, nil
	}
	return nil, nil
}

func runTodoList(_ *cobra.Command, _ []string) error {
	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	ps := proj.NewStore(db.Conn())

	opts := todo.ListOptions{
		ShowDone:       todoShowDone,
		AllProjects:    todoShowAll,
		IncludeSomeday: todoIncludeSomeday,
	}

	var projectPath *string
	if !todoShowAll {
		projectPath, err = resolveTodoProject(ps, todoProjectName)
		if err != nil {
			return err
		}
		opts.ProjectPath = projectPath
	}

	// Always resolve the cwd project for urgency scoring boost, independent of
	// the --all flag. When --all is set, projectPath is nil (no filter) but we
	// still want the current-project boost to apply for tasks in the active project.
	cwdProject, err := resolveTodoProject(ps, "")
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	now := time.Now()
	opts.CurrentProjectPath = cwdProject
	w := urgencyWeightsFromConfig(cfg)
	opts.Weights = &w
	opts.ReferenceTime = now

	ts := todo.NewStore(db.Conn())
	todos, err := ts.List(opts)
	if err != nil {
		return err
	}

	// Launch interactive TUI when connected to a terminal.
	if tui.IsTTY() {
		return runTodoTUI(ts, todos, projectPath, todoShowAll)
	}

	return printTodoList(todos, ts, projectPath, todoShowAll)
}

func runTodoAdd(_ *cobra.Command, args []string) error {
	title := strings.Join(args, " ")

	prio := parsePriority(todoPriority)
	due := parseDueDate(todoDue)

	var tags []string
	if todoTags != "" {
		tags = strings.Split(todoTags, ",")
		for i := range tags {
			tags[i] = strings.TrimSpace(tags[i])
		}
	}

	schedule, err := todo.ParseSchedule(todoScheduleFlag)
	if err != nil {
		return fmt.Errorf("%w\n  Use: %s", err, ui.Accent.Render("--schedule today|soon|later|someday"))
	}

	recurrence := todo.RecurrenceNone
	if todoEveryFlag != "" {
		recurrence, err = todo.ParseRecurrence(todoEveryFlag)
		if err != nil {
			return fmt.Errorf("%w\n  Use: %s", err, ui.Accent.Render("--every day|weekday|week|month"))
		}
	}

	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	ps := proj.NewStore(db.Conn())
	projectPath, err := resolveTodoProject(ps, todoProjectName)
	if err != nil {
		return err
	}

	ts := todo.NewStore(db.Conn())
	id, err := ts.Add(title, todoNoteFlag, prio, tags, due, projectPath, schedule, recurrence)
	if err != nil {
		return err
	}

	icon := todo.PriorityIcon(prio)
	fmt.Printf("  %s Added %s %s\n", ui.Success.Render("✓"), icon, ui.Accent.Render(fmt.Sprintf("#%d", id)))
	fmt.Printf("    %s\n", title)

	if projectPath != nil {
		projName := filepath.Base(*projectPath)
		fmt.Printf("    Project: %s\n", ui.Muted.Render(projName))
	}

	if schedule != todo.ScheduleLater {
		fmt.Printf("    Schedule: %s\n", todo.FormatScheduleTag(schedule))
	}

	if due != nil {
		fmt.Printf("    Due: %s\n", ui.Muted.Render(due.Format("Mon, Jan 2")))
	}

	if recurrence != todo.RecurrenceNone {
		fmt.Printf("    Recurrence: %s\n", ui.Muted.Render("↻ "+todo.RecurrenceLabel(recurrence)))
	}

	fmt.Println()

	return nil
}

func runTodoDone(_ *cobra.Command, args []string) error {
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("%q is not a valid todo ID — use %s to see IDs", args[0], ui.Accent.Render("mine todo"))
	}

	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	ts := todo.NewStore(db.Conn())

	// Get the todo first for display
	t, err := ts.Get(id)
	if err != nil {
		return err
	}

	spawnedID, spawnedDue, err := ts.Complete(id)
	if err != nil {
		return err
	}

	fmt.Printf("  %s Done! %s\n", ui.Success.Render("✓"), ui.Muted.Render(t.Title))

	if spawnedID > 0 {
		dueStr := "today"
		if spawnedDue != nil {
			dueStr = spawnedDue.Format("Mon, Jan 2")
		}
		fmt.Printf("  %s Next occurrence spawned: %s (due %s)\n",
			ui.Muted.Render("↻"),
			ui.Accent.Render(fmt.Sprintf("#%d", spawnedID)),
			ui.Muted.Render(dueStr),
		)
	}

	// Check remaining
	open, _, _, _ := ts.Count(nil)
	if open == 0 {
		fmt.Println(ui.Success.Render("  " + ui.IconParty + " All clear! Nothing left to do."))
	} else {
		fmt.Printf("  %s\n", ui.Muted.Render(fmt.Sprintf("  %d remaining", open)))
	}
	fmt.Println()

	return nil
}

func runTodoRm(_ *cobra.Command, args []string) error {
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("%q is not a valid todo ID — use %s to see IDs", args[0], ui.Accent.Render("mine todo"))
	}

	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	ts := todo.NewStore(db.Conn())
	if err := ts.Delete(id); err != nil {
		return err
	}

	fmt.Printf("  %s Removed #%d\n", ui.Success.Render("✓"), id)
	fmt.Println()
	return nil
}

func parsePriority(s string) int {
	switch strings.ToLower(s) {
	case "low", "l", "1":
		return todo.PrioLow
	case "high", "h", "3":
		return todo.PrioHigh
	case "crit", "critical", "c", "4", "!":
		return todo.PrioCrit
	default:
		return todo.PrioMedium
	}
}

func parseDueDate(s string) *time.Time {
	if s == "" {
		return nil
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	switch strings.ToLower(s) {
	case "today":
		return &today
	case "tomorrow", "tom":
		t := today.AddDate(0, 0, 1)
		return &t
	case "next-week", "nextweek", "nw":
		t := today.AddDate(0, 0, 7)
		return &t
	case "next-month", "nm":
		t := today.AddDate(0, 1, 0)
		return &t
	}

	// Try parsing as date
	formats := []string{"2006-01-02", "01/02/2006", "Jan 2", "January 2"}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			// If no year in format, use current year
			if t.Year() == 0 {
				t = time.Date(now.Year(), t.Month(), t.Day(), 0, 0, 0, 0, now.Location())
			}
			return &t
		}
	}

	return nil
}
