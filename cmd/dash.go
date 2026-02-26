package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/proj"
	"github.com/rnwolfe/mine/internal/store"
	"github.com/rnwolfe/mine/internal/todo"
	"github.com/rnwolfe/mine/internal/tui"
	"github.com/spf13/cobra"
)

var dashCmd = &cobra.Command{
	Use:   "dash",
	Short: "Open the interactive dashboard",
	Long: `Opens the full TUI dashboard showing todos, focus stats, and project context.

Keyboard shortcuts:
  t          Open the full todo TUI
  d          Start a 25-minute dig focus session
  r          Refresh all panel data
  q / Ctrl+C Quit`,
	RunE: hook.Wrap("dash", runDash),
}

func init() {
	rootCmd.AddCommand(dashCmd)
}

// runDashTUI runs the interactive dashboard loop. It handles re-launching
// after the todo TUI returns or after a dig session ends.
func runDashTUI() error {
	db, err := store.Open()
	if err != nil {
		return fmt.Errorf("opening store: %w", err)
	}
	defer db.Close()

	for {
		action, err := tui.RunDash(db.Conn())
		if err != nil {
			return err
		}
		switch action {
		case tui.DashActionOpenTodo:
			if err := openTodoFromDash(db); err != nil {
				return err
			}
		case tui.DashActionStartDig:
			sessionStart := time.Now()
			result, err := tui.RunDig(25*time.Minute, "25m", "")
			if err != nil {
				return err
			}
			if result.Completed || (result.Canceled && result.Elapsed >= 5*time.Minute) {
				recordDigSession(result.Elapsed, nil, result.Completed, sessionStart)
			}
		default:
			return nil
		}
	}
}

// runDash is the Cobra handler for `mine dash`.
func runDash(_ *cobra.Command, _ []string) error {
	return runDashTUI()
}

// openTodoFromDash launches the full todo TUI and applies the resulting actions.
func openTodoFromDash(db *store.DB) error {
	ps := proj.NewStore(db.Conn())
	p, _ := ps.FindForCWD()

	var projPath *string
	if p != nil {
		projPath = &p.Path
	}

	ts := todo.NewStore(db.Conn())
	todos, err := ts.List(todo.ListOptions{ProjectPath: projPath})
	if err != nil {
		return fmt.Errorf("loading todos: %w", err)
	}

	actions, err := tui.RunTodo(todos, projPath, false)
	if err != nil {
		return fmt.Errorf("todo tui: %w", err)
	}

	var failedActions []string
	for _, a := range actions {
		switch a.Type {
		case "toggle":
			if _, _, err := ts.Complete(a.ID); err != nil {
				failedActions = append(failedActions, fmt.Sprintf("toggle #%d: %v", a.ID, err))
			}
		case "delete":
			if err := ts.Delete(a.ID); err != nil {
				failedActions = append(failedActions, fmt.Sprintf("delete #%d: %v", a.ID, err))
			}
		case "add":
			if a.Text != "" {
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
		return fmt.Errorf("some todo actions failed: %s", strings.Join(failedActions, "; "))
	}

	return nil
}
