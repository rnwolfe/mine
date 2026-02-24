package todo

import (
	"strings"
	"testing"
	"time"
)

// --- ParseRecurrence ---

func TestParseRecurrence_ValidInputs(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"d", RecurrenceDaily},
		{"day", RecurrenceDaily},
		{"daily", RecurrenceDaily},
		{"D", RecurrenceDaily},
		{"DAILY", RecurrenceDaily},
		{"wd", RecurrenceWeekday},
		{"weekday", RecurrenceWeekday},
		{"WD", RecurrenceWeekday},
		{"w", RecurrenceWeekly},
		{"week", RecurrenceWeekly},
		{"weekly", RecurrenceWeekly},
		{"W", RecurrenceWeekly},
		{"m", RecurrenceMonthly},
		{"month", RecurrenceMonthly},
		{"monthly", RecurrenceMonthly},
		{"M", RecurrenceMonthly},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got, err := ParseRecurrence(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("ParseRecurrence(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestParseRecurrence_InvalidInput(t *testing.T) {
	invalids := []string{"biweekly", "yearly", "y", "never", ""}
	for _, s := range invalids {
		t.Run(s, func(t *testing.T) {
			_, err := ParseRecurrence(s)
			if err == nil {
				t.Fatalf("expected error for input %q", s)
			}
		})
	}
}

func TestParseRecurrence_ErrorMessageContainsValidValues(t *testing.T) {
	_, err := ParseRecurrence("biweekly")
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	for _, v := range []string{"day", "weekday", "week", "month"} {
		if !strings.Contains(msg, v) {
			t.Errorf("expected %q in error message, got: %s", v, msg)
		}
	}
}

// --- nextDueDate ---

func TestNextDueDate_Daily(t *testing.T) {
	base := time.Date(2026, 2, 24, 0, 0, 0, 0, time.UTC) // Tuesday
	got := nextDueDate(base, RecurrenceDaily)
	want := time.Date(2026, 2, 25, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("daily: got %s, want %s", got.Format("2006-01-02"), want.Format("2006-01-02"))
	}
}

func TestNextDueDate_Weekday_SkipsSaturday(t *testing.T) {
	// Friday → skip Saturday, skip Sunday → Monday
	base := time.Date(2026, 2, 27, 0, 0, 0, 0, time.UTC) // Friday
	got := nextDueDate(base, RecurrenceWeekday)
	want := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC) // Monday
	if !got.Equal(want) {
		t.Errorf("weekday from Friday: got %s, want %s", got.Format("2006-01-02"), want.Format("2006-01-02"))
	}
}

func TestNextDueDate_Weekday_FromSaturday(t *testing.T) {
	// Saturday → skip Saturday, skip Sunday → Monday
	base := time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC) // Saturday
	got := nextDueDate(base, RecurrenceWeekday)
	want := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC) // Monday
	if !got.Equal(want) {
		t.Errorf("weekday from Saturday: got %s, want %s", got.Format("2006-01-02"), want.Format("2006-01-02"))
	}
}

func TestNextDueDate_Weekday_Tuesday(t *testing.T) {
	// Tuesday → Wednesday
	base := time.Date(2026, 2, 24, 0, 0, 0, 0, time.UTC) // Tuesday
	got := nextDueDate(base, RecurrenceWeekday)
	want := time.Date(2026, 2, 25, 0, 0, 0, 0, time.UTC) // Wednesday
	if !got.Equal(want) {
		t.Errorf("weekday from Tuesday: got %s, want %s", got.Format("2006-01-02"), want.Format("2006-01-02"))
	}
}

func TestNextDueDate_Weekly(t *testing.T) {
	base := time.Date(2026, 2, 24, 0, 0, 0, 0, time.UTC)
	got := nextDueDate(base, RecurrenceWeekly)
	want := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("weekly: got %s, want %s", got.Format("2006-01-02"), want.Format("2006-01-02"))
	}
}

func TestNextDueDate_Monthly_SameDay(t *testing.T) {
	base := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	got := nextDueDate(base, RecurrenceMonthly)
	want := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("monthly same day: got %s, want %s", got.Format("2006-01-02"), want.Format("2006-01-02"))
	}
}

func TestNextDueDate_Monthly_ClampToMonthEnd(t *testing.T) {
	// Jan 31 → Feb only has 28 days in 2026, so should clamp to Feb 28
	base := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	got := nextDueDate(base, RecurrenceMonthly)
	want := time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("monthly month-end clamp: got %s, want %s", got.Format("2006-01-02"), want.Format("2006-01-02"))
	}
}

func TestNextDueDate_Monthly_MarchToApril(t *testing.T) {
	// March 31 → April only has 30 days, clamp to April 30
	base := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	got := nextDueDate(base, RecurrenceMonthly)
	want := time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("march→april clamp: got %s, want %s", got.Format("2006-01-02"), want.Format("2006-01-02"))
	}
}

// --- Store recurrence integration tests ---

