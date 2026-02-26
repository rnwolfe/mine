package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rnwolfe/mine/internal/config"
	"github.com/rnwolfe/mine/internal/grow"
	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/store"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

// Flags for grow commands.
var (
	growGoalDeadline string
	growGoalTarget   float64
	growGoalUnit     string

	growLogMinutes int
	growLogGoal    int
	growLogSkill   string

	growSkillCategory string
)

var growCmd = &cobra.Command{
	Use:   "grow",
	Short: "Track career growth, learning streaks, and skill development",
	Long: `Monitor your learning journey: set goals, log activities, build streaks,
and track self-assessed skill levels — all local, all yours.

Run ` + "`mine grow`" + ` to see your dashboard: active goals, streak, and top skills.`,
	RunE: hook.Wrap("grow", runGrowDashboard),
}

var growGoalCmd = &cobra.Command{
	Use:   "goal",
	Short: "Manage learning goals",
}

var growGoalAddCmd = &cobra.Command{
	Use:   "add <title>",
	Short: "Add a learning or career goal",
	Args:  cobra.MinimumNArgs(1),
	RunE:  hook.Wrap("grow.goal.add", runGrowGoalAdd),
}

var growGoalListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show active goals with progress",
	RunE:  hook.Wrap("grow.goal.list", runGrowGoalList),
}

var growGoalDoneCmd = &cobra.Command{
	Use:     "done <id>",
	Aliases: []string{"complete"},
	Short:   "Mark a goal complete",
	Args:    cobra.ExactArgs(1),
	RunE:    hook.Wrap("grow.goal.done", runGrowGoalDone),
}

var growLogCmd = &cobra.Command{
	Use:   "log [note]",
	Short: "Log a learning activity",
	Long: `Record time spent on learning. Optionally link to a goal and tag a skill.

Examples:
  mine grow log "Read The Rust Book ch. 3" --minutes 45 --goal 1 --skill Rust
  mine grow log --minutes 30 --skill "System Design"`,
	RunE: hook.Wrap("grow.log", runGrowLog),
}

var growStreakCmd = &cobra.Command{
	Use:   "streak",
	Short: "Show current and longest learning streak",
	RunE:  hook.Wrap("grow.streak", runGrowStreak),
}

var growSkillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Show self-assessed skill levels",
	RunE:  hook.Wrap("grow.skills", runGrowSkills),
}

var growSkillsSetCmd = &cobra.Command{
	Use:   "set <name> <1-5>",
	Short: "Set self-assessed level for a skill",
	Args:  cobra.ExactArgs(2),
	RunE:  hook.Wrap("grow.skills.set", runGrowSkillsSet),
}

var growReviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Weekly/monthly summary: activities, goal progress, streak",
	RunE:  hook.Wrap("grow.review", runGrowReview),
}

func init() {
	// Build command tree
	growGoalCmd.AddCommand(growGoalAddCmd)
	growGoalCmd.AddCommand(growGoalListCmd)
	growGoalCmd.AddCommand(growGoalDoneCmd)
	growSkillsCmd.AddCommand(growSkillsSetCmd)

	growCmd.AddCommand(growGoalCmd)
	growCmd.AddCommand(growLogCmd)
	growCmd.AddCommand(growStreakCmd)
	growCmd.AddCommand(growSkillsCmd)
	growCmd.AddCommand(growReviewCmd)

	// Flags for goal add
	growGoalAddCmd.Flags().StringVar(&growGoalDeadline, "deadline", "", "Deadline date (YYYY-MM-DD)")
	growGoalAddCmd.Flags().Float64Var(&growGoalTarget, "target", 0, "Target value (e.g. 50 for 50 hrs)")
	growGoalAddCmd.Flags().StringVar(&growGoalUnit, "unit", "hrs", "Unit for target (e.g. hrs, sessions)")

	// Flags for log
	growLogCmd.Flags().IntVar(&growLogMinutes, "minutes", 0, "Minutes spent on this activity")
	growLogCmd.Flags().IntVar(&growLogGoal, "goal", 0, "Link to a goal ID")
	growLogCmd.Flags().StringVar(&growLogSkill, "skill", "", "Skill tag for this activity")

	// Flags for skills set
	growSkillsSetCmd.Flags().StringVar(&growSkillCategory, "category", "general", "Category for the skill")
}

