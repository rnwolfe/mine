package cmd

import (
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
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

var todoNextCmd = &cobra.Command{
	Use:   "next [n]",
	Short: "Show the highest-urgency tasks — what should you work on?",
	Long: `Surface the most urgent open tasks using a weighted urgency score.

Urgency accounts for: overdue status, schedule bucket, priority, task age,
and whether the task belongs to the current project.

Someday tasks are always excluded. Use 'mine todo next 3' to see the top 3.`,
	Args: cobra.MaximumNArgs(1),
	RunE: hook.Wrap("todo.next", runTodoNext),
}

func runTodoNext(_ *cobra.Command, args []string) error {
	count := 1
	if len(args) > 0 {
		n, err := strconv.Atoi(args[0])
		if err != nil || n <= 0 {
			return fmt.Errorf("%q is not a valid count — use %s",
				args[0], ui.Accent.Render("mine todo next [n]"))
		}
		count = n
	}

	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	ps := proj.NewStore(db.Conn())
	projectPath, err := resolveTodoProject(ps, "")
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	weights := urgencyWeightsFromConfig(cfg)

	// Capture now once so sorting and rendering use the same reference instant.
	now := time.Now()

	ts := todo.NewStore(db.Conn())
	todos, err := ts.List(todo.ListOptions{
		AllProjects:        false,
		ProjectPath:        projectPath,
		Sort:               todo.SortUrgency,
		CurrentProjectPath: projectPath,
		Weights:            &weights,
		ReferenceTime:      now,
	})
	if err != nil {
		return err
	}

	if len(todos) == 0 {
		fmt.Println()
		fmt.Println(ui.Success.Render("  " + ui.IconParty + " All clear! No open tasks."))
		fmt.Println()
		return nil
	}

	if count > len(todos) {
		count = len(todos)
	}

	fmt.Println()
	for rank, t := range todos[:count] {
		printTodoCard(t, rank+1, now, projectPath)
	}
	return nil
}

// cardMetaIndent is the number of spaces to indent metadata lines in a todo card,
// computed to align under the title text:
//   - "  " (2) + rank "%2d." (3) + " " (1) + prio emoji (2) + " " (1) + sched (2) + " " (1) = 12
const cardMetaIndent = "            "

// printTodoCard prints a detailed card for a single todo.
func printTodoCard(t todo.Todo, rank int, now time.Time, currentProjectPath *string) {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	rankStr := ui.Muted.Render(fmt.Sprintf("%2d.", rank))
	prio := todo.PriorityIcon(t.Priority)
	schedTag := todo.FormatScheduleTag(t.Schedule)
	title := ui.Accent.Render(t.Title)

	fmt.Printf("  %s %s %s %s\n", rankStr, prio, schedTag, title)

	// ID and priority label — indented to align under the title.
	fmt.Printf("%s%s  %s\n",
		cardMetaIndent,
		ui.Muted.Render(fmt.Sprintf("#%d", t.ID)),
		ui.Muted.Render(todo.PriorityLabel(t.Priority)+" priority"),
	)

	// Due date (if set)
	if t.DueDate != nil {
		due := *t.DueDate
		// Build dueDay in the same location as today to avoid timezone mismatches
		// when due dates are stored as date-only strings (parsed as UTC).
		dueDay := time.Date(due.Year(), due.Month(), due.Day(), 0, 0, 0, 0, now.Location())
		var dueStr string
		switch {
		case dueDay.Before(today):
			dueStr = ui.Error.Render("overdue: " + due.Format("Jan 2"))
		case dueDay.Equal(today):
			dueStr = ui.Warning.Render("due today!")
		default:
			dueStr = ui.Muted.Render("due " + due.Format("Mon, Jan 2"))
		}
		fmt.Printf("%s%s\n", cardMetaIndent, dueStr)
	}

	// Project: show only when the task belongs to a project other than the current one.
	if t.ProjectPath != nil && (currentProjectPath == nil || *t.ProjectPath != *currentProjectPath) {
		projName := filepath.Base(*t.ProjectPath)
		fmt.Printf("%s%s\n", cardMetaIndent, ui.Muted.Render("@"+projName))
	}

	// Tags
	if len(t.Tags) > 0 {
		fmt.Printf("%s%s\n", cardMetaIndent, ui.Muted.Render("["+strings.Join(t.Tags, ", ")+"]"))
	}

	// Age — only show if CreatedAt parsed successfully (non-zero) and todo is at least 1 day old.
	if !t.CreatedAt.IsZero() {
		age := int(today.Sub(
			time.Date(t.CreatedAt.Year(), t.CreatedAt.Month(), t.CreatedAt.Day(), 0, 0, 0, 0, now.Location()),
		).Hours() / 24)
		if age > 0 {
			fmt.Printf("%s%s\n", cardMetaIndent, ui.Muted.Render(fmt.Sprintf("%d day(s) old", age)))
		}
	}

	fmt.Println()
}

// urgencyWeightsFromConfig builds urgency weights from config, using defaults for any unset field.
func urgencyWeightsFromConfig(cfg *config.Config) todo.UrgencyWeights {
	w := todo.DefaultUrgencyWeights()
	u := cfg.Todo.Urgency
	if u.Overdue != nil {
		w.Overdue = *u.Overdue
	}
	if u.ScheduleToday != nil {
		w.ScheduleToday = *u.ScheduleToday
	}
	if u.ScheduleSoon != nil {
		w.ScheduleSoon = *u.ScheduleSoon
	}
	if u.ScheduleLater != nil {
		w.ScheduleLater = *u.ScheduleLater
	}
	if u.PriorityCrit != nil {
		w.PriorityCrit = *u.PriorityCrit
	}
	if u.PriorityHigh != nil {
		w.PriorityHigh = *u.PriorityHigh
	}
	if u.PriorityMed != nil {
		w.PriorityMed = *u.PriorityMed
	}
	if u.PriorityLow != nil {
		w.PriorityLow = *u.PriorityLow
	}
	if u.AgeCap != nil {
		w.AgeCap = *u.AgeCap
	}
	if u.ProjectBoost != nil {
		w.ProjectBoost = *u.ProjectBoost
	}
	return w
}
