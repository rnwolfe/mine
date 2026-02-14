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
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		completed_at DATETIME
	)`)
	if err != nil {
		t.Fatal(err)
	}

	return db
}

func TestAddAndList(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	id, err := s.Add("Test todo", PrioHigh, []string{"test"}, nil)
	if err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	if id != 1 {
		t.Fatalf("expected id 1, got %d", id)
	}

	todos, err := s.List(false)
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

	id, _ := s.Add("Complete me", PrioMedium, nil, nil)
	if err := s.Complete(id); err != nil {
		t.Fatalf("Complete failed: %v", err)
	}

	// Should not appear in default list
	todos, _ := s.List(false)
	if len(todos) != 0 {
		t.Fatalf("expected 0 open todos, got %d", len(todos))
	}

	// Should appear with showDone=true
	todos, _ = s.List(true)
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

	id, _ := s.Add("Delete me", PrioLow, nil, nil)
	if err := s.Delete(id); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	todos, _ := s.List(true)
	if len(todos) != 0 {
		t.Fatalf("expected 0 todos, got %d", len(todos))
	}
}

func TestCount(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	s := NewStore(db)

	s.Add("One", PrioMedium, nil, nil)
	s.Add("Two", PrioHigh, nil, nil)

	yesterday := time.Now().AddDate(0, 0, -1)
	s.Add("Overdue", PrioCrit, nil, &yesterday)

	open, total, overdue, err := s.Count()
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

	id, _ := s.Add("Original", PrioLow, nil, nil)

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
