package tmux

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/rnwolfe/mine/internal/config"
)

// Layout represents a saved tmux window/pane arrangement.
type Layout struct {
	Name    string         `toml:"name"`
	Windows []WindowLayout `toml:"windows"`
}

// WindowLayout describes a single window within a layout.
type WindowLayout struct {
	Name       string       `toml:"name"`
	Layout     string       `toml:"layout"` // tmux layout string (e.g. "main-vertical")
	PaneCount  int          `toml:"pane_count"`
	Panes      []PaneLayout `toml:"panes"`
}

// PaneLayout describes a single pane within a window.
type PaneLayout struct {
	Dir     string `toml:"dir"`
	Command string `toml:"command,omitempty"`
}

// layoutDir returns the directory where layouts are stored.
func layoutDir() string {
	return filepath.Join(config.GetPaths().ConfigDir, "tmux", "layouts")
}

// layoutPath returns the file path for a named layout.
func layoutPath(name string) string {
	return filepath.Join(layoutDir(), name+".toml")
}

// SaveLayout captures the current session's layout and persists it.
func SaveLayout(name string) error {
	layout, err := captureLayout(name)
	if err != nil {
		return err
	}
	return writeLayout(layout)
}

// LoadLayout restores a saved layout into the current session.
func LoadLayout(name string) error {
	layout, err := ReadLayout(name)
	if err != nil {
		return err
	}
	return applyLayout(layout)
}

// ReadLayout reads a layout from disk without applying it.
func ReadLayout(name string) (*Layout, error) {
	path := layoutPath(name)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("layout %q not found", name)
		}
		return nil, fmt.Errorf("reading layout: %w", err)
	}

	var layout Layout
	if err := toml.Unmarshal(data, &layout); err != nil {
		return nil, fmt.Errorf("parsing layout %q: %w", name, err)
	}
	return &layout, nil
}

// ListLayouts returns the names of all saved layouts.
func ListLayouts() ([]string, error) {
	dir := layoutDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".toml") {
			continue
		}
		names = append(names, strings.TrimSuffix(e.Name(), ".toml"))
	}
	return names, nil
}

// DeleteLayout removes a saved layout.
func DeleteLayout(name string) error {
	path := layoutPath(name)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("layout %q not found", name)
		}
		return err
	}
	return nil
}

// --- internal helpers ---

// captureLayout reads the current tmux session and builds a Layout.
func captureLayout(name string) (*Layout, error) {
	// Get windows in current session.
	out, err := tmuxCmd("list-windows", "-F",
		"#{window_name}\t#{window_layout}\t#{window_panes}")
	if err != nil {
		return nil, fmt.Errorf("listing windows: %w", err)
	}

	layout := &Layout{Name: name}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 3 {
			continue
		}

		paneCount := 1
		fmt.Sscanf(parts[2], "%d", &paneCount)

		wl := WindowLayout{
			Name:      parts[0],
			Layout:    parts[1],
			PaneCount: paneCount,
		}

		// Capture pane details for this window.
		paneOut, err := tmuxCmd("list-panes", "-t", parts[0], "-F",
			"#{pane_current_path}\t#{pane_current_command}")
		if err == nil {
			for _, paneLine := range strings.Split(paneOut, "\n") {
				paneLine = strings.TrimSpace(paneLine)
				if paneLine == "" {
					continue
				}
				pp := strings.SplitN(paneLine, "\t", 2)
				pl := PaneLayout{Dir: pp[0]}
				if len(pp) > 1 {
					pl.Command = pp[1]
				}
				wl.Panes = append(wl.Panes, pl)
			}
		}

		layout.Windows = append(layout.Windows, wl)
	}

	return layout, nil
}

// writeLayout persists a Layout to disk as TOML.
func writeLayout(layout *Layout) error {
	dir := layoutDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	path := layoutPath(layout.Name)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	f.WriteString("# mine tmux layout — saved by: mine tmux layout save\n\n")
	return toml.NewEncoder(f).Encode(layout)
}

// WriteLayout is exported for testing — writes a layout struct to disk.
func WriteLayout(layout *Layout) error {
	return writeLayout(layout)
}

// applyLayout creates windows and panes from a saved layout.
func applyLayout(layout *Layout) error {
	for i, w := range layout.Windows {
		if i == 0 {
			// Rename the current window rather than creating a new one.
			if _, err := tmuxCmd("rename-window", w.Name); err != nil {
				return fmt.Errorf("renaming window to %q: %w", w.Name, err)
			}
		} else {
			if _, err := tmuxCmd("new-window", "-n", w.Name); err != nil {
				return fmt.Errorf("creating window %q: %w", w.Name, err)
			}
		}

		// Create additional panes.
		for j := 1; j < w.PaneCount; j++ {
			if _, err := tmuxCmd("split-window", "-t", w.Name); err != nil {
				return fmt.Errorf("splitting pane %d in window %q: %w", j, w.Name, err)
			}
		}

		// Apply the layout string.
		if w.Layout != "" {
			if _, err := tmuxCmd("select-layout", "-t", w.Name, w.Layout); err != nil {
				return fmt.Errorf("applying layout to window %q: %w", w.Name, err)
			}
		}

		// Set pane directories.
		for j, p := range w.Panes {
			if p.Dir != "" {
				paneTarget := fmt.Sprintf("%s.%d", w.Name, j)
				if _, err := tmuxCmd("send-keys", "-t", paneTarget,
					fmt.Sprintf("cd %q && clear", p.Dir), "Enter"); err != nil {
					return fmt.Errorf("sending cd to pane %s: %w", paneTarget, err)
				}
			}
		}
	}

	// Select the first window.
	if len(layout.Windows) > 0 {
		if _, err := tmuxCmd("select-window", "-t", layout.Windows[0].Name); err != nil {
			return fmt.Errorf("selecting window %q: %w", layout.Windows[0].Name, err)
		}
	}

	return nil
}
