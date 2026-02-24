package todo

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"time"
)

// Stats holds completion velocity and streak metrics for todos.
type Stats struct {
	Streak         int
	LongestStreak  int
	CompletedWeek  int
	CompletedMonth int
	AvgClose       time.Duration
	TotalFocus     time.Duration // from dig_sessions if available
	HasFocusData   bool          // true if dig_sessions table exists and has data
	ByProject      []ProjectStats
}

// ProjectStats holds per-project breakdown statistics.
type ProjectStats struct {
	Name      string
	Open      int
	Completed int
	AvgClose  time.Duration
}

// GetStats computes completion stats, optionally scoped to a project path.
// If projectPath is nil, returns stats across all todos.
// now is used as the reference time for streak and weekly/monthly calculations.
func GetStats(db *sql.DB, projectPath *string, now time.Time) (*Stats, error) {
	stats := &Stats{}

	var err error

	// Completion streak (consecutive days with >= 1 completion).
	stats.Streak, stats.LongestStreak, err = computeStreak(db, projectPath, now)
	if err != nil {
		return nil, fmt.Errorf("computing streak: %w", err)
	}

	// Weekly count (Monday-start weeks).
	weekStart := startOfWeek(now)
	stats.CompletedWeek, err = countCompletedSince(db, projectPath, weekStart)
	if err != nil {
		return nil, fmt.Errorf("counting weekly completions: %w", err)
	}

	// Monthly count (calendar month boundary).
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	stats.CompletedMonth, err = countCompletedSince(db, projectPath, monthStart)
	if err != nil {
		return nil, fmt.Errorf("counting monthly completions: %w", err)
	}

	// Average close time for completed todos.
	stats.AvgClose, err = avgCloseTime(db, projectPath)
	if err != nil {
		return nil, fmt.Errorf("computing avg close time: %w", err)
	}

	// Total focus time from dig_sessions (graceful fallback if table absent).
	stats.TotalFocus, stats.HasFocusData, err = totalFocusTime(db, projectPath)
	if err != nil {
		return nil, fmt.Errorf("computing focus time: %w", err)
	}

	// Per-project breakdown (always computed; callers may choose not to display
	// it when scoped to a single project).
	stats.ByProject, err = projectBreakdown(db)
	if err != nil {
		return nil, fmt.Errorf("computing project breakdown: %w", err)
	}

	return stats, nil
}

// startOfWeek returns the Monday at 00:00:00 of the week containing t.
func startOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday → 7 in ISO week numbering
	}
	daysBack := weekday - 1 // Monday is day 1
	monday := t.AddDate(0, 0, -daysBack)
	return time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, t.Location())
}

