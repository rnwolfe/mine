package meta

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"

	"github.com/rnwolfe/mine/internal/version"
)

// SystemInfo holds auto-detected environment info for bug reports.
type SystemInfo struct {
	Version string
	OS      string
	Arch    string
}

// CollectSystemInfo gathers the current runtime environment.
func CollectSystemInfo() SystemInfo {
	return SystemInfo{
		Version: version.Short(),
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
	}
}

// ValidateTitle checks that an issue title meets minimum requirements.
func ValidateTitle(title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return errors.New("title cannot be empty")
	}
	if len(title) < 5 {
		return errors.New("title must be at least 5 characters")
	}
	return nil
}

// ValidateRequired checks that a field is non-empty.
func ValidateRequired(value, name string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", name)
	}
	return nil
}

// FormatFeatureRequest builds the markdown body for a feature request issue.
func FormatFeatureRequest(description, useCase string) string {
	var b strings.Builder
	b.WriteString("## Description\n\n")
	b.WriteString(description)
	b.WriteString("\n\n## Use Case\n\n")
	b.WriteString(useCase)
	b.WriteString("\n\n---\n*Submitted via `mine meta fr`*\n")
	return b.String()
}

// FormatBugReport builds the markdown body for a bug report issue.
func FormatBugReport(steps, expected, actual string, info SystemInfo) string {
	var b strings.Builder
	b.WriteString("## Steps to Reproduce\n\n")
	b.WriteString(steps)
	b.WriteString("\n\n## Expected Behavior\n\n")
	b.WriteString(expected)
	b.WriteString("\n\n## Actual Behavior\n\n")
	b.WriteString(actual)
	b.WriteString("\n\n## System Info\n\n")
	b.WriteString(fmt.Sprintf("- **mine version:** %s\n", info.Version))
	b.WriteString(fmt.Sprintf("- **OS:** %s/%s\n", info.OS, info.Arch))
	b.WriteString("\n---\n*Submitted via `mine meta bug`*\n")
	return b.String()
}

// apiKeyPattern matches well-known API key formats to avoid false positives
// from generic identifiers like "secret_configuration_value".
var apiKeyPattern = regexp.MustCompile(`(?i)\b(?:` +
	`sk-[a-zA-Z0-9]{20,}|` + // Stripe / OpenAI keys
	`gh[pousr]_[a-zA-Z0-9]{36,}|` + // GitHub tokens
	`xox[abprs]-[a-zA-Z0-9-]{10,48}|` + // Slack tokens
	`ya29\.[0-9A-Za-z_-]{20,}|` + // Google OAuth tokens
	`AIza[0-9A-Za-z_-]{20,}` + // Google API keys
	`)\b`)

// RedactPII strips sensitive data from text.
func RedactPII(s string) string {
	// Strip home directory paths.
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		s = strings.ReplaceAll(s, home, "~")
	}
	// Mask API key patterns.
	s = apiKeyPattern.ReplaceAllString(s, "[REDACTED]")
	return s
}

// execCommand is a variable so tests can replace exec.Command with a stub.
var execCommand = exec.Command

// lookPath is a variable so tests can replace exec.LookPath with a stub.
var lookPath = exec.LookPath

// CheckGH verifies that the gh CLI is installed and authenticated.
func CheckGH() error {
	if _, err := lookPath("gh"); err != nil {
		return errors.New("gh CLI not found — install it from https://cli.github.com")
	}
	cmd := execCommand("gh", "auth", "status")
	if err := cmd.Run(); err != nil {
		return errors.New("gh is not authenticated — run `gh auth login` first")
	}
	return nil
}

// IssueArgs returns the gh CLI arguments for creating an issue.
func IssueArgs(title, body, label string) []string {
	return []string{
		"issue", "create",
		"--repo", "rnwolfe/mine",
		"--title", title,
		"--body", body,
		"--label", label,
	}
}

// CreateIssue submits a GitHub issue via gh CLI and returns the issue URL.
func CreateIssue(title, body, label string) (string, error) {
	args := IssueArgs(title, body, label)
	cmd := execCommand("gh", args...)
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("gh issue create failed: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", fmt.Errorf("gh issue create failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
