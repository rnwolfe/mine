package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/proj"
	"github.com/rnwolfe/mine/internal/store"
	"github.com/rnwolfe/mine/internal/todo"
	"github.com/rnwolfe/mine/internal/tui"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

var digSimple bool
var digTodoID int

var digCmd = &cobra.Command{
	Use:   "dig [duration]",
	Short: "Start a deep work / focus session",
	Long: `Start a timed focus session. Tracks your deep work streaks.

Duration examples: 25m, 1h, 45m, 2h
Default: 25m (pomodoro)

In an interactive terminal, launches a full-screen focus timer.
Use --simple to keep the original inline progress output.

Use --todo <id> to link the session to a specific task. When run inside a
registered project, a task picker is offered automatically.

Keyboard shortcuts (full-screen mode):
  q / Ctrl+C   End session early`,
	RunE: hook.Wrap("dig", runDig),
}

var digStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "View your deep work stats",
	RunE:  hook.Wrap("dig.stats", runDigStats),
}

func init() {
	rootCmd.AddCommand(digCmd)
	digCmd.AddCommand(digStatsCmd)
	digCmd.Flags().BoolVar(&digSimple, "simple", false, "Use simple inline timer output instead of full-screen TUI")
	digCmd.Flags().IntVar(&digTodoID, "todo", 0, "Link session to a task by ID (e.g. --todo 12)")
}

func runDig(_ *cobra.Command, args []string) error {
	duration := 25 * time.Minute
	label := "25m"

	if len(args) > 0 {
		d, err := time.ParseDuration(args[0])
		if err != nil {
			return fmt.Errorf("invalid duration %q — try: 25m, 1h, 45m", args[0])
		}
		duration = d
		label = args[0]
	}

	// Resolve optional linked todo.
	var linkedTodoID *int
	var taskTitle string

	if digTodoID > 0 {
		// Validate the todo exists before starting the session.
		db, err := store.Open()
		if err != nil {
			return err
		}
		t, err := todo.NewStore(db.Conn()).Get(digTodoID)
		db.Close()
		if err != nil {
			return fmt.Errorf("todo #%d not found", digTodoID)
		}
		id := digTodoID
		linkedTodoID = &id
		taskTitle = t.Title
	} else if tui.IsTTY() {
		// Offer a task picker when inside a project with open tasks.
		picked, err := pickProjectTask()
		if err == nil && picked != nil {
			linkedTodoID = &picked.ID
			taskTitle = picked.Title
		}
	}

	// Use full-screen TUI when connected to a terminal and --simple not set.
	if tui.IsTTY() && !digSimple {
		return runDigTUI(duration, label, linkedTodoID, taskTitle)
	}

	return runDigSimple(duration, label, linkedTodoID, taskTitle)
}

// todoPickerItem adapts a todo.Todo for use in the tui.Picker.
type todoPickerItem struct {
	t todo.Todo
}

func (i todoPickerItem) FilterValue() string { return i.t.Title }
func (i todoPickerItem) Title() string {
	return fmt.Sprintf("#%d  %s  %s", i.t.ID, todo.PriorityIcon(i.t.Priority), i.t.Title)
}
func (i todoPickerItem) Description() string {
	return fmt.Sprintf("%s · %s", todo.PriorityLabel(i.t.Priority), todo.ScheduleLabel(i.t.Schedule))
}

// pickProjectTask shows a TUI picker for open tasks in the current project.
// Returns nil when the user skips, cancels, or no project/tasks are available.
func pickProjectTask() (*todo.Todo, error) {
	db, err := store.Open()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	ps := proj.NewStore(db.Conn())
	p, err := ps.FindForCWD()
	if err != nil || p == nil {
		return nil, nil
	}

	ts := todo.NewStore(db.Conn())
	todos, err := ts.List(todo.ListOptions{ProjectPath: &p.Path})
	if err != nil || len(todos) == 0 {
		return nil, nil
	}

	items := make([]tui.Item, len(todos))
	for i, t := range todos {
		items[i] = todoPickerItem{t}
	}

	chosen, err := tui.Run(items,
		tui.WithTitle("Pick a task to focus on (Esc to skip)"),
		tui.WithPrompt("task> "),
	)
	if err != nil || chosen == nil {
		return nil, nil
	}

	item := chosen.(todoPickerItem)
	return &item.t, nil
}

