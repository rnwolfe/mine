package env

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rnwolfe/mine/internal/store"
)

func setupTestManager(t *testing.T, passphrase string) (*Manager, string, func()) {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, "data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, "config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmp, "cache"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmp, "state"))

	db, err := store.Open()
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	mgr := newWithBase(db.Conn(), filepath.Join(tmp, "envs"), passphrase)
	projectPath := filepath.Join(tmp, "project")
	return mgr, projectPath, func() { _ = db.Close() }
}

func TestSetVarEncryptsAndLoads(t *testing.T) {
	mgr, projectPath, done := setupTestManager(t, "secret-pass")
	defer done()

	if err := mgr.SetVar(projectPath, "local", "API_KEY", "super-secret"); err != nil {
		t.Fatalf("SetVar: %v", err)
	}

	got, err := mgr.LoadProfile(projectPath, "local")
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}
	if got["API_KEY"] != "super-secret" {
		t.Fatalf("API_KEY mismatch: got %q", got["API_KEY"])
	}

	raw, err := os.ReadFile(mgr.profilePath(projectPath, "local"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if strings.Contains(string(raw), "super-secret") {
		t.Fatalf("profile file contains plaintext secret")
	}
}

func TestActiveProfileSwitchAndCurrent(t *testing.T) {
	mgr, projectPath, done := setupTestManager(t, "secret-pass")
	defer done()

	active, err := mgr.ActiveProfile(projectPath)
	if err != nil {
		t.Fatalf("ActiveProfile: %v", err)
	}
	if active != "local" {
		t.Fatalf("expected default profile local, got %q", active)
	}

	if err := mgr.SetVar(projectPath, "staging", "PORT", "8080"); err != nil {
		t.Fatalf("SetVar: %v", err)
	}
	if err := mgr.SwitchProfile(projectPath, "staging"); err != nil {
		t.Fatalf("SwitchProfile: %v", err)
	}

	name, vars, err := mgr.CurrentProfile(projectPath)
	if err != nil {
		t.Fatalf("CurrentProfile: %v", err)
	}
	if name != "staging" {
		t.Fatalf("expected staging profile, got %q", name)
	}
	if vars["PORT"] != "8080" {
		t.Fatalf("expected PORT=8080, got %q", vars["PORT"])
	}
}

func TestDiffAndTemplate(t *testing.T) {
	mgr, projectPath, done := setupTestManager(t, "secret-pass")
	defer done()

	if err := mgr.SaveProfile(projectPath, "a", map[string]string{"A": "1", "B": "2"}); err != nil {
		t.Fatalf("SaveProfile a: %v", err)
	}
	if err := mgr.SaveProfile(projectPath, "b", map[string]string{"B": "9", "C": "3"}); err != nil {
		t.Fatalf("SaveProfile b: %v", err)
	}

	d, err := mgr.Diff(projectPath, "a", "b")
	if err != nil {
		t.Fatalf("Diff: %v", err)
	}
	if strings.Join(d.Added, ",") != "C" {
		t.Fatalf("added mismatch: %#v", d.Added)
	}
	if strings.Join(d.Removed, ",") != "A" {
		t.Fatalf("removed mismatch: %#v", d.Removed)
	}
	if strings.Join(d.Changed, ",") != "B" {
		t.Fatalf("changed mismatch: %#v", d.Changed)
	}

	keys, err := mgr.TemplateKeys(projectPath, "a")
	if err != nil {
		t.Fatalf("TemplateKeys: %v", err)
	}
	if strings.Join(keys, ",") != "A,B" {
		t.Fatalf("template keys mismatch: %#v", keys)
	}
}

func TestExportLines(t *testing.T) {
	mgr, projectPath, done := setupTestManager(t, "secret-pass")
	defer done()

	if err := mgr.SaveProfile(projectPath, "local", map[string]string{
		"API_KEY": "ab'cd",
		"EMPTY":   "",
	}); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	posix, err := mgr.ExportLines(projectPath, "local", "posix")
	if err != nil {
		t.Fatalf("ExportLines posix: %v", err)
	}
	if len(posix) != 2 {
		t.Fatalf("expected 2 posix lines, got %d", len(posix))
	}
	if !strings.Contains(strings.Join(posix, "\n"), "export API_KEY='ab'\"'\"'cd'") {
		t.Fatalf("posix escaping missing: %v", posix)
	}

	fish, err := mgr.ExportLines(projectPath, "local", "fish")
	if err != nil {
		t.Fatalf("ExportLines fish: %v", err)
	}
	if !strings.Contains(strings.Join(fish, "\n"), "set -gx API_KEY 'ab\\'cd'") {
		t.Fatalf("fish escaping missing: %v", fish)
	}
}

func TestWrongPassphrase(t *testing.T) {
	mgr, projectPath, done := setupTestManager(t, "secret-pass")
	defer done()

	if err := mgr.SetVar(projectPath, "local", "TOKEN", "abc"); err != nil {
		t.Fatalf("SetVar: %v", err)
	}

	other := newWithBase(mgr.db, mgr.baseDir, "wrong-pass")
	_, err := other.LoadProfile(projectPath, "local")
	if !errors.Is(err, ErrWrongPassphrase) {
		t.Fatalf("expected ErrWrongPassphrase, got %v", err)
	}
}

func TestValidateHelpers(t *testing.T) {
	if err := ValidateKey("BAD-KEY"); err == nil {
		t.Fatalf("expected invalid key error")
	}
	if err := ValidateProfileName("bad/name"); err == nil {
		t.Fatalf("expected invalid profile name error")
	}
	if got := MaskValue("abcdef"); got != "ab**ef" {
		t.Fatalf("unexpected mask result: %q", got)
	}
}

func TestCurrentProfileDefaultMissingReturnsEmpty(t *testing.T) {
	mgr, projectPath, done := setupTestManager(t, "secret-pass")
	defer done()

	name, vars, err := mgr.CurrentProfile(projectPath)
	if err != nil {
		t.Fatalf("CurrentProfile: %v", err)
	}
	if name != "local" {
		t.Fatalf("expected local, got %q", name)
	}
	if len(vars) != 0 {
		t.Fatalf("expected empty vars, got %#v", vars)
	}
}

func TestUnsetVarAndListProfiles(t *testing.T) {
	mgr, projectPath, done := setupTestManager(t, "secret-pass")
	defer done()

	if err := mgr.SaveProfile(projectPath, "local", map[string]string{"KEEP": "1", "DROP": "2"}); err != nil {
		t.Fatalf("SaveProfile local: %v", err)
	}
	if err := mgr.SaveProfile(projectPath, "staging", map[string]string{"A": "1"}); err != nil {
		t.Fatalf("SaveProfile staging: %v", err)
	}

	if err := mgr.UnsetVar(projectPath, "local", "DROP"); err != nil {
		t.Fatalf("UnsetVar: %v", err)
	}

	got, err := mgr.LoadProfile(projectPath, "local")
	if err != nil {
		t.Fatalf("LoadProfile local: %v", err)
	}
	if _, ok := got["DROP"]; ok {
		t.Fatalf("expected DROP to be removed, got %#v", got)
	}
	if got["KEEP"] != "1" {
		t.Fatalf("expected KEEP=1, got %#v", got)
	}

	profiles, err := mgr.ListProfiles(projectPath)
	if err != nil {
		t.Fatalf("ListProfiles: %v", err)
	}
	if strings.Join(profiles, ",") != "local,staging" {
		t.Fatalf("unexpected profiles: %#v", profiles)
	}
}

func TestSwitchProfileRequiresExistingProfile(t *testing.T) {
	mgr, projectPath, done := setupTestManager(t, "secret-pass")
	defer done()

	if err := mgr.SwitchProfile(projectPath, "missing"); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist, got %v", err)
	}
}

