package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/rnwolfe/mine/internal/store"
	"github.com/rnwolfe/mine/internal/todo"
)

// digTestEnv sets up isolated XDG dirs for dig tests.
func digTestEnv(t *testing.T) {
	t.Helper()
	configTestEnv(t)
}

func TestRunDig_InvalidTodoID(t *testing.T) {
	digTestEnv(t)
	digTodoID = 999
	defer func() { digTodoID = 0 }()

	err := runDig(nil, []string{})
	if err == nil {
		t.Fatal("expected error for non-existent todo ID")
	}
	if !strings.Contains(err.Error(), "#999") {
		t.Errorf("error should mention todo #999, got: %v", err)
	}
}

func TestRunDig_InvalidDuration(t *testing.T) {
	digTestEnv(t)
	digTodoID = 0

	err := runDig(nil, []string{"notaduration"})
	if err == nil {
		t.Fatal("expected error for invalid duration")
	}
	if !strings.Contains(err.Error(), "invalid duration") {
		t.Errorf("expected 'invalid duration' in error, got: %v", err)
	}
}

func TestRecordDigSession_UpdatesKV(t *testing.T) {
	digTestEnv(t)

	recordDigSession(25*time.Minute, nil, true, time.Now().Add(-25*time.Minute))

	db, err := store.Open()
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	defer db.Close()

	var totalMins int
	db.Conn().QueryRow(`SELECT CAST(value AS INTEGER) FROM kv WHERE key = 'dig_total_mins'`).Scan(&totalMins)
	if totalMins != 25 {
		t.Fatalf("expected 25 total mins, got %d", totalMins)
	}
}

func TestRecordDigSession_InsertsDigSession(t *testing.T) {
	digTestEnv(t)

	recordDigSession(30*time.Minute, nil, true, time.Now().Add(-30*time.Minute))

	db, err := store.Open()
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	defer db.Close()

	var count int
	db.Conn().QueryRow(`SELECT COUNT(*) FROM dig_sessions`).Scan(&count)
	if count != 1 {
		t.Fatalf("expected 1 dig_session, got %d", count)
	}

	var durationSecs int
	var completed int
	db.Conn().QueryRow(`SELECT duration_secs, completed FROM dig_sessions`).Scan(&durationSecs, &completed)
	if durationSecs != 1800 {
		t.Fatalf("expected 1800 secs, got %d", durationSecs)
	}
	if completed != 1 {
		t.Fatalf("expected completed=1, got %d", completed)
	}
}

func TestRecordDigSession_WithTodoID(t *testing.T) {
	digTestEnv(t)

	// Create a real todo first.
	db, err := store.Open()
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	ts := todo.NewStore(db.Conn())
	todoID, err := ts.Add("test task", "", todo.PrioMedium, nil, nil, nil, todo.ScheduleLater, todo.RecurrenceNone)
	db.Close()
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	recordDigSession(25*time.Minute, &todoID, true, time.Now().Add(-25*time.Minute))

	db, err = store.Open()
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	defer db.Close()

	var storedTodoID int
	err = db.Conn().QueryRow(`SELECT todo_id FROM dig_sessions WHERE todo_id IS NOT NULL`).Scan(&storedTodoID)
	if err != nil {
		t.Fatalf("querying dig_session todo_id: %v", err)
	}
	if storedTodoID != todoID {
		t.Fatalf("expected todo_id=%d, got %d", todoID, storedTodoID)
	}
}

func TestRecordDigSession_EarlyCancel_MarkedNotCompleted(t *testing.T) {
	digTestEnv(t)

	// 10 minutes — counts (>= 5min), but not completed
	recordDigSession(10*time.Minute, nil, false, time.Now().Add(-10*time.Minute))

	db, err := store.Open()
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	defer db.Close()

	var completed int
	db.Conn().QueryRow(`SELECT completed FROM dig_sessions`).Scan(&completed)
	if completed != 0 {
		t.Fatalf("expected completed=0 for early cancel, got %d", completed)
	}
}

func TestFormatFocusTime(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{15 * time.Minute, "[15m]"},
		{30 * time.Minute, "[30m]"},
		{time.Hour, "[1h 0m]"},
		{time.Hour + 25*time.Minute, "[1h 25m]"},
		{2*time.Hour + 15*time.Minute, "[2h 15m]"},
	}
	for _, tt := range tests {
		got := formatFocusTime(tt.d)
		if got != tt.want {
			t.Errorf("formatFocusTime(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestPrintTodoList_ShowsFocusTime(t *testing.T) {
	digTestEnv(t)

	db, err := store.Open()
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	defer db.Close()

	ts := todo.NewStore(db.Conn())
	todoID, err := ts.Add("focused task", "", todo.PrioMedium, nil, nil, nil, todo.ScheduleLater, todo.RecurrenceNone)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	// Insert a dig session linked to this todo.
	if _, err := db.Conn().Exec(
		`INSERT INTO dig_sessions (todo_id, duration_secs, completed, ended_at) VALUES (?, ?, 1, CURRENT_TIMESTAMP)`,
		todoID, 1500, // 25 minutes
	); err != nil {
		t.Fatalf("insert dig_session: %v", err)
	}

	todos, err := ts.List(todo.ListOptions{AllProjects: true})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	out := captureStdout(t, func() {
		printTodoList(todos, ts, nil, false)
	})

	if !strings.Contains(out, "[25m]") {
		t.Errorf("expected [25m] focus time in output, got:\n%s", out)
	}
}

func TestPrintTodoList_OmitsFocusTimeWhenZero(t *testing.T) {
	digTestEnv(t)

	db, err := store.Open()
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	defer db.Close()

	ts := todo.NewStore(db.Conn())
	_, err = ts.Add("unfocused task", "", todo.PrioMedium, nil, nil, nil, todo.ScheduleLater, todo.RecurrenceNone)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	todos, err := ts.List(todo.ListOptions{AllProjects: true})
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	out := captureStdout(t, func() {
		printTodoList(todos, ts, nil, false)
	})

	if strings.Contains(out, "[0m]") {
		t.Errorf("should not show [0m] when no focus time, got:\n%s", out)
	}
}
