package stash

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// BenchmarkCommit_SmallFile measures Commit performance with a single ~1 KB tracked file.
//
// NOTE: git subprocess overhead (git add, git status, git commit, git rev-parse)
// dominates the measured time. The Go-level allocations are a small fraction.
// This is expected and reflects real-world Commit cost.
func BenchmarkCommit_SmallFile(b *testing.B) {
	b.ReportAllocs()
	stashDir, homeDir := setupEnv(b)

	srcPath := filepath.Join(homeDir, "small.txt")
	safeName := "small.txt"
	content := bytes.Repeat([]byte("x"), 1024)

	if err := os.MkdirAll(stashDir, 0o755); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(srcPath, content, 0o644); err != nil {
		b.Fatal(err)
	}
	manifestEntry := srcPath + " -> " + safeName + "\n"
	if err := os.WriteFile(ManifestPath(), []byte("# mine stash manifest\n"+manifestEntry), 0o644); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stashDir, safeName), content, 0o644); err != nil {
		b.Fatal(err)
	}

	// Warm-up commit: initializes git repo and creates the initial commit.
	// Excluded from measurement via b.ResetTimer below.
	if _, err := Commit("warmup"); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Mutate source file so each iteration has something to commit.
		prefix := fmt.Sprintf("iter-%d-", i)
		copy(content, []byte(prefix))
		if err := os.WriteFile(srcPath, content, 0o644); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()

		if _, err := Commit(fmt.Sprintf("iter %d", i)); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCommit_LargeFile measures Commit performance with a single ~10 MB tracked file.
//
// NOTE: I/O dominates here — os.WriteFile copies 10 MB to disk each iteration
// (excluded from timing via StopTimer) and git hashes/stores a 10 MB blob.
// If this benchmark reveals I/O as a bottleneck, consider streaming writes
// instead of loading the entire file into memory in Commit.
func BenchmarkCommit_LargeFile(b *testing.B) {
	b.ReportAllocs()
	stashDir, homeDir := setupEnv(b)

	srcPath := filepath.Join(homeDir, "large.txt")
	safeName := "large.txt"
	const size = 10 * 1024 * 1024
	content := bytes.Repeat([]byte("x"), size)

	if err := os.MkdirAll(stashDir, 0o755); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(srcPath, content, 0o644); err != nil {
		b.Fatal(err)
	}
	manifestEntry := srcPath + " -> " + safeName + "\n"
	if err := os.WriteFile(ManifestPath(), []byte("# mine stash manifest\n"+manifestEntry), 0o644); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stashDir, safeName), content, 0o644); err != nil {
		b.Fatal(err)
	}

	// Warm-up commit: initializes git repo and creates the initial commit.
	// Excluded from measurement via b.ResetTimer below.
	if _, err := Commit("warmup"); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Patch prefix bytes so content differs each iteration; reuse the buffer
		// to avoid a 10 MB allocation per iteration.
		prefix := fmt.Sprintf("iter-%d-", i)
		copy(content, []byte(prefix))
		if err := os.WriteFile(srcPath, content, 0o644); err != nil {
			b.Fatal(err)
		}
		b.StartTimer()

		if _, err := Commit(fmt.Sprintf("iter %d", i)); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCommit_ManyFiles measures Commit performance with 50 tracked files (~1 KB each).
