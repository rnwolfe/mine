package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rnwolfe/mine/internal/todo"
	"github.com/rnwolfe/mine/internal/ui"
)

// TodoAction represents an action taken in the todo TUI.
type TodoAction struct {
	Type string // "toggle", "delete", "add", "quit"
	ID   int
	Text string
}

// TodoModel is a full interactive Bubbletea model for managing todos.
type TodoModel struct {
	todos    []todo.Todo
	cursor   int
	filter   string
	filtered []todo.Todo
	mode     todoMode

	// add mode state
	addInput string

	// terminal dimensions
	width  int
	height int

	// pending actions to apply after quitting
	Actions []TodoAction

	quitting bool
}

type todoMode int

const (
	todoModeNormal todoMode = iota
	todoModeFilter
	todoModeAdd
)

// NewTodoModel creates a new TodoModel with the given todos.
func NewTodoModel(todos []todo.Todo) *TodoModel {
	m := &TodoModel{
		todos:  todos,
		width:  80,
		height: 24,
	}
	m.applyFilter()
	return m
}

// RunTodo launches the interactive todo TUI. Returns actions for the caller to apply.
func RunTodo(todos []todo.Todo) ([]TodoAction, error) {
	m := NewTodoModel(todos)
	prog := tea.NewProgram(m, tea.WithAltScreen())
	result, err := prog.Run()
	if err != nil {
		return nil, fmt.Errorf("todo tui: %w", err)
	}
	final := result.(*TodoModel)
	return final.Actions, nil
}

func (m *TodoModel) Init() tea.Cmd {
	return nil
}

func (m *TodoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *TodoModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case todoModeFilter:
		return m.handleFilterKey(msg)
	case todoModeAdd:
		return m.handleAddKey(msg)
	default:
		return m.handleNormalKey(msg)
	}
}

func (m *TodoModel) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q", "esc":
		m.quitting = true
		return m, tea.Quit

	case "j", "down":
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		}

	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}

	case "g":
		m.cursor = 0

	case "G":
		if len(m.filtered) > 0 {
			m.cursor = len(m.filtered) - 1
		}

	case "x", " ", "enter":
		if len(m.filtered) > 0 {
			t := m.filtered[m.cursor]
			m.Actions = append(m.Actions, TodoAction{Type: "toggle", ID: t.ID})
			// Toggle locally for immediate feedback
			for i, item := range m.todos {
				if item.ID == t.ID {
					m.todos[i].Done = !m.todos[i].Done
					break
				}
			}
			m.applyFilter()
			if m.cursor >= len(m.filtered) && m.cursor > 0 {
				m.cursor = len(m.filtered) - 1
			}
		}

	case "d":
		if len(m.filtered) > 0 {
			t := m.filtered[m.cursor]
			m.Actions = append(m.Actions, TodoAction{Type: "delete", ID: t.ID})
			// Remove locally
			for i, item := range m.todos {
				if item.ID == t.ID {
					m.todos = append(m.todos[:i], m.todos[i+1:]...)
					break
				}
			}
			m.applyFilter()
			if m.cursor >= len(m.filtered) && m.cursor > 0 {
				m.cursor = len(m.filtered) - 1
			}
		}

	case "a":
		m.mode = todoModeAdd
		m.addInput = ""

	case "/":
		m.mode = todoModeFilter
		m.filter = ""
		m.applyFilter()
		m.cursor = 0
	}

	return m, nil
}

func (m *TodoModel) handleFilterKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = todoModeNormal
		m.filter = ""
		m.applyFilter()
		m.cursor = 0

	case "enter":
		m.mode = todoModeNormal

	case "backspace":
		if len(m.filter) > 0 {
			runes := []rune(m.filter)
			m.filter = string(runes[:len(runes)-1])
			m.applyFilter()
			m.cursor = 0
		}

	default:
		if len(msg.String()) == 1 {
			m.filter += msg.String()
			m.applyFilter()
			m.cursor = 0
		}
	}
	return m, nil
}

func (m *TodoModel) handleAddKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = todoModeNormal
		m.addInput = ""

	case "enter":
		text := strings.TrimSpace(m.addInput)
		if text != "" {
			m.Actions = append(m.Actions, TodoAction{Type: "add", Text: text})
			// Add a temporary entry locally so it shows immediately
			now := time.Now()
			m.todos = append(m.todos, todo.Todo{
				ID:        -len(m.Actions), // temp negative ID
				Title:     text,
				Priority:  todo.PrioMedium,
				CreatedAt: now,
				UpdatedAt: now,
			})
			m.applyFilter()
			if len(m.filtered) > 0 {
				m.cursor = len(m.filtered) - 1
			} else {
				m.cursor = 0
			}
		}
		m.mode = todoModeNormal
		m.addInput = ""

	case "backspace":
		if len(m.addInput) > 0 {
			runes := []rune(m.addInput)
			m.addInput = string(runes[:len(runes)-1])
		}

	default:
		// Accept printable characters
		if len(msg.Runes) > 0 {
			m.addInput += string(msg.Runes)
		}
	}
	return m, nil
}

