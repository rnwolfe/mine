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

	id, err := s.Add("Test todo", PrioHigh, []string{"test"}, nil, nil)
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
}

func TestComplete(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	id, _ := s.Add("Complete me", PrioMedium, nil, nil, nil)
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

	id, _ := s.Add("Delete me", PrioLow, nil, nil, nil)
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

	s.Add("One", PrioMedium, nil, nil, nil)
	s.Add("Two", PrioHigh, nil, nil, nil)

	yesterday := time.Now().AddDate(0, 0, -1)
	s.Add("Overdue", PrioCrit, nil, &yesterday, nil)

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

	id, _ := s.Add("Original", PrioLow, nil, nil, nil)

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
	id, err := s.Add("project task", PrioMedium, nil, nil, &projPath)
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

	s.Add("global task", PrioMedium, nil, nil, nil)
	s.Add("alpha task", PrioMedium, nil, nil, &projA)
	s.Add("beta task", PrioMedium, nil, nil, &projB)

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

	s.Add("global task", PrioMedium, nil, nil, nil)
	s.Add("alpha task", PrioHigh, nil, nil, &projA)
	s.Add("other project task", PrioLow, nil, nil, strPtr("/projects/other"))

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
	id, _ := s.Add("alpha done", PrioMedium, nil, nil, &projA)
	s.Complete(id)
	s.Add("alpha open", PrioMedium, nil, nil, &projA)

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