func runDigTUI(duration time.Duration, label string, todoID *int, taskTitle string) error {
	result, err := tui.RunDig(duration, label, taskTitle)
	if err != nil {
		return err
	}

	fmt.Println()
	if result.Completed {
		fmt.Printf("  %s %s of focused work. Nice.\n", ui.IconGem, ui.Accent.Render(label))
		recordDigSession(duration, todoID, true)
		if todoID != nil {
			maybeMarkTodoDone(*todoID, taskTitle)
		}
	} else if result.Canceled {
		if result.Elapsed >= 5*time.Minute {
			recordDigSession(result.Elapsed, todoID, false)
			ui.Ok(fmt.Sprintf("Session ended early after %s. Still counts! Logged.", result.Elapsed))
			if todoID != nil {
				maybeMarkTodoDone(*todoID, taskTitle)
			}
		} else {
			fmt.Printf("  %s Session ended early after %s\n", ui.IconMine, result.Elapsed)
			fmt.Println(ui.Muted.Render("  Too short to count. Try again!"))
		}
	}
	fmt.Println()
	return nil
}

func runDigSimple(duration time.Duration, label string, todoID *int, taskTitle string) error {
	fmt.Println()
	fmt.Printf("  %s Deep work session: %s\n", ui.IconDig, ui.Accent.Render(label))
	if taskTitle != "" {
		fmt.Printf("  %s\n", ui.Muted.Render("Focusing on: "+taskTitle))
	}
	fmt.Println(ui.Muted.Render("  Focus. You've got this. Ctrl+C to end early."))
	fmt.Println()

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	start := time.Now()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			elapsed := time.Since(start).Round(time.Second)
			fmt.Println()
			fmt.Printf("\n  %s Session ended early after %s\n", ui.IconMine, elapsed)
			if elapsed >= 5*time.Minute {
				recordDigSession(elapsed, todoID, false)
				ui.Ok(fmt.Sprintf("Still counts! %s logged.", elapsed))
				if todoID != nil {
					maybeMarkTodoDone(*todoID, taskTitle)
				}
			} else {
				fmt.Println(ui.Muted.Render("  Too short to count. Try again!"))
			}
			fmt.Println()
			return nil

		case <-ticker.C:
			elapsed := time.Since(start)
			remaining := duration - elapsed
			if remaining <= 0 {
				fmt.Printf("\r  %s  ", ui.Success.Render("Done!"))
				fmt.Println()
				fmt.Println()
				fmt.Printf("  %s %s of focused work. Nice.\n", ui.IconGem, ui.Accent.Render(label))
				recordDigSession(duration, todoID, true)
				if todoID != nil {
					maybeMarkTodoDone(*todoID, taskTitle)
				}
				fmt.Println()
				return nil
			}

			mins := int(remaining.Minutes())
			secs := int(remaining.Seconds()) % 60
			bar := progressBar(elapsed, duration, 30)
			fmt.Printf("\r  %s %s %02d:%02d remaining", ui.IconDig, bar, mins, secs)
		}
	}
}

// maybeMarkTodoDone prompts the user to mark the linked task complete after a session.
func maybeMarkTodoDone(todoID int, taskTitle string) {
	maybeMarkTodoDoneWithReader(bufio.NewReader(os.Stdin), todoID, taskTitle)
}

// maybeMarkTodoDoneWithReader is the testable entry point for the completion prompt.
func maybeMarkTodoDoneWithReader(reader *bufio.Reader, todoID int, taskTitle string) {
	fmt.Printf("\n  Mark %s done? (y/n): ", ui.Accent.Render(fmt.Sprintf("#%d", todoID)))
	if taskTitle != "" {
		fmt.Printf("%s\n  > ", ui.Muted.Render(taskTitle))
	}

	answer, _ := reader.ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))

	if answer == "y" || answer == "yes" {
		db, err := store.Open()
		if err != nil {
			fmt.Printf("  %s Could not open store: %v\n", ui.IconMine, err)
			return
		}
		defer db.Close()

		ts := todo.NewStore(db.Conn())
		if err := ts.Complete(todoID); err != nil {
			fmt.Printf("  %s Could not mark done: %v\n", ui.IconMine, err)
			return
		}
		fmt.Printf("  %s Marked %s done.\n", ui.Success.Render("✓"), ui.Accent.Render(fmt.Sprintf("#%d", todoID)))
	}
}

