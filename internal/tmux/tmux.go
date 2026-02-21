package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Session represents a tmux session.
type Session struct {
	Name     string
	Windows  int
	Created  time.Time
	Attached bool
}

// FilterValue implements tui.Item for fuzzy matching.
func (s Session) FilterValue() string { return s.Name }

// Title implements tui.Item.
func (s Session) Title() string { return s.Name }

// Description implements tui.Item — window count and attached status.
func (s Session) Description() string {
	w := "window"
	if s.Windows != 1 {
		w = "windows"
	}
	desc := fmt.Sprintf("%d %s", s.Windows, w)
	if s.Attached {
		desc += "  (attached)"
	}
	return desc
}

// Available returns true if the tmux binary is found in PATH.
func Available() bool {
	_, err := exec.LookPath("tmux")
	return err == nil
}

// InsideTmux returns true if we're currently inside a tmux session.
func InsideTmux() bool {
	return os.Getenv("TMUX") != ""
}

// ListSessions returns all active tmux sessions, or an empty slice if the
// server is not running.
func ListSessions() ([]Session, error) {
	return listSessionsFunc()
}

// listSessionsFunc is the internal implementation, replaceable for testing.
var listSessionsFunc = listSessionsReal

func listSessionsReal() ([]Session, error) {
	out, err := tmuxCmd("list-sessions", "-F",
		"#{session_name}\t#{session_windows}\t#{session_created}\t#{session_attached}")
	if err != nil {
		// "no server running" is a normal condition — return empty.
		if strings.Contains(err.Error(), "no server running") ||
			strings.Contains(err.Error(), "no current") ||
			strings.Contains(err.Error(), "error connecting") {
			return nil, nil
		}
		return nil, fmt.Errorf("listing sessions: %w", err)
	}
	return parseSessions(out), nil
}

// NewSession creates a new tmux session. If name is empty, it auto-names
// from the current working directory basename. Returns the resolved session name.
func NewSession(name string) (string, error) {
	if name == "" {
		dir, err := os.Getwd()
		if err != nil {
			return "", err
		}
		name = filepath.Base(dir)
	}

	args := []string{"new-session", "-d", "-s", name}
	_, err := tmuxCmd(args...)
	if err != nil {
		return "", fmt.Errorf("creating session %q: %w", name, err)
	}
	return name, nil
}

// AttachSession attaches to (or switches to) the named session.
// If inside tmux, it uses switch-client; otherwise it uses attach-session.
func AttachSession(name string) error {
	if InsideTmux() {
		return tmuxExec("switch-client", "-t", name)
	}
	return tmuxExec("attach-session", "-t", name)
}

// KillSession destroys the named session.
func KillSession(name string) error {
	_, err := tmuxCmd("kill-session", "-t", name)
	if err != nil {
		return fmt.Errorf("killing session %q: %w", name, err)
	}
	return nil
}

// RenameSession renames an existing tmux session from oldName to newName.
func RenameSession(oldName, newName string) error {
	if newName == "" {
		return fmt.Errorf("new session name cannot be empty")
	}
	_, err := tmuxCmd("rename-session", "-t", oldName, newName)
	if err != nil {
		return fmt.Errorf("renaming session %q to %q: %w", oldName, newName, err)
	}
	return nil
}

// FuzzyFindSession returns the first session whose name fuzzy-matches the query,
// or an error if no match is found. Used by attach/kill for flexible name matching.
func FuzzyFindSession(query string, sessions []Session) (*Session, error) {
	// Exact match first.
	for i := range sessions {
		if strings.EqualFold(sessions[i].Name, query) {
			return &sessions[i], nil
		}
	}
	// Prefix match.
	q := strings.ToLower(query)
	for i := range sessions {
		if strings.HasPrefix(strings.ToLower(sessions[i].Name), q) {
			return &sessions[i], nil
		}
	}
	// Substring match.
	for i := range sessions {
		if strings.Contains(strings.ToLower(sessions[i].Name), q) {
			return &sessions[i], nil
		}
	}
	return nil, fmt.Errorf("no session matching %q", query)
}

// --- internal helpers ---

// tmuxCmd runs tmux with args and returns combined output.
// It is a package-level var so tests can inject stubs.
var tmuxCmd = tmuxCmdReal

func tmuxCmdReal(args ...string) (string, error) {
	cmd := exec.Command("tmux", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %s", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

// tmuxExec replaces the current process with tmux (for attach/switch).
func tmuxExec(args ...string) error {
	bin, err := exec.LookPath("tmux")
	if err != nil {
		return fmt.Errorf("tmux not found: %w", err)
	}
	argv := append([]string{"tmux"}, args...)
	return execSyscall(bin, argv, os.Environ())
}

// execSyscall is the process replacement function, replaceable for testing.
var execSyscall = defaultExecSyscall

func defaultExecSyscall(binary string, argv []string, envv []string) error {
	// syscall.Exec replaces the current process image with tmux, so mine
	// disappears from the process tree entirely. The user returns to their
	// original shell when they detach from the tmux session.
	return syscall.Exec(binary, argv, envv)
}

// parseSessions parses tmux list-sessions formatted output.
func parseSessions(raw string) []Session {
	if raw == "" {
		return nil
	}
	var sessions []Session
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 {
			continue
		}

		windows, _ := strconv.Atoi(parts[1])
		created, _ := strconv.ParseInt(parts[2], 10, 64)
		attached := parts[3] == "1"

		sessions = append(sessions, Session{
			Name:     parts[0],
			Windows:  windows,
			Created:  time.Unix(created, 0),
			Attached: attached,
		})
	}
	return sessions
}
