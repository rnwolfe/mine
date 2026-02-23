package agents

import (
	"fmt"
	"strings"
	"time"
)

// LogEntry represents a single commit in the agents store history.
type LogEntry struct {
	Hash    string
	Short   string
	Date    time.Time
	Message string
}

// InitGitRepo initializes a git repo in the agents store directory if one doesn't
// already exist.
func InitGitRepo() error {
	dir := Dir()

	if IsGitRepo() {
		return nil
	}

	if _, err := gitCmd(dir, "init"); err != nil {
		return fmt.Errorf("git init: %w", err)
	}

	// Configure committer identity for the agents repo.
	if _, err := gitCmd(dir, "config", "user.name", "mine-agents"); err != nil {
		return fmt.Errorf("git config user.name: %w", err)
	}
	if _, err := gitCmd(dir, "config", "user.email", "agents@mine.local"); err != nil {
		return fmt.Errorf("git config user.email: %w", err)
	}

	return nil
}

// Commit snapshots the current state of the agents store with a message.
// Initializes the git repo if needed.
func Commit(message string) (string, error) {
	dir := Dir()

	if err := InitGitRepo(); err != nil {
		return "", err
	}

	// Stage everything.
	if _, err := gitCmd(dir, "add", "-A"); err != nil {
		return "", fmt.Errorf("git add: %w", err)
	}

	// Check if there's anything to commit.
	status, err := gitCmd(dir, "status", "--porcelain")
	if err != nil {
		return "", fmt.Errorf("git status: %w", err)
	}
	if strings.TrimSpace(status) == "" {
		return "", fmt.Errorf("nothing to commit — all files up to date")
	}

	if _, err := gitCmd(dir, "commit", "-m", message); err != nil {
		return "", fmt.Errorf("git commit: %w", err)
	}

	// Return the short hash.
	hash, err := gitCmd(dir, "rev-parse", "--short", "HEAD")
	if err != nil {
		return "", fmt.Errorf("getting commit hash: %w", err)
	}
	return strings.TrimSpace(hash), nil
}

// HasCommits returns true if the agents store git repo has at least one commit.
func HasCommits() bool {
	if !IsGitRepo() {
		return false
	}
	_, err := gitCmd(Dir(), "rev-parse", "--verify", "HEAD")
	return err == nil
}

// Log returns the commit history for the agents store, optionally filtered
// to a specific file.
func Log(file string) ([]LogEntry, error) {
	dir := Dir()

	if !IsGitRepo() {
		return nil, fmt.Errorf("no version history yet — run `mine agents init` first")
	}

	args := []string{"log", "--format=%H|%h|%aI|%s"}
	if file != "" {
		args = append(args, "--", file)
	}

	out, err := gitCmd(dir, args...)
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}

	if strings.TrimSpace(out) == "" {
		return nil, nil
	}

	var entries []LogEntry
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		parts := strings.SplitN(line, "|", 4)
		if len(parts) != 4 {
			continue
		}
		t, _ := time.Parse(time.RFC3339, parts[2])
		entries = append(entries, LogEntry{
			Hash:    parts[0],
			Short:   parts[1],
			Date:    t,
			Message: parts[3],
		})
	}
	return entries, nil
}
