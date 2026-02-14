package ui

import (
	"testing"
)

func TestGreet(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"", "⛏ Hey there!"},
		{"Ryan", "⛏ Hey Ryan!"},
		{"World", "⛏ Hey World!"},
	}

	for _, tt := range tests {
		got := Greet(tt.name)
		if got != tt.expected {
			t.Errorf("Greet(%q) = %q, want %q", tt.name, got, tt.expected)
		}
	}
}

func TestIconConstants(t *testing.T) {
	// Verify icons are non-empty strings
	icons := []string{
		IconPick, IconGem, IconGold, IconTodo, IconDone, IconOverdue,
		IconTools, IconPackage, IconVault, IconGrow, IconStar, IconFire,
		IconWarn, IconError, IconOk, IconArrow, IconDot, IconDig,
	}
	for i, icon := range icons {
		if icon == "" {
			t.Errorf("Icon at index %d is empty", i)
		}
	}
}
