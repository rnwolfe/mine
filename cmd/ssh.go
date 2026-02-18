package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/rnwolfe/mine/internal/hook"
	"github.com/rnwolfe/mine/internal/ssh"
	"github.com/rnwolfe/mine/internal/tui"
	"github.com/rnwolfe/mine/internal/ui"
	"github.com/spf13/cobra"
)

var sshCmd = &cobra.Command{
	Use:   "ssh",
	Short: "SSH config management and connection helpers",
	Long:  `Manage SSH hosts, keys, tunnels, and connections.`,
	RunE:  hook.Wrap("ssh", runSSH),
}

func init() {
	rootCmd.AddCommand(sshCmd)
	sshCmd.AddCommand(sshHostsCmd)
	sshCmd.AddCommand(sshAddCmd)
	sshCmd.AddCommand(sshRemoveCmd)
	sshCmd.AddCommand(sshKeygenCmd)
	sshCmd.AddCommand(sshCopyIDCmd)
	sshCmd.AddCommand(sshKeysCmd)
	sshCmd.AddCommand(sshTunnelCmd)
}

// --- mine ssh (bare) — fuzzy host picker ---

func runSSH(_ *cobra.Command, _ []string) error {
	hosts, err := ssh.ReadHosts()
	if err != nil {
		return err
	}

	if len(hosts) == 0 {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No SSH hosts configured."))
		fmt.Printf("  Add one: %s\n", ui.Accent.Render("mine ssh add <alias>"))
		fmt.Println()
		return nil
	}

	// Non-TTY fallback: plain list.
	if !tui.IsTTY() {
		return printSSHHostList(hosts)
	}

	// Interactive fuzzy picker.
	items := make([]tui.Item, len(hosts))
	for i := range hosts {
		items[i] = hosts[i]
	}

	chosen, err := tui.Run(items,
		tui.WithTitle(ui.IconVault+" Select SSH host"),
		tui.WithHeight(12),
	)
	if err != nil {
		return err
	}
	if chosen == nil {
		return nil // user canceled
	}

	return sshConnectFunc(chosen.Title())
}

// sshConnectFunc is the replaceable function for connecting to an SSH host.
var sshConnectFunc = sshConnectReal

func sshConnectReal(alias string) error {
	cmd := exec.Command("ssh", alias)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// --- mine ssh hosts ---

var sshHostsCmd = &cobra.Command{
	Use:   "hosts",
	Short: "List configured SSH hosts",
	RunE:  hook.Wrap("ssh.hosts", runSSHHosts),
}

func runSSHHosts(_ *cobra.Command, _ []string) error {
	hosts, err := ssh.ReadHosts()
	if err != nil {
		return err
	}
	if len(hosts) == 0 {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No SSH hosts configured."))
		fmt.Printf("  Add one: %s\n", ui.Accent.Render("mine ssh add <alias>"))
		fmt.Println()
		return nil
	}
	return printSSHHostList(hosts)
}

func printSSHHostList(hosts []ssh.Host) error {
	fmt.Println()
	fmt.Println(ui.Title.Render("  SSH Hosts"))
	fmt.Println()
	for _, h := range hosts {
		alias := ui.Accent.Render(fmt.Sprintf("  %-20s", h.Alias))
		desc := ui.Muted.Render(h.Description())
		fmt.Printf("%s %s\n", alias, desc)
	}
	fmt.Println()
	return nil
}

// --- mine ssh add ---

var sshAddCmd = &cobra.Command{
	Use:   "add [alias]",
	Short: "Add a host to ~/.ssh/config",
	Long: `Interactively generate a Host block and append it to ~/.ssh/config.
If alias is provided, it is used as the Host entry name.`,
	Args: cobra.MaximumNArgs(1),
	RunE: hook.Wrap("ssh.add", runSSHAdd),
}

func runSSHAdd(_ *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	prompt := func(label, defaultVal string) (string, error) {
		if defaultVal != "" {
			fmt.Printf("  %s [%s]: ", label, ui.Muted.Render(defaultVal))
		} else {
			fmt.Printf("  %s: ", label)
		}
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			return defaultVal, nil
		}
		return line, nil
	}

	fmt.Println()
	fmt.Println(ui.Title.Render("  Add SSH Host"))
	fmt.Println()

	alias := ""
	if len(args) > 0 {
		alias = args[0]
		fmt.Printf("  Alias: %s\n", ui.Accent.Render(alias))
	} else {
		var err error
		alias, err = prompt("Alias (e.g. myserver)", "")
		if err != nil {
			return err
		}
		if alias == "" {
			return fmt.Errorf("alias is required")
		}
	}

	// Validate alias: must not contain whitespace or wildcard characters.
	if strings.ContainsAny(alias, " \t\n?*") {
		return fmt.Errorf("invalid alias %q: must not contain whitespace or wildcard characters", alias)
	}

	hostname, err := prompt("HostName (IP or FQDN)", alias)
	if err != nil {
		return err
	}

	user, err := prompt("User", "")
	if err != nil {
		return err
	}

	port, err := prompt("Port", "22")
	if err != nil {
		return err
	}
	if port == "22" {
		port = "" // omit default port
	}

	keyFile, err := prompt("IdentityFile", "")
	if err != nil {
		return err
	}

	h := ssh.Host{
		Alias:    alias,
		Hostname: hostname,
		User:     user,
		Port:     port,
		KeyFile:  keyFile,
	}

	if err := ssh.AppendHost(h); err != nil {
		return err
	}

	fmt.Println()
	ui.Ok(fmt.Sprintf("Added host %q to %s", alias, ssh.ConfigPath()))
	fmt.Println()
	return nil
}

