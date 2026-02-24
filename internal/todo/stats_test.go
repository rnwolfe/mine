package todo

import (
	"testing"
	"time"
)

// setupStatsTestDB creates an in-memory DB with the todos + dig_sessions tables.
func setupStatsTestDB(t *testing.T) interface{ Query(string, ...any) (interface{}, error) } {
	t.Helper()
	return nil // use setupTestDB from todo_test.go
}

// completedAt inserts a todo that was completed at the specified time,
// setting both created_at and completed_at explicitly.
func insertCompletedAtTime(t *testing.T, s *Store, title string, createdAt, completedAt time.Time) int {
	t.Helper()
	res, err := s.db.Exec(
		`INSERT INTO todos (title, priority, done, created_at, completed_at, updated_at)
		 VALUES (?, 2, 1, ?, ?, ?)`,
		title,
		createdAt.UTC().Format("2006-01-02 15:04:05"),
		completedAt.UTC().Format("2006-01-02 15:04:05"),
		completedAt.UTC().Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		t.Fatalf("insertCompletedAtTime: %v", err)
	}
	id, _ := res.LastInsertId()
	return int(id)
}

// insertOpenTodo inserts an open (not-done) todo.
func insertOpenTodo(t *testing.T, s *Store, title string, projPath *string) int {
	t.Helper()
	id, err := s.Add(title, "", PrioMedium, nil, nil, projPath, ScheduleLater)
	if err != nil {
		t.Fatalf("insertOpenTodo: %v", err)
	}
	return id
}

// TestStartOfWeek validates Monday-start week calculation.
func TestStartOfWeek(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantDate string
	}{
		{"monday", "2026-02-23", "2026-02-23"},
		{"tuesday", "2026-02-24", "2026-02-23"},
		{"wednesday", "2026-02-25", "2026-02-23"},
		{"sunday", "2026-03-01", "2026-02-23"},
		{"saturday", "2026-02-28", "2026-02-23"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input, _ := time.Parse("2006-01-02", tc.input)
			got := startOfWeek(input)
			if got.Format("2006-01-02") != tc.wantDate {
				t.Errorf("startOfWeek(%s) = %s, want %s", tc.input, got.Format("2006-01-02"), tc.wantDate)
			}
			if got.Weekday() != time.Monday {
				t.Errorf("startOfWeek result is not Monday: %s", got.Weekday())
			}
		})
	}
}

// TestComputeStreak_Empty returns zeros when no completions exist.
func TestComputeStreak_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	now, _ := time.Parse("2006-01-02", "2026-02-24")
	cur, longest, err := computeStreak(db, nil, now)
	if err != nil {
		t.Fatalf("computeStreak: %v", err)
	}
	if cur != 0 || longest != 0 {
		t.Errorf("expected (0,0), got (%d,%d)", cur, longest)
	}
}

// TestComputeStreak_TodayOnly: completion today → streak = 1.
func TestComputeStreak_TodayOnly(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	now, _ := time.Parse("2006-01-02", "2026-02-24")

	insertCompletedAtTime(t, s, "task", now.AddDate(0, 0, -1), now)

	cur, longest, err := computeStreak(db, nil, now)
	if err != nil {
		t.Fatalf("computeStreak: %v", err)
	}
	if cur != 1 {
		t.Errorf("expected current=1, got %d", cur)
	}
	if longest < 1 {
		t.Errorf("expected longest>=1, got %d", longest)
	}
}

// TestComputeStreak_ConsecutiveDays: 3 consecutive days → streak = 3.
func TestComputeStreak_ConsecutiveDays(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	now, _ := time.Parse("2006-01-02", "2026-02-24")

	for i := 0; i < 3; i++ {
		day := now.AddDate(0, 0, -i)
		insertCompletedAtTime(t, s, "task", day.AddDate(0, 0, -1), day)
	}

	cur, longest, err := computeStreak(db, nil, now)
	if err != nil {
		t.Fatalf("computeStreak: %v", err)
	}
	if cur != 3 {
		t.Errorf("expected current=3, got %d", cur)
	}
	if longest < 3 {
		t.Errorf("expected longest>=3, got %d", longest)
	}
}

