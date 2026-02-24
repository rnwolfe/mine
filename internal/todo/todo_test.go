package todo

import (
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`CREATE TABLE todos (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		body TEXT DEFAULT '',
		priority INTEGER DEFAULT 2,
		done INTEGER DEFAULT 0,
		due_date TEXT,
		tags TEXT DEFAULT '',
		project_path TEXT,
		schedule TEXT DEFAULT 'later',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		completed_at DATETIME
	)`)
	if err != nil {
		t.Fatal(err)
	}

	return db
}

func strPtr(s string) *string { return &s }

func TestAddAndList(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	id, err := s.Add("Test todo", PrioHigh, []string{"test"}, nil, nil, ScheduleLater)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if id != 1 {
		t.Fatalf("expected id 1, got %d", id)
	}

	todos, err := s.List(ListOptions{})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(todos))
	}
	if todos[0].Title != "Test todo" {
		t.Fatalf("expected title 'Test todo', got %q", todos[0].Title)
	}
	if todos[0].Priority != PrioHigh {
		t.Fatalf("expected priority %d, got %d", PrioHigh, todos[0].Priority)
	}
	if todos[0].Schedule != ScheduleLater {
		t.Fatalf("expected schedule %q, got %q", ScheduleLater, todos[0].Schedule)
	}
}

func TestComplete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	id, _ := s.Add("Complete me", PrioMedium, nil, nil, nil, ScheduleLater)
	if err := s.Complete(id); err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	// Should not appear in default list
	todos, _ := s.List(ListOptions{})
	if len(todos) != 0 {
		t.Fatalf("expected 0 open todos, got %d", len(todos))
	}

	// Should appear with ShowDone=true
	todos, _ = s.List(ListOptions{ShowDone: true})
	if len(todos) != 1 {
		t.Fatalf("expected 1 total todo, got %d", len(todos))
	}
	if !todos[0].Done {
		t.Fatal("expected todo to be done")
	}
}

func TestDelete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	id, _ := s.Add("Delete me", PrioLow, nil, nil, nil, ScheduleLater)
	if err := s.Delete(id); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	todos, _ := s.List(ListOptions{ShowDone: true})
	if len(todos) != 0 {
		t.Fatalf("expected 0 todos, got %d", len(todos))
	}
}

func TestCount(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	s.Add("One", PrioMedium, nil, nil, nil, ScheduleLater)
	s.Add("Two", PrioHigh, nil, nil, nil, ScheduleLater)

	yesterday := time.Now().AddDate(0, 0, -1)
	s.Add("Overdue", PrioCrit, nil, &yesterday, nil, ScheduleLater)

	open, total, overdue, err := s.Count(nil)
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if open != 3 {
		t.Fatalf("expected 3 open, got %d", open)
	}
	if total != 3 {
		t.Fatalf("expected 3 total, got %d", total)
	}
	if overdue != 1 {
		t.Fatalf("expected 1 overdue, got %d", overdue)
	}
}

func TestEdit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	id, _ := s.Add("Original", PrioLow, nil, nil, nil, ScheduleLater)

	newTitle := "Edited"
	newPrio := PrioHigh
	if err := s.Edit(id, &newTitle, &newPrio); err != nil {
		t.Fatalf("Edit failed: %v", err)
	}

	todo, err := s.Get(id)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if todo.Title != "Edited" {
		t.Fatalf("expected title 'Edited', got %q", todo.Title)
	}
	if todo.Priority != PrioHigh {
		t.Fatalf("expected priority %d, got %d", PrioHigh, todo.Priority)
	}
}

func TestPriorityLabel(t *testing.T) {
	tests := []struct {
		prio  int
		label string
	}{
		{PrioLow, "low"},
		{PrioMedium, "med"},
		{PrioHigh, "high"},
		{PrioCrit, "crit"},
		{99, "?"},
	}
	for _, tt := range tests {
		got := PriorityLabel(tt.prio)
		if got != tt.label {
			t.Errorf("PriorityLabel(%d) = %q, want %q", tt.prio, got, tt.label)
		}
	}
}

func TestAdd_WithProjectPath(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	projPath := "/home/user/myproject"
	id, err := s.Add("project task", PrioMedium, nil, nil, &projPath, ScheduleLater)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	got, err := s.Get(id)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.ProjectPath == nil {
		t.Fatal("expected ProjectPath to be set")
	}
	if *got.ProjectPath != projPath {
		t.Fatalf("expected ProjectPath %q, got %q", projPath, *got.ProjectPath)
	}
}

func TestList_ProjectFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	projA := "/projects/alpha"
	projB := "/projects/beta"

	s.Add("global task", PrioMedium, nil, nil, nil, ScheduleLater)
	s.Add("alpha task", PrioMedium, nil, nil, &projA, ScheduleLater)
	s.Add("beta task", PrioMedium, nil, nil, &projB, ScheduleLater)

	t.Run("global_only", func(t *testing.T) {
		// nil project path → global only
		todos, err := s.List(ListOptions{ProjectPath: nil})
		if err != nil {
			t.Fatal(err)
		}
		if len(todos) != 1 {
			t.Fatalf("expected 1 global todo, got %d", len(todos))
		}
		if todos[0].Title != "global task" {
			t.Fatalf("expected 'global task', got %q", todos[0].Title)
		}
	})

	t.Run("project_plus_global", func(t *testing.T) {
		// project A → alpha task + global task
		todos, err := s.List(ListOptions{ProjectPath: &projA})
		if err != nil {
			t.Fatal(err)
		}
		if len(todos) != 2 {
			t.Fatalf("expected 2 todos (project + global), got %d", len(todos))
		}
	})

	t.Run("all_projects", func(t *testing.T) {
		// AllProjects → all 3
		todos, err := s.List(ListOptions{AllProjects: true})
		if err != nil {
			t.Fatal(err)
		}
		if len(todos) != 3 {
			t.Fatalf("expected 3 todos, got %d", len(todos))
		}
	})
}