// --- mine ssh remove ---

var sshRemoveCmd = &cobra.Command{
	Use:   "remove <alias>",
	Short: "Remove a host from ~/.ssh/config",
	Args:  cobra.ExactArgs(1),
	RunE:  hook.Wrap("ssh.remove", runSSHRemove),
}

func runSSHRemove(_ *cobra.Command, args []string) error {
	alias := args[0]
	if err := ssh.RemoveHost(alias); err != nil {
		return err
	}
	ui.Ok(fmt.Sprintf("Removed host %q from %s", alias, ssh.ConfigPath()))
	return nil
}

// --- mine ssh keygen ---

var sshKeygenCmd = &cobra.Command{
	Use:   "keygen [name]",
	Short: "Generate an ed25519 SSH key pair",
	Long: `Generate an ed25519 SSH key pair with sensible defaults.
The key is saved to ~/.ssh/<name> (and ~/.ssh/<name>.pub).
Defaults to "id_ed25519" if name is not provided.`,
	Args: cobra.MaximumNArgs(1),
	RunE: hook.Wrap("ssh.keygen", runSSHKeygen),
}

func runSSHKeygen(_ *cobra.Command, args []string) error {
	name := "id_ed25519"
	if len(args) > 0 {
		name = args[0]
	}

	fmt.Println()
	fmt.Printf("  Generating ed25519 key: %s\n", ui.Accent.Render("~/.ssh/"+name))
	fmt.Println()

	pubPath, err := ssh.Keygen(name, "")
	if err != nil {
		return err
	}

	ui.Ok("Key generated: " + pubPath)
	fmt.Println()
	ui.Tip("Add to a host with: mine ssh copyid <alias>")
	fmt.Println()
	return nil
}

// --- mine ssh copyid ---

var sshCopyIDCmd = &cobra.Command{
	Use:   "copyid <alias>",
	Short: "Copy your public key to a remote host",
	Long:  `Copy your default public key to a remote SSH host using ssh-copy-id.`,
	Args:  cobra.ExactArgs(1),
	RunE:  hook.Wrap("ssh.copyid", runSSHCopyID),
}

func runSSHCopyID(_ *cobra.Command, args []string) error {
	alias := args[0]
	fmt.Println()
	fmt.Printf("  Copying public key to %s...\n", ui.Accent.Render(alias))
	fmt.Println()

	if err := ssh.CopyID(alias); err != nil {
		return err
	}

	fmt.Println()
	ui.Ok("Public key copied to " + alias)
	fmt.Println()
	return nil
}

// --- mine ssh keys ---

var sshKeysCmd = &cobra.Command{
	Use:   "keys",
	Short: "List SSH keys with fingerprints and host associations",
	RunE:  hook.Wrap("ssh.keys", runSSHKeys),
}

func runSSHKeys(_ *cobra.Command, _ []string) error {
	keys, err := ssh.ListKeys()
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		fmt.Println()
		fmt.Println(ui.Muted.Render("  No SSH keys found in ~/.ssh"))
		fmt.Printf("  Generate one: %s\n", ui.Accent.Render("mine ssh keygen"))
		fmt.Println()
		return nil
	}

	fmt.Println()
	fmt.Println(ui.Title.Render("  SSH Keys"))
	fmt.Println()

	for _, k := range keys {
		fmt.Printf("  %s\n", ui.Accent.Render(k.Name))
		if k.Fingerprint != "" {
			fmt.Printf("    %s %s\n", ui.Muted.Render("fingerprint"), k.Fingerprint)
		}
		if len(k.UsedBy) > 0 {
			fmt.Printf("    %s %s\n", ui.Muted.Render("used by    "), strings.Join(k.UsedBy, ", "))
		}
		fmt.Printf("    %s %s\n", ui.Muted.Render("path       "), k.PrivatePath)
		fmt.Println()
	}

	return nil
}

// --- mine ssh tunnel ---

var sshTunnelCmd = &cobra.Command{
	Use:   "tunnel <alias> <local:remote>",
	Short: "Start an SSH port-forward tunnel",
	Long: `Start an SSH port-forwarding tunnel to a configured host.

Examples:
  mine ssh tunnel myserver 8080:80      # forward local 8080 → remote 80
  mine ssh tunnel db 5433:5432          # forward local 5433 → remote 5432`,
	Args: cobra.ExactArgs(2),
	RunE: hook.Wrap("ssh.tunnel", runSSHTunnel),
}

func runSSHTunnel(_ *cobra.Command, args []string) error {
	alias := args[0]
	portSpec := args[1]

	local, remote, err := ssh.ParsePortSpec(portSpec)
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Printf("  Tunnel: %s → %s via %s\n",
		ui.Accent.Render("localhost:"+local),
		ui.Accent.Render("remote:"+remote),
		ui.Accent.Render(alias),
	)
	fmt.Println(ui.Muted.Render("  Press Ctrl+C to stop."))
	fmt.Println()

	return ssh.StartTunnel(alias, portSpec)
}
