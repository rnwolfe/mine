package cmd

import (
	"strconv"
	"testing"
	"time"

	"github.com/rnwolfe/mine/internal/grow"
	"github.com/rnwolfe/mine/internal/store"
)

// growTestEnv sets up isolated XDG dirs for grow tests.
func growTestEnv(t *testing.T) {
	t.Helper()
	configTestEnv(t)
}

func TestRunGrowGoalAdd_Basic(t *testing.T) {
	growTestEnv(t)
	growGoalDeadline = ""
	growGoalTarget = 50
	growGoalUnit = "hrs"

	err := runGrowGoalAdd(nil, []string{"Learn Rust"})
	if err != nil {
		t.Fatalf("runGrowGoalAdd: %v", err)
	}

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	gs := grow.NewStore(db.Conn())
	goals, err := gs.ListGoals()
	if err != nil {
		t.Fatal(err)
	}
	if len(goals) != 1 {
		t.Fatalf("expected 1 goal, got %d", len(goals))
	}
	if goals[0].Title != "Learn Rust" {
		t.Errorf("title = %q, want %q", goals[0].Title, "Learn Rust")
	}
	if goals[0].TargetValue != 50 {
		t.Errorf("target_value = %v, want 50", goals[0].TargetValue)
	}
	if goals[0].Unit != "hrs" {
		t.Errorf("unit = %q, want %q", goals[0].Unit, "hrs")
	}
}

func TestRunGrowGoalAdd_WithDeadline(t *testing.T) {
	growTestEnv(t)
	growGoalDeadline = "2026-06-01"
	growGoalTarget = 50
	growGoalUnit = "hrs"

	err := runGrowGoalAdd(nil, []string{"Learn Rust"})
	if err != nil {
		t.Fatalf("runGrowGoalAdd: %v", err)
	}

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	gs := grow.NewStore(db.Conn())
	goals, err := gs.ListGoals()
	if err != nil {
		t.Fatal(err)
	}
	if len(goals) != 1 {
		t.Fatalf("expected 1 goal, got %d", len(goals))
	}
	if goals[0].Deadline == nil {
		t.Fatal("expected deadline to be set")
	}
	expected := "2026-06-01"
	if goals[0].Deadline.Format("2006-01-02") != expected {
		t.Errorf("deadline = %q, want %q", goals[0].Deadline.Format("2006-01-02"), expected)
	}
}

func TestRunGrowGoalAdd_InvalidDeadline(t *testing.T) {
	growTestEnv(t)
	growGoalDeadline = "not-a-date"
	growGoalTarget = 0
	growGoalUnit = "hrs"

	err := runGrowGoalAdd(nil, []string{"Learn Rust"})
	if err == nil {
		t.Fatal("expected error for invalid deadline, got nil")
	}
}

func TestRunGrowGoalList_Empty(t *testing.T) {
	growTestEnv(t)

	err := runGrowGoalList(nil, nil)
	if err != nil {
		t.Fatalf("runGrowGoalList: %v", err)
	}
}

func TestRunGrowGoalList_WithGoals(t *testing.T) {
	growTestEnv(t)
	growGoalDeadline = ""
	growGoalTarget = 10
	growGoalUnit = "sessions"

	if err := runGrowGoalAdd(nil, []string{"Practice Piano"}); err != nil {
		t.Fatalf("add goal: %v", err)
	}

	err := runGrowGoalList(nil, nil)
	if err != nil {
		t.Fatalf("runGrowGoalList: %v", err)
	}
}

func TestRunGrowGoalDone(t *testing.T) {
	growTestEnv(t)
	growGoalDeadline = ""
	growGoalTarget = 0
	growGoalUnit = "hrs"

	if err := runGrowGoalAdd(nil, []string{"Quick Goal"}); err != nil {
		t.Fatalf("add goal: %v", err)
	}

	// Get the ID
	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	gs := grow.NewStore(db.Conn())
	goals, _ := gs.ListGoals()
	db.Close()

	if len(goals) != 1 {
		t.Fatalf("expected 1 goal before completion, got %d", len(goals))
	}
	id := goals[0].ID

	if err := runGrowGoalDone(nil, []string{strconv.Itoa(id)}); err != nil {
		t.Fatalf("runGrowGoalDone: %v", err)
	}

	db, err = store.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	gs = grow.NewStore(db.Conn())
	remaining, _ := gs.ListGoals()
	if len(remaining) != 0 {
		t.Errorf("expected 0 active goals after done, got %d", len(remaining))
	}
}

func TestRunGrowLog_Basic(t *testing.T) {
	growTestEnv(t)
	growLogMinutes = 45
	growLogGoal = 0
	growLogSkill = "Rust"

	err := runGrowLog(nil, []string{"Read The Rust Book ch. 3"})
	if err != nil {
		t.Fatalf("runGrowLog: %v", err)
	}

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	gs := grow.NewStore(db.Conn())
	activities, err := gs.AllActivities()
	if err != nil {
		t.Fatal(err)
	}
	if len(activities) != 1 {
		t.Fatalf("expected 1 activity, got %d", len(activities))
	}
	if activities[0].Note != "Read The Rust Book ch. 3" {
		t.Errorf("note = %q, want %q", activities[0].Note, "Read The Rust Book ch. 3")
	}
	if activities[0].Minutes != 45 {
		t.Errorf("minutes = %d, want 45", activities[0].Minutes)
	}
	if activities[0].Skill != "Rust" {
		t.Errorf("skill = %q, want %q", activities[0].Skill, "Rust")
	}
}