// TestComputeStreak_GapBreaksStreak: a gap in dates resets the current streak.
func TestComputeStreak_GapBreaksStreak(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	now, _ := time.Parse("2006-01-02", "2026-02-24")

	// Today and yesterday (streak=2), then gap, then 5 days 2 weeks ago.
	insertCompletedAtTime(t, s, "t1", now.AddDate(0, 0, -1), now)
	insertCompletedAtTime(t, s, "t2", now.AddDate(0, 0, -2), now.AddDate(0, 0, -1))
	// Gap here
	for i := 14; i <= 18; i++ {
		day := now.AddDate(0, 0, -i)
		insertCompletedAtTime(t, s, "old", day.AddDate(0, 0, -1), day)
	}

	cur, longest, err := computeStreak(db, nil, now)
	if err != nil {
		t.Fatalf("computeStreak: %v", err)
	}
	if cur != 2 {
		t.Errorf("expected current=2 (today + yesterday), got %d", cur)
	}
	// Longest should be the 5-day run from 2 weeks ago.
	if longest != 5 {
		t.Errorf("expected longest=5, got %d", longest)
	}
}

// TestComputeStreak_YesterdayOnly: no completion today but completion yesterday →
// streak is still 1 (hasn't been broken yet).
func TestComputeStreak_YesterdayOnly(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	now, _ := time.Parse("2006-01-02", "2026-02-24")
	yesterday := now.AddDate(0, 0, -1)

	insertCompletedAtTime(t, s, "task", yesterday.AddDate(0, 0, -1), yesterday)

	cur, _, err := computeStreak(db, nil, now)
	if err != nil {
		t.Fatalf("computeStreak: %v", err)
	}
	if cur != 1 {
		t.Errorf("expected current=1 (yesterday counts), got %d", cur)
	}
}

// TestComputeStreak_StaleData: last completion was 3+ days ago → streak = 0.
func TestComputeStreak_StaleData(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	now, _ := time.Parse("2006-01-02", "2026-02-24")
	old := now.AddDate(0, 0, -5)

	insertCompletedAtTime(t, s, "old task", old.AddDate(0, 0, -1), old)

	cur, _, err := computeStreak(db, nil, now)
	if err != nil {
		t.Fatalf("computeStreak: %v", err)
	}
	if cur != 0 {
		t.Errorf("expected current=0 for stale data, got %d", cur)
	}
}

// TestCountCompletedSince_WeeklyBoundary validates Monday-start weekly count.
func TestCountCompletedSince_WeeklyBoundary(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	// now = Wednesday 2026-02-25; week start = Monday 2026-02-23
	now, _ := time.Parse("2006-01-02", "2026-02-25")
	weekStart := startOfWeek(now)

	// Completions: Monday, Tuesday, Wednesday this week (3), and last Sunday (out of week).
	thisWeek := []string{"2026-02-23", "2026-02-24", "2026-02-25"}
	for _, ds := range thisWeek {
		d, _ := time.Parse("2006-01-02", ds)
		insertCompletedAtTime(t, s, "task "+ds, d.AddDate(0, 0, -1), d)
	}
	// Last Sunday — outside this week.
	lastSunday, _ := time.Parse("2006-01-02", "2026-02-22")
	insertCompletedAtTime(t, s, "old task", lastSunday.AddDate(0, 0, -1), lastSunday)

	count, err := countCompletedSince(db, nil, weekStart)
	if err != nil {
		t.Fatalf("countCompletedSince: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3 completions this week, got %d", count)
	}
}

// TestCountCompletedSince_MonthlyBoundary validates calendar month count.
func TestCountCompletedSince_MonthlyBoundary(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	now, _ := time.Parse("2006-01-02", "2026-02-24")
	monthStart := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	// 5 completions in February.
	for i := 0; i < 5; i++ {
		d := time.Date(2026, 2, i+1, 12, 0, 0, 0, time.UTC)
		insertCompletedAtTime(t, s, "feb task", d.AddDate(0, 0, -1), d)
	}
	// 3 completions in January (prior month).
	for i := 0; i < 3; i++ {
		d := time.Date(2026, 1, 20+i, 12, 0, 0, 0, time.UTC)
		insertCompletedAtTime(t, s, "jan task", d.AddDate(0, 0, -1), d)
	}

	_ = now
	count, err := countCompletedSince(db, nil, monthStart)
	if err != nil {
		t.Fatalf("countCompletedSince: %v", err)
	}
	if count != 5 {
		t.Errorf("expected 5 completions this month, got %d", count)
	}
}

