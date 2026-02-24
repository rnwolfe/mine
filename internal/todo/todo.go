package todo

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Priority levels.
const (
	PrioLow    = 1
	PrioMedium = 2
	PrioHigh   = 3
	PrioCrit   = 4
)

// Valid schedule bucket values.
const (
	ScheduleToday   = "today"
	ScheduleSoon    = "soon"
	ScheduleLater   = "later"
	ScheduleSomeday = "someday"
)

// Valid recurrence values.
const (
	RecurrenceNone    = "none"
	RecurrenceDaily   = "daily"
	RecurrenceWeekday = "weekday"
	RecurrenceWeekly  = "weekly"
	RecurrenceMonthly = "monthly"
)

// Note represents a timestamped annotation on a todo.
type Note struct {
	ID        int
	Body      string
	CreatedAt time.Time
}

// Todo represents a single task.
type Todo struct {
	ID          int
	Title       string
	Body        string
	Priority    int
	Done        bool
	DueDate     *time.Time
	Tags        []string
	ProjectPath *string
	Schedule    string
	Recurrence  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time
	// Notes is populated only by GetWithNotes(), not List(), for performance.
	Notes []Note
}

// SortMode controls the sort order returned by List.
type SortMode int

const (
	// SortUrgency sorts by computed urgency score (default).
	SortUrgency SortMode = iota
	// SortLegacy sorts by priority DESC, created_at ASC (original behavior).
	SortLegacy
)

// ListOptions configures which todos to return from List.
type ListOptions struct {
	// ShowDone includes completed todos in the result.
	ShowDone bool
	// IncludeSomeday includes todos with schedule='someday' (hidden by default).
	IncludeSomeday bool
	// ProjectPath filters to a specific project plus global todos.
	// nil means global-only (no project filter).
	// Set AllProjects = true to ignore this field and return everything.
	ProjectPath *string
	// AllProjects returns todos from all projects and global.
	AllProjects bool
	// Sort controls the sort order. Default (zero value) is SortUrgency.
	Sort SortMode
	// CurrentProjectPath is the active project for urgency scoring.
	// Used only when Sort == SortUrgency.
	CurrentProjectPath *string
	// Weights overrides default urgency weights.
	// nil means use DefaultUrgencyWeights. An explicit all-zero *UrgencyWeights
	// is used as-is, allowing callers to fully disable all scoring factors.
	// Used only when Sort == SortUrgency.
	Weights *UrgencyWeights
	// ReferenceTime is the "now" used for urgency scoring and due-date checks.
	// Zero value means time.Now() is called at sort time. Set this explicitly
	// when you need sorting and rendering to use the same instant (e.g. around midnight).
	// Used only when Sort == SortUrgency.
	ReferenceTime time.Time
}

// PriorityLabel returns a human-readable priority string.
func PriorityLabel(p int) string {
	switch p {
	case PrioCrit:
		return "crit"
	case PrioHigh:
		return "high"
	case PrioMedium:
		return "med"
	case PrioLow:
		return "low"
	default:
		return "?"
	}
}

// PriorityIcon returns a colored icon for the priority.
func PriorityIcon(p int) string {
	switch p {
	case PrioCrit:
		return "🔴"
	case PrioHigh:
		return "🟠"
	case PrioMedium:
		return "🟡"
	case PrioLow:
		return "🟢"
	default:
		return "⚪"
	}
}

// ParseSchedule validates and normalizes a schedule bucket string.
// Accepts full names and short aliases: t=today, s=soon, l=later, sd=someday.
func ParseSchedule(s string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "today", "t":
		return ScheduleToday, nil
	case "soon", "s":
		return ScheduleSoon, nil
	case "later", "l":
		return ScheduleLater, nil
	case "someday", "sd":
		return ScheduleSomeday, nil
	default:
		return "", fmt.Errorf("invalid schedule %q — valid values: today (t), soon (s), later (l), someday (sd)", s)
	}
}

// ParseRecurrence validates and normalizes a recurrence string.
// Accepts short aliases: d/day/daily, wd/weekday, w/week/weekly, m/month/monthly.
func ParseRecurrence(s string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "d", "day", "daily":
		return RecurrenceDaily, nil
	case "wd", "weekday":
		return RecurrenceWeekday, nil
	case "w", "week", "weekly":
		return RecurrenceWeekly, nil
	case "m", "month", "monthly":
		return RecurrenceMonthly, nil
	default:
		return "", fmt.Errorf("invalid recurrence %q — valid values: day (d), weekday (wd), week (w), month (m)", s)
	}
}

