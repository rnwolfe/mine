package cmd

import (
	"fmt"
	"strings"

	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/tmux"
	"github.com/rnwolfe/mine/internal/tui"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

func init() {
	tmuxLayoutCmd.AddCommand(tmuxLayoutSaveCmd)
	tmuxLayoutCmd.AddCommand(tmuxLayoutLoadCmd)
	tmuxLayoutCmd.AddCommand(tmuxLayoutLsCmd)
	tmuxLayoutCmd.AddCommand(tmuxLayoutPreviewCmd)
	tmuxLayoutCmd.AddCommand(tmuxLayoutDeleteCmd)
}

// --- mine tmux layout ---

var tmuxLayoutCmd = &cobra.Command{
	Use:   "layout",
	Short: "Save and restore window layouts",
	RunE:  hook.Wrap("tmux.layout", runTmuxLayoutHelp),
}

func runTmuxLayoutHelp(_ *cobra.Command, _ []string) error {
	// When inside tmux, check saved layouts and act accordingly.
	if tmux.InsideTmux() {
		if !tmux.Available() {
			return fmt.Errorf("tmux not found in PATH")
		}
		names, err := tmux.ListLayouts()
		if err != nil {
			return err
		}

		if len(names) == 0 {
			fmt.Println()
			fmt.Println(ui.Muted.Render("  No saved layouts."))
			fmt.Printf("  Save one: %s\n", ui.Accent.Render("mine tmux layout save <name>"))
			fmt.Println()
			return nil
		}

		// Open interactive picker when a TTY is available.
		if tui.IsTTY() {
			items := make([]tui.Item, len(names))
			for i, n := range names {
				desc := ""
				if layout, err := tmux.ReadLayout(n); err == nil {
					w := "windows"
					if len(layout.Windows) == 1 {
						w = "window"
					}
					desc = fmt.Sprintf("%d %s", len(layout.Windows), w)
				} else {
					desc = "(error reading)"
				}
				items[i] = layoutItem{name: n, description: desc}
			}

			chosen, err := tui.Run(items,
				tui.WithTitle(ui.IconMine+"Load layout"),
				tui.WithHeight(12),
			)
			if err != nil {
				return err
			}
			if chosen == nil {
				return nil // user canceled
			}

			name := chosen.Title()
			if err := tmux.LoadLayout(name); err != nil {
				return err
			}

			ui.Ok(fmt.Sprintf("Layout %s restored", ui.Accent.Render(name)))
			fmt.Println()
			return nil
		}
	}

	// Fallback: show help text when outside tmux or non-TTY.
	fmt.Println()
	fmt.Println(ui.Title.Render("  Tmux Layouts"))
	fmt.Println()
	fmt.Printf("  %s  %s\n", ui.Accent.Render("mine tmux layout save <name>"), ui.Muted.Render("Save current layout"))
	fmt.Printf("  %s  %s\n", ui.Accent.Render("mine tmux layout load <name>"), ui.Muted.Render("Restore a layout"))
	fmt.Printf("  %s  %s\n", ui.Accent.Render("mine tmux layout ls"), ui.Muted.Render("List saved layouts"))
	fmt.Printf("  %s  %s\n", ui.Accent.Render("mine tmux layout preview <name>"), ui.Muted.Render("Preview layout details"))
	fmt.Printf("  %s  %s\n", ui.Accent.Render("mine tmux layout delete <name>"), ui.Muted.Render("Delete a saved layout"))
	fmt.Println()
	return nil
}

// --- mine tmux layout save ---

var tmuxLayoutSaveCmd = &cobra.Command{
	Use:   "save <name>",
	Short: "Save the current window/pane layout",
	Args:  cobra.ExactArgs(1),
	RunE:  hook.Wrap("tmux.layout.save", runTmuxLayoutSave),
}

func runTmuxLayoutSave(_ *cobra.Command, args []string) error {
	if !tmux.Available() {
		return fmt.Errorf("tmux not found in PATH")
	}
	if !tmux.InsideTmux() {
		return fmt.Errorf("not inside a tmux session — attach first")
	}

	name := args[0]
	if err := tmux.SaveLayout(name); err != nil {
		return err
	}

	ui.Ok(fmt.Sprintf("Layout %s saved", ui.Accent.Render(name)))
	fmt.Printf("  Restore with: %s\n", ui.Muted.Render("mine tmux layout load "+name))
	fmt.Println()
	return nil
}

// --- mine tmux layout load ---

var tmuxLayoutLoadCmd = &cobra.Command{
	Use:   "load [name]",
	Short: "Restore a saved layout",
	Args:  cobra.MaximumNArgs(1),
	RunE:  hook.Wrap("tmux.layout.load", runTmuxLayoutLoad),
}

func runTmuxLayoutLoad(_ *cobra.Command, args []string) error {
	if !tmux.Available() {
		return fmt.Errorf("tmux not found in PATH")
	}
	if !tmux.InsideTmux() {
		return fmt.Errorf("not inside a tmux session — attach first")
	}

	var name string

	if len(args) > 0 {
		name = args[0]
	} else {
		// No name given: open fuzzy picker (TTY) or list layouts (non-TTY).
		names, err := tmux.ListLayouts()
		if err != nil {
			return err
		}

		if len(names) == 0 {
			fmt.Println()
			fmt.Println(ui.Muted.Render("  No saved layouts."))
			fmt.Printf("  Save one: %s\n", ui.Accent.Render("mine tmux layout save <name>"))
			fmt.Println()
			return nil
		}

		if !tui.IsTTY() {
			fmt.Println()
			fmt.Println(ui.Muted.Render("  Specify a layout name or run interactively in a terminal."))
			fmt.Println()
			for _, n := range names {
				fmt.Printf("  %s\n", ui.Accent.Render(n))
			}
			fmt.Println()
			return fmt.Errorf("no layout name given — specify a name: mine tmux layout load <name>")
		}

		items := make([]tui.Item, len(names))
		for i, n := range names {
			desc := ""
			if layout, err := tmux.ReadLayout(n); err == nil {
				w := "windows"
				if len(layout.Windows) == 1 {
					w = "window"
				}
				desc = fmt.Sprintf("%d %s", len(layout.Windows), w)
			} else {
				desc = "(error reading)"
			}
			items[i] = layoutItem{name: n, description: desc}
		}

		chosen, err := tui.Run(items,
			tui.WithTitle(ui.IconMine+"Load layout"),
			tui.WithHeight(12),
		)
		if err != nil {
			return err
		}
		if chosen == nil {
			return nil // user canceled
		}
		name = chosen.Title()
	}

	if err := tmux.LoadLayout(name); err != nil {
		return err
	}

	ui.Ok(fmt.Sprintf("Layout %s restored", ui.Accent.Render(name)))
	fmt.Println()
	return nil
}

// --- mine tmux layout preview ---

var tmuxLayoutPreviewCmd = &cobra.Command{
	Use:   "preview <name>",
	Short: "Show layout details without loading it",
	Long:  `Display layout name, save timestamp, and window details without entering or modifying any tmux session. Works outside of tmux.`,
	Args:  cobra.ExactArgs(1),
	RunE:  hook.Wrap("tmux.layout.preview", runTmuxLayoutPreview),
}

func runTmuxLayoutPreview(_ *cobra.Command, args []string) error {
	name := args[0]
	layout, err := tmux.ReadLayout(name)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("  %s  %s\n", ui.Muted.Render("Layout:"), ui.Accent.Render(layout.Name))
	if !layout.SavedAt.IsZero() {
		fmt.Printf("  %s  %s\n", ui.Muted.Render("Saved:"), ui.Muted.Render(layout.SavedAt.Format("2006-01-02 15:04")))
	}
	fmt.Println()

	if len(layout.Windows) == 0 {
		fmt.Println(ui.Muted.Render("  No windows in layout."))
		fmt.Println()
		return nil
	}

	fmt.Printf("  %s  %s  %s\n",
		ui.Muted.Render(fmt.Sprintf("%-20s", "Window")),
		ui.Muted.Render(fmt.Sprintf("%-6s", "Panes")),
		ui.Muted.Render("Directory"),
	)
	for _, w := range layout.Windows {
		dir := ""
		if len(w.Panes) > 0 {
			dir = w.Panes[0].Dir
		}
		fmt.Printf("  %s  %s  %s\n",
			ui.Accent.Render(fmt.Sprintf("%-20s", w.Name)),
			fmt.Sprintf("%-6d", w.PaneCount),
			ui.Muted.Render(dir),
		)
	}
	fmt.Println()
	return nil
}

// layoutItem adapts a layout name for the TUI fuzzy picker.
type layoutItem struct {
	name        string
	description string
}

func (l layoutItem) FilterValue() string { return l.name }
func (l layoutItem) Title() string       { return l.name }
func (l layoutItem) Description() string { return l.description }

// --- mine tmux layout ls ---

var tmuxLayoutLsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List saved layouts",
	RunE:    hook.Wrap("tmux.layout.ls", runTmuxLayoutLs),
}

