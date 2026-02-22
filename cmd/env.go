package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"
	"syscall"

	"github.com/rnwolfe/mine/internal/env"
	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/store"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/rnwolfe/mine/internal/vault"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	envReveal    bool
	envShellType string
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Encrypted env profiles — no more .env file sprawl",
	Long:  "Per-project encrypted env profiles. Set secrets safely, export to your shell, and never commit credentials again.",
	RunE:  hook.Wrap("env", runEnvBare),
}

func init() {
	rootCmd.AddCommand(envCmd)

	envCmd.AddCommand(envShowCmd)
	envCmd.AddCommand(envSetCmd)
	envCmd.AddCommand(envUnsetCmd)
	envCmd.AddCommand(envDiffCmd)
	envCmd.AddCommand(envSwitchCmd)
	envCmd.AddCommand(envExportCmd)
	envCmd.AddCommand(envTemplateCmd)
	envCmd.AddCommand(envInjectCmd)
	envCmd.AddCommand(envEditCmd)

	envShowCmd.Flags().BoolVar(&envReveal, "reveal", false, "Show raw values (default: masked)")
	envExportCmd.Flags().StringVar(&envShellType, "shell", "posix", "Export format: posix or fish")
}

var envShowCmd = &cobra.Command{
	Use:   "show [profile]",
	Short: "Show env vars for a profile (values masked by default)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  hook.Wrap("env.show", runEnvShow),
}

var envSetCmd = &cobra.Command{
	Use:   "set KEY=VALUE | KEY",
	Short: "Set a variable in the active profile",
	Args:  cobra.ExactArgs(1),
	RunE:  hook.Wrap("env.set", runEnvSet),
}

var envUnsetCmd = &cobra.Command{
	Use:   "unset KEY",
	Short: "Remove a variable from the active profile",
	Args:  cobra.ExactArgs(1),
	RunE:  hook.Wrap("env.unset", runEnvUnset),
}

var envDiffCmd = &cobra.Command{
	Use:   "diff <profile-a> <profile-b>",
	Short: "See what's different between two profiles",
	Args:  cobra.ExactArgs(2),
	RunE:  hook.Wrap("env.diff", runEnvDiff),
}

var envSwitchCmd = &cobra.Command{
	Use:   "switch <profile>",
	Short: "Switch active profile for the current project",
	Args:  cobra.ExactArgs(1),
	RunE:  hook.Wrap("env.switch", runEnvSwitch),
}

var envExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Print shell export lines for the active profile",
	Args:  cobra.NoArgs,
	RunE:  hook.Wrap("env.export", runEnvExport),
}

var envTemplateCmd = &cobra.Command{
	Use:   "template",
	Short: "Generate a .env.example from the active profile",
	Args:  cobra.NoArgs,
	RunE:  hook.Wrap("env.template", runEnvTemplate),
}

var envInjectCmd = &cobra.Command{
	Use:   "inject -- <command> [args...]",
	Short: "Run a command with env vars injected",
	Args:  cobra.ArbitraryArgs,
	RunE:  hook.Wrap("env.inject", runEnvInject),
}

var envEditCmd = &cobra.Command{
	Use:   "edit [profile]",
	Short: "Open a profile in $EDITOR for bulk editing",
	Args:  cobra.MaximumNArgs(1),
	RunE:  hook.Wrap("env.edit", runEnvEdit),
}

func runEnvBare(_ *cobra.Command, _ []string) error {
	return showActive(false)
}

func runEnvShow(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		return showActive(envReveal)
	}
	m, projectPath, err := envManager()
	if err != nil {
		return err
	}
	defer m.Close()
	vars, err := m.manager.LoadProfile(projectPath, args[0])
	if err != nil {
		return err
	}
	return printEnvProfile(args[0], vars, envReveal)
}

func runEnvSet(_ *cobra.Command, args []string) error {
	key, value, err := parseSetArg(args[0])
	if err != nil {
		return err
	}
	m, projectPath, err := envManager()
	if err != nil {
		return err
	}
	defer m.Close()
	profile, err := m.manager.ActiveProfile(projectPath)
	if err != nil {
		return err
	}
	if err := m.manager.SetVar(projectPath, profile, key, value); err != nil {
		return err
	}
	ui.Ok(fmt.Sprintf("%s saved to profile %s", ui.Accent.Render(key), ui.Muted.Render(profile)))
	return nil
}

func runEnvUnset(_ *cobra.Command, args []string) error {
	key := strings.TrimSpace(args[0])
	if err := env.ValidateKey(key); err != nil {
		return err
	}
	m, projectPath, err := envManager()
	if err != nil {
		return err
	}
	defer m.Close()
	profile, err := m.manager.ActiveProfile(projectPath)
	if err != nil {
		return err
	}
	if err := m.manager.UnsetVar(projectPath, profile, key); err != nil {
		return err
	}
	ui.Ok(fmt.Sprintf("%s removed from profile %s", ui.Accent.Render(key), ui.Muted.Render(profile)))
	return nil
}