// computeStreak returns (current, longest) streak lengths by examining
// completed_at dates. A streak is consecutive calendar days with >= 1
// completion, counted backward from today. If today has no completions but
// yesterday does, the streak is still active (user hasn't completed today yet).
func computeStreak(db *sql.DB, projectPath *string, now time.Time) (current int, longest int, err error) {
	query := `SELECT DISTINCT DATE(completed_at) FROM todos WHERE done = 1 AND completed_at IS NOT NULL`
	var args []any
	if projectPath != nil {
		query += ` AND project_path = ?`
		args = append(args, *projectPath)
	}
	query += ` ORDER BY DATE(completed_at) DESC`

	rows, err := db.Query(query, args...)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	var dates []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return 0, 0, err
		}
		dates = append(dates, d)
	}
	if err := rows.Err(); err != nil {
		return 0, 0, err
	}

	if len(dates) == 0 {
		return 0, 0, nil
	}

	utcNow := now.UTC()
	today := utcNow.Format("2006-01-02")
	yesterday := utcNow.AddDate(0, 0, -1).Format("2006-01-02")

	// Current streak: starting from the most recent completion date (which must
	// be today or yesterday for the streak to be active).
	activeStart := -1
	if dates[0] == today || dates[0] == yesterday {
		activeStart = 0
	}

	if activeStart >= 0 {
		current = 1
		for i := 1; i < len(dates); i++ {
			prev, _ := time.Parse("2006-01-02", dates[i-1])
			curr, _ := time.Parse("2006-01-02", dates[i])
			// Each step must be exactly one day back.
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

	longest = 1
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

	return current, longest, nil
}

// countCompletedSince returns the number of completed todos with completed_at >= since.
func countCompletedSince(db *sql.DB, projectPath *string, since time.Time) (int, error) {
	sinceStr := since.UTC().Format("2006-01-02 15:04:05")
	query := `SELECT COUNT(*) FROM todos WHERE done = 1 AND completed_at >= ?`
	args := []any{sinceStr}
	if projectPath != nil {
		query += ` AND project_path = ?`
		args = append(args, *projectPath)
	}
	var count int
	err := db.QueryRow(query, args...).Scan(&count)
	return count, err
}

// avgCloseTime returns the average duration between created_at and completed_at
// for all completed todos matching the optional project filter.
func avgCloseTime(db *sql.DB, projectPath *string) (time.Duration, error) {
	query := `SELECT COALESCE(AVG(julianday(completed_at) - julianday(created_at)), 0)
	          FROM todos WHERE done = 1 AND completed_at IS NOT NULL`
	var args []any
	if projectPath != nil {
		query += ` AND project_path = ?`
		args = append(args, *projectPath)
	}
	var days float64
	if err := db.QueryRow(query, args...).Scan(&days); err != nil {
		return 0, err
	}
	return time.Duration(days * float64(24*time.Hour)), nil
}

// totalFocusTime returns the total accumulated focus time from dig_sessions.
// Returns (0, false, nil) if the dig_sessions table does not exist.
// Returns (duration, true, nil) when focus data is present.
func totalFocusTime(db *sql.DB, projectPath *string) (time.Duration, bool, error) {
	// Check table existence first — graceful fallback for Phase 6.
	var tableCount int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='dig_sessions'`,
	).Scan(&tableCount); err != nil || tableCount == 0 {
		return 0, false, nil
	}

	var secs int64
	var err error
	if projectPath != nil {
		err = db.QueryRow(
			`SELECT COALESCE(SUM(ds.duration_secs), 0)
			 FROM dig_sessions ds
			 JOIN todos t ON ds.todo_id = t.id
			 WHERE t.project_path = ?`,
			*projectPath,
		).Scan(&secs)
	} else {
		err = db.QueryRow(
			`SELECT COALESCE(SUM(duration_secs), 0) FROM dig_sessions`,
		).Scan(&secs)
	}
	if err != nil {
		return 0, false, err
	}
	return time.Duration(secs) * time.Second, secs > 0, nil
}

// projectBreakdown returns per-project open/completed counts and average close time,
// grouped by project_path. Null project_path is shown as "(global)".
func projectBreakdown(db *sql.DB) ([]ProjectStats, error) {
	rows, err := db.Query(`
		SELECT
			project_path,
			SUM(CASE WHEN done = 0 THEN 1 ELSE 0 END) AS open,
			SUM(CASE WHEN done = 1 THEN 1 ELSE 0 END) AS completed,
			COALESCE(AVG(CASE WHEN done = 1 AND completed_at IS NOT NULL
				THEN julianday(completed_at) - julianday(created_at) END), 0) AS avg_days
		FROM todos
		GROUP BY project_path
		ORDER BY completed DESC, open DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []ProjectStats
	for rows.Next() {
		var ps ProjectStats
		var projPath sql.NullString
		var avgDays float64
		if err := rows.Scan(&projPath, &ps.Open, &ps.Completed, &avgDays); err != nil {
			return nil, err
		}
		if projPath.Valid && projPath.String != "" {
			ps.Name = filepath.Base(projPath.String)
		} else {
			ps.Name = "(global)"
		}
		ps.AvgClose = time.Duration(avgDays * float64(24*time.Hour))
		result = append(result, ps)
	}
	return result, rows.Err()
}
