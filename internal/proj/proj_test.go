package proj

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func setupStore(t *testing.T) (*Store, *sql.DB) {
	t.Helper()

	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, "data"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmp, "cache"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmp, "state"))

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`CREATE TABLE projects (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		path TEXT NOT NULL UNIQUE,
		last_accessed TEXT,
		created_at TEXT
	)`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE kv (
		key TEXT PRIMARY KEY,
		value TEXT,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		t.Fatal(err)
	}

	return NewStore(db), db
}

func TestAddAndList(t *testing.T) {
	s, _ := setupStore(t)
	origBranch := gitBranchAtPath
	gitBranchAtPath = func(string) string { return "main" }
	t.Cleanup(func() { gitBranchAtPath = origBranch })

	dir := t.TempDir()
	p, err := s.Add(dir)
	if err != nil {
		t.Fatalf("Add() error: %v", err)
	}
	if p.Name == "" || p.Path == "" {
		t.Fatalf("expected name/path to be populated, got %#v", p)
	}

	got, err := s.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 project, got %d", len(got))
	}
	if got[0].Branch != "main" {
		t.Fatalf("expected branch main, got %q", got[0].Branch)
	}
}

func TestAddDuplicate(t *testing.T) {
	s, _ := setupStore(t)
	dir := t.TempDir()
	if _, err := s.Add(dir); err != nil {
		t.Fatalf("first Add() failed: %v", err)
	}
	_, err := s.Add(dir)
	if !errors.Is(err, ErrProjectExists) {
		t.Fatalf("expected ErrProjectExists, got %v", err)
	}
}

func TestRemove(t *testing.T) {
	s, _ := setupStore(t)
	dir := t.TempDir()
	p, err := s.Add(dir)
	if err != nil {
		t.Fatalf("Add() error: %v", err)
	}
	if err := s.Remove(p.Name); err != nil {
		t.Fatalf("Remove() error: %v", err)
	}

	projects, err := s.List()
	if err != nil {
		t.Fatalf("List() error: %v", err)
	}
	if len(projects) != 0 {
		t.Fatalf("expected no projects after remove, got %d", len(projects))
	}
}

func TestScanDepth(t *testing.T) {
	s, _ := setupStore(t)
	root := t.TempDir()

	mkRepo := func(path string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Join(path, ".git"), 0o755); err != nil {
			t.Fatalf("mkdir .git: %v", err)
		}
	}

	level1 := filepath.Join(root, "l1")
	level2 := filepath.Join(root, "nested", "l2")
	level4 := filepath.Join(root, "deep", "a", "b", "l4")
	mkRepo(level1)
	mkRepo(level2)
	mkRepo(level4)

	added, err := s.Scan(root, 3)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if len(added) != 2 {
		t.Fatalf("expected 2 repos at depth<=3, got %d", len(added))
	}

	added, err = s.Scan(root, 4)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if len(added) != 1 {
		t.Fatalf("expected one additional repo at depth 4, got %d", len(added))
	}
}

func TestOpenTracksCurrentAndPrevious(t *testing.T) {
	s, _ := setupStore(t)

	first, err := s.Add(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	second, err := s.Add(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := s.Open(first.Name); err != nil {
		t.Fatalf("open first failed: %v", err)
	}
	res, err := s.Open(second.Name)
	if err != nil {
		t.Fatalf("open second failed: %v", err)
	}
	if res.Previous != first.Name {
		t.Fatalf("expected previous %q, got %q", first.Name, res.Previous)
	}

	prev, err := s.PreviousName()
	if err != nil {
		t.Fatalf("PreviousName() error: %v", err)
	}
	if prev != first.Name {
		t.Fatalf("expected previous state %q, got %q", first.Name, prev)
	}

	openPrev, err := s.OpenPrevious()
	if err != nil {
		t.Fatalf("OpenPrevious() error: %v", err)
	}
	if openPrev.Project.Name != first.Name {
		t.Fatalf("expected OpenPrevious to resolve %q, got %q", first.Name, openPrev.Project.Name)
	}
}

func TestProjectConfigSettings(t *testing.T) {
	s, _ := setupStore(t)
	p, err := s.Add(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	if err := s.SetSetting(p.Name, "ssh_host", "prod-box"); err != nil {
		t.Fatalf("SetSetting ssh_host failed: %v", err)
	}
	if err := s.SetSetting(p.Name, "env_file", ".env.local"); err != nil {
		t.Fatalf("SetSetting env_file failed: %v", err)
	}

	got, err := s.GetSetting(p.Name, "ssh_host")
	if err != nil {
		t.Fatalf("GetSetting() error: %v", err)
	}
	if got != "prod-box" {
		t.Fatalf("expected ssh_host prod-box, got %q", got)
	}

	cfgPath := s.paths.ProjectsFile
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("projects.toml missing: %v", err)
	}
	if len(data) == 0 {
		t.Fatalf("projects.toml should not be empty")
	}
}

func TestAddRejectsFilePath(t *testing.T) {
	s, _ := setupStore(t)

	root := t.TempDir()
	file := filepath.Join(root, "README.md")
	if err := os.WriteFile(file, []byte("hi"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if _, err := s.Add(file); err == nil {
		t.Fatal("expected error for non-directory path")
	}
}

func TestAddRejectsRootPathName(t *testing.T) {
	s, _ := setupStore(t)

	if _, err := s.Add(string(filepath.Separator)); err == nil {
		t.Fatal("expected error for root path project name")
	}
}

func TestScanRejectsNegativeDepth(t *testing.T) {
	s, _ := setupStore(t)

	if _, err := s.Scan(t.TempDir(), -1); err == nil {
		t.Fatal("expected error for negative scan depth")
	}
}

func TestOpenPreviousWithoutState(t *testing.T) {
	s, _ := setupStore(t)

	if _, err := s.OpenPrevious(); err == nil {
		t.Fatal("expected error when previous project is not tracked")
	}
}

func TestOpenPropagatesKVStateErrors(t *testing.T) {
	s, db := setupStore(t)
	p, err := s.Add(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec(`DROP TABLE kv`); err != nil {
		t.Fatalf("drop kv: %v", err)
	}

	if _, err := s.Open(p.Name); err == nil {
		t.Fatal("expected error when kv table is unavailable")
	}
}

func TestSetSettingValidation(t *testing.T) {
	s, _ := setupStore(t)
	p, err := s.Add(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	if err := s.SetSetting("", "ssh_host", "prod-box"); err == nil {
		t.Fatal("expected error for missing project name")
	}

	if err := s.SetSetting("does-not-exist", "ssh_host", "prod-box"); err == nil {
		t.Fatal("expected error for unknown project")
	}

	if err := s.SetSetting(p.Name, "not_a_key", "x"); err == nil {
		t.Fatal("expected error for unknown key")
	}
}

func TestGetSettingUnknownKey(t *testing.T) {
	s, _ := setupStore(t)
	p, err := s.Add(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetSetting(p.Name, "ssh_host", "prod-box"); err != nil {
		t.Fatalf("SetSetting ssh_host failed: %v", err)
	}

	if _, err := s.GetSetting(p.Name, "not_a_key"); err == nil {
		t.Fatal("expected error for unknown key")
	}
}

func TestGetSettingUnknownKeyWithoutProjectConfig(t *testing.T) {
	s, _ := setupStore(t)

	if _, err := s.GetSetting("missing", "not_a_key"); err == nil {
		t.Fatal("expected error for unknown key even when project has no settings")
	}
}
