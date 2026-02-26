package todo

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/rnwolfe/mine/internal/ui"
)

// Column display widths for consistent alignment across CLI and TUI renderers.
// Use these constants with lipgloss.NewStyle().Width(N).Render() for ANSI-safe padding.
const (
	// ColWidthID is the display width of the ID column: "#" + up to 3 digits.
	ColWidthID = 4
	// ColWidthPrio is the display width of the priority icon column (emoji = 2 cells).
	ColWidthPrio = 2
	// ColWidthSched is the display width of the schedule tag column ("▸T", "▸S", etc.).
	ColWidthSched = 2
)

// FormatScheduleTag returns a fixed-width (ColWidthSched) styled schedule indicator.
// The compact form is always exactly 2 display columns wide and suitable for both
// CLI list output and TUI renderers. This consolidates renderScheduleTag (cmd/todo.go)
// and tuiScheduleTag (internal/tui/todo.go) into a single shared helper.
func FormatScheduleTag(schedule string) string {
	switch schedule {
	case ScheduleToday:
		return lipgloss.NewStyle().Width(ColWidthSched).Render(ui.ScheduleTodayStyle.Render("▸T"))
	case ScheduleSoon:
		return lipgloss.NewStyle().Width(ColWidthSched).Render(ui.ScheduleSoonStyle.Render("▸S"))
	case ScheduleSomeday:
		return lipgloss.NewStyle().Width(ColWidthSched).Render(ui.Muted.Render("▸?"))
	default: // later
		return lipgloss.NewStyle().Width(ColWidthSched).Render(ui.Muted.Render("▸·"))
	}
}
