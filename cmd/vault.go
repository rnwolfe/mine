package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/rnwolfe/mine/internal/vault"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var vaultCmd = &cobra.Command{
	Use:   "vault",
	Short: "Lock away secrets — encrypted at rest with age",
	Long:  `Store API keys, tokens, and credentials encrypted at rest. Set MINE_VAULT_PASSPHRASE to skip the prompt.`,
	RunE:  hook.Wrap("vault", runVaultHelp),
}

var (
	vaultGetCopy   bool
	vaultImportFile string
	vaultExportFile string
)

func init() {
	vaultCmd.AddCommand(vaultSetCmd)
	vaultCmd.AddCommand(vaultGetCmd)
	vaultCmd.AddCommand(vaultListCmd)
	vaultCmd.AddCommand(vaultRmCmd)
	vaultCmd.AddCommand(vaultExportCmd)
	vaultCmd.AddCommand(vaultImportCmd)

	vaultGetCmd.Flags().BoolVar(&vaultGetCopy, "copy", false, "Copy secret to clipboard instead of printing")
	vaultExportCmd.Flags().StringVarP(&vaultExportFile, "output", "o", "", "Output file path (default: stdout)")
	vaultImportCmd.Flags().StringVarP(&vaultImportFile, "file", "f", "", "Input file path (default: stdin)")
}

func runVaultHelp(_ *cobra.Command, _ []string) error {
	fmt.Println()
	fmt.Println(ui.Title.Render("  "+ui.IconVault+" mine vault") + ui.Muted.Render(" — secrets, locked away"))
	fmt.Println()
	fmt.Println(ui.Muted.Render("  Your API keys and tokens, encrypted with age. Never stored in plaintext."))
	fmt.Println(ui.Muted.Render("  Set MINE_VAULT_PASSPHRASE to skip the password prompt."))
	fmt.Println()
	fmt.Println(ui.Accent.Render("  Commands:"))
	fmt.Println()
	fmt.Printf("    %s  Store or update a secret\n", ui.KeyStyle.Render("set <key> <value>"))
	fmt.Printf("    %s  Retrieve a secret\n", ui.KeyStyle.Render("get <key>"))
	fmt.Printf("    %s  List all stored keys\n", ui.KeyStyle.Render("list"))
	fmt.Printf("    %s  Delete a secret permanently\n", ui.KeyStyle.Render("rm <key>"))
	fmt.Printf("    %s  Export encrypted vault for backup\n", ui.KeyStyle.Render("export"))
	fmt.Printf("    %s  Restore vault from a backup\n", ui.KeyStyle.Render("import <file>"))
	fmt.Println()
	fmt.Println(ui.Accent.Render("  Examples:"))
	fmt.Println()
	fmt.Printf("    %s\n", ui.Muted.Render("mine vault set ai.claude.api_key sk-ant-..."))
	fmt.Printf("    %s\n", ui.Muted.Render("mine vault get ai.claude.api_key"))
	fmt.Printf("    %s\n", ui.Muted.Render("mine vault get ai.claude.api_key --copy"))
	fmt.Printf("    %s\n", ui.Muted.Render("mine vault list"))
	fmt.Printf("    %s\n", ui.Muted.Render("mine vault export -o vault-backup.age"))
	fmt.Println()
	ui.Tip("set MINE_VAULT_PASSPHRASE to avoid re-entering your passphrase every time")
	fmt.Println()
	return nil
}

// vaultSetCmd stores a secret in the vault.
var vaultSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Store or update a secret (encrypted)",
	Long:  `Encrypt and store a secret. If the key already exists, it is overwritten.`,
	Args:  cobra.ExactArgs(2),
	RunE:  hook.Wrap("vault.set", runVaultSet),
}

func runVaultSet(_ *cobra.Command, args []string) error {
	key, value := args[0], args[1]

	passphrase, err := readPassphrase(false)
	if err != nil {
		return err
	}

	v := vault.New(passphrase)
	if err := v.Set(key, value); err != nil {
		return formatVaultError(err)
	}

	ui.Ok(fmt.Sprintf("%s locked away in the vault", ui.Accent.Render(key)))
	return nil
}