func runTmuxLayoutLs(_ *cobra.Command, _ []string) error {
	names, err := tmux.ListLayouts()
	if err != nil {
		return err
	}

	if len(names) == 0 {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No saved layouts."))
		fmt.Printf("  Save one: %s\n", ui.Accent.Render("mine tmux layout save <name>"))
		fmt.Println()
		return nil
	}

	fmt.Println()
	for _, name := range names {
		layout, err := tmux.ReadLayout(name)
		if err != nil {
			fmt.Printf("  %s %s\n", ui.Accent.Render(name), ui.Muted.Render("(error reading)"))
			continue
		}

		windows := fmt.Sprintf("%d windows", len(layout.Windows))
		if len(layout.Windows) == 1 {
			windows = "1 window"
		}

		windowNames := make([]string, len(layout.Windows))
		for i, w := range layout.Windows {
			windowNames[i] = w.Name
		}

		savedAt := ""
		if !layout.SavedAt.IsZero() {
			savedAt = "  " + ui.Muted.Render(layout.SavedAt.Format("2006-01-02 15:04"))
		}

		fmt.Printf("  %s  %s  %s%s\n",
			ui.Accent.Render(fmt.Sprintf("%-16s", name)),
			ui.Muted.Render(fmt.Sprintf("%-12s", windows)),
			ui.Muted.Render("["+strings.Join(windowNames, ", ")+"]"),
			savedAt,
		)
	}
	fmt.Println()
	return nil
}

