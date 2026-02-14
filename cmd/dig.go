package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rnwolfe/mine/internal/store"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

var digCmd = &cobra.Command{
	Use:   "dig [duration]",
	Short: "Start a deep work / focus session",
	Long: `Start a timed focus session. Tracks your deep work streaks.

Duration examples: 25m, 1h, 45m, 2h
Default: 25m (pomodoro)`,
	RunE: runDig,
}

var digStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "View your deep work stats",
	RunE:  runDigStats,
}

func init() {
	rootCmd.AddCommand(digCmd)
	digCmd.AddCommand(digStatsCmd)
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

	fmt.Println()
	fmt.Printf("  %s Deep work session: %s\n", ui.IconDig, ui.Accent.Render(label))
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
			fmt.Printf("\n  %s Session ended early after %s\n", ui.IconPick, elapsed)
			if elapsed >= 5*time.Minute {
				recordDigSession(elapsed)
				ui.Ok(fmt.Sprintf("Still counts! %s logged.", elapsed))
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
				recordDigSession(duration)
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

func recordDigSession(duration time.Duration) {
	db, err := store.Open()
	if err != nil {
		return
	}
	defer db.Close()

	mins := int(duration.Minutes())
	today := time.Now().Format("2006-01-02")

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

	// Total time
	var totalMins int
	db.Conn().QueryRow(`SELECT CAST(value AS INTEGER) FROM kv WHERE key = 'dig_total_mins'`).Scan(&totalMins)
	ui.Kv("Total", fmt.Sprintf("%dh %dm", totalMins/60, totalMins%60))
	ui.Kv("Last", lastDate)

	fmt.Println()
	return nil
}