// RecurrenceLabel returns a short display label for a recurrence value.
func RecurrenceLabel(r string) string {
	switch r {
	case RecurrenceDaily:
		return "daily"
	case RecurrenceWeekday:
		return "weekday"
	case RecurrenceWeekly:
		return "weekly"
	case RecurrenceMonthly:
		return "monthly"
	default:
		return ""
	}
}

// nextDueDate computes the next due date based on the recurrence frequency.
// base is the current due date (or today if no due date was set).
func nextDueDate(base time.Time, recurrence string) time.Time {
	switch recurrence {
	case RecurrenceDaily:
		return base.AddDate(0, 0, 1)
	case RecurrenceWeekday:
		next := base.AddDate(0, 0, 1)
		for next.Weekday() == time.Saturday || next.Weekday() == time.Sunday {
			next = next.AddDate(0, 0, 1)
		}
		return next
	case RecurrenceWeekly:
		return base.AddDate(0, 0, 7)
	case RecurrenceMonthly:
		// Clamp to end of next month if day exceeds it.
		y, m, d := base.Date()
		nextMonth := time.Date(y, m+1, 1, 0, 0, 0, 0, base.Location())
		// Last day of next month.
		lastDay := time.Date(nextMonth.Year(), nextMonth.Month()+1, 0, 0, 0, 0, 0, base.Location()).Day()
		if d > lastDay {
			d = lastDay
		}
		return time.Date(nextMonth.Year(), nextMonth.Month(), d, 0, 0, 0, 0, base.Location())
	default:
		return base
	}
}

// ScheduleLabel returns a short display label for a schedule bucket.
func ScheduleLabel(schedule string) string {
	switch schedule {
	case ScheduleToday:
		return "today"
	case ScheduleSoon:
		return "soon"
	case ScheduleLater:
		return "later"
	case ScheduleSomeday:
		return "someday"
	default:
		return "later"
	}
}

// Store handles todo persistence.
type Store struct {
	db *sql.DB
}

// NewStore creates a new todo store.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Add creates a new todo and returns its ID.
// body sets the initial description/context for the todo (may be empty).
// recurrence is one of the Recurrence* constants (or "" / "none" for non-recurring).
func (s *Store) Add(title string, body string, priority int, tags []string, due *time.Time, projectPath *string, schedule string, recurrence string) (int, error) {
	tagStr := strings.Join(tags, ",")
	var dueStr *string
	if due != nil {
		d := due.Format("2006-01-02")
		dueStr = &d
	}
	if schedule == "" {
		schedule = ScheduleLater
	}
	if recurrence == "" {
		recurrence = RecurrenceNone
	}

	res, err := s.db.Exec(
		`INSERT INTO todos (title, body, priority, tags, due_date, project_path, schedule, recurrence) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		title, body, priority, tagStr, dueStr, projectPath, schedule, recurrence,
	)
	if err != nil {
		return 0, err
	}

	id, _ := res.LastInsertId()
	return int(id), nil
}

// SetSchedule updates the schedule bucket for a todo.
func (s *Store) SetSchedule(id int, schedule string) error {
	if _, err := ParseSchedule(schedule); err != nil {
		return err
	}
	res, err := s.db.Exec(
		`UPDATE todos SET schedule = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		schedule, id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("todo #%d not found", id)
	}
	return nil
}

// Complete marks a todo as done. For recurring tasks it also spawns the next occurrence.
// Returns (spawnedID, spawnedDue, err) where spawnedID > 0 if a new occurrence was created.
func (s *Store) Complete(id int) (spawnedID int, spawnedDue *time.Time, err error) {
	// Fetch the todo before completing so we have recurrence/due info.
	t, err := s.Get(id)
	if err != nil {
		return 0, nil, err
	}

	res, execErr := s.db.Exec(
		`UPDATE todos SET done = 1, completed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND done = 0`,
		id,
	)
	if execErr != nil {
		return 0, nil, execErr
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return 0, nil, fmt.Errorf("todo #%d not found or already done", id)
	}

	// Spawn next occurrence for recurring tasks.
	if t.Recurrence != "" && t.Recurrence != RecurrenceNone {
		base := time.Now()
		if t.DueDate != nil {
			base = *t.DueDate
		}
		// Strip time component — only keep date.
		base = time.Date(base.Year(), base.Month(), base.Day(), 0, 0, 0, 0, base.Location())
		next := nextDueDate(base, t.Recurrence)
		spawnedID, err = s.Add(t.Title, t.Body, t.Priority, t.Tags, &next, t.ProjectPath, ScheduleToday, t.Recurrence)
		if err != nil {
			return 0, nil, fmt.Errorf("spawning next occurrence: %w", err)
		}
		return spawnedID, &next, nil
	}

	return 0, nil, nil
}