// --- mine tmux layout delete ---

var tmuxLayoutDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a saved layout",
	Args:  cobra.MaximumNArgs(1),
	RunE:  hook.Wrap("tmux.layout.delete", runTmuxLayoutDelete),
}

func runTmuxLayoutDelete(_ *cobra.Command, args []string) error {
	if len(args) == 1 {
		name := args[0]
		if err := tmux.DeleteLayout(name); err != nil {
			return err
		}
		ui.Ok(fmt.Sprintf("Layout %s deleted", ui.Accent.Render(name)))
		fmt.Println()
		return nil
	}

	names, err := tmux.ListLayouts()
	if err != nil {
		return err
	}

	if len(names) == 0 {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No saved layouts."))
		fmt.Println()
		return nil
	}

	// No name: use picker if TTY, else show list and require a name.
	if !tui.IsTTY() {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  Specify a layout name or run interactively in a terminal."))
		return printLayoutList(names)
	}

	items := make([]tui.Item, len(names))
	for i, n := range names {
		items[i] = layoutItem{name: n}
	}

	chosen, err := tui.Run(items,
		tui.WithTitle(ui.IconMine+"Delete layout"),
		tui.WithHeight(12),
	)
	if err != nil {
		return err
	}
	if chosen == nil {
		return nil // user canceled
	}

	name := chosen.Title()
	if err := tmux.DeleteLayout(name); err != nil {
		return err
	}
	ui.Ok(fmt.Sprintf("Layout %s deleted", ui.Accent.Render(name)))
	fmt.Println()
	return nil
}

// --- helpers ---

func printLayoutList(names []string) error {
	fmt.Println()
	for _, name := range names {
		fmt.Printf("  %s\n", ui.Accent.Render(name))
	}
	fmt.Println()
	return nil
}
