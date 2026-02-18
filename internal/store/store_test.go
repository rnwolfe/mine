package store

import (
	"os"
	"path/filepath"
	"testing"
)

// setupTestXDG sets XDG env vars to a temp directory for isolated testing.
func setupTestXDG(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", tmpDir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmpDir, "cache"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmpDir, "state"))
	return tmpDir
}

func TestOpenAndClose(t *testing.T) {
	tmpDir := setupTestXDG(t)

	db, err := Open()
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	if db.Conn() == nil {
		t.Fatal("Conn() returned nil")
	}

	// Verify database file was created
	dbPath := filepath.Join(tmpDir, "mine", "mine.db")
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("Database file not created at %s: %v", dbPath, err)
	}
}

func TestMigrationsCreateTables(t *testing.T) {
	setupTestXDG(t)

	db, err := Open()
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	// Check all expected tables exist
	tables := []string{"migrations", "todos", "goals", "streaks", "kv", "env_projects"}
	for _, table := range tables {
		var name string
		err := db.Conn().QueryRow(
			"SELECT name FROM sqlite_master WHERE type='table' AND name=?", table,
		).Scan(&name)
		if err != nil {
			t.Errorf("Table %q not found: %v", table, err)
		}
	}
}

func TestWALMode(t *testing.T) {
	setupTestXDG(t)

	db, err := Open()
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	var journalMode string
	err = db.Conn().QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		t.Fatalf("Querying journal_mode failed: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("Expected WAL mode, got %q", journalMode)
	}
}

func TestDoubleOpen(t *testing.T) {
	setupTestXDG(t)

	db1, err := Open()
	if err != nil {
		t.Fatalf("First Open failed: %v", err)
	}
	defer db1.Close()

	// Opening again should not fail (migrations are idempotent)
	db2, err := Open()
	if err != nil {
		t.Fatalf("Second Open failed: %v", err)
	}
	defer db2.Close()
}
