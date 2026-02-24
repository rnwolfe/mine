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
	todoPriority          string
	todoDue               string
	todoTags              string
	todoShowDone          bool
	todoShowAll           bool
	todoProjectName       string
	todoScheduleFlag      string
	todoIncludeSomeday    bool
	todoNoteFlag          string
	todoStatsProjectFlag  string
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

var todoEditCmd = &cobra.Command{
	Use:   "edit <id> <new title>",
	Short: "Rename a todo",
	Args:  cobra.MinimumNArgs(2),
	RunE:  hook.Wrap("todo.edit", runTodoEdit),
}

var todoScheduleCmd = &cobra.Command{
	Use:   "schedule <id> <when>",
	Short: "Set the scheduling intent for a todo",
	Long: `Set the scheduling bucket for a todo. Buckets represent when you intend to work on it:

  today    — tackle it today (alias: t)
  soon     — coming up, within a few days (alias: s)
  later    — on the radar, not urgent (alias: l)
  someday  — aspirational, hidden from default view (alias: sd)

Someday tasks are hidden from the default list. Use 'mine todo --someday' to see them.`,
	Args: cobra.ExactArgs(2),
	RunE: hook.Wrap("todo.schedule", runTodoSchedule),
}

var todoNoteCmd = &cobra.Command{
	Use:   "note <id> <text>",
	Short: "Append a timestamped annotation to a task",
	Args:  cobra.ExactArgs(2),
	RunE:  hook.Wrap("todo.note", runTodoNote),
}

var todoShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Display full task detail including notes",
	Args:  cobra.ExactArgs(1),
	RunE:  hook.Wrap("todo.show", runTodoShow),
}

var todoStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "View completion velocity and streak metrics",
	Long: `Show completion analytics derived from your task history.

Displays:
  - Completion streak (consecutive days with >= 1 completion)
  - Tasks completed this week (Monday-start) and this month
  - Average time-to-close for completed tasks
  - Total focus time from linked dig sessions (if available)
  - Per-project breakdown of open/completed counts

Use --project to scope stats to a single named project.`,
	RunE: hook.Wrap("todo.stats", runTodoStats),
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

func runTodoTUI(ts *todo.Store, todos []todo.Todo, projectPath *string, showAll bool) error {
	actions, err := tui.RunTodo(todos, projectPath, showAll)
	if err != nil {
		return err
	}

	// Apply actions returned from the TUI.
	var failedActions []string
	for _, a := range actions {
		switch a.Type {
		case "toggle":
			t, err := ts.Get(a.ID)
			if err != nil {
				failedActions = append(failedActions, fmt.Sprintf("toggle #%d: %v", a.ID, err))
				continue
			}
			if t.Done {
				if err := ts.Uncomplete(a.ID); err != nil {
					failedActions = append(failedActions, fmt.Sprintf("uncomplete #%d: %v", a.ID, err))
				}
			} else {
				if err := ts.Complete(a.ID); err != nil {
					failedActions = append(failedActions, fmt.Sprintf("complete #%d: %v", a.ID, err))
				}
			}
		case "delete":
			if err := ts.Delete(a.ID); err != nil {
				failedActions = append(failedActions, fmt.Sprintf("delete #%d: %v", a.ID, err))
			}
		case "add":
			if strings.TrimSpace(a.Text) != "" {
				if _, err := ts.Add(a.Text, "", todo.PrioMedium, nil, nil, a.ProjectPath, todo.ScheduleLater); err != nil {
					failedActions = append(failedActions, fmt.Sprintf("add %q: %v", a.Text, err))
				}
			}
		case "schedule":
			if err := ts.SetSchedule(a.ID, a.Schedule); err != nil {
				failedActions = append(failedActions, fmt.Sprintf("schedule #%d: %v", a.ID, err))
			}
		}
	}

	if len(failedActions) > 0 {
		fmt.Println(ui.Warning.Render("Some actions failed:"))
		for _, msg := range failedActions {
			fmt.Println("  " + msg)
		}
	}

	return nil
}

