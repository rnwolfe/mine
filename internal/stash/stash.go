package stash

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/rnwolfe/mine/internal/config"
)

// Entry represents a tracked file in the stash.
type Entry struct {
	Source   string // Absolute source path
	SafeName string // Name in stash directory
}

// LogEntry represents a single commit in the stash history.
type LogEntry struct {
	Hash    string
	Short   string
	Date    time.Time
	Message string
}

// Dir returns the stash directory path.
func Dir() string {
	return filepath.Join(config.GetPaths().DataDir, "stash")
}

// ManifestPath returns the path to the manifest file.
func ManifestPath() string {
	return filepath.Join(Dir(), ".mine-stash")
}

// IsGitRepo returns true if the stash directory is a git repository.
func IsGitRepo() bool {
	_, err := os.Stat(filepath.Join(Dir(), ".git"))
	return err == nil
}

// InitGitRepo initializes a git repo in the stash directory if one doesn't exist.
func InitGitRepo() error {
	dir := Dir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating stash directory: %w", err)
	}

	if IsGitRepo() {
		return nil
	}

	if _, err := gitCmd(dir, "init"); err != nil {
		return fmt.Errorf("git init: %w", err)
	}

	// Configure committer identity for the stash repo.
	if _, err := gitCmd(dir, "config", "user.name", "mine-stash"); err != nil {
		return fmt.Errorf("git config user.name: %w", err)
	}
	if _, err := gitCmd(dir, "config", "user.email", "stash@mine.local"); err != nil {
		return fmt.Errorf("git config user.email: %w", err)
	}

	return nil
}

// ReadManifest parses the manifest file and returns tracked entries.
func ReadManifest() ([]Entry, error) {
	data, err := os.ReadFile(ManifestPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	entries := []Entry{} // non-nil: distinguishes "file exists, no entries" from "file missing" (nil)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, " -> ", 2)
		if len(parts) != 2 {
			continue
		}
		entries = append(entries, Entry{
			Source:   parts[0],
			SafeName: parts[1],
		})
	}
	return entries, scanner.Err()
}

// FindEntry looks up a manifest entry by source path or safe name.
func FindEntry(name string) (*Entry, error) {
	entries, err := ReadManifest()
	if err != nil {
		return nil, err
	}

	home, _ := os.UserHomeDir()

	// First, look for exact / explicit matches.
	for _, e := range entries {
		// Match against safe name, source path, or ~-relative path.
		display := strings.Replace(e.Source, home+"/", "~/", 1)
		if e.SafeName == name || e.Source == name || display == name {
			return &e, nil
		}
	}

	// If no exact match, consider basename matches but require uniqueness.
	var candidates []Entry
	for _, e := range entries {
		if filepath.Base(e.Source) == name || filepath.Base(e.SafeName) == name {
			candidates = append(candidates, e)
		}
	}

	switch len(candidates) {
	case 0:
		return nil, fmt.Errorf("no tracked file matching %q", name)
	case 1:
		return &candidates[0], nil
	default:
		return nil, fmt.Errorf("multiple tracked files share the name %q; please use a full or ~-relative path or safe name", name)
	}
}

// validateSafeName checks that a SafeName is safe for use as a filename in the stash directory.
func validateSafeName(safeName string) error {
	if safeName == "" {
		return fmt.Errorf("empty SafeName")
	}
	if strings.Contains(safeName, "..") || strings.ContainsAny(safeName, "/\\:") {
		return fmt.Errorf("unsafe SafeName %q", safeName)
	}
	return nil
}

