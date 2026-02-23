package agents

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DiffOptions controls the behavior of the Diff operation.
type DiffOptions struct {
	Agent string // filter to a single agent name; empty means all
}

// DiffEntry describes the diff result for a single link entry.
type DiffEntry struct {
	Link  LinkEntry
	State LinkHealthState
	Lines []string // diff output lines; empty when entries are identical or not applicable
	Err   error
}

// Diff computes content differences between the canonical store and linked targets.
//
// Diff semantics by link state:
//   - linked (symlink): always matches canonical (same inode) — no diff lines
//   - linked (copy): matches canonical — no diff lines
//   - diverged (copy-mode): content differs — diff lines shown
//   - replaced (regular file where symlink expected): diff between canonical and replacement
//   - broken/unlinked: reported as an error, no diff lines
func Diff(opts DiffOptions) ([]DiffEntry, error) {
	if !IsInitialized() {
		return nil, fmt.Errorf("agents store not initialized — run %s first", "mine agents init")
	}

	m, err := ReadManifest()
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	storeDir := Dir()
	var entries []DiffEntry

	for _, link := range m.Links {
		if opts.Agent != "" && link.Agent != opts.Agent {
			continue
		}

		h := CheckLinkHealth(link, storeDir)
		entry := DiffEntry{
			Link:  link,
			State: h.State,
		}

		switch h.State {
		case LinkHealthLinked:
			// Symlink or matching copy — no diff to show.

		case LinkHealthDiverged, LinkHealthReplaced:
			// Copy diverged or regular file where symlink expected — show diff.
			sourcePath := filepath.Join(storeDir, link.Source)
			lines, diffErr := diffPaths(sourcePath, link.Target)
			if diffErr != nil {
				entry.Err = diffErr
			} else {
				entry.Lines = lines
			}

		case LinkHealthBroken:
			entry.Err = fmt.Errorf("symlink broken or target missing")

		case LinkHealthUnlinked:
			entry.Err = fmt.Errorf("target does not exist")
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// diffPaths returns unified-diff lines between paths a (canonical) and b (target).
// It attempts to use `git diff --no-index` for proper unified diff output, and
// falls back to a simple line-based diff when git is not available.
func diffPaths(a, b string) ([]string, error) {
	lines, ok, err := runGitDiffNoIndex(a, b)
	if err == nil {
		if !ok {
			return nil, nil // no differences
		}
		return lines, nil
	}

	// git not available or failed — fall back to built-in diff.
	return fallbackDiff(a, b)
}

// runGitDiffNoIndex runs `git diff --no-index` and returns the output lines.
// ok is true when differences were found (git exit code 1), false when identical.
// err is non-nil only for genuine errors (not the expected exit code 1).
func runGitDiffNoIndex(a, b string) (lines []string, ok bool, err error) {
	cmd := exec.Command("git", "diff", "--no-index", "--", a, b)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()
	if runErr == nil {
		// Exit code 0 — no differences.
		return nil, false, nil
	}

	exitErr, isExit := runErr.(*exec.ExitError)
	if isExit && exitErr.ExitCode() == 1 {
		// Exit code 1 — differences found; this is normal for git diff.
		raw := stdout.String()
		if raw == "" {
			return nil, false, nil
		}
		result := strings.Split(strings.TrimRight(raw, "\n"), "\n")
		return result, true, nil
	}

	// Genuine error (git not found, permissions, etc.).
	msg := strings.TrimSpace(stderr.String())
	if msg == "" {
		msg = runErr.Error()
	}
	return nil, false, fmt.Errorf("git diff --no-index: %s", msg)
}

// fallbackDiff produces a simplified diff without external tools.
// For files it does a line-based comparison; for directories it recurses.
func fallbackDiff(a, b string) ([]string, error) {
	aInfo, err := os.Stat(a)
	if err != nil {
		return nil, fmt.Errorf("reading source %s: %w", a, err)
	}

	if aInfo.IsDir() {
		return fallbackDirDiff(a, b)
	}
	return fallbackFileDiff(a, b, a, b)
}

// fallbackFileDiff produces a simple unified-diff for two files.
// It uses an LCS-based line comparison to correctly handle duplicate lines
// and preserve ordering.
func fallbackFileDiff(aPath, bPath, aLabel, bLabel string) ([]string, error) {
	aData, err := os.ReadFile(aPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", aPath, err)
	}
	bData, err := os.ReadFile(bPath)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", bPath, err)
	}
	if bytes.Equal(aData, bData) {
		return nil, nil
	}

	aLines := strings.Split(strings.TrimRight(string(aData), "\n"), "\n")
	bLines := strings.Split(strings.TrimRight(string(bData), "\n"), "\n")

	var result []string
	result = append(result, fmt.Sprintf("--- %s", aLabel))
	result = append(result, fmt.Sprintf("+++ %s", bLabel))
	result = append(result, lcsLineDiff(aLines, bLines)...)
	return result, nil
}

// lcsLineDiff computes a line-level diff using longest common subsequence.
// This correctly handles duplicate lines and preserves ordering, unlike a
// set-membership approach.
func lcsLineDiff(a, b []string) []string {
	m, n := len(a), len(b)

	// Build LCS length table.
	dp := make([][]int, m+1)
	for i := range dp {
		dp[i] = make([]int, n+1)
	}
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if a[i-1] == b[j-1] {
				dp[i][j] = dp[i-1][j-1] + 1
			} else if dp[i-1][j] > dp[i][j-1] {
				dp[i][j] = dp[i-1][j]
			} else {
				dp[i][j] = dp[i][j-1]
			}
		}
	}

	// Backtrack through the LCS table to produce diff lines.
	var result []string
	i, j := m, n
	for i > 0 || j > 0 {
		switch {
		case i > 0 && j > 0 && a[i-1] == b[j-1]:
			// Common line — omit context lines for brevity.
			i--
			j--
		case j > 0 && (i == 0 || dp[i][j-1] >= dp[i-1][j]):
			result = append([]string{"+" + b[j-1]}, result...)
			j--
		default:
			result = append([]string{"-" + a[i-1]}, result...)
			i--
		}
	}
	return result
}