//
// NOTE: All 50 files share the same content within each iteration (same prefix mutation).
// git deduplicates identical blobs, so this may be slightly optimistic vs a real-world
// scenario where each file has unique content. The measured cost reflects file-count
// overhead (manifest scan, git add staging, N file copies) rather than blob uniqueness.
// If this benchmark shows disproportionate time vs BenchmarkCommit_SmallFile
// beyond a 50× I/O factor, the bottleneck is likely in the manifest scan or
// git add staging overhead for many files. Open a follow-up issue to investigate.
func BenchmarkCommit_ManyFiles(b *testing.B) {
	b.ReportAllocs()
	stashDir, homeDir := setupEnv(b)

	const fileCount = 50
	const fileSize = 1024
	content := bytes.Repeat([]byte("x"), fileSize)

	if err := os.MkdirAll(stashDir, 0o755); err != nil {
		b.Fatal(err)
	}

	srcPaths := make([]string, fileCount)
	var manifestLines strings.Builder
	manifestLines.WriteString("# mine stash manifest\n")
	for i := 0; i < fileCount; i++ {
		name := fmt.Sprintf("file%02d.txt", i)
		srcPath := filepath.Join(homeDir, name)
		srcPaths[i] = srcPath
		if err := os.WriteFile(srcPath, content, 0o644); err != nil {
			b.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(stashDir, name), content, 0o644); err != nil {
			b.Fatal(err)
		}
		manifestLines.WriteString(srcPath + " -> " + name + "\n")
	}
	if err := os.WriteFile(ManifestPath(), []byte(manifestLines.String()), 0o644); err != nil {
		b.Fatal(err)
	}

	// Warm-up commit: initializes git repo and creates the initial commit.
	// Excluded from measurement via b.ResetTimer below.
	if _, err := Commit("warmup"); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Mutate all 50 source files with a unique prefix so there is something to commit.
		prefix := fmt.Sprintf("iter-%d-", i)
		copy(content, []byte(prefix))
		for _, srcPath := range srcPaths {
			if err := os.WriteFile(srcPath, content, 0o644); err != nil {
				b.Fatal(err)
			}
		}
		b.StartTimer()

		if _, err := Commit(fmt.Sprintf("iter %d", i)); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkReadManifest_ManyEntries measures ReadManifest parse performance with a
// 100-entry manifest. No git operations or real source files are needed —
// ReadManifest only reads and parses the manifest text file.
//
// NOTE: ReadManifest scans entries linearly (O(n)). If this benchmark reveals
// non-linear scaling as entry count grows, consider indexing the manifest into
// a map[SafeName]Entry on first load and opening a follow-up issue.
func BenchmarkReadManifest_ManyEntries(b *testing.B) {
	b.ReportAllocs()
	stashDir, homeDir := setupEnv(b)

	if err := os.MkdirAll(stashDir, 0o755); err != nil {
		b.Fatal(err)
	}

	// Build a synthetic 100-entry manifest; no real source files needed —
	// ReadManifest only reads and parses the manifest text, it does not stat source paths.
	var manifestLines strings.Builder
	manifestLines.WriteString("# mine stash manifest\n")
	for i := 0; i < 100; i++ {
		name := fmt.Sprintf("file%03d.txt", i)
		srcPath := filepath.Join(homeDir, name)
		manifestLines.WriteString(srcPath + " -> " + name + "\n")
	}
	if err := os.WriteFile(ManifestPath(), []byte(manifestLines.String()), 0o644); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ReadManifest(); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRestoreToSource_LargeFile measures RestoreToSource performance for a ~10 MB
// stashed file. Each iteration reads 10 MB from git object storage and writes it to
// both the source path and the stash copy.
//
// NOTE: RestoreToSource calls FindEntry (which calls ReadManifest) on every invocation.
// With a single-entry manifest this is negligible, but with 100+ entries the O(n)
// scan in FindEntry would compound the per-call cost. See BenchmarkReadManifest_ManyEntries
// for manifest-scan isolation.
//
// NOTE: After the first iteration the OS page cache serves the git object read from
// RAM. Subsequent iterations will appear faster than a cold-cache read would be.
// Interpret the per-op numbers as warm-cache throughput, not cold-start latency.
func BenchmarkRestoreToSource_LargeFile(b *testing.B) {
	b.ReportAllocs()
	stashDir, homeDir := setupEnv(b)

	srcPath := filepath.Join(homeDir, "large.txt")
	safeName := "large.txt"
	const size = 10 * 1024 * 1024
	content := bytes.Repeat([]byte("x"), size)

	if err := os.MkdirAll(stashDir, 0o755); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(srcPath, content, 0o644); err != nil {
		b.Fatal(err)
	}
	manifestEntry := srcPath + " -> " + safeName + "\n"
	if err := os.WriteFile(ManifestPath(), []byte("# mine stash manifest\n"+manifestEntry), 0o644); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stashDir, safeName), content, 0o644); err != nil {
		b.Fatal(err)
	}

	// Commit once so there is a HEAD to restore from.
	if _, err := Commit("initial"); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// RestoreToSource is idempotent: writes the same HEAD content each iteration.
		// No source mutation needed between iterations.
		if _, err := RestoreToSource(safeName, "HEAD", false); err != nil {
			b.Fatal(err)
		}
	}
}
