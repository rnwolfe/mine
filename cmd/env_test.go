package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadEnvPassphrasePrefersEnvVar(t *testing.T) {
	t.Setenv("MINE_ENV_PASSPHRASE", "from-env")
	t.Setenv("MINE_VAULT_PASSPHRASE", "from-vault")

	got, err := readEnvPassphrase()
	if err != nil {
		t.Fatalf("readEnvPassphrase: %v", err)
	}
	if got != "from-env" {
		t.Fatalf("expected from-env, got %q", got)
	}
}

func TestReadEnvPassphraseFallsBackToVault(t *testing.T) {
	t.Setenv("MINE_ENV_PASSPHRASE", "")
	t.Setenv("MINE_VAULT_PASSPHRASE", "from-vault")

	got, err := readEnvPassphrase()
	if err != nil {
		t.Fatalf("readEnvPassphrase: %v", err)
	}
	if got != "from-vault" {
		t.Fatalf("expected from-vault, got %q", got)
	}
}

func TestParseSetArgInline(t *testing.T) {
	key, value, err := parseSetArg("API_KEY=secret")
	if err != nil {
		t.Fatalf("parseSetArg: %v", err)
	}
	if key != "API_KEY" || value != "secret" {
		t.Fatalf("unexpected parse result: %q=%q", key, value)
	}
}

func TestParseSetArgFromStdin(t *testing.T) {
	original := os.Stdin
	defer func() { os.Stdin = original }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	if _, err := w.WriteString("super-secret\n"); err != nil {
		t.Fatalf("write stdin: %v", err)
	}
	_ = w.Close()
	os.Stdin = r

	key, value, err := parseSetArg("TOKEN")
	if err != nil {
		t.Fatalf("parseSetArg: %v", err)
	}
	if key != "TOKEN" || value != "super-secret" {
		t.Fatalf("unexpected parse result: %q=%q", key, value)
	}
}

func TestParseSetArgRequiresValue(t *testing.T) {
	original := os.Stdin
	defer func() { os.Stdin = original }()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	_ = w.Close()
	os.Stdin = r

	_, _, err = parseSetArg("TOKEN")
	if err == nil || !strings.Contains(err.Error(), "value") {
		t.Fatalf("expected value required error, got %v", err)
	}
}

func TestRunEnvEditMissingEditor(t *testing.T) {
	t.Setenv("EDITOR", "")

	err := runEnvEdit(nil, nil)
	if err == nil {
		t.Fatal("expected error when $EDITOR is not set")
	}
	if !strings.Contains(err.Error(), "EDITOR") {
		t.Errorf("error should mention EDITOR, got: %v", err)
	}
	if !strings.Contains(err.Error(), "mine env set") {
		t.Errorf("error should mention fallback command, got: %v", err)
	}
}

func TestParseEnvFile(t *testing.T) {
	content := "# comment\nAPI_KEY=secret\nPORT=8080\nEMPTY=\nMULTI=a=b=c\n"
	vars, invalidKeys := parseEnvFile(content)
	if len(invalidKeys) != 0 {
		t.Fatalf("unexpected invalid keys: %v", invalidKeys)
	}
	if vars["API_KEY"] != "secret" {
		t.Errorf("API_KEY: got %q", vars["API_KEY"])
	}
	if vars["PORT"] != "8080" {
		t.Errorf("PORT: got %q", vars["PORT"])
	}
	if vars["EMPTY"] != "" {
		t.Errorf("EMPTY: got %q", vars["EMPTY"])
	}
	if vars["MULTI"] != "a=b=c" {
		t.Errorf("MULTI: got %q", vars["MULTI"])
	}
}

func TestParseEnvFileInvalidKeys(t *testing.T) {
	content := "VALID=ok\nINVALID-KEY=bad\n123_STARTS_NUM=bad\n"
	vars, invalidKeys := parseEnvFile(content)
	if _, ok := vars["VALID"]; !ok {
		t.Error("VALID key should be in result")
	}
	if len(invalidKeys) != 2 {
		t.Fatalf("expected 2 invalid keys, got %v", invalidKeys)
	}
}

func TestRunEnvEditEditorNonzeroExit(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmpDir, "data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmpDir, "cache"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmpDir, "state"))
	t.Setenv("MINE_ENV_PASSPHRASE", "test-pass")
	t.Chdir(tmpDir)

	// Create a fake editor that exits non-zero.
	editorScript := filepath.Join(tmpDir, "bad-editor.sh")
	if err := os.WriteFile(editorScript, []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatalf("WriteFile editor script: %v", err)
	}
	t.Setenv("EDITOR", editorScript)

	err := runEnvEdit(nil, nil)
	if err == nil {
		t.Fatal("expected error from editor non-zero exit")
	}
	if !strings.Contains(err.Error(), "no changes saved") {
		t.Errorf("error should mention no changes saved, got: %v", err)
	}
}

func TestRunEnvEditInvalidKeyInFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmpDir, "data"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpDir, "config"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(tmpDir, "cache"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmpDir, "state"))
	t.Setenv("MINE_ENV_PASSPHRASE", "test-pass")
	t.Chdir(tmpDir)

	// Create a fake editor that writes an invalid key.
	editorScript := filepath.Join(tmpDir, "invalid-editor.sh")
	script := "#!/bin/sh\nprintf 'INVALID-KEY=value\\n' > \"$1\"\n"
	if err := os.WriteFile(editorScript, []byte(script), 0o755); err != nil {
		t.Fatalf("WriteFile editor script: %v", err)
	}
	t.Setenv("EDITOR", editorScript)

	err := runEnvEdit(nil, nil)
	if err == nil {
		t.Fatal("expected error for invalid key in edited file")
	}
	if !strings.Contains(err.Error(), "INVALID-KEY") {
		t.Errorf("error should name the invalid key, got: %v", err)
	}
	if !strings.Contains(err.Error(), "No changes saved") {
		t.Errorf("error should mention no changes saved, got: %v", err)
	}
}

func TestMergedEnv(t *testing.T) {
	base := []string{
		"A=old",
		"B=2",
		"A=stale",
		"NOSEP",
	}
	overrides := map[string]string{
		"A": "new",
		"C": "3",
	}

	lines := mergedEnv(base, overrides)
	if len(lines) != 3 {
		t.Fatalf("expected 3 entries, got %d (%v)", len(lines), lines)
	}
	joined := strings.Join(lines, ",")
	if strings.Contains(joined, "A=old") || strings.Contains(joined, "A=stale") {
		t.Fatalf("unexpected stale A value in merged env: %v", lines)
	}
	if !strings.Contains(joined, "A=new") || !strings.Contains(joined, "B=2") || !strings.Contains(joined, "C=3") {
		t.Fatalf("unexpected env entries: %v", lines)
	}
}
