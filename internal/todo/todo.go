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

// Todo represents a single task.
type Todo struct {
	ID          int
	Title       string
	Body        string
	Priority    int
	Done        bool
	DueDate     *time.Time
	Tags        []string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CompletedAt *time.Time
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

// Store handles todo persistence.
type Store struct {
	db *sql.DB
}

// NewStore creates a new todo store.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Add creates a new todo and returns its ID.
func (s *Store) Add(title string, priority int, tags []string, due *time.Time) (int, error) {
	tagStr := strings.Join(tags, ",")
	var dueStr *string
	if due != nil {
		d := due.Format("2006-01-02")
		dueStr = &d
	}

	res, err := s.db.Exec(
		`INSERT INTO todos (title, priority, tags, due_date) VALUES (?, ?, ?, ?)`,
		title, priority, tagStr, dueStr,
	)
	if err != nil {
		return 0, err
	}

	id, _ := res.LastInsertId()
	return int(id), nil
}

// Complete marks a todo as done.
func (s *Store) Complete(id int) error {
	res, err := s.db.Exec(
		`UPDATE todos SET done = 1, completed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ? AND done = 0`,
		id,
	)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("todo #%d not found or already done", id)
	}
	return nil
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

// List returns todos matching the filter.
func (s *Store) List(showDone bool) ([]Todo, error) {
	query := `SELECT id, title, body, priority, done, due_date, tags, created_at, updated_at, completed_at FROM todos`
	if !showDone {
		query += ` WHERE done = 0`
	}
	query += ` ORDER BY priority DESC, created_at ASC`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var todos []Todo
	for rows.Next() {
		var t Todo
		var doneInt int
		var dueStr, tagStr sql.NullString
		var completedAt sql.NullTime
		var createdStr, updatedStr string

		if err := rows.Scan(&t.ID, &t.Title, &t.Body, &t.Priority, &doneInt, &dueStr, &tagStr, &createdStr, &updatedStr, &completedAt); err != nil {
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
		if completedAt.Valid {
			t.CompletedAt = &completedAt.Time
		}
		t.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", createdStr)
		t.UpdatedAt, _ = time.Parse("2006-01-02 15:04:05", updatedStr)

		todos = append(todos, t)
	}
	return todos, nil
}

// Count returns the number of open and total todos.
func (s *Store) Count() (open int, total int, overdue int, err error) {
	err = s.db.QueryRow(`SELECT COUNT(*) FROM todos WHERE done = 0`).Scan(&open)
	if err != nil {
		return
	}
	err = s.db.QueryRow(`SELECT COUNT(*) FROM todos`).Scan(&total)
	if err != nil {
		return
	}
	today := time.Now().Format("2006-01-02")
	err = s.db.QueryRow(`SELECT COUNT(*) FROM todos WHERE done = 0 AND due_date IS NOT NULL AND due_date < ?`, today).Scan(&overdue)
	return
}

// Get returns a single todo by ID.
func (s *Store) Get(id int) (*Todo, error) {
	var t Todo
	var doneInt int
	var dueStr, tagStr sql.NullString
	var completedAt sql.NullTime
	var createdStr, updatedStr string

	err := s.db.QueryRow(
		`SELECT id, title, body, priority, done, due_date, tags, created_at, updated_at, completed_at FROM todos WHERE id = ?`,
		id,
	).Scan(&t.ID, &t.Title, &t.Body, &t.Priority, &doneInt, &dueStr, &tagStr, &createdStr, &updatedStr, &completedAt)
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