func TestStoreComplete_NonRecurring_NoSpawn(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	id, err := s.Add("regular task", "", PrioMedium, nil, nil, nil, ScheduleLater, RecurrenceNone)
	if err != nil {
		t.Fatal(err)
	}

	spawnedID, spawnedDue, err := s.Complete(id)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if spawnedID != 0 {
		t.Errorf("expected no spawn for non-recurring task, got spawnedID=%d", spawnedID)
	}
	if spawnedDue != nil {
		t.Errorf("expected nil spawnedDue for non-recurring task")
	}
}

func TestStoreComplete_Recurring_SpawnsNext(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	due := time.Date(2026, 2, 24, 0, 0, 0, 0, time.UTC)
	id, err := s.Add("weekly task", "", PrioHigh, []string{"work"}, &due, nil, ScheduleLater, RecurrenceWeekly)
	if err != nil {
		t.Fatal(err)
	}

	spawnedID, spawnedDue, err := s.Complete(id)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if spawnedID == 0 {
		t.Fatal("expected spawned ID for recurring task")
	}
	if spawnedDue == nil {
		t.Fatal("expected spawned due date")
	}

	wantDue := time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC)
	if !spawnedDue.Equal(wantDue) {
		t.Errorf("spawned due: got %s, want %s", spawnedDue.Format("2006-01-02"), wantDue.Format("2006-01-02"))
	}

	// The spawned task should exist as open
	spawned, err := s.Get(spawnedID)
	if err != nil {
		t.Fatalf("Get spawned: %v", err)
	}
	if spawned.Done {
		t.Error("spawned task should be open")
	}
	if spawned.Title != "weekly task" {
		t.Errorf("spawned title: got %q, want %q", spawned.Title, "weekly task")
	}
	if spawned.Priority != PrioHigh {
		t.Errorf("spawned priority: got %d, want %d", spawned.Priority, PrioHigh)
	}
	if spawned.Recurrence != RecurrenceWeekly {
		t.Errorf("spawned recurrence: got %q, want %q", spawned.Recurrence, RecurrenceWeekly)
	}
	if spawned.Schedule != ScheduleToday {
		t.Errorf("spawned schedule: got %q, want %q", spawned.Schedule, ScheduleToday)
	}
	if spawned.DueDate == nil || !spawned.DueDate.Equal(wantDue) {
		t.Errorf("spawned due date mismatch")
	}
	if len(spawned.Tags) != 1 || spawned.Tags[0] != "work" {
		t.Errorf("spawned tags: got %v, want [work]", spawned.Tags)
	}
}

func TestStoreComplete_Recurring_NoDue_UsesToday(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	// Task with no due date — base date should be today
	id, err := s.Add("daily standup", "", PrioMedium, nil, nil, nil, ScheduleLater, RecurrenceDaily)
	if err != nil {
		t.Fatal(err)
	}

	spawnedID, spawnedDue, err := s.Complete(id)
	if err != nil {
		t.Fatalf("Complete: %v", err)
	}
	if spawnedID == 0 {
		t.Fatal("expected spawn for recurring task without due date")
	}

	// Due date should be today + 1 day
	today := time.Now()
	expectedBase := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	wantDue := expectedBase.AddDate(0, 0, 1)
	if spawnedDue == nil {
		t.Fatal("expected non-nil spawned due date")
	}
	if !spawnedDue.Equal(wantDue) {
		t.Errorf("no-due-date base: got %s, want %s", spawnedDue.Format("2006-01-02"), wantDue.Format("2006-01-02"))
	}
}

func TestStoreComplete_Recurring_InheritsProject(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	projPath := "/projects/myapp"
	due := time.Date(2026, 2, 24, 0, 0, 0, 0, time.UTC)
	id, err := s.Add("team standup", "", PrioMedium, nil, &due, &projPath, ScheduleLater, RecurrenceWeekday)
	if err != nil {
		t.Fatal(err)
	}

	spawnedID, _, err := s.Complete(id)
	if err != nil {
		t.Fatal(err)
	}

	spawned, err := s.Get(spawnedID)
	if err != nil {
		t.Fatalf("Get spawned: %v", err)
	}
	if spawned.ProjectPath == nil || *spawned.ProjectPath != projPath {
		t.Errorf("spawned project path: got %v, want %q", spawned.ProjectPath, projPath)
	}
}

