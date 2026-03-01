package cmd

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/rnwolfe/mine/internal/todo"
	"github.com/rnwolfe/mine/internal/tui"
	"github.com/rnwolfe/mine/internal/ui"
)

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
				spawnedID, spawnedDue, err := ts.Complete(a.ID)
				if err != nil {
					failedActions = append(failedActions, fmt.Sprintf("complete #%d: %v", a.ID, err))
				} else if spawnedID > 0 {
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
			}
		case "delete":
			if err := ts.Delete(a.ID); err != nil {
				failedActions = append(failedActions, fmt.Sprintf("delete #%d: %v", a.ID, err))
			}
		case "add":
			if strings.TrimSpace(a.Text) != "" {
				if _, err := ts.Add(a.Text, "", todo.PrioMedium, nil, nil, a.ProjectPath, todo.ScheduleLater, todo.RecurrenceNone); err != nil {
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

		id := lipgloss.NewStyle().Width(todo.ColWidthID).Render(ui.Muted.Render(fmt.Sprintf("#%d", t.ID)))
		prio := todo.FormatPriorityIcon(t.Priority)
		title := t.Title
		if t.Done {
			title = ui.Muted.Render(title)
		}

		schedTag := todo.FormatScheduleTag(t.Schedule)
		recurTag := ""
		if t.Recurrence != "" && t.Recurrence != todo.RecurrenceNone {
			recurTag = " " + ui.Muted.Render("↻")
		}
		line := fmt.Sprintf("  %s %s %s %s %s%s", marker, id, prio, schedTag, title, recurTag)

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