func TestCount_ProjectScoped(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	projA := "/projects/alpha"

	s.Add("global task", PrioMedium, nil, nil, nil, ScheduleLater)
	s.Add("alpha task", PrioHigh, nil, nil, &projA, ScheduleLater)
	s.Add("other project task", PrioLow, nil, nil, strPtr("/projects/other"), ScheduleLater)

	// Count nil → all todos
	open, total, _, err := s.Count(nil)
	if err != nil {
		t.Fatalf("Count(nil) failed: %v", err)
	}
	if open != 3 {
		t.Fatalf("Count(nil) open: expected 3, got %d", open)
	}
	if total != 3 {
		t.Fatalf("Count(nil) total: expected 3, got %d", total)
	}

	// Count scoped to projA → alpha task + global task
	open, total, _, err = s.Count(&projA)
	if err != nil {
		t.Fatalf("Count(&projA) failed: %v", err)
	}
	if open != 2 {
		t.Fatalf("Count(&projA) open: expected 2, got %d", open)
	}
	if total != 2 {
		t.Fatalf("Count(&projA) total: expected 2, got %d", total)
	}
}

func TestList_ShowDone_WithProject(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	projA := "/projects/alpha"
	id, _ := s.Add("alpha done", PrioMedium, nil, nil, &projA, ScheduleLater)
	s.Complete(id)
	s.Add("alpha open", PrioMedium, nil, nil, &projA, ScheduleLater)

	// Without ShowDone: only open
	todos, _ := s.List(ListOptions{ProjectPath: &projA})
	if len(todos) != 1 {
		t.Fatalf("expected 1 open todo, got %d", len(todos))
	}

	// With ShowDone: both
	todos, _ = s.List(ListOptions{ProjectPath: &projA, ShowDone: true})
	if len(todos) != 2 {
		t.Fatalf("expected 2 todos with ShowDone, got %d", len(todos))
	}
}

// --- Schedule tests ---

func TestParseSchedule(t *testing.T) {
	tests := []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"today", ScheduleToday, false},
		{"t", ScheduleToday, false},
		{"TODAY", ScheduleToday, false},
		{"soon", ScheduleSoon, false},
		{"s", ScheduleSoon, false},
		{"later", ScheduleLater, false},
		{"l", ScheduleLater, false},
		{"someday", ScheduleSomeday, false},
		{"sd", ScheduleSomeday, false},
		{"SOMEDAY", ScheduleSomeday, false},
		{"invalid", "", true},
		{"", "", true},
		{"mañana", "", true},
	}

	for _, tt := range tests {
		got, err := ParseSchedule(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseSchedule(%q): expected error, got nil", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseSchedule(%q): unexpected error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseSchedule(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSetSchedule(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	id, err := s.Add("Test", PrioMedium, nil, nil, nil, ScheduleLater)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// Update to today
	if err := s.SetSchedule(id, ScheduleToday); err != nil {
		t.Fatalf("SetSchedule failed: %v", err)
	}

	got, err := s.Get(id)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Schedule != ScheduleToday {
		t.Fatalf("expected schedule %q, got %q", ScheduleToday, got.Schedule)
	}
}

func TestSetSchedule_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	err := s.SetSchedule(9999, ScheduleToday)
	if err == nil {
		t.Fatal("expected error for non-existent todo ID")
	}
}

func TestList_ExcludesSomebodyByDefault(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	s.Add("later task", PrioMedium, nil, nil, nil, ScheduleLater)
	s.Add("today task", PrioMedium, nil, nil, nil, ScheduleToday)
	s.Add("someday task", PrioMedium, nil, nil, nil, ScheduleSomeday)

	// Default (IncludeSomeday=false): someday excluded
	todos, err := s.List(ListOptions{AllProjects: true})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(todos) != 2 {
		t.Fatalf("expected 2 todos (someday excluded), got %d", len(todos))
	}
	for _, td := range todos {
		if td.Schedule == ScheduleSomeday {
			t.Error("expected someday task to be excluded from default list")
		}
	}
}

func TestList_IncludeSomeday(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	s.Add("later task", PrioMedium, nil, nil, nil, ScheduleLater)
	s.Add("someday task", PrioMedium, nil, nil, nil, ScheduleSomeday)

	// With IncludeSomeday=true: all 2 todos
	todos, err := s.List(ListOptions{AllProjects: true, IncludeSomeday: true})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(todos) != 2 {
		t.Fatalf("expected 2 todos with IncludeSomeday, got %d", len(todos))
	}

	found := false
	for _, td := range todos {
		if td.Schedule == ScheduleSomeday {
			found = true
		}
	}
	if !found {
		t.Error("expected someday task in list with IncludeSomeday=true")
	}
}

func TestAdd_WithSchedule(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	id, err := s.Add("urgent task", PrioHigh, nil, nil, nil, ScheduleToday)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	got, err := s.Get(id)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Schedule != ScheduleToday {
		t.Fatalf("expected schedule %q, got %q", ScheduleToday, got.Schedule)
	}
}

func TestScheduleLabel(t *testing.T) {
	tests := []struct {
		schedule string
		want     string
	}{
		{ScheduleToday, "today"},
		{ScheduleSoon, "soon"},
		{ScheduleLater, "later"},
		{ScheduleSomeday, "someday"},
		{"unknown", "later"},
	}
	for _, tt := range tests {
		got := ScheduleLabel(tt.schedule)
		if got != tt.want {
			t.Errorf("ScheduleLabel(%q) = %q, want %q", tt.schedule, got, tt.want)
		}
	}
}