// Uncomplete marks a todo as not done.
func (s *Store) Uncomplete(id int) error {
	_, err := s.db.Exec(
		`UPDATE todos SET done = 0, completed_at = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		id,
	)
	return err
}

// Delete removes a todo.
func (s *Store) Delete(id int) error {
	res, err := s.db.Exec(`DELETE FROM todos WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("todo #%d not found", id)
	}
	return nil
}

// List returns todos matching the given options.
func (s *Store) List(opts ListOptions) ([]Todo, error) {
	query := `SELECT id, title, body, priority, done, due_date, tags, project_path, schedule, recurrence, created_at, updated_at, completed_at FROM todos`

	var conditions []string
	var args []any

	if !opts.ShowDone {
		conditions = append(conditions, "done = 0")
	}

	if !opts.IncludeSomeday {
		conditions = append(conditions, "COALESCE(schedule, 'later') != 'someday'")
	}

	if !opts.AllProjects {
		if opts.ProjectPath != nil {
			// Show this project's todos plus global (null project_path) todos.
			conditions = append(conditions, "(project_path = ? OR project_path IS NULL)")
			args = append(args, *opts.ProjectPath)
		} else {
			// Outside any project: show only global todos.
			conditions = append(conditions, "project_path IS NULL")
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	// Legacy sort happens in SQL; urgency sort happens in Go after fetch.
	if opts.Sort == SortLegacy {
		query += " ORDER BY priority DESC, created_at ASC"
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		var t Todo
		var doneInt int
		var dueStr, tagStr, projPath, scheduleStr, recurrenceStr sql.NullString
		var completedAt sql.NullTime
		var createdStr, updatedStr string

		if err := rows.Scan(&t.ID, &t.Title, &t.Body, &t.Priority, &doneInt, &dueStr, &tagStr, &projPath, &scheduleStr, &recurrenceStr, &createdStr, &updatedStr, &completedAt); err != nil {
			return nil, err
		}

		t.Done = doneInt == 1
		if dueStr.Valid && dueStr.String != "" {
			if parsed, err := time.Parse("2006-01-02", dueStr.String); err == nil {
				t.DueDate = &parsed
			}
		}
		if tagStr.Valid && tagStr.String != "" {
			t.Tags = strings.Split(tagStr.String, ",")
		}
		if projPath.Valid && projPath.String != "" {
			s := projPath.String
			t.ProjectPath = &s
		}
		if scheduleStr.Valid && scheduleStr.String != "" {
			t.Schedule = scheduleStr.String
		} else {
			t.Schedule = ScheduleLater
		}
		if recurrenceStr.Valid && recurrenceStr.String != "" {
			t.Recurrence = recurrenceStr.String
		} else {
			t.Recurrence = RecurrenceNone
		}
		if completedAt.Valid {
			t.CompletedAt = &completedAt.Time
		}
		t.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdStr)
		t.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedStr)

		todos = append(todos, t)
	}

	// Apply urgency sort (default) in Go after fetching.
	if opts.Sort == SortUrgency {
		w := opts.Weights
		if w == nil {
			def := DefaultUrgencyWeights()
			w = &def
		}
		ref := opts.ReferenceTime
		if ref.IsZero() {
			ref = time.Now()
		}
		SortByUrgency(todos, ref, opts.CurrentProjectPath, *w)
	}

	return todos, nil
}

// Count returns the number of open and total todos, optionally scoped to a project.
// projectPath nil returns counts across all todos (no project filter).
// projectPath non-nil scopes to that project plus global (null project_path) todos.
func (s *Store) Count(projectPath *string) (open int, total int, overdue int, err error) {
	today := time.Now().Format("2006-01-02")

	if projectPath != nil {
		p := *projectPath
		err = s.db.QueryRow(
			`SELECT COUNT(*) FROM todos WHERE done = 0 AND (project_path = ? OR project_path IS NULL)`, p,
		).Scan(&open)
		if err != nil {
			return
		}
		err = s.db.QueryRow(
			`SELECT COUNT(*) FROM todos WHERE project_path = ? OR project_path IS NULL`, p,
		).Scan(&total)
		if err != nil {
			return
		}
		err = s.db.QueryRow(
			`SELECT COUNT(*) FROM todos WHERE done = 0 AND due_date IS NOT NULL AND due_date < ? AND (project_path = ? OR project_path IS NULL)`,
			today, p,
		).Scan(&overdue)
		return
	}

	err = s.db.QueryRow(`SELECT COUNT(*) FROM todos WHERE done = 0`).Scan(&open)
	if err != nil {
		return
	}
	err = s.db.QueryRow(`SELECT COUNT(*) FROM todos`).Scan(&total)
	if err != nil {
		return
	}
	err = s.db.QueryRow(`SELECT COUNT(*) FROM todos WHERE done = 0 AND due_date IS NOT NULL AND due_date < ?`, today).Scan(&overdue)
	return
}

