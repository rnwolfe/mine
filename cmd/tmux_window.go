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

var tmuxWindowSession string

// Injectable for testing.
var (
	listWindowsFunc   = tmux.ListWindows
	newWindowFunc     = tmux.NewWindow
	killWindowFunc    = tmux.KillWindow
	renameWindowFunc  = tmux.RenameWindow
	currentSessionFunc = tmux.CurrentSession
)

func init() {
	tmuxCmd.AddCommand(tmuxWindowCmd)
	tmuxWindowCmd.AddCommand(tmuxWindowLsCmd)
	tmuxWindowCmd.AddCommand(tmuxWindowNewCmd)
	tmuxWindowCmd.AddCommand(tmuxWindowKillCmd)
	tmuxWindowCmd.AddCommand(tmuxWindowRenameCmd)

	// --session flag is inherited by all window subcommands.
	tmuxWindowCmd.PersistentFlags().StringVar(&tmuxWindowSession, "session", "", "Target session (defaults to current)")
}

// resolveWindowSession returns the session name from the --session flag or
// the current tmux session. Errors if neither is available.
func resolveWindowSession(flag string) (string, error) {
	if flag != "" {
		return flag, nil
	}
	if !tmux.InsideTmux() {
		return "", fmt.Errorf("not inside a tmux session — use %s to target a specific session",
			ui.Accent.Render("--session <name>"))
	}
	return currentSessionFunc()
}

// --- mine tmux window ---

var tmuxWindowCmd = &cobra.Command{
	Use:   "window",
	Short: "Manage windows within a tmux session",
	Long:  `List, create, kill, and rename windows in a tmux session.`,
	RunE:  hook.Wrap("tmux.window", runTmuxWindowHelp),
}

func runTmuxWindowHelp(_ *cobra.Command, _ []string) error {
	fmt.Println()
	fmt.Println(ui.Title.Render("  Tmux Windows"))
	fmt.Println()
	fmt.Printf("  %s  %s\n", ui.Accent.Render("mine tmux window ls"), ui.Muted.Render("List windows in current session"))
	fmt.Printf("  %s  %s\n", ui.Accent.Render("mine tmux window new <name>"), ui.Muted.Render("Create a new window"))
	fmt.Printf("  %s  %s\n", ui.Accent.Render("mine tmux window kill [name]"), ui.Muted.Render("Kill a window"))
	fmt.Printf("  %s  %s\n", ui.Accent.Render("mine tmux window rename [old] [new]"), ui.Muted.Render("Rename a window"))
	fmt.Println()
	return nil
}

// --- mine tmux window ls ---

var tmuxWindowLsCmd = &cobra.Command{
	Use:     "ls",
	Aliases: []string{"list"},
	Short:   "List windows in the current session",
	RunE:    hook.Wrap("tmux.window.ls", runTmuxWindowLs),
}

func runTmuxWindowLs(_ *cobra.Command, _ []string) error {
	if !tmux.Available() {
		return fmt.Errorf("tmux not found in PATH — install tmux first")
	}

	session, err := resolveWindowSession(tmuxWindowSession)
	if err != nil {
		return err
	}

	windows, err := listWindowsFunc(session)
	if err != nil {
		return err
	}

	return printWindowList(session, windows)
}

// --- mine tmux window new ---

var tmuxWindowNewCmd = &cobra.Command{
	Use:   "new <name>",
	Short: "Create a new window in the current session",
	Args:  cobra.ExactArgs(1),
	RunE:  hook.Wrap("tmux.window.new", runTmuxWindowNew),
}

func runTmuxWindowNew(_ *cobra.Command, args []string) error {
	if !tmux.Available() {
		return fmt.Errorf("tmux not found in PATH — install tmux first")
	}

	session, err := resolveWindowSession(tmuxWindowSession)
	if err != nil {
		return err
	}

	name := args[0]
	if err := newWindowFunc(session, name); err != nil {
		return err
	}

	ui.Ok(fmt.Sprintf("Window %s created in session %s",
		ui.Accent.Render(name), ui.Accent.Render(session)))
	fmt.Println()
	return nil
}

// --- mine tmux window kill ---

var tmuxWindowKillCmd = &cobra.Command{
	Use:   "kill [name]",
	Short: "Kill a window in the current session",
	Args:  cobra.MaximumNArgs(1),
	RunE:  hook.Wrap("tmux.window.kill", runTmuxWindowKill),
}

