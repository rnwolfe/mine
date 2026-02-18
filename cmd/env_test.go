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
	if err == nil || !strings.Contains(err.Error(), "value required") {
		t.Fatalf("expected value required error, got %v", err)
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
