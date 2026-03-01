package cmd

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/proj"
	"github.com/rnwolfe/mine/internal/store"
	"github.com/rnwolfe/mine/internal/todo"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

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

var todoRecurringCmd = &cobra.Command{
	Use:   "recurring",
	Short: "List all active recurring task definitions",
	Long: `List all open tasks that have a recurrence frequency set.

Recurring tasks automatically spawn a new occurrence when completed.
Frequencies: day, weekday, week, month.`,
	RunE: hook.Wrap("todo.recurring", runTodoRecurring),
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

func runTodoRecurring(_ *cobra.Command, _ []string) error {
	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	ts := todo.NewStore(db.Conn())
	todos, err := ts.ListRecurring()
	if err != nil {
		return fmt.Errorf("listing recurring todos: %w", err)
	}

	if len(todos) == 0 {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No recurring tasks yet."))
		fmt.Printf("  Create one: %s\n", ui.Accent.Render(`mine todo add "Review PRs" --every week`))
		fmt.Println()
		return nil
	}

	fmt.Println()
	for _, t := range todos {
		id := lipgloss.NewStyle().Width(todo.ColWidthID).Render(ui.Muted.Render(fmt.Sprintf("#%d", t.ID)))
		prio := todo.FormatPriorityIcon(t.Priority)
		freq := ui.Muted.Render("↻ " + todo.RecurrenceLabel(t.Recurrence))

		line := fmt.Sprintf("  %s %s %s  %s", id, prio, freq, t.Title)

		if t.DueDate != nil {
			line += ui.Muted.Render(fmt.Sprintf(" (next: %s)", t.DueDate.Format("Jan 2")))
		}
		if t.ProjectPath != nil {
			projName := filepath.Base(*t.ProjectPath)
			line += ui.Muted.Render(fmt.Sprintf(" @%s", projName))
		}
		fmt.Println(line)
	}
	fmt.Println()
	fmt.Println(ui.Muted.Render(fmt.Sprintf("  %d recurring task(s)", len(todos))))
	fmt.Println()

	return nil
}