// Get returns a single todo by ID.
func (s *Store) Get(id int) (*Todo, error) {
	var t Todo
	var doneInt int
	var dueStr, tagStr, projPath, scheduleStr, recurrenceStr sql.NullString
	var completedAt sql.NullTime
	var createdStr, updatedStr string

	err := s.db.QueryRow(
		`SELECT id, title, body, priority, done, due_date, tags, project_path, schedule, recurrence, created_at, updated_at, completed_at FROM todos WHERE id = ?`,
		id,
	).Scan(&t.ID, &t.Title, &t.Body, &t.Priority, &doneInt, &dueStr, &tagStr, &projPath, &scheduleStr, &recurrenceStr, &createdStr, &updatedStr, &completedAt)
	if err != nil {
		return nil, fmt.Errorf("todo #%d not found", id)
	}

	t.Done = doneInt == 1
	if dueStr.Valid && dueStr.String != "" {
		if parsed, err := time.Parse("2006-01-02", dueStr.String); err == nil {
			t.DueDate = &parsed
		}
	}
	if tagStr.Valid && tagStr.String != "" {
		t.Tags = strings.Split(tagStr.String, ",")
	}
	if projPath.Valid && projPath.String != "" {
		s := projPath.String
		t.ProjectPath = &s
	}
	if scheduleStr.Valid && scheduleStr.String != "" {
		t.Schedule = scheduleStr.String
	} else {
		t.Schedule = ScheduleLater
	}
	if recurrenceStr.Valid && recurrenceStr.String != "" {
		t.Recurrence = recurrenceStr.String
	} else {
		t.Recurrence = RecurrenceNone
	}
	if completedAt.Valid {
		t.CompletedAt = &completedAt.Time
	}
	t.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdStr)
	t.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedStr)

	return &t, nil
}

// Edit updates a todo's title and/or priority.
func (s *Store) Edit(id int, title *string, priority *int) error {
	sets := []string{}
	args := []any{}

	if title != nil {
		sets = append(sets, "title = ?")
		args = append(args, *title)
	}
	if priority != nil {
		sets = append(sets, "priority = ?")
		args = append(args, *priority)
	}
	if len(sets) == 0 {
		return nil
	}

	sets = append(sets, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, id)

	query := fmt.Sprintf("UPDATE todos SET %s WHERE id = ?", strings.Join(sets, ", "))
	_, err := s.db.Exec(query, args...)
	return err
}

// AddNote appends a timestamped annotation to an existing todo.
// Returns an error if the todo does not exist.
// Updates the parent todo's updated_at in the same transaction.
func (s *Store) AddNote(todoID int, body string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}

	var exists int
	if err = tx.QueryRow(`SELECT COUNT(*) FROM todos WHERE id = ?`, todoID).Scan(&exists); err != nil {
		tx.Rollback()
		return err
	}
	if exists == 0 {
		tx.Rollback()
		return fmt.Errorf("todo #%d not found", todoID)
	}

	if _, err = tx.Exec(`INSERT INTO todo_notes (todo_id, body) VALUES (?, ?)`, todoID, body); err != nil {
		tx.Rollback()
		return err
	}

	if _, err = tx.Exec(`UPDATE todos SET updated_at = CURRENT_TIMESTAMP WHERE id = ?`, todoID); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}

// FocusTime returns the total accumulated focus time for a given todo from dig_sessions.
// Returns 0 if there are no sessions or the todo has no linked sessions.
func (s *Store) FocusTime(todoID int) (time.Duration, error) {
	var secs int64
	err := s.db.QueryRow(
		`SELECT COALESCE(SUM(duration_secs), 0) FROM dig_sessions WHERE todo_id = ?`,
		todoID,
	).Scan(&secs)
	if err != nil {
		return 0, fmt.Errorf("fetching focus time for todo #%d: %w", todoID, err)
	}
	return time.Duration(secs) * time.Second, nil
}

