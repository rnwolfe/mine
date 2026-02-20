package stash

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// Concurrency notes
// -----------------
// These tests exercise concurrent stash operations using goroutines and a
// sync.WaitGroup. t.Parallel() is intentionally omitted at the outer level
// because each test relies on t.Setenv for per-test stash directory isolation;
// t.Setenv and t.Parallel() are mutually exclusive in Go's testing framework.
// Concurrent behavior is still exercised — multiple goroutines run within each
// test — and the -race flag will catch any Go-level data races.
//
// Run with: go test -race ./internal/stash/...

// TestConcurrentTrackDifferentFiles verifies that concurrent TrackFile calls
// on distinct source files produce a valid, non-corrupted manifest. Every
// goroutine tracks a unique file; all entries must appear in the manifest after
// all goroutines complete.
func TestConcurrentTrackDifferentFiles(t *testing.T) {
	stashDir, homeDir := setupTestEnv(t)
	if err := os.MkdirAll(stashDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Initialise an empty manifest header so appends have a consistent base.
	if err := os.WriteFile(ManifestPath(), []byte("# mine stash manifest\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	const n = 10

	// Create all source files from the main goroutine (t.Fatal is only safe
	// from the test goroutine).
	paths := make([]string, n)
	for i := 0; i < n; i++ {
		paths[i] = createTestFile(t, homeDir, fmt.Sprintf("file%d.txt", i), fmt.Sprintf("content %d", i))
	}

	var wg sync.WaitGroup
	errs := make([]error, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = TrackFile(paths[i])
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d: TrackFile error: %v", i, err)
		}
	}

	entries, err := ReadManifest()
	if err != nil {
		t.Fatalf("ReadManifest() after concurrent track: %v", err)
	}

	// All n files must be present; no entry should be malformed.
	if len(entries) != n {
		t.Errorf("manifest has %d entries after %d concurrent tracks on different files, want %d", len(entries), n, n)
	}

	for _, e := range entries {
		if e.Source == "" || e.SafeName == "" {
			t.Errorf("corrupt manifest entry: %+v", e)
		}
		// Stash copy must exist.
		stashCopy := filepath.Join(stashDir, e.SafeName)
		if _, err := os.Stat(stashCopy); err != nil {
			t.Errorf("stash copy missing for %s: %v", e.SafeName, err)
		}
	}
}

// TestConcurrentTrackSameFile verifies that concurrent TrackFile calls for the
// same source file do not corrupt the manifest. The manifest must remain
// parseable and contain at least one well-formed entry for the source file.
//
// NOTE: The current implementation has a TOCTOU race: multiple goroutines may
// read the manifest before any append, observe the source as untracked, and
// each write a duplicate entry. This is a known limitation of the read-check-
// append pattern in TrackFile. A mutex or file-level lock is needed to fix it.
// A follow-up issue should be opened to address this.
func TestConcurrentTrackSameFile(t *testing.T) {
	stashDir, homeDir := setupTestEnv(t)
	if err := os.MkdirAll(stashDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ManifestPath(), []byte("# mine stash manifest\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	source := createTestFile(t, homeDir, ".zshrc", "export PATH=$PATH")

	const n = 10
	var wg sync.WaitGroup
	errs := make([]error, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = TrackFile(source)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d: TrackFile error: %v", i, err)
		}
	}

	// The manifest must be parseable regardless of duplicate entries.
	entries, err := ReadManifest()
	if err != nil {
		t.Fatalf("ReadManifest() after concurrent same-file track: %v", err)
	}

	if len(entries) == 0 {
		t.Fatal("manifest has no entries after concurrent track of same file")
	}

	// Every entry must be well-formed (no torn writes).
	for _, e := range entries {
		if e.Source == "" || e.SafeName == "" {
			t.Errorf("corrupt manifest entry: %+v", e)
		}
		if !strings.HasPrefix(e.Source, homeDir) {
			t.Errorf("manifest entry has unexpected source %q (want prefix %q)", e.Source, homeDir)
		}
	}

	// The stash copy must exist and contain the expected content.
	if len(entries) > 0 {
		stashCopy := filepath.Join(stashDir, entries[0].SafeName)
		data, err := os.ReadFile(stashCopy)
		if err != nil {
			t.Errorf("stash copy not readable after concurrent track: %v", err)
		} else if string(data) != "export PATH=$PATH" {
			t.Errorf("stash copy has unexpected content %q", string(data))
		}
	}
}

// TestConcurrentRestore verifies that concurrent RestoreToSource calls for the
// same file do not produce a corrupted output. The source file must contain a
// complete, recognisable version after all goroutines complete.
func TestConcurrentRestore(t *testing.T) {
	stashDir, homeDir := setupTestEnv(t)
	source := createTestFile(t, homeDir, ".zshrc", "version 1")
	setupManifest(t, stashDir, source, ".zshrc", "version 1")

	if _, err := Commit("v1"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("version 2"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Commit("v2"); err != nil {
		t.Fatal(err)
	}

	const n = 10
	var wg sync.WaitGroup
	errs := make([]error, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = RestoreToSource(".zshrc", "HEAD")
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d: RestoreToSource error: %v", i, err)
		}
	}

	// Source file must contain a complete valid version — no torn writes.
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("reading source after concurrent restore: %v", err)
	}
	got := string(data)
	if got != "version 1" && got != "version 2" {
		t.Errorf("source file has unexpected content after concurrent restore: %q (want 'version 1' or 'version 2')", got)
	}
}

// TestConcurrentCommit verifies that concurrent Commit calls leave the git
// repository in a valid, queryable state. Not all commits are expected to
// succeed — git serialises writes via its index lock — but the repository must
// not be corrupted: git log must succeed and return parseable history.
//
// NOTE: git's index lock (.git/index.lock) means that concurrent Commit calls
// will often fail with an error containing "index.lock". This is expected
// behaviour from git, not a stash-layer bug. The test therefore accepts errors
// from individual goroutines as long as at least one commit succeeds and the
// repo remains queryable.
func TestConcurrentCommit(t *testing.T) {
	stashDir, homeDir := setupTestEnv(t)
	if err := os.MkdirAll(stashDir, 0o755); err != nil {
		t.Fatal(err)
	}

	const numFiles = 5
	sources := make([]string, numFiles)
	for i := 0; i < numFiles; i++ {
		sources[i] = createTestFile(t, homeDir, fmt.Sprintf("cfg%d.txt", i), fmt.Sprintf("content %d", i))
	}

	// Build manifest and stash copies from the main goroutine.
	var sb strings.Builder
	sb.WriteString("# mine stash manifest\n")
	for i, src := range sources {
		safeName := fmt.Sprintf("cfg%d.txt", i)
		sb.WriteString(src + " -> " + safeName + "\n")
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(stashDir, safeName), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(ManifestPath(), []byte(sb.String()), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create the initial commit to initialise the git repo.
	if _, err := Commit("initial"); err != nil {
		t.Fatal(err)
	}

	// Modify all source files so there is content to commit.
	for i, src := range sources {
		if err := os.WriteFile(src, []byte(fmt.Sprintf("updated %d", i)), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	const n = 8
	var wg sync.WaitGroup
	errs := make([]error, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, errs[i] = Commit(fmt.Sprintf("concurrent commit %d", i))
		}(i)
	}
	wg.Wait()

	// At least one commit must succeed.
	successCount := 0
	for _, err := range errs {
		if err == nil {
			successCount++
		}
	}
	if successCount == 0 {
		// Print errors to aid diagnosis.
		for i, err := range errs {
			if err != nil {
				t.Logf("goroutine %d error: %v", i, err)
			}
		}
		t.Error("all concurrent commits failed; at least one should succeed")
	}

	// Git repository must be in a valid, queryable state.
	logs, err := Log("")
	if err != nil {
		t.Fatalf("Log() after concurrent commits: %v", err)
	}
	if len(logs) < 2 { // initial + at least one concurrent
		t.Errorf("expected at least 2 log entries after concurrent commits, got %d", len(logs))
	}

	// Every log entry must have a non-empty hash.
	for _, le := range logs {
		if le.Hash == "" || le.Short == "" {
			t.Errorf("log entry has empty hash: %+v", le)
		}
	}
}
