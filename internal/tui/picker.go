package tui

import (
	"fmt"
	"io"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
	"github.com/rnwolfe/mine/internal/ui"
)

// Item is the interface that list items must implement for the picker.
type Item interface {
	// FilterValue returns the string used for fuzzy matching.
	FilterValue() string
	// Title returns the main display text.
	Title() string
	// Description returns optional secondary text (can be empty).
	Description() string
}

// PickerOption configures a Picker.
type PickerOption func(*Picker)

// WithTitle sets the heading displayed above the picker.
func WithTitle(title string) PickerOption {
	return func(p *Picker) { p.title = title }
}

// WithPrompt sets the search prompt character(s).
func WithPrompt(prompt string) PickerOption {
	return func(p *Picker) { p.prompt = prompt }
}

// WithHeight sets the maximum visible items (0 = auto).
func WithHeight(h int) PickerOption {
	return func(p *Picker) { p.height = h }
}

// Picker is a reusable fuzzy-search list selector built on Bubbletea.
// Use Run() for the common case, or create a Picker and drive it manually.
type Picker struct {
	title  string
	prompt string
	height int

	items    []Item
	filtered []scored
	query    string
	cursor   int
	offset   int // viewport scroll offset
	chosen   Item
	canceled bool

	termWidth  int
	termHeight int
}

type scored struct {
	item  Item
	score int
}

// NewPicker creates a Picker with the given items and options.
func NewPicker(items []Item, opts ...PickerOption) *Picker {
	p := &Picker{
		prompt:     "> ",
		height:     10,
		items:      items,
		termWidth:  80,
		termHeight: 24,
	}
	for _, opt := range opts {
		opt(p)
	}
	p.applyFilter()
	return p
}

// Run is the convenience entry point: show a picker and return the selected item.
// Returns nil and no error if the user canceled.
func Run(items []Item, opts ...PickerOption) (Item, error) {
	return RunWithOutput(items, os.Stdout, opts...)
}

// RunWithOutput renders the picker UI to the provided writer.
// Useful when stdout is captured but an interactive TTY UI is still desired.
func RunWithOutput(items []Item, output io.Writer, opts ...PickerOption) (Item, error) {
	p := NewPicker(items, opts...)
	prog := tea.NewProgram(p, tea.WithAltScreen(), tea.WithOutput(output))
	m, err := prog.Run()
	if err != nil {
		return nil, fmt.Errorf("picker: %w", err)
	}
	result := m.(*Picker)
	if result.canceled {
		return nil, nil
	}
	return result.chosen, nil
}

// IsTTY returns true when stdin is connected to a terminal.
func IsTTY() bool {
	return isatty.IsTerminal(os.Stdin.Fd()) || isatty.IsCygwinTerminal(os.Stdin.Fd())
}

// IsOutputTTY returns true when stdout is connected to a terminal.
// Use this to guard TUI output — piped commands (e.g. mine | cat) have a TTY
// stdin but non-TTY stdout, so checking stdout is the correct signal for
// whether ANSI / full-screen output is appropriate.
func IsOutputTTY() bool {
	return isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
}

// --- Bubbletea model implementation ---

func (p *Picker) Init() tea.Cmd {
	return nil
}

func (p *Picker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		p.termWidth = msg.Width
		p.termHeight = msg.Height
		return p, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			p.canceled = true
			return p, tea.Quit

		case "enter":
			if len(p.filtered) > 0 {
				p.chosen = p.filtered[p.cursor].item
			}
			return p, tea.Quit

		case "up", "ctrl+p":
			if p.cursor > 0 {
				p.cursor--
				if p.cursor < p.offset {
					p.offset = p.cursor
				}
			}
			return p, nil

		case "down", "ctrl+n":
			if p.cursor < len(p.filtered)-1 {
				p.cursor++
				vis := p.visibleHeight()
				if p.cursor >= p.offset+vis {
					p.offset = p.cursor - vis + 1
				}
			}
			return p, nil

		case "backspace":
			if len(p.query) > 0 {
				runes := []rune(p.query)
				p.query = string(runes[:len(runes)-1])
				p.applyFilter()
			}
			return p, nil

		default:
			if len(msg.String()) == 1 {
				p.query += msg.String()
				p.applyFilter()
			}
			return p, nil
		}
	}
	return p, nil
}

