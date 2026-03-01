package cmd

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/store"
	"github.com/rnwolfe/mine/internal/todo"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

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

	schedLabel := todo.FormatScheduleTag(schedule)
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
	fmt.Printf("  %s %s %s\n", idStr, prio, ui.Accent.Render(t.Title))

	// Details row: schedule, priority label, due date
	details := fmt.Sprintf("  Schedule: %s  Priority: %s",
		todo.ScheduleLabel(t.Schedule),
		todo.PriorityLabel(t.Priority),
	)
	if t.DueDate != nil {
		details += fmt.Sprintf("  Due: %s", t.DueDate.Format("Jan 2"))
	}
	if t.Recurrence != "" && t.Recurrence != todo.RecurrenceNone {
		details += fmt.Sprintf("  Recurrence: ↻ %s", todo.RecurrenceLabel(t.Recurrence))
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
