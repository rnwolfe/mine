package version

import (
	"runtime/debug"
	"strings"
	"testing"
)

func TestFull(t *testing.T) {
	result := Full()
	if result == "" {
		t.Fatal("Full() returned empty string")
	}
	if !strings.Contains(result, Version) {
		t.Errorf("Full() %q does not contain version %q", result, Version)
	}
}

func TestShort(t *testing.T) {
	result := Short()
	if result != Version {
		t.Errorf("Short() = %q, want %q", result, Version)
	}
}

// withDefaults runs fn with Version, Commit, and Date reset to their ldflag
// defaults, restoring the original values when fn returns.
func withDefaults(fn func()) {
	origVersion, origCommit, origDate := Version, Commit, Date
	Version, Commit, Date = "dev", "none", "unknown"
	defer func() {
		Version, Commit, Date = origVersion, origCommit, origDate
	}()
	fn()
}

func buildInfo(mainVersion string, settings map[string]string) *debug.BuildInfo {
	info := &debug.BuildInfo{
		Main: debug.Module{Version: mainVersion},
	}
	for k, v := range settings {
		info.Settings = append(info.Settings, debug.BuildSetting{Key: k, Value: v})
	}
	return info
}

// TestBackfillFromBuildInfo_AllDefaults verifies that when all three vars are at
// their ldflag defaults, backfillFromBuildInfo fills them from build info.
func TestBackfillFromBuildInfo_AllDefaults(t *testing.T) {
	withDefaults(func() {
		info := buildInfo("v0.1.0", map[string]string{
			"vcs.revision": "abcdef1234567",
			"vcs.time":     "2024-01-15T10:00:00Z",
		})
		backfillFromBuildInfo(info)

		if Version != "v0.1.0" {
			t.Errorf("Version = %q, want %q", Version, "v0.1.0")
		}
		if Commit != "abcdef1" {
			t.Errorf("Commit = %q, want %q", Commit, "abcdef1")
		}
		if Date != "2024-01-15T10:00:00Z" {
			t.Errorf("Date = %q, want %q", Date, "2024-01-15T10:00:00Z")
		}
	})
}

// TestBackfillFromBuildInfo_LdflagsPrecedence verifies that non-default ldflag
// values are not overwritten by backfillFromBuildInfo.
func TestBackfillFromBuildInfo_LdflagsPrecedence(t *testing.T) {
	origVersion, origCommit, origDate := Version, Commit, Date
	defer func() {
		Version, Commit, Date = origVersion, origCommit, origDate
	}()

	// Simulate ldflags having already set real values.
	Version = "v1.2.3"
	Commit = "deadbee"
	Date = "2025-06-01T00:00:00Z"

	info := buildInfo("v0.0.1", map[string]string{
		"vcs.revision": "aaaaaaa",
		"vcs.time":     "2000-01-01T00:00:00Z",
	})
	backfillFromBuildInfo(info)

	if Version != "v1.2.3" {
		t.Errorf("Version overwritten: got %q, want %q", Version, "v1.2.3")
	}
	if Commit != "deadbee" {
		t.Errorf("Commit overwritten: got %q, want %q", Commit, "deadbee")
	}
	if Date != "2025-06-01T00:00:00Z" {
		t.Errorf("Date overwritten: got %q, want %q", Date, "2025-06-01T00:00:00Z")
	}
}

// TestBackfillFromBuildInfo_DevelVersion verifies that when info.Main.Version is
// "(devel)" (built from HEAD without a tag), Version stays "dev" but Commit and
// Date are still populated from VCS stamps.
func TestBackfillFromBuildInfo_DevelVersion(t *testing.T) {
	withDefaults(func() {
		info := buildInfo("(devel)", map[string]string{
			"vcs.revision": "cafebabe1234",
			"vcs.time":     "2025-02-21T08:30:00Z",
		})
		backfillFromBuildInfo(info)

		if Version != "dev" {
			t.Errorf("Version = %q, want %q (devel should keep 'dev')", Version, "dev")
		}
		if Commit != "cafebab" {
			t.Errorf("Commit = %q, want %q", Commit, "cafebab")
		}
		if Date != "2025-02-21T08:30:00Z" {
			t.Errorf("Date = %q, want %q", Date, "2025-02-21T08:30:00Z")
		}
	})
}

// TestBackfillFromBuildInfo_NilInfo verifies that passing nil does not panic
// and leaves variables unchanged.
func TestBackfillFromBuildInfo_NilInfo(t *testing.T) {
	withDefaults(func() {
		backfillFromBuildInfo(nil)

		if Version != "dev" {
			t.Errorf("Version changed on nil info: %q", Version)
		}
		if Commit != "none" {
			t.Errorf("Commit changed on nil info: %q", Commit)
		}
		if Date != "unknown" {
			t.Errorf("Date changed on nil info: %q", Date)
		}
	})
}

// TestBackfillFromBuildInfo_ShortRevision verifies that short revisions (≤7 chars)
// are kept as-is rather than being truncated.
func TestBackfillFromBuildInfo_ShortRevision(t *testing.T) {
	withDefaults(func() {
		info := buildInfo("v0.2.0", map[string]string{
			"vcs.revision": "abc123",
		})
		backfillFromBuildInfo(info)

		if Commit != "abc123" {
			t.Errorf("Commit = %q, want %q", Commit, "abc123")
		}
	})
}

// TestBackfillFromBuildInfo_EmptyVCS verifies that missing VCS settings leave
// Commit and Date at their defaults.
func TestBackfillFromBuildInfo_EmptyVCS(t *testing.T) {
	withDefaults(func() {
		info := buildInfo("v0.3.0", nil)
		backfillFromBuildInfo(info)

		if Version != "v0.3.0" {
			t.Errorf("Version = %q, want v0.3.0", Version)
		}
		if Commit != "none" {
			t.Errorf("Commit = %q, want none (no VCS setting)", Commit)
		}
		if Date != "unknown" {
			t.Errorf("Date = %q, want unknown (no VCS setting)", Date)
		}
	})
}
