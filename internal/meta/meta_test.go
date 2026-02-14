package meta

import (
	"os"
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

	if got := body; got == "" {
		t.Fatal("FormatFeatureRequest returned empty string")
	}
	assertContains(t, body, "## Description")
	assertContains(t, body, "Add export to CSV")
	assertContains(t, body, "## Use Case")
	assertContains(t, body, "share todos with my team")
	assertContains(t, body, "mine meta fr")
}

func TestFormatBugReport(t *testing.T) {
	info := SystemInfo{Version: "v0.2.0", OS: "linux", Arch: "amd64"}
	body := FormatBugReport("Run mine todo add", "Todo gets added", "Crash on empty title", info)

	assertContains(t, body, "## Steps to Reproduce")
	assertContains(t, body, "Run mine todo add")
	assertContains(t, body, "## Expected Behavior")
	assertContains(t, body, "Todo gets added")
	assertContains(t, body, "## Actual Behavior")
	assertContains(t, body, "Crash on empty title")
	assertContains(t, body, "v0.2.0")
	assertContains(t, body, "linux/amd64")
	assertContains(t, body, "mine meta bug")
}

func TestRedactPII(t *testing.T) {
	home, _ := os.UserHomeDir()

	tests := []struct {
		name     string
		input    string
		notWant  string
		wantHas  string
	}{
		{
			name:    "redacts home dir",
			input:   home + "/projects/mine",
			notWant: home,
			wantHas: "~/projects/mine",
		},
		{
			name:    "redacts API key pattern",
			input:   "my key is sk-abc123def456ghi789jklmnop",
			notWant: "sk-abc123def456ghi789jklmnop",
			wantHas: "[REDACTED]",
		},
		{
			name:    "redacts secret pattern",
			input:   "secret_abcdefghijklmnopqr",
			notWant: "secret_abcdefghijklmnopqr",
			wantHas: "[REDACTED]",
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
			got := RedactPII(tt.input)
			if tt.notWant != "" && containsStr(got, tt.notWant) {
				t.Errorf("RedactPII(%q) still contains %q", tt.input, tt.notWant)
			}
			if !containsStr(got, tt.wantHas) {
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

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !containsStr(s, substr) {
		t.Errorf("expected string to contain %q, got:\n%s", substr, s)
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (substr == "" || findSubstr(s, substr))
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