func printTodoList(todos []todo.Todo, ts *todo.Store, projectPath *string, showAll bool) error {
	if len(todos) == 0 {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No todos yet. Life is good?"))
		fmt.Println()
		fmt.Printf("  Add one: %s\n", ui.Accent.Render(`mine todo add "something important"`))
		fmt.Println()
		return nil
	}

	// Fetch accumulated focus times for all listed todos in one query.
	ids := make([]int, len(todos))
	for i, t := range todos {
		ids[i] = t.ID
	}
	focusTimes, _ := ts.FocusTimeMap(ids) // non-critical; missing focus time is fine

	fmt.Println()
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	for _, t := range todos {
		marker := " "
		if t.Done {
			marker = ui.Success.Render("✓")
		}

		id := ui.Muted.Render(fmt.Sprintf("#%-3d", t.ID))
		prio := todo.PriorityIcon(t.Priority)
		title := t.Title
		if t.Done {
			title = ui.Muted.Render(title)
		}

		schedTag := renderScheduleTag(t.Schedule)
		line := fmt.Sprintf("  %s %s %s %s %s", marker, id, prio, schedTag, title)

		// Due date annotation
		if t.DueDate != nil && !t.Done {
			due := *t.DueDate
			dueDay := time.Date(due.Year(), due.Month(), due.Day(), 0, 0, 0, 0, due.Location())
			switch {
			case dueDay.Before(today):
				line += ui.Error.Render(fmt.Sprintf(" (overdue: %s)", due.Format("Jan 2")))
			case dueDay.Equal(today):
				line += ui.Warning.Render(" (due today!)")
			case dueDay.Before(today.AddDate(0, 0, 7)):
				line += ui.Muted.Render(fmt.Sprintf(" (due %s)", due.Format("Mon")))
			default:
				line += ui.Muted.Render(fmt.Sprintf(" (due %s)", due.Format("Jan 2")))
			}
		}

		// Tags
		if len(t.Tags) > 0 {
			tags := ui.Muted.Render(" [" + strings.Join(t.Tags, ", ") + "]")
			line += tags
		}

		// Focus time annotation — only when > 0
		if ft, ok := focusTimes[t.ID]; ok && ft > 0 {
			line += " " + ui.Muted.Render(formatFocusTime(ft))
		}

		// Project annotation when viewing across all projects
		if showAll && t.ProjectPath != nil {
			projName := filepath.Base(*t.ProjectPath)
			line += ui.Muted.Render(fmt.Sprintf(" @%s", projName))
		}

		fmt.Println(line)
	}

	open, _, overdue, _ := ts.Count(projectPath)
	fmt.Println()
	summary := ui.Muted.Render(fmt.Sprintf("  %d open", open))
	if overdue > 0 {
		summary += ui.Error.Render(fmt.Sprintf(" · %d overdue", overdue))
	}
	fmt.Println(summary)
	fmt.Println()

	return nil
}

// formatFocusTime formats a duration as [Xh Ym] or [Ym] for list display.
func formatFocusTime(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("[%dh %dm]", h, m)
	}
	return fmt.Sprintf("[%dm]", m)
}

