package grow

import (
	"time"
)

// StreakInfo holds current and longest streak values.
type StreakInfo struct {
	Current int
	Longest int
}

// ComputeStreak calculates the current and longest learning streaks from a list
// of activity dates (in "YYYY-MM-DD" format, sorted descending).
//
// A streak is consecutive calendar days with >= 1 activity. The current streak
// is not broken if the user logged yesterday but not yet today (grace period).
//
// The dates slice must be sorted descending (most recent first).
// now is used as the reference for "today" / "yesterday".
func ComputeStreak(dates []string, now time.Time) StreakInfo {
	if len(dates) == 0 {
		return StreakInfo{}
	}

	utcNow := now.UTC()
	today := utcNow.Format("2006-01-02")
	yesterday := utcNow.AddDate(0, 0, -1).Format("2006-01-02")

	// Current streak: starting from the most recent date, which must be today
	// or yesterday for the streak to be active.
	var current int
	if dates[0] == today || dates[0] == yesterday {
		current = 1
		for i := 1; i < len(dates); i++ {
			prev, _ := time.Parse("2006-01-02", dates[i-1])
			curr, _ := time.Parse("2006-01-02", dates[i])
			if prev.AddDate(0, 0, -1).Format("2006-01-02") == curr.Format("2006-01-02") {
				current++
			} else {
				break
			}
		}
	}

	// Longest streak: scan all dates in ascending order.
	asc := make([]string, len(dates))
	for i, d := range dates {
		asc[len(dates)-1-i] = d
	}

	longest := 1
	run := 1
	for i := 1; i < len(asc); i++ {
		prev, _ := time.Parse("2006-01-02", asc[i-1])
		curr, _ := time.Parse("2006-01-02", asc[i])
		if curr.AddDate(0, 0, -1).Format("2006-01-02") == prev.Format("2006-01-02") {
			run++
			if run > longest {
				longest = run
			}
		} else {
			run = 1
		}
	}

	if current > longest {
		longest = current
	}

	return StreakInfo{Current: current, Longest: longest}
}

// GetStreak retrieves activity dates from the store and computes the streak.
func (s *Store) GetStreak(now time.Time) (StreakInfo, error) {
	dates, err := s.ActivityDatesDesc()
	if err != nil {
		return StreakInfo{}, err
	}
	return ComputeStreak(dates, now), nil
}
