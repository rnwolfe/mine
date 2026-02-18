package env

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"filippo.io/age"
	"filippo.io/age/armor"

	"github.com/rnwolfe/mine/internal/config"
)

const defaultProfile = "local"

var (
	// ErrWrongPassphrase is returned when decryption fails due to a bad passphrase.
	ErrWrongPassphrase = errors.New("wrong passphrase")
	// ErrCorruptedProfile is returned when encrypted profile data cannot be parsed.
	ErrCorruptedProfile = errors.New("env profile is corrupted or unreadable")
	keyPattern          = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	profilePattern      = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)
)

type profileData struct {
	Vars map[string]string `json:"vars"`
}

type Diff struct {
	Added   []string
	Removed []string
	Changed []string
}

// Manager handles encrypted profile storage and active-profile tracking.
type Manager struct {
	db         *sql.DB
	baseDir    string
	passphrase string
}

// New creates a manager using default XDG paths.
func New(db *sql.DB, passphrase string) *Manager {
	return &Manager{
		db:         db,
		baseDir:    config.GetPaths().EnvDir,
		passphrase: passphrase,
	}
}

func newWithBase(db *sql.DB, baseDir, passphrase string) *Manager {
	return &Manager{
		db:         db,
		baseDir:    baseDir,
		passphrase: passphrase,
	}
}

func ValidateKey(key string) error {
	if !keyPattern.MatchString(key) {
		return fmt.Errorf("invalid key %q (expected [A-Za-z_][A-Za-z0-9_]*)", key)
	}
	return nil
}

func ValidateProfileName(name string) error {
	if !profilePattern.MatchString(name) {
		return fmt.Errorf("invalid profile name %q", name)
	}
	return nil
}

func MaskValue(v string) string {
	if v == "" {
		return ""
	}
	if len(v) <= 4 {
		return strings.Repeat("*", len(v))
	}
	return v[:2] + strings.Repeat("*", len(v)-4) + v[len(v)-2:]
}

func (m *Manager) ActiveProfile(projectPath string) (string, error) {
	var profile string
	err := m.db.QueryRow(
		"SELECT active_profile FROM env_projects WHERE project_path = ?",
		projectPath,
	).Scan(&profile)
	if err == nil {
		return profile, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	return defaultProfile, nil
}

func (m *Manager) SwitchProfile(projectPath, name string) error {
	if err := ValidateProfileName(name); err != nil {
		return err
	}
	if _, err := m.LoadProfile(projectPath, name); err != nil {
		return err
	}
	_, err := m.db.Exec(
		`INSERT INTO env_projects(project_path, active_profile)
		 VALUES(?, ?)
		 ON CONFLICT(project_path) DO UPDATE SET active_profile=excluded.active_profile, updated_at=CURRENT_TIMESTAMP`,
		projectPath, name,
	)
	return err
}

func (m *Manager) CurrentProfile(projectPath string) (string, map[string]string, error) {
	name, err := m.ActiveProfile(projectPath)
	if err != nil {
		return "", nil, err
	}
	vars, err := m.LoadProfile(projectPath, name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && name == defaultProfile {
			return name, map[string]string{}, nil
		}
		return "", nil, err
	}
	return name, vars, nil
}

func (m *Manager) LoadProfile(projectPath, name string) (map[string]string, error) {
	if err := ValidateProfileName(name); err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(m.profilePath(projectPath, name))
	if err != nil {
		return nil, err
	}
	data, err := decrypt(raw, m.passphrase)
	if err != nil {
		return nil, err
	}
	for key := range data.Vars {
		if err := ValidateKey(key); err != nil {
			return nil, fmt.Errorf("%w: invalid key %q", ErrCorruptedProfile, key)
		}
	}
	return data.Vars, nil
}

func (m *Manager) SaveProfile(projectPath, name string, vars map[string]string) error {
	if err := ValidateProfileName(name); err != nil {
		return err
	}
	for k := range vars {
		if err := ValidateKey(k); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(m.projectDir(projectPath), 0o700); err != nil {
		return err
	}
	data := &profileData{Vars: map[string]string{}}
	for k, v := range vars {
		data.Vars[k] = v
	}
	enc, err := encrypt(data, m.passphrase)
	if err != nil {
		return err
	}
	return atomicWrite(m.profilePath(projectPath, name), enc)
}

func (m *Manager) SetVar(projectPath, profile, key, value string) error {
	if err := ValidateKey(key); err != nil {
		return err
	}
	vars, err := m.LoadProfile(projectPath, profile)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		vars = map[string]string{}
	}
	vars[key] = value
	return m.SaveProfile(projectPath, profile, vars)
}

func (m *Manager) UnsetVar(projectPath, profile, key string) error {
	if err := ValidateKey(key); err != nil {
		return err
	}
	vars, err := m.LoadProfile(projectPath, profile)
	if err != nil {
		return err
	}
	delete(vars, key)
	return m.SaveProfile(projectPath, profile, vars)
}

func (m *Manager) ListProfiles(projectPath string) ([]string, error) {
	entries, err := os.ReadDir(m.projectDir(projectPath))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []string{}, nil
		}
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".age") {
			continue
		}
		out = append(out, strings.TrimSuffix(e.Name(), ".age"))
	}
	sort.Strings(out)
	return out, nil
}

