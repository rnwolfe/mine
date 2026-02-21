package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/tmux"
	"github.com/rnwolfe/mine/internal/tui"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

var tmuxProjectLayout string

var tmuxCmd = &cobra.Command{
	Use:     "tmux",
	Aliases: []string{"tx"},
	Short:   "Tmux session management and layouts",
	Long:    `Manage tmux sessions, attach, and save/restore window layouts.`,
	RunE:    hook.Wrap("tmux", runTmux),
}

func init() {
	rootCmd.AddCommand(tmuxCmd)

	tmuxCmd.AddCommand(tmuxNewCmd)
	tmuxCmd.AddCommand(tmuxLsCmd)
	tmuxCmd.AddCommand(tmuxAttachCmd)
	tmuxCmd.AddCommand(tmuxKillCmd)
	tmuxCmd.AddCommand(tmuxRenameCmd)
	tmuxCmd.AddCommand(tmuxLayoutCmd)
	tmuxCmd.AddCommand(tmuxProjectCmd)

	tmuxLayoutCmd.AddCommand(tmuxLayoutSaveCmd)
	tmuxLayoutCmd.AddCommand(tmuxLayoutLoadCmd)
	tmuxLayoutCmd.AddCommand(tmuxLayoutLsCmd)
	tmuxLayoutCmd.AddCommand(tmuxLayoutDeleteCmd)

	tmuxProjectCmd.Flags().StringVar(&tmuxProjectLayout, "layout", "", "Apply a saved layout on creation (skipped on attach)")
}

// --- mine tmux (bare) — fuzzy session picker ---

func runTmux(_ *cobra.Command, _ []string) error {
	if !tmux.Available() {
		return fmt.Errorf("tmux not found in PATH — install tmux first")
	}

	sessions, err := tmux.ListSessions()
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No tmux sessions running."))
		fmt.Printf("  Create one: %s\n", ui.Accent.Render("mine tmux new"))
		fmt.Println()
		return nil
	}

	// Non-TTY fallback: plain list.
	if !tui.IsTTY() {
		return printSessionList(sessions)
	}

	// Interactive fuzzy picker.
	items := make([]tui.Item, len(sessions))
	for i := range sessions {
		items[i] = sessions[i]
	}

	chosen, err := tui.Run(items,
		tui.WithTitle(ui.IconPick+"Select tmux session"),
		tui.WithHeight(12),
	)
	if err != nil {
		return err
	}
	if chosen == nil {
		return nil // user canceled
	}

	return tmux.AttachSession(chosen.Title())
}

// --- mine tmux new ---

var tmuxNewCmd = &cobra.Command{
	Use:   "new [name]",
	Short: "Create a new tmux session",
	Long:  `Create a named tmux session. Auto-names from the current directory if omitted.`,
	Args:  cobra.MaximumNArgs(1),
	RunE:  hook.Wrap("tmux.new", runTmuxNew),
}

func runTmuxNew(_ *cobra.Command, args []string) error {
	if !tmux.Available() {
		return fmt.Errorf("tmux not found in PATH")
	}

	name := ""
	if len(args) > 0 {
		name = args[0]
	}

	resolved, err := tmux.NewSession(name, "")
	if err != nil {
		return err
	}

	ui.Ok(fmt.Sprintf("Session %s created", ui.Accent.Render(resolved)))
	fmt.Printf("  Attach: %s\n", ui.Muted.Render("mine tmux attach "+resolved))
	fmt.Println()
	return nil
}

// --- mine tmux project ---

var tmuxProjectCmd = &cobra.Command{
	Use:     "project [dir]",
	Aliases: []string{"proj"},
	Short:   "Create or attach to a session for a project directory",
	Long: `Create a new tmux session named after the project directory, or attach if
one already exists. Session name is derived from the directory basename.

If --layout is specified, the saved layout is applied after creating a new
session (not applied when attaching to an existing one). The layout must
already exist or an error is returned.`,
	Args: cobra.MaximumNArgs(1),
	RunE: hook.Wrap("tmux.project", runTmuxProject),
}