func TestCorruptedProfile(t *testing.T) {
	mgr, projectPath, done := setupTestManager(t, "secret-pass")
	defer done()

	if err := os.MkdirAll(filepath.Dir(mgr.profilePath(projectPath, "local")), 0o700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(mgr.profilePath(projectPath, "local"), []byte("not-age-data"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := mgr.LoadProfile(projectPath, "local")
	if !errors.Is(err, ErrCorruptedProfile) {
		t.Fatalf("expected ErrCorruptedProfile, got %v", err)
	}
}

func TestEditRoundTrip(t *testing.T) {
	mgr, projectPath, done := setupTestManager(t, "secret-pass")
	defer done()

	if err := mgr.SaveProfile(projectPath, "local", map[string]string{
		"API_KEY": "abc",
		"PORT":    "8080",
	}); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}

	vars, err := mgr.LoadProfile(projectPath, "local")
	if err != nil {
		t.Fatalf("LoadProfile: %v", err)
	}

	vars["PORT"] = "9000"
	vars["NEW_VAR"] = "hello"
	delete(vars, "API_KEY")

	if err := mgr.SaveProfile(projectPath, "local", vars); err != nil {
		t.Fatalf("SaveProfile after edit: %v", err)
	}

	got, err := mgr.LoadProfile(projectPath, "local")
	if err != nil {
		t.Fatalf("LoadProfile after edit: %v", err)
	}
	if got["PORT"] != "9000" {
		t.Errorf("expected PORT=9000, got %q", got["PORT"])
	}
	if got["NEW_VAR"] != "hello" {
		t.Errorf("expected NEW_VAR=hello, got %q", got["NEW_VAR"])
	}
	if _, ok := got["API_KEY"]; ok {
		t.Errorf("expected API_KEY to be deleted")
	}
}

func TestSaveProfileInvalidKeyRejection(t *testing.T) {
	mgr, projectPath, done := setupTestManager(t, "secret-pass")
	defer done()

	err := mgr.SaveProfile(projectPath, "local", map[string]string{
		"VALID_KEY":   "ok",
		"INVALID-KEY": "bad",
	})
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
	if !strings.Contains(err.Error(), "INVALID-KEY") {
		t.Errorf("error should mention the invalid key, got: %v", err)
	}

	_, loadErr := mgr.LoadProfile(projectPath, "local")
	if !errors.Is(loadErr, os.ErrNotExist) {
		t.Errorf("profile should not exist after failed save, got loadErr: %v", loadErr)
	}
}

func TestCorruptedProfileInvalidKey(t *testing.T) {
	mgr, projectPath, done := setupTestManager(t, "secret-pass")
	defer done()

	data := &profileData{
		Vars: map[string]string{"BAD-KEY": "value"},
	}
	enc, err := encrypt(data, "secret-pass")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if err := atomicWrite(mgr.profilePath(projectPath, "local"), enc); err != nil {
		t.Fatalf("atomicWrite: %v", err)
	}

	_, err = mgr.LoadProfile(projectPath, "local")
	if !errors.Is(err, ErrCorruptedProfile) {
		t.Fatalf("expected ErrCorruptedProfile, got %v", err)
	}
}
