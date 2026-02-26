package cmd

import (
	"fmt"
	"time"

	"github.com/rnwolfe/mine/internal/grow"
	"github.com/rnwolfe/mine/internal/store"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

func runGrowDashboard(_ *cobra.Command, _ []string) error {
	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	gs := grow.NewStore(db.Conn())
	now := time.Now()

	ui.Puts("")
	ui.Puts(ui.Title.Render("  " + ui.IconGrow + " Growth Dashboard"))
	ui.Puts("")

	// Streak
	streak, err := gs.GetStreak(now)
	if err != nil {
		return fmt.Errorf("computing streak: %w", err)
	}
	if streak.Current > 0 {
		streakStr := fmt.Sprintf("%d day", streak.Current)
		if streak.Current != 1 {
			streakStr += "s"
		}
		if streak.Longest > streak.Current {
			streakStr += fmt.Sprintf(" %s (longest: %d)", ui.IconFire, streak.Longest)
		} else {
			streakStr += " " + ui.IconFire
		}
		ui.Kv("Streak", streakStr)
	} else {
		ui.Kv("Streak", ui.Muted.Render("none — log an activity to start!"))
	}

	// Active goals
	goals, err := gs.ListGoals()
	if err != nil {
		return fmt.Errorf("listing goals: %w", err)
	}
	ui.Kv("Active goals", fmt.Sprintf("%d", len(goals)))

	// Skills summary
	skills, err := gs.ListSkills()
	if err != nil {
		return fmt.Errorf("listing skills: %w", err)
	}
	if len(skills) > 0 {
		ui.Kv("Skills", fmt.Sprintf("%d tracked", len(skills)))
	}

	ui.Puts("")

	// Show goals if any
	if len(goals) > 0 {
		ui.Puts(ui.Muted.Render("  Goals:"))
		for _, g := range goals {
			printGoalLine(g)
		}
		ui.Puts("")
	}

	// Show top skills (up to 5)
	if len(skills) > 0 {
		ui.Puts(ui.Muted.Render("  Top skills:"))
		shown := skills
		if len(shown) > 5 {
			shown = shown[:5]
		}
		for _, sk := range shown {
			ui.Putsf("    %-20s %s", sk.Name, grow.SkillLevelDots(sk.Level))
		}
		ui.Puts("")
	}

	// Tips
	if len(goals) == 0 {
		ui.Tip("`mine grow goal add \"Learn Rust\" --target 50 --unit hrs` to set a goal.")
	} else if streak.Current == 0 {
		ui.Tip("`mine grow log --minutes 30` to log today's activity and start a streak.")
	}

	ui.Puts("")
	return nil
}

func runGrowReview(_ *cobra.Command, _ []string) error {
	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	gs := grow.NewStore(db.Conn())
	now := time.Now()

	fmt.Println()
	ui.Puts(ui.Title.Render("  " + ui.IconGrow + " Growth Review"))
	fmt.Println()

	// Streak
	streak, err := gs.GetStreak(now)
	if err != nil {
		return fmt.Errorf("computing streak: %w", err)
	}
	streakStr := fmt.Sprintf("%d days (longest: %d)", streak.Current, streak.Longest)
	ui.Kv("Streak", streakStr)

	// Activities this week
	weekStart := growStartOfWeek(now.UTC())
	weekActivities, err := gs.ListActivities(weekStart)
	if err != nil {
		return fmt.Errorf("listing weekly activities: %w", err)
	}
	weekMins := totalMinutes(weekActivities)
	ui.Kv("This week", fmt.Sprintf("%d activities, %s", len(weekActivities), formatMins(weekMins)))

	// Activities this month
	nowUTC := now.UTC()
	monthStart := time.Date(nowUTC.Year(), nowUTC.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthActivities, err := gs.ListActivities(monthStart)
	if err != nil {
		return fmt.Errorf("listing monthly activities: %w", err)
	}
	monthMins := totalMinutes(monthActivities)
	ui.Kv("This month", fmt.Sprintf("%d activities, %s", len(monthActivities), formatMins(monthMins)))

	// Goals
	goals, err := gs.ListGoals()
	if err != nil {
		return fmt.Errorf("listing goals: %w", err)
	}
	if len(goals) > 0 {
		fmt.Println()
		ui.Puts(ui.Muted.Render("  Active goals:"))
		for _, g := range goals {
			printGoalLine(g)
		}
	}

	// Recent activities (this week)
	if len(weekActivities) > 0 {
		fmt.Println()
		ui.Puts(ui.Muted.Render("  This week's activities:"))
		limit := weekActivities
		if len(limit) > 10 {
			limit = limit[:10]
		}
		for _, a := range limit {
			dateStr := a.CreatedAt.Format("Mon Jan 2")
			note := a.Note
			if note == "" && a.Skill != "" {
				note = a.Skill
			}
			if note == "" {
				note = ui.Muted.Render("(no note)")
			}
			minStr := ""
			if a.Minutes > 0 {
				minStr = ui.Muted.Render(fmt.Sprintf(" [%dm]", a.Minutes))
			}
			fmt.Printf("    %s  %s%s\n", ui.Muted.Render(dateStr), note, minStr)
		}
	}

	fmt.Println()
	return nil
}

// growStartOfWeek returns the Monday at 00:00:00 of the week containing t.
func growStartOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	daysBack := weekday - 1
	monday := t.AddDate(0, 0, -daysBack)
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, t.Location())
}

// totalMinutes sums minutes across a slice of activities.
func totalMinutes(activities []grow.Activity) int {
	total := 0
	for _, a := range activities {
		total += a.Minutes
	}
	return total
}

// formatMins formats a minute count as "2h 15m" or "45m".
func formatMins(mins int) string {
	if mins == 0 {
		return "0m"
	}
	h := mins / 60
	m := mins % 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}