func runTmuxWindowKill(_ *cobra.Command, args []string) error {
	if !tmux.Available() {
		return fmt.Errorf("tmux not found in PATH — install tmux first")
	}

	session, err := resolveWindowSession(tmuxWindowSession)
	if err != nil {
		return err
	}

	windows, err := listWindowsFunc(session)
	if err != nil {
		return err
	}

	if len(windows) == 0 {
		return fmt.Errorf("no windows in session %q", session)
	}

	var target *tmux.Window

	if len(args) > 0 {
		name := args[0]
		w := tmux.FindWindowByName(name, windows)
		if w == nil {
			return fmt.Errorf("no window named %q in session %q", name, session)
		}
		target = w
	} else {
		if !tui.IsTTY() {
			fmt.Println()
			fmt.Println(ui.Muted.Render("  Specify a window name or run interactively in a terminal."))
			return printWindowList(session, windows)
		}

		items := make([]tui.Item, len(windows))
		for i := range windows {
			items[i] = windows[i]
		}

		chosen, err := tui.Run(items,
			tui.WithTitle(ui.IconMine+"Kill window"),
			tui.WithHeight(12),
		)
		if err != nil {
			return err
		}
		if chosen == nil {
			return nil // user canceled
		}

		for i := range windows {
			if windows[i].Name == chosen.Title() {
				target = &windows[i]
				break
			}
		}
	}

	if err := killWindowFunc(session, target.Name); err != nil {
		return err
	}

	ui.Ok(fmt.Sprintf("Killed window %s in session %s",
		ui.Accent.Render(target.Name), ui.Accent.Render(session)))
	fmt.Println()
	return nil
}

// --- mine tmux window rename ---

var tmuxWindowRenameCmd = &cobra.Command{
	Use:   "rename [old] [new]",
	Short: "Rename a window in the current session",
	Long: `Rename a window interactively or directly.

  2 args: rename directly without prompts
  1 arg:  select window by name, then prompt for new name
  0 args: open TUI picker to select window, then prompt for new name`,
	Args: cobra.MaximumNArgs(2),
	RunE: hook.Wrap("tmux.window.rename", runTmuxWindowRename),
}

func runTmuxWindowRename(_ *cobra.Command, args []string) error {
	if !tmux.Available() {
		return fmt.Errorf("tmux not found in PATH — install tmux first")
	}

	session, err := resolveWindowSession(tmuxWindowSession)
	if err != nil {
		return err
	}

	// 2 args: direct rename, no prompts.
	if len(args) == 2 {
		oldName, newName := args[0], args[1]
		if newName == "" {
			return fmt.Errorf("new window name cannot be empty")
		}
		if err := renameWindowFunc(session, oldName, newName); err != nil {
			return err
		}
		ui.Ok(fmt.Sprintf("Renamed window %s → %s in session %s",
			ui.Accent.Render(oldName), ui.Accent.Render(newName), ui.Accent.Render(session)))
		fmt.Println()
		return nil
	}

	windows, err := listWindowsFunc(session)
	if err != nil {
		return err
	}

	if len(windows) == 0 {
		return fmt.Errorf("no windows in session %q", session)
	}

	var oldName string

	if len(args) == 1 {
		name := args[0]
		w := tmux.FindWindowByName(name, windows)
		if w == nil {
			return fmt.Errorf("no window named %q in session %q", name, session)
		}
		oldName = w.Name
	} else {
		// 0 args: use TUI picker if TTY, else list and return.
		if !tui.IsTTY() {
			fmt.Println()
			fmt.Println(ui.Muted.Render("  Specify a window name or run interactively in a terminal."))
			return printWindowList(session, windows)
		}

		items := make([]tui.Item, len(windows))
		for i := range windows {
			items[i] = windows[i]
		}

		chosen, err := tui.Run(items,
			tui.WithTitle(ui.IconMine+"Rename window"),
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
		return fmt.Errorf("new window name cannot be empty")
	}

	if err := renameWindowFunc(session, oldName, newName); err != nil {
		return err
	}

	ui.Ok(fmt.Sprintf("Renamed window %s → %s in session %s",
		ui.Accent.Render(oldName), ui.Accent.Render(newName), ui.Accent.Render(session)))
	fmt.Println()
	return nil
}

// --- helpers ---

func printWindowList(session string, windows []tmux.Window) error {
	if len(windows) == 0 {
		fmt.Println()
		fmt.Printf("  %s\n", ui.Muted.Render("No windows in session "+session+"."))
		fmt.Printf("  Create one: %s\n", ui.Accent.Render("mine tmux window new <name>"))
		fmt.Println()
		return nil
	}

	fmt.Println()
	for _, w := range windows {
		marker := " "
		if w.Active {
			marker = ui.Success.Render("*")
		}
		fmt.Printf("  %s %-20s %s\n",
			marker,
			ui.Accent.Render(w.Name),
			ui.Muted.Render(fmt.Sprintf("index %d", w.Index)),
		)
	}
	fmt.Println()
	return nil
}