func runGrowGoalAdd(_ *cobra.Command, args []string) error {
	title := strings.Join(args, " ")

	var deadline *time.Time
	if growGoalDeadline != "" {
		t, err := time.Parse("2006-01-02", growGoalDeadline)
		if err != nil {
			return fmt.Errorf("invalid deadline %q — expected %s",
				growGoalDeadline, ui.Accent.Render("YYYY-MM-DD"))
		}
		deadline = &t
	}

	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	gs := grow.NewStore(db.Conn())
	id, err := gs.AddGoal(title, deadline, growGoalTarget, growGoalUnit)
	if err != nil {
		return fmt.Errorf("adding goal: %w", err)
	}

	fmt.Printf("  %s Goal added %s\n", ui.Success.Render("✓"), ui.Accent.Render(fmt.Sprintf("#%d", id)))
	fmt.Printf("    %s\n", title)
	if deadline != nil {
		fmt.Printf("    Deadline: %s\n", ui.Muted.Render(deadline.Format("Jan 2, 2006")))
	}
	if growGoalTarget > 0 {
		fmt.Printf("    Target: %s\n", ui.Muted.Render(fmt.Sprintf("%.0f %s", growGoalTarget, growGoalUnit)))
	}
	fmt.Println()
	return nil
}

func runGrowGoalList(_ *cobra.Command, _ []string) error {
	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	gs := grow.NewStore(db.Conn())
	goals, err := gs.ListGoals()
	if err != nil {
		return fmt.Errorf("listing goals: %w", err)
	}

	if len(goals) == 0 {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No active goals yet."))
		fmt.Printf("  Add one: %s\n",
			ui.Accent.Render(`mine grow goal add "Learn Rust" --target 50 --unit hrs`))
		fmt.Println()
		return nil
	}

	fmt.Println()
	for _, g := range goals {
		printGoalLine(g)
	}
	fmt.Println()
	fmt.Println(ui.Muted.Render(fmt.Sprintf("  %d active goal(s)", len(goals))))
	fmt.Println()
	return nil
}

func printGoalLine(g grow.Goal) {
	id := ui.Muted.Render(fmt.Sprintf("#%d", g.ID))
	line := fmt.Sprintf("    %s %s", id, g.Title)

	if g.TargetValue > 0 {
		pct := g.CurrentValue / g.TargetValue * 100
		if pct > 100 {
			pct = 100
		}
		bar := growProgressBar(pct, 20)
		line += fmt.Sprintf("\n        %s %.0f/%.0f %s (%.0f%%)",
			bar, g.CurrentValue, g.TargetValue, g.Unit, pct)
	}

	if g.Deadline != nil {
		line += ui.Muted.Render(fmt.Sprintf("  due %s", g.Deadline.Format("Jan 2")))
	}

	fmt.Println(line)
}

// growProgressBar renders a simple ASCII progress bar for grow commands.
func growProgressBar(pct float64, width int) string {
	filled := int(pct / 100 * float64(width))
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return ui.Accent.Render("[") + bar + ui.Accent.Render("]")
}

func runGrowGoalDone(_ *cobra.Command, args []string) error {
	id, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("%q is not a valid goal ID — use %s to see IDs",
			args[0], ui.Accent.Render("mine grow goal list"))
	}

	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	gs := grow.NewStore(db.Conn())
	g, err := gs.GetGoal(id)
	if err != nil {
		return err
	}

	if err := gs.DoneGoal(id); err != nil {
		return err
	}

	fmt.Printf("  %s Goal complete! %s\n", ui.Success.Render("✓"), ui.Muted.Render(g.Title))
	fmt.Println()
	return nil
}

