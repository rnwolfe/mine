package dig

import (
	"database/sql"
	"fmt"
	"time"
)

// Stats holds aggregate dig session statistics.
type Stats struct {
	CurrentStreak int
	LongestStreak int
	TotalMins     int
	LastDate      string
	SessionCount  int
	LinkedTasks   int
}

// Store provides persistence for dig sessions.
type Store struct {
	db *sql.DB
}

// NewStore creates a new Store backed by db.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// RecordSession inserts a dig session, updates the streak, and increments total minutes.
// Returns (totalMins, nil) on success. Streak updates proceed even if the
// session insert fails (non-atomic by design, matching original behavior).
func (s *Store) RecordSession(duration time.Duration, todoID *int, completed bool, startedAt time.Time) (int, error) {
	mins := int(duration.Minutes())
	secs := int(duration.Seconds())
	today := time.Now().Format("2006-01-02")
	comp := 0
	if completed {
		comp = 1
	}

	var sessionErr error
	if _, err := s.db.Exec(
		`INSERT INTO dig_sessions (todo_id, duration_secs, completed, started_at, ended_at) VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)`,
		todoID, secs, comp, startedAt.UTC().Format("2006-01-02 15:04:05"),
	); err != nil {
		sessionErr = fmt.Errorf("recording session: %w", err)
		// fall through — streak and KV updates are independent of session recording
	}

	// Update streak (non-fatal).
	_ = s.UpdateStreak(today)

	// Update total minutes in KV.
	var total int
	if err := s.db.QueryRow(`SELECT CAST(value AS INTEGER) FROM kv WHERE key = 'dig_total_mins'`).Scan(&total); err != nil {
		if err != sql.ErrNoRows {
			if sessionErr != nil {
				return total, fmt.Errorf("recording session (non-fatal error: %v); reading total minutes: %w", sessionErr, err)
			}
			return total, fmt.Errorf("reading total minutes: %w", err)
		}
		// sql.ErrNoRows — no existing total, start from zero.
		total = 0
	}

	total += mins
	if _, err := s.db.Exec(
		`INSERT OR REPLACE INTO kv (key, value, updated_at) VALUES ('dig_total_mins', ?, CURRENT_TIMESTAMP)`,
		fmt.Sprintf("%d", total),
	); err != nil {
		if sessionErr != nil {
			return total, fmt.Errorf("recording session (non-fatal error: %v); updating total minutes: %w", sessionErr, err)
		}
		return total, fmt.Errorf("updating total minutes: %w", err)
	}

	return total, sessionErr
}

// UpdateStreak updates the dig streak row for the given date string (YYYY-MM-DD).
// On the first session ever, it inserts a new streak row with current=1, longest=1.
// On consecutive days it increments current (and longest if needed).
// On a broken streak it resets current to 1.
func (s *Store) UpdateStreak(today string) error {
	var lastDate string
	var current, longest int
	err := s.db.QueryRow(`SELECT last_date, current, longest FROM streaks WHERE name = 'dig'`).Scan(&lastDate, &current, &longest)
	if err != nil {
		if err == sql.ErrNoRows {
			// First session ever.
			_, err = s.db.Exec(`INSERT INTO streaks (name, current, longest, last_date) VALUES ('dig', 1, 1, ?)`, today)
			return err
		}
		return fmt.Errorf("select streak row: %w", err)
	}

	if lastDate == today {
		return nil // Already logged today; don't increment streak.
	}

	todayTime, err := time.Parse("2006-01-02", today)
	if err != nil {
		return fmt.Errorf("invalid today date %q: %w", today, err)
	}
	yesterday := todayTime.AddDate(0, 0, -1).Format("2006-01-02")

	if lastDate == yesterday {
		current++
		if current > longest {
			longest = current
		}
		_, err = s.db.Exec(`UPDATE streaks SET current = ?, longest = ?, last_date = ? WHERE name = 'dig'`, current, longest, today)
	} else {
		// Streak broken — reset current but preserve longest.
		_, err = s.db.Exec(`UPDATE streaks SET current = 1, last_date = ? WHERE name = 'dig'`, today)
	}
	return err
}

// GetStats returns aggregate statistics for dig sessions.
// Always returns a non-nil *Stats. When no sessions exist, all fields are zero and err is nil.
// On partial DB failures, returns whatever stats were computed along with the error.
func (s *Store) GetStats() (*Stats, error) {
	stats := &Stats{}
	err := s.db.QueryRow(
		`SELECT current, longest, last_date FROM streaks WHERE name = 'dig'`,
	).Scan(&stats.CurrentStreak, &stats.LongestStreak, &stats.LastDate)
	if err != nil {
		if err == sql.ErrNoRows {
			return stats, nil // no sessions recorded yet; zero values are correct
		}
		return stats, err // return partial (zero) stats with error
	}

	err = s.db.QueryRow(`SELECT CAST(value AS INTEGER) FROM kv WHERE key = 'dig_total_mins'`).Scan(&stats.TotalMins)
	if err != nil {
		if err == sql.ErrNoRows {
			stats.TotalMins = 0
		} else {
			return stats, fmt.Errorf("query dig_total_mins: %w", err)
		}
	}

	err = s.db.QueryRow(`SELECT COUNT(*) FROM dig_sessions`).Scan(&stats.SessionCount)
	if err != nil {
		return stats, fmt.Errorf("count dig_sessions: %w", err)
	}

	if stats.SessionCount > 0 {
		err = s.db.QueryRow(`SELECT COUNT(DISTINCT todo_id) FROM dig_sessions WHERE todo_id IS NOT NULL`).Scan(&stats.LinkedTasks)
		if err != nil {
			return stats, fmt.Errorf("count linked dig_sessions tasks: %w", err)
		}
	}

	return stats, nil
}
