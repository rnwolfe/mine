package ssh

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Tunnel represents an SSH port-forwarding tunnel.
type Tunnel struct {
	Alias  string
	Local  string
	Remote string
}

// ParsePortSpec parses a "local:remote" port spec into local and remote parts.
// Accepts "8080:80", "8080:host:80", etc.
func ParsePortSpec(spec string) (local, remote string, err error) {
	if spec == "" {
		return "", "", fmt.Errorf("port spec is empty; use format local:remote (e.g. 8080:80)")
	}
	idx := strings.Index(spec, ":")
	if idx < 0 {
		return "", "", fmt.Errorf("invalid port spec %q: expected local:remote (e.g. 8080:80)", spec)
	}
	local = spec[:idx]
	remote = spec[idx+1:]
	if local == "" || remote == "" {
		return "", "", fmt.Errorf("invalid port spec %q: both local and remote required", spec)
	}
	return local, remote, nil
}

// StartTunnel starts an SSH port-forwarding tunnel in the foreground.
// alias is the SSH config alias; portSpec is "localPort:remotePort".
func StartTunnel(alias, portSpec string) error {
	return StartTunnelFunc(alias, portSpec)
}

// StartTunnelFunc is the replaceable implementation of StartTunnel (for testing).
var StartTunnelFunc = startTunnelReal

func startTunnelReal(alias, portSpec string) error {
	local, remote, err := ParsePortSpec(portSpec)
	if err != nil {
		return err
	}

	// -N: don't execute remote command, just forward
	// -L: local port forward local:remote
	// -o ExitOnForwardFailure: exit if the port can't be forwarded
	cmd := exec.Command("ssh",
		"-N",
		"-o", "ExitOnForwardFailure=yes",
		"-L", fmt.Sprintf("%s:localhost:%s", local, remote),
		alias,
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tunnel %s → %s via %s: %w", local, remote, alias, err)
	}
	return nil
}