func TestStoreComplete_Recurring_SpawnsMultipleGenerations(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	// Start with a daily recurring task that has an initial due date.
	initialDue := time.Date(2026, 2, 24, 0, 0, 0, 0, time.UTC)
	id, err := s.Add("daily recurring chain", "", PrioMedium, nil, &initialDue, nil, ScheduleLater, RecurrenceDaily)
	if err != nil {
		t.Fatal(err)
	}

	// Complete the original task (A) to spawn the first successor (B).
	spawnedID1, spawnedDue1, err := s.Complete(id)
	if err != nil {
		t.Fatalf("Complete first generation: %v", err)
	}
	if spawnedID1 == 0 {
		t.Fatal("expected spawned recurring task after first completion")
	}
	if spawnedDue1 == nil {
		t.Fatal("expected non-nil due date for first spawned recurring task")
	}

	spawned1, err := s.Get(spawnedID1)
	if err != nil {
		t.Fatalf("Get first spawned task: %v", err)
	}
	if spawned1.Recurrence != RecurrenceDaily {
		t.Fatalf("first spawned recurrence: got %q, want %q", spawned1.Recurrence, RecurrenceDaily)
	}

	// Complete the first spawned task (B) to spawn the next successor (C).
	spawnedID2, spawnedDue2, err := s.Complete(spawnedID1)
	if err != nil {
		t.Fatalf("Complete second generation: %v", err)
	}
	if spawnedID2 == 0 {
		t.Fatal("expected spawned recurring task after second completion")
	}
	if spawnedDue2 == nil {
		t.Fatal("expected non-nil due date for second spawned recurring task")
	}

	spawned2, err := s.Get(spawnedID2)
	if err != nil {
		t.Fatalf("Get second spawned task: %v", err)
	}
	if spawned2.Recurrence != RecurrenceDaily {
		t.Fatalf("second spawned recurrence: got %q, want %q", spawned2.Recurrence, RecurrenceDaily)
	}

	// Ensure the due date is moving forward along the chain.
	if !spawnedDue2.After(*spawnedDue1) {
		t.Fatalf("second spawned due date should be after first: got %v, first %v", spawnedDue2, spawnedDue1)
	}
}

func TestListRecurring_ReturnsOnlyRecurring(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	s.Add("non-recurring", "", PrioMedium, nil, nil, nil, ScheduleLater, RecurrenceNone)
	s.Add("weekly task", "", PrioHigh, nil, nil, nil, ScheduleLater, RecurrenceWeekly)
	s.Add("daily task", "", PrioMedium, nil, nil, nil, ScheduleLater, RecurrenceDaily)

	recurring, err := s.ListRecurring()
	if err != nil {
		t.Fatalf("ListRecurring: %v", err)
	}
	if len(recurring) != 2 {
		t.Fatalf("expected 2 recurring tasks, got %d", len(recurring))
	}
	for _, task := range recurring {
		if task.Recurrence == RecurrenceNone || task.Recurrence == "" {
			t.Errorf("non-recurring task in ListRecurring result: %q", task.Title)
		}
	}
}

func TestListRecurring_ExcludesDoneTasks(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	// Mark a recurring task done via raw SQL (no spawn) to test filtering.
	_, err := db.Exec(
		`INSERT INTO todos (title, priority, done, recurrence, schedule) VALUES (?, 2, 1, ?, ?)`,
		"done recurring", RecurrenceWeekly, ScheduleLater,
	)
	if err != nil {
		t.Fatalf("inserting done task: %v", err)
	}

	s.Add("open recurring", "", PrioMedium, nil, nil, nil, ScheduleLater, RecurrenceWeekly) //nolint:errcheck

	recurring, err := s.ListRecurring()
	if err != nil {
		t.Fatalf("ListRecurring: %v", err)
	}
	if len(recurring) != 1 {
		t.Fatalf("expected 1 recurring open task, got %d", len(recurring))
	}
	if recurring[0].Title != "open recurring" {
		t.Errorf("expected 'open recurring', got %q", recurring[0].Title)
	}
}

func TestDemoteProject_DemotesOpenTodos(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	projPath := "/projects/myapp"
	s.Add("proj task 1", "", PrioMedium, nil, nil, &projPath, ScheduleLater, RecurrenceNone)
	s.Add("proj task 2", "", PrioHigh, nil, nil, &projPath, ScheduleLater, RecurrenceWeekly)
	s.Add("global task", "", PrioLow, nil, nil, nil, ScheduleLater, RecurrenceNone)

	n, err := s.DemoteProject(projPath)
	if err != nil {
		t.Fatalf("DemoteProject: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 demoted tasks, got %d", n)
	}

	// All tasks should now be global
	todos, err := s.List(ListOptions{AllProjects: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range todos {
		if task.ProjectPath != nil {
			t.Errorf("task %q still has project path %q after demotion", task.Title, *task.ProjectPath)
		}
	}
}

func TestDemoteProject_DoesNotDemoteCompletedTodos(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	projPath := "/projects/myapp"
	// Insert a done task directly so we don't trigger spawn logic.
	_, err := db.Exec(
		`INSERT INTO todos (title, priority, done, project_path, recurrence, schedule) VALUES (?, 2, 1, ?, ?, ?)`,
		"done proj task", projPath, RecurrenceNone, ScheduleLater,
	)
	if err != nil {
		t.Fatalf("inserting done task: %v", err)
	}

	n, demoteErr := s.DemoteProject(projPath)
	if demoteErr != nil {
		t.Fatal(demoteErr)
	}
	// Completed tasks should NOT be demoted (WHERE done = 0 in DemoteProject).
	if n != 0 {
		t.Errorf("expected 0 demoted tasks (completed tasks not demoted), got %d", n)
	}
}