func (p *Picker) View() string {
	var b strings.Builder

	// Title
	if p.title != "" {
		b.WriteString("  " + ui.Title.Render(p.title) + "\n\n")
	}

	// Query input
	cursor := lipgloss.NewStyle().Foreground(ui.Gold).Bold(true).Render(p.prompt)
	b.WriteString("  " + cursor + p.query + blinkCursor() + "\n\n")

	// Filtered list
	vis := p.visibleHeight()
	end := p.offset + vis
	if end > len(p.filtered) {
		end = len(p.filtered)
	}

	if len(p.filtered) == 0 {
		b.WriteString("  " + ui.Muted.Render("No matches") + "\n")
	} else {
		for i := p.offset; i < end; i++ {
			item := p.filtered[i].item
			isSelected := i == p.cursor

			line := p.renderItem(item, isSelected)
			b.WriteString(line + "\n")
		}
	}

	// Status bar
	b.WriteString("\n")
	status := ui.Muted.Render(fmt.Sprintf("  %d/%d", len(p.filtered), len(p.items)))
	help := ui.Muted.Render(" · ↑↓ navigate · enter select · esc cancel")
	b.WriteString(status + help + "\n")

	return b.String()
}

// --- internal helpers ---

func (p *Picker) visibleHeight() int {
	h := p.height
	if h <= 0 || h > p.termHeight-6 {
		h = p.termHeight - 6
	}
	if h < 3 {
		h = 3
	}
	// When any visible item has a description, renderItem returns 2 terminal lines.
	// Halve the item budget so the picker does not overflow the terminal.
	if p.hasDescriptions() {
		h = h / 2
		if h < 2 {
			h = 2
		}
	}
	return h
}

// hasDescriptions returns true if any currently-filtered item has a non-empty description.
func (p *Picker) hasDescriptions() bool {
	for _, s := range p.filtered {
		if s.item.Description() != "" {
			return true
		}
	}
	return false
}

func (p *Picker) applyFilter() {
	p.filtered = nil
	if p.query == "" {
		for _, item := range p.items {
			p.filtered = append(p.filtered, scored{item: item, score: 0})
		}
	} else {
		for _, item := range p.items {
			if ok, sc := FuzzyMatch(p.query, item.FilterValue()); ok {
				p.filtered = append(p.filtered, scored{item: item, score: sc})
			}
		}
		// Sort by score descending (higher is better).
		sortScored(p.filtered)
	}
	p.cursor = 0
	p.offset = 0
}

func (p *Picker) renderItem(item Item, selected bool) string {
	pointer := "  "
	titleStyle := lipgloss.NewStyle()
	descStyle := ui.Muted.Copy()

	if selected {
		pointer = ui.Accent.Render(ui.IconArrow + " ")
		titleStyle = lipgloss.NewStyle().Foreground(ui.Gold).Bold(true)
	}

	// Truncate title to prevent overflow: termWidth minus 6 chars for prefix + margin.
	rawTitle := item.Title()
	maxWidth := p.termWidth - 6
	if maxWidth < 10 {
		maxWidth = 10
	}
	if lipgloss.Width(rawTitle) > maxWidth {
		runes := []rune(rawTitle)
		for len(runes) > 0 && lipgloss.Width(string(runes)+"…") > maxWidth {
			runes = runes[:len(runes)-1]
		}
		rawTitle = string(runes) + "…"
	}
	title := titleStyle.Render(rawTitle)

	// Description rendered on a second line, indented to align under the title.
	d := item.Description()
	if d == "" {
		return "  " + pointer + title
	}
	return "  " + pointer + title + "\n    " + descStyle.Render(d)
}

func blinkCursor() string {
	return lipgloss.NewStyle().Foreground(ui.Gold).Render("▎")
}

// sortScored sorts by score descending using insertion sort (stable, good for small N).
func sortScored(items []scored) {
	for i := 1; i < len(items); i++ {
		key := items[i]
		j := i - 1
		for j >= 0 && items[j].score < key.score {
			items[j+1] = items[j]
			j--
		}
		items[j+1] = key
	}
}