func runEnvDiff(_ *cobra.Command, args []string) error {
	m, projectPath, err := envManager()
	if err != nil {
		return err
	}
	defer m.Close()
	d, err := m.manager.Diff(projectPath, args[0], args[1])
	if err != nil {
		return err
	}
	fmt.Printf("  %s %s vs %s\n", ui.Title.Render("Diff"), ui.Accent.Render(args[0]), ui.Accent.Render(args[1]))
	fmt.Println()

	for _, k := range d.Added {
		fmt.Printf("  %s %s\n", ui.Success.Render("+"), ui.Accent.Render(k))
	}
	for _, k := range d.Removed {
		fmt.Printf("  %s %s\n", ui.Error.Render("-"), ui.Accent.Render(k))
	}
	for _, k := range d.Changed {
		fmt.Printf("  %s %s\n", ui.Warning.Render("~"), ui.Accent.Render(k))
	}
	if len(d.Added) == 0 && len(d.Removed) == 0 && len(d.Changed) == 0 {
		fmt.Printf("  %s\n", ui.Muted.Render("No differences."))
	}
	return nil
}

func runEnvSwitch(_ *cobra.Command, args []string) error {
	m, projectPath, err := envManager()
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.manager.SwitchProfile(projectPath, args[0]); err != nil {
		return err
	}
	ui.Ok(fmt.Sprintf("Switched to profile %s", ui.Accent.Render(args[0])))
	fmt.Printf("  Run %s to load it into your shell.\n", ui.Accent.Render("menv"))
	return nil
}

func runEnvExport(_ *cobra.Command, _ []string) error {
	m, projectPath, err := envManager()
	if err != nil {
		return err
	}
	defer m.Close()
	profile, err := m.manager.ActiveProfile(projectPath)
	if err != nil {
		return err
	}
	shellName := strings.ToLower(envShellType)
	if shellName != "posix" && shellName != "fish" {
		return fmt.Errorf("unknown shell %q — use --shell posix or --shell fish", shellName)
	}
	lines, err := m.manager.ExportLines(projectPath, profile, shellName)
	if err != nil {
		return err
	}
	fmt.Println(strings.Join(lines, "\n"))
	return nil
}

func runEnvTemplate(_ *cobra.Command, _ []string) error {
	m, projectPath, err := envManager()
	if err != nil {
		return err
	}
	defer m.Close()
	profile, err := m.manager.ActiveProfile(projectPath)
	if err != nil {
		return err
	}
	keys, err := m.manager.TemplateKeys(projectPath, profile)
	if err != nil {
		return err
	}
	for _, k := range keys {
		fmt.Printf("%s=\n", k)
	}
	return nil
}

func runEnvInject(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command provided — usage: mine env inject -- <command> [args...]")
	}
	m, projectPath, err := envManager()
	if err != nil {
		return err
	}
	defer m.Close()
	profile, err := m.manager.ActiveProfile(projectPath)
	if err != nil {
		return err
	}
	vars, err := m.manager.LoadProfile(projectPath, profile)
	if err != nil {
		return err
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = mergedEnv(os.Environ(), vars)
	return cmd.Run()
}

func runEnvEdit(_ *cobra.Command, args []string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		return fmt.Errorf(
			"$EDITOR is not set\n\nSet it in your shell profile:\n  export EDITOR=vim\n\nOr set individual vars without an editor:\n  %s",
			ui.Accent.Render("mine env set KEY=VALUE"),
		)
	}

	m, projectPath, err := envManager()
	if err != nil {
		return err
	}
	defer m.Close()

	var profile string
	if len(args) > 0 {
		profile = args[0]
	} else {
		profile, err = m.manager.ActiveProfile(projectPath)
		if err != nil {
			return err
		}
	}

	vars, err := m.manager.LoadProfile(projectPath, profile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if len(args) > 0 {
				return fmt.Errorf(
					"profile %q does not exist\n\nCreate variables in the active profile first with:\n  %s",
					profile, ui.Accent.Render("mine env set KEY=VALUE"),
				)
			}
			vars = map[string]string{}
		} else {
			return err
		}
	}

	// Create temp file; deferred cleanup zero-fills then removes it on all paths.
	tmp, err := os.CreateTemp(os.TempDir(), "mine-env-*.env")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Truncate(tmpPath, 0)
		_ = os.Remove(tmpPath)
	}()
	if err := os.Chmod(tmpPath, 0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("setting temp file permissions: %w", err)
	}

	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(vars[k])
		sb.WriteString("\n")
	}
	if _, err := tmp.WriteString(sb.String()); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	parts := strings.Fields(editor)
	editorArgs := append(parts[1:], tmpPath)
	cmd := exec.Command(parts[0], editorArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("editor exited with an error — no changes saved: %w", err)
	}

	content, err := os.ReadFile(tmpPath)
	if err != nil {
		return fmt.Errorf("reading temp file after edit: %w", err)
	}

	edited, invalidKeys := parseEnvFile(string(content))
	if len(invalidKeys) > 0 {
		return fmt.Errorf(
			"invalid key names in edited file: %s\n\nNo changes saved. Keys must match [A-Za-z_][A-Za-z0-9_]*",
			strings.Join(invalidKeys, ", "),
		)
	}

	if err := m.manager.SaveProfile(projectPath, profile, edited); err != nil {
		return fmt.Errorf("saving profile: %w", err)
	}

	ui.Ok(fmt.Sprintf("Profile %s saved (%d var(s))", ui.Accent.Render(profile), len(edited)))
	return nil
}

