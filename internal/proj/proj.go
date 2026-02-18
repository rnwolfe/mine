package proj

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/rnwolfe/mine/internal/config"
)

var ErrProjectExists = errors.New("project already registered")

// Project is a registered project workspace.
type Project struct {
	Name         string
	Path         string
	LastAccessed time.Time
	Branch       string
}

// FilterValue implements tui.Item.
func (p Project) FilterValue() string { return p.Name + " " + p.Path }

// Title implements tui.Item.
func (p Project) Title() string { return p.Name }

// Description implements tui.Item.
func (p Project) Description() string {
	if p.Branch != "" {
		return p.Path + " (" + p.Branch + ")"
	}
	return p.Path
}

// OpenResult is returned by open operations.
type OpenResult struct {
	Project  Project
	Previous string
}

// Settings are persisted in projects.toml and keyed by project name.
type Settings struct {
	DefaultBranch string `toml:"default_branch,omitempty"`
	EnvFile       string `toml:"env_file,omitempty"`
	TmuxLayout    string `toml:"tmux_layout,omitempty"`
	SSHHost       string `toml:"ssh_host,omitempty"`
	SSHTunnel     string `toml:"ssh_tunnel,omitempty"`
}

type settingsFile struct {
	Projects map[string]Settings `toml:"projects"`
}

var gitBranchAtPath = func(path string) string {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	branch := strings.TrimSpace(string(out))
	if branch == "" || branch == "HEAD" {
		return ""
	}
	return branch
}

// Store owns project registry and state persistence.
type Store struct {
	db    *sql.DB
	paths config.Paths
}

func NewStore(db *sql.DB) *Store {
	return &Store{
		db:    db,
		paths: config.GetPaths(),
	}
}

func (s *Store) Add(path string) (*Project, error) {
	if strings.TrimSpace(path) == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("resolve cwd: %w", err)
		}
		path = cwd
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("stat path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("project path must be a directory")
	}

	name := filepath.Base(abs)
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("invalid project name")
	}

	var existing string
	err = s.db.QueryRow(`SELECT name FROM projects WHERE name = ? OR path = ?`, name, abs).Scan(&existing)
	if err == nil {
		return nil, fmt.Errorf("%w: %s", ErrProjectExists, existing)
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("check duplicates: %w", err)
	}

	_, err = s.db.Exec(
		`INSERT INTO projects (name, path, created_at) VALUES (?, ?, ?)`,
		name, abs, time.Now().UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return nil, fmt.Errorf("insert project: %w", err)
	}

	return &Project{Name: name, Path: abs}, nil
}

