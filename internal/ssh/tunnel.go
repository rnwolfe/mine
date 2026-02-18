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

// PortSpec holds the parsed components of a port-forwarding spec.
type PortSpec struct {
	Local      string // local port
	RemoteHost string // remote host (defaults to "localhost")
	RemotePort string // remote port
}

// ParsePortSpec parses a port spec into its components.
// Accepted formats:
//   - "8080:80"          → local 8080, remote localhost:80
//   - "8080:host:80"     → local 8080, remote host:80
//
// Returns (local, remoteHost:remotePort, err) so callers get the full remote address.
func ParsePortSpec(spec string) (local, remoteAddr string, err error) {
	if spec == "" {
		return "", "", fmt.Errorf("port spec is empty; use format local:remote (e.g. 8080:80)")
	}
	ps, parseErr := parsePortSpec(spec)
	if parseErr != nil {
		return "", "", parseErr
	}
	return ps.Local, ps.RemoteHost + ":" + ps.RemotePort, nil
}

// parsePortSpec parses a port spec into a PortSpec struct.
func parsePortSpec(spec string) (PortSpec, error) {
	parts := strings.SplitN(spec, ":", 3)
	switch len(parts) {
	case 2:
		// "local:remotePort"
		local, remotePort := parts[0], parts[1]
		if local == "" || remotePort == "" {
			return PortSpec{}, fmt.Errorf("invalid port spec %q: both local and remote required", spec)
		}
		return PortSpec{Local: local, RemoteHost: "localhost", RemotePort: remotePort}, nil
	case 3:
		// "local:remoteHost:remotePort"
		local, remoteHost, remotePort := parts[0], parts[1], parts[2]
		if local == "" || remoteHost == "" || remotePort == "" {
			return PortSpec{}, fmt.Errorf("invalid port spec %q: local, remote host, and remote port all required", spec)
		}
		return PortSpec{Local: local, RemoteHost: remoteHost, RemotePort: remotePort}, nil
	default:
		return PortSpec{}, fmt.Errorf("invalid port spec %q: expected local:remote or local:host:remote", spec)
	}
}

// StartTunnel starts an SSH port-forwarding tunnel in the foreground.
// alias is the SSH config alias; portSpec is "localPort:remotePort" or "localPort:remoteHost:remotePort".
func StartTunnel(alias, portSpec string) error {
	return StartTunnelFunc(alias, portSpec)
}

// StartTunnelFunc is the replaceable implementation of StartTunnel (for testing).
var StartTunnelFunc = startTunnelReal

func startTunnelReal(alias, portSpec string) error {
	ps, err := parsePortSpec(portSpec)
	if err != nil {
		return err
	}

	// -N: don't execute remote command, just forward
	// -L: local port forward local:remoteHost:remotePort
	// -o ExitOnForwardFailure: exit if the port can't be forwarded
	cmd := exec.Command("ssh",
		"-N",
		"-o", "ExitOnForwardFailure=yes",
		"-L", fmt.Sprintf("%s:%s:%s", ps.Local, ps.RemoteHost, ps.RemotePort),
		alias,
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tunnel %s → %s:%s via %s: %w", ps.Local, ps.RemoteHost, ps.RemotePort, alias, err)
	}
	return nil
}