// vaultGetCmd retrieves a secret from the vault.
var vaultGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Retrieve a secret",
	Long:  `Decrypt and print a secret to stdout. Use --copy to copy to clipboard.`,
	Args:  cobra.ExactArgs(1),
	RunE:  hook.Wrap("vault.get", runVaultGet),
}

func runVaultGet(_ *cobra.Command, args []string) error {
	key := args[0]

	passphrase, err := readPassphrase(false)
	if err != nil {
		return err
	}

	v := vault.New(passphrase)
	value, err := v.Get(key)
	if err != nil {
		return formatVaultError(err)
	}

	if vaultGetCopy {
		if err := copyToClipboard(value); err != nil {
			return fmt.Errorf("clipboard copy failed: %v — install xclip or xsel (Linux) or use pbcopy (macOS), or drop --copy to print instead", err)
		}
		ui.Ok(fmt.Sprintf("%s copied to clipboard", ui.Accent.Render(key)))
		return nil
	}

	fmt.Print(value)
	if !strings.HasSuffix(value, "\n") {
		fmt.Println()
	}
	return nil
}

// vaultListCmd lists all secret keys.
var vaultListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all stored secret keys (values stay hidden)",
	Long:  `Print all stored secret keys. Values are never displayed.`,
	Args:  cobra.NoArgs,
	RunE:  hook.Wrap("vault.list", runVaultList),
}

func runVaultList(_ *cobra.Command, _ []string) error {
	passphrase, err := readPassphrase(false)
	if err != nil {
		return err
	}

	v := vault.New(passphrase)
	keys, err := v.List()
	if err != nil {
		return formatVaultError(err)
	}

	fmt.Println()
	fmt.Println(ui.Title.Render("  Vault Secrets"))
	fmt.Println()

	if len(keys) == 0 {
		fmt.Println(ui.Muted.Render("  No secrets stored yet."))
		fmt.Println()
		fmt.Printf("  Get started: %s\n", ui.Accent.Render("mine vault set <key> <value>"))
		fmt.Println()
		return nil
	}

	for _, k := range keys {
		fmt.Printf("  %s %s\n", ui.IconVault, ui.KeyStyle.Render(k))
	}

	fmt.Println()
	fmt.Printf(ui.Muted.Render("  %d secret(s) stored in %s\n"), len(keys), ui.Muted.Render(v.Path()))
	fmt.Println()
	return nil
}

// vaultRmCmd deletes a secret.
var vaultRmCmd = &cobra.Command{
	Use:   "rm <key>",
	Short: "Permanently delete a secret",
	Long:  `Remove a secret from the vault permanently.`,
	Args:  cobra.ExactArgs(1),
	RunE:  hook.Wrap("vault.rm", runVaultRm),
}

func runVaultRm(_ *cobra.Command, args []string) error {
	key := args[0]

	passphrase, err := readPassphrase(false)
	if err != nil {
		return err
	}

	v := vault.New(passphrase)
	if err := v.Delete(key); err != nil {
		return formatVaultError(err)
	}

	ui.Ok(fmt.Sprintf("%s removed from the vault", ui.Accent.Render(key)))
	return nil
}

// vaultExportCmd exports the encrypted vault file.
var vaultExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export an encrypted vault backup",
	Long:  `Write the encrypted vault blob to a file or stdout. The export is still encrypted — safe to store anywhere.`,
	Args:  cobra.NoArgs,
	RunE:  hook.Wrap("vault.export", runVaultExport),
}

func runVaultExport(_ *cobra.Command, _ []string) error {
	passphrase, err := readPassphrase(false)
	if err != nil {
		return err
	}

	v := vault.New(passphrase)

	var w io.Writer
	if vaultExportFile != "" {
		// Validate path is not a directory and parent exists.
		if err := validateExportPath(vaultExportFile); err != nil {
			return err
		}
		f, err := os.Create(vaultExportFile)
		if err != nil {
			return fmt.Errorf("creating export file: %w", err)
		}
		defer f.Close()
		w = f
	} else {
		w = os.Stdout
	}

	if err := v.Export(w); err != nil {
		return formatVaultError(err)
	}

	if vaultExportFile != "" {
		fmt.Fprintf(os.Stderr, "%s Vault exported to %s\n", ui.IconOk, ui.Accent.Render(vaultExportFile))
	}
	return nil
}

