package grow

import (
	"testing"
	"time"
)

func mustDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestComputeStreak_Empty(t *testing.T) {
	now := mustDate("2026-02-26")
	info := ComputeStreak(nil, now)
	if info.Current != 0 || info.Longest != 0 {
		t.Fatalf("expected 0,0 for empty dates, got %d,%d", info.Current, info.Longest)
	}
}

func TestComputeStreak_TodayOnly(t *testing.T) {
	now := mustDate("2026-02-26")
	dates := []string{"2026-02-26"}
	info := ComputeStreak(dates, now)
	if info.Current != 1 {
		t.Errorf("current streak = %d, want 1", info.Current)
	}
	if info.Longest != 1 {
		t.Errorf("longest streak = %d, want 1", info.Longest)
	}
}

func TestComputeStreak_YesterdayOnly_StillActive(t *testing.T) {
	now := mustDate("2026-02-26")
	dates := []string{"2026-02-25"}
	info := ComputeStreak(dates, now)
	if info.Current != 1 {
		t.Errorf("current streak = %d, want 1 (grace: logged yesterday)", info.Current)
	}
}

func TestComputeStreak_ConsecutiveThreeDays(t *testing.T) {
	now := mustDate("2026-02-26")
	dates := []string{"2026-02-26", "2026-02-25", "2026-02-24"}
	info := ComputeStreak(dates, now)
	if info.Current != 3 {
		t.Errorf("current streak = %d, want 3", info.Current)
	}
	if info.Longest != 3 {
		t.Errorf("longest streak = %d, want 3", info.Longest)
	}
}

func TestComputeStreak_BrokenStreak(t *testing.T) {
	// Gap two days ago — streak should be 0 from today (no activity today or yesterday).
	now := mustDate("2026-02-26")
	dates := []string{"2026-02-24", "2026-02-23"}
	info := ComputeStreak(dates, now)
	if info.Current != 0 {
		t.Errorf("current streak = %d, want 0 (streak broken)", info.Current)
	}
	if info.Longest != 2 {
		t.Errorf("longest streak = %d, want 2", info.Longest)
	}
}

func TestComputeStreak_LongestLongerThanCurrent(t *testing.T) {
	now := mustDate("2026-02-26")
	// Old 5-day run: Feb 10–14
	// Current 2-day run: Feb 25–26
	dates := []string{
		"2026-02-26",
		"2026-02-25",
		"2026-02-14",
		"2026-02-13",
		"2026-02-12",
		"2026-02-11",
		"2026-02-10",
	}
	info := ComputeStreak(dates, now)
	if info.Current != 2 {
		t.Errorf("current streak = %d, want 2", info.Current)
	}
	if info.Longest != 5 {
		t.Errorf("longest streak = %d, want 5", info.Longest)
	}
}

func TestComputeStreak_NotBrokenYesterdayOnly(t *testing.T) {
	// User logged yesterday but not today — grace period keeps streak alive.
	now := mustDate("2026-02-26")
	dates := []string{"2026-02-25", "2026-02-24", "2026-02-23"}
	info := ComputeStreak(dates, now)
	if info.Current != 3 {
		t.Errorf("current streak = %d, want 3 (grace period active)", info.Current)
	}
}

func TestComputeStreak_SingleOldDate(t *testing.T) {
	now := mustDate("2026-02-26")
	dates := []string{"2026-01-01"}
	info := ComputeStreak(dates, now)
	if info.Current != 0 {
		t.Errorf("current streak = %d, want 0 (old date)", info.Current)
	}
	if info.Longest != 1 {
		t.Errorf("longest streak = %d, want 1", info.Longest)
	}
}
