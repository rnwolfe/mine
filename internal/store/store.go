package store

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/rnwolfe/mine/internal/config"
	_ "modernc.org/sqlite"
)

// DB wraps the SQLite connection.
type DB struct {
	conn *sql.DB
}

// Open opens (or creates) the mine database.
func Open() (*DB, error) {
	paths := config.GetPaths()
	if err := paths.EnsureDirs(); err != nil {
		return nil, fmt.Errorf("creating data dirs: %w", err)
	}

	conn, err := sql.Open("sqlite", paths.DBFile+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Performance pragmas
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA cache_size=-64000", // 64MB cache
		"PRAGMA foreign_keys=ON",
		"PRAGMA temp_store=MEMORY",
	}
	for _, p := range pragmas {
		if _, err := conn.Exec(p); err != nil {
			conn.Close()
			return nil, fmt.Errorf("setting pragma %q: %w", p, err)
		}
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return db, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

// Conn returns the raw sql.DB for direct queries.
func (db *DB) Conn() *sql.DB {
	return db.conn
}

// migrate runs all schema migrations.
func (db *DB) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS migrations (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		// Todos table
		`CREATE TABLE IF NOT EXISTS todos (
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
		)`,
		// Growth tracking
		`CREATE TABLE IF NOT EXISTS goals (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			category TEXT DEFAULT 'general',
			target_value REAL DEFAULT 0,
			current_value REAL DEFAULT 0,
			unit TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		// Streaks
		`CREATE TABLE IF NOT EXISTS streaks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			current INTEGER DEFAULT 0,
			longest INTEGER DEFAULT 0,
			last_date TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		// Key-value store for misc state
		`CREATE TABLE IF NOT EXISTS kv (
			key TEXT PRIMARY KEY,
			value TEXT,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		// Per-project active environment profile state
		`CREATE TABLE IF NOT EXISTS env_projects (
			project_path TEXT PRIMARY KEY,
			active_profile TEXT NOT NULL DEFAULT 'local',
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		// Project registry
		`CREATE TABLE IF NOT EXISTS projects (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			path TEXT NOT NULL UNIQUE,
			last_accessed TEXT,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_projects_name ON projects(name)`,
		`CREATE INDEX IF NOT EXISTS idx_projects_path ON projects(path)`,
	}

	for _, m := range migrations {
		if _, err := db.conn.Exec(m); err != nil {
			return fmt.Errorf("migration failed: %w\nSQL: %s", err, m)
		}
	}

	// ALTER TABLE migrations cannot use IF NOT EXISTS — handle idempotently.
	// SQLite raises "duplicate column name: X" when a column already exists.
	// The modernc.org/sqlite pure-Go driver preserves this exact error string
	// (it mirrors the SQLite C library wording), so the string match is stable.
	// See: https://www.sqlite.org/lang_altertable.html
	alterMigrations := []string{
		`ALTER TABLE todos ADD COLUMN project_path TEXT`,
	}
	for _, m := range alterMigrations {
		if _, err := db.conn.Exec(m); err != nil {
			if !strings.Contains(err.Error(), "duplicate column name") {
				return fmt.Errorf("migration failed: %w\nSQL: %s", err, m)
			}
		}
	}

	// Indexes for new columns.
	if _, err := db.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_todos_project_path ON todos(project_path)`); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	return nil
}
