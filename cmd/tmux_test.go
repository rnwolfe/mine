package cmd

import (
	"testing"
)

func TestLayoutItemFilterValue(t *testing.T) {
	item := layoutItem{name: "dev-setup", description: "3 windows"}
	if got := item.FilterValue(); got != "dev-setup" {
		t.Errorf("FilterValue: want %q, got %q", "dev-setup", got)
	}
}

func TestLayoutItemTitle(t *testing.T) {
	item := layoutItem{name: "dev-setup", description: "3 windows"}
	if got := item.Title(); got != "dev-setup" {
		t.Errorf("Title: want %q, got %q", "dev-setup", got)
	}
}

func TestLayoutItemDescription(t *testing.T) {
	item := layoutItem{name: "dev-setup", description: "3 windows"}
	if got := item.Description(); got != "3 windows" {
		t.Errorf("Description: want %q, got %q", "3 windows", got)
	}
}

func TestLayoutItemEmptyDescription(t *testing.T) {
	item := layoutItem{name: "minimal"}
	if got := item.Description(); got != "" {
		t.Errorf("Description: want empty, got %q", got)
	}
}

func TestTmuxLayoutLoadCmdAcceptsZeroArgs(t *testing.T) {
	if err := tmuxLayoutLoadCmd.Args(tmuxLayoutLoadCmd, []string{}); err != nil {
		t.Errorf("expected 0 args to be valid, got: %v", err)
	}
}

func TestTmuxLayoutLoadCmdAcceptsOneArg(t *testing.T) {
	if err := tmuxLayoutLoadCmd.Args(tmuxLayoutLoadCmd, []string{"dev-setup"}); err != nil {
		t.Errorf("expected 1 arg to be valid, got: %v", err)
	}
}

func TestTmuxLayoutLoadCmdRejectsTwoArgs(t *testing.T) {
	if err := tmuxLayoutLoadCmd.Args(tmuxLayoutLoadCmd, []string{"dev-setup", "extra"}); err == nil {
		t.Error("expected 2 args to be rejected, but no error")
	}
}
