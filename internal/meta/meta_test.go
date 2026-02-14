package meta

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestValidateTitle(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		wantErr bool
	}{
		{"valid title", "Add dark mode support", false},
		{"minimum length", "hello", false},
		{"empty", "", true},
		{"whitespace only", "   ", true},
		{"too short", "hi", true},
		{"four chars", "abcd", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTitle(tt.title)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTitle(%q) error = %v, wantErr %v", tt.title, err, tt.wantErr)
			}
		})
	}
}

func TestValidateRequired(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		field   string
		wantErr bool
	}{
		{"non-empty", "something", "description", false},
		{"empty", "", "description", true},
		{"whitespace", "  \t ", "description", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRequired(tt.value, tt.field)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRequired(%q, %q) error = %v, wantErr %v", tt.value, tt.field, err, tt.wantErr)
			}
		})
	}
}

func TestFormatFeatureRequest(t *testing.T) {
	body := FormatFeatureRequest("Add export to CSV", "I need to share todos with my team")

	if body == "" {
		t.Fatal("FormatFeatureRequest returned empty string")
	}
	for _, want := range []string{"## Description", "Add export to CSV", "## Use Case", "share todos with my team", "mine meta fr"} {
		if !strings.Contains(body, want) {
			t.Errorf("expected body to contain %q, got:\n%s", want, body)
		}
	}
}

func TestFormatBugReport(t *testing.T) {
	info := SystemInfo{Version: "v0.2.0", OS: "linux", Arch: "amd64"}
	body := FormatBugReport("Run mine todo add", "Todo gets added", "Crash on empty title", info)

	for _, want := range []string{
		"## Steps to Reproduce", "Run mine todo add",
		"## Expected Behavior", "Todo gets added",
		"## Actual Behavior", "Crash on empty title",
		"v0.2.0", "linux/amd64", "mine meta bug",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("expected body to contain %q, got:\n%s", want, body)
		}
	}
}

func TestRedactPII(t *testing.T) {
	home, err := os.UserHomeDir()

	tests := []struct {
		name    string
		input   string
		notWant string
		wantHas string
		skip    bool
	}{
		{
			name:    "redacts home dir",
			input:   home + "/projects/mine",
			notWant: home,
			wantHas: "~/projects/mine",
			skip:    err != nil || home == "",
		},
		{
			name:    "redacts Stripe-style key",
			input:   "my key is sk-abc123def456ghi789jklmnop",
			notWant: "sk-abc123def456ghi789jklmnop",
			wantHas: "[REDACTED]",
		},
		{
			name:    "redacts GitHub token",
			input:   "token ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij",
			notWant: "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghij",
			wantHas: "[REDACTED]",
		},
		{
			name:    "does not redact generic identifiers",
			input:   "secret_configuration_value is fine",
			notWant: "",
			wantHas: "secret_configuration_value",
		},
		{
			name:    "leaves clean text alone",
			input:   "just a normal description",
			notWant: "",
			wantHas: "just a normal description",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skip {
				t.Skip("home directory not available in this environment")
			}
			got := RedactPII(tt.input)
			if tt.notWant != "" && strings.Contains(got, tt.notWant) {
				t.Errorf("RedactPII(%q) still contains %q", tt.input, tt.notWant)
			}
			if !strings.Contains(got, tt.wantHas) {
				t.Errorf("RedactPII(%q) = %q, want it to contain %q", tt.input, got, tt.wantHas)
			}
		})
	}
}

func TestCollectSystemInfo(t *testing.T) {
	info := CollectSystemInfo()
	if info.OS == "" {
		t.Error("OS should not be empty")
	}
	if info.Arch == "" {
		t.Error("Arch should not be empty")
	}
	if info.Version == "" {
		t.Error("Version should not be empty")
	}
}

func TestIssueArgs(t *testing.T) {
	args := IssueArgs("My Title", "Body text", "bug")

	expected := []string{
		"issue", "create",
		"--repo", "rnwolfe/mine",
		"--title", "My Title",
		"--body", "Body text",
		"--label", "bug",
	}
	if len(args) != len(expected) {
		t.Fatalf("IssueArgs returned %d args, want %d", len(args), len(expected))
	}
	for i, want := range expected {
		if args[i] != want {
			t.Errorf("IssueArgs[%d] = %q, want %q", i, args[i], want)
		}
	}
}

func TestCheckGH_NotInstalled(t *testing.T) {
	origLookPath := lookPath
	defer func() { lookPath = origLookPath }()

	lookPath = func(file string) (string, error) {
		return "", exec.ErrNotFound
	}

	err := CheckGH()
	if err == nil {
		t.Fatal("expected error when gh not found")
	}
	if !strings.Contains(err.Error(), "gh CLI not found") {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestCheckGH_NotAuthenticated(t *testing.T) {
	origLookPath := lookPath
	origExecCommand := execCommand
	defer func() {
		lookPath = origLookPath
		execCommand = origExecCommand
	}()

	lookPath = func(file string) (string, error) {
		return "/usr/bin/gh", nil
	}
	// Use Go's own test binary with a nonexistent flag to guarantee a
	// cross-platform non-zero exit without relying on platform-specific
	// commands like "false".
	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command(os.Args[0], "-test.run=^$", "-test.nonexistent-flag")
	}

	err := CheckGH()
	if err == nil {
		t.Fatal("expected error when gh not authenticated")
	}
	if !strings.Contains(err.Error(), "not authenticated") {
		t.Errorf("unexpected error message: %v", err)
	}
}