// renderScheduleTag returns a styled schedule indicator for list output.
func renderScheduleTag(schedule string) string {
	label := "[" + todo.ScheduleLabel(schedule) + "]"
	switch schedule {
	case todo.ScheduleToday:
		return ui.ScheduleTodayStyle.Render(label)
	case todo.ScheduleSoon:
		return ui.ScheduleSoonStyle.Render(label)
	default: // later, someday — muted
		return ui.Muted.Render(label)
	}
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
	id, err := ts.Add(title, todoNoteFlag, prio, tags, due, projectPath, schedule)
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
		fmt.Printf("    Schedule: %s\n", renderScheduleTag(schedule))
	}

	if due != nil {
		fmt.Printf("    Due: %s\n", ui.Muted.Render(due.Format("Mon, Jan 2")))
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

	if err := ts.Complete(id); err != nil {
		return err
	}

	fmt.Printf("  %s Done! %s\n", ui.Success.Render("✓"), ui.Muted.Render(t.Title))

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

func runTodoEdit(_ *cobra.Command, args []string) error {
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("%q is not a valid todo ID — use %s to see IDs", args[0], ui.Accent.Render("mine todo"))
	}
	newTitle := strings.Join(args[1:], " ")

	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	ts := todo.NewStore(db.Conn())
	if err := ts.Edit(id, &newTitle, nil); err != nil {
		return err
	}

	fmt.Printf("  %s Updated #%d → %s\n", ui.Success.Render("✓"), id, newTitle)
	fmt.Println()
	return nil
}

func runTodoSchedule(_ *cobra.Command, args []string) error {
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("%q is not a valid todo ID — use %s to see IDs", args[0], ui.Accent.Render("mine todo"))
	}

	schedule, err := todo.ParseSchedule(args[1])
	if err != nil {
		return fmt.Errorf("%w\n  Valid values: %s",
			err,
			ui.Accent.Render("today (t), soon (s), later (l), someday (sd)"))
	}

	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	ts := todo.NewStore(db.Conn())
	if err := ts.SetSchedule(id, schedule); err != nil {
		return fmt.Errorf("scheduling todo #%d: %w", id, err)
	}

	schedLabel := renderScheduleTag(schedule)
	fmt.Printf("  %s Scheduled #%d → %s\n", ui.Success.Render("✓"), id, schedLabel)
	fmt.Println()
	return nil
}

func runTodoNote(_ *cobra.Command, args []string) error {
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("%q is not a valid todo ID — use %s to see IDs", args[0], ui.Accent.Render("mine todo"))
	}
	text := args[1]

	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	ts := todo.NewStore(db.Conn())
	if err := ts.AddNote(id, text); err != nil {
		return err
	}

	preview := text
	if len(preview) > 60 {
		preview = preview[:57] + "…"
	}
	fmt.Printf("  %s Note added to %s — %q\n", ui.Success.Render("✓"), ui.Accent.Render(fmt.Sprintf("#%d", id)), preview)
	fmt.Println()
	return nil
}

func runTodoShow(_ *cobra.Command, args []string) error {
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
	t, err := ts.GetWithNotes(id)
	if err != nil {
		return err
	}

	printTodoDetail(*t)
	return nil
}

// printTodoDetail renders a full detail card for a single todo including body and notes.
func printTodoDetail(t todo.Todo) {
	now := time.Now()

	fmt.Println()

	// Header: ID, priority icon, title
	idStr := ui.Muted.Render(fmt.Sprintf("#%d", t.ID))
	prio := todo.PriorityIcon(t.Priority)
	fmt.Printf("  %s %s  %s\n", idStr, prio, ui.Accent.Render(t.Title))

	// Details row: schedule, priority label, due date
	details := fmt.Sprintf("  Schedule: %s  Priority: %s",
		todo.ScheduleLabel(t.Schedule),
		todo.PriorityLabel(t.Priority),
	)
	if t.DueDate != nil {
		details += fmt.Sprintf("  Due: %s", t.DueDate.Format("Jan 2"))
	}
	fmt.Println(ui.Muted.Render(details))

	// Project and tags (if set)
	if t.ProjectPath != nil || len(t.Tags) > 0 {
		extra := "  "
		if t.ProjectPath != nil {
			extra += fmt.Sprintf("Project: %s  ", filepath.Base(*t.ProjectPath))
		}
		if len(t.Tags) > 0 {
			extra += fmt.Sprintf("Tags: %s", strings.Join(t.Tags, ", "))
		}
		fmt.Println(ui.Muted.Render(extra))
	}

	// Timestamps
	fmt.Println()
	created := todoTimeAgo(t.CreatedAt, now)
	updated := todoTimeAgo(t.UpdatedAt, now)
	fmt.Printf("  %s\n", ui.Muted.Render(fmt.Sprintf("Created %s  Updated %s", created, updated)))

	// Body (initial context from --note on add)
	if t.Body != "" {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  Body:"))
		fmt.Printf("    %s\n", t.Body)
	}

	// Notes from todo_notes
	if len(t.Notes) > 0 {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  Notes:"))
		for _, n := range t.Notes {
			ts := n.CreatedAt.Format("2006-01-02 15:04")
			fmt.Printf("    %s  %s\n", ui.Muted.Render(ts), n.Body)
		}
	}

	fmt.Println()
}