// fallbackDirDiff recurses into both directories, diffing each file it finds.
func fallbackDirDiff(a, b string) ([]string, error) {
	aEntries, err := os.ReadDir(a)
	if err != nil {
		return nil, fmt.Errorf("reading dir %s: %w", a, err)
	}

	var allLines []string
	seen := make(map[string]bool, len(aEntries))

	for _, entry := range aEntries {
		seen[entry.Name()] = true
		aPath := filepath.Join(a, entry.Name())
		bPath := filepath.Join(b, entry.Name())

		if entry.IsDir() {
			lines, recurseErr := fallbackDirDiff(aPath, bPath)
			if recurseErr != nil {
				allLines = append(allLines,
					fmt.Sprintf("--- %s", aPath),
					fmt.Sprintf("+++ %s (error diffing directory: %v)", bPath, recurseErr),
				)
			} else {
				allLines = append(allLines, lines...)
			}
			continue
		}

		lines, fileErr := fallbackFileDiff(aPath, bPath, aPath, bPath)
		if fileErr != nil {
			allLines = append(allLines,
				fmt.Sprintf("--- %s", aPath),
				fmt.Sprintf("+++ %s (error diffing file: %v)", bPath, fileErr),
			)
		} else {
			allLines = append(allLines, lines...)
		}
	}

	// Report files that exist only in the target.
	bEntries, err := os.ReadDir(b)
	if err != nil {
		return allLines, fmt.Errorf("reading dir %s: %w", b, err)
	}
	for _, entry := range bEntries {
		if seen[entry.Name()] || entry.IsDir() {
			continue
		}
		allLines = append(allLines,
			fmt.Sprintf("--- %s", filepath.Join(a, entry.Name())),
			fmt.Sprintf("+++ %s (only in target)", filepath.Join(b, entry.Name())),
		)
	}

	return allLines, nil
}