func runTmuxProject(cmd *cobra.Command, args []string) error {
	if !tmux.Available() {
		return fmt.Errorf("tmux not found in PATH — install tmux first")
	}

	var dir string
	if len(args) > 0 {
		dir = args[0]
	}

	// Pre-validate layout before doing any session work.
	layout := tmuxProjectLayout
	if layout != "" {
		if _, err := tmux.ReadLayout(layout); err != nil {
			return fmt.Errorf("layout %q not found — save it first with: mine tmux layout save %s", layout, layout)
		}
	}

	resolvedDir, sessionName, exists, err := tmux.ResolveProjectSession(dir)
	if err != nil {
		return err
	}

	if exists {
		fmt.Println()
		fmt.Printf("  Session %s already running — attaching\n", ui.Accent.Render(sessionName))
		fmt.Println()
		return tmux.AttachSession(sessionName)
	}

	// Create the session (detached) starting in the project directory.
	if _, err := tmux.NewSession(sessionName, resolvedDir); err != nil {
		return err
	}

	// Apply layout to the new session before attaching.
	if layout != "" {
		if err := tmux.LoadLayoutToSession(layout, sessionName); err != nil {
			return err
		}
	}

	ui.Ok(fmt.Sprintf("Session %s created", ui.Accent.Render(sessionName)))
	fmt.Println()
	return tmux.AttachSession(sessionName)
}

// --- mine tmux ls ---

var tmuxLsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List tmux sessions",
	RunE:    hook.Wrap("tmux.ls", runTmuxLs),
}

func runTmuxLs(_ *cobra.Command, _ []string) error {
	if !tmux.Available() {
		return fmt.Errorf("tmux not found in PATH")
	}

	sessions, err := tmux.ListSessions()
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No sessions running."))
		fmt.Println()
		return nil
	}

	return printSessionList(sessions)
}

// --- mine tmux attach ---

var tmuxAttachCmd = &cobra.Command{
	Use:     "attach [name]",
	Aliases: []string{"a"},
	Short:   "Attach or switch to a session (fuzzy match)",
	Args:    cobra.MaximumNArgs(1),
	RunE:    hook.Wrap("tmux.attach", runTmuxAttach),
}

func runTmuxAttach(_ *cobra.Command, args []string) error {
	if !tmux.Available() {
		return fmt.Errorf("tmux not found in PATH")
	}

	sessions, err := tmux.ListSessions()
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		return fmt.Errorf("no tmux sessions running")
	}

	// If name given, fuzzy-match it.
	if len(args) > 0 {
		s, err := tmux.FuzzyFindSession(args[0], sessions)
		if err != nil {
			return err
		}
		return tmux.AttachSession(s.Name)
	}

	// No name: use picker if TTY, else show list.
	if !tui.IsTTY() {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  Specify a session name or run interactively in a terminal."))
		return printSessionList(sessions)
	}

	items := make([]tui.Item, len(sessions))
	for i := range sessions {
		items[i] = sessions[i]
	}

	chosen, err := tui.Run(items,
		tui.WithTitle(ui.IconPick+"Attach to session"),
		tui.WithHeight(12),
	)
	if err != nil {
		return err
	}
	if chosen == nil {
		return nil
	}

	return tmux.AttachSession(chosen.Title())
}

// --- mine tmux kill ---

var tmuxKillCmd = &cobra.Command{
	Use:   "kill [name]",
	Short: "Kill a tmux session",
	Args:  cobra.MaximumNArgs(1),
	RunE:  hook.Wrap("tmux.kill", runTmuxKill),
}

func runTmuxKill(_ *cobra.Command, args []string) error {
	if !tmux.Available() {
		return fmt.Errorf("tmux not found in PATH")
	}

	sessions, err := tmux.ListSessions()
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		return fmt.Errorf("no tmux sessions running")
	}

	var target *tmux.Session

	if len(args) > 0 {
		// Name provided — fuzzy-match it.
		s, err := tmux.FuzzyFindSession(args[0], sessions)
		if err != nil {
			return err
		}
		target = s
	} else {
		// No name: use picker if TTY, else show list and require a name.
		if !tui.IsTTY() {
			fmt.Println()
			fmt.Println(ui.Muted.Render("  Specify a session name or run interactively in a terminal."))
			return printSessionList(sessions)
		}

		items := make([]tui.Item, len(sessions))
		for i := range sessions {
			items[i] = sessions[i]
		}

		chosen, err := tui.Run(items,
			tui.WithTitle(ui.IconPick+"Kill session"),
			tui.WithHeight(12),
		)
		if err != nil {
			return err
		}
		if chosen == nil {
			return nil // user canceled
		}

		for i := range sessions {
			if sessions[i].Name == chosen.Title() {
				target = &sessions[i]
				break
			}
		}
	}

	if err := tmux.KillSession(target.Name); err != nil {
		return err
	}

	ui.Ok(fmt.Sprintf("Killed session %s", ui.Accent.Render(target.Name)))
	fmt.Println()
	return nil
}

