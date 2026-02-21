package cmd

import (
	"os"
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
	t.Setenv("MINE_ENV_PASSPHRASE", "test")

	err := runEnvEdit(nil, nil)
	if err == nil {
		t.Fatal("expected error when $EDITOR is not set")
	}
	if !strings.Contains(err.Error(), "$EDITOR") {
		t.Fatalf("expected $EDITOR mention in error, got: %v", err)
	}
}

func TestParseEnvFileValid(t *testing.T) {
	content := "API_KEY=secret\nDB_URL=postgres://localhost/db\n"
	vars, err := parseEnvFile(content)
	if err != nil {
		t.Fatalf("parseEnvFile: %v", err)
	}
	if vars["API_KEY"] != "secret" {
		t.Fatalf("API_KEY: got %q", vars["API_KEY"])
	}
	if vars["DB_URL"] != "postgres://localhost/db" {
		t.Fatalf("DB_URL: got %q", vars["DB_URL"])
	}
}

func TestParseEnvFileInvalidKey(t *testing.T) {
	content := "VALID_KEY=ok\n123INVALID=bad\n"
	_, err := parseEnvFile(content)
	if err == nil {
		t.Fatal("expected error for invalid key")
	}
	if !strings.Contains(err.Error(), "123INVALID") {
		t.Fatalf("expected invalid key name in error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "changes discarded") {
		t.Fatalf("expected 'changes discarded' in error, got: %v", err)
	}
}

func TestParseEnvFileIgnoresBlankAndComments(t *testing.T) {
	content := "# comment line\n\nAPI_KEY=value\n   \n#another comment\n"
	vars, err := parseEnvFile(content)
	if err != nil {
		t.Fatalf("parseEnvFile: %v", err)
	}
	if len(vars) != 1 {
		t.Fatalf("expected 1 var, got %d: %v", len(vars), vars)
	}
	if vars["API_KEY"] != "value" {
		t.Fatalf("API_KEY: got %q", vars["API_KEY"])
	}
}

func TestParseEnvFileValueWithEquals(t *testing.T) {
	content := "TOKEN=abc=def=ghi\n"
	vars, err := parseEnvFile(content)
	if err != nil {
		t.Fatalf("parseEnvFile: %v", err)
	}
	if vars["TOKEN"] != "abc=def=ghi" {
		t.Fatalf("TOKEN: got %q", vars["TOKEN"])
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
