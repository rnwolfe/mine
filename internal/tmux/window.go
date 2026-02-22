package tmux

import (
	"fmt"
	"strconv"
	"strings"
)

// Window represents a tmux window within a session.
type Window struct {
	Index  int
	Name   string
	Active bool
}

// FilterValue implements tui.Item for fuzzy matching.
func (w Window) FilterValue() string { return w.Name }

// Title implements tui.Item.
func (w Window) Title() string { return w.Name }

// Description implements tui.Item — index and active indicator.
func (w Window) Description() string {
	desc := fmt.Sprintf("index %d", w.Index)
	if w.Active {
		desc += "  (active)"
	}
	return desc
}

// CurrentSession returns the name of the currently attached tmux session.
// Returns an error if not inside tmux or the query fails.
var CurrentSession = currentSessionReal

func currentSessionReal() (string, error) {
	if !InsideTmux() {
		return "", fmt.Errorf("not inside a tmux session")
	}
	out, err := tmuxCmd("display-message", "-p", "#{session_name}")
	if err != nil {
		return "", fmt.Errorf("getting current session name: %w", err)
	}
	name := strings.TrimSpace(out)
	if name == "" {
		return "", fmt.Errorf("could not determine current session name")
	}
	return name, nil
}

// ListWindows returns all windows in the named session.
func ListWindows(session string) ([]Window, error) {
	out, err := tmuxCmd("list-windows", "-t", session, "-F",
		"#{window_index}\t#{window_name}\t#{window_active}")
	if err != nil {
		return nil, fmt.Errorf("listing windows for session %q: %w", session, err)
	}
	return parseWindows(out), nil
}

// NewWindow creates a new named window in the given session.
func NewWindow(session, name string) error {
	args := []string{"new-window", "-t", session}
	if name != "" {
		args = append(args, "-n", name)
	}
	_, err := tmuxCmd(args...)
	if err != nil {
		return fmt.Errorf("creating window %q in session %q: %w", name, session, err)
	}
	return nil
}

// KillWindow destroys the named window in the given session.
func KillWindow(session, name string) error {
	target := session + ":" + name
	_, err := tmuxCmd("kill-window", "-t", target)
	if err != nil {
		return fmt.Errorf("killing window %q in session %q: %w", name, session, err)
	}
	return nil
}

// RenameWindow renames a window within a session from oldName to newName.
func RenameWindow(session, oldName, newName string) error {
	if newName == "" {
		return fmt.Errorf("new window name cannot be empty")
	}
	target := session + ":" + oldName
	_, err := tmuxCmd("rename-window", "-t", target, newName)
	if err != nil {
		return fmt.Errorf("renaming window %q to %q in session %q: %w", oldName, newName, session, err)
	}
	return nil
}

// FindWindowByName returns the window with the given exact name, or nil if not found.
func FindWindowByName(name string, windows []Window) *Window {
	for i := range windows {
		if windows[i].Name == name {
			return &windows[i]
		}
	}
	return nil
}

// parseWindows parses tmux list-windows formatted output.
func parseWindows(raw string) []Window {
	if raw == "" {
		return nil
	}
	var windows []Window
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}
		index, _ := strconv.Atoi(parts[0])
		active := parts[2] == "1"
		windows = append(windows, Window{
			Index:  index,
			Name:   parts[1],
			Active: active,
		})
	}
	return windows
}
