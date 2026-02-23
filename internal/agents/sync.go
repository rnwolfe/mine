package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SyncSetRemote configures the git remote for the agents store.
// If a remote named "origin" already exists, it is updated. Otherwise it is added.
func SyncSetRemote(url string) error {
	dir := Dir()
	if !IsGitRepo() {
		return fmt.Errorf("no version history yet — run `mine agents commit` first")
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

// SyncRemoteURL returns the configured remote URL, or empty string if none.
func SyncRemoteURL() string {
	dir := Dir()
	out, _ := gitCmd(dir, "remote", "get-url", "origin")
	return strings.TrimSpace(out)
}

// SyncPush pushes the agents store to the configured remote.
func SyncPush() error {
	dir := Dir()
	if !IsGitRepo() {
		return fmt.Errorf("no version history yet — run `mine agents commit` first")
	}

	remote := SyncRemoteURL()
	if remote == "" {
		return fmt.Errorf("no remote configured — run `mine agents sync remote <url>` first")
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

// SyncPullResult holds a summary of the distribution actions taken after a pull.
type SyncPullResult struct {
	CopiedLinks int
}

// SyncPull pulls from the configured remote with rebase. After pulling, copy-mode
// links are re-copied to their targets. Symlink-mode links are automatically
// up-to-date and require no action.
func SyncPull() error {
	_, err := SyncPullWithResult()
	return err
}

// SyncPullWithResult pulls from the configured remote and returns a summary of
// distribution actions taken. Copy-mode links are re-copied to their targets
// after the pull. Symlink-mode links are already up-to-date via the symlink.
func SyncPullWithResult() (*SyncPullResult, error) {
	dir := Dir()
	if !IsGitRepo() {
		return nil, fmt.Errorf("no version history yet — run `mine agents commit` first")
	}

	remote := SyncRemoteURL()
	if remote == "" {
		return nil, fmt.Errorf("no remote configured — run `mine agents sync remote <url>` first")
	}

	if _, err := gitCmd(dir, "pull", "--rebase", "origin"); err != nil {
		return nil, fmt.Errorf("pull failed — you may need to resolve conflicts manually in %s: %w", dir, err)
	}

	// Re-read manifest: it may have changed on the remote.
	manifest, err := ReadManifest()
	if err != nil {
		return nil, fmt.Errorf("reading manifest after pull: %w", err)
	}

	result := &SyncPullResult{}
	for _, link := range manifest.Links {
		if link.Mode != "copy" {
			continue
		}

		srcPath := filepath.Join(dir, link.Source)
		data, err := os.ReadFile(srcPath)
		if err != nil {
			// Source file missing in updated store — skip.
			continue
		}

		// Preserve existing target permissions if available.
		mode := os.FileMode(0o644)
		if info, err := os.Stat(link.Target); err == nil {
			mode = info.Mode().Perm()
		}

		// Ensure target's parent directory exists.
		if err := os.MkdirAll(filepath.Dir(link.Target), 0o755); err != nil {
			return nil, fmt.Errorf("creating directory for %s: %w", link.Target, err)
		}

		if err := os.WriteFile(link.Target, data, mode); err != nil {
			return nil, fmt.Errorf("re-copying %s to %s: %w", link.Source, link.Target, err)
		}
		result.CopiedLinks++
	}

	return result, nil
}