// todoTimeAgo returns a human-readable relative time string.
func todoTimeAgo(t time.Time, now time.Time) string {
	d := now.Sub(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case d < 24*time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	default:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
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

func runTodoStats(_ *cobra.Command, _ []string) error {
	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	ps := proj.NewStore(db.Conn())

	var projectPath *string
	if todoStatsProjectFlag != "" {
		projectPath, err = resolveTodoProject(ps, todoStatsProjectFlag)
		if err != nil {
			return err
		}
	}

	now := time.Now()
	stats, err := todo.GetStats(db.Conn(), projectPath, now)
	if err != nil {
		return fmt.Errorf("computing stats: %w", err)
	}

	printTodoStats(stats, projectPath)
	return nil
}

func printTodoStats(stats *todo.Stats, projectPath *string) {
	ui.Puts("")
	ui.Puts(ui.Title.Render("  Task Stats"))
	ui.Puts("")

	if stats.CompletedMonth == 0 && stats.CompletedWeek == 0 && stats.Streak == 0 {
		ui.Puts(ui.Muted.Render("  No completions yet. Knock one out: ") + ui.Accent.Render("mine todo done <id>"))
		ui.Puts("")
		return
	}

	// Streak line: "5 days (longest: 12)" or "1 day (longest: 1)".
	streakStr := fmt.Sprintf("%d day", stats.Streak)
	if stats.Streak != 1 {
		streakStr += "s"
	}
	if stats.LongestStreak > 0 {
		streakStr += fmt.Sprintf(" %s (longest: %d)", ui.IconFire, stats.LongestStreak)
	}
	ui.Kv("Streak", streakStr)
	ui.Kv("This week", fmt.Sprintf("%d completed", stats.CompletedWeek))
	ui.Kv("This month", fmt.Sprintf("%d completed", stats.CompletedMonth))

	if stats.AvgClose > 0 {
		days := stats.AvgClose.Hours() / 24
		ui.Kv("Avg close", fmt.Sprintf("%.1f days", days))
	}

	if stats.HasFocusData && stats.TotalFocus > 0 {
		h := int(stats.TotalFocus.Hours())
		m := int(stats.TotalFocus.Minutes()) % 60
		if h > 0 {
			ui.Kv("Focus time", fmt.Sprintf("%dh %dm", h, m))
		} else {
			ui.Kv("Focus time", fmt.Sprintf("%dm", m))
		}
	}

	// Per-project breakdown only when not scoped to a single project.
	if projectPath == nil && len(stats.ByProject) > 0 {
		ui.Puts("")
		ui.Puts(ui.Muted.Render("  By project:"))
		for _, p := range stats.ByProject {
			avgStr := ""
			if p.AvgClose > 0 {
				days := p.AvgClose.Hours() / 24
				avgStr = fmt.Sprintf("  avg %.1fd", days)
			}
			ui.Putsf("    %-14s %3d open  %3d done%s",
				p.Name, p.Open, p.Completed, avgStr)
		}
	}

	ui.Puts("")
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
