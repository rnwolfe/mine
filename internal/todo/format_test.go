package todo

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestFormatScheduleTag_EqualWidth(t *testing.T) {
	schedules := []string{ScheduleToday, ScheduleSoon, ScheduleLater, ScheduleSomeday}

	// All four schedule values must produce a tag with the same display width.
	wantWidth := ColWidthSched
	for _, s := range schedules {
		tag := FormatScheduleTag(s)
		got := lipgloss.Width(tag)
		if got != wantWidth {
			t.Errorf("FormatScheduleTag(%q) has display width %d, want %d", s, got, wantWidth)
		}
	}
}

func TestFormatScheduleTag_UnknownSchedule(t *testing.T) {
	// Unknown schedule values should fall through to the "later" default.
	tag := FormatScheduleTag("bogus")
	got := lipgloss.Width(tag)
	if got != ColWidthSched {
		t.Errorf("FormatScheduleTag(\"bogus\") has display width %d, want %d", got, ColWidthSched)
	}
}

func TestFormatPriorityIcon_EqualWidth(t *testing.T) {
	priorities := []int{PrioCrit, PrioHigh, PrioMedium, PrioLow}

	// All four priority values must produce an icon with the same display width.
	wantWidth := ColWidthPrio
	for _, p := range priorities {
		icon := FormatPriorityIcon(p)
		got := lipgloss.Width(icon)
		if got != wantWidth {
			t.Errorf("FormatPriorityIcon(%d) has display width %d, want %d", p, got, wantWidth)
		}
	}
}
