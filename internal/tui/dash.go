package tui

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rnwolfe/mine/internal/proj"
	"github.com/rnwolfe/mine/internal/todo"
	"github.com/rnwolfe/mine/internal/ui"
)

// DashAction indicates what action triggered the dashboard exit.
type DashAction int

const (
	// DashActionQuit means the user pressed q or ctrl+c.
	DashActionQuit DashAction = iota
	// DashActionOpenTodo means the user pressed t to open the todo TUI.
	DashActionOpenTodo
	// DashActionStartDig means the user pressed d to start a dig session.
	DashActionStartDig
)

// DashData holds all loaded panel data for the dashboard.
type DashData struct {
	Todos        []todo.Todo
	TodoOpen     int
	TodoOverdue  int
	Streak       int
	WeekDone     int
	TotalFocus   time.Duration
	HasFocusData bool
	Project      *proj.Project
}

type dashDataMsg DashData
type dashErrMsg struct{ err error }

// DashModel is the Bubbletea model for the mine dashboard.
type DashModel struct {
	data    DashData
	db      *sql.DB
	ps      *proj.Store
	width   int
	height  int
	loading bool
	err     error
	action  DashAction
}

// NewDashModel creates a new DashModel connected to the given DB.
func NewDashModel(db *sql.DB) *DashModel {
	return &DashModel{
		db:      db,
		ps:      proj.NewStore(db),
		width:   80,
		height:  24,
		loading: true,
	}
}

// RunDash runs the dashboard TUI once and returns the exit action.
// The caller is responsible for the outer loop (re-launching after todo TUI or dig).
func RunDash(db *sql.DB) (DashAction, error) {
	m := NewDashModel(db)
	prog := tea.NewProgram(m, tea.WithAltScreen())
	result, err := prog.Run()
	if err != nil {
		return DashActionQuit, fmt.Errorf("dashboard: %w", err)
	}
	final := result.(*DashModel)
	return final.action, nil
}

// --- Bubbletea model interface ---

func (m *DashModel) Init() tea.Cmd {
	return m.loadData()
}

func (m *DashModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case dashDataMsg:
		m.data = DashData(msg)
		m.loading = false
		m.err = nil
		return m, nil

	case dashErrMsg:
		m.err = msg.err
		m.loading = false
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *DashModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.action = DashActionQuit
		return m, tea.Quit
	case "t":
		if !m.loading {
			m.action = DashActionOpenTodo
			return m, tea.Quit
		}
	case "d":
		if !m.loading {
			m.action = DashActionStartDig
			return m, tea.Quit
		}
	case "r":
		m.loading = true
		return m, m.loadData()
	}
	return m, nil
}

func (m *DashModel) View() string {
	if m.loading {
		return "\n  " + ui.Muted.Render("Loading…") + "\n"
	}
	if m.err != nil {
		return "\n  " + ui.Error.Render("Error: "+m.err.Error()) + "\n"
	}
	switch {
	case m.width < 60:
		return m.renderMinimal()
	case m.width >= 120:
		return m.renderTwoColumn()
	default:
		return m.renderStacked()
	}
}

// --- Layout builders ---

func (m *DashModel) renderTwoColumn() string {
	leftW := m.width/2 - 2
	rightW := m.width - leftW - 4

	left := lipgloss.NewStyle().Width(leftW).Render(
		renderTodosPanel(m.data.Todos, m.data.TodoOpen, m.data.TodoOverdue, leftW),
	)
	right := lipgloss.NewStyle().Width(rightW).Render(
		lipgloss.JoinVertical(lipgloss.Left,
			renderFocusPanel(m.data, rightW),
			"",
			renderProjectPanel(m.data.Project, m.data.TodoOpen, rightW),
		),
	)

	cols := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	return cols + "\n\n" + renderHelpBar() + "\n"
}

func (m *DashModel) renderStacked() string {
	w := m.width - 4
	parts := []string{
		renderTodosPanel(m.data.Todos, m.data.TodoOpen, m.data.TodoOverdue, w),
		"",
		renderFocusPanel(m.data, w),
	}
	if m.data.Project != nil {
		parts = append(parts, "", renderProjectPanel(m.data.Project, m.data.TodoOpen, w))
	}
	return lipgloss.JoinVertical(lipgloss.Left, parts...) + "\n\n" + renderHelpBar() + "\n"
}

func (m *DashModel) renderMinimal() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString("  " + ui.Title.Render(ui.IconMine+"mine") + "\n\n")
	b.WriteString(fmt.Sprintf("  %s %d open\n", ui.IconTodo, m.data.TodoOpen))
	if m.data.TodoOverdue > 0 {
		b.WriteString("  " + ui.Error.Render(fmt.Sprintf("%d overdue", m.data.TodoOverdue)) + "\n")
	}
	if m.data.Streak > 0 {
		b.WriteString(fmt.Sprintf("  %s streak: %d days\n", ui.IconFire, m.data.Streak))
	}
	if m.data.Project != nil {
		b.WriteString(fmt.Sprintf("  %s %s\n", ui.IconProject, m.data.Project.Name))
	}
	b.WriteString("\n  " + ui.Muted.Render("q quit · t todos · d dig · r refresh") + "\n")
	return b.String()
}

// --- Panel renderers (pure functions — no model state needed) ---

