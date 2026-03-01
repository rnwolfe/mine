package dig_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/rnwolfe/mine/internal/dig"
	_ "modernc.org/sqlite"
)

// openTestDB creates an in-memory SQLite database with the tables required by dig.Store.
// The schema matches production: foreign keys are enabled and dig_sessions.todo_id
// references todos(id) with ON DELETE SET NULL.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	migrations := []string{
		`PRAGMA foreign_keys = ON`,
		`CREATE TABLE IF NOT EXISTS todos (
			id INTEGER PRIMARY KEY AUTOINCREMENT
		)`,
		`CREATE TABLE IF NOT EXISTS dig_sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			todo_id INTEGER REFERENCES todos(id) ON DELETE SET NULL,
			duration_secs INTEGER NOT NULL,
			completed INTEGER DEFAULT 0,
			started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			ended_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS streaks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			current INTEGER DEFAULT 0,
			longest INTEGER DEFAULT 0,
			last_date TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS kv (
			key TEXT PRIMARY KEY,
			value TEXT,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
	}
	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			t.Fatalf("migration: %v", err)
		}
	}
	return db
}

func TestRecordSession_InsertsRow(t *testing.T) {
	db := openTestDB(t)
	s := dig.NewStore(db)

	_, err := s.RecordSession(25*time.Minute, nil, true, time.Now().Add(-25*time.Minute))
	if err != nil {
		t.Fatalf("RecordSession: %v", err)
	}

	var count, durationSecs, completed int
	if err := db.QueryRow(`SELECT COUNT(*), duration_secs, completed FROM dig_sessions`).Scan(&count, &durationSecs, &completed); err != nil {
		t.Fatalf("scan row: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 row, got %d", count)
	}
	if durationSecs != 1500 {
		t.Fatalf("expected 1500 secs, got %d", durationSecs)
	}
	if completed != 1 {
		t.Fatalf("expected completed=1, got %d", completed)
	}
}

func TestRecordSession_NotCompleted(t *testing.T) {
	db := openTestDB(t)
	s := dig.NewStore(db)

	if _, err := s.RecordSession(10*time.Minute, nil, false, time.Now().Add(-10*time.Minute)); err != nil {
		t.Fatalf("RecordSession: %v", err)
	}

	var completed int
	if err := db.QueryRow(`SELECT completed FROM dig_sessions`).Scan(&completed); err != nil {
		t.Fatalf("scan completed: %v", err)
	}
	if completed != 0 {
		t.Fatalf("expected completed=0, got %d", completed)
	}
}

func TestRecordSession_AccumulatesKV(t *testing.T) {
	db := openTestDB(t)
	s := dig.NewStore(db)

	total, err := s.RecordSession(25*time.Minute, nil, true, time.Now().Add(-25*time.Minute))
	if err != nil {
		t.Fatalf("RecordSession: %v", err)
	}
	if total != 25 {
		t.Fatalf("expected 25 total mins, got %d", total)
	}

	total, err = s.RecordSession(30*time.Minute, nil, true, time.Now().Add(-30*time.Minute))
	if err != nil {
		t.Fatalf("RecordSession (2nd): %v", err)
	}
	if total != 55 {
		t.Fatalf("expected 55 total mins after two sessions, got %d", total)
	}
}

func TestRecordSession_WithTodoID(t *testing.T) {
	db := openTestDB(t)
	s := dig.NewStore(db)

	// Insert the todo row so the FK constraint is satisfied.
	if _, err := db.Exec(`INSERT INTO todos (id) VALUES (42)`); err != nil {
		t.Fatalf("insert todo: %v", err)
	}

	todoID := 42
	if _, err := s.RecordSession(25*time.Minute, &todoID, true, time.Now().Add(-25*time.Minute)); err != nil {
		t.Fatalf("RecordSession: %v", err)
	}

	var storedTodoID int
	if err := db.QueryRow(`SELECT todo_id FROM dig_sessions WHERE todo_id IS NOT NULL`).Scan(&storedTodoID); err != nil {
		t.Fatalf("querying todo_id: %v", err)
	}
	if storedTodoID != todoID {
		t.Fatalf("expected todo_id=%d, got %d", todoID, storedTodoID)
	}
}

func TestUpdateStreak_FirstSession(t *testing.T) {
	db := openTestDB(t)
	s := dig.NewStore(db)
	today := time.Now().Format("2006-01-02")

	if err := s.UpdateStreak(today); err != nil {
		t.Fatalf("UpdateStreak: %v", err)
	}

	var current, longest int
	var lastDate string
	if err := db.QueryRow(`SELECT current, longest, last_date FROM streaks WHERE name = 'dig'`).Scan(&current, &longest, &lastDate); err != nil {
		t.Fatalf("scan streak: %v", err)
	}
	if current != 1 {
		t.Errorf("current = %d, want 1", current)
	}
	if longest != 1 {
		t.Errorf("longest = %d, want 1", longest)
	}
	if lastDate != today {
		t.Errorf("last_date = %q, want %q", lastDate, today)
	}
}

func TestUpdateStreak_SameDayIdempotent(t *testing.T) {
	db := openTestDB(t)
	s := dig.NewStore(db)
	today := time.Now().Format("2006-01-02")

	db.Exec(`INSERT INTO streaks (name, current, longest, last_date) VALUES ('dig', 3, 5, ?)`, today)

	if err := s.UpdateStreak(today); err != nil {
		t.Fatalf("UpdateStreak: %v", err)
	}

	var current int
	if err := db.QueryRow(`SELECT current FROM streaks WHERE name = 'dig'`).Scan(&current); err != nil {
		t.Fatalf("scan current: %v", err)
	}
	if current != 3 {
		t.Errorf("current = %d, want 3 (same-day should not increment)", current)
	}
}

func TestUpdateStreak_ConsecutiveDay(t *testing.T) {
	db := openTestDB(t)
	s := dig.NewStore(db)
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	today := time.Now().Format("2006-01-02")

	db.Exec(`INSERT INTO streaks (name, current, longest, last_date) VALUES ('dig', 1, 1, ?)`, yesterday)

	if err := s.UpdateStreak(today); err != nil {
		t.Fatalf("UpdateStreak: %v", err)
	}

	var current, longest int
	if err := db.QueryRow(`SELECT current, longest FROM streaks WHERE name = 'dig'`).Scan(&current, &longest); err != nil {
		t.Fatalf("scan streak: %v", err)
	}
	if current != 2 {
		t.Errorf("current = %d, want 2", current)
	}
	if longest != 2 {
		t.Errorf("longest = %d, want 2", longest)
	}
}

func TestUpdateStreak_BrokenStreak(t *testing.T) {
	db := openTestDB(t)
	s := dig.NewStore(db)
	twoDaysAgo := time.Now().AddDate(0, 0, -2).Format("2006-01-02")
	today := time.Now().Format("2006-01-02")

	db.Exec(`INSERT INTO streaks (name, current, longest, last_date) VALUES ('dig', 5, 10, ?)`, twoDaysAgo)

	if err := s.UpdateStreak(today); err != nil {
		t.Fatalf("UpdateStreak: %v", err)
	}

	var current, longest int
	if err := db.QueryRow(`SELECT current, longest FROM streaks WHERE name = 'dig'`).Scan(&current, &longest); err != nil {
		t.Fatalf("scan streak: %v", err)
	}
	if current != 1 {
		t.Errorf("current = %d, want 1 (streak broken)", current)
	}
	if longest != 10 {
		t.Errorf("longest = %d, want 10 (preserved)", longest)
	}
}

func TestGetStats_NoSessions(t *testing.T) {
	db := openTestDB(t)
	s := dig.NewStore(db)

	stats, err := s.GetStats()
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.CurrentStreak != 0 {
		t.Errorf("CurrentStreak = %d, want 0 for empty db", stats.CurrentStreak)
	}
	if stats.TotalMins != 0 {
		t.Errorf("TotalMins = %d, want 0 for empty db", stats.TotalMins)
	}
	if stats.SessionCount != 0 {
		t.Errorf("SessionCount = %d, want 0 for empty db", stats.SessionCount)
	}
}

func TestGetStats_WithData(t *testing.T) {
	db := openTestDB(t)
	s := dig.NewStore(db)
	today := time.Now().Format("2006-01-02")

	db.Exec(`INSERT INTO streaks (name, current, longest, last_date) VALUES ('dig', 3, 7, ?)`, today)
	db.Exec(`INSERT OR REPLACE INTO kv (key, value, updated_at) VALUES ('dig_total_mins', '150', CURRENT_TIMESTAMP)`)
	db.Exec(`INSERT INTO dig_sessions (duration_secs, completed) VALUES (1500, 1)`)
	db.Exec(`INSERT INTO dig_sessions (duration_secs, completed) VALUES (1800, 1)`)

	stats, err := s.GetStats()
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.CurrentStreak != 3 {
		t.Errorf("CurrentStreak = %d, want 3", stats.CurrentStreak)
	}
	if stats.LongestStreak != 7 {
		t.Errorf("LongestStreak = %d, want 7", stats.LongestStreak)
	}
	if stats.TotalMins != 150 {
		t.Errorf("TotalMins = %d, want 150", stats.TotalMins)
	}
	if stats.SessionCount != 2 {
		t.Errorf("SessionCount = %d, want 2", stats.SessionCount)
	}
	if stats.LastDate != today {
		t.Errorf("LastDate = %q, want %q", stats.LastDate, today)
	}
}

func TestGetStats_LinkedTasks(t *testing.T) {
	db := openTestDB(t)
	s := dig.NewStore(db)
	today := time.Now().Format("2006-01-02")

	// Insert todos so the FK constraint on dig_sessions.todo_id is satisfied.
	db.Exec(`INSERT INTO todos (id) VALUES (1)`)
	db.Exec(`INSERT INTO todos (id) VALUES (2)`)

	db.Exec(`INSERT INTO streaks (name, current, longest, last_date) VALUES ('dig', 1, 1, ?)`, today)
	db.Exec(`INSERT INTO dig_sessions (todo_id, duration_secs, completed) VALUES (1, 1500, 1)`)
	db.Exec(`INSERT INTO dig_sessions (todo_id, duration_secs, completed) VALUES (1, 1800, 1)`)
	db.Exec(`INSERT INTO dig_sessions (todo_id, duration_secs, completed) VALUES (2, 900, 1)`)

	stats, err := s.GetStats()
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.SessionCount != 3 {
		t.Errorf("SessionCount = %d, want 3", stats.SessionCount)
	}
	if stats.LinkedTasks != 2 {
		t.Errorf("LinkedTasks = %d, want 2 (distinct todo IDs)", stats.LinkedTasks)
	}
}