// Commit snapshots the current stash state with a message.
// Initializes the git repo on first commit.
func Commit(message string) (string, error) {
	dir := Dir()

	if err := InitGitRepo(); err != nil {
		return "", err
	}

	// Refresh: re-copy all tracked files into stash before committing.
	entries, err := ReadManifest()
	if err != nil {
		return "", fmt.Errorf("reading manifest: %w", err)
	}
	for _, e := range entries {
		if err := validateSafeName(e.SafeName); err != nil {
			return "", fmt.Errorf("invalid manifest entry for %s: %w", e.Source, err)
		}
		src := e.Source
		dst := filepath.Join(dir, e.SafeName)
		data, err := os.ReadFile(src)
		if err != nil {
			if os.IsNotExist(err) {
				// Source was deleted: remove stash copy so git will record the deletion.
				if remErr := os.Remove(dst); remErr != nil && !os.IsNotExist(remErr) {
					return "", fmt.Errorf("removing deleted file %s from stash: %w", e.SafeName, remErr)
				}
				continue
			}
			// Other read errors should abort the commit.
			return "", fmt.Errorf("reading source %s: %w", src, err)
		}
		// Preserve file mode: prefer existing stash file mode, then source file mode, then default.
		mode := os.FileMode(0o644)
		if info, err := os.Stat(dst); err == nil {
			mode = info.Mode()
		} else if info, err := os.Stat(src); err == nil {
			mode = info.Mode()
		}
		if err := os.WriteFile(dst, data, mode); err != nil {
			return "", fmt.Errorf("copying %s: %w", e.SafeName, err)
		}
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

	// Commit.
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

// Log returns the commit history, optionally filtered to a specific file.
func Log(file string) ([]LogEntry, error) {
	dir := Dir()

	if !IsGitRepo() {
		return nil, fmt.Errorf("no version history yet — run `mine stash commit` first")
	}

	args := []string{"log", "--format=%H|%h|%aI|%s"}
	if file != "" {
		entry, err := FindEntry(file)
		if err != nil {
			return nil, err
		}
		args = append(args, "--", entry.SafeName)
	}

	out, err := gitCmd(dir, args...)
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}

	if strings.TrimSpace(out) == "" {
		return nil, nil
	}

	var entries []LogEntry
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
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
	return entries, scanner.Err()
}

// Restore restores a tracked file to a previous version.
// If version is empty, restores from the latest commit.
func Restore(file string, version string) ([]byte, error) {
	dir := Dir()

	if !IsGitRepo() {
		return nil, fmt.Errorf("no version history yet — run `mine stash commit` first")
	}

	entry, err := FindEntry(file)
	if err != nil {
		return nil, err
	}

	if version == "" {
		version = "HEAD"
	}

	// Get the file content at the specified version.
	content, err := gitCmd(dir, "show", version+":"+entry.SafeName)
	if err != nil {
		return nil, fmt.Errorf("version %s not found for %s", version, file)
	}

	return []byte(content), nil
}

// RestoreToSource restores a file to its original source location.
func RestoreToSource(file string, version string) error {
	entry, err := FindEntry(file)
	if err != nil {
		return err
	}

	content, err := Restore(file, version)
	if err != nil {
		return err
	}

	// Determine permissions for the source file: preserve existing mode when present,
	// otherwise use a restrictive default.
	srcPerm := os.FileMode(0o600)
	if info, err := os.Stat(entry.Source); err == nil {
		srcPerm = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat source %s: %w", entry.Source, err)
	}

	if err := os.WriteFile(entry.Source, content, srcPerm); err != nil {
		return fmt.Errorf("writing to %s: %w", entry.Source, err)
	}

	// Also update the stash copy.
	stashPath := filepath.Join(Dir(), entry.SafeName)

	stashPerm := os.FileMode(0o600)
	if info, err := os.Stat(stashPath); err == nil {
		stashPerm = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat stash copy %s: %w", stashPath, err)
	}

	if err := os.WriteFile(stashPath, content, stashPerm); err != nil {
		return fmt.Errorf("updating stash copy: %w", err)
	}

	return nil
}

// SyncSetRemote configures the git remote for the stash repo.
func SyncSetRemote(url string) error {
	dir := Dir()
	if !IsGitRepo() {
		return fmt.Errorf("no version history yet — run `mine stash commit` first")
	}

	// Check if remote already exists.
	existing, _ := gitCmd(dir, "remote", "get-url", "origin")
	if strings.TrimSpace(existing) != "" {
		if _, err := gitCmd(dir, "remote", "set-url", "origin", url); err != nil {
			return fmt.Errorf("updating remote: %w", err)
		}
	} else {
		if _, err := gitCmd(dir, "remote", "add", "origin", url); err != nil {
			return fmt.Errorf("adding remote: %w", err)
		}
	}
	return nil
}

// SyncPush pushes the stash repo to the configured remote.
func SyncPush() error {
	dir := Dir()
	if !IsGitRepo() {
		return fmt.Errorf("no version history yet — run `mine stash commit` first")
	}

	// Check remote exists.
	remote, _ := gitCmd(dir, "remote", "get-url", "origin")
	if strings.TrimSpace(remote) == "" {
		return fmt.Errorf("no remote configured — run `mine stash sync remote <url>` first")
	}

	// Get current branch name.
	branch, err := gitCmd(dir, "branch", "--show-current")
	if err != nil {
		return fmt.Errorf("getting branch: %w", err)
	}
	branch = strings.TrimSpace(branch)
	if branch == "" {
		branch = "main"
	}

	if _, err := gitCmd(dir, "push", "-u", "origin", branch); err != nil {
		return fmt.Errorf("push failed: %w", err)
	}
	return nil
}

// SyncPull pulls from the configured remote.
func SyncPull() error {
	dir := Dir()
	if !IsGitRepo() {
		return fmt.Errorf("no version history yet — run `mine stash commit` first")
	}

	// Check remote exists.
	remote, _ := gitCmd(dir, "remote", "get-url", "origin")
	if strings.TrimSpace(remote) == "" {
		return fmt.Errorf("no remote configured — run `mine stash sync remote <url>` first")
	}

	if _, err := gitCmd(dir, "pull", "--rebase", "origin"); err != nil {
		return fmt.Errorf("pull failed — you may need to resolve conflicts manually in %s: %w", dir, err)
	}

	// After pull, restore tracked files to their source locations.
	entries, err := ReadManifest()
	if err != nil {
		return fmt.Errorf("reading manifest: %w", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("determining home directory: %w", err)
	}
	sep := string(os.PathSeparator)

	for _, e := range entries {
		// Validate SafeName to avoid path traversal within the stash directory.
		if err := validateSafeName(e.SafeName); err != nil {
			return fmt.Errorf("invalid manifest entry for %s: %w", e.Source, err)
		}

		// Validate Source: must be an absolute path under the user's home directory.
		if e.Source == "" {
			return fmt.Errorf("invalid manifest entry: empty Source for SafeName %q", e.SafeName)
		}
		srcPath := filepath.Clean(e.Source)
		if !filepath.IsAbs(srcPath) {
			return fmt.Errorf("invalid manifest entry for %s: source path is not absolute", e.Source)
		}
		rel, err := filepath.Rel(home, srcPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping manifest entry with unresolvable source path: %s\n", e.Source)
			continue
		}
		if rel == ".." || strings.HasPrefix(rel, ".."+sep) {
			fmt.Fprintf(os.Stderr, "warning: skipping manifest entry with source outside home directory: %s\n", e.Source)
			continue
		}

		stashPath := filepath.Join(dir, e.SafeName)
		data, err := os.ReadFile(stashPath)
		if err != nil {
			// If the stash file is missing or unreadable, skip this entry.
			continue
		}

		// Preserve existing file mode if the source file already exists.
		mode := os.FileMode(0o644)
		if info, err := os.Stat(srcPath); err == nil {
			mode = info.Mode().Perm()
		}

		if err := os.WriteFile(srcPath, data, mode); err != nil {
			return fmt.Errorf("restoring %s: %w", e.Source, err)
		}
	}
	return nil
}

// SyncRemoteURL returns the configured remote URL, or empty string if none.
func SyncRemoteURL() string {
	dir := Dir()
	out, _ := gitCmd(dir, "remote", "get-url", "origin")
	return strings.TrimSpace(out)
}

// gitCmd runs a git command in the stash directory and returns stdout.
func gitCmd(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s", msg)
	}
	return stdout.String(), nil
}
