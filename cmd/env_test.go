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

func TestMapToEnv(t *testing.T) {
	values := map[string]string{
		"B": "2",
		"A": "1",
	}
	lines := mapToEnv(values)
	if len(lines) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(lines))
	}
	joined := strings.Join(lines, ",")
	if !strings.Contains(joined, "A=1") || !strings.Contains(joined, "B=2") {
		t.Fatalf("unexpected env entries: %v", lines)
	}
}
