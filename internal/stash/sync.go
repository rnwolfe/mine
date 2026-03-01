package stash

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

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

	// Resolve home dir once for all entry validation.
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("determining home directory: %w", err)
	}
	for _, e := range entries {
		// Validate SafeName and Source path safety invariants.
		if err := validateEntryWithHome(e, home); err != nil {
			return fmt.Errorf("invalid manifest entry for %s: %w", e.Source, err)
		}
		srcPath := filepath.Clean(e.Source)

		stashPath := filepath.Join(dir, e.SafeName)
		data, err := os.ReadFile(stashPath)
		if err != nil {
			// If the stash file is missing or unreadable, skip this entry.
			continue
		}

		// Preserve existing file mode if the source file already exists.
		// Remove before recreating so that read-only source files (e.g. 0444)
		// can be written without a permission error (same strategy as RestoreToSource).
		mode := os.FileMode(0o644)
		if info, err := os.Stat(srcPath); err == nil {
			mode = info.Mode().Perm()
			if err := os.Remove(srcPath); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("removing existing %s before sync restore: %w", e.Source, err)
			}
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