func (m *Manager) Diff(projectPath, a, b string) (Diff, error) {
	left, err := m.LoadProfile(projectPath, a)
	if err != nil {
		return Diff{}, err
	}
	right, err := m.LoadProfile(projectPath, b)
	if err != nil {
		return Diff{}, err
	}

	var d Diff
	for k, lv := range left {
		rv, ok := right[k]
		if !ok {
			d.Removed = append(d.Removed, k)
			continue
		}
		if lv != rv {
			d.Changed = append(d.Changed, k)
		}
	}
	for k := range right {
		if _, ok := left[k]; !ok {
			d.Added = append(d.Added, k)
		}
	}
	sort.Strings(d.Added)
	sort.Strings(d.Removed)
	sort.Strings(d.Changed)
	return d, nil
}

func (m *Manager) ExportLines(projectPath, profile, shellName string) ([]string, error) {
	vars, err := m.LoadProfile(projectPath, profile)
	if err != nil {
		return nil, err
	}
	keys := sortedKeys(vars)
	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		v := vars[k]
		if shellName == "fish" {
			lines = append(lines, fmt.Sprintf("set -gx %s %s", k, fishQuote(v)))
			continue
		}
		lines = append(lines, fmt.Sprintf("export %s=%s", k, shellQuote(v)))
	}
	return lines, nil
}

func (m *Manager) TemplateKeys(projectPath, profile string) ([]string, error) {
	vars, err := m.LoadProfile(projectPath, profile)
	if err != nil {
		return nil, err
	}
	return sortedKeys(vars), nil
}

func (m *Manager) ProjectPath() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Clean(wd), nil
}

func (m *Manager) profilePath(projectPath, profile string) string {
	return filepath.Join(m.projectDir(projectPath), profile+".age")
}

func (m *Manager) projectDir(projectPath string) string {
	sum := sha256.Sum256([]byte(projectPath))
	return filepath.Join(m.baseDir, hex.EncodeToString(sum[:]))
}

func sortedKeys(mv map[string]string) []string {
	keys := make([]string, 0, len(mv))
	for k := range mv {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func encrypt(data *profileData, passphrase string) ([]byte, error) {
	if strings.TrimSpace(passphrase) == "" {
		return nil, fmt.Errorf("env passphrase required")
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	recipient, err := age.NewScryptRecipient(passphrase)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	aw := armor.NewWriter(&buf)
	w, err := age.Encrypt(aw, recipient)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(payload); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	if err := aw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decrypt(raw []byte, passphrase string) (*profileData, error) {
	identity, err := age.NewScryptIdentity(passphrase)
	if err != nil {
		return nil, err
	}
	r, err := age.Decrypt(armor.NewReader(bytes.NewReader(raw)), identity)
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "no identity matched") || strings.Contains(msg, "incorrect") {
			return nil, fmt.Errorf("%w: %v", ErrWrongPassphrase, err)
		}
		return nil, fmt.Errorf("%w: %v", ErrCorruptedProfile, err)
	}
	plaintext, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCorruptedProfile, err)
	}
	var data profileData
	if err := json.Unmarshal(plaintext, &data); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCorruptedProfile, err)
	}
	if data.Vars == nil {
		data.Vars = map[string]string{}
	}
	return &data, nil
}

func atomicWrite(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".env-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	success := false
	defer func() {
		if !success {
			_ = os.Remove(tmpName)
		}
	}()
	if err := os.Chmod(tmpName, 0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	success = true
	return nil
}

func shellQuote(v string) string {
	return "'" + strings.ReplaceAll(v, "'", `'"'"'`) + "'"
}

func fishQuote(v string) string {
	if v == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(v, "'", `'\''`) + "'"
}