func (s *Store) Remove(name string) error {
	res, err := s.db.Exec(`DELETE FROM projects WHERE name = ?`, strings.TrimSpace(name))
	if err != nil {
		return fmt.Errorf("remove project: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("project %q not found", name)
	}
	return nil
}

func (s *Store) List() ([]Project, error) {
	rows, err := s.db.Query(`SELECT name, path, last_accessed FROM projects ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		var last sql.NullString
		if err := rows.Scan(&p.Name, &p.Path, &last); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		if last.Valid {
			p.LastAccessed = parseTime(last.String)
		}
		p.Branch = gitBranchAtPath(p.Path)
		projects = append(projects, p)
	}
	return projects, nil
}

func (s *Store) Get(name string) (*Project, error) {
	var p Project
	var last sql.NullString
	err := s.db.QueryRow(
		`SELECT name, path, last_accessed FROM projects WHERE name = ?`,
		strings.TrimSpace(name),
	).Scan(&p.Name, &p.Path, &last)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("project %q not found", name)
		}
		return nil, fmt.Errorf("load project: %w", err)
	}

	if last.Valid {
		p.LastAccessed = parseTime(last.String)
	}
	p.Branch = gitBranchAtPath(p.Path)
	return &p, nil
}

func (s *Store) Open(name string) (*OpenResult, error) {
	p, err := s.Get(name)
	if err != nil {
		return nil, err
	}

	current, _ := s.getKV("proj.current")
	if current != "" && current != p.Name {
		_ = s.setKV("proj.previous", current)
	}

	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := s.db.Exec(`UPDATE projects SET last_accessed = ? WHERE name = ?`, now, p.Name); err != nil {
		return nil, fmt.Errorf("update project access: %w", err)
	}
	if err := s.setKV("proj.current", p.Name); err != nil {
		return nil, err
	}

	p.LastAccessed = parseTime(now)
	return &OpenResult{Project: *p, Previous: current}, nil
}

func (s *Store) OpenPrevious() (*OpenResult, error) {
	prev, err := s.getKV("proj.previous")
	if err != nil {
		return nil, err
	}
	if prev == "" {
		return nil, fmt.Errorf("no previous project tracked yet")
	}
	return s.Open(prev)
}

func (s *Store) Current() (*Project, error) {
	current, err := s.getKV("proj.current")
	if err != nil {
		return nil, err
	}
	if current == "" {
		return nil, nil
	}
	return s.Get(current)
}

func (s *Store) CurrentName() (string, error) {
	return s.getKV("proj.current")
}

func (s *Store) PreviousName() (string, error) {
	return s.getKV("proj.previous")
}

func (s *Store) Scan(root string, depth int) ([]Project, error) {
	if strings.TrimSpace(root) == "" {
		root = "."
	}
	if depth < 0 {
		return nil, fmt.Errorf("depth must be >= 0")
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve scan root: %w", err)
	}

	var added []Project
	err = filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil
		}
		if !d.IsDir() {
			return nil
		}

		rel, _ := filepath.Rel(absRoot, path)
		level := depthFromRel(rel)
		if level > depth {
			return filepath.SkipDir
		}

		if d.Name() == ".git" {
			return filepath.SkipDir
		}

		if level > 0 && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}

		if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
			p, err := s.Add(path)
			if err != nil {
				if errors.Is(err, ErrProjectExists) {
					return filepath.SkipDir
				}
				return nil
			}
			added = append(added, *p)
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan projects: %w", err)
	}

	sort.Slice(added, func(i, j int) bool { return added[i].Name < added[j].Name })
	return added, nil
}

func SupportedConfigKeys() []string {
	return []string{"default_branch", "env_file", "tmux_layout", "ssh_host", "ssh_tunnel"}
}

func (s *Store) GetSetting(projectName, key string) (string, error) {
	sf, err := s.readSettingsFile()
	if err != nil {
		return "", err
	}
	cfg, ok := sf.Projects[projectName]
	if !ok {
		return "", nil
	}
	return settingValue(cfg, key)
}

func (s *Store) SetSetting(projectName, key, value string) error {
	if strings.TrimSpace(projectName) == "" {
		return fmt.Errorf("project name is required")
	}
	if _, err := s.Get(projectName); err != nil {
		return err
	}

	sf, err := s.readSettingsFile()
	if err != nil {
		return err
	}
	if sf.Projects == nil {
		sf.Projects = map[string]Settings{}
	}

	cfg := sf.Projects[projectName]
	if err := applySetting(&cfg, key, value); err != nil {
		return err
	}
	sf.Projects[projectName] = cfg

	return s.writeSettingsFile(sf)
}

func (s *Store) readSettingsFile() (*settingsFile, error) {
	data, err := os.ReadFile(s.paths.ProjectsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &settingsFile{Projects: map[string]Settings{}}, nil
		}
		return nil, fmt.Errorf("read projects config: %w", err)
	}

	var sf settingsFile
	if err := toml.Unmarshal(data, &sf); err != nil {
		return nil, fmt.Errorf("parse projects config: %w", err)
	}
	if sf.Projects == nil {
		sf.Projects = map[string]Settings{}
	}
	return &sf, nil
}

func (s *Store) writeSettingsFile(sf *settingsFile) error {
	if err := s.paths.EnsureDirs(); err != nil {
		return err
	}

	f, err := os.Create(s.paths.ProjectsFile)
	if err != nil {
		return fmt.Errorf("create projects config: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString("# mine project settings — generated by mine proj config\n\n"); err != nil {
		return err
	}
	return toml.NewEncoder(f).Encode(sf)
}

func (s *Store) getKV(key string) (string, error) {
	var value sql.NullString
	err := s.db.QueryRow(`SELECT value FROM kv WHERE key = ?`, key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read state %q: %w", key, err)
	}
	if !value.Valid {
		return "", nil
	}
	return value.String, nil
}

func (s *Store) setKV(key, value string) error {
	_, err := s.db.Exec(
		`INSERT INTO kv (key, value, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP`,
		key, value,
	)
	if err != nil {
		return fmt.Errorf("save state %q: %w", key, err)
	}
	return nil
}

func parseTime(raw string) time.Time {
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05.000Z",
	} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t
		}
	}
	return time.Time{}
}

func depthFromRel(rel string) int {
	if rel == "." || rel == "" {
		return 0
	}
	return strings.Count(rel, string(filepath.Separator)) + 1
}

func applySetting(cfg *Settings, key, value string) error {
	switch key {
	case "default_branch":
		cfg.DefaultBranch = value
	case "env_file":
		cfg.EnvFile = value
	case "tmux_layout":
		cfg.TmuxLayout = value
	case "ssh_host":
		cfg.SSHHost = value
	case "ssh_tunnel":
		cfg.SSHTunnel = value
	default:
		return fmt.Errorf("unknown key %q", key)
	}
	return nil
}

func settingValue(cfg Settings, key string) (string, error) {
	switch key {
	case "default_branch":
		return cfg.DefaultBranch, nil
	case "env_file":
		return cfg.EnvFile, nil
	case "tmux_layout":
		return cfg.TmuxLayout, nil
	case "ssh_host":
		return cfg.SSHHost, nil
	case "ssh_tunnel":
		return cfg.SSHTunnel, nil
	default:
		return "", fmt.Errorf("unknown key %q", key)
	}
}