// --- mine tmux rename ---

var tmuxRenameCmd = &cobra.Command{
	Use:   "rename [old] [new]",
	Short: "Rename a tmux session",
	Long: `Rename a tmux session interactively or directly.

  2 args: rename directly without prompts
  1 arg:  fuzzy-match session by name, then prompt for new name
  0 args: open TUI picker to select session, then prompt for new name`,
	Args: cobra.MaximumNArgs(2),
	RunE: hook.Wrap("tmux.rename", runTmuxRename),
}

func runTmuxRename(_ *cobra.Command, args []string) error {
	if !tmux.Available() {
		return fmt.Errorf("tmux not found in PATH")
	}

	sessions, err := tmux.ListSessions()
	if err != nil {
		return err
	}

	if len(sessions) == 0 {
		return fmt.Errorf("no tmux sessions running")
	}

	// 2 args: direct rename, no prompts.
	if len(args) == 2 {
		oldName, newName := args[0], args[1]
		if newName == "" {
			return fmt.Errorf("new session name cannot be empty")
		}
		if err := tmux.RenameSession(oldName, newName); err != nil {
			return err
		}
		ui.Ok(fmt.Sprintf("Renamed session %s → %s", ui.Accent.Render(oldName), ui.Accent.Render(newName)))
		fmt.Println()
		return nil
	}

	// Resolve the old session name.
	var oldName string

	if len(args) == 1 {
		// 1 arg: fuzzy-match the session.
		s, err := tmux.FuzzyFindSession(args[0], sessions)
		if err != nil {
			return err
		}
		oldName = s.Name
	} else {
		// 0 args: use TUI picker if TTY, else require a name.
		if !tui.IsTTY() {
			fmt.Println()
			fmt.Println(ui.Muted.Render("  Specify a session name or run interactively in a terminal."))
			return printSessionList(sessions)
		}

		items := make([]tui.Item, len(sessions))
		for i := range sessions {
			items[i] = sessions[i]
		}

		chosen, err := tui.Run(items,
			tui.WithTitle(ui.IconPick+"Rename session"),
			tui.WithHeight(12),
		)
		if err != nil {
			return err
		}
		if chosen == nil {
			return nil // user canceled
		}
		oldName = chosen.Title()
	}

	// Prompt for new name.
	fmt.Fprintf(os.Stderr, "  New name for %s: ", ui.Accent.Render(oldName))
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	newName := strings.TrimSpace(line)

	if newName == "" {
		return fmt.Errorf("new session name cannot be empty")
	}

	if err := tmux.RenameSession(oldName, newName); err != nil {
		return err
	}

	ui.Ok(fmt.Sprintf("Renamed session %s → %s", ui.Accent.Render(oldName), ui.Accent.Render(newName)))
	fmt.Println()
	return nil
}

// --- mine tmux layout ---

var tmuxLayoutCmd = &cobra.Command{
	Use:   "layout",
	Short: "Save and restore window layouts",
	RunE:  hook.Wrap("tmux.layout", runTmuxLayoutHelp),
}

func runTmuxLayoutHelp(_ *cobra.Command, _ []string) error {
	fmt.Println()
	fmt.Println(ui.Title.Render("  Tmux Layouts"))
	fmt.Println()
	fmt.Printf("  %s  %s\n", ui.Accent.Render("mine tmux layout save <name>"), ui.Muted.Render("Save current layout"))
	fmt.Printf("  %s  %s\n", ui.Accent.Render("mine tmux layout load <name>"), ui.Muted.Render("Restore a layout"))
	fmt.Printf("  %s  %s\n", ui.Accent.Render("mine tmux layout ls"), ui.Muted.Render("List saved layouts"))
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
			tui.WithTitle(ui.IconPick+"Load layout"),
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
		tui.WithTitle(ui.IconPick+"Delete layout"),
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

func printSessionList(sessions []tmux.Session) error {
	fmt.Println()
	for _, s := range sessions {
		marker := " "
		if s.Attached {
			marker = ui.Success.Render("*")
		}

		w := "windows"
		if s.Windows == 1 {
			w = "window"
		}

		fmt.Printf("  %s %-20s %s\n",
			marker,
			ui.Accent.Render(s.Name),
			ui.Muted.Render(fmt.Sprintf("%d %s", s.Windows, w)),
		)
	}
	fmt.Println()
	return nil
}