// FocusTimeMap returns a map of todo ID to total accumulated focus time for all given IDs.
// Only IDs with at least one dig session are included in the result.
func (s *Store) FocusTimeMap(todoIDs []int) (map[int]time.Duration, error) {
	if len(todoIDs) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(todoIDs))
	args := make([]any, len(todoIDs))
	for i, id := range todoIDs {
		placeholders[i] = "?"
		args[i] = id
	}
	query := fmt.Sprintf(
		`SELECT todo_id, COALESCE(SUM(duration_secs), 0) FROM dig_sessions WHERE todo_id IN (%s) GROUP BY todo_id`,
		strings.Join(placeholders, ","),
	)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("fetching focus time map: %w", err)
	}
	defer rows.Close()

	result := make(map[int]time.Duration)
	for rows.Next() {
		var id int
		var secs int64
		if err := rows.Scan(&id, &secs); err != nil {
			return nil, err
		}
		if secs > 0 {
			result[id] = time.Duration(secs) * time.Second
		}
	}
	return result, rows.Err()
}

// GetWithNotes returns a todo by ID with all its timestamped notes populated,
// ordered by created_at ASC. Notes are not populated by Get() or List().
func (s *Store) GetWithNotes(id int) (*Todo, error) {
	t, err := s.Get(id)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(
		`SELECT id, body, created_at FROM todo_notes WHERE todo_id = ? ORDER BY created_at ASC, id ASC`,
		id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var n Note
		var createdStr string
		if err := rows.Scan(&n.ID, &n.Body, &createdStr); err != nil {
			return nil, err
		}
		n.CreatedAt, err = time.Parse(time.RFC3339, createdStr)
		if err != nil {
			// Fallback for SQLite-native "YYYY-MM-DD HH:MM:SS" format.
			n.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdStr)
			if err != nil {
				return nil, fmt.Errorf("parsing note created_at %q: %w", createdStr, err)
			}
		}
		t.Notes = append(t.Notes, n)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return t, nil
}

// ListRecurring returns all open todos that have a recurrence set (i.e. recurrence != 'none').
func (s *Store) ListRecurring() ([]Todo, error) {
	rows, err := s.db.Query(
		`SELECT id, title, body, priority, done, due_date, tags, project_path, schedule, recurrence, created_at, updated_at, completed_at
		 FROM todos WHERE done = 0 AND recurrence IS NOT NULL AND recurrence != 'none'
		 ORDER BY created_at ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		var t Todo
		var doneInt int
		var dueStr, tagStr, projPath, scheduleStr, recurrenceStr sql.NullString
		var completedAt sql.NullTime
		var createdStr, updatedStr string

		if err := rows.Scan(&t.ID, &t.Title, &t.Body, &t.Priority, &doneInt, &dueStr, &tagStr, &projPath, &scheduleStr, &recurrenceStr, &createdStr, &updatedStr, &completedAt); err != nil {
			return nil, err
		}

		t.Done = doneInt == 1
		if dueStr.Valid && dueStr.String != "" {
			if parsed, err := time.Parse("2006-01-02", dueStr.String); err == nil {
				t.DueDate = &parsed
			}
		}
		if tagStr.Valid && tagStr.String != "" {
			t.Tags = strings.Split(tagStr.String, ",")
		}
		if projPath.Valid && projPath.String != "" {
			p := projPath.String
			t.ProjectPath = &p
		}
		if scheduleStr.Valid && scheduleStr.String != "" {
			t.Schedule = scheduleStr.String
		} else {
			t.Schedule = ScheduleLater
		}
		if recurrenceStr.Valid && recurrenceStr.String != "" {
			t.Recurrence = recurrenceStr.String
		} else {
			t.Recurrence = RecurrenceNone
		}
		if completedAt.Valid {
			t.CompletedAt = &completedAt.Time
		}
		t.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdStr)
		t.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedStr)

		todos = append(todos, t)
	}

	return todos, rows.Err()
}

// DemoteProject sets project_path = NULL for all open todos that match the given project path.
// Returns the number of affected rows.
func (s *Store) DemoteProject(projectPath string) (int, error) {
	res, err := s.db.Exec(
		`UPDATE todos SET project_path = NULL, updated_at = CURRENT_TIMESTAMP WHERE project_path = ? AND done = 0`,
		projectPath,
	)
	if err != nil {
		return 0, fmt.Errorf("demoting project todos: %w", err)
	}
	n, _ := res.RowsAffected()
	return int(n), nil
}