// vaultImportCmd imports an encrypted vault backup.
var vaultImportCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import an encrypted vault backup",
	Long: `Replace the current vault with an encrypted backup.
The import file must be a valid age-encrypted vault blob created by 'mine vault export'.
The current vault is replaced entirely (no merge). This operation requires the same passphrase used during export.`,
	Args: cobra.MaximumNArgs(1),
	RunE: hook.Wrap("vault.import", runVaultImport),
}

func runVaultImport(_ *cobra.Command, args []string) error {
	passphrase, err := readPassphrase(false)
	if err != nil {
		return err
	}

	var r io.Reader
	importPath := vaultImportFile
	if len(args) > 0 {
		importPath = args[0]
	}

	if importPath != "" {
		// Validate the path.
		if err := validateImportPath(importPath); err != nil {
			return err
		}
		f, err := os.Open(importPath)
		if err != nil {
			return fmt.Errorf("opening import file: %w", err)
		}
		defer f.Close()
		r = f
	} else {
		r = os.Stdin
	}

	v := vault.New(passphrase)
	if err := v.Import(r); err != nil {
		return formatVaultError(err)
	}

	ui.Ok("Vault imported — all secrets restored")
	return nil
}

// readPassphrase reads the vault passphrase from MINE_VAULT_PASSPHRASE env var
// or prompts the user securely (no echo).
func readPassphrase(confirm bool) (string, error) {
	if p := os.Getenv("MINE_VAULT_PASSPHRASE"); p != "" {
		return p, nil
	}

	// Prompt interactively.
	if !term.IsTerminal(int(syscall.Stdin)) {
		return "", fmt.Errorf("vault passphrase required — set MINE_VAULT_PASSPHRASE or run interactively")
	}

	fmt.Fprint(os.Stderr, ui.Muted.Render("  Vault passphrase: "))
	passBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("reading passphrase: %w", err)
	}

	passphrase := strings.TrimSpace(string(passBytes))
	if passphrase == "" {
		return "", fmt.Errorf("passphrase can't be empty — set MINE_VAULT_PASSPHRASE or type it when prompted")
	}

	if confirm {
		fmt.Fprint(os.Stderr, ui.Muted.Render("  Confirm passphrase: "))
		confirmBytes, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Fprintln(os.Stderr)
		if err != nil {
			return "", fmt.Errorf("reading passphrase confirmation: %w", err)
		}
		if string(passBytes) != string(confirmBytes) {
			return "", fmt.Errorf("passphrases do not match")
		}
	}

	return passphrase, nil
}

// formatVaultError wraps vault errors with actionable messages.
func formatVaultError(err error) error {
	if errors.Is(err, vault.ErrWrongPassphrase) {
		return fmt.Errorf("wrong passphrase — double-check MINE_VAULT_PASSPHRASE or try again interactively")
	}
	if errors.Is(err, vault.ErrCorruptedVault) {
		return fmt.Errorf("vault appears corrupted — restore from a backup with `mine vault import <file>`")
	}
	return err
}

// copyToClipboard copies text to the system clipboard using platform tools.
func copyToClipboard(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		// Try xclip first, then xsel, then wl-copy (Wayland)
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.Command("wl-copy")
		} else {
			return fmt.Errorf("no clipboard tool found (install xclip, xsel, or wl-copy)")
		}
	case "windows":
		cmd = exec.Command("clip")
	default:
		return fmt.Errorf("clipboard not supported on %s", runtime.GOOS)
	}

	cmd.Stdin = strings.NewReader(text)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// validateExportPath checks the export destination path is valid.
func validateExportPath(path string) error {
	clean := filepath.Clean(path)
	// Check parent directory exists.
	dir := filepath.Dir(clean)
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("export directory does not exist: %s", dir)
	}
	return nil
}

// validateImportPath checks the import source path is valid and exists.
func validateImportPath(path string) error {
	clean := filepath.Clean(path)
	info, err := os.Stat(clean)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("import file not found: %s", path)
		}
		return fmt.Errorf("checking import file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("import path must be a file, not a directory: %s", path)
	}
	return nil
}