func TestRunGrowLog_UpdatesGoalProgress(t *testing.T) {
	growTestEnv(t)
	growGoalDeadline = ""
	growGoalTarget = 50
	growGoalUnit = "hrs"

	if err := runGrowGoalAdd(nil, []string{"Learn Go"}); err != nil {
		t.Fatalf("add goal: %v", err)
	}

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	gs := grow.NewStore(db.Conn())
	goals, _ := gs.ListGoals()
	db.Close()

	if len(goals) == 0 {
		t.Fatal("no goals created")
	}
	goalID := goals[0].ID

	growLogMinutes = 60
	growLogGoal = goalID
	growLogSkill = "Go"

	if err := runGrowLog(nil, []string{"Go concurrency patterns"}); err != nil {
		t.Fatalf("runGrowLog: %v", err)
	}

	db, err = store.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	gs = grow.NewStore(db.Conn())
	g, err := gs.GetGoal(goalID)
	if err != nil {
		t.Fatal(err)
	}
	// current_value should be 60 (summed minutes)
	if g.CurrentValue != 60 {
		t.Errorf("current_value = %v, want 60", g.CurrentValue)
	}
}

func TestRunGrowLog_InvalidGoalID(t *testing.T) {
	growTestEnv(t)
	growLogMinutes = 30
	growLogGoal = 9999
	growLogSkill = ""

	err := runGrowLog(nil, []string{"test"})
	if err == nil {
		t.Fatal("expected error for invalid goal ID, got nil")
	}
}

func TestRunGrowStreak(t *testing.T) {
	growTestEnv(t)
	growLogMinutes = 30
	growLogGoal = 0
	growLogSkill = ""

	if err := runGrowLog(nil, []string{"day 1"}); err != nil {
		t.Fatalf("log activity: %v", err)
	}

	err := runGrowStreak(nil, nil)
	if err != nil {
		t.Fatalf("runGrowStreak: %v", err)
	}
}

func TestRunGrowSkillsSet_Valid(t *testing.T) {
	growTestEnv(t)
	growSkillCategory = "programming"

	err := runGrowSkillsSet(nil, []string{"Rust", "3"})
	if err != nil {
		t.Fatalf("runGrowSkillsSet: %v", err)
	}

	db, err := store.Open()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	gs := grow.NewStore(db.Conn())
	skills, err := gs.ListSkills()
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
	if skills[0].Name != "Rust" {
		t.Errorf("name = %q, want %q", skills[0].Name, "Rust")
	}
	if skills[0].Level != 3 {
		t.Errorf("level = %d, want 3", skills[0].Level)
	}
	if skills[0].Category != "programming" {
		t.Errorf("category = %q, want %q", skills[0].Category, "programming")
	}
}

func TestRunGrowSkillsSet_InvalidLevel(t *testing.T) {
	growTestEnv(t)
	growSkillCategory = "general"

	err := runGrowSkillsSet(nil, []string{"Rust", "6"})
	if err == nil {
		t.Fatal("expected error for level 6, got nil")
	}
}

func TestRunGrowSkills_Empty(t *testing.T) {
	growTestEnv(t)

	err := runGrowSkills(nil, nil)
	if err != nil {
		t.Fatalf("runGrowSkills: %v", err)
	}
}

func TestRunGrowSkills_WithData(t *testing.T) {
	growTestEnv(t)
	growSkillCategory = "general"

	if err := runGrowSkillsSet(nil, []string{"Go", "4"}); err != nil {
		t.Fatalf("set skill: %v", err)
	}

	err := runGrowSkills(nil, nil)
	if err != nil {
		t.Fatalf("runGrowSkills: %v", err)
	}
}

func TestRunGrowReview_Empty(t *testing.T) {
	growTestEnv(t)

	err := runGrowReview(nil, nil)
	if err != nil {
		t.Fatalf("runGrowReview: %v", err)
	}
}

func TestRunGrowDashboard(t *testing.T) {
	growTestEnv(t)

	err := runGrowDashboard(nil, nil)
	if err != nil {
		t.Fatalf("runGrowDashboard: %v", err)
	}
}

func TestGrowStreakNotBrokenYesterday(t *testing.T) {
	// Unit-level test: verify ComputeStreak grace period.
	now := time.Date(2026, 2, 26, 12, 0, 0, 0, time.UTC)
	// Only logged yesterday, not today.
	dates := []string{"2026-02-25", "2026-02-24", "2026-02-23"}
	info := grow.ComputeStreak(dates, now)
	if info.Current != 3 {
		t.Errorf("current streak = %d, want 3 (grace period: yesterday counts)", info.Current)
	}
}

func TestSkillLevelDots(t *testing.T) {
	cases := []struct {
		level int
		want  string
	}{
		{1, "●○○○○"},
		{3, "●●●○○"},
		{5, "●●●●●"},
	}
	for _, tc := range cases {
		got := grow.SkillLevelDots(tc.level)
		if got != tc.want {
			t.Errorf("SkillLevelDots(%d) = %q, want %q", tc.level, got, tc.want)
		}
	}
}