// TestAvgCloseTime_CompletedOnly validates that only completed tasks count.
func TestAvgCloseTime_CompletedOnly(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	// Completed task: created at T, completed 2 days later.
	base := time.Date(2026, 2, 10, 12, 0, 0, 0, time.UTC)
	insertCompletedAtTime(t, s, "done", base, base.AddDate(0, 0, 2))

	// Open task (should be excluded from avg).
	insertOpenTodo(t, s, "open", nil)

	avg, err := avgCloseTime(db, nil)
	if err != nil {
		t.Fatalf("avgCloseTime: %v", err)
	}

	// Should be approximately 2 days.
	twodays := 2 * 24 * time.Hour
	diff := avg - twodays
	if diff < 0 {
		diff = -diff
	}
	if diff > time.Hour {
		t.Errorf("expected avg ~2 days, got %v", avg)
	}
}

// TestAvgCloseTime_Empty returns zero when no completed tasks.
func TestAvgCloseTime_Empty(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	insertOpenTodo(t, s, "open task", nil)

	avg, err := avgCloseTime(db, nil)
	if err != nil {
		t.Fatalf("avgCloseTime: %v", err)
	}
	if avg != 0 {
		t.Errorf("expected 0 for no completed tasks, got %v", avg)
	}
}

// TestAvgCloseTime_MultipleCompleted validates averaging over multiple tasks.
func TestAvgCloseTime_MultipleCompleted(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	base := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	// 1-day close and 3-day close → avg 2 days.
	insertCompletedAtTime(t, s, "task1", base, base.AddDate(0, 0, 1))
	insertCompletedAtTime(t, s, "task2", base, base.AddDate(0, 0, 3))

	avg, err := avgCloseTime(db, nil)
	if err != nil {
		t.Fatalf("avgCloseTime: %v", err)
	}

	twodays := 2 * 24 * time.Hour
	diff := avg - twodays
	if diff < 0 {
		diff = -diff
	}
	if diff > time.Hour {
		t.Errorf("expected avg ~2 days, got %v", avg)
	}
}

// TestGetStats_NoCompletions returns a valid zero-value Stats, not an error.
func TestGetStats_NoCompletions(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	now := time.Now()
	stats, err := GetStats(db, nil, now)
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats == nil {
		t.Fatal("expected non-nil stats")
	}
	if stats.Streak != 0 || stats.CompletedWeek != 0 || stats.CompletedMonth != 0 {
		t.Errorf("expected zero stats for empty DB, got: %+v", stats)
	}
}

// TestGetStats_WithData validates that populated data flows through correctly.
func TestGetStats_WithData(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	now, _ := time.Parse("2006-01-02", "2026-02-24")

	// Two completions today.
	insertCompletedAtTime(t, s, "t1", now.AddDate(0, 0, -2), now)
	insertCompletedAtTime(t, s, "t2", now.AddDate(0, 0, -1), now)

	stats, err := GetStats(db, nil, now)
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.Streak < 1 {
		t.Errorf("expected streak >= 1, got %d", stats.Streak)
	}
	if stats.CompletedMonth < 2 {
		t.Errorf("expected >= 2 completed this month, got %d", stats.CompletedMonth)
	}
}

// TestProjectBreakdown_GlobalLabel verifies null project_path shows as "(global)".
func TestProjectBreakdown_GlobalLabel(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	// Add a global (nil project) open todo.
	insertOpenTodo(t, s, "global task", nil)

	breakdown, err := projectBreakdown(db)
	if err != nil {
		t.Fatalf("projectBreakdown: %v", err)
	}
	if len(breakdown) == 0 {
		t.Fatal("expected at least one entry in breakdown")
	}
	found := false
	for _, p := range breakdown {
		if p.Name == "(global)" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected '(global)' in breakdown names, got: %v", breakdown)
	}
}

// TestProjectBreakdown_ProjectName verifies project_path base name is used.
func TestProjectBreakdown_ProjectName(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	proj := "/home/user/projects/myapp"
	insertOpenTodo(t, s, "proj task", &proj)

	breakdown, err := projectBreakdown(db)
	if err != nil {
		t.Fatalf("projectBreakdown: %v", err)
	}
	found := false
	for _, p := range breakdown {
		if p.Name == "myapp" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'myapp' in breakdown, got: %v", breakdown)
	}
}
