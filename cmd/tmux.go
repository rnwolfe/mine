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
	tmuxCmd.AddCommand(tmuxLayoutCmd)

	tmuxLayoutCmd.AddCommand(tmuxLayoutSaveCmd)
	tmuxLayoutCmd.AddCommand(tmuxLayoutLoadCmd)
	tmuxLayoutCmd.AddCommand(tmuxLayoutLsCmd)
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

	resolved, err := tmux.NewSession(name)
	if err != nil {
		return err
	}

	ui.Ok(fmt.Sprintf("Session %s created", ui.Accent.Render(resolved)))
	fmt.Printf("  Attach: %s\n", ui.Muted.Render("mine tmux attach "+resolved))
	fmt.Println()
	return nil
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
	Args:  cobra.ExactArgs(1),
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

	s, err := tmux.FuzzyFindSession(args[0], sessions)
	if err != nil {
		return err
	}

	if err := tmux.KillSession(s.Name); err != nil {
		return err
	}

	ui.Ok(fmt.Sprintf("Killed session %s", ui.Accent.Render(s.Name)))
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
	Use:   "load <name>",
	Short: "Restore a saved layout",
	Args:  cobra.ExactArgs(1),
	RunE:  hook.Wrap("tmux.layout.load", runTmuxLayoutLoad),
}

func runTmuxLayoutLoad(_ *cobra.Command, args []string) error {
	if !tmux.Available() {
		return fmt.Errorf("tmux not found in PATH")
	}
	if !tmux.InsideTmux() {
		return fmt.Errorf("not inside a tmux session — attach first")
	}

	name := args[0]
	if err := tmux.LoadLayout(name); err != nil {
		return err
	}

	ui.Ok(fmt.Sprintf("Layout %s restored", ui.Accent.Render(name)))
	fmt.Println()
	return nil
}

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

		fmt.Printf("  %s  %s  %s\n",
			ui.Accent.Render(fmt.Sprintf("%-16s", name)),
			ui.Muted.Render(fmt.Sprintf("%-12s", windows)),
			ui.Muted.Render("["+strings.Join(windowNames, ", ")+"]"),
		)
	}
	fmt.Println()
	return nil
}

// --- helpers ---

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