func (m *TodoModel) applyFilter() {
	m.filtered = nil
	q := strings.ToLower(m.filter)
	for _, t := range m.todos {
		if q == "" {
			m.filtered = append(m.filtered, t)
			continue
		}
		if ok, _ := FuzzyMatch(q, t.Title); ok {
			m.filtered = append(m.filtered, t)
		}
	}
}

func (m *TodoModel) View() string {
	var b strings.Builder

	// Header
	header := ui.Title.Render("  " + ui.IconTodo + " Todo")
	if m.filter != "" {
		header += ui.Muted.Render(fmt.Sprintf("  filter: %q", m.filter))
	}
	b.WriteString(header + "\n\n")

	// Item list
	visHeight := m.height - 8 // reserve space for header, input, status bar
	if visHeight < 3 {
		visHeight = 3
	}

	// Calculate scroll offset
	offset := 0
	if m.cursor >= visHeight {
		offset = m.cursor - visHeight + 1
	}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	if len(m.filtered) == 0 {
		if m.filter != "" {
			b.WriteString("  " + ui.Muted.Render("No matches. Press esc to clear filter.") + "\n")
		} else {
			b.WriteString("  " + ui.Muted.Render("No todos. Press 'a' to add one.") + "\n")
		}
	} else {
		end := offset + visHeight
		if end > len(m.filtered) {
			end = len(m.filtered)
		}
		for i := offset; i < end; i++ {
			t := m.filtered[i]
			selected := i == m.cursor

			line := m.renderTodoItem(t, selected, today)
			b.WriteString(line + "\n")
		}
	}

	b.WriteString("\n")

	// Input area (filter or add mode)
	switch m.mode {
	case todoModeFilter:
		prompt := lipgloss.NewStyle().Foreground(ui.Gold).Bold(true).Render("/")
		b.WriteString("  " + prompt + " " + m.filter + blinkCursor() + "\n")
	case todoModeAdd:
		prompt := lipgloss.NewStyle().Foreground(ui.Emerald).Bold(true).Render("add:")
		b.WriteString("  " + prompt + " " + m.addInput + blinkCursor() + "\n")
	default:
		b.WriteString("\n")
	}

	b.WriteString("\n")

	// Status bar
	open := 0
	for _, t := range m.todos {
		if !t.Done {
			open++
		}
	}
	countStr := ui.Muted.Render(fmt.Sprintf("  %d/%d shown · %d open", len(m.filtered), len(m.todos), open))
	b.WriteString(countStr + "\n")

	// Help line
	var help string
	switch m.mode {
	case todoModeFilter:
		help = ui.Muted.Render("  esc clear · enter confirm")
	case todoModeAdd:
		help = ui.Muted.Render("  enter save · esc cancel")
	default:
		help = ui.Muted.Render("  j/k move · x toggle · a add · d delete · / filter · q quit")
	}
	b.WriteString(help + "\n")

	return b.String()
}

func (m *TodoModel) renderTodoItem(t todo.Todo, selected bool, today time.Time) string {
	pointer := "  "
	titleStyle := lipgloss.NewStyle()

	if selected {
		pointer = ui.Accent.Render(ui.IconArrow + " ")
		titleStyle = lipgloss.NewStyle().Foreground(ui.Gold).Bold(true)
	}

	// Done marker
	marker := " "
	if t.Done {
		marker = ui.Success.Render("✓")
	}

	id := ui.Muted.Render(fmt.Sprintf("#%-3d", t.ID))
	if t.ID < 0 {
		id = ui.Muted.Render("new ")
	}
	prio := todo.PriorityIcon(t.Priority)
	title := t.Title
	if t.Done {
		title = ui.Muted.Render(title)
	} else {
		title = titleStyle.Render(title)
	}

	line := fmt.Sprintf("  %s %s %s %s %s", pointer, marker, id, prio, title)

	// Due annotation
	if t.DueDate != nil && !t.Done {
		due := *t.DueDate
		dueDay := time.Date(due.Year(), due.Month(), due.Day(), 0, 0, 0, 0, due.Location())
		switch {
		case dueDay.Before(today):
			line += ui.Error.Render(fmt.Sprintf(" (overdue: %s)", due.Format("Jan 2")))
		case dueDay.Equal(today):
			line += ui.Warning.Render(" (due today!)")
		case dueDay.Before(today.AddDate(0, 0, 7)):
			line += ui.Muted.Render(fmt.Sprintf(" (due %s)", due.Format("Mon")))
		default:
			line += ui.Muted.Render(fmt.Sprintf(" (due %s)", due.Format("Jan 2")))
		}
	}

	// Tags
	if len(t.Tags) > 0 {
		line += ui.Muted.Render(" [" + strings.Join(t.Tags, ", ") + "]")
	}

	return line
}
