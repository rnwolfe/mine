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

// SafeNameFor returns the stash-safe filename for an absolute source path.
// The home-relative portion of the path has "/" replaced with "__".
func SafeNameFor(source string) string {
	home, _ := os.UserHomeDir()
	relPath := strings.TrimPrefix(source, home+"/")
	return strings.ReplaceAll(relPath, "/", "__")
}

// TrackFile copies source into the stash directory and registers it in the
// manifest. Returns the Entry for the newly tracked file.
//
// NOTE: The read-check-append sequence used to update the manifest is NOT
// atomic. Concurrent calls may produce duplicate manifest entries (TOCTOU
// race). A mutex or file-level lock is required for correct concurrent use.
// See follow-up issue for the planned fix.
func TrackFile(source string) (*Entry, error) {
	info, err := os.Stat(source)
	if err != nil {
		return nil, fmt.Errorf("can't find %s", source)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("can't track directories yet (coming soon)")
	}

	dir := Dir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	safeName := SafeNameFor(source)
	dest := filepath.Join(dir, safeName)

	data, err := os.ReadFile(source)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", source, err)
	}
	if err := os.WriteFile(dest, data, info.Mode()); err != nil {
		return nil, fmt.Errorf("writing to stash: %w", err)
	}

	// Update manifest.
	// WARNING: The read-check-append sequence below is NOT atomic. Concurrent
	// callers may both observe the manifest before any append, causing both to
	// write an entry — resulting in duplicate lines for the same source. This
	// is a known TOCTOU limitation; a follow-up issue tracks the fix.
	manifestPath := ManifestPath()
	manifest, _ := os.ReadFile(manifestPath)
	entry := source + " -> " + safeName + "\n"
	if !strings.Contains(string(manifest), source) {
		f, err := os.OpenFile(manifestPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		if _, err := f.WriteString(entry); err != nil {
			return nil, err
		}
	}

	return &Entry{Source: source, SafeName: safeName}, nil
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

// validateEntryWithHome is the core validation logic for a manifest entry,
// accepting a pre-resolved home directory so callers can compute it once per
// operation when validating multiple entries in a loop.
func validateEntryWithHome(e Entry, home string) error {
	if err := validateSafeName(e.SafeName); err != nil {
		return err
	}
	if e.Source == "" {
		return fmt.Errorf("empty Source")
	}
	srcPath := filepath.Clean(e.Source)
	if !filepath.IsAbs(srcPath) {
		return fmt.Errorf("source path is not absolute: %q", e.Source)
	}
	sep := string(os.PathSeparator)
	rel, err := filepath.Rel(home, srcPath)
	if err != nil {
		return fmt.Errorf("source path %q is not resolvable relative to home directory %q: %w", e.Source, home, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+sep) {
		return fmt.Errorf("source path %q escapes home directory", e.Source)
	}
	return nil
}

// ValidateEntry validates the safety invariants of a manifest entry.
// It checks that SafeName is non-empty and free of path traversal characters,
// that Source is non-empty and an absolute path, and that Source does not
// escape the user's home directory.
// It avoids explicit filesystem checks — file existence, readability, and type
// checks are the caller's responsibility — but may consult OS user info to
// resolve the home directory.
func ValidateEntry(e Entry) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("determining home directory: %w", err)
	}
	return validateEntryWithHome(e, home)
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
	// Resolve home dir once for all entry validation.
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determining home directory: %w", err)
	}
	for _, e := range entries {
		if err := validateEntryWithHome(e, home); err != nil {
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
// Returns the Entry for the restored file to avoid duplicate FindEntry calls.
//
// When force is false (default), the restored file inherits the current source
// file's permissions, falling back to 0644 if the source does not exist yet.
// When force is true, the restored file uses the permissions recorded in the
// stash copy (captured at track/commit time), overriding the current source
// file's permissions.
func RestoreToSource(file string, version string, force bool) (*Entry, error) {
	entry, err := FindEntry(file)
	if err != nil {
		return nil, err
	}

	content, err := Restore(file, version)
	if err != nil {
		return nil, err
	}

	stashPath := filepath.Join(Dir(), entry.SafeName)

	// Determine permissions for the restored file.
	// In all cases the existing source file is removed before recreating so that
	// read-only source files (e.g. 0444) can be written without a permission
	// error — on most Unix filesystems, the directory write permission governs
	// deletion, not the file's own mode bits.
	srcPerm := os.FileMode(0o644)
	if force {
		// Use permissions from the stash copy (captured at track/commit time),
		// ignoring the current source file's permissions.
		if info, err := os.Stat(stashPath); err == nil {
			srcPerm = info.Mode().Perm()
		}
		if err := os.Remove(entry.Source); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("removing existing %s before restore: %w", entry.Source, err)
		}
	} else {
		// Preserve existing source file permissions when present,
		// otherwise fall back to 0644.
		if info, err := os.Stat(entry.Source); err == nil {
			srcPerm = info.Mode().Perm()
			if err := os.Remove(entry.Source); err != nil && !os.IsNotExist(err) {
				return nil, fmt.Errorf("removing existing %s before restore: %w", entry.Source, err)
			}
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("stat source %s: %w", entry.Source, err)
		}
	}

	if err := os.WriteFile(entry.Source, content, srcPerm); err != nil {
		return nil, fmt.Errorf("writing to %s: %w", entry.Source, err)
	}

	// Also update the stash copy.
	stashPerm := os.FileMode(0o600)
	if info, err := os.Stat(stashPath); err == nil {
		stashPerm = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("stat stash copy %s: %w", stashPath, err)
	}

	if err := os.WriteFile(stashPath, content, stashPerm); err != nil {
		return nil, fmt.Errorf("updating stash copy: %w", err)
	}

	return entry, nil
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