// parseEnvFile parses KEY=VALUE lines from env file content.
// Blank lines and lines starting with # are ignored.
// Returns the parsed vars and a list of invalid key names found.
func parseEnvFile(content string) (map[string]string, []string) {
	vars := make(map[string]string)
	var invalidKeys []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, _ := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if err := env.ValidateKey(key); err != nil {
			invalidKeys = append(invalidKeys, key)
			continue
		}
		vars[key] = val
	}
	return vars, invalidKeys
}

func showActive(reveal bool) error {
	m, projectPath, err := envManager()
	if err != nil {
		return err
	}
	defer m.Close()
	profile, vars, err := m.manager.CurrentProfile(projectPath)
	if err != nil {
		return err
	}
	return printEnvProfile(profile, vars, reveal)
}

func printEnvProfile(profile string, vars map[string]string, reveal bool) error {
	fmt.Printf("  %s %s\n", ui.Title.Render("Profile"), ui.Accent.Render(profile))
	if len(vars) == 0 {
		fmt.Printf("  %s\n", ui.Muted.Render("No variables set."))
		return nil
	}
	fmt.Println()
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := vars[k]
		if !reveal {
			v = env.MaskValue(v)
		}
		fmt.Printf("  %s=%s\n", k, v)
	}
	return nil
}

type envSession struct {
	manager *env.Manager
	db      *store.DB
}

func (s *envSession) Close() {
	_ = s.db.Close()
}

func envManager() (*envSession, string, error) {
	passphrase, err := readEnvPassphrase()
	if err != nil {
		return nil, "", err
	}
	db, err := store.Open()
	if err != nil {
		return nil, "", err
	}
	m := env.New(db.Conn(), passphrase)
	projectPath, err := m.ProjectPath()
	if err != nil {
		_ = db.Close()
		return nil, "", err
	}
	return &envSession{manager: m, db: db}, projectPath, nil
}

// readEnvPassphrase reads the env passphrase using the following resolution order:
//  1. MINE_ENV_PASSPHRASE env var (always wins)
//  2. MINE_VAULT_PASSPHRASE env var
//  3. OS keychain (via vaultKeychainStore, shared with vault commands)
//  4. Interactive TTY prompt
func readEnvPassphrase() (string, error) {
	if p := os.Getenv("MINE_ENV_PASSPHRASE"); p != "" {
		return p, nil
	}
	if p := os.Getenv("MINE_VAULT_PASSPHRASE"); p != "" {
		return p, nil
	}

	// Check OS keychain before prompting.
	if p, err := vaultKeychainStore.Get(vault.ServiceName); err == nil && p != "" {
		return p, nil
	}

	if !term.IsTerminal(int(syscall.Stdin)) {
		return "", fmt.Errorf("env passphrase required — set MINE_ENV_PASSPHRASE, MINE_VAULT_PASSPHRASE, run %s, or run interactively",
			ui.Accent.Render("mine vault unlock"))
	}
	fmt.Fprint(os.Stderr, ui.Muted.Render("  Env passphrase: "))
	passBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("reading passphrase: %w", err)
	}
	passphrase := strings.TrimSpace(string(passBytes))
	if passphrase == "" {
		return "", fmt.Errorf("passphrase can't be empty — set MINE_ENV_PASSPHRASE or type it when prompted")
	}
	return passphrase, nil
}

func parseSetArg(arg string) (string, string, error) {
	key, val, hasValue := strings.Cut(arg, "=")
	key = strings.TrimSpace(key)
	if err := env.ValidateKey(key); err != nil {
		return "", "", err
	}
	if hasValue {
		return key, val, nil
	}
	// If value omitted, read from stdin without echo when interactive.
	if term.IsTerminal(int(syscall.Stdin)) {
		fmt.Fprint(os.Stderr, ui.Muted.Render("  Value: "))
		b, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", "", err
		}
		return key, string(b), nil
	}
	b, err := io.ReadAll(os.Stdin)
	if err != nil {
		return "", "", err
	}
	value := strings.TrimRight(string(b), "\r\n")
	if value == "" {
		return "", "", errors.New("value is required — use KEY=VALUE or pipe the value: echo 'secret' | mine env set KEY")
	}
	return key, value, nil
}

func mergedEnv(base []string, overrides map[string]string) []string {
	outMap := make(map[string]string, len(base)+len(overrides))
	for _, entry := range base {
		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		outMap[key] = value
	}
	for k, v := range overrides {
		outMap[k] = v
	}
	keys := make([]string, 0, len(outMap))
	for k := range outMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k+"="+outMap[k])
	}
	return out
}