// renderTodosPanel renders the top-5 urgent todos.
func renderTodosPanel(todos []todo.Todo, open, overdue, width int) string {
	var b strings.Builder

	countStr := fmt.Sprintf(" %d open", open)
	if overdue > 0 {
		countStr += ui.Error.Render(fmt.Sprintf(" · %d overdue!", overdue))
	}
	b.WriteString("  " + ui.Title.Render(ui.IconTodo+" Todos") + ui.Muted.Render(countStr) + "\n\n")

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	shown := todos
	if len(shown) > 5 {
		shown = shown[:5]
	}

	if len(shown) == 0 {
		b.WriteString("  " + ui.Muted.Render("All clear! Press 't' to add a task.") + "\n")
	} else {
		for _, t := range shown {
			b.WriteString(renderDashTodoItem(t, today, width) + "\n")
		}
	}

	if len(todos) > 5 {
		b.WriteString("  " + ui.Muted.Render(fmt.Sprintf("…and %d more", len(todos)-5)) + "\n")
	}

	return b.String()
}

// renderDashTodoItem renders a single read-only todo row for the dashboard.
func renderDashTodoItem(t todo.Todo, today time.Time, width int) string {
	id := lipgloss.NewStyle().Width(todo.ColWidthID).Render(ui.Muted.Render(fmt.Sprintf("#%d", t.ID)))
	prio := todo.FormatPriorityIcon(t.Priority)
	sched := todo.FormatScheduleTag(t.Schedule)

	// Compute available title width from total width minus fixed columns and spacing.
	// Format: "  %s %s %s %s" — 2 leading spaces + 3 separating spaces between columns.
	maxTitle := width - (2 + 3 + todo.ColWidthID + todo.ColWidthPrio + todo.ColWidthSched)
	if maxTitle < 10 {
		maxTitle = 10
	}
	title := t.Title
	if lipgloss.Width(title) > maxTitle {
		runes := []rune(title)
		for len(runes) > 0 && lipgloss.Width(string(runes)+"…") > maxTitle {
			runes = runes[:len(runes)-1]
		}
		title = string(runes) + "…"
	}
	if t.Done {
		title = ui.Muted.Render(title)
	}

	line := fmt.Sprintf("  %s %s %s %s", id, prio, sched, title)

	if t.DueDate != nil && !t.Done {
		due := *t.DueDate
		dueDay := time.Date(due.Year(), due.Month(), due.Day(), 0, 0, 0, 0, due.Location())
		switch {
		case dueDay.Before(today):
			line += ui.Error.Render(fmt.Sprintf(" (overdue: %s)", due.Format("Jan 2")))
		case dueDay.Equal(today):
			line += ui.Warning.Render(" (due today!)")
		}
	}

	return line
}

// renderFocusPanel renders the focus stats panel.
// The width parameter is reserved for future responsive layout; currently unused.
func renderFocusPanel(data DashData, _ int) string {
	var b strings.Builder

	b.WriteString("  " + ui.Title.Render(ui.IconDig+" Focus") + "\n\n")

	if data.Streak > 0 {
		b.WriteString(fmt.Sprintf("  %s %d-day streak\n", ui.IconFire, data.Streak))
	} else {
		b.WriteString("  " + ui.Muted.Render("No streak yet — start one today!") + "\n")
	}

	b.WriteString(fmt.Sprintf("  %s This week: %d done\n", ui.IconDone, data.WeekDone))

	if data.HasFocusData && data.TotalFocus > 0 {
		h := int(data.TotalFocus.Hours())
		m := int(data.TotalFocus.Minutes()) % 60
		b.WriteString(fmt.Sprintf("  %s Total focus: %dh %dm\n", ui.IconGold, h, m))
	}

	return b.String()
}

// renderProjectPanel renders the current project context panel.
// The width parameter is reserved for future responsive layout; currently unused.
func renderProjectPanel(p *proj.Project, openTodos, _ int) string {
	var b strings.Builder

	b.WriteString("  " + ui.Title.Render(ui.IconProject+" Project") + "\n\n")

	if p == nil {
		b.WriteString("  " + ui.Muted.Render("Not in a registered project.") + "\n")
		return b.String()
	}

	b.WriteString(fmt.Sprintf("  %s %s\n", ui.Accent.Render("▸"), p.Name))
	if p.Branch != "" {
		b.WriteString(fmt.Sprintf("  %s branch: %s\n", ui.Muted.Render("·"), p.Branch))
	}
	b.WriteString(fmt.Sprintf("  %s %d open todos\n", ui.Muted.Render("·"), openTodos))

	return b.String()
}

// renderHelpBar renders the keyboard shortcuts hint.
func renderHelpBar() string {
	return ui.Muted.Render("  t todos · d dig · r refresh · q quit")
}

// --- Data loading ---

func (m *DashModel) loadData() tea.Cmd {
	return func() tea.Msg {
		data := DashData{}
		now := time.Now()

		p, _ := m.ps.FindForCWD()
		// Enrich with branch info (FindForCWD skips git lookup for speed).
		if p != nil {
			if full, err := m.ps.Get(p.Name); err == nil {
				p = full
			}
		}
		data.Project = p

		var projPath *string
		if p != nil {
			projPath = &p.Path
		}

		ts := todo.NewStore(m.db)
		open, _, overdue, err := ts.Count(projPath)
		if err != nil {
			return dashErrMsg{err}
		}
		data.TodoOpen = open
		data.TodoOverdue = overdue

		// When projPath is nil (not inside a project), use AllProjects so the
		// list and count both cover all todos — Count(nil) already counts globally.
		listOpts := todo.ListOptions{
			ProjectPath:   projPath,
			AllProjects:   projPath == nil,
			ReferenceTime: now,
		}
		todos, err := ts.List(listOpts)
		if err != nil {
			return dashErrMsg{err}
		}
		data.Todos = todos

		stats, err := todo.GetStats(m.db, projPath, now)
		if err != nil {
			return dashErrMsg{err}
		}
		if stats != nil {
			data.Streak = stats.Streak
			data.WeekDone = stats.CompletedWeek
			data.TotalFocus = stats.TotalFocus
			data.HasFocusData = stats.HasFocusData
		}

		return dashDataMsg(data)
	}
}