func progressBar(elapsed, total time.Duration, width int) string {
	pct := float64(elapsed) / float64(total)
	filled := int(pct * float64(width))
	if filled > width {
		filled = width
	}

	bar := ui.Success.Render(repeatChar('█', filled))
	bar += ui.Muted.Render(repeatChar('░', width-filled))
	return bar
}

func repeatChar(ch rune, n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]rune, n)
	for i := range b {
		b[i] = ch
	}
	return string(b)
}

func recordDigSession(duration time.Duration, todoID *int, completed bool) {
	db, err := store.Open()
	if err != nil {
		return
	}
	defer db.Close()

	mins := int(duration.Minutes())
	secs := int(duration.Seconds())
	today := time.Now().Format("2006-01-02")
	comp := 0
	if completed {
		comp = 1
	}

	// Insert into dig_sessions table.
	if _, err := db.Conn().Exec(
		`INSERT INTO dig_sessions (todo_id, duration_secs, completed, ended_at) VALUES (?, ?, ?, CURRENT_TIMESTAMP)`,
		todoID, secs, comp,
	); err != nil {
		fmt.Printf("  %s Warning: could not record session: %v\n", ui.IconMine, err)
		return
	}

	// Update streak
	var lastDate string
	var current, longest int
	err = db.Conn().QueryRow(`SELECT last_date, current, longest FROM streaks WHERE name = 'dig'`).Scan(&lastDate, &current, &longest)

	if err != nil {
		// First session ever
		db.Conn().Exec(`INSERT INTO streaks (name, current, longest, last_date) VALUES ('dig', 1, 1, ?)`, today)
	} else {
		yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
		if lastDate == today {
			// Already logged today, don't increment streak
		} else if lastDate == yesterday {
			current++
			if current > longest {
				longest = current
			}
			db.Conn().Exec(`UPDATE streaks SET current = ?, longest = ?, last_date = ? WHERE name = 'dig'`, current, longest, today)
		} else {
			// Streak broken
			db.Conn().Exec(`UPDATE streaks SET current = 1, last_date = ? WHERE name = 'dig'`, today)
		}
	}

	// Store total minutes in KV
	var totalMins int
	db.Conn().QueryRow(`SELECT CAST(value AS INTEGER) FROM kv WHERE key = 'dig_total_mins'`).Scan(&totalMins)
	totalMins += mins
	db.Conn().Exec(`INSERT OR REPLACE INTO kv (key, value, updated_at) VALUES ('dig_total_mins', ?, CURRENT_TIMESTAMP)`, fmt.Sprintf("%d", totalMins))

	ui.Ok(fmt.Sprintf("%dm logged. %dh %dm total deep work.", mins, totalMins/60, totalMins%60))
}

func runDigStats(_ *cobra.Command, _ []string) error {
	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	fmt.Println()
	fmt.Println(ui.Title.Render("  Deep Work Stats"))
	fmt.Println()

	// Streak
	var current, longest int
	var lastDate string
	err = db.Conn().QueryRow(`SELECT current, longest, last_date FROM streaks WHERE name = 'dig'`).Scan(&current, &longest, &lastDate)
	if err != nil {
		fmt.Println(ui.Muted.Render("  No sessions yet. Start with: mine dig"))
		fmt.Println()
		return nil
	}

	ui.Kv("Streak", fmt.Sprintf("%d days %s", current, ui.IconFire))
	ui.Kv("Best", fmt.Sprintf("%d days", longest))

	// Total time from KV (legacy)
	var totalMins int
	db.Conn().QueryRow(`SELECT CAST(value AS INTEGER) FROM kv WHERE key = 'dig_total_mins'`).Scan(&totalMins)
	ui.Kv("Total", fmt.Sprintf("%dh %dm", totalMins/60, totalMins%60))
	ui.Kv("Last", lastDate)

	// Session table stats
	var sessionCount int
	db.Conn().QueryRow(`SELECT COUNT(*) FROM dig_sessions`).Scan(&sessionCount)
	if sessionCount > 0 {
		ui.Kv("Sessions", fmt.Sprintf("%d", sessionCount))

		// Tasks with linked sessions
		var linkedTasks int
		db.Conn().QueryRow(`SELECT COUNT(DISTINCT todo_id) FROM dig_sessions WHERE todo_id IS NOT NULL`).Scan(&linkedTasks)
		if linkedTasks > 0 {
			ui.Kv("Tasks focused", fmt.Sprintf("%d", linkedTasks))
		}
	}

	fmt.Println()
	return nil
}
