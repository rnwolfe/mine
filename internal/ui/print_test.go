package ui

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

// captureStdout redirects os.Stdout to a pipe, runs f, then returns what was written.
func captureStdout(f func()) string {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	os.Stdout = w
	f()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r) //nolint:errcheck
	return buf.String()
}

// captureStderr redirects os.Stderr to a pipe, runs f, then returns what was written.
func captureStderr(f func()) string {
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	os.Stderr = w
	f()
	w.Close()
	os.Stderr = old
	var buf bytes.Buffer
	io.Copy(&buf, r) //nolint:errcheck
	return buf.String()
}

func TestPuts(t *testing.T) {
	got := captureStdout(func() { Puts("hello world") })
	if !strings.Contains(got, "hello world") {
		t.Errorf("Puts: output %q does not contain %q", got, "hello world")
	}
}

func TestPutsf(t *testing.T) {
	got := captureStdout(func() { Putsf("count: %d", 42) })
	if !strings.Contains(got, "count: 42") {
		t.Errorf("Putsf: output %q does not contain %q", got, "count: 42")
	}
}

func TestWarn(t *testing.T) {
	got := captureStdout(func() { Warn("disk is full") })
	if !strings.Contains(got, "disk is full") {
		t.Errorf("Warn: output %q does not contain %q", got, "disk is full")
	}
}

func TestErr(t *testing.T) {
	got := captureStderr(func() { Err("something failed") })
	if !strings.Contains(got, "something failed") {
		t.Errorf("Err: stderr %q does not contain %q", got, "something failed")
	}
}

func TestOk(t *testing.T) {
	got := captureStdout(func() { Ok("all done") })
	if !strings.Contains(got, "all done") {
		t.Errorf("Ok: output %q does not contain %q", got, "all done")
	}
}

func TestInf(t *testing.T) {
	got := captureStdout(func() { Inf("just so you know") })
	if !strings.Contains(got, "just so you know") {
		t.Errorf("Inf: output %q does not contain %q", got, "just so you know")
	}
}

func TestHeader(t *testing.T) {
	got := captureStdout(func() { Header("My Section") })
	if !strings.Contains(got, "My Section") {
		t.Errorf("Header: output %q does not contain %q", got, "My Section")
	}
	// Header prints a separator line of dashes; verify at least one is present.
	if !strings.Contains(got, "─") {
		t.Errorf("Header: output %q should contain a separator line", got)
	}
}

func TestTip(t *testing.T) {
	got := captureStdout(func() { Tip("run mine --help for options") })
	if !strings.Contains(got, "run mine --help for options") {
		t.Errorf("Tip: output %q does not contain %q", got, "run mine --help for options")
	}
}

func TestKv(t *testing.T) {
	got := captureStdout(func() { Kv("status", "active") })
	if !strings.Contains(got, "status") {
		t.Errorf("Kv: output %q does not contain key %q", got, "status")
	}
	if !strings.Contains(got, "active") {
		t.Errorf("Kv: output %q does not contain value %q", got, "active")
	}
}

func TestGreet(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"", "▸ Hey there!"},
		{"Ryan", "▸ Hey Ryan!"},
		{"World", "▸ Hey World!"},
	}

	for _, tt := range tests {
		got := Greet(tt.name)
		if got != tt.expected {
			t.Errorf("Greet(%q) = %q, want %q", tt.name, got, tt.expected)
		}
	}
}

func TestIconConstants(t *testing.T) {
	// Verify icons are non-empty strings
	icons := []string{
		IconMine, IconGem, IconGold, IconTodo, IconDone, IconOverdue,
		IconTools, IconPackage, IconVault, IconGrow, IconStar, IconFire,
		IconWarn, IconError, IconOk, IconArrow, IconDot, IconDig,
	}
	for i, icon := range icons {
		if icon == "" {
			t.Errorf("Icon at index %d is empty", i)
		}
	}
}
