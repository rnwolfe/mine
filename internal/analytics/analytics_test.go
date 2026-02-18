package analytics

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// testDB creates a temporary SQLite database with the kv table.
func testDB(t *testing.T) *sql.DB {
	t.Helper()
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS kv (
		key TEXT PRIMARY KEY,
		value TEXT,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestBuildPayload_Fields(t *testing.T) {
	// Capture date before building payload to avoid midnight rollover flake
	today := time.Now().Format("2006-01-02")
	p := BuildPayload("test-id-123", "todo")

	if p.InstallID != "test-id-123" {
		t.Errorf("InstallID = %q, want %q", p.InstallID, "test-id-123")
	}
	if p.Command != "todo" {
		t.Errorf("Command = %q, want %q", p.Command, "todo")
	}
	if p.OS != runtime.GOOS {
		t.Errorf("OS = %q, want %q", p.OS, runtime.GOOS)
	}
	if p.Arch != runtime.GOARCH {
		t.Errorf("Arch = %q, want %q", p.Arch, runtime.GOARCH)
	}
	if p.Date != today {
		t.Errorf("Date = %q, want %q", p.Date, today)
	}
}

func TestBuildPayload_NoExtraFields(t *testing.T) {
	p := BuildPayload("id", "cmd")

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}

	expected := map[string]bool{
		"install_id": true,
		"version":    true,
		"os":         true,
		"arch":       true,
		"command":    true,
		"date":       true,
	}

	for key := range m {
		if !expected[key] {
			t.Errorf("unexpected field %q in payload", key)
		}
	}

	if len(m) != len(expected) {
		t.Errorf("payload has %d fields, want %d", len(m), len(expected))
	}
}

func TestPing_OptedOut(t *testing.T) {
	var called atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called.Store(true)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	if err := os.MkdirAll(filepath.Join(tmp, "mine"), 0o755); err != nil {
		t.Fatal(err)
	}

	db := testDB(t)

	// enabled = false
	Ping(db, "todo", false, srv.URL)

	if called.Load() {
		t.Error("Ping should not send when disabled")
	}
}

func TestPing_SendsPayload(t *testing.T) {
	var received Payload
	var called atomic.Bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called.Store(true)
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("expected application/json, got %s", ct)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("failed to decode payload: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	if err := os.MkdirAll(filepath.Join(tmp, "mine"), 0o755); err != nil {
		t.Fatal(err)
	}

	db := testDB(t)

	Ping(db, "craft", true, srv.URL)

	if !called.Load() {
		t.Fatal("expected HTTP call to analytics endpoint")
	}

	if received.Command != "craft" {
		t.Errorf("Command = %q, want %q", received.Command, "craft")
	}
	if !isValidUUID(received.InstallID) {
		t.Errorf("InstallID should be a valid UUID, got %q", received.InstallID)
	}
	if received.OS != runtime.GOOS {
		t.Errorf("OS = %q, want %q", received.OS, runtime.GOOS)
	}
}

func TestPing_DailyDedup(t *testing.T) {
	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	if err := os.MkdirAll(filepath.Join(tmp, "mine"), 0o755); err != nil {
		t.Fatal(err)
	}

	db := testDB(t)

	// First call should send
	Ping(db, "todo", true, srv.URL)
	if callCount.Load() != 1 {
		t.Fatalf("first call: expected 1 HTTP call, got %d", callCount.Load())
	}

	// Second call same command, same day — should be deduped
	Ping(db, "todo", true, srv.URL)
	if callCount.Load() != 1 {
		t.Fatalf("second call: expected 1 HTTP call (deduped), got %d", callCount.Load())
	}

	// Different command same day — should send
	Ping(db, "craft", true, srv.URL)
	if callCount.Load() != 2 {
		t.Fatalf("different command: expected 2 HTTP calls, got %d", callCount.Load())
	}
}

func TestPing_DedupResetsNextDay(t *testing.T) {
	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	if err := os.MkdirAll(filepath.Join(tmp, "mine"), 0o755); err != nil {
		t.Fatal(err)
	}

	db := testDB(t)

	// Simulate a ping from yesterday by manually setting the dedup key
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	_, err := db.Exec(
		`INSERT INTO kv (key, value) VALUES ('analytics:last_ping:todo', ?)`,
		yesterday,
	)
	if err != nil {
		t.Fatal(err)
	}

	// Should send because yesterday's dedup is stale
	Ping(db, "todo", true, srv.URL)
	if callCount.Load() != 1 {
		t.Fatalf("expected 1 HTTP call after day reset, got %d", callCount.Load())
	}
}

func TestPing_NetworkFailureSilent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	if err := os.MkdirAll(filepath.Join(tmp, "mine"), 0o755); err != nil {
		t.Fatal(err)
	}

	db := testDB(t)

	// Point to a closed server — should not panic or return error
	Ping(db, "todo", true, "http://127.0.0.1:1") // port 1 will refuse
}

func TestPing_ServerErrorNoDedupWrite(t *testing.T) {
	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmp)
	if err := os.MkdirAll(filepath.Join(tmp, "mine"), 0o755); err != nil {
		t.Fatal(err)
	}

	db := testDB(t)

	// First call — server returns 500
	Ping(db, "todo", true, srv.URL)
	if callCount.Load() != 1 {
		t.Fatalf("first call: expected 1 HTTP call, got %d", callCount.Load())
	}

	// Verify dedup key was NOT written
	var val string
	err := db.QueryRow("SELECT value FROM kv WHERE key = 'analytics:last_ping:todo'").Scan(&val)
	if err == nil {
		t.Fatalf("dedup key should not be set after server error, but got %q", val)
	}

	// Second call should retry (not deduped)
	Ping(db, "todo", true, srv.URL)
	if callCount.Load() != 2 {
		t.Fatalf("second call: expected 2 HTTP calls (no dedup after error), got %d", callCount.Load())
	}
}

func TestShouldShowNotice_FirstTime(t *testing.T) {
	db := testDB(t)

	if !ShouldShowNotice(db) {
		t.Error("expected ShouldShowNotice to return true on first call")
	}
}

func TestShouldShowNotice_OnlyOnceAfterMark(t *testing.T) {
	db := testDB(t)

	if !ShouldShowNotice(db) {
		t.Error("expected ShouldShowNotice to return true before marking")
	}

	MarkNoticeShown(db)

	if ShouldShowNotice(db) {
		t.Error("expected ShouldShowNotice to return false after MarkNoticeShown")
	}
}

func TestShouldShowNotice_TrueUntilMarked(t *testing.T) {
	db := testDB(t)

	// Multiple calls before marking should all return true
	if !ShouldShowNotice(db) {
		t.Error("expected true on first call")
	}
	if !ShouldShowNotice(db) {
		t.Error("expected true on second call (not yet marked)")
	}

	MarkNoticeShown(db)

	if ShouldShowNotice(db) {
		t.Error("expected false after marking")
	}
}
