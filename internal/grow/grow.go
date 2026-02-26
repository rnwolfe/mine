package grow

import (
	"database/sql"
	"fmt"
	"time"
)

// Goal represents a learning or career goal.
type Goal struct {
	ID           int
	Title        string
	Deadline     *time.Time
	TargetValue  float64
	CurrentValue float64
	Unit         string
	Done         bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Activity represents a logged learning activity.
type Activity struct {
	ID        int
	GoalID    *int
	Skill     string
	Note      string
	Minutes   int
	CreatedAt time.Time
}

// Skill represents a self-assessed skill level.
type Skill struct {
	ID        int
	Name      string
	Category  string
	Level     int
	UpdatedAt time.Time
}

// Store handles grow persistence.
type Store struct {
	db *sql.DB
}

// NewStore creates a new grow store.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// AddGoal creates a new learning goal and returns its ID.
func (s *Store) AddGoal(title string, deadline *time.Time, targetValue float64, unit string) (int, error) {
	var deadlineStr *string
	if deadline != nil {
		d := deadline.Format("2006-01-02")
		deadlineStr = &d
	}

	res, err := s.db.Exec(
		`INSERT INTO grow_goals (title, deadline, target_value, unit) VALUES (?, ?, ?, ?)`,
		title, deadlineStr, targetValue, unit,
	)
	if err != nil {
		return 0, fmt.Errorf("adding goal: %w", err)
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

// scanGoalRow scans a single goal from a sql.Row.
func scanGoalRow(row *sql.Row) (*Goal, error) {
	var g Goal
	var doneInt int
	var deadlineStr sql.NullString
	var createdStr, updatedStr string

	if err := row.Scan(&g.ID, &g.Title, &deadlineStr, &g.TargetValue, &g.CurrentValue, &g.Unit, &doneInt, &createdStr, &updatedStr); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	g.Done = doneInt == 1
	if deadlineStr.Valid && deadlineStr.String != "" {
		if t, parseErr := time.Parse("2006-01-02", deadlineStr.String); parseErr == nil {
			g.Deadline = &t
		}
	}
	g.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdStr)
	g.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedStr)
	return &g, nil
}

// GetGoal returns a single goal by ID.
func (s *Store) GetGoal(id int) (*Goal, error) {
	row := s.db.QueryRow(
		`SELECT id, title, deadline, target_value, current_value, unit, done, created_at, updated_at
		 FROM grow_goals WHERE id = ?`, id,
	)
	g, err := scanGoalRow(row)
	if err != nil {
		return nil, fmt.Errorf("getting goal #%d: %w", id, err)
	}
	if g == nil {
		return nil, fmt.Errorf("goal #%d not found", id)
	}
	return g, nil
}

// ListGoals returns all active (not done) goals.
func (s *Store) ListGoals() ([]Goal, error) {
	rows, err := s.db.Query(
		`SELECT id, title, deadline, target_value, current_value, unit, done, created_at, updated_at
		 FROM grow_goals WHERE done = 0 ORDER BY created_at ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanGoalRows(rows)
}

// DoneGoal marks a goal as complete.
func (s *Store) DoneGoal(id int) error {
	res, err := s.db.Exec(
		`UPDATE grow_goals SET done = 1, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND done = 0`,
		id,
	)
	if err != nil {
		return fmt.Errorf("completing goal: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("goal #%d not found or already done", id)
	}
	return nil
}

// LogActivity records a learning activity. Returns the new activity ID.
// If a goalID is provided, the goal's current_value is updated by summing
// all activity minutes for that goal.
func (s *Store) LogActivity(note string, minutes int, goalID *int, skill string) (int, error) {
	res, err := s.db.Exec(
		`INSERT INTO grow_activities (goal_id, skill, note, minutes) VALUES (?, ?, ?, ?)`,
		goalID, skill, note, minutes,
	)
	if err != nil {
		return 0, fmt.Errorf("logging activity: %w", err)
	}
	id, _ := res.LastInsertId()

	// Update goal current_value by summing activity minutes.
	if goalID != nil {
		if err := s.refreshGoalProgress(*goalID); err != nil {
			return int(id), fmt.Errorf("updating goal progress: %w", err)
		}
	}
	return int(id), nil
}

// refreshGoalProgress recalculates goal.current_value from summed activity minutes.
func (s *Store) refreshGoalProgress(goalID int) error {
	_, err := s.db.Exec(
		`UPDATE grow_goals
		 SET current_value = (
			 SELECT COALESCE(SUM(minutes), 0) FROM grow_activities WHERE goal_id = ?
		 ), updated_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		goalID, goalID,
	)
	return err
}

// ListActivities returns activities created on or after since (by date), ordered DESC.
// since must be in UTC to match the UTC dates stored by SQLite's CURRENT_TIMESTAMP.
func (s *Store) ListActivities(since time.Time) ([]Activity, error) {
	sinceStr := since.Format("2006-01-02")
	rows, err := s.db.Query(
		`SELECT id, goal_id, skill, note, minutes, created_at
		 FROM grow_activities WHERE DATE(created_at) >= ? ORDER BY created_at DESC`,
		sinceStr,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanActivityRows(rows)
}

// AllActivities returns all activities ordered by created_at DESC.
func (s *Store) AllActivities() ([]Activity, error) {
	rows, err := s.db.Query(
		`SELECT id, goal_id, skill, note, minutes, created_at
		 FROM grow_activities ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanActivityRows(rows)
}

// SetSkill upserts a skill level.
func (s *Store) SetSkill(name string, category string, level int) error {
	if level < 1 || level > 5 {
		return fmt.Errorf("level must be between 1 and 5, got %d", level)
	}
	_, err := s.db.Exec(
		`INSERT INTO grow_skills (name, category, level)
		 VALUES (?, ?, ?)
		 ON CONFLICT(name) DO UPDATE SET
		   level = excluded.level,
		   category = CASE WHEN excluded.category != '' THEN excluded.category ELSE grow_skills.category END,
		   updated_at = CURRENT_TIMESTAMP`,
		name, category, level,
	)
	if err != nil {
		return fmt.Errorf("setting skill: %w", err)
	}
	return nil
}

// ListSkills returns all skills ordered by name.
func (s *Store) ListSkills() ([]Skill, error) {
	rows, err := s.db.Query(
		`SELECT id, name, category, level, updated_at FROM grow_skills ORDER BY name ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var skills []Skill
	for rows.Next() {
		var sk Skill
		var updatedStr string
		if err := rows.Scan(&sk.ID, &sk.Name, &sk.Category, &sk.Level, &updatedStr); err != nil {
			return nil, err
		}
		sk.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedStr)
		skills = append(skills, sk)
	}
	return skills, rows.Err()
}

// ActivityDatesDesc returns all distinct DATE(created_at) values from grow_activities,
// ordered descending. Used for streak computation.
func (s *Store) ActivityDatesDesc() ([]string, error) {
	rows, err := s.db.Query(
		`SELECT DISTINCT DATE(created_at) FROM grow_activities ORDER BY DATE(created_at) DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dates []string
	for rows.Next() {
		var d string
		if err := rows.Scan(&d); err != nil {
			return nil, err
		}
		dates = append(dates, d)
	}
	return dates, rows.Err()
}

// SkillLevelDots renders a skill level as filled/empty dots, e.g. "●●●○○" for level 3.
func SkillLevelDots(level int) string {
	const filled = "●"
	const empty = "○"
	s := ""
	for i := 1; i <= 5; i++ {
		if i <= level {
			s += filled
		} else {
			s += empty
		}
	}
	return s
}

// scanGoalRows scans sql.Rows into a slice of Goal.
func scanGoalRows(rows *sql.Rows) ([]Goal, error) {
	var goals []Goal
	for rows.Next() {
		var g Goal
		var doneInt int
		var deadlineStr sql.NullString
		var createdStr, updatedStr string

		if err := rows.Scan(&g.ID, &g.Title, &deadlineStr, &g.TargetValue, &g.CurrentValue, &g.Unit, &doneInt, &createdStr, &updatedStr); err != nil {
			return nil, err
		}
		g.Done = doneInt == 1
		if deadlineStr.Valid && deadlineStr.String != "" {
			if t, parseErr := time.Parse("2006-01-02", deadlineStr.String); parseErr == nil {
				g.Deadline = &t
			}
		}
		g.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdStr)
		g.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedStr)
		goals = append(goals, g)
	}
	return goals, rows.Err()
}

// scanActivityRows scans sql.Rows into a slice of Activity.
func scanActivityRows(rows *sql.Rows) ([]Activity, error) {
	var activities []Activity
	for rows.Next() {
		var a Activity
		var goalID sql.NullInt64
		var skill sql.NullString
		var createdStr string

		if err := rows.Scan(&a.ID, &goalID, &skill, &a.Note, &a.Minutes, &createdStr); err != nil {
			return nil, err
		}
		if goalID.Valid {
			id := int(goalID.Int64)
			a.GoalID = &id
		}
		if skill.Valid {
			a.Skill = skill.String
		}
		a.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdStr)
		activities = append(activities, a)
	}
	return activities, rows.Err()
}