func runGrowLog(_ *cobra.Command, args []string) error {
	note := ""
	if len(args) > 0 {
		note = strings.Join(args, " ")
	}

	// Fall back to config default if --minutes not set.
	minutes := growLogMinutes
	if minutes == 0 {
		cfg, err := config.Load()
		if err == nil && cfg.Grow.DefaultMinutes > 0 {
			minutes = cfg.Grow.DefaultMinutes
		}
	}

	var goalID *int
	if growLogGoal > 0 {
		id := growLogGoal
		goalID = &id
	}

	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	gs := grow.NewStore(db.Conn())

	// Validate goal exists if provided.
	if goalID != nil {
		if _, err := gs.GetGoal(*goalID); err != nil {
			return fmt.Errorf("goal #%d not found — use %s to see IDs",
				*goalID, ui.Accent.Render("mine grow goal list"))
		}
	}

	actID, err := gs.LogActivity(note, minutes, goalID, growLogSkill)
	if err != nil {
		return fmt.Errorf("logging activity: %w", err)
	}

	fmt.Printf("  %s Activity logged %s\n", ui.Success.Render("✓"), ui.Accent.Render(fmt.Sprintf("#%d", actID)))
	if note != "" {
		fmt.Printf("    %s\n", note)
	}
	if minutes > 0 {
		fmt.Printf("    Time: %s\n", ui.Muted.Render(fmt.Sprintf("%d min", minutes)))
	}
	if growLogSkill != "" {
		fmt.Printf("    Skill: %s\n", ui.Muted.Render(growLogSkill))
	}
	if goalID != nil {
		g, _ := gs.GetGoal(*goalID)
		if g != nil && g.TargetValue > 0 {
			pct := g.CurrentValue / g.TargetValue * 100
			fmt.Printf("    Goal %s: %.0f/%.0f %s (%.0f%%)\n",
				ui.Accent.Render(fmt.Sprintf("#%d", g.ID)),
				g.CurrentValue, g.TargetValue, g.Unit, pct)
		}
	}

	// Show updated streak.
	now := time.Now()
	streak, err := gs.GetStreak(now)
	if err == nil && streak.Current > 0 {
		streakStr := fmt.Sprintf("%d day", streak.Current)
		if streak.Current != 1 {
			streakStr += "s"
		}
		fmt.Printf("    Streak: %s %s\n", ui.Accent.Render(streakStr), ui.IconFire)
	}

	fmt.Println()
	return nil
}

func runGrowStreak(_ *cobra.Command, _ []string) error {
	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	gs := grow.NewStore(db.Conn())
	now := time.Now()
	streak, err := gs.GetStreak(now)
	if err != nil {
		return fmt.Errorf("computing streak: %w", err)
	}

	fmt.Println()
	ui.Puts(ui.Title.Render("  " + ui.IconFire + " Learning Streak"))
	fmt.Println()

	if streak.Current == 0 && streak.Longest == 0 {
		fmt.Println(ui.Muted.Render("  No activities logged yet."))
		fmt.Printf("  Start your streak: %s\n", ui.Accent.Render("mine grow log --minutes 30"))
		fmt.Println()
		return nil
	}

	currentStr := fmt.Sprintf("%d day", streak.Current)
	if streak.Current != 1 {
		currentStr += "s"
	}
	if streak.Current > 0 {
		currentStr += " " + ui.IconFire
	}
	ui.Kv("Current", currentStr)

	longestStr := fmt.Sprintf("%d day", streak.Longest)
	if streak.Longest != 1 {
		longestStr += "s"
	}
	ui.Kv("Longest", longestStr)

	if streak.Current == 0 {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  Streak not active — log an activity to restart."))
	}

	fmt.Println()
	return nil
}

func runGrowSkills(_ *cobra.Command, _ []string) error {
	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	gs := grow.NewStore(db.Conn())
	skills, err := gs.ListSkills()
	if err != nil {
		return fmt.Errorf("listing skills: %w", err)
	}

	if len(skills) == 0 {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No skills tracked yet."))
		fmt.Printf("  Add one: %s\n", ui.Accent.Render(`mine grow skills set Rust 2`))
		fmt.Println()
		return nil
	}

	fmt.Println()
	ui.Puts(ui.Title.Render("  Skills"))
	fmt.Println()

	// Group by category.
	byCat := make(map[string][]grow.Skill)
	catOrder := []string{}
	for _, sk := range skills {
		if _, seen := byCat[sk.Category]; !seen {
			catOrder = append(catOrder, sk.Category)
		}
		byCat[sk.Category] = append(byCat[sk.Category], sk)
	}

	for _, cat := range catOrder {
		fmt.Printf("  %s\n", ui.Subtitle.Render(cat))
		for _, sk := range byCat[cat] {
			dots := grow.SkillLevelDots(sk.Level)
			ui.Putsf("    %-24s %s  (%d/5)", sk.Name, dots, sk.Level)
		}
		fmt.Println()
	}

	return nil
}

func runGrowSkillsSet(_ *cobra.Command, args []string) error {
	name := args[0]
	level, err := strconv.Atoi(args[1])
	if err != nil || level < 1 || level > 5 {
		return fmt.Errorf("level must be a number between 1 and 5 — got %q\n  Example: %s",
			args[1], ui.Accent.Render("mine grow skills set Rust 3"))
	}

	db, err := store.Open()
	if err != nil {
		return err
	}
	defer db.Close()

	gs := grow.NewStore(db.Conn())
	if err := gs.SetSkill(name, growSkillCategory, level); err != nil {
		return err
	}

	dots := grow.SkillLevelDots(level)
	fmt.Printf("  %s %s → %s\n", ui.Success.Render("✓"), ui.Accent.Render(name), dots)
	fmt.Println()
	return nil
}
